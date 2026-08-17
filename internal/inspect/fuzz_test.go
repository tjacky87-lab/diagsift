package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzOpen(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte{'P', 'K', 3, 4})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 2<<20 {
			t.Skip()
		}
		path := filepath.Join(t.TempDir(), "input.zip")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Open(path)
	})
}
