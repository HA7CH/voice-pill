package format

import (
	"fmt"
	"strings"
)

// FormatVTT returns WebVTT subtitle format.
func FormatVTT(r *Result) string {
	segments := r.Segments
	if len(segments) == 0 {
		segments = []Segment{{
			ID:    0,
			Start: 0,
			End:   r.Duration,
			Text:  r.Text,
		}}
	}

	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, seg := range segments {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%s --> %s\n", formatVTTTime(seg.Start), formatVTTTime(seg.End)))
		b.WriteString(seg.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func formatVTTTime(seconds float64) string {
	total := int(seconds * 1000)
	ms := total % 1000
	total /= 1000
	s := total % 60
	total /= 60
	m := total % 60
	h := total / 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
