package asr

// TranscriptAggregator merges ASR interim, definite, and final results
// into a single transcript. Handles VAD segment resets by detecting
// when final text suddenly shortens (new segment started).
//
// Ported from koe's Rust implementation (koe-asr/src/transcript.rs).
type TranscriptAggregator struct {
	interimText  string
	definiteText string
	finalText    string
	hasFinal     bool
	hasDefinite  bool

	// interimHistory tracks interim revisions for debugging.
	interimHistory []string
}

// NewAggregator creates a new TranscriptAggregator.
func NewAggregator() *TranscriptAggregator {
	return &TranscriptAggregator{}
}

// UpdateInterim records a new interim (partial) result.
func (a *TranscriptAggregator) UpdateInterim(text string) {
	if text == "" {
		return
	}
	if len(a.interimHistory) == 0 || a.interimHistory[len(a.interimHistory)-1] != text {
		a.interimHistory = append(a.interimHistory, text)
	}
	a.interimText = text
}

// UpdateDefinite records a second-pass confirmed result.
func (a *TranscriptAggregator) UpdateDefinite(text string) {
	if text == "" {
		return
	}
	a.hasDefinite = true
	a.definiteText = text
}

// UpdateFinal records a final (third-pass or VAD-end) result.
// Uses overlap detection to correctly merge multiple segments.
func (a *TranscriptAggregator) UpdateFinal(text string) {
	a.hasFinal = true
	if text == "" {
		return
	}

	if a.finalText == "" {
		a.finalText = text
	} else if len(text) >= len(a.finalText) && text[:len(a.finalText)] == a.finalText {
		// New final extends or replaces the current (same utterance refresh)
		a.finalText = text
	} else if len(a.finalText) >= len(text) && a.finalText[:len(text)] == text {
		// Stale replay of earlier content — ignore
		return
	} else {
		// New segment: strip overlap between existing tail and incoming head
		overlap := longestOverlap(a.finalText, text)
		a.finalText += text[overlap:]
	}

	// Clear interim since its segment is now finalized
	a.interimText = ""
}

// BestText returns the best available text.
// Priority: final > definite > interim.
func (a *TranscriptAggregator) BestText() string {
	if a.hasFinal && a.finalText != "" {
		return a.finalText
	}
	if a.hasDefinite && a.definiteText != "" {
		return a.definiteText
	}
	return a.interimText
}

// LivePreview returns a merged view of confirmed final text + current interim.
func (a *TranscriptAggregator) LivePreview() string {
	if a.finalText == "" {
		return a.interimText
	}
	if a.interimText == "" {
		return a.finalText
	}
	// Check if one contains the other
	if len(a.interimText) >= len(a.finalText) && a.interimText[:len(a.finalText)] == a.finalText {
		return a.interimText
	}
	if len(a.finalText) >= len(a.interimText) && a.finalText[:len(a.interimText)] == a.interimText {
		return a.finalText
	}
	// Merge with overlap detection
	overlap := longestOverlap(a.finalText, a.interimText)
	return a.finalText + a.interimText[overlap:]
}

// HasFinal returns true if at least one final result has been received.
func (a *TranscriptAggregator) HasFinal() bool {
	return a.hasFinal
}

// HasAnyText returns true if any text has been accumulated.
func (a *TranscriptAggregator) HasAnyText() bool {
	return a.finalText != "" || a.definiteText != "" || a.interimText != ""
}

// InterimHistory returns the last maxEntries interim revisions.
func (a *TranscriptAggregator) InterimHistory(maxEntries int) []string {
	if len(a.interimHistory) <= maxEntries {
		return a.interimHistory
	}
	return a.interimHistory[len(a.interimHistory)-maxEntries:]
}

// longestOverlap finds the longest k where tail ends with head[:k].
func longestOverlap(tail, head string) int {
	maxLen := len(tail)
	if len(head) < maxLen {
		maxLen = len(head)
	}
	// Check from longest to shortest overlap
	for k := maxLen; k > 0; k-- {
		if tail[len(tail)-k:] == head[:k] {
			return k
		}
	}
	return 0
}
