package exa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// eventStream wraps a JSON-RPC message the way the server does.
func eventStream(t *testing.T, message string) string {
	t.Helper()
	return "event: message\ndata: " + message + "\n\n"
}

// searchBody builds the text content of a search result.
func searchBody(t *testing.T, results []Result) string {
	t.Helper()
	payload, err := json.Marshal(searchPayload{Results: results})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	message, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result":  rpcResult{Content: []rpcContent{{Type: contentTypeText, Text: string(payload)}}},
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return string(message)
}

func TestSearchDecodesEventStream(t *testing.T) {
	t.Parallel()

	var request rpcRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("tools"); got != searchTool {
			t.Errorf("tools = %q, want %q", got, searchTool)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(eventStream(t, searchBody(t, []Result{{URL: "https://example.com", Summary: "A page"}}))))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTP: server.Client()}
	results, err := client.Search(t.Context(), "a query", 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 || results[0].URL != "https://example.com" || results[0].Summary != "A page" {
		t.Errorf("results = %+v", results)
	}
	if request.Method != methodCall || request.Params.Name != searchTool {
		t.Errorf("request = %+v", request)
	}
	args := request.Params.Arguments
	if args.Query != "a query" || args.NumResults != 3 || !args.EnableSummary || args.SummaryQuery != "a query" {
		t.Errorf("arguments = %+v", args)
	}
}

func TestSearchDecodesPlainJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(searchBody(t, []Result{{URL: "https://example.com"}})))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTP: server.Client()}
	results, err := client.Search(t.Context(), "a query", 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results = %+v", results)
	}
}

func TestSearchReportsServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("slow down\n"))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTP: server.Client()}
	_, err := client.Search(t.Context(), "a query", 1)
	if err == nil {
		t.Fatal("Search returned no error")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %v, want the status code", err)
	}
}

func TestSearchReportsRPCError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(eventStream(t, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"Tool not found"}}`)))
	}))
	defer server.Close()

	client := &Client{Endpoint: server.URL, HTTP: server.Client()}
	_, err := client.Search(t.Context(), "a query", 1)
	if err == nil || !strings.Contains(err.Error(), "Tool not found") {
		t.Errorf("error = %v, want the MCP error message", err)
	}
}

func TestURLAddsTheTool(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":                                  DefaultEndpoint + "?tools=" + searchTool,
		"https://example.com/mcp":           "https://example.com/mcp?tools=" + searchTool,
		"https://example.com/mcp?token=abc": "https://example.com/mcp?token=abc&tools=" + searchTool,
	}
	for endpoint, want := range tests {
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()

			if got := (&Client{Endpoint: endpoint}).url(); got != want {
				t.Errorf("url() = %q, want %q", got, want)
			}
		})
	}
}
