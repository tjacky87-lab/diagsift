package manifest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tjacky87-lab/diagsift/internal/policy"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func Validate(m Manifest) error {
	if m.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q", m.APIVersion)
	}
	if m.Kind != Kind {
		return fmt.Errorf("unsupported kind %q", m.Kind)
	}
	if !identifier.MatchString(m.Metadata.Name) {
		return fmt.Errorf("metadata.name must match %s", identifier)
	}
	if _, err := policy.EffectiveLimits(m.Limits.MaxFiles, m.Limits.MaxTotalBytes, m.Limits.MaxDuration.Duration); err != nil {
		return fmt.Errorf("invalid limits: %w", err)
	}
	if len(m.Roots) == 0 {
		return fmt.Errorf("at least one explicit root is required")
	}
	rootIDs := make(map[string]struct{}, len(m.Roots))
	for _, root := range m.Roots {
		if !identifier.MatchString(root.ID) {
			return fmt.Errorf("invalid root id %q", root.ID)
		}
		if _, exists := rootIDs[root.ID]; exists {
			return fmt.Errorf("duplicate root id %q", root.ID)
		}
		if strings.TrimSpace(root.Path) == "" {
			return fmt.Errorf("root %q path is required", root.ID)
		}
		rootIDs[root.ID] = struct{}{}
	}
	if len(m.Collectors) == 0 || len(m.Collectors) > policy.HardMaxCollectors {
		return fmt.Errorf("collectors count must be between 1 and %d", policy.HardMaxCollectors)
	}
	collectorIDs := make(map[string]struct{}, len(m.Collectors))
	pathCount := 0
	for _, collector := range m.Collectors {
		if !identifier.MatchString(collector.ID) {
			return fmt.Errorf("invalid collector id %q", collector.ID)
		}
		if _, exists := collectorIDs[collector.ID]; exists {
			return fmt.Errorf("duplicate collector id %q", collector.ID)
		}
		collectorIDs[collector.ID] = struct{}{}
		if err := validateCollector(collector, rootIDs); err != nil {
			return fmt.Errorf("collector %q: %w", collector.ID, err)
		}
		pathCount += len(collector.Paths)
	}
	if pathCount > policy.HardMaxPaths {
		return fmt.Errorf("manifest has %d paths, exceeding hard ceiling %d", pathCount, policy.HardMaxPaths)
	}
	return validateRedactions(m.Redactions)
}

func validateCollector(c Collector, roots map[string]struct{}) error {
	switch c.Type {
	case "file":
		if _, exists := roots[c.Root]; !exists {
			return fmt.Errorf("references unknown root %q", c.Root)
		}
		if len(c.Paths) == 0 {
			return fmt.Errorf("file collector requires paths")
		}
		for _, path := range c.Paths {
			if _, err := SafeRelativePath(path); err != nil {
				return fmt.Errorf("invalid file path %q: %w", path, err)
			}
		}
		if _, err := SafeArchivePath(c.ArchivePrefix); err != nil {
			return fmt.Errorf("invalid archivePrefix: %w", err)
		}
		if c.MaxBytes <= 0 || c.MaxBytes > policy.HardMaxFileBytes {
			return fmt.Errorf("maxBytes must be between 1 and %d", policy.HardMaxFileBytes)
		}
		if hasCommandFields(c) || len(c.Fields) > 0 {
			return fmt.Errorf("fields and command properties are not valid for file collectors")
		}
	case "system":
		if len(c.Fields) == 0 {
			return fmt.Errorf("system collector requires fields")
		}
		allowed := map[string]bool{"os": true, "arch": true}
		seen := map[string]bool{}
		for _, field := range c.Fields {
			if !allowed[field] {
				return fmt.Errorf("unsupported system field %q", field)
			}
			if seen[field] {
				return fmt.Errorf("duplicate system field %q", field)
			}
			seen[field] = true
		}
		if c.Root != "" || len(c.Paths) > 0 || c.ArchivePrefix != "" || c.MaxBytes != 0 || hasCommandFields(c) {
			return fmt.Errorf("file and command properties are not valid for system collectors")
		}
	case "command":
		if err := policy.ValidateCommand(c.Executable, c.Args); err != nil {
			return err
		}
		if c.Timeout.Duration <= 0 || c.Timeout.Duration > policy.HardMaxCommandTimeout {
			return fmt.Errorf("timeout must be between 1ns and %s", policy.HardMaxCommandTimeout)
		}
		if c.MaxOutputBytes <= 0 || c.MaxOutputBytes > policy.HardMaxCommandBytes {
			return fmt.Errorf("maxOutputBytes must be between 1 and %d", policy.HardMaxCommandBytes)
		}
		if c.Root != "" || len(c.Paths) > 0 || c.ArchivePrefix != "" || c.MaxBytes != 0 || len(c.Fields) > 0 {
			return fmt.Errorf("file and system properties are not valid for command collectors")
		}
	default:
		return fmt.Errorf("unsupported type %q", c.Type)
	}
	return nil
}

func hasCommandFields(c Collector) bool {
	return c.Executable != "" || len(c.Args) > 0 || c.Timeout.Duration != 0 || c.MaxOutputBytes != 0
}

func validateRedactions(redactions Redactions) error {
	known := map[string]bool{
		"credentials": true, "private-keys": true, "bearer-tokens": true,
		"url-credentials": true, "connection-strings": true, "paths": true,
	}
	seen := map[string]bool{}
	for _, builtin := range redactions.Builtins {
		if !known[builtin] {
			return fmt.Errorf("unknown built-in redaction %q", builtin)
		}
		if seen[builtin] {
			return fmt.Errorf("duplicate built-in redaction %q", builtin)
		}
		seen[builtin] = true
	}
	customIDs := map[string]bool{}
	for _, rule := range redactions.Custom {
		if !identifier.MatchString(rule.ID) {
			return fmt.Errorf("invalid custom redaction id %q", rule.ID)
		}
		if customIDs[rule.ID] {
			return fmt.Errorf("duplicate custom redaction id %q", rule.ID)
		}
		customIDs[rule.ID] = true
		if rule.Pattern == "" || len(rule.Pattern) > policy.HardMaxRegexLength {
			return fmt.Errorf("custom redaction %q pattern length must be between 1 and %d", rule.ID, policy.HardMaxRegexLength)
		}
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("custom redaction %q pattern is invalid", rule.ID)
		}
	}
	return nil
}
