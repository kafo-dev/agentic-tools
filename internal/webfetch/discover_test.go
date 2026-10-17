package webfetch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const (
	htmlMedia     = "text/html; charset=utf-8"
	markdownMedia = "text/markdown; charset=utf-8"
)

// page is one response of the test site.
type page struct {
	body  string
	media string
	head  map[string]string
}

// serve starts a site that answers the pages it holds, and answers 404 for
// every other path.
func serve(t *testing.T, pages map[string]page) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		for name, value := range answer.head {
			w.Header().Set(name, value)
		}
		w.Header().Set("Content-Type", answer.media)
		w.Write([]byte(answer.body))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// htmlPage and markdownPage build the two kinds of page the tests serve.
func htmlPage(body string) page {
	return page{body: body, media: htmlMedia}
}

func markdownPage(body string) page {
	return page{body: body, media: markdownMedia}
}

// read fetches one path from a site and returns the result.
func read(t *testing.T, base, path string) Result {
	t.Helper()

	result, err := Read(New(Options{}), NewRegistry(), base+path)
	if err != nil {
		t.Fatalf("Read(%s): %v", path, err)
	}
	return result
}

func TestReadTakesTheMarkdownAPageServes(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{"/guide": markdownPage("# Guide\n")})

	result := read(t, base, "/guide")

	if result.Method != MethodNegotiated {
		t.Errorf("method = %q, want %q", result.Method, MethodNegotiated)
	}
	if result.Content != "# Guide\n" {
		t.Errorf("content = %q, want %q", result.Content, "# Guide\n")
	}
	if result.URL != base+"/guide" {
		t.Errorf("url = %q, want %q", result.URL, base+"/guide")
	}
}

func TestReadFollowsTheAlternateInTheHTML(t *testing.T) {
	t.Parallel()

	body := `<html><head><link rel="alternate" type="text/markdown" href="/guide.alt.md">` +
		`</head><body><p>Text</p></body></html>`
	base := serve(t, map[string]page{
		"/guide":         htmlPage(body),
		"/guide.alt.md":  markdownPage("# Alternate\n"),
		"/guide.html.md": markdownPage("# Never\n"),
	})

	result := read(t, base, "/guide")

	if result.Method != MethodAlternate {
		t.Errorf("method = %q, want %q", result.Method, MethodAlternate)
	}
	if result.URL != base+"/guide.alt.md" {
		t.Errorf("url = %q, want %q", result.URL, base+"/guide.alt.md")
	}
}

func TestReadFollowsTheAlternateInTheHeader(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide": {
			body:  "<html><body><p>Text</p></body></html>",
			media: htmlMedia,
			head: map[string]string{
				"Link": `</guide.alt.md>; rel="alternate"; type="text/markdown", </feed>; rel="alternate"; type="application/atom+xml"`,
			},
		},
		"/guide.alt.md": markdownPage("# Alternate\n"),
	})

	result := read(t, base, "/guide")

	if result.Method != MethodAlternate {
		t.Errorf("method = %q, want %q", result.Method, MethodAlternate)
	}
	if result.URL != base+"/guide.alt.md" {
		t.Errorf("url = %q, want %q", result.URL, base+"/guide.alt.md")
	}
}

func TestReadFollowsTheMarkdownVariant(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide.html": htmlPage("<html><body><p>Text</p></body></html>"),
		"/guide.md":   markdownPage("# Variant\n"),
	})

	result := read(t, base, "/guide.html")

	if result.Method != MethodVariant {
		t.Errorf("method = %q, want %q", result.Method, MethodVariant)
	}
	if result.URL != base+"/guide.md" {
		t.Errorf("url = %q, want %q", result.URL, base+"/guide.md")
	}
}

