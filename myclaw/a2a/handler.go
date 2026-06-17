package a2a

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler processes incoming A2A JSON-RPC 2.0 requests and feeds them into
// the agent via the provided send function. send must block until the agent
// has finished responding and must return the full response text.
type Handler struct {
	send func(text string) (string, error)
}

// NewHandler returns an A2A HTTP handler. sendFn is called with the extracted
// text of each incoming message; it should feed the text into the agent and
// return the complete response.
func NewHandler(sendFn func(text string) (string, error)) *Handler {
	return &Handler{send: sendFn}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrResponse(nil, CodeParseError, "parse error: "+err.Error()))
		return
	}

	if req.JSONRPC != "2.0" {
		writeJSON(w, ErrResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\""))
		return
	}

	if req.Method != "message/send" {
		writeJSON(w, ErrResponse(req.ID, CodeMethodNotFound, "method not found: "+req.Method))
		return
	}

	// Params is { "message": { "role": "...", "parts": [...] } }
	params, err := extractParams(req.Params)
	if err != nil {
		writeJSON(w, ErrResponse(req.ID, CodeInvalidParams, "invalid params: "+err.Error()))
		return
	}

	text := collectText(params)
	if text == "" {
		writeJSON(w, ErrResponse(req.ID, CodeInvalidParams, "no text content in message parts"))
		return
	}

	reply, err := h.send(text)
	if err != nil {
		writeJSON(w, ErrResponse(req.ID, CodeInternalError, "agent error: "+err.Error()))
		return
	}

	result := Message{
		Role:  "agent",
		Parts: []Part{TextPart(reply)},
	}
	writeJSON(w, OKResponse(req.ID, result))
}

// extractParams marshals req.Params back to JSON and decodes the expected shape.
func extractParams(raw any) (Message, error) {
	var wrapper struct {
		Message Message `json:"message"`
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return Message{}, err
	}
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return Message{}, err
	}
	return wrapper.Message, nil
}

// collectText concatenates the text fields of all text Parts in a Message.
func collectText(m Message) string {
	var sb strings.Builder
	for _, p := range m.Parts {
		if p.Type == "text" && p.Text != "" {
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
