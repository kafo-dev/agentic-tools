package exa

// Protocol constants of the MCP server.
const (
	jsonRPCVersion  = "2.0"
	methodCall      = "tools/call"
	contentTypeText = "text"
	requestID       = 1
)

// dataPrefix starts each message of the server's event stream.
var dataPrefix = []byte("data:")

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Method  string    `json:"method"`
	Params  rpcParams `json:"params"`
}

// rpcParams names the tool and its arguments.
type rpcParams struct {
	Name      string     `json:"name"`
	Arguments searchArgs `json:"arguments"`
}

// searchArgs holds the arguments of web_search_advanced_exa. The server
// rejects unknown fields, so this struct stays minimal.
type searchArgs struct {
	Query             string `json:"query"`
	NumResults        int    `json:"numResults"`
	EnableSummary     bool   `json:"enableSummary"`
	SummaryQuery      string `json:"summaryQuery"`
	TextMaxCharacters int    `json:"textMaxCharacters"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	Error  *rpcError `json:"error"`
	Result rpcResult `json:"result"`
}

// rpcError carries a JSON-RPC error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcResult holds the tool's content blocks.
type rpcResult struct {
	Content []rpcContent `json:"content"`
}

// rpcContent is one content block; a search answers with JSON in a text block.
type rpcContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// searchPayload is the JSON document inside the text block.
type searchPayload struct {
	Results []Result `json:"results"`
}
