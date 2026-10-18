package websearch

import (
	"strings"
	"testing"

	"agentic.kafo.dev/tools/pkg/exa"
)

func TestSummarize(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result exa.Result
		want   string
	}{
		"summary folded onto one line": {
			result: exa.Result{Summary: "Summary:\n- One fact.\n- Another fact."},
			want:   "One fact.; Another fact.",
		},
		"title when no summary": {
			result: exa.Result{Title: "A title", Text: "text"},
			want:   "A title",
		},
		"page text when nothing else": {
			result: exa.Result{Text: "First line.\nSecond line."},
			want:   "First line.",
		},
		"tabs removed": {
			result: exa.Result{Summary: "a\tb"},
			want:   "a b",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := summarize(test.result); got != test.want {
				t.Errorf("summarize(%+v) = %q, want %q", test.result, got, test.want)
			}
		})
	}
}

func TestSummarizeTruncates(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("word ", summaryChars)
	got := summarize(exa.Result{Summary: long})

	if len([]rune(got)) > summaryChars+1 {
		t.Errorf("summary holds %d runes, want at most %d", len([]rune(got)), summaryChars+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("summary does not mark the cut: %q", got)
	}
}
