// Package plan creates a deterministic, side-effect-free collection preview.
package plan

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/policy"
)

type Plan struct {
	APIVersion   string      `json:"apiVersion"`
	ManifestPath string      `json:"manifestPath"`
	ManifestHash string      `json:"manifestSHA256"`
	Name         string      `json:"name"`
	Limits       Limits      `json:"limits"`
	Roots        []Root      `json:"roots"`
	Collectors   []Collector `json:"collectors"`
	Redactions   []string    `json:"redactions"`
	Warnings     []string    `json:"warnings"`
}

type Limits struct {
	MaxFiles      int    `json:"maxFiles"`
	MaxTotalBytes int64  `json:"maxTotalBytes"`
	MaxDuration   string `json:"maxDuration"`
}

type Root struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type Collector struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Root           string   `json:"root,omitempty"`
	Paths          []string `json:"paths,omitempty"`
	ArchivePrefix  string   `json:"archivePrefix,omitempty"`
	MaxBytes       int64    `json:"maxBytes,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	Executable     string   `json:"executable,omitempty"`
	Args           []string `json:"args,omitempty"`
	Timeout        string   `json:"timeout,omitempty"`
	MaxOutputBytes int64    `json:"maxOutputBytes,omitempty"`
}

func Build(loaded manifest.Loaded) (Plan, error) {
	limits, err := policy.EffectiveLimits(
		loaded.Manifest.Limits.MaxFiles,
		loaded.Manifest.Limits.MaxTotalBytes,
		loaded.Manifest.Limits.MaxDuration.Duration,
	)
	if err != nil {
		return Plan{}, err
	}
	base := filepath.Dir(loaded.Path)
	result := Plan{
		APIVersion:   loaded.Manifest.APIVersion,
		ManifestPath: loaded.Path,
		ManifestHash: loaded.SHA256,
		Name:         loaded.Manifest.Metadata.Name,
		Limits: Limits{
			MaxFiles:      limits.MaxFiles,
			MaxTotalBytes: limits.MaxTotalBytes,
			MaxDuration:   limits.MaxDuration.String(),
		},
		Warnings: []string{
			"Redaction reduces risk but does not guarantee that a bundle is safe to share.",
			"Allowed child executables are not sandboxed and may have their own side effects.",
		},
	}
	for _, root := range loaded.Manifest.Roots {
		value := root.Path
		if !filepath.IsAbs(value) {
			value = filepath.Join(base, value)
		}
		abs, err := filepath.Abs(value)
		if err != nil {
			return Plan{}, fmt.Errorf("resolve root %q: %w", root.ID, err)
		}
		result.Roots = append(result.Roots, Root{ID: root.ID, Path: filepath.Clean(abs)})
	}
	for _, c := range loaded.Manifest.Collectors {
		entry := Collector{
			ID: c.ID, Type: c.Type, Root: c.Root, Paths: clone(c.Paths),
			ArchivePrefix: c.ArchivePrefix, MaxBytes: c.MaxBytes,
			Fields: clone(c.Fields), Executable: c.Executable, Args: clone(c.Args),
			MaxOutputBytes: c.MaxOutputBytes,
		}
		if c.Timeout.Duration != 0 {
			entry.Timeout = c.Timeout.String()
		}
		result.Collectors = append(result.Collectors, entry)
	}
	for _, builtin := range loaded.Manifest.Redactions.Builtins {
		result.Redactions = append(result.Redactions, "builtin:"+builtin)
	}
	for _, custom := range loaded.Manifest.Redactions.Custom {
		result.Redactions = append(result.Redactions, "custom:"+custom.ID)
	}
	if result.Redactions == nil {
		result.Redactions = []string{}
	}
	return result, nil
}

func (p Plan) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (p Plan) Deadline(now time.Time) time.Time {
	duration, _ := time.ParseDuration(p.Limits.MaxDuration)
	return now.Add(duration)
}

func SortForReport(p *Plan) {
	sort.Slice(p.Roots, func(i, j int) bool { return p.Roots[i].ID < p.Roots[j].ID })
}

func clone(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}
