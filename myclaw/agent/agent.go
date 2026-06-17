package agent

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"myclaw/tools"
)

// Message is a unit of work for the agent loop.
type Message struct {
	Content string
	Source  string                    // "cli", "web", "scheduler"
	ReplyTo func(string)              // called with each response text chunk
	Done    func()                    // called once the full response is complete
	OnTool  func(name, status string) // called on tool events
}

// RunAgent runs the agent loop, reading from msgCh until the channel is
// closed or ctx is cancelled. Each Message carries its own reply callbacks
// so that CLI, scheduler, and future WebSocket sources are handled uniformly.
func RunAgent(ctx context.Context, client *openai.Client, model string, systemPrompt string, registry *tools.Registry, msgCh <-chan Message) error {
	messages := []openai.ChatCompletionMessageParamUnion{
		systemMessage(systemPrompt),
	}
	toolDefs := buildToolDefs(registry)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nGoodbye!")
			return nil
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}
			// Non-CLI sources get a visual header so the user can see what fired.
			if msg.Source != "cli" {
				fmt.Printf("\n[%s] %s\n", msg.Source, msg.Content)
			}
			messages = append(messages, userMessage(msg.Content))
			var err error
			messages, err = agentTurn(ctx, client, model, messages, toolDefs, registry, msg)
			if err != nil {
				if msg.ReplyTo != nil {
					msg.ReplyTo(fmt.Sprintf("error: %v", err))
				}
			}
			if msg.Done != nil {
				msg.Done()
			}
		}
	}
}

// MakeSendFn returns a function that sends text to the agent via ch as an
// "a2a" source and blocks until the agent finishes, returning the full reply.
// Safe to call from multiple goroutines concurrently.
func MakeSendFn(ch chan<- Message) func(text string) (string, error) {
	return func(text string) (string, error) {
		var buf strings.Builder
		doneCh := make(chan struct{})
		ch <- Message{
			Content: text,
			Source:  "a2a",
			ReplyTo: func(t string) { buf.WriteString(t) },
			Done:    func() { close(doneCh) },
			OnTool:  func(_, _ string) {},
		}
		<-doneCh
		return buf.String(), nil
	}
}

// StartCLIInput reads lines from stdin and sends them to ch as "cli" Messages.
// It blocks after each send until Done() is called so the prompt is only
// reprinted after the agent finishes responding, keeping the terminal clean.
// It runs until stdin is closed or ctx is cancelled.
func StartCLIInput(ctx context.Context, ch chan<- Message) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Print("> ")
			continue
		}
		if input == "exit" {
			fmt.Println("Goodbye!")
			return
		}

		doneCh := make(chan struct{})
		select {
		case <-ctx.Done():
			return
		case ch <- Message{
			Content: input,
			Source:  "cli",
			ReplyTo: func(text string) { fmt.Print(text) },
			Done: func() {
				fmt.Println()
				close(doneCh)
			},
			OnTool: func(name, status string) {
				fmt.Fprintf(os.Stderr, "[tool %s: %s]\n", name, status)
			},
		}:
		}

		// Wait for the agent to finish before printing the next prompt.
		select {
		case <-doneCh:
		case <-ctx.Done():
			return
		}
		fmt.Print("> ")
	}
}

func agentTurn(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessageParamUnion, toolDefs []openai.ChatCompletionToolParam, registry *tools.Registry, msg Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	for {
		params := openai.ChatCompletionNewParams{
			Model:    model,
			Messages: messages,
		}
		if len(toolDefs) > 0 {
			params.Tools = toolDefs
		}

		sseStream := client.Chat.Completions.NewStreaming(ctx, params)
		acc := openai.ChatCompletionAccumulator{}
		events := streamResponse(ctx, &acc, sseStream)

		var fullContent string
		var toolCalls []openai.ChatCompletionMessageToolCall

		for event := range events {
			switch event.Type {
			case EventText:
				if msg.ReplyTo != nil {
					msg.ReplyTo(event.Text)
				}
			case EventDone:
				fullContent = event.FullContent
				toolCalls = event.ToolCalls
			case EventError:
				return messages, event.Err
			}
		}

		messages = append(messages, assistantMessage(fullContent, toolCalls))

		if len(toolCalls) == 0 {
			return messages, nil
		}

		// Separate streaming text from tool-call output with a newline.
		if msg.ReplyTo != nil {
			msg.ReplyTo("\n")
		}

		for _, tc := range toolCalls {
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			tool, ok := registry.Get(toolName)
			if !ok {
				errMsg := fmt.Sprintf("unknown tool: %s", toolName)
				if msg.OnTool != nil {
					msg.OnTool(toolName, "unknown")
				}
				messages = append(messages, toolMessage(tc.ID, errMsg))
				continue
			}

			slog.Info("tool call", "tool", toolName, "source", msg.Source)
			if msg.OnTool != nil {
				msg.OnTool(toolName, "calling")
			}

			result, err := tool.Execute(ctx, []byte(toolArgs))
			if err != nil {
				slog.Warn("tool error", "tool", toolName, "err", err)
				errMsg := fmt.Sprintf("tool error: %v", err)
				if msg.OnTool != nil {
					msg.OnTool(toolName, "error")
				}
				messages = append(messages, toolMessage(tc.ID, errMsg))
				continue
			}

			slog.Info("tool done", "tool", toolName)
			if msg.OnTool != nil {
				msg.OnTool(toolName, "done")
			}
			messages = append(messages, toolMessage(tc.ID, result))
		}
	}
}

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

func systemMessage(content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionMessageParamUnion{
		OfSystem: &openai.ChatCompletionSystemMessageParam{
			Content: openai.ChatCompletionSystemMessageParamContentUnion{
				OfString: openai.String(content),
			},
		},
	}
}

func userMessage(content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: openai.String(content),
			},
		},
	}
}

func assistantMessage(content string, toolCalls []openai.ChatCompletionMessageToolCall) openai.ChatCompletionMessageParamUnion {
	msg := &openai.ChatCompletionAssistantMessageParam{
		Content: openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: openai.String(content),
		},
	}
	if len(toolCalls) > 0 {
		tcParams := make([]openai.ChatCompletionMessageToolCallParam, len(toolCalls))
		for i, tc := range toolCalls {
			tcParams[i] = openai.ChatCompletionMessageToolCallParam{
				ID:   tc.ID,
				Type: "function",
				Function: openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		msg.ToolCalls = tcParams
	}
	return openai.ChatCompletionMessageParamUnion{
		OfAssistant: msg,
	}
}

func toolMessage(toolCallID, content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionMessageParamUnion{
		OfTool: &openai.ChatCompletionToolMessageParam{
			ToolCallID: toolCallID,
			Content: openai.ChatCompletionToolMessageParamContentUnion{
				OfString: openai.String(content),
			},
		},
	}
}
