package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"myclaw/agent"
	"myclaw/memory"
	"myclaw/scheduler"
	"myclaw/tools"
	"myclaw/web"
)

const defaultModel = "gpt-4o"

const baseSystemPrompt = "You are a helpful assistant. You have access to tools for reading files, listing directories, writing files, and running shell commands. Use them when appropriate to help the user."

// memoryTokenBudget caps injected memories at ~2000 tokens (≈8000 chars).
const memoryTokenBudget = 8000

func main() {
	baseURL := os.Getenv("CLAW_BASE_URL")
	apiKey := os.Getenv("CLAW_API_KEY")
	model := os.Getenv("CLAW_MODEL")
	port := os.Getenv("CLAW_PORT")
	if port == "" {
		port = "8080"
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "CLAW_API_KEY environment variable is required")
		os.Exit(1)
	}

	if model == "" {
		model = defaultModel
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	memStore, err := memory.NewStore(".claw_memory")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create memory store: %v\n", err)
		os.Exit(1)
	}

	// Buffered so the scheduler callback never blocks when the agent is busy.
	msgCh := make(chan agent.Message, 16)

	sched, err := scheduler.New("scheduler/tasks.json", func(description string) {
		msgCh <- agent.Message{
			Content: description,
			Source:  "scheduler",
			ReplyTo: func(text string) { fmt.Print(text) },
			Done:    func() { fmt.Println() },
			OnTool: func(name, status string) {
				fmt.Fprintf(os.Stderr, "[tool %s: %s]\n", name, status)
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
		if err := web.Start(port); err != nil {
			fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		}
	}()

	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := agent.RunAgent(ctx, &client, model, prompt, registry, msgCh); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}
