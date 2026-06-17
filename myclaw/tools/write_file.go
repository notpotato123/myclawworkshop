package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WriteFile struct{}

func (t WriteFile) Name() string        { return "write_file" }
func (t WriteFile) Description() string { return "Write content to a file at the given path." }

func (t WriteFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to write to.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file.",
			},
		},
		"required":             []string{"path", "content"},
		"additionalProperties": false,
	}
}

func (t WriteFile) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	absPath := p.Path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(cwd, absPath)
	}
	absPath = filepath.Clean(absPath)

	if !strings.HasPrefix(absPath, cwd+string(filepath.Separator)) && absPath != cwd {
		return "", fmt.Errorf("path traversal is not allowed: %s escapes the working directory", p.Path)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create directories: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(p.Content), 0o644); err != nil {
		return "", fmt.Errorf("cannot write file: %w", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(p.Content), p.Path), nil
}
