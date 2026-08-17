package manifest

import "testing"

func FuzzSafeArchivePath(f *testing.F) {
	for _, seed := range []string{"collectors/logs/app.log", "../escape", `C:\\escape`, "a/../b", "bundle.json"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		result, err := SafeArchivePath(value)
		if err == nil {
			if second, secondErr := SafeArchivePath(result); secondErr != nil || second != result {
				t.Fatalf("accepted path is not stable: %q -> %q, %v", value, second, secondErr)
			}
		}
	})
}
