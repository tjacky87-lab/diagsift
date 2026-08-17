package collector_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/collector"
	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/policy"
)

func TestFileAndSystemCollectionWithPartialFailures(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "logs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "app.log"), []byte("synthetic log\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "binary.dat"), []byte{'x', 0, 'y'}, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, preview := load(t, dir, `
  - id: logs
    type: file
    root: project
    paths: [logs/app.log, logs/binary.dat, missing.log]
    maxBytes: 1024
  - id: platform
    type: system
    fields: [os, arch]
`)
	result := collector.CollectNonCommand(context.Background(), loaded, preview, collector.IdentitySanitizer{})
	if len(result.Entries) != 2 {
		t.Fatalf("entries=%d errors=%#v", len(result.Entries), result.Errors)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected binary and missing errors, got %#v", result.Errors)
	}
	if result.Entries[0].Name != "collectors/logs/logs/app.log" {
		t.Fatalf("unexpected file entry %q", result.Entries[0].Name)
	}
	if strings.Contains(string(result.Entries[1].Data), "hostname") {
		t.Fatalf("system collector exposed hostname: %q", result.Entries[1].Data)
	}
}

func TestDirectoryPathRejectedWithoutWalking(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "logs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "must-not-be-collected.log"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, preview := load(t, dir, `
  - id: logs
    type: file
    root: project
    paths: [logs]
    maxBytes: 1024
`)
	result := collector.CollectNonCommand(context.Background(), loaded, preview, collector.IdentitySanitizer{})
	if len(result.Entries) != 0 || len(result.Errors) != 1 || result.Errors[0].Code != "directory-unsupported" {
		t.Fatalf("unexpected directory result %#v", result)
	}
}

func TestFileTruncationAndTotalLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.log"), []byte(strings.Repeat("a", 20)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.log"), []byte(strings.Repeat("b", 20)), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, preview := loadWithLimits(t, dir, 1, 10, `
  - id: logs
    type: file
    root: project
    paths: [a.log, b.log]
    maxBytes: 8
`)
	result := collector.CollectNonCommand(context.Background(), loaded, preview, collector.IdentitySanitizer{})
	if len(result.Entries) != 1 || !result.Entries[0].Truncated || len(result.Entries[0].Data) != 8 {
		t.Fatalf("unexpected result %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Code != "limit" {
		t.Fatalf("expected limit error, got %#v", result.Errors)
	}
}

func TestSymlinkSkipped(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.log")
	if err := os.WriteFile(outside, []byte("must not collect"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked.log")
	if err := os.Symlink(outside, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
	loaded, preview := load(t, dir, `
  - id: logs
    type: file
    root: project
    paths: [linked.log]
    maxBytes: 1024
`)
	result := collector.CollectNonCommand(context.Background(), loaded, preview, collector.IdentitySanitizer{})
	if len(result.Entries) != 0 || len(result.Errors) != 1 || result.Errors[0].Code != "link-skipped" {
		t.Fatalf("unexpected result %#v", result)
	}
}

func TestCommandNeverRunsInNonCommandCollection(t *testing.T) {
	dir := t.TempDir()
	loaded, preview := load(t, dir, `
  - id: command
    type: command
    executable: definitely-not-run
    args: [version]
    timeout: 1s
    maxOutputBytes: 100
`)
	result := collector.CollectNonCommand(context.Background(), loaded, preview, collector.IdentitySanitizer{})
	if len(result.Entries) != 0 || len(result.Errors) != 1 || result.Errors[0].Code != "unavailable" {
		t.Fatalf("unexpected result %#v", result)
	}
}

func TestCollectorErrorsAreBoundedWithOneTerminalSummary(t *testing.T) {
	sink := collector.NewSink(policy.Limits{MaxFiles: 1, MaxTotalBytes: 1, MaxDuration: 1})
	for index := 0; index < policy.HardMaxRecordedErrors+20; index++ {
		sink.Error("synthetic", "synthetic", "synthetic collector error")
	}
	result := sink.Result()
	if len(result.Errors) != policy.HardMaxRecordedErrors {
		t.Fatalf("errors=%d want=%d", len(result.Errors), policy.HardMaxRecordedErrors)
	}
	summaries := 0
	for _, entry := range result.Errors {
		if entry.Code == "errors-suppressed" {
			summaries++
		}
	}
	if summaries != 1 || result.Errors[len(result.Errors)-1].Code != "errors-suppressed" {
		t.Fatalf("expected one terminal suppression summary, got %#v", result.Errors)
	}
}

func load(t *testing.T, dir, collectors string) (manifest.Loaded, plan.Plan) {
	return loadWithLimits(t, dir, 20, 1<<20, collectors)
}

func loadWithLimits(t *testing.T, dir string, maxFiles int, maxBytes int64, collectors string) (manifest.Loaded, plan.Plan) {
	t.Helper()
	input := `apiVersion: diagsift/v1alpha1
kind: DiagnosticBundle
metadata: {name: collector-test}
limits:
  maxFiles: ` + fmtInt(maxFiles) + `
  maxTotalBytes: ` + fmtInt64(maxBytes) + `
  maxDuration: 5s
roots: [{id: project, path: .}]
collectors:` + collectors
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

func fmtInt(value int) string { return fmtInt64(int64(value)) }

func fmtInt64(value int64) string {
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
