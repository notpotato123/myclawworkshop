package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"myclaw/memory"
)

type Recall struct {
	Store *memory.Store
}

func (t Recall) Name() string { return "recall" }
func (t Recall) Description() string {
	return "Retrieve previously saved memories. Provide a query to search for relevant memories, or leave query empty to list all memory keys."
}

func (t Recall) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search term to find relevant memories. Leave empty to list all memory keys.",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
	}
}

func (t Recall) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	if p.Query == "" {
		keys, err := t.Store.List()
		if err != nil {
			return "", fmt.Errorf("listing memories: %w", err)
		}
		if len(keys) == 0 {
			return "No memories stored yet.", nil
		}
		return "Memory keys:\n- " + strings.Join(keys, "\n- "), nil
	}

	results, err := t.Store.Search(p.Query)
	if err != nil {
		return "", fmt.Errorf("searching memories: %w", err)
	}
	if len(results) == 0 {
		return fmt.Sprintf("No memories found matching %q.", p.Query), nil
	}
	return strings.Join(results, "\n"), nil
}
