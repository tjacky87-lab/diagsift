package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/app"
)

func TestBasicExampleEndToEnd(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate integration test")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	manifestPath := filepath.Join(repository, "examples", "basic", "diagsift.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "basic.zip")
	var stdout, stderr bytes.Buffer
	if code := app.Run([]string{"collect", manifestPath, "--output", output, "--yes"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("collect code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"inspect", output}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("inspect code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"hashesVerified": true`) {
		t.Fatalf("unexpected inspect output %s", stdout.String())
	}
}
