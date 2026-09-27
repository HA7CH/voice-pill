package format

import (
	"fmt"
	"strings"
)

// Segment represents a recognized speech segment with timing.
type Segment struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Result holds the final transcription result.
type Result struct {
	Text     string    `json:"text"`
	Language string    `json:"language,omitempty"`
	Duration float64   `json:"duration,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

// Format renders a Result in the specified format.
func Format(r *Result, format string) (string, string, error) {
	switch strings.ToLower(format) {
	case "text":
		return r.Text, "text/plain", nil
	case "json":
		return FormatJSON(r), "application/json", nil
	case "verbose_json":
		return FormatVerboseJSON(r), "application/json", nil
	case "srt":
		return FormatSRT(r), "text/plain", nil
	case "vtt":
		return FormatVTT(r), "text/plain", nil
	default:
		return "", "", fmt.Errorf("unsupported format: %s", format)
	}
}
