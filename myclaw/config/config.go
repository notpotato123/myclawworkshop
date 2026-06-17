// Package config loads all runtime settings from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds every tunable setting for the claw binary.
type Config struct {
	BaseURL    string // CLAW_BASE_URL   — optional OpenAI-compatible base URL
	APIKey     string // CLAW_API_KEY    — required
	Model      string // CLAW_MODEL      — LLM model name
	Port       string // CLAW_PORT       — HTTP listen port
	MemoryDir  string // CLAW_MEMORY_DIR — directory for persistent memories
	TasksFile  string // CLAW_TASKS_FILE — path for the scheduler JSON file
}

const (
	DefaultModel     = "gpt-4o"
	DefaultPort      = "8080"
	DefaultMemoryDir = ".claw_memory"
	DefaultTasksFile = "scheduler/tasks.json"
)

// Load reads environment variables and returns a Config with defaults applied.
// It returns an error if any required variable is missing.
func Load() (*Config, error) {
	c := &Config{
		BaseURL:   os.Getenv("CLAW_BASE_URL"),
		APIKey:    os.Getenv("CLAW_API_KEY"),
		Model:     envOr("CLAW_MODEL", DefaultModel),
		Port:      envOr("CLAW_PORT", DefaultPort),
		MemoryDir: envOr("CLAW_MEMORY_DIR", DefaultMemoryDir),
		TasksFile: envOr("CLAW_TASKS_FILE", DefaultTasksFile),
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("CLAW_API_KEY environment variable is required")
	}
	return c, nil
}

// Log prints a configuration summary via slog. The API key is redacted.
func (c *Config) Log() {
	key := "(set)"
	if len(c.APIKey) > 8 {
		key = c.APIKey[:4] + strings.Repeat("*", len(c.APIKey)-8) + c.APIKey[len(c.APIKey)-4:]
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "(default)"
	}
	slog.Info("configuration",
		"model", c.Model,
		"port", c.Port,
		"base_url", baseURL,
		"api_key", key,
		"memory_dir", c.MemoryDir,
		"tasks_file", c.TasksFile,
	)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
