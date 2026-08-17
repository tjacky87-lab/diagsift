// Package workspace manages private temporary staging for already-sanitized data.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/report"
)

type Workspace struct {
	path string
}

func New() (*Workspace, error) {
	path, err := os.MkdirTemp("", "diagsift-")
	if err != nil {
		return nil, fmt.Errorf("create private workspace: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		_ = os.RemoveAll(path)
		return nil, fmt.Errorf("protect private workspace: %w", err)
	}
	return &Workspace{path: path}, nil
}

func (w *Workspace) Stage(entries []report.Entry) error {
	for _, entry := range entries {
		name, err := manifest.SafeArchivePath(entry.Name)
		if err != nil || name == "" {
			return fmt.Errorf("refuse unsafe staged entry")
		}
		target := filepath.Join(w.path, filepath.FromSlash(name))
		rel, err := filepath.Rel(w.path, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("refuse workspace escape")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("create staging directory: %w", err)
		}
		if err := os.WriteFile(target, entry.Data, 0o600); err != nil {
			return fmt.Errorf("stage sanitized entry: %w", err)
		}
	}
	return nil
}

func (w *Workspace) Open(name string) (*os.File, error) {
	safe, err := manifest.SafeArchivePath(name)
	if err != nil || safe == "" {
		return nil, fmt.Errorf("refuse unsafe staged entry")
	}
	return os.Open(filepath.Join(w.path, filepath.FromSlash(safe)))
}

func (w *Workspace) Remove() error {
	if w == nil || w.path == "" {
		return nil
	}
	return os.RemoveAll(w.path)
}
