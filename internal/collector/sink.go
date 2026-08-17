package collector

import (
	"fmt"
	"strings"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/policy"
	"github.com/tjacky87-lab/diagsift/internal/report"
)

type Sink struct {
	limits           policy.Limits
	result           report.Result
	names            map[string]struct{}
	total            int64
	errorsSuppressed bool
}

func NewSink(limits policy.Limits) *Sink {
	return &Sink{limits: limits, names: make(map[string]struct{})}
}

func (s *Sink) Add(entry report.Entry) error {
	name, err := manifest.SafeArchivePath(entry.Name)
	if err != nil || name == "" {
		return fmt.Errorf("unsafe archive entry name")
	}
	key := strings.ToLower(name)
	if _, exists := s.names[key]; exists {
		return fmt.Errorf("archive entry name collision")
	}
	if len(s.result.Entries) >= s.limits.MaxFiles {
		return fmt.Errorf("file count limit reached")
	}
	if int64(len(entry.Data)) > s.limits.MaxTotalBytes-s.total {
		return fmt.Errorf("total byte limit reached")
	}
	entry.Name = name
	entry.Size = int64(len(entry.Data))
	entry.Data = append([]byte(nil), entry.Data...)
	s.names[key] = struct{}{}
	s.total += entry.Size
	s.result.Entries = append(s.result.Entries, entry)
	return nil
}

func (s *Sink) Error(collector, code, message string) {
	if s.errorsSuppressed {
		return
	}
	if len(s.result.Errors) >= policy.HardMaxRecordedErrors-1 {
		s.result.Errors = append(s.result.Errors, report.Error{
			Collector: "diagsift",
			Code:      "errors-suppressed",
			Message:   "collector error limit reached; additional errors suppressed",
		})
		s.errorsSuppressed = true
		return
	}
	s.result.Errors = append(s.result.Errors, report.Error{Collector: collector, Code: code, Message: message})
}

func (s *Sink) Result() report.Result {
	return s.result
}
