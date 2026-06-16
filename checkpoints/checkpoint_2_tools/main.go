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
	"github.com/openai/openai-go/shared"
	"myclaw/tools"
)

const defaultModel = "gpt-4o"

const systemPrompt = "You are a helpful assistant. You have access to tools for reading files and listing directories. Use them when appropriate to help the user."

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

	// Register tools.
	registry := tools.NewRegistry()
	for _, t := range []tools.Tool{
		tools.ReadFile{},
		tools.ListDirectory{},
	} {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register tool %s: %v\n", t.Name(), err)
			os.Exit(1)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := RunAgent(ctx, &client, model, registry); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}

// RunAgent runs the main agent loop with conversation history and tool support.
func RunAgent(ctx context.Context, client *openai.Client, model string, registry *tools.Registry) error {
	messages := []openai.ChatCompletionMessageParamUnion{
		{OfSystem: &openai.ChatCompletionSystemMessageParam{
			Content: openai.ChatCompletionSystemMessageParamContentUnion{
				OfString: openai.String(systemPrompt),
			},
		}},
	}

	toolDefs := buildToolDefs(registry)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

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

		messages = append(messages, openai.ChatCompletionMessageParamUnion{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(input),
				},
			},
		})

		var err error
		messages, err = agentTurn(ctx, client, model, messages, toolDefs, registry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}

// agentTurn handles one turn, potentially involving multiple tool-call rounds.
func agentTurn(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessageParamUnion, toolDefs []openai.ChatCompletionToolParam, registry *tools.Registry) ([]openai.ChatCompletionMessageParamUnion, error) {
	for {
		params := openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
		}
		if len(toolDefs) > 0 {
			params.Tools = toolDefs
		}

		resp, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return messages, fmt.Errorf("LLM call failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return messages, fmt.Errorf("no response from LLM")
		}

		choice := resp.Choices[0]
		content := choice.Message.Content
		toolCalls := choice.Message.ToolCalls

		// Append assistant message.
		assistantMsg := &openai.ChatCompletionAssistantMessageParam{
			Content: openai.ChatCompletionAssistantMessageParamContentUnion{
				OfString: openai.String(content),
			},
		}
		if len(toolCalls) > 0 {
			tcParams := make([]openai.ChatCompletionMessageToolCallParam, len(toolCalls))
			for i, tc := range toolCalls {
				tcParams[i] = tc.ToParam()
			}
			assistantMsg.ToolCalls = tcParams
		}
		messages = append(messages, openai.ChatCompletionMessageParamUnion{OfAssistant: assistantMsg})

		// If no tool calls, print the response and return.
		if choice.FinishReason != "tool_calls" || len(toolCalls) == 0 {
			fmt.Println(content)
			return messages, nil
		}

		// Execute tool calls.
		for _, tc := range toolCalls {
			tool, ok := registry.Get(tc.Function.Name)
			if !ok {
				errMsg := fmt.Sprintf("unknown tool: %s", tc.Function.Name)
				fmt.Fprintf(os.Stderr, "[tool error: %s]\n", errMsg)
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						ToolCallID: tc.ID,
						Content: openai.ChatCompletionToolMessageParamContentUnion{
							OfString: openai.String(errMsg),
						},
					},
				})
				continue
			}

			fmt.Fprintf(os.Stderr, "[calling tool: %s]\n", tc.Function.Name)

			result, err := tool.Execute(ctx, []byte(tc.Function.Arguments))
			if err != nil {
				errMsg := fmt.Sprintf("tool error: %v", err)
				fmt.Fprintf(os.Stderr, "[%s]\n", errMsg)
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						ToolCallID: tc.ID,
						Content: openai.ChatCompletionToolMessageParamContentUnion{
							OfString: openai.String(errMsg),
						},
					},
				})
				continue
			}

			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					ToolCallID: tc.ID,
					Content: openai.ChatCompletionToolMessageParamContentUnion{
						OfString: openai.String(result),
					},
				},
			})
		}

		// Loop back to send tool results to the LLM.
	}
}

// buildToolDefs converts registered tools into the OpenAI tool format.
func buildToolDefs(registry *tools.Registry) []openai.ChatCompletionToolParam {
	allTools := registry.All()
	defs := make([]openai.ChatCompletionToolParam, 0, len(allTools))
	for _, t := range allTools {
		defs = append(defs, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        t.Name(),
				Description: openai.String(t.Description()),
				Parameters:  shared.FunctionParameters(t.Schema()),
			},
		})
	}
	return defs
}