func TestReadFollowsTheLlmsTxtEntry(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/docs/api/guide.html":   htmlPage("<html><body><p>Text</p></body></html>"),
		"/docs/api/llms.txt":     markdownPage("# API\n\n- [Guide](guide.md.txt)\n"),
		"/docs/api/guide.md.txt": markdownPage("# From the index\n"),
	})

	result := read(t, base, "/docs/api/guide.html")

	if result.Method != MethodLlmsTxt {
		t.Errorf("method = %q, want %q", result.Method, MethodLlmsTxt)
	}
	if result.URL != base+"/docs/api/guide.md.txt" {
		t.Errorf("url = %q, want %q", result.URL, base+"/docs/api/guide.md.txt")
	}
}

func TestReadFollowsTheDescribedbyLocation(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide.html": {
			body:  "<html><body><p>Text</p></body></html>",
			media: htmlMedia,
			head:  map[string]string{"Link": `</docs/llms.txt>; rel="describedby"`},
		},
		"/docs/llms.txt": markdownPage("# Docs\n\n- [Guide](/guide.md.txt)\n"),
		"/guide.md.txt":  markdownPage("# Indexed\n"),
	})

	result := read(t, base, "/guide.html")

	if result.Method != MethodLlmsTxt {
		t.Errorf("method = %q, want %q", result.Method, MethodLlmsTxt)
	}
	if result.URL != base+"/guide.md.txt" {
		t.Errorf("url = %q, want %q", result.URL, base+"/guide.md.txt")
	}
}

func TestReadCutsTheHTMLWithTheSiteRule(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide": htmlPage(`<html><body><nav><p>Menu</p></nav>` +
			`<div id="doc"><p>Body text.</p></div></body></html>`),
	})
	rules := NewRegistry()
	rules.Add(Rule{Name: "docs", Hosts: []string{"127.0.0.1"}, Select: "#doc"})

	result, err := Read(New(Options{}), rules, base+"/guide")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if want := MethodRulePrefix + "docs"; result.Method != want {
		t.Errorf("method = %q, want %q", result.Method, want)
	}
	if !strings.Contains(result.Content, "Body text.") {
		t.Errorf("content = %q, want the selected text", result.Content)
	}
	if strings.Contains(result.Content, "Menu") {
		t.Errorf("content = %q, want no navigation", result.Content)
	}
}

func TestReadFollowsTheMirrorOfTheSiteRule(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide":  htmlPage("<html><body><nav><p>Menu</p></nav></body></html>"),
		"/raw.md": markdownPage("# Raw\n"),
	})
	rules := NewRegistry()
	rules.Add(Rule{
		Name:  "raw",
		Hosts: []string{"127.0.0.1"},
		Mirror: func(*url.URL) (string, bool) {
			return base + "/raw.md", true
		},
	})

	result, err := Read(New(Options{}), rules, base+"/guide")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if want := MethodRulePrefix + "raw"; result.Method != want {
		t.Errorf("method = %q, want %q", result.Method, want)
	}
	if result.URL != base+"/raw.md" {
		t.Errorf("url = %q, want %q", result.URL, base+"/raw.md")
	}
}

func TestReadFallsBackWhenTheMirrorDeclines(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{
		"/guide": htmlPage(`<html><body><article><p>Body text.</p></article></body></html>`),
	})
	rules := NewRegistry()
	rules.Add(Rule{
		Name:   "raw",
		Hosts:  []string{"127.0.0.1"},
		Mirror: func(*url.URL) (string, bool) { return "", false },
	})

	result, err := Read(New(Options{}), rules, base+"/guide")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if result.Method != MethodGeneric {
		t.Errorf("method = %q, want %q", result.Method, MethodGeneric)
	}
}

func TestReadReportsAnErrorStatus(t *testing.T) {
	t.Parallel()

	base := serve(t, map[string]page{})

	_, err := Read(New(Options{}), NewRegistry(), base+"/gone")
	if err == nil {
		t.Fatal("Read returned no error for a 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want the status code", err)
	}
}

func TestReadReportsAFailedTransfer(t *testing.T) {
	t.Parallel()

	_, err := Read(New(Options{}), NewRegistry(), "http://127.0.0.1:1/unreachable")
	if err == nil {
		t.Fatal("Read returned no error for an unreachable host")
	}
}
