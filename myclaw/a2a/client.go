package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultTimeout = 30 * time.Second

// SendMessage posts a JSON-RPC 2.0 message/send request to peerURL and returns
// the peer's response text. peerURL should be the direct A2A endpoint or a
// relay URL (e.g. http://host/a2a).
func SendMessage(ctx context.Context, peerURL, text string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	rpcReq := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "message/send",
		Params: map[string]any{
			"message": Message{
				Role:  "user",
				Parts: []Part{TextPart(text)},
			},
		},
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peerURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending message to %s: %w", peerURL, err)
	}
	defer resp.Body.Close()

	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("peer error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Result is a Message; marshal/unmarshal to extract it.
	b, _ := json.Marshal(rpcResp.Result)
	var msg Message
	if err := json.Unmarshal(b, &msg); err != nil {
		return "", fmt.Errorf("decoding peer message: %w", err)
	}
	return collectText(msg), nil
}

// A2AEndpoint returns the A2A endpoint URL for a given base URL.
func A2AEndpoint(baseURL string) string {
	return baseURL + "/a2a"
}
