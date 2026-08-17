package inspect_test

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tjacky87-lab/diagsift/internal/bundle"
	inspectbundle "github.com/tjacky87-lab/diagsift/internal/inspect"
)

func TestRejectsMalformedAndUnsafeArchives(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.zip")
		if err := os.WriteFile(path, []byte("not a zip"), 0o600); err != nil {
			t.Fatal(err)
		}
		assertRejected(t, path)
	})
	for name, entries := range map[string][]zipEntry{
		"traversal": {{name: "../escape.txt", data: []byte("x")}},
		"drive":     {{name: "C:/escape.txt", data: []byte("x")}},
		"duplicate": {{name: "same.txt", data: []byte("x")}, {name: "same.txt", data: []byte("y")}},
		"collision": {{name: "Case.txt", data: []byte("x")}, {name: "case.txt", data: []byte("y")}},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.zip")
			writeZIP(t, path, entries)
			assertRejected(t, path)
		})
	}
}

func TestRejectsCorruptHash(t *testing.T) {
	payload := []byte("synthetic")
	metadata := baseMetadata([]bundle.EntryMetadata{{Name: "payload.txt", SHA256: strings.Repeat("0", 64), Size: int64(len(payload))}})
	path := filepath.Join(t.TempDir(), "corrupt.zip")
	writeZIP(t, path, []zipEntry{{name: "bundle.json", data: mustJSON(t, metadata)}, {name: "payload.txt", data: payload}})
	assertRejectedWith(t, path, "hash verification")
}

func TestRejectsCompressionBombRatio(t *testing.T) {
	payload := []byte(strings.Repeat("0", 8<<20))
	sum := sha256.Sum256(payload)
	metadata := baseMetadata([]bundle.EntryMetadata{{Name: "payload.txt", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}})
	path := filepath.Join(t.TempDir(), "ratio.zip")
	writeZIP(t, path, []zipEntry{{name: "bundle.json", data: mustJSON(t, metadata)}, {name: "payload.txt", data: payload}})
	assertRejectedWith(t, path, "compression-ratio")
}

func TestRejectsExcessiveEntryCount(t *testing.T) {
	entries := make([]zipEntry, inspectbundle.MaxArchiveEntries+1)
	for index := range entries {
		entries[index] = zipEntry{name: "entry-" + decimal(index), data: nil}
	}
	path := filepath.Join(t.TempDir(), "many.zip")
	writeZIP(t, path, entries)
	assertRejectedWith(t, path, "entry count")
}

type zipEntry struct {
	name string
	data []byte
}

func writeZIP(t *testing.T, path string, entries []zipEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for _, entry := range entries {
		writer, err := archive.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func baseMetadata(entries []bundle.EntryMetadata) bundle.Metadata {
	return bundle.Metadata{
		FormatVersion: bundle.FormatVersion, DiagSiftVersion: "test", Name: "inspect-test",
		ManifestSHA256: strings.Repeat("0", 64), CreatedAt: time.Now().UTC().Format(time.RFC3339),
		EntryCount: len(entries), Entries: entries,
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertRejected(t *testing.T, path string) {
	t.Helper()
	if _, err := inspectbundle.Open(path); err == nil {
		t.Fatal("expected bundle rejection")
	}
}

func assertRejectedWith(t *testing.T, path, expected string) {
	t.Helper()
	_, err := inspectbundle.Open(path)
	if err == nil {
		t.Fatal("expected bundle rejection")
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected rejection containing %q, got %q", expected, err)
	}
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[position:])
}
