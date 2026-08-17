package collector

import (
	"os"
	"runtime"
)

// MinimalEnvironment returns only values needed for basic process operation.
// It intentionally excludes generic token, key, credential, cloud, and session
// variables and never serializes the parent environment.
func MinimalEnvironment() []string {
	names := []string{"PATH"}
	if runtime.GOOS == "windows" {
		names = append(names, "SystemRoot", "WINDIR", "TEMP", "TMP", "PATHEXT")
	} else {
		names = append(names, "TMPDIR", "LANG", "LC_ALL", "TZ")
	}
	result := make([]string, 0, len(names))
	for _, name := range names {
		if value, exists := os.LookupEnv(name); exists && value != "" {
			result = append(result, name+"="+value)
		}
	}
	return result
}
