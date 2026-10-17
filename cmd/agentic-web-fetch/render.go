package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"agents.kafo.dev/tools/pkg/webfetch"
)

// contentItem is the payload of `-json`, one line of JSON.
type contentItem struct {
	URL     string `json:"url"`
	Method  string `json:"method"`
	Content string `json:"content"`
}

// render writes the result to stdout: the content alone, or one line of JSON
// carrying the URL, the method, and the content.
func render(stdout io.Writer, result webfetch.Result, asJSON bool) error {
	if asJSON {
		return renderJSON(stdout, result)
	}
	if _, err := io.WriteString(stdout, endWithBreak(result.Content)); err != nil {
		return fmt.Errorf("write content: %w", err)
	}
	return nil
}

// renderJSON writes the result as one line of JSON, with the content escaped
// onto that line.
func renderJSON(stdout io.Writer, result webfetch.Result) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(contentItem(result)); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

// endWithBreak returns text with exactly one line break at its end.
func endWithBreak(text string) string {
	return strings.TrimRight(text, "\n") + "\n"
}
