package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultModel = "gpt-4o"

const systemPrompt = "You are a helpful assistant."

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

	// Build client options.
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	// Set up context with Ctrl+C handling.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := RunAgent(ctx, &client, model); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}

// RunAgent runs the main agent loop with conversation history.
func RunAgent(ctx context.Context, client *openai.Client, model string) error {
	// Build the conversation history starting with the system message.
	messages := []openai.ChatCompletionMessageParamUnion{
		{OfSystem: &openai.ChatCompletionSystemMessageParam{
			Content: openai.ChatCompletionSystemMessageParamContentUnion{
				OfString: openai.String(systemPrompt),
			},
		}},
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		// Check context before blocking on input.
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			fmt.Println()
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" {
			fmt.Println("Goodbye!")
			return nil
		}

		// Append user message.
		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(input),
				},
			},
		})

		// Send to LLM.
		resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling LLM: %v\n", err)
			continue
		}

		if len(resp.Choices) == 0 {
			fmt.Fprintln(os.Stderr, "No response from LLM")
			continue
		}

		content := resp.Choices[0].Message.Content
		fmt.Println(content)

		// Append assistant message to history.
		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(content),
				},
			},
		})
	}
}
