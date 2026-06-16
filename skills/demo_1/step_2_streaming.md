# Skill: Streaming Responses

> **Pacing:** Feed this skill to your agent ONE step at a time. After each "Stop here" marker, wait for the instructor before continuing to the next step.

## Context
Our agent works but responses appear all at once after the LLM finishes generating. For a good user experience (and to feel like a real claw), we need streaming - tokens appearing as they're generated. This is where Go's concurrency model shines.

## Step 1: Basic streaming

Replace the non-streaming chat completion call with the streaming variant.

- Use the streaming API from `openai-go` (check the SDK docs for the streaming method)
- As each chunk arrives, print the text delta to stdout immediately (no newline between chunks)
- After the stream completes, print a final newline
- The conversation history still needs to capture the full response (accumulate chunks)

### Acceptance criteria
- [ ] Responses appear token by token in the terminal
- [ ] The full response is still appended to conversation history
- [ ] Error handling works (stream errors don't crash the agent)
- [ ] The user experience is noticeably better than before

### Stop here
Ask your agent a question that requires a long answer. Verify tokens stream in smoothly.

## Step 2: Handle tool calls in streaming mode

Streaming with tool calls is trickier - tool call deltas arrive across multiple chunks and need to be assembled before execution.

- Use `openai.ChatCompletionAccumulator` to handle assembly - it collects deltas across chunks
- After the stream ends, check `acc.Choices[0].Message.ToolCalls` for assembled tool calls
- Text deltas and tool call deltas can be interleaved - handle both
- **Proxy quirk:** litellm may send the entire tool call in a single chunk rather than as incremental deltas. The accumulator handles this fine, but be aware when debugging.

**Streaming pattern:**
```go
sseStream := client.Chat.Completions.NewStreaming(ctx, params)
acc := openai.ChatCompletionAccumulator{}

for sseStream.Next() {
    chunk := sseStream.Current()
    acc.AddChunk(chunk)
    
    if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
        fmt.Print(chunk.Choices[0].Delta.Content)  // stream to terminal
    }
}

// After stream ends:
fullContent := acc.Choices[0].Message.Content
toolCalls := acc.Choices[0].Message.ToolCalls
```

### Acceptance criteria
- [ ] Tool calls work correctly in streaming mode
- [ ] Tool call arguments are fully assembled before execution (no partial JSON)
- [ ] After tool execution, the follow-up response also streams
- [ ] Multi-tool-call responses work (LLM calls two tools in one turn)

### Stop here
Test: Ask "what files are in this directory and show me the contents of go.mod" - the agent should call both tools and then stream a response about the results.

## Step 3: Goroutine and channel architecture (optional enhancement)

If time permits, refactor the streaming to use a dedicated goroutine and channel:

- A goroutine reads from the SSE stream and sends chunks into a channel
- The main loop reads from the channel and handles display/accumulation
- This separates concerns: the stream reader doesn't care about display logic

This pattern will be useful later when we add the web UI (the channel can feed a WebSocket instead of stdout).

### Acceptance criteria
- [ ] Stream reading and display are in separate goroutines
- [ ] Communication happens via a typed channel (e.g., `chan StreamEvent`)
- [ ] Clean shutdown: when context is cancelled, the goroutine exits and the channel is closed
- [ ] No goroutine leaks (verify with race detector: `go run -race .`)

### Stop here
Run `go run -race .` and have a conversation. Verify no race conditions are reported.
