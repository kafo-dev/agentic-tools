package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// wantUsage is the heading the help text opens with.
const wantUsage = "Usage of " + command + ":"

// searchResponse is one MCP answer, carrying a single result.
const searchResponse = `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text",` +
	`"text":"{\"results\":[{\"url\":\"https://example.com\",\"summary\":\"A page\"}]}"}]}}`

// noEnv reports an empty environment.
func noEnv(string) string { return "" }

// envFor points the command at endpoint.
func envFor(endpoint string) func(string) string {
	return func(key string) string {
		if key == envEndpoint {
			return endpoint
		}
		return ""
	}
}

// stubSearch starts an MCP endpoint that answers one result, and reports the
// query it received.
func stubSearch(t *testing.T) (string, <-chan string) {
	t.Helper()

	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- searchQuery(t, r)
		w.Write([]byte(searchResponse))
	}))
	t.Cleanup(server.Close)

	return server.URL, received
}

// searchQuery reads the query out of a JSON-RPC search request.
func searchQuery(t *testing.T, r *http.Request) string {
	t.Helper()

	var request struct {
		Params struct {
			Arguments struct {
				Query string `json:"query"`
			} `json:"arguments"`
		} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Errorf("decode request: %v", err)
	}
	return request.Params.Arguments.Query
}

func TestDoSearchesForTheQueryFromStdin(t *testing.T) {
	t.Parallel()

	endpoint, queries := stubSearch(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := do(nil, envFor(endpoint), strings.NewReader("golang generics\ntutorial\n"), stdout, stderr)

	if code != exitOK {
		t.Fatalf("do returned %d, want %d; stderr: %s", code, exitOK, stderr)
	}
	if got := <-queries; got != "golang generics tutorial" {
		t.Errorf("query = %q, want %q", got, "golang generics tutorial")
	}
	if got, want := stdout.String(), "https://example.com\tA page\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestDoPrefersArgumentsOverStdin(t *testing.T) {
	t.Parallel()

	endpoint, queries := stubSearch(t)
	stdin := strings.NewReader("from standard input")

	code := do([]string{"from", "arguments"}, envFor(endpoint), stdin, &bytes.Buffer{}, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	if got := <-queries; got != "from arguments" {
		t.Errorf("query = %q, want %q", got, "from arguments")
	}
}

func TestDoPrintsJSON(t *testing.T) {
	t.Parallel()

	endpoint, _ := stubSearch(t)
	stdout := &bytes.Buffer{}

	code := do([]string{"-json", "query"}, envFor(endpoint), nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	line := strings.TrimSuffix(stdout.String(), "\n")
	if strings.ContainsRune(line, '\n') {
		t.Errorf("stdout holds more than one line: %q", stdout.String())
	}
	var items []struct {
		URL     string `json:"url"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(line), &items); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if len(items) != 1 || items[0].URL != "https://example.com" || items[0].Summary != "A page" {
		t.Errorf("items = %+v", items)
	}
}

func TestDoReportsSearchFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := do([]string{"query"}, envFor(server.URL), nil, stdout, stderr)

	if code != exitFail {
		t.Fatalf("do returned %d, want %d", code, exitFail)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr.String(), "429") {
		t.Errorf("stderr = %q, want the status code", stderr)
	}
}

func TestDoRejectsMalformedInvocations(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"no query":     {},
		"unknown flag": {"-verbose", "query"},
		"bad count":    {"-n", "0", "query"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			code := do(args, noEnv, nil, stdout, stderr)

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
	code := do(nil, noEnv, strings.NewReader("\n\t \n"), stdout, stderr)

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
	input := strings.Repeat("word ", queryBytes/len("word ")+1)

	code := do(nil, noEnv, strings.NewReader(input), stdout, stderr)

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
	code := do([]string{"-h"}, noEnv, nil, stdout, &bytes.Buffer{})

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

func TestDoPrintsSkillWithoutQuery(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	code := do([]string{"-skill"}, noEnv, nil, stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("do returned %d, want %d", code, exitOK)
	}
	document := stdout.String()
	if !strings.HasPrefix(document, "---\nname: agentic-web-search\n") {
		t.Errorf("stdout does not start with the skill frontmatter: %q", document)
	}
	if !strings.Contains(document, wantUsage) {
		t.Errorf("stdout does not carry the usage line: %q", document)
	}
	if !strings.Contains(document, "positional argument") {
		t.Errorf("stdout does not carry the notes: %q", document)
	}
	if strings.Contains(document, "{{usage}}") {
		t.Errorf("stdout left the placeholder in place: %q", document)
	}
}
