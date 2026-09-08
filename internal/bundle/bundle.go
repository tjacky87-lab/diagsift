// Package bundle creates local ordinary ZIP diagnostic bundles.
package bundle

import (
	"archive/zip"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tjacky87-lab/diagsift/internal/collector"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/policy"
	"github.com/tjacky87-lab/diagsift/internal/report"
	"github.com/tjacky87-lab/diagsift/internal/workspace"
)

const FormatVersion = "diagsift.bundle/v1alpha1"

type Metadata struct {
	FormatVersion   string          `json:"formatVersion"`
	DiagSiftVersion string          `json:"diagsiftVersion"`
	Name            string          `json:"name"`
	ManifestSHA256  string          `json:"manifestSHA256"`
	CreatedAt       string          `json:"createdAt"`
	Partial         bool            `json:"partial"`
	EntryCount      int             `json:"entryCount"`
	ErrorCount      int             `json:"errorCount"`
	Entries         []EntryMetadata `json:"entries"`
	Warnings        []string        `json:"warnings"`
}

type EntryMetadata struct {
	Name       string         `json:"name"`
	SHA256     string         `json:"sha256"`
	Size       int64          `json:"size"`
	Collector  string         `json:"collector,omitempty"`
	Truncated  bool           `json:"truncated,omitempty"`
	Redactions map[string]int `json:"redactions,omitempty"`
}

type Result struct {
	Entries int
	Errors  int
	Partial bool
}

func Create(ctx context.Context, loaded manifest.Loaded, preview plan.Plan, output, version string, sanitizer collector.Sanitizer) (Result, error) {
	if _, err := os.Lstat(output); err == nil {
		return Result{}, fmt.Errorf("output already exists")
	} else if !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("output path is unavailable")
	}
	deadlineContext, cancel := context.WithTimeout(ctx, loaded.Manifest.Limits.MaxDuration.Duration)
	defer cancel()
	collected := collector.Collect(deadlineContext, loaded, preview, sanitizer)

	entries := append([]report.Entry(nil), collected.Entries...)
	if len(collected.Errors) > 0 {
		data, err := json.MarshalIndent(collected.Errors, "", "  ")
		if err != nil {
			return Result{}, fmt.Errorf("encode sanitized errors")
		}
		entries = append(entries, report.Entry{Name: "errors.json", Data: append(data, '\n')})
	}
	entries = append(entries, report.Entry{
		Name: "REVIEW_BEFORE_SHARING.txt",
		Data: []byte("Review every entry before sharing. Redaction reduces risk but does not guarantee safety.\n" +
			"DiagSift did not upload this bundle. You decide whether and how to share it.\n" +
			"Allowed command executables are not sandboxed.\n"),
	})

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	metadata := Metadata{
		FormatVersion: FormatVersion, DiagSiftVersion: version,
		Name: loaded.Manifest.Metadata.Name, ManifestSHA256: loaded.SHA256,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Partial:   len(collected.Errors) > 0, ErrorCount: len(collected.Errors),
		Warnings: []string{
			"Redaction reduces risk but does not guarantee that a bundle is safe to share.",
			"Allowed command executables are not sandboxed.",
		},
	}
	for index := range entries {
		sum := sha256.Sum256(entries[index].Data)
		entries[index].SHA256 = hex.EncodeToString(sum[:])
		entries[index].Size = int64(len(entries[index].Data))
		metadata.Entries = append(metadata.Entries, EntryMetadata{
			Name: entries[index].Name, SHA256: entries[index].SHA256, Size: entries[index].Size,
			Collector: entries[index].Collector, Truncated: entries[index].Truncated,
			Redactions: entries[index].Redactions,
		})
	}
	metadata.EntryCount = len(metadata.Entries)
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode bundle metadata")
	}
	entries = append(entries, report.Entry{Name: "bundle.json", Data: append(metadataData, '\n')})
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	staging, err := workspace.New()
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = staging.Remove() }()
	if err := staging.Stage(entries); err != nil {
		return Result{}, err
	}
	if err := writeAtomicZIP(output, staging, entries); err != nil {
		return Result{}, err
	}
	return Result{Entries: len(collected.Entries), Errors: len(collected.Errors), Partial: len(collected.Errors) > 0}, nil
}

func writeAtomicZIP(output string, staging *workspace.Workspace, entries []report.Entry) error {
	abs, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path")
	}
	parent := filepath.Dir(abs)
	temporary, err := os.CreateTemp(parent, ".diagsift-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary bundle")
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary bundle")
	}
	archive := zip.NewWriter(temporary)
	archive.RegisterCompressor(zip.Deflate, newDeflater)
	for _, entry := range entries {
		method, err := compressionMethod(entry.Data)
		if err != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return err
		}
		header := &zip.FileHeader{Name: entry.Name, Method: method, Modified: time.Unix(0, 0).UTC()}
		writer, err := archive.CreateHeader(header)
		if err != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return fmt.Errorf("create archive entry")
		}
		file, err := staging.Open(entry.Name)
		if err != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return fmt.Errorf("open staged entry")
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return fmt.Errorf("write archive entry")
		}
	}
	if err := archive.Close(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("finalize archive")
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync archive")
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close archive")
	}
	// Linking the completed same-directory temporary file publishes it atomically
	// and refuses ALL existing names, including dangling symlinks. Stat + Rename
	// is not sufficient: Rename can replace a file created between those calls.
	if err := os.Link(temporaryName, abs); err != nil {
		return fmt.Errorf("publish bundle without overwriting (destination must support hard links): %w", err)
	}
	return nil
}

// Large repetitive logs can exceed the inspector's bomb-ratio ceiling despite
// being within every collection limit. Store those entries without compression;
// keep the same inspection ceiling for untrusted bundles.
func compressionMethod(data []byte) (uint16, error) {
	if len(data) <= 1<<20 {
		return zip.Deflate, nil
	}
	var count compressedSize
	writer, err := newDeflater(&count)
	if err != nil {
		return 0, fmt.Errorf("initialize compression size check")
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return 0, fmt.Errorf("check compressed entry size")
	}
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("finalize compression size check")
	}
	if uint64(len(data)) > policy.HardMaxCompressionRatio*count.bytes {
		return zip.Store, nil
	}
	return zip.Deflate, nil
}

type compressedSize struct{ bytes uint64 }

func (c *compressedSize) Write(data []byte) (int, error) {
	c.bytes += uint64(len(data))
	return len(data), nil
}

// Use the same fixed level for the size check and the actual ZIP stream.
func newDeflater(w io.Writer) (io.WriteCloser, error) {
	return flate.NewWriter(w, 5)
}
