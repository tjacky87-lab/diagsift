package collector_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tjacky87-lab/diagsift/internal/collector"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/redact"
)

func TestCommandHelper(t *testing.T) {
	separator := -1
	for index, value := range os.Args {
		if value == "--" {
			separator = index
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	switch os.Args[separator+1] {
	case "output":
		fmt.Printf("Bearer invalid.synthetic.command.token secret-env=%s", os.Getenv("DIAGSIFT_TEST_SECRET"))
		fmt.Fprint(os.Stderr, "password=INVALID_SYNTHETIC_STDERR")
	case "inherited-pipe":
		child := exec.Command(os.Args[0], "-test.run=TestCommandHelper", "--", "sleep")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Println(child.Process.Pid)
	case "sleep":
		time.Sleep(5 * time.Second)
	}
	os.Exit(0)
}

func TestCommandOutputRedactedAndSecretEnvironmentExcluded(t *testing.T) {
	t.Setenv("DIAGSIFT_TEST_SECRET", "INVALID_SYNTHETIC_PARENT_SECRET")
	loaded, preview := commandManifest(t, "output", "2s", 4096)
	r, err := redact.New(manifest.Redactions{Builtins: []string{"bearer-tokens", "credentials"}})
	if err != nil {
		t.Fatal(err)
	}
	result := collector.Collect(context.Background(), loaded, preview, r)
	if len(result.Entries) != 2 || len(result.Errors) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	combined := string(result.Entries[0].Data) + string(result.Entries[1].Data)
	for _, forbidden := range []string{"invalid.synthetic.command.token", "INVALID_SYNTHETIC_STDERR", "INVALID_SYNTHETIC_PARENT_SECRET"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("forbidden canary survived: %s", forbidden)
		}
	}
}

func TestCommandTimeout(t *testing.T) {
	loaded, preview := commandManifest(t, "sleep", "100ms", 128)
	r, _ := redact.New(manifest.Redactions{})
	started := time.Now()
	result := collector.Collect(context.Background(), loaded, preview, r)
	if time.Since(started) > 3*time.Second {
		t.Fatal("timed-out command was not terminated promptly")
	}
	if len(result.Errors) != 1 || result.Errors[0].Code != "timeout" {
		t.Fatalf("unexpected errors %#v", result.Errors)
	}
}

func TestCommandOutputBounded(t *testing.T) {
	loaded, preview := commandManifest(t, "output", "2s", 8)
	r, _ := redact.New(manifest.Redactions{Builtins: []string{"bearer-tokens", "credentials"}})
	result := collector.Collect(context.Background(), loaded, preview, r)
	if len(result.Entries) != 2 {
		t.Fatalf("unexpected result %#v", result)
	}
	for _, entry := range result.Entries {
		if !entry.Truncated || len(entry.Data) > 8 {
			t.Fatalf("output not bounded: %#v", entry)
		}
	}
}

func commandManifest(t *testing.T, mode, timeout string, maxOutput int64) (manifest.Loaded, plan.Plan) {
	t.Helper()
	dir := t.TempDir()
	executable, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: command-test}
limits: {maxFiles: 10, maxTotalBytes: 65536, maxDuration: 10s}
roots: [{id: project, path: .}]
collectors:
  - id: helper
    type: command
    executable: ` + strconv.Quote(executable) + `
    args: ["-test.run=TestCommandHelper", "--", "` + mode + `"]
    timeout: ` + timeout + `
    maxOutputBytes: ` + strconv.FormatInt(maxOutput, 10) + `
redactions:
  builtins: [bearer-tokens, credentials]
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

func TestCommandDoesNotWaitForInheritedPipes(t *testing.T) {
	loaded, preview := commandManifest(t, "inherited-pipe", "500ms", 128)
	r, err := redact.New(manifest.Redactions{})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	result := collector.Collect(context.Background(), loaded, preview, r)
	// The product intentionally does not contain Windows descendants. Clean up
	// this test's child so it cannot lock the temporary test executable.
	for _, entry := range result.Entries {
		if strings.HasSuffix(entry.Name, "/stdout.txt") {
			pid, err := strconv.Atoi(strings.TrimSpace(string(entry.Data)))
			if err != nil {
				t.Fatal(err)
			}
			process, err := os.FindProcess(pid)
			if err == nil {
				defer func() { _ = process.Kill(); _, _ = process.Wait() }()
			}
		}
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("collection waited for descendant-held output pipes")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected timeout or incomplete-output error")
	}
}
