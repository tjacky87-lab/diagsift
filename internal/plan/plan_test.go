package plan_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
)

func TestBuildIsDeterministicAndDoesNotExecute(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "must-not-exist")
	manifestPath := filepath.Join(dir, "diagsift.yaml")
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: test-plan}
limits: {maxFiles: 5, maxTotalBytes: 4096, maxDuration: 5s}
roots: [{id: project, path: .}]
collectors:
  - id: side-effect-check
    type: command
    executable: test-helper
    args: ['` + sentinel + `']
    timeout: 1s
    maxOutputBytes: 100
`
	if err := os.WriteFile(manifestPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := manifest.Load(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := plan.Build(loaded)
	if err != nil {
		t.Fatal(err)
	}
	second, err := plan.Build(loaded)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := first.JSON()
	b, _ := second.JSON()
	if string(a) != string(b) {
		t.Fatalf("plan output is not deterministic:\n%s\n%s", a, b)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("plan caused side effect; stat error = %v", err)
	}
}
