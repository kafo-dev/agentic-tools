package webfetch

import (
	"fmt"
	"net/http"
	"net/url"
	pathpkg "path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Result holds one page's content and the branch that produced it.
type Result struct {
	// URL is the address the content came from, after redirects and
	// negotiation.
	URL string
	// Method names the branch that produced the content: one of
	// [MethodNegotiated], [MethodAlternate], [MethodVariant], [MethodLlmsTxt]
	// and [MethodGeneric], or [MethodRulePrefix] followed by a rule's name.
	Method string
	// Content holds the markdown.
	Content string
}

// Method names the branch [Read] took.
const (
	// MethodNegotiated marks content the site served in a form meant for
	// agents.
	MethodNegotiated = "negotiated"
	// MethodAlternate marks a `rel="alternate" type="text/markdown"` link,
	// from the HTML or from a `Link:` header.
	MethodAlternate = "alternate"
	// MethodVariant marks the `.md` variant of the URL.
	MethodVariant = "md-variant"
	// MethodLlmsTxt marks an `llms.txt` file or one of its entries.
	MethodLlmsTxt = "llms.txt"
	// MethodRulePrefix opens the name of the site rule that served the page.
	MethodRulePrefix = "rule:"
	// MethodGeneric marks HTML with its furniture stripped.
	MethodGeneric = "generic"
)

const (
	// alternateRel and describedRel are the link relations the llms.txt
	// proposal defines: one points at a page's markdown, the other at the
	// llms.txt file that covers the page.
	alternateRel = "alternate"
	describedRel = "describedby"
	// markdownType is the media type of a markdown alternate.
	markdownType = "text/markdown"
	// llmsFile names the index file, and markdownExt the extension of a
	// markdown variant.
	llmsFile    = "llms.txt"
	markdownExt = ".md"
)

// llmsLink matches the destination of a Markdown link, the shape an llms.txt
// entry takes.
var llmsLink = regexp.MustCompile(`\]\(\s*([^\s)]+)`)

// Read fetches rawURL and returns its content as markdown. The branches run in
// the order the llms.txt proposal ranks them: what the site served, what it
// advertises, the `.md` variant, the llms.txt file, a site rule, and finally
// the HTML already in hand.
func Read(fetcher *Fetcher, rules *Registry, rawURL string) (Result, error) {
	page, err := fetcher.Get(rawURL, acceptMarkdown)
	if err != nil {
		return Result{}, err
	}
	if page.Status != http.StatusOK {
		return Result{}, fmt.Errorf("GET %s: HTTP %d", page.URL, page.Status)
	}
	if !page.IsHTML() {
		return Result{URL: page.URL.String(), Method: MethodNegotiated, Content: page.Text()}, nil
	}
	if result, ok := probeAlternate(fetcher, page); ok {
		return result, nil
	}
	if result, ok := probeVariant(fetcher, page.URL); ok {
		return result, nil
	}
	if result, ok := probeLlmsTxt(fetcher, page); ok {
		return result, nil
	}
	if result, ok := probeMirror(fetcher, rules, page); ok {
		return result, nil
	}
	return extract(rules, page)
}

// probeAlternate follows the markdown alternate a page advertises, in its
// `Link:` headers or in its HTML.
func probeAlternate(fetcher *Fetcher, page *Page) (Result, bool) {
	candidates := linkTargets(page.Header, alternateRel, markdownType)
	candidates = append(candidates, htmlLinks(page.Body, `link[rel~="alternate"][type="text/markdown"]`)...)
	for _, candidate := range resolveAll(page.URL, candidates) {
		if result, ok := probe(fetcher, candidate, MethodAlternate); ok {
			return result, true
		}
	}
	return Result{}, false
}

// probeVariant follows the `.md` variant of the page URL.
func probeVariant(fetcher *Fetcher, page *url.URL) (Result, bool) {
	for _, candidate := range variantURLs(page) {
		if candidate == page.String() {
			continue
		}
		if result, ok := probe(fetcher, candidate, MethodVariant); ok {
			return result, true
		}
	}
	return Result{}, false
}

// probeLlmsTxt reads the llms.txt file that covers the page, from the location
// the page advertises or, failing that, from the page's directory up to the
// site root. It then follows the entry the file lists for the page.
func probeLlmsTxt(fetcher *Fetcher, page *Page) (Result, bool) {
	locations := linkTargets(page.Header, describedRel, "")
	locations = append(locations, htmlLinks(page.Body, `link[rel~="describedby"]`)...)
	locations = append(locations, llmsLocations(page.URL)...)
	for _, location := range resolveAll(page.URL, locations) {
		if result, ok := readLlmsTxt(fetcher, location, page.URL); ok {
			return result, true
		}
	}
	return Result{}, false
}

// readLlmsTxt reads one llms.txt location and returns the markdown of the
// entry it lists for page.
func readLlmsTxt(fetcher *Fetcher, location string, page *url.URL) (Result, bool) {
	index, err := fetcher.Get(location, acceptMarkdown)
	if err != nil {
		return Result{}, false
	}
	if index.IsHTML() {
		return Result{}, false
	}
	if len(index.Body) == 0 {
		return Result{}, false
	}
	entry, ok := llmsEntry(index, page)
	if !ok {
		return Result{}, false
	}
	return probe(fetcher, entry, MethodLlmsTxt)
}

// probeMirror follows the mirror of a site rule: a raw file, an API, or a host
// that serves the same content without the page furniture. The mirror may hold
// HTML, which the rule's selectors then cut down.
func probeMirror(fetcher *Fetcher, rules *Registry, page *Page) (Result, bool) {
	rule, ok := rules.Lookup(page.URL.Hostname())
	if !ok {
		return Result{}, false
	}
	if rule.Mirror == nil {
		return Result{}, false
	}
	mirror, mirrored := rule.Mirror(page.URL)
	if !mirrored {
		return Result{}, false
	}
	answer, err := fetcher.Get(mirror, acceptMarkdown)
	if err != nil {
		return Result{}, false
	}
	if result, found := offeredMarkdown(answer, MethodRulePrefix+rule.Name); found {
		return result, true
	}
	if answer.Status != http.StatusOK {
		return Result{}, false
	}
	content, err := readableText(answer.Body, rule)
	if err != nil {
		return Result{}, false
	}
	return Result{URL: answer.URL.String(), Method: MethodRulePrefix + rule.Name, Content: content}, true
}

// extract cuts the readable part out of the HTML already in hand, through the
// rule that serves the host when one exists.
func extract(rules *Registry, page *Page) (Result, error) {
	rule, hasRule := rules.Lookup(page.URL.Hostname())
	content, err := readableText(page.Body, rule)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", page.URL, err)
	}
	return Result{URL: page.URL.String(), Method: methodFor(rule, hasRule), Content: content}, nil
}

