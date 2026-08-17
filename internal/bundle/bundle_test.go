package bundle_test

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/bundle"
	inspectbundle "github.com/tjacky87-lab/diagsift/internal/inspect"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/redact"
)

var syntheticCanaries = []string{
	"-----BEGIN PRIVATE KEY-----\nINVALID-SYNTHETIC-FINAL-KEY\n-----END PRIVATE KEY-----",
	"Bearer invalid.synthetic.final.token",
	"password=INVALID_SYNTHETIC_FINAL_PASSWORD",
	"https://synthetic-final:invalid-password@example.invalid/path",
	"Server=example.invalid;User Id=synthetic-final;Password=invalid-final;",
	`C:\Users\SyntheticFinal\project`,
	"/home/synthetic-final/project",
}

func TestCanariesAbsentFromEveryDecompressedEntry(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "canaries.log")
	if err := os.WriteFile(inputPath, []byte(strings.Join(syntheticCanaries, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, preview := loadManifest(t, dir, "canaries.log")
	r, err := redact.New(loaded.Manifest.Redactions)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "bundle.zip")
	result, err := bundle.Create(context.Background(), loaded, preview, output, "test", r)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Partial || result.Errors != 1 || result.Entries != 1 {
		t.Fatalf("unexpected result %#v", result)
	}
	summary, err := inspectbundle.Open(output)
	if err != nil || !summary.HashesVerified {
		t.Fatalf("inspect summary=%#v err=%v", summary, err)
	}

	archive, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := archive.Close(); err != nil {
			t.Errorf("close archive: %v", err)
		}
	}()
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		for _, canary := range syntheticCanaries {
			if strings.Contains(string(data), canary) {
				t.Fatalf("synthetic canary survived in decompressed entry %q", file.Name)
			}
		}
	}
}

func TestCreateRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input.log"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, preview := loadManifest(t, dir, "input.log")
	r, _ := redact.New(loaded.Manifest.Redactions)
	output := filepath.Join(dir, "existing.zip")
	if err := os.WriteFile(output, []byte("preserve me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Create(context.Background(), loaded, preview, output, "test", r); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	data, _ := os.ReadFile(output)
	if string(data) != "preserve me" {
		t.Fatal("existing output was modified")
	}
}

func loadManifest(t *testing.T, dir, collectedPath string) (manifest.Loaded, plan.Plan) {
	t.Helper()
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: bundle-test}
limits: {maxFiles: 10, maxTotalBytes: 1048576, maxDuration: 10s}
roots: [{id: project, path: .}]
collectors:
  - id: input
    type: file
    root: project
    paths: [` + collectedPath + `, missing-synthetic.log]
    maxBytes: 1048576
redactions:
  builtins: [private-keys, bearer-tokens, credentials, url-credentials, connection-strings, paths]
`
	path := filepath.Join(dir, "diagsift.yaml")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.Build(loaded)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, preview
}
