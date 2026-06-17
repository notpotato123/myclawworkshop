package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func gameServerURL() string {
	if url := os.Getenv("GAME_SERVER_URL"); url != "" {
		return url
	}
	return "http://localhost:9090"
}

func gamePost(ctx context.Context, path string, payload interface{}) (string, error) {
	url := gameServerURL() + path
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return string(respBody), nil
}
