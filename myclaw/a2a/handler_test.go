package a2a

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerMessageSend(t *testing.T) {
	h := NewHandler(func(text string) (string, error) {
		return "hello back: " + text, nil
	})

	body, _ := json.Marshal(RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "message/send",
		Params: map[string]any{
			"message": map[string]any{
				"role":  "user",
				"parts": []map[string]any{{"type": "text", "text": "ping"}},
			},
		},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp RPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandlerUnknownMethod(t *testing.T) {
	h := NewHandler(func(text string) (string, error) { return "", nil })

	body, _ := json.Marshal(RPCRequest{JSONRPC: "2.0", ID: 1, Method: "unknown/method"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader(body)))

	var resp RPCResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("expected method-not-found error, got %+v", resp)
	}
}

func TestHandlerInvalidJSON(t *testing.T) {
	h := NewHandler(func(text string) (string, error) { return "", nil })

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/a2a", bytes.NewReader([]byte("not json"))))

	var resp RPCResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error == nil || resp.Error.Code != CodeParseError {
		t.Fatalf("expected parse error, got %+v", resp)
	}
}

func TestHandlerWrongHTTPMethod(t *testing.T) {
	h := NewHandler(func(text string) (string, error) { return "", nil })

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/a2a", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
