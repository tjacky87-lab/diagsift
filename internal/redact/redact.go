// Package redact removes configured high-risk text patterns before durable output.
// Redaction reduces risk; it is not proof that content is safe to share.
package redact

import (
	"fmt"
	"regexp"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
)

type rule struct {
	id      string
	pattern *regexp.Regexp
}

type Redactor struct {
	rules []rule
}

var builtinPatterns = map[string]string{
	"private-keys":       `(?s)-----BEGIN(?: [A-Z0-9]+)? PRIVATE KEY-----.*?-----END(?: [A-Z0-9]+)? PRIVATE KEY-----`,
	"bearer-tokens":      `(?i)\bBearer[ \t]+[A-Za-z0-9._~+/=-]{8,}`,
	"credentials":        `(?i)\b(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key)[ \t]*[:=][ \t]*[^\s,;]{4,}`,
	"url-credentials":    `(?i)\b[a-z][a-z0-9+.-]*://[^\s/:@]+:[^\s/@]+@`,
	"connection-strings": `(?i)\b(?:password|pwd|user[ _]?id|uid)[ \t]*=[ \t]*[^;\r\n]{1,256}`,
	"paths":              `(?i)(?:[A-Z]:\\Users\\[^\\\s]+|/(?:home|Users)/[^/\s]+)`,
}

func New(config manifest.Redactions) (*Redactor, error) {
	result := &Redactor{}
	for _, id := range config.Builtins {
		pattern, exists := builtinPatterns[id]
		if !exists {
			return nil, fmt.Errorf("unknown built-in redaction %q", id)
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compile built-in redaction %q", id)
		}
		result.rules = append(result.rules, rule{id: id, pattern: compiled})
	}
	for _, custom := range config.Custom {
		compiled, err := regexp.Compile(custom.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compile custom redaction %q", custom.ID)
		}
		result.rules = append(result.rules, rule{id: custom.ID, pattern: compiled})
	}
	return result, nil
}

func (r *Redactor) Sanitize(input []byte) ([]byte, map[string]int, error) {
	output := append([]byte(nil), input...)
	counts := make(map[string]int)
	for _, current := range r.rules {
		matches := current.pattern.FindAllIndex(output, -1)
		if len(matches) == 0 {
			continue
		}
		counts[current.id] += len(matches)
		replacement := []byte("[REDACTED:" + current.id + "]")
		output = current.pattern.ReplaceAllLiteral(output, replacement)
	}
	if len(counts) == 0 {
		return output, nil, nil
	}
	return output, counts, nil
}

// Stream buffers bounded collector input in memory so matches split across any
// write boundary are evaluated as one text stream before durable output.
type Stream struct {
	redactor *Redactor
	data     []byte
	limit    int64
	overflow bool
}

func (r *Redactor) Stream(limit int64) *Stream {
	return &Stream{redactor: r, limit: limit}
}

func (s *Stream) Write(data []byte) (int, error) {
	original := len(data)
	remaining := s.limit - int64(len(s.data))
	if remaining <= 0 {
		s.overflow = true
		return original, nil
	}
	if int64(len(data)) > remaining {
		data = data[:remaining]
		s.overflow = true
	}
	s.data = append(s.data, data...)
	return original, nil
}

func (s *Stream) Finish() ([]byte, map[string]int, bool, error) {
	output, counts, err := s.redactor.Sanitize(s.data)
	return output, counts, s.overflow, err
}
