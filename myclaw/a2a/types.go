// Package a2a implements the Agent-to-Agent (A2A) protocol types and
// JSON-RPC 2.0 envelope structures for inter-agent communication.
package a2a

// ── Core A2A types ────────────────────────────────────────────────────────────

// AgentCard describes an agent's identity and capabilities so peers can
// discover what it is and what it can do.
type AgentCard struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`         // base URL where this agent is reachable
	Version     string   `json:"version"`
	Skills      []string `json:"skills"`      // human-readable capability labels
}

// Message is a single-turn communication between two agents, carrying one
// or more Parts and a role identifying the sender (e.g. "user" or "agent").
type Message struct {
	Role  string `json:"role"`  // "user" | "agent"
	Parts []Part `json:"parts"`
}

// Part is the smallest content unit inside a Message: text, a file reference,
// or arbitrary structured data.
type Part struct {
	Type string `json:"type"` // "text" | "file" | "data"

	// text
	Text string `json:"text,omitempty"`

	// file
	MimeType string `json:"mimeType,omitempty"`
	Data     []byte `json:"data,omitempty"`     // inline bytes (base64 in JSON)
	FileURL  string `json:"fileUrl,omitempty"`  // or a URL reference

	// data
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TextPart is a convenience constructor for a plain-text Part.
func TextPart(text string) Part {
	return Part{Type: "text", Text: text}
}

// ── JSON-RPC 2.0 envelope types ───────────────────────────────────────────────

// RPCRequest is the JSON-RPC 2.0 request envelope.
type RPCRequest struct {
	JSONRPC string `json:"jsonrpc"` // always "2.0"
	ID      any    `json:"id"`      // string | number | null
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// RPCResponse is the JSON-RPC 2.0 response envelope.
// Exactly one of Result or Error is set.
type RPCResponse struct {
	JSONRPC string     `json:"jsonrpc"` // always "2.0"
	ID      any        `json:"id"`
	Result  any        `json:"result,omitempty"`
	Error   *RPCError  `json:"error,omitempty"`
}

// RPCError is the JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// OKResponse builds a successful RPCResponse.
func OKResponse(id, result any) RPCResponse {
	return RPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

// ErrResponse builds an error RPCResponse.
func ErrResponse(id any, code int, message string) RPCResponse {
	return RPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}}
}
