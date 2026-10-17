package webfetch

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Render writes result to stdout: the content alone, or one line of JSON
// carrying the URL, the method, and the content. In plain mode the note that
// names the branch goes to notes; a caller that wants silence points it at
// [io.Discard].
func Render(stdout, notes io.Writer, result Result, asJSON bool) error {
	if asJSON {
		return renderJSON(stdout, result)
	}
	if _, err := io.WriteString(stdout, endWithBreak(result.Content)); err != nil {
		return fmt.Errorf("write content: %w", err)
	}
	if _, err := fmt.Fprintf(notes, "↳ %s — %s\n", result.URL, result.Method); err != nil {
		return fmt.Errorf("write note: %w", err)
	}
	return nil
}

// contentItem is the payload of `-json`, one line of JSON.
type contentItem struct {
	URL     string `json:"url"`
	Method  string `json:"method"`
	Content string `json:"content"`
}

// renderJSON writes the result as one line of JSON, with the content escaped
// onto that line.
func renderJSON(stdout io.Writer, result Result) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	item := contentItem(result)
	if err := encoder.Encode(item); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

// endWithBreak returns text with exactly one line break at its end.
func endWithBreak(text string) string {
	return strings.TrimRight(text, "\n") + "\n"
}
