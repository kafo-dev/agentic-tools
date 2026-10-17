package webfetch

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

// page is an HTML page with the furniture a real site carries.
const furniturePage = `<html><head><title>Title</title><style>p{color:red}</style></head>` +
	`<body><nav><a href="/">Home</a></nav><article><h1>Title</h1><p>Body text.</p></article>` +
	`<footer><p>Copyright</p></footer><script>var tracked = 1</script></body></html>`

func TestReadableTextDropsTheFurniture(t *testing.T) {
	t.Parallel()

	content, err := readableText([]byte(furniturePage), Rule{})
	if err != nil {
		t.Fatalf("readableText: %v", err)
	}

	if !strings.Contains(content, "Body text.") {
		t.Errorf("content = %q, want the article text", content)
	}
	for _, unwanted := range []string{"Home", "Copyright", "var tracked"} {
		if strings.Contains(content, unwanted) {
			t.Errorf("content = %q, want no %q", content, unwanted)
		}
	}
}

func TestReadableTextHonoursTheRule(t *testing.T) {
	t.Parallel()

	body := []byte(`<html><body><div id="doc"><p>Body text.</p><p class="legal">Terms.</p></div>` +
		`<div id="aside"><p>Elsewhere.</p></div></body></html>`)
	rule := Rule{Select: "#doc", Drop: []string{".legal"}}

	content, err := readableText(body, rule)
	if err != nil {
		t.Fatalf("readableText: %v", err)
	}

	if !strings.Contains(content, "Body text.") {
		t.Errorf("content = %q, want the selected text", content)
	}
	for _, unwanted := range []string{"Terms.", "Elsewhere."} {
		if strings.Contains(content, unwanted) {
			t.Errorf("content = %q, want no %q", content, unwanted)
		}
	}
}

func TestReadableTextReportsAnEmptyBody(t *testing.T) {
	t.Parallel()

	if content, err := readableText([]byte(`<html><body></body></html>`), Rule{}); err == nil {
		t.Errorf("readableText returned %q and no error", content)
	}
}

func TestTidyCollapsesTheBlankLines(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"a  \n\n\n\nb\n": "a\n\nb\n",
		"\n\n# Title\n":  "# Title\n",
		"":               "",
	}
	for text, want := range tests {
		t.Run(text, func(t *testing.T) {
			t.Parallel()

			if got := tidy(text); got != want {
				t.Errorf("tidy(%q) = %q, want %q", text, got, want)
			}
		})
	}
}

func TestNormPathFoldsExtensions(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/docs/guide":         "/docs/guide",
		"/docs/guide.html":    "/docs/guide",
		"/docs/guide.md":      "/docs/guide",
		"/docs/guide.md.txt":  "/docs/guide",
		"/docs/":              "/docs",
		"/docs/index.html":    "/docs",
		"/docs/guide.html.md": "/docs/guide",
		"/":                   "/",
		"":                    "/",
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if got := normPath(path); got != want {
				t.Errorf("normPath(%q) = %q, want %q", path, got, want)
			}
		})
	}
}

func TestVariantURLs(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"https://example.com/docs/guide.html?v=1#top": {
			"https://example.com/docs/guide.md",
			"https://example.com/docs/guide.html.md",
		},
		"https://example.com/docs/": {
			"https://example.com/docs/index.md",
			"https://example.com/docs/index.html.md",
		},
		"https://example.com/docs/guide": {
			"https://example.com/docs/guide.md",
		},
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			page, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parse %q: %v", raw, err)
			}
			if got := variantURLs(page); !slices.Equal(got, want) {
				t.Errorf("variantURLs(%q) = %q, want %q", raw, got, want)
			}
		})
	}
}

func TestLlmsLocationsWalkToTheRoot(t *testing.T) {
	t.Parallel()

	page, err := url.Parse("https://example.com/docs/api/guide.html?v=1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{
		"https://example.com/docs/api/llms.txt",
		"https://example.com/docs/llms.txt",
		"https://example.com/llms.txt",
	}

	if got := llmsLocations(page); !slices.Equal(got, want) {
		t.Errorf("llmsLocations = %q, want %q", got, want)
	}
}
