package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"agents.kafo.dev/tools/pkg/webfetch"
)

// sample is one result, as the renderer receives it.
var sample = webfetch.Result{
	URL:     "https://example.com/guide",
	Method:  webfetch.MethodVariant,
	Content: "# Guide\n\nBody text.\n",
}

func TestRenderWritesTheContentAlone(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	if err := render(stdout, sample, false); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := stdout.String(); got != sample.Content {
		t.Errorf("stdout = %q, want %q", got, sample.Content)
	}
}

func TestRenderEndsTheContentWithOneBreak(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	unterminated := webfetch.Result{URL: sample.URL, Method: sample.Method, Content: "# Guide\n\n\n"}

	if err := render(stdout, unterminated, false); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got, want := stdout.String(), "# Guide\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRenderWritesOneLineOfJSON(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	if err := render(stdout, sample, true); err != nil {
		t.Fatalf("render: %v", err)
	}
	line := strings.TrimSuffix(stdout.String(), "\n")
	if strings.ContainsRune(line, '\n') {
		t.Fatalf("stdout holds more than one line: %q", stdout.String())
	}
	var item struct {
		URL     string `json:"url"`
		Method  string `json:"method"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if item.URL != sample.URL {
		t.Errorf("url = %q, want %q", item.URL, sample.URL)
	}
	if item.Method != sample.Method {
		t.Errorf("method = %q, want %q", item.Method, sample.Method)
	}
	if item.Content != sample.Content {
		t.Errorf("content = %q, want %q", item.Content, sample.Content)
	}
}
