package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Loaded struct {
	Manifest Manifest
	Path     string
	SHA256   string
}

func Load(path string) (Loaded, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve manifest path: %w", err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return Loaded{}, fmt.Errorf("read manifest: %w", err)
	}
	if len(data) > 1<<20 {
		return Loaded{}, fmt.Errorf("manifest exceeds 1 MiB limit")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var document Manifest
	if err := dec.Decode(&document); err != nil {
		return Loaded{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Loaded{}, fmt.Errorf("manifest must contain exactly one YAML document")
		}
		return Loaded{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := Validate(document); err != nil {
		return Loaded{}, err
	}
	sum := sha256.Sum256(data)
	return Loaded{
		Manifest: document,
		Path:     filepath.Clean(abs),
		SHA256:   hex.EncodeToString(sum[:]),
	}, nil
}

func SafeRelativePath(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("path contains NUL")
	}
	if filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return "", fmt.Errorf("path must be relative")
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must not escape its root")
	}
	return clean, nil
}

func SafeArchivePath(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.Contains(value, "\\") || strings.Contains(value, ":") || strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("archive path must use forward slashes and contain no NUL")
	}
	if strings.HasPrefix(value, "/") || filepath.VolumeName(value) != "" {
		return "", fmt.Errorf("archive path must be relative")
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("archive path contains an unsafe segment")
		}
	}
	return strings.Join(parts, "/"), nil
}
