package websearch

import (
	"encoding/json"
	"io"
	"strings"

	"agentic.kafo.dev/tools/pkg/exa"
)

const (
	// summaryChars caps a summary; a longer one pushes the URLs off screen.
	summaryChars = 300
	// summaryJoiner separates the lines of a folded summary.
	summaryJoiner = "; "
)

// item is one result as `-json` renders it.
type item struct {
	URL     string `json:"url"`
	Summary string `json:"summary"`
}

// render writes the results to w, as tab-separated lines or as one line of
// JSON.
func render(w io.Writer, results []exa.Result, asJSON bool) error {
	if asJSON {
		items := make([]item, 0, len(results))
		for _, result := range results {
			items = append(items, item{URL: result.URL, Summary: summarize(result)})
		}
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(items)
	}

	var out strings.Builder
	for _, result := range results {
		out.WriteString(result.URL)
		out.WriteByte('\t')
		out.WriteString(summarize(result))
		out.WriteByte('\n')
	}
	_, err := io.WriteString(w, out.String())
	return err
}

// summarize picks the best one-line description of a result: Exa's summary,
// the title, or the first line of the page text.
func summarize(result exa.Result) string {
	for _, candidate := range []string{result.Summary, result.Title, firstLine(result.Text)} {
		if line := oneLine(candidate); line != "" {
			return line
		}
	}
	return ""
}

// firstLine returns the text before the first newline.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}

// oneLine folds Exa's multi-line summary into a single line.
func oneLine(text string) string {
	text = strings.TrimPrefix(strings.TrimSpace(text), "Summary:")
	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*• \t"))
		if line != "" {
			parts = append(parts, line)
		}
	}
	return truncate(strings.Join(strings.Fields(strings.Join(parts, summaryJoiner)), " "))
}

// truncate cuts s to limit runes, marking the cut.
func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= summaryChars {
		return s
	}
	return strings.TrimRight(string(runes[:summaryChars]), " ") + "…"
}
