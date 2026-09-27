package asr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregator_BasicInterim(t *testing.T) {
	a := NewAggregator()

	a.UpdateInterim("你")
	assert.Equal(t, "你", a.BestText())
	assert.Equal(t, "你", a.LivePreview())

	a.UpdateInterim("你好")
	assert.Equal(t, "你好", a.BestText())
}

func TestAggregator_BasicFinal(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("你好世界")
	assert.Equal(t, "你好世界", a.BestText())
	assert.True(t, a.HasFinal())
}

func TestAggregator_InterimThenFinal(t *testing.T) {
	a := NewAggregator()
	a.UpdateInterim("你好")
	a.UpdateInterim("你好世界")
	a.UpdateFinal("你好世界。")
	assert.Equal(t, "你好世界。", a.BestText())
}

func TestAggregator_DefinitePriority(t *testing.T) {
	a := NewAggregator()
	a.UpdateInterim("你好")
	a.UpdateDefinite("你好世界")
	// definite > interim
	assert.Equal(t, "你好世界", a.BestText())
}

func TestAggregator_FinalOverDefinite(t *testing.T) {
	a := NewAggregator()
	a.UpdateDefinite("你好世界")
	a.UpdateFinal("你好世界。")
	assert.Equal(t, "你好世界。", a.BestText())
}

func TestAggregator_FinalSameUtteranceRefresh(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("你好")
	a.UpdateFinal("你好世界")
	assert.Equal(t, "你好世界", a.BestText())
}

func TestAggregator_FinalStaleReplay(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("你好世界")
	a.UpdateFinal("你好") // stale, shorter prefix — should be ignored
	assert.Equal(t, "你好世界", a.BestText())
}

func TestAggregator_FinalNewSegment(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("第一句话。")
	a.UpdateFinal("第二句话。") // new segment, no overlap
	assert.Equal(t, "第一句话。第二句话。", a.BestText())
}

func TestAggregator_FinalSegmentOverlap(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("你好世界")
	a.UpdateFinal("世界真美") // overlaps on "世界"
	assert.Equal(t, "你好世界真美", a.BestText())
}

func TestAggregator_LivePreviewMerge(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("第一句。")
	a.UpdateInterim("第二句")
	assert.Equal(t, "第一句。第二句", a.LivePreview())
}

func TestAggregator_LivePreviewInterimExtendsFinal(t *testing.T) {
	a := NewAggregator()
	a.UpdateFinal("你好")
	a.UpdateInterim("你好世界")
	assert.Equal(t, "你好世界", a.LivePreview())
}

func TestAggregator_EmptyTextIgnored(t *testing.T) {
	a := NewAggregator()
	a.UpdateInterim("")
	a.UpdateFinal("")
	assert.Equal(t, "", a.BestText())
	assert.False(t, a.HasAnyText())
}

func TestAggregator_InterimHistory(t *testing.T) {
	a := NewAggregator()
	a.UpdateInterim("a")
	a.UpdateInterim("ab")
	a.UpdateInterim("abc")
	a.UpdateInterim("abc") // duplicate should not be added

	history := a.InterimHistory(10)
	assert.Equal(t, []string{"a", "ab", "abc"}, history)
}

func TestAggregator_InterimHistoryTruncation(t *testing.T) {
	a := NewAggregator()
	for i := 0; i < 20; i++ {
		a.UpdateInterim(string(rune('A' + i)))
	}
	history := a.InterimHistory(5)
	assert.Len(t, history, 5)
}

func TestLongestOverlap(t *testing.T) {
	tests := []struct {
		tail     string
		head     string
		expected int
	}{
		{"hello", "hello world", 5},
		{"abc", "def", 0},
		{"你好世界", "世界真美", 6}, // "世界" is 6 bytes in UTF-8
		{"", "abc", 0},
		{"abc", "", 0},
		{"abcdef", "defghi", 3},
	}

	for _, tt := range tests {
		t.Run(tt.tail+"_"+tt.head, func(t *testing.T) {
			result := longestOverlap(tt.tail, tt.head)
			assert.Equal(t, tt.expected, result)
		})
	}
}
