// Package manifest strictly decodes and validates the DiagSift manifest.
package manifest

import "time"

const (
	APIVersion = "diagsift/v1alpha1"
	Kind       = "DiagnosticBundle"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

type Manifest struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Limits     Limits      `yaml:"limits" json:"limits"`
	Roots      []Root      `yaml:"roots" json:"roots"`
	Collectors []Collector `yaml:"collectors" json:"collectors"`
	Redactions Redactions  `yaml:"redactions,omitempty" json:"redactions,omitempty"`
}

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type Limits struct {
	MaxFiles      int      `yaml:"maxFiles" json:"maxFiles"`
	MaxTotalBytes int64    `yaml:"maxTotalBytes" json:"maxTotalBytes"`
	MaxDuration   Duration `yaml:"maxDuration" json:"maxDuration"`
}

type Root struct {
	ID   string `yaml:"id" json:"id"`
	Path string `yaml:"path" json:"path"`
}

type Collector struct {
	ID             string   `yaml:"id" json:"id"`
	Type           string   `yaml:"type" json:"type"`
	Root           string   `yaml:"root,omitempty" json:"root,omitempty"`
	Paths          []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	ArchivePrefix  string   `yaml:"archivePrefix,omitempty" json:"archivePrefix,omitempty"`
	MaxBytes       int64    `yaml:"maxBytes,omitempty" json:"maxBytes,omitempty"`
	Fields         []string `yaml:"fields,omitempty" json:"fields,omitempty"`
	Executable     string   `yaml:"executable,omitempty" json:"executable,omitempty"`
	Args           []string `yaml:"args,omitempty" json:"args,omitempty"`
	Timeout        Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	MaxOutputBytes int64    `yaml:"maxOutputBytes,omitempty" json:"maxOutputBytes,omitempty"`
}

type Redactions struct {
	Builtins []string     `yaml:"builtins,omitempty" json:"builtins,omitempty"`
	Custom   []CustomRule `yaml:"custom,omitempty" json:"custom,omitempty"`
}

type CustomRule struct {
	ID      string `yaml:"id" json:"id"`
	Pattern string `yaml:"pattern" json:"pattern"`
}
