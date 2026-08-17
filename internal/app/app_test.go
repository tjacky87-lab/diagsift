package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/app"
)

func TestValidateAndPlan(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "diagsift.yaml")
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: cli-test}
limits: {maxFiles: 5, maxTotalBytes: 4096, maxDuration: 5s}
roots: [{id: project, path: .}]
collectors:
  - id: platform
    type: system
    fields: [os, arch]
`
	if err := os.WriteFile(manifestPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"validate", "plan"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := app.Run([]string{command, manifestPath}, strings.NewReader(""), &stdout, &stderr); code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
			if stderr.Len() != 0 || stdout.Len() == 0 {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestInvalidManifestExitCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diagsift.yaml")
	if err := os.WriteFile(path, []byte("apiVersion: wrong\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := app.Run([]string{"validate", path}, strings.NewReader(""), &stdout, &stderr); code != app.ExitValidation {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestCollectConsentAndConsoleCanarySafety(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input.log"), []byte("Bearer invalid.synthetic.console.token"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "diagsift.yaml")
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: app-collect}
limits: {maxFiles: 5, maxTotalBytes: 4096, maxDuration: 5s}
roots: [{id: project, path: .}]
collectors:
  - id: input
    type: file
    root: project
    paths: [input.log]
    maxBytes: 1024
redactions: {builtins: [bearer-tokens]}
`
	if err := os.WriteFile(manifestPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "bundle.zip")
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"collect", manifestPath, "--output", output, "--yes"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, channel := range []string{stdout.String(), stderr.String()} {
		if strings.Contains(channel, "invalid.synthetic.console.token") {
			t.Fatal("synthetic canary reached console output")
		}
	}

	secondOutput := filepath.Join(dir, "cancelled.zip")
	stdout.Reset()
	stderr.Reset()
	code = app.Run([]string{"collect", manifestPath, "--output", secondOutput}, strings.NewReader("no\n"), &stdout, &stderr)
	if code != app.ExitConsent {
		t.Fatalf("expected consent exit, got %d", code)
	}
	if _, err := os.Stat(secondOutput); !os.IsNotExist(err) {
		t.Fatalf("bundle created without consent: %v", err)
	}
}
