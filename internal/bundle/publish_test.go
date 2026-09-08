package bundle

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/tjacky87-lab/diagsift/internal/report"
	"github.com/tjacky87-lab/diagsift/internal/workspace"
)

func TestConcurrentPublicationHasOnlyOneWinner(t *testing.T) {
	staging, err := workspace.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = staging.Remove() }()
	entries := []report.Entry{{Name: "test.txt", Data: []byte("synthetic")}}
	if err := staging.Stage(entries); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "bundle.zip")
	const writers = 8
	results := make(chan error, writers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- writeAtomicZIP(output, staging, entries)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("got %d successful publications, want 1", successes)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
	temporary, err := filepath.Glob(filepath.Join(filepath.Dir(output), ".diagsift-*.tmp"))
	if err != nil || len(temporary) != 0 {
		t.Fatalf("temporary files not cleaned: %v %v", temporary, err)
	}
}
