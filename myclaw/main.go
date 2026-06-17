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
	"myclaw/tools"
)

const defaultModel = "gpt-4o"

const systemPrompt = "You are a helpful assistant. You have access to tools for reading files, listing directories, writing files, and running shell commands. Use them when appropriate to help the user."

func main() {
	baseURL := os.Getenv("CLAW_BASE_URL")
	apiKey := os.Getenv("CLAW_API_KEY")
	model := os.Getenv("CLAW_MODEL")

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

	registry := tools.NewRegistry()
	for _, t := range []tools.Tool{
		tools.ReadFile{},
		tools.ListDirectory{},
		tools.WriteFile{},
		tools.RunCommand{},
		tools.Remember{Store: memStore},
		tools.Recall{Store: memStore},
	} {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register tool %s: %v\n", t.Name(), err)
			os.Exit(1)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := agent.RunAgent(ctx, &client, model, systemPrompt, registry); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}
