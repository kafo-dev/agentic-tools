package webfetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		page Page
		want bool
	}{
		"html media type": {
			page: Page{Type: "text/html"},
			want: true,
		},
		"html body under another media type": {
			page: Page{Type: "text/plain", Body: []byte("  <!DOCTYPE html>\n<html><body>")},
			want: true,
		},
		"markdown": {
			page: Page{Type: "text/markdown", Body: []byte("# Guide\n")},
			want: false,
		},
		"json": {
			page: Page{Type: "application/json", Body: []byte(`{"a": 1}`)},
			want: false,
		},
		"empty": {
			page: Page{Type: "", Body: nil},
			want: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := test.page.IsHTML(); got != test.want {
				t.Errorf("IsHTML() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestGetAsksForTheMediaTypesItIsGiven(t *testing.T) {
	t.Parallel()

	headers := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("# Guide\n"))
	}))
	t.Cleanup(server.Close)

	page, err := New(Options{}).Get(server.URL, "text/markdown")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	sent := <-headers
	if got, want := sent.Get("Accept"), "text/markdown"; got != want {
		t.Errorf("Accept = %q, want %q", got, want)
	}
	if got, want := sent.Get("User-Agent"), UserAgent; got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
	if page.Type != "text/markdown" {
		t.Errorf("Type = %q, want %q", page.Type, "text/markdown")
	}
}

func TestGetReturnsAnErrorStatusAsAPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	page, err := New(Options{}).Get(server.URL, "text/markdown")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if page.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", page.Status, http.StatusNotFound)
	}
}

func TestGetReportsTheAddressTheResponseCameFrom(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte("# Guide\n"))
	}))
	t.Cleanup(server.Close)

	page, err := New(Options{}).Get(server.URL+"/old", "text/markdown")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got, want := page.URL.String(), server.URL+"/new"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}
