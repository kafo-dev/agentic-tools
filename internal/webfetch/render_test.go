package webfetch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// sample is one result, as the renderer receives it.
var sample = Result{
	URL:     "https://example.com/guide",
	Method:  MethodVariant,
	Content: "# Guide\n\nBody text.\n",
}

func TestRenderWritesTheContentAlone(t *testing.T) {
	t.Parallel()

	stdout, notes := &bytes.Buffer{}, &bytes.Buffer{}

	if err := Render(stdout, notes, sample, false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := stdout.String(); got != sample.Content {
		t.Errorf("stdout = %q, want %q", got, sample.Content)
	}
	if !strings.Contains(notes.String(), sample.URL) {
		t.Errorf("notes = %q, want the URL", notes.String())
	}
	if !strings.Contains(notes.String(), sample.Method) {
		t.Errorf("notes = %q, want the method", notes.String())
	}
}

func TestRenderEndsTheContentWithOneBreak(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	unterminated := Result{URL: sample.URL, Method: sample.Method, Content: "# Guide\n\n\n"}

	if err := Render(stdout, &bytes.Buffer{}, unterminated, false); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got, want := stdout.String(), "# Guide\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRenderWritesOneLineOfJSON(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}

	if err := Render(stdout, &bytes.Buffer{}, sample, true); err != nil {
		t.Fatalf("Render: %v", err)
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
