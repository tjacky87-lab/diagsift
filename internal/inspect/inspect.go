// Package inspect validates a DiagSift ZIP without extracting it.
package inspect

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/tjacky87-lab/diagsift/internal/bundle"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/policy"
	"github.com/tjacky87-lab/diagsift/internal/report"
)

var safeBundleName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

const (
	MaxArchiveEntries      = policy.HardMaxFiles + 16
	MaxArchiveUncompressed = uint64(policy.HardMaxTotalBytes + 8<<20)
	MaxEntryUncompressed   = uint64(policy.HardMaxTotalBytes + 1<<20)
	MaxCompressionRatio    = uint64(1000)
)

type Summary struct {
	FormatVersion     string `json:"formatVersion"`
	Name              string `json:"name"`
	CreatedAt         string `json:"createdAt"`
	Partial           bool   `json:"partial"`
	EntryCount        int    `json:"entryCount"`
	ErrorCount        int    `json:"errorCount"`
	TotalUncompressed uint64 `json:"totalUncompressed"`
	HashesVerified    bool   `json:"hashesVerified"`
}

func Open(path string) (Summary, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return Summary{}, fmt.Errorf("bundle is not a readable ZIP")
	}
	defer func() { _ = archive.Close() }()
	if len(archive.File) == 0 || len(archive.File) > MaxArchiveEntries {
		return Summary{}, fmt.Errorf("bundle entry count is outside safety limits")
	}
	files := make(map[string]*zip.File, len(archive.File))
	var total uint64
	for _, file := range archive.File {
		name, err := manifest.SafeArchivePath(file.Name)
		if err != nil || name == "" || file.FileInfo().IsDir() {
			return Summary{}, fmt.Errorf("bundle contains an unsafe entry name")
		}
		key := strings.ToLower(name)
		if _, exists := files[key]; exists {
			return Summary{}, fmt.Errorf("bundle contains duplicate or colliding entry names")
		}
		if file.UncompressedSize64 > MaxEntryUncompressed {
			return Summary{}, fmt.Errorf("bundle entry exceeds the uncompressed size limit")
		}
		if file.UncompressedSize64 > 1<<20 {
			if file.CompressedSize64 == 0 || file.UncompressedSize64 > MaxCompressionRatio*file.CompressedSize64 {
				return Summary{}, fmt.Errorf("bundle entry exceeds the compression-ratio limit")
			}
		}
		if total > MaxArchiveUncompressed-file.UncompressedSize64 {
			return Summary{}, fmt.Errorf("bundle exceeds the total uncompressed size limit")
		}
		total += file.UncompressedSize64
		files[key] = file
	}
	metadataFile, exists := files["bundle.json"]
	if !exists {
		return Summary{}, fmt.Errorf("bundle metadata is missing")
	}
	metadataData, err := readBounded(metadataFile, 2<<20)
	if err != nil {
		return Summary{}, fmt.Errorf("bundle metadata is unreadable")
	}
	var metadata bundle.Metadata
	decoder := json.NewDecoder(bytes.NewReader(metadataData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return Summary{}, fmt.Errorf("bundle metadata is malformed")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Summary{}, fmt.Errorf("bundle metadata has trailing content")
	}
	if metadata.FormatVersion != bundle.FormatVersion || !safeBundleName.MatchString(metadata.Name) {
		return Summary{}, fmt.Errorf("bundle metadata version or name is invalid")
	}
	manifestHash, err := hex.DecodeString(metadata.ManifestSHA256)
	if err != nil || len(manifestHash) != sha256.Size {
		return Summary{}, fmt.Errorf("bundle metadata manifest hash is invalid")
	}
	if _, err := time.Parse(time.RFC3339, metadata.CreatedAt); err != nil {
		return Summary{}, fmt.Errorf("bundle metadata creation time is invalid")
	}
	if metadata.ErrorCount < 0 || metadata.Partial != (metadata.ErrorCount > 0) {
		return Summary{}, fmt.Errorf("bundle metadata partial status is inconsistent")
	}
	if metadata.EntryCount != len(metadata.Entries) || metadata.EntryCount != len(archive.File)-1 {
		return Summary{}, fmt.Errorf("bundle metadata entry count does not match archive")
	}
	accounted := make(map[string]bool, len(metadata.Entries))
	for _, expected := range metadata.Entries {
		name, err := manifest.SafeArchivePath(expected.Name)
		if err != nil || name == "" || strings.EqualFold(name, "bundle.json") {
			return Summary{}, fmt.Errorf("bundle metadata contains an unsafe entry")
		}
		key := strings.ToLower(name)
		if accounted[key] {
			return Summary{}, fmt.Errorf("bundle metadata contains duplicate entries")
		}
		file, exists := files[key]
		if !exists {
			return Summary{}, fmt.Errorf("bundle metadata references a missing entry")
		}
		data, err := readBounded(file, MaxEntryUncompressed)
		if err != nil {
			return Summary{}, fmt.Errorf("bundle entry is unreadable")
		}
		if int64(len(data)) != expected.Size {
			return Summary{}, fmt.Errorf("bundle entry size does not match metadata")
		}
		for _, count := range expected.Redactions {
			if count < 0 {
				return Summary{}, fmt.Errorf("bundle metadata has an invalid redaction count")
			}
		}
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), expected.SHA256) {
			return Summary{}, fmt.Errorf("bundle entry hash verification failed")
		}
		accounted[key] = true
	}
	if len(accounted) != len(files)-1 {
		return Summary{}, fmt.Errorf("bundle contains an unaccounted entry")
	}
	_, hasErrors := accounted["errors.json"]
	if hasErrors != (metadata.ErrorCount > 0) {
		return Summary{}, fmt.Errorf("bundle errors report does not match metadata")
	}
	if hasErrors {
		errorData, err := readBounded(files["errors.json"], 2<<20)
		if err != nil {
			return Summary{}, fmt.Errorf("bundle errors report is unreadable")
		}
		var errors []report.Error
		errorDecoder := json.NewDecoder(bytes.NewReader(errorData))
		errorDecoder.DisallowUnknownFields()
		if err := errorDecoder.Decode(&errors); err != nil || len(errors) != metadata.ErrorCount {
			return Summary{}, fmt.Errorf("bundle errors report is inconsistent")
		}
		if err := errorDecoder.Decode(&trailing); err != io.EOF {
			return Summary{}, fmt.Errorf("bundle errors report has trailing content")
		}
	}
	if !accounted[strings.ToLower("REVIEW_BEFORE_SHARING.txt")] {
		return Summary{}, fmt.Errorf("bundle review notice is missing")
	}
	return Summary{
		FormatVersion: metadata.FormatVersion, Name: metadata.Name, CreatedAt: metadata.CreatedAt,
		Partial: metadata.Partial, EntryCount: metadata.EntryCount, ErrorCount: metadata.ErrorCount,
		TotalUncompressed: total, HashesVerified: true,
	}, nil
}

func (s Summary) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func readBounded(file *zip.File, limit uint64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) > limit {
		return nil, fmt.Errorf("entry exceeds limit")
	}
	return data, nil
}
