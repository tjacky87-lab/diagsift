// Package policy defines non-configurable product safety ceilings and command policy.
package policy

import (
	"fmt"
	"path"
	"strings"
	"time"
)

const (
	HardMaxFiles          = 1_000
	HardMaxTotalBytes     = int64(64 << 20)
	HardMaxFileBytes      = int64(8 << 20)
	HardMaxCommandBytes   = int64(4 << 20)
	HardMaxDuration       = 5 * time.Minute
	HardMaxCommandTimeout = 1 * time.Minute
	HardMaxRegexLength    = 512
	HardMaxCollectors     = 100
	HardMaxPaths          = 500
	HardMaxRecordedErrors = 256
)

type Limits struct {
	MaxFiles      int           `json:"maxFiles"`
	MaxTotalBytes int64         `json:"maxTotalBytes"`
	MaxDuration   time.Duration `json:"maxDuration"`
}

func EffectiveLimits(maxFiles int, maxTotalBytes int64, maxDuration time.Duration) (Limits, error) {
	if maxFiles <= 0 || maxTotalBytes <= 0 || maxDuration <= 0 {
		return Limits{}, fmt.Errorf("all limits must be greater than zero")
	}
	if maxFiles > HardMaxFiles {
		return Limits{}, fmt.Errorf("maxFiles %d exceeds hard ceiling %d", maxFiles, HardMaxFiles)
	}
	if maxTotalBytes > HardMaxTotalBytes {
		return Limits{}, fmt.Errorf("maxTotalBytes %d exceeds hard ceiling %d", maxTotalBytes, HardMaxTotalBytes)
	}
	if maxDuration > HardMaxDuration {
		return Limits{}, fmt.Errorf("maxDuration %s exceeds hard ceiling %s", maxDuration, HardMaxDuration)
	}
	return Limits{MaxFiles: maxFiles, MaxTotalBytes: maxTotalBytes, MaxDuration: maxDuration}, nil
}

// ValidateCommand rejects shell interpreters entirely. This is a policy guard,
// not a sandbox: an allowed non-shell executable can still have arbitrary
// filesystem or network side effects of its own.
func ValidateCommand(executable string, args []string) error {
	normalizedExecutable := strings.ReplaceAll(strings.TrimSpace(executable), "\\", "/")
	base := strings.ToLower(path.Base(normalizedExecutable))
	if base == "" {
		return fmt.Errorf("command executable is required")
	}
	if strings.HasSuffix(base, ".bat") || strings.HasSuffix(base, ".cmd") {
		return fmt.Errorf("Windows batch file execution is prohibited")
	}
	for _, suffix := range []string{".exe", ".com"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	switch base {
	case "cmd", "powershell", "pwsh", "sh", "bash", "zsh", "dash", "ksh":
		return fmt.Errorf("shell interpreter execution is prohibited")
	}
	return nil
}
