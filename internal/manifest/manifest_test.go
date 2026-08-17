package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
)

const valid = `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata:
  name: test-bundle
limits:
  maxFiles: 10
  maxTotalBytes: 1048576
  maxDuration: 30s
roots:
  - id: project
    path: .
collectors:
  - id: logs
    type: file
    root: project
    paths: ["fixtures/app.log"]
    maxBytes: 1024
redactions:
  builtins: [credentials]
`

func TestLoadValid(t *testing.T) {
	loaded, err := manifest.Load(writeManifest(t, valid))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Manifest.Metadata.Name != "test-bundle" || len(loaded.SHA256) != 64 {
		t.Fatalf("unexpected loaded manifest: %#v", loaded)
	}
}

func TestLoadFailsClosed(t *testing.T) {
	tests := map[string]string{
		"unknown field":       strings.Replace(valid, "  name: test-bundle", "  name: test-bundle\n  surprise: true", 1),
		"unsupported version": strings.Replace(valid, manifest.APIVersion, "diagsift/v9", 1),
		"duplicate collector": strings.Replace(valid, "  - id: logs", "  - id: logs\n    type: system\n    fields: [os]\n  - id: logs", 1),
		"raised ceiling":      strings.Replace(valid, "maxFiles: 10", "maxFiles: 1001", 1),
		"root escape":         strings.Replace(valid, "fixtures/app.log", "../private.txt", 1),
		"hostname field": `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: test-bundle}
limits: {maxFiles: 10, maxTotalBytes: 1048576, maxDuration: 30s}
roots: [{id: project, path: .}]
collectors:
  - id: platform
    type: system
    fields: [os, hostname]
`,
		"shell command": `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: test-bundle}
limits: {maxFiles: 10, maxTotalBytes: 1048576, maxDuration: 30s}
roots: [{id: project, path: .}]
collectors:
  - id: bad
    type: command
    executable: C:\\Windows\\System32\\cmd.EXE
    args: [/C, whoami]
    timeout: 5s
    maxOutputBytes: 1000
`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := manifest.Load(writeManifest(t, input)); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func TestArchivePathSafety(t *testing.T) {
	for _, value := range []string{"../x", "/x", `C:\\x`, `a\\b`, "a//b", "a/./b"} {
		if _, err := manifest.SafeArchivePath(value); err == nil {
			t.Errorf("SafeArchivePath(%q) unexpectedly succeeded", value)
		}
	}
}

func writeManifest(t *testing.T, input string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "diagsift.yaml")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
