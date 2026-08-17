// Package report defines sanitized collection results shared by bundle and inspect.
package report

type Entry struct {
	Name       string         `json:"name"`
	Data       []byte         `json:"-"`
	SHA256     string         `json:"sha256,omitempty"`
	Size       int64          `json:"size"`
	Collector  string         `json:"collector,omitempty"`
	Truncated  bool           `json:"truncated,omitempty"`
	Redactions map[string]int `json:"redactions,omitempty"`
}

type Error struct {
	Collector string `json:"collector"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type Result struct {
	Entries []Entry `json:"entries"`
	Errors  []Error `json:"errors,omitempty"`
}
