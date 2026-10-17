// Package exa queries Exa's hosted MCP server. Its keyless tier answers
// without an account or an API key, inside a per-IP rate limit.
package exa

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultEndpoint serves Exa's keyless MCP tier.
const DefaultEndpoint = "https://mcp.exa.ai/mcp"

const (
	// searchTool names the tool that carries the filters and the summary
	// switch; the plain search tool has neither.
	searchTool = "web_search_advanced_exa"
	// snippetChars caps the page text Exa returns. The summary holds the
	// content this client needs, and the text is only a fallback.
	snippetChars = 400
	// maxBody caps one response.
	maxBody = 8 << 20
	// requestTimeout bounds one search, which Exa may answer from a live crawl.
	requestTimeout = 60 * time.Second
)

// Result holds one search hit.
type Result struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text"`
}

// Client calls an MCP endpoint. The zero value works: it uses
// [DefaultEndpoint] and a default [net/http.Client].
type Client struct {
	// Endpoint overrides [DefaultEndpoint] when set.
	Endpoint string
	// HTTP overrides the default client when set.
	HTTP *http.Client
}

// Search returns at most limit results for query.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      requestID,
		Method:  methodCall,
		Params: rpcParams{
			Name: searchTool,
			Arguments: searchArgs{
				Query:             query,
				NumResults:        limit,
				EnableSummary:     true,
				SummaryQuery:      query,
				TextMaxCharacters: snippetChars,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: %s: %s", resp.Status, firstLine(string(raw)))
	}
	return decodeResults(raw)
}

// url returns the endpoint with the wanted tool enabled.
func (c *Client) url() string {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		// A malformed endpoint reaches the request unchanged and fails there,
		// with the parse error in hand.
		return endpoint
	}
	query := parsed.Query()
	query.Set("tools", searchTool)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: requestTimeout}
}

// decodeResults unwraps the JSON-RPC envelope, then the search payload inside
// its text content.
func decodeResults(raw []byte) ([]Result, error) {
	var envelope rpcResponse
	if err := json.Unmarshal(payload(raw), &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	for _, content := range envelope.Result.Content {
		if content.Type != contentTypeText {
			continue
		}
		var payload searchPayload
		if err := json.Unmarshal([]byte(content.Text), &payload); err != nil {
			return nil, fmt.Errorf("decode results: %w", err)
		}
		return payload.Results, nil
	}
	return nil, errors.New("response carried no text content")
}

// payload returns the JSON-RPC message. The server answers with an SSE stream
// of `data:` lines; a plain JSON body passes through.
func payload(raw []byte) []byte {
	var message []byte
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(nil, maxBody)
	for scanner.Scan() {
		if rest, ok := bytes.CutPrefix(scanner.Bytes(), dataPrefix); ok {
			message = bytes.TrimSpace(rest)
		}
	}
	if message != nil {
		return message
	}
	return bytes.TrimSpace(raw)
}

// firstLine returns the first line of s, for a status body too long to print.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
