package webfetch

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
)

// boilerplate lists the elements that never carry a page's own text.
var boilerplate = []string{
	"script", "style", "noscript", "template", "svg", "iframe", "canvas",
	"object", "embed", "dialog", "form", "button", "nav", "header", "footer",
	"aside",
}

// contentSelectors lists the elements that hold a page's own text, likely
// first.
var contentSelectors = []string{
	"article", "main", "[role=main]", "#main-content", "#content", ".markdown-body",
}

// parseHTML parses an HTML body into a document.
func parseHTML(body []byte) (*goquery.Document, error) {
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	return document, nil
}

// readableText returns the readable part of an HTML body as markdown, cut down
// by the rule's selectors when it names any.
func readableText(body []byte, rule Rule) (string, error) {
	document, err := parseHTML(body)
	if err != nil {
		return "", err
	}
	dropNoise(document, rule.Drop)
	content, ok := pickContent(document, rule.Select)
	if !ok {
		return "", errors.New("no readable part in the HTML")
	}
	converted, err := htmltomarkdown.ConvertString(content)
	if err != nil {
		return "", fmt.Errorf("convert HTML: %w", err)
	}
	return tidy(converted), nil
}

// dropNoise removes the page furniture, and the rule's own noise with it.
func dropNoise(document *goquery.Document, extra []string) {
	for _, selector := range append(slices.Clone(boilerplate), extra...) {
		document.Find(selector).Remove()
	}
}

// pickContent returns the inner HTML of the readable part: the element the
// rule names, then the elements sites use for their content, then the body.
func pickContent(document *goquery.Document, selector string) (string, bool) {
	if selector != "" {
		if html, ok := firstHTML(document, selector); ok {
			return html, true
		}
	}
	for _, candidate := range contentSelectors {
		if html, ok := firstHTML(document, candidate); ok {
			return html, true
		}
	}
	return firstHTML(document, "body")
}

// firstHTML returns the inner HTML of the first element a selector matches,
// and reports false when the selector matches nothing or matches emptiness.
func firstHTML(document *goquery.Document, selector string) (string, bool) {
	element := document.Find(selector).First()
	if element.Length() == 0 {
		return "", false
	}
	html, err := element.Html()
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(html) == "" {
		return "", false
	}
	return html, true
}

// tidy trims the trailing spaces and the runs of blank lines that conversion
// leaves, and ends the text with one line break.
func tidy(text string) string {
	var out strings.Builder
	blank := false
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if line == "" && blank {
			continue
		}
		blank = line == ""
		out.WriteString(line)
		out.WriteByte('\n')
	}
	trimmed := strings.Trim(out.String(), "\n")
	if trimmed == "" {
		return ""
	}
	return trimmed + "\n"
}