// methodFor names the method an extraction ran under. A rule that matched the
// host but brought neither a selector nor an extra drop leaves the generic
// pass to read the page, and the output says so.
func methodFor(rule Rule, hasRule bool) string {
	if hasRule && (rule.Select != "" || len(rule.Drop) > 0) {
		return MethodRulePrefix + rule.Name
	}
	return MethodGeneric
}

// probe fetches one candidate and returns its content when it holds markdown
// rather than HTML. A candidate that fails to arrive, or that arrives as HTML,
// yields nothing: the caller then tries the next branch.
func probe(fetcher *Fetcher, candidate, method string) (Result, bool) {
	page, err := fetcher.Get(candidate, acceptMarkdown)
	if err != nil {
		return Result{}, false
	}
	return offeredMarkdown(page, method)
}

// offeredMarkdown returns the content of a page the site meant for agents, and
// reports false for HTML, for an empty body, and for an error status.
func offeredMarkdown(page *Page, method string) (Result, bool) {
	if page.Status != http.StatusOK {
		return Result{}, false
	}
	if page.IsHTML() {
		return Result{}, false
	}
	if len(page.Body) == 0 {
		return Result{}, false
	}
	return Result{URL: page.URL.String(), Method: method, Content: page.Text()}, true
}

// variantURLs lists the markdown variants of u: the file name with its
// extension replaced by `.md`, the URL with `.md` appended, and `index.md` for
// a directory.
func variantURLs(u *url.URL) []string {
	base := *u
	base.RawQuery, base.Fragment = "", ""
	path := base.Path
	if strings.HasSuffix(path, "/") {
		return []string{withPath(&base, path+"index.md"), withPath(&base, path+"index.html.md")}
	}
	var candidates []string
	if extension := pathpkg.Ext(path); extension != "" {
		candidates = append(candidates, withPath(&base, strings.TrimSuffix(path, extension)+markdownExt))
	}
	return append(candidates, withPath(&base, path+markdownExt))
}

// llmsLocations lists the llms.txt addresses that may cover page: the one of
// the page's directory, then the ones of every directory above it up to the
// site root.
func llmsLocations(page *url.URL) []string {
	base := *page
	base.RawQuery, base.Fragment = "", ""
	directory := directoryOf(base.Path)
	var locations []string
	for {
		locations = append(locations, withPath(&base, directory+llmsFile))
		if directory == "/" {
			return locations
		}
		directory = directoryOf(strings.TrimSuffix(directory, "/"))
	}
}

