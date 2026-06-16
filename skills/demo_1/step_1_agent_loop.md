# Skill: The Agent Loop + Tool System

> **Pacing:** Feed this skill to your agent ONE step at a time. After each "Stop here" marker, wait for the instructor before continuing to the next step.

## Context
We're building a claw (autonomous AI agent) in Go from scratch. This first step implements the core reasoning loop and tool system together - the heartbeat of any AI agent. We'll use the `openai-go` SDK pointed at an OpenAI-compatible endpoint.

### SDK reference: openai-go message types
The `openai-go` SDK uses union types for messages. Here's the pattern you'll need:

```go
import (
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    "github.com/openai/openai-go/shared"
)

// Creating a client pointed at the litellm proxy:
client := openai.NewClient(
    option.WithAPIKey(os.Getenv("CLAW_API_KEY")),
    option.WithBaseURL(os.Getenv("CLAW_BASE_URL")),
)

// Message types use a union struct with Of* fields:
systemMsg := openai.ChatCompletionMessageParamUnion{
    OfSystem: &openai.ChatCompletionSystemMessageParam{
        Content: openai.ChatCompletionSystemMessageParamContentUnion{
            OfString: openai.String("You are a helpful assistant."),
        },
    },
}

// For user messages:
userMsg := openai.ChatCompletionMessageParamUnion{
    OfUser: &openai.ChatCompletionUserMessageParam{
        Content: openai.ChatCompletionUserMessageParamContentUnion{
            OfString: openai.String(input),
        },
    },
}

// For assistant messages (after getting a response):
assistantMsg := openai.ChatCompletionMessageParamUnion{
    OfAssistant: &openai.ChatCompletionAssistantMessageParam{
        Content: openai.ChatCompletionAssistantMessageParamContentUnion{
            OfString: openai.String(response.Choices[0].Message.Content),
        },
    },
}

// For tool result messages:
toolMsg := openai.ChatCompletionMessageParamUnion{
    OfTool: &openai.ChatCompletionToolMessageParam{
        ToolCallID: toolCallID,
        Content: openai.ChatCompletionToolMessageParamContentUnion{
            OfString: openai.String(result),
        },
    },
}

// openai.String() wraps a string for optional fields
// The model name for our proxy: "claude-haiku" (default) or "claude-sonnet"
```

## Step 1: Project setup and first LLM call

Create a new Go module and install the OpenAI Go SDK.

```
go mod init myclaw
go get github.com/openai/openai-go
```

Create `main.go` with:
- A `main()` function that reads config from environment variables: `CLAW_BASE_URL` (LLM endpoint), `CLAW_API_KEY` (API key), `CLAW_MODEL` (model name, default `"claude-haiku"`)
- Create the client using `option.WithAPIKey()` and `option.WithBaseURL()` as shown above
- A conversation loop: read user input from stdin, send to the LLM, print the response, repeat
- Maintain conversation history as `[]openai.ChatCompletionMessageParamUnion`
- System prompt as the first message: "You are a helpful assistant."
- Exit cleanly on "exit" or Ctrl+C (use `signal.NotifyContext`)

### Acceptance criteria
- [ ] `go build ./...` succeeds
- [ ] Multi-turn conversation works (the LLM remembers earlier context)
- [ ] No hardcoded API keys anywhere
- [ ] Ctrl+C triggers graceful shutdown via context cancellation
- [ ] `go vet ./...` has no warnings

### Stop here
Have a short conversation with your agent. Ask a follow-up that requires context from the first answer. Verify it remembers.

## Step 2: Define the Tool interface and first tools

Create a `tools` package with a Go interface that all tools must implement:

```go
type Tool interface {
    Name() string
    Description() string
    Schema() map[string]any  // JSON schema for parameters
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}
```

Also create a `Registry` that holds registered tools and can look them up by name.

Then implement two tools:

**read_file:**
- Parameters: `{"path": "string"}` (required)
- Reads the file at the given path and returns its contents
- Should have a size limit to avoid blowing up the context window (you decide what's reasonable)

**list_directory:**
- Parameters: `{"path": "string"}` (optional, defaults to ".")
- Lists files and directories at the given path
- Choose a clear output format that helps the LLM distinguish files from directories

Register both tools in the registry.

### Acceptance criteria
- [ ] The `Tool` interface is defined with all four methods
- [ ] A `Registry` struct exists with `Register(Tool)` and `Get(name string) (Tool, bool)` methods
- [ ] Both tools implement the interface and work correctly
- [ ] `go build ./...` succeeds

### Stop here
Review the interface design. Is it clean? Does it follow Go conventions?

## Step 3: Wire tools into the agent loop

Modify the agent loop to:
1. Convert registered tools into the OpenAI function/tool format and include them in the chat completion request
2. After receiving a response, check if it contains tool calls (not just text)
3. If tool calls are present: for each tool call, look up the tool in the registry, execute it, and append the result as a tool message to the conversation history
4. Loop back and send the updated history to the LLM (the LLM will now see the tool results and formulate a response)
5. Only display text to the user when the LLM responds with a regular message (not a tool call)

The flow becomes:
```
User input -> LLM -> [tool call?] -> execute tool -> LLM -> [tool call?] -> ... -> text response -> display
```

**Important note on litellm proxy behavior:**
- The proxy may return `finish_reason: "stop"` even when tool calls are present (OpenAI's API returns `"tool_calls"`). Check for tool calls by their presence in the response, not by the finish reason.
- When building the assistant message for conversation history, construct `ChatCompletionMessageToolCallParam` manually rather than using `.ToParam()` on the accumulated tool calls - the accumulator may produce incomplete JSON when the proxy sends tool calls in a single chunk rather than as deltas.

**How to register tools with the LLM and handle tool calls:**
```go
// Converting tools to the LLM format:
toolDef := openai.ChatCompletionToolParam{
    Function: shared.FunctionDefinitionParam{
        Name:        tool.Name(),
        Description: openai.String(tool.Description()),
        Parameters:  shared.FunctionParameters(tool.Schema()),
    },
}

// Building assistant message with tool calls manually:
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
assistantMsg := openai.ChatCompletionMessageParamUnion{
    OfAssistant: &openai.ChatCompletionAssistantMessageParam{
        Content: openai.ChatCompletionAssistantMessageParamContentUnion{
            OfString: openai.String(content),
        },
        ToolCalls: tcParams,
    },
}
```

### Acceptance criteria
- [ ] The LLM can decide to call tools based on user input
- [ ] Tool results are fed back to the LLM correctly
- [ ] The LLM can chain multiple tool calls before responding
- [ ] If a tool errors, the error message is sent back to the LLM (so it can try something else)
- [ ] Regular conversation still works (not everything triggers a tool call)

### Stop here
Test: Ask your agent "what files are in the current directory?" - it should use list_directory. Ask "read the go.mod file" - it should use read_file. Ask "what's 2+2?" - it should just answer without tools.

## Mandatory review (5 minutes)

Before continuing to the next skill, run these two prompts with your agent:

> Generate a mermaid diagram showing: the agent loop, the tool registry, the LLM client, and how data flows between them when a tool call happens.

> Explain the tool dispatch flow step by step: what happens from user input to tool execution to final response?

Read both outputs. Do they match your understanding? This is not optional - understanding the architecture now prevents confusion in Demo 2.
