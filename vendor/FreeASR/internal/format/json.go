package format

import (
	"bytes"
	"encoding/json"
)

// FormatJSON returns OpenAI-compatible JSON output.
func FormatJSON(r *Result) string {
	out := struct {
		Text string `json:"text"`
	}{Text: r.Text}
	return marshalNoEscape(out)
}

// FormatVerboseJSON returns OpenAI-compatible verbose JSON output.
func FormatVerboseJSON(r *Result) string {
	out := struct {
		Task     string    `json:"task"`
		Language string    `json:"language"`
		Duration float64   `json:"duration"`
		Text     string    `json:"text"`
		Segments []Segment `json:"segments"`
	}{
		Task:     "transcribe",
		Language: r.Language,
		Duration: r.Duration,
		Text:     r.Text,
		Segments: r.Segments,
	}
	if out.Language == "" {
		out.Language = "zh"
	}
	if out.Segments == nil {
		out.Segments = []Segment{}
	}
	return marshalIndentNoEscape(out)
}

func marshalNoEscape(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	// Encode adds a trailing newline, trim it
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return string(b)
}

func marshalIndentNoEscape(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return string(b)
}
