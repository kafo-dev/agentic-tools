package webfetch

import "testing"

func TestLookupFindsTheRegisteredHost(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Add(Rule{Name: "docs", Hosts: []string{"example.com"}, Select: "#doc"})

	rule, ok := registry.Lookup("example.com")
	if !ok {
		t.Fatal("Lookup found no rule for a registered host")
	}
	if rule.Name != "docs" {
		t.Errorf("rule.Name = %q, want %q", rule.Name, "docs")
	}
}

func TestLookupIgnoresAnUnregisteredHost(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Add(Rule{Name: "docs", Hosts: []string{"example.com"}})

	if rule, ok := registry.Lookup("other.example.net"); ok {
		t.Errorf("Lookup found %+v for an unregistered host", rule)
	}
}

func TestLookupMatchesAWildcard(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"subdomain":         "docs.example.com",
		"nested subdomain":  "v2.docs.example.com",
		"the domain itself": "example.com",
	}
	for name, host := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			registry := NewRegistry()
			registry.Add(Rule{Name: "docs", Hosts: []string{"*.example.com"}})

			if _, ok := registry.Lookup(host); !ok {
				t.Errorf("Lookup found no rule for %q", host)
			}
		})
	}
}

func TestLookupPrefersAnExactHost(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Add(Rule{Name: "wildcard", Hosts: []string{"*.example.com"}})
	registry.Add(Rule{Name: "exact", Hosts: []string{"docs.example.com"}})

	rule, ok := registry.Lookup("docs.example.com")
	if !ok {
		t.Fatal("Lookup found no rule")
	}
	if rule.Name != "exact" {
		t.Errorf("rule.Name = %q, want %q", rule.Name, "exact")
	}
}

func TestLookupFoldsTheHost(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Add(Rule{Name: "docs", Hosts: []string{"Docs.Example.COM."}})

	if _, ok := registry.Lookup("docs.example.com"); !ok {
		t.Error("Lookup missed a host that differs in case and trailing dot")
	}
}

func TestAddReplacesAnEarlierRule(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Add(Rule{Name: "first", Hosts: []string{"example.com"}})
	registry.Add(Rule{Name: "second", Hosts: []string{"example.com"}})

	rule, ok := registry.Lookup("example.com")
	if !ok {
		t.Fatal("Lookup found no rule")
	}
	if rule.Name != "second" {
		t.Errorf("rule.Name = %q, want %q", rule.Name, "second")
	}
}
