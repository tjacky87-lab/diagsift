// Package collector implements bounded local diagnostic collectors.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/plan"
	"github.com/tjacky87-lab/diagsift/internal/platform"
	"github.com/tjacky87-lab/diagsift/internal/policy"
	"github.com/tjacky87-lab/diagsift/internal/report"
)

type Sanitizer interface {
	Sanitize([]byte) ([]byte, map[string]int, error)
}

type IdentitySanitizer struct{}

func (IdentitySanitizer) Sanitize(input []byte) ([]byte, map[string]int, error) {
	return append([]byte(nil), input...), nil, nil
}

func CollectNonCommand(ctx context.Context, loaded manifest.Loaded, preview plan.Plan, sanitizer Sanitizer) report.Result {
	return collect(ctx, loaded, preview, sanitizer, false)
}

func Collect(ctx context.Context, loaded manifest.Loaded, preview plan.Plan, sanitizer Sanitizer) report.Result {
	return collect(ctx, loaded, preview, sanitizer, true)
}

func collect(ctx context.Context, loaded manifest.Loaded, preview plan.Plan, sanitizer Sanitizer, commands bool) report.Result {
	limits, _ := policy.EffectiveLimits(
		loaded.Manifest.Limits.MaxFiles,
		loaded.Manifest.Limits.MaxTotalBytes,
		loaded.Manifest.Limits.MaxDuration.Duration,
	)
	sink := NewSink(limits)
	roots := make(map[string]string, len(preview.Roots))
	for _, root := range preview.Roots {
		roots[root.ID] = root.Path
	}
	for _, spec := range loaded.Manifest.Collectors {
		if err := ctx.Err(); err != nil {
			sink.Error(spec.ID, "deadline", "global collection deadline reached")
			break
		}
		switch spec.Type {
		case "file":
			collectFiles(ctx, sink, spec, roots[spec.Root], sanitizer)
		case "system":
			collectSystem(sink, spec, sanitizer)
		case "command":
			if commands {
				collectCommand(ctx, sink, spec, sanitizer)
			} else {
				sink.Error(spec.ID, "unavailable", "command collector is unavailable")
			}
		}
	}
	return sink.Result()
}

func collectFiles(ctx context.Context, sink *Sink, spec manifest.Collector, root string, sanitizer Sanitizer) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		sink.Error(spec.ID, "root-unavailable", "collection root is unavailable")
		return
	}
	if platform.IsLinkOrReparse(root, rootInfo) {
		sink.Error(spec.ID, "link-skipped", "collection root is a link or reparse point")
		return
	}
	if !rootInfo.IsDir() {
		sink.Error(spec.ID, "root-unavailable", "collection root is not a directory")
		return
	}
	for _, relative := range spec.Paths {
		if ctx.Err() != nil {
			sink.Error(spec.ID, "deadline", "global collection deadline reached")
			return
		}
		clean, err := manifest.SafeRelativePath(relative)
		if err != nil {
			sink.Error(spec.ID, "unsafe-path", "collector path failed safety validation")
			continue
		}
		target := filepath.Join(root, clean)
		if !within(root, target) {
			sink.Error(spec.ID, "root-escape", "collector path escaped its root")
			continue
		}
		info, err := os.Lstat(target)
		if err != nil {
			sink.Error(spec.ID, "missing", "configured path is unavailable")
			continue
		}
		if err := rejectLinkComponents(root, target); err != nil {
			sink.Error(spec.ID, "link-skipped", "collector path contains a link or reparse point")
			continue
		}
		if info.IsDir() {
			sink.Error(spec.ID, "directory-unsupported", "configured path is a directory; only explicit regular files are supported")
			continue
		}
		collectFile(sink, spec, root, target, info, sanitizer)
	}
}

func collectFile(sink *Sink, spec manifest.Collector, root, path string, info os.FileInfo, sanitizer Sanitizer) {
	if !info.Mode().IsRegular() {
		sink.Error(spec.ID, "unsupported", "non-regular file skipped")
		return
	}
	if platform.IsLinkOrReparse(path, info) {
		sink.Error(spec.ID, "link-skipped", "link or reparse point skipped")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		sink.Error(spec.ID, "permission", "configured file could not be opened")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, spec.MaxBytes+1))
	if err != nil {
		sink.Error(spec.ID, "read", "configured file could not be read")
		return
	}
	truncated := int64(len(data)) > spec.MaxBytes
	if truncated {
		data = data[:spec.MaxBytes]
	}
	if !isText(data) {
		sink.Error(spec.ID, "binary-skipped", "binary or unsupported encoding skipped")
		return
	}
	clean, counts, err := sanitizer.Sanitize(data)
	if err != nil {
		sink.Error(spec.ID, "redaction", "text sanitization failed")
		return
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		sink.Error(spec.ID, "root-escape", "configured file escaped its root")
		return
	}
	archive := "collectors/" + spec.ID + "/"
	if spec.ArchivePrefix != "" {
		archive += spec.ArchivePrefix + "/"
	}
	archive += filepath.ToSlash(relative)
	if err := sink.Add(report.Entry{Name: archive, Data: clean, Collector: spec.ID, Truncated: truncated, Redactions: counts}); err != nil {
		sink.Error(spec.ID, "limit", "bounded collection sink rejected an entry")
	}
}

func collectSystem(sink *Sink, spec manifest.Collector, sanitizer Sanitizer) {
	values := make(map[string]string, len(spec.Fields))
	for _, field := range spec.Fields {
		switch field {
		case "os":
			values[field] = runtime.GOOS
		case "arch":
			values[field] = runtime.GOARCH
		}
	}
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		sink.Error(spec.ID, "encode", "system information could not be encoded")
		return
	}
	data = append(data, '\n')
	clean, counts, err := sanitizer.Sanitize(data)
	if err != nil {
		sink.Error(spec.ID, "redaction", "text sanitization failed")
		return
	}
	if err := sink.Add(report.Entry{Name: "collectors/" + spec.ID + "/system.json", Data: clean, Collector: spec.ID, Redactions: counts}); err != nil {
		sink.Error(spec.ID, "limit", "bounded collection sink rejected an entry")
	}
}

func rejectLinkComponents(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if platform.IsLinkOrReparse(current, info) {
			return fmt.Errorf("link component")
		}
	}
	return nil
}

func within(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isText(data []byte) bool {
	return !strings.ContainsRune(string(data), '\x00') && utf8.Valid(data)
}
