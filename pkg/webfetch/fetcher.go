// Package webfetch reads a URL as markdown for a caller that pays per token:
// it asks the site for a form meant for agents, follows the llms.txt
// conventions, and falls back to the readable part of the HTML.
//
// The package follows [the /llms.txt proposal] for content negotiation, for
// the `.md` variants of a URL, and for the `llms.txt` file itself. A [Rule] in
// the [Registry] gives a site its own selector, its own furniture list, or a
// mirror, for the pages the generic pass reads badly.
//
// [the /llms.txt proposal]: https://llmstxt.org/
package webfetch

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

const (
	// UserAgent identifies the command to a site, and names the product token
	// a robots.txt file matches.
	UserAgent = "agentic-web-fetch/1.0 (+https://github.com/kafo-dev/agentic-tools)"
	// DefaultTimeout bounds one request.
	DefaultTimeout = 32 * time.Second
	// acceptMarkdown asks for markdown first, and keeps HTML as the fallback.
	acceptMarkdown = "text/markdown, text/html;q=0.9, text/plain;q=0.8, */*;q=0.1"
	// maxBody caps one response body: an agent reads a page, not a download.
	maxBody = 8 << 20
	// sniffBytes is how much of a body the HTML check reads.
	sniffBytes = 2048
	// htmlMarker and doctypeMarker appear in the head of an HTML body.
	htmlMarker    = "<html"
	doctypeMarker = "<!doctype html"
	// collectionKey names the slot a request leaves its collection in.
	collectionKey = "webfetch.collection"
)

// Fetcher gets pages with [colly], under the rules the site publishes in its
// robots.txt file. A Fetcher holds one collector, so the rules it has read
// serve every request and a site answers for its robots.txt once.
type Fetcher struct {
	collector *colly.Collector
}

// Options configures a [Fetcher]. The zero value keeps every default.
type Options struct {
	// Timeout bounds one request; zero and below keep [DefaultTimeout].
	Timeout time.Duration
	// UserAgent replaces [UserAgent] when it carries text.
	UserAgent string
}

// New returns a fetcher.
func New(opts Options) *Fetcher {
	return &Fetcher{collector: newCollector(opts)}
}

// newCollector builds the collector that serves every request.
func newCollector(opts Options) *colly.Collector {
	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = UserAgent
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	collector := colly.NewCollector(colly.UserAgent(userAgent), colly.AllowURLRevisit())
	// A response with an error status carries headers and a body worth
	// probing, so it reaches the response callback rather than the error
	// callback.
	collector.ParseHTTPErrorResponse = true
	// The rules a site publishes decide which of its pages this tool reads.
	collector.IgnoreRobotsTxt = false
	collector.MaxBodySize = maxBody
	collector.SetRequestTimeout(timeout)
	collector.OnResponse(func(response *colly.Response) {
		collected, ok := response.Ctx.GetAny(collectionKey).(*collection)
		if !ok {
			return
		}
		collected.page = &Page{
			URL:    response.Request.URL,
			Status: response.StatusCode,
			Type:   mediaType(response.Headers.Get("Content-Type")),
			Header: *response.Headers,
			Body:   response.Body,
		}
	})
	return collector
}

// Page is one HTTP response.
type Page struct {
	// URL is the address the response came from, after redirects.
	URL *url.URL
	// Status is the HTTP status code.
	Status int
	// Type is the media type of the body, without its parameters.
	Type string
	// Header holds the response headers.
	Header http.Header
	// Body holds the response body.
	Body []byte
}

// Get fetches rawURL, asking for the media types in accept. A response with an
// error status still arrives as a page; a failed transfer, and a URL the
// site's robots.txt file disallows, return an error.
func (f *Fetcher) Get(rawURL, accept string) (*Page, error) {
	collected := &collection{}
	ctx := colly.NewContext()
	ctx.Put(collectionKey, collected)

	headers := http.Header{"Accept": {accept}}
	if err := f.collector.Request(http.MethodGet, rawURL, nil, ctx, headers); err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	if collected.page == nil {
		return nil, fmt.Errorf("fetch %s: no response", rawURL)
	}
	return collected.page, nil
}

// IsHTML reports whether the body holds HTML: by its media type, or by the
// markers a server that mislabels it leaves in the head.
func (p *Page) IsHTML() bool {
	if strings.Contains(p.Type, "html") {
		return true
	}
	head := strings.ToLower(string(p.Body[:min(len(p.Body), sniffBytes)]))
	return strings.Contains(head, doctypeMarker) || strings.Contains(head, htmlMarker)
}

// Text returns the body as text.
func (p *Page) Text() string {
	return string(p.Body)
}

// collection carries the response of one request out of colly's callbacks,
// which report the response rather than return it.
type collection struct {
	page *Page
}

// mediaType returns the media type of a `Content-Type` header value, without
// its parameters.
func mediaType(contentType string) string {
	media, _, _ := strings.Cut(contentType, ";")
	return strings.ToLower(strings.TrimSpace(media))
}
