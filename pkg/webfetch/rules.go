package webfetch

import (
	"net/url"
	"strings"
)

// subdomain marks a host entry that serves the domain and its subdomains.
const subdomain = "*."

// Rule tells the reader how to read one site's pages.
type Rule struct {
	// Name labels the rule in the reader's output, which reports it as
	// `rule:<name>`.
	Name string
	// Hosts lists the hosts the rule serves. An entry starting with `*.`
	// serves that domain and every subdomain of it.
	Hosts []string
	// Mirror rewrites a page URL into one that serves the same content in a
	// form an agent reads without cleaning: a raw file, an API, a mirror that
	// allows scraping. It reports false when the URL has no mirror.
	Mirror func(u *url.URL) (string, bool)
	// Select holds the CSS selector of the readable part. An empty selector
	// leaves the choice to the generic pass.
	Select string
	// Drop lists CSS selectors of page furniture to remove before conversion,
	// in addition to the elements the generic pass always drops.
	Drop []string
}

// Registry holds the rules, indexed by host.
type Registry struct {
	byHost map[string]Rule
}

// NewRegistry returns a registry that holds no rules.
func NewRegistry() *Registry {
	return &Registry{byHost: make(map[string]Rule)}
}

// Add registers rules. A later rule for the same host replaces an earlier one.
func (r *Registry) Add(rules ...Rule) {
	for _, rule := range rules {
		for _, host := range rule.Hosts {
			r.byHost[hostKey(host)] = rule
		}
	}
}

// Lookup returns the rule that serves host, preferring an exact host over a
// wildcard entry.
func (r *Registry) Lookup(host string) (Rule, bool) {
	host = hostKey(host)
	if rule, ok := r.byHost[host]; ok {
		return rule, true
	}
	for domain := host; domain != ""; domain = parentDomain(domain) {
		if rule, ok := r.byHost[subdomain+domain]; ok {
			return rule, true
		}
	}
	return Rule{}, false
}

// hostKey folds a host or a host entry into the registry's key form, so a
// trailing dot or a capital letter never hides a rule.
func hostKey(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimSuffix(host, ".")
}

// parentDomain returns the labels above host's leftmost one: "example.com"
// for "docs.example.com", and, for "example.com", "com". It returns "" when
// no dot remains, so a caller walks the host until its labels run out.
func parentDomain(host string) string {
	_, domain, ok := strings.Cut(host, ".")
	if !ok {
		return ""
	}
	return domain
}
