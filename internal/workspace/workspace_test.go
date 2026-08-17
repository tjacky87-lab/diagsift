package workspace_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/redact"
	"github.com/tjacky87-lab/diagsift/internal/report"
	"github.com/tjacky87-lab/diagsift/internal/workspace"
)

func TestPrivateStaging(t *testing.T) {
	w, err := workspace.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	}()
	if err := w.Stage([]report.Entry{{Name: "collectors/test/out.txt", Data: []byte("synthetic")}}); err != nil {
		t.Fatal(err)
	}
	file, err := w.Open("collectors/test/out.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("close staged file: %v", err)
		}
	}()
	got, err := io.ReadAll(file)
	if err != nil || string(got) != "synthetic" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestOnlyRedactedCanaryIsStaged(t *testing.T) {
	r, err := redact.New(manifest.Redactions{Builtins: []string{"bearer-tokens"}})
	if err != nil {
		t.Fatal(err)
	}
	canary := []byte("Bearer invalid.synthetic.staging.token")
	clean, counts, err := r.Sanitize(canary)
	if err != nil || counts["bearer-tokens"] != 1 {
		t.Fatalf("counts=%v err=%v", counts, err)
	}
	w, err := workspace.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	}()
	if err := w.Stage([]report.Entry{{Name: "collectors/test/out.txt", Data: clean}}); err != nil {
		t.Fatal(err)
	}
	file, err := w.Open("collectors/test/out.txt")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil || bytes.Contains(data, canary) {
		t.Fatalf("staged data=%q err=%v", data, err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestStagingRejectsTraversal(t *testing.T) {
	w, err := workspace.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	}()
	if err := w.Stage([]report.Entry{{Name: "../escape", Data: []byte("x")}}); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
