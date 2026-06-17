package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"myclaw/memory"
)

type Remember struct {
	Store *memory.Store
}

func (t Remember) Name() string { return "remember" }
func (t Remember) Description() string {
	return "Save important information to persistent memory. Use this to store facts, preferences, or context that may be useful in future conversations. The key should be a short descriptive label."
}

func (t Remember) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key": map[string]any{
				"type":        "string",
				"description": "A short descriptive label for this memory (e.g. 'user_name', 'project_goal').",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The information to remember.",
			},
		},
		"required":             []string{"key", "content"},
		"additionalProperties": false,
	}
}

func (t Remember) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct {
		Key     string `json:"key"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Key == "" {
		return "", fmt.Errorf("key is required")
	}
	if p.Content == "" {
		return "", fmt.Errorf("content is required")
	}
	if err := t.Store.Save(p.Key, p.Content); err != nil {
		return "", fmt.Errorf("saving memory: %w", err)
	}
	return fmt.Sprintf("Memory saved: %q", p.Key), nil
}
