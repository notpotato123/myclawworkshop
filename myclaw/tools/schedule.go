package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"myclaw/scheduler"
)

// Schedule lets the LLM create or list scheduled tasks.
type Schedule struct {
	Sched *scheduler.Scheduler
}

func (t Schedule) Name() string { return "schedule" }
func (t Schedule) Description() string {
	return `Schedule a task to run after a delay, list pending tasks, or cancel one.
Use action "schedule" to create a task (required fields: description, delay).
Use action "list" to see all pending tasks with their IDs.
Use action "cancel" with either id or description to delete a task (description does a partial match).
delay is a Go duration string: "30m", "1h", "24h", "2h30m", etc.
Set recurring to true for tasks that should repeat at the same interval.`
}

func (t Schedule) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"schedule", "list", "cancel"},
				"description": `"schedule" to create a task, "list" to show pending tasks, "cancel" to delete a task by ID.`,
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Task ID to cancel. Shown in list output and in the confirmation when a task is created.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "What the agent should do when the task fires.",
			},
			"delay": map[string]any{
				"type":        "string",
				"description": `How long until the task fires, e.g. "30m", "1h", "24h".`,
			},
			"recurring": map[string]any{
				"type":        "boolean",
				"description": "If true the task re-fires at the same interval indefinitely.",
			},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
}

func (t Schedule) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct {
		Action      string `json:"action"`
		Description string `json:"description"`
		Delay       string `json:"delay"`
		Recurring   bool   `json:"recurring"`
		ID          string `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	switch p.Action {
	case "list":
		tasks := t.Sched.List()
		if len(tasks) == 0 {
			return "No tasks scheduled.", nil
		}
		out := fmt.Sprintf("%d scheduled task(s):\n", len(tasks))
		for _, task := range tasks {
			recur := ""
			if task.Recurring {
				recur = fmt.Sprintf(" (recurring every %s)", task.Interval)
			}
			out += fmt.Sprintf("- [%s] fires at %s%s: %s\n",
				task.ID,
				task.ExecuteAt.Local().Format("2006-01-02 15:04:05"),
				recur,
				task.Description,
			)
		}
		return out, nil

	case "schedule":
		if p.Description == "" {
			return "", fmt.Errorf("description is required")
		}
		if p.Delay == "" {
			return "", fmt.Errorf("delay is required")
		}
		interval, err := time.ParseDuration(p.Delay)
		if err != nil {
			return "", fmt.Errorf("invalid delay %q: %w", p.Delay, err)
		}
		if interval <= 0 {
			return "", fmt.Errorf("delay must be positive")
		}
		fireAt := time.Now().Add(interval)
		task := scheduler.Task{
			ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
			Description: p.Description,
			ExecuteAt:   fireAt,
			Recurring:   p.Recurring,
			Interval:    interval,
		}
		if err := t.Sched.Add(task); err != nil {
			return "", fmt.Errorf("adding task: %w", err)
		}
		recur := ""
		if p.Recurring {
			recur = fmt.Sprintf(", recurring every %s", interval)
		}
		return fmt.Sprintf("Task scheduled (id: %s) for %s%s.", task.ID, fireAt.Local().Format("2006-01-02 15:04:05"), recur), nil

	case "cancel":
		if p.ID == "" && p.Description == "" {
			return "", fmt.Errorf("provide either id or description to identify the task to cancel")
		}
		id := p.ID
		if id == "" {
			// Find by description (case-insensitive substring match).
			needle := strings.ToLower(p.Description)
			for _, task := range t.Sched.List() {
				if strings.Contains(strings.ToLower(task.Description), needle) {
					id = task.ID
					break
				}
			}
			if id == "" {
				return "", fmt.Errorf("no task found matching description %q", p.Description)
			}
		}
		if err := t.Sched.Remove(id); err != nil {
			return "", fmt.Errorf("removing task: %w", err)
		}
		return fmt.Sprintf("Task %q cancelled.", id), nil

	default:
		return "", fmt.Errorf("unknown action %q: use \"schedule\", \"list\", or \"cancel\"", p.Action)
	}
}
