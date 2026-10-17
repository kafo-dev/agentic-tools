package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// wantUsage is the heading the help text opens with, and markdownBody the page
// the stub server serves.
const (
	wantUsage    = "Usage of " + command + ":"
	markdownBody = "# Guide\n"
)

// markdownServer serves one markdown page, whatever the request asks for.
func markdownServer(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(markdownBody))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func TestDoFetchesTheURLFromStdin(t *testing.T) {
	t.Parallel()

	base := markdownServer(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := do(nil, strings.NewReader(base+"/guide\n"), stdout, stderr)

	if code != exitOK {
		t.Fatalf("do returned %d, want %d; stderr: %s", code, exitOK, stderr)
	}
	if got, want := stdout.String(), markdownBody; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if !strings.Contains(stderr.String(), "negotiated") {
		t.Errorf("stderr = %q, want the branch", stderr)
	}
}

func TestDoFetchesTheURLFromTheArgument(t *testing.T) {
	t.Parallel()

	base := markdownServer(t)
	stdout := &bytes.Buffer{}

	code := do([]string{base + "/guide"}, nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	if got, want := stdout.String(), markdownBody; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestDoPrintsJSON(t *testing.T) {
	t.Parallel()

	base := markdownServer(t)
	stdout := &bytes.Buffer{}

	code := do([]string{"-json", base + "/guide"}, nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	line := strings.TrimSuffix(stdout.String(), "\n")
	var item struct {
		URL     string `json:"url"`
		Method  string `json:"method"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		t.Fatalf("stdout is not one line of JSON: %v", err)
	}
	if item.URL != base+"/guide" || item.Method != "negotiated" || item.Content != markdownBody {
		t.Errorf("item = %+v", item)
	}
}

func TestDoQuietDropsTheNote(t *testing.T) {
	t.Parallel()

	base := markdownServer(t)
	stderr := &bytes.Buffer{}

	code := do([]string{"-q", base + "/guide"}, nil, &bytes.Buffer{}, stderr)

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestDoReportsAFailedFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := do([]string{server.URL}, nil, stdout, stderr)

	if code != exitFail {
		t.Fatalf("do returned %d, want %d", code, exitFail)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr.String(), "500") {
		t.Errorf("stderr = %q, want the status code", stderr)
	}
}

func TestDoRejectsMalformedInvocations(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"unknown flag":    {"-verbose", "example.com"},
		"two URLs":        {"example.com", "example.net"},
		"no timeout":      {"-t", "0s", "example.com"},
		"unknown timeout": {"-t", "soon", "example.com"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			code := do(args, nil, stdout, stderr)

			if code != exitUsage {
				t.Fatalf("do(%q) returned %d, want %d", args, code, exitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr.String(), wantUsage) {
				t.Errorf("stderr = %q, want the usage text", stderr)
			}
		})
	}
}

func TestDoRejectsAnEmptyStdin(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := do(nil, strings.NewReader("\n\t \n"), stdout, stderr)

	if code != exitUsage {
		t.Fatalf("do returned %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), wantUsage) {
		t.Errorf("stderr = %q, want the usage text", stderr)
	}
}

func TestDoRejectsAnOversizedStdin(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	input := strings.Repeat("a", urlBytes+1)

	code := do(nil, strings.NewReader(input), stdout, stderr)

	if code != exitUsage {
		t.Fatalf("do returned %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "standard input") {
		t.Errorf("stderr = %q, want the standard input error", stderr)
	}
}

func TestDoPrintsHelp(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	code := do([]string{"-h"}, nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), wantUsage) {
		t.Errorf("stdout = %q, want the usage text", stdout)
	}
	if !strings.Contains(stdout.String(), "positional argument") {
		t.Errorf("stdout = %q, want the notes", stdout)
	}
}

func TestDoPrintsSkillWithoutURL(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	code := do([]string{"-skill"}, nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	document := stdout.String()
	if !strings.HasPrefix(document, "---\nname: agentic-web-fetch\n") {
		t.Errorf("stdout does not start with the skill frontmatter: %q", document)
	}
	if !strings.Contains(document, wantUsage) {
		t.Errorf("stdout does not carry the usage line: %q", document)
	}
	if strings.Contains(document, "{{usage}}") {
		t.Errorf("stdout left the placeholder in place: %q", document)
	}
}

func TestParseTakesABareAddress(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"example.com/docs": "https://example.com/docs",
		"https://broken":   "https://broken",
		"http://plain":     "http://plain",
	}
	for argument, want := range tests {
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			opts, err := parse([]string{argument}, nil)
			if err != nil {
				t.Fatalf("parse(%q): %v", argument, err)
			}
			if opts.target != want {
				t.Errorf("target = %q, want %q", opts.target, want)
			}
		})
	}
}
