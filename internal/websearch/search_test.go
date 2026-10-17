package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"agents.kafo.dev/tools/pkg/exa"
)

// fakeSearcher returns canned results, or a canned error.
type fakeSearcher struct {
	results []exa.Result
	err     error
	query   string
	limit   int
}

func (f *fakeSearcher) Search(_ context.Context, query string, limit int) ([]exa.Result, error) {
	f.query = query
	f.limit = limit
	return f.results, f.err
}

func TestRunPrintsTabSeparatedLines(t *testing.T) {
	t.Parallel()

	client := &fakeSearcher{results: []exa.Result{
		{URL: "https://go.dev/doc/go1.27", Summary: "Summary:\n- Generic methods land.\n- json/v2 arrives."},
		{URL: "https://go.dev/blog/go1.27", Title: "Go 1.27 is released"},
		{URL: "https://example.com/empty"},
	}}
	stdout := &bytes.Buffer{}

	opts := Options{Query: "go 1.27 release notes", Results: 3}
	if err := Run(context.Background(), client, opts, stdout); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	want := "https://go.dev/doc/go1.27\tGeneric methods land.; json/v2 arrives.\n" +
		"https://go.dev/blog/go1.27\tGo 1.27 is released\n" +
		"https://example.com/empty\t\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if client.query != "go 1.27 release notes" {
		t.Errorf("query = %q, want %q", client.query, "go 1.27 release notes")
	}
	if client.limit != 3 {
		t.Errorf("limit = %d, want 3", client.limit)
	}
}

func TestRunPrintsJSON(t *testing.T) {
	t.Parallel()

	client := &fakeSearcher{results: []exa.Result{
		{URL: "https://example.com/a?x=1&y=2", Summary: "A page"},
	}}
	stdout := &bytes.Buffer{}

	opts := Options{Query: "query", Results: 2, JSON: true}
	if err := Run(context.Background(), client, opts, stdout); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	line := strings.TrimSuffix(stdout.String(), "\n")
	if strings.ContainsRune(line, '\n') {
		t.Errorf("stdout holds more than one line: %q", stdout.String())
	}
	if strings.Contains(line, `\u0026`) {
		t.Errorf("JSON escaped the URL: %s", line)
	}
	var items []item
	if err := json.Unmarshal([]byte(line), &items); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if len(items) != 1 || items[0].URL != "https://example.com/a?x=1&y=2" || items[0].Summary != "A page" {
		t.Errorf("items = %+v", items)
	}
}

func TestRunWithoutResults(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	opts := Options{Query: "nothing", JSON: true}
	if err := Run(context.Background(), &fakeSearcher{}, opts, stdout); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	if got := stdout.String(); got != "[]\n" {
		t.Errorf("stdout = %q, want %q", got, "[]\n")
	}
}

func TestRunReportsSearchFailure(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	client := &fakeSearcher{err: errors.New("mcp error -32000: rate limited")}

	err := Run(context.Background(), client, Options{Query: "query"}, stdout)

	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("Run returned %v, want the search error", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}
