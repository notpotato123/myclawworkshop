package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"myclaw/agent"
	"myclaw/memory"
	"myclaw/scheduler"
	"myclaw/tools"
	"myclaw/web"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultModel = "qwen"

const baseSystemPrompt = "You are a helpful assistant. You have access to tools for reading files, listing directories, writing files, and running shell commands. Use them when appropriate to help the user."

// memoryTokenBudget caps injected memories at ~2000 tokens (≈8000 chars).
const memoryTokenBudget = 8000

func main() {
	baseURL := os.Getenv("CLAW_BASE_URL")
	apiKey := os.Getenv("CLAW_API_KEY")
	model := os.Getenv("CLAW_MODEL")
	port := os.Getenv("CLAW_PORT")

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "CLAW_API_KEY environment variable is required")
		os.Exit(1)
	}
	if model == "" {
		model = defaultModel
	}
	if port == "" {
		port = "8080"
	}

	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)

	memStore, err := memory.NewStore(".claw_memory")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create memory store: %v\n", err)
		os.Exit(1)
	}

	// msgCh is the single channel all input sources feed into the agent loop.
	msgCh := make(chan agent.Message, 16)

	// hub broadcasts agent output to all connected WebSocket clients.
	hub := web.NewHub()

	sched, err := scheduler.New("scheduler/tasks.json", func(description string) {
		// Notify the web UI that a scheduled task is firing.
		hub.Broadcast(web.SystemMsg("Scheduled task: " + description))
		msgCh <- agent.Message{
			Content: description,
			Source:  "scheduler",
			ReplyTo: func(text string) {
				fmt.Print(text)
				hub.Broadcast(web.ChunkMsg(text))
			},
			Done: func() {
				fmt.Println()
				hub.Broadcast(web.DoneMsg())
			},
			OnTool: func(name, status string) {
				fmt.Fprintf(os.Stderr, "[tool %s: %s]\n", name, status)
				hub.Broadcast(web.ToolMsg(name, status))
			},
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create scheduler: %v\n", err)
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	for _, t := range []tools.Tool{
		tools.ReadFile{},
		tools.ListDirectory{},
		tools.WriteFile{},
		tools.RunCommand{},
		tools.Remember{Store: memStore},
		tools.Recall{Store: memStore},
		tools.Schedule{Sched: sched},
	} {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register tool %s: %v\n", t.Name(), err)
			os.Exit(1)
		}
	}

	prompt := baseSystemPrompt
	if memories := memStore.Dump(memoryTokenBudget); memories != "" {
		prompt += "\n\n## Memories\nThe following information was saved from previous sessions:\n" + memories
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go sched.Run(ctx)
	go agent.StartCLIInput(ctx, msgCh)
	go func() {
		srv := web.NewServer(hub, msgCh)
		if err := srv.Start(port); err != nil {
			fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		}
	}()

	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := agent.RunAgent(ctx, &client, model, prompt, registry, msgCh); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}
