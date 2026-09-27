package format

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleResult() *Result {
	return &Result{
		Text:     "你好世界",
		Language: "zh",
		Duration: 2.5,
		Segments: []Segment{
			{ID: 0, Start: 0, End: 2.5, Text: "你好世界"},
		},
	}
}

func TestFormatText(t *testing.T) {
	output, contentType, err := Format(sampleResult(), "text")
	require.NoError(t, err)
	assert.Equal(t, "你好世界", output)
	assert.Equal(t, "text/plain", contentType)
}

func TestFormatJSON(t *testing.T) {
	output, contentType, err := Format(sampleResult(), "json")
	require.NoError(t, err)
	assert.Contains(t, output, `"text":"你好世界"`)
	assert.Equal(t, "application/json", contentType)
}

func TestFormatVerboseJSON(t *testing.T) {
	output, contentType, err := Format(sampleResult(), "verbose_json")
	require.NoError(t, err)
	assert.Contains(t, output, `"task": "transcribe"`)
	assert.Contains(t, output, `"duration": 2.5`)
	assert.Contains(t, output, `"segments"`)
	assert.Equal(t, "application/json", contentType)
}

func TestFormatSRT(t *testing.T) {
	output, _, err := Format(sampleResult(), "srt")
	require.NoError(t, err)
	assert.Contains(t, output, "1\n")
	assert.Contains(t, output, "00:00:00,000 --> 00:00:02,500")
	assert.Contains(t, output, "你好世界")
}

func TestFormatVTT(t *testing.T) {
	output, _, err := Format(sampleResult(), "vtt")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(output, "WEBVTT"))
	assert.Contains(t, output, "00:00:00.000 --> 00:00:02.500")
	assert.Contains(t, output, "你好世界")
}

func TestFormatUnsupported(t *testing.T) {
	_, _, err := Format(sampleResult(), "xml")
	assert.Error(t, err)
}

func TestFormatSRTTime(t *testing.T) {
	assert.Equal(t, "00:00:00,000", formatSRTTime(0))
	assert.Equal(t, "00:00:01,500", formatSRTTime(1.5))
	assert.Equal(t, "01:02:03,456", formatSRTTime(3723.456))
}

func TestFormatVTTTime(t *testing.T) {
	assert.Equal(t, "00:00:00.000", formatVTTTime(0))
	assert.Equal(t, "00:00:01.500", formatVTTTime(1.5))
	assert.Equal(t, "01:02:03.456", formatVTTTime(3723.456))
}