// llmsEntry returns the link an llms.txt file lists for page, resolved against
// the address the file arrived from.
func llmsEntry(index *Page, page *url.URL) (string, bool) {
	want := normPath(page.Path)
	for _, target := range llmsLink.FindAllStringSubmatch(index.Text(), -1) {
		resolved, err := index.URL.Parse(target[1])
		if err != nil {
			continue
		}
		if normPath(resolved.Path) == want {
			return resolved.String(), true
		}
	}
	return "", false
}

// directoryOf returns the directory of a URL path, with its trailing slash.
func directoryOf(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	if index := strings.LastIndex(path, "/"); index >= 0 {
		return path[:index+1]
	}
	return "/"
}

// withPath returns u with its path replaced, keeping the rest.
func withPath(u *url.URL, path string) string {
	copied := *u
	copied.Path = path
	return copied.String()
}

// resolveAll resolves relative links against base and drops the duplicates,
// keeping the order.
func resolveAll(base *url.URL, links []string) []string {
	resolved := make([]string, 0, len(links))
	seen := make(map[string]bool, len(links))
	for _, link := range links {
		target, err := base.Parse(strings.TrimSpace(link))
		if err != nil {
			continue
		}
		if seen[target.String()] {
			continue
		}
		seen[target.String()] = true
		resolved = append(resolved, target.String())
	}
	return resolved
}

// normPath folds a URL path for comparison: it drops a trailing slash, a
// trailing `index`, and repeated markdown and HTML extensions, so that
// `/docs/guide.html`, `/docs/guide.md`, and `/docs/guide` compare equal.
func normPath(path string) string {
	path = pathpkg.Clean("/" + strings.Trim(path, "/"))
	for {
		trimmed := trimExt(path)
		if trimmed == path {
			break
		}
		path = trimmed
	}
	path = strings.TrimSuffix(path, "/index")
	if path == "" {
		return "/"
	}
	return path
}

// pageExtensions lists the suffixes a markdown variant replaces.
var pageExtensions = []string{markdownExt, ".markdown", ".html", ".htm", ".txt"}

// trimExt drops one markdown or HTML extension from path.
func trimExt(path string) string {
	for _, extension := range pageExtensions {
		if trimmed, found := strings.CutSuffix(path, extension); found {
			return trimmed
		}
	}
	return path
}

// linkTargets returns the URLs of the `Link:` headers whose relation is rel
// and whose type parameter carries mediaType. An empty mediaType asks for no
// type parameter.
func linkTargets(header http.Header, rel, mediaType string) []string {
	var targets []string
	for _, value := range header.Values("Link") {
		for target, params := range linkEntries(value) {
			if !hasParameter(params, "rel", rel) {
				continue
			}
			if mediaType != "" && !hasParameter(params, "type", mediaType) {
				continue
			}
			targets = append(targets, target)
		}
	}
	return targets
}

// linkEntries returns the targets of one `Link:` header value, each with the
// parameters that follow it.
func linkEntries(value string) map[string]map[string]string {
	entries := map[string]map[string]string{}
	for entry := range strings.SplitSeq(value, ",") {
		target, rest, found := strings.Cut(strings.TrimSpace(entry), ";")
		if !found {
			continue
		}
		target = strings.TrimSpace(target)
		if !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
			continue
		}
		entries[target[1:len(target)-1]] = parameters(rest)
	}
	return entries
}

// parameters reads the `name=value` pairs of one `Link:` header entry.
func parameters(value string) map[string]string {
	params := map[string]string{}
	for pair := range strings.SplitSeq(value, ";") {
		name, param, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		params[strings.ToLower(strings.TrimSpace(name))] = strings.Trim(strings.TrimSpace(param), `"`)
	}
	return params
}

// hasParameter reports whether params carries name, with value as one of its
// space-separated words.
func hasParameter(params map[string]string, name, value string) bool {
	for word := range strings.FieldsSeq(params[name]) {
		if strings.EqualFold(word, value) {
			return true
		}
	}
	return false
}

// htmlLinks returns the href targets of the HTML elements a CSS selector
// matches.
func htmlLinks(body []byte, selector string) []string {
	document, err := parseHTML(body)
	if err != nil {
		return nil
	}
	var hrefs []string
	document.Find(selector).Each(func(_ int, element *goquery.Selection) {
		if href, ok := element.Attr("href"); ok {
			hrefs = append(hrefs, href)
		}
	})
	return hrefs
}
