// Package websearch implements agentic-web-search: one Exa search, rendered as
// URLs with a one-line summary each.
package websearch

import (
	"context"
	"io"

	"agentic.kafo.dev/tools/pkg/exa"
)

// Searcher fetches the results for a query.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]exa.Result, error)
}

// Options holds one search request.
type Options struct {
	// Query is the natural-language query to send.
	Query string
	// Results caps the number of results to request.
	Results int
	// JSON renders the results as one line of JSON rather than one line each.
	JSON bool
}

// Run searches for opts.Query and writes the results to w. A search with no
// hits writes an empty list and reports no error.
func Run(ctx context.Context, client Searcher, opts Options, w io.Writer) error {
	results, err := client.Search(ctx, opts.Query, opts.Results)
	if err != nil {
		return err
	}
	return render(w, results, opts.JSON)
}
