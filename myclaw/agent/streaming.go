package agent

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

type StreamEventType int

const (
	EventText StreamEventType = iota
	EventToolCallDelta
	EventDone
	EventError
)

type StreamEvent struct {
	Type         StreamEventType
	Text         string
	ToolCalls    []openai.ChatCompletionMessageToolCall
	FullContent  string
	FinishReason string
	Err          error
}

func streamResponse(ctx context.Context, stream *openai.ChatCompletionAccumulator, sseStream interface {
	Next() bool
	Current() openai.ChatCompletionChunk
	Err() error
	Close() error
}) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)
		defer sseStream.Close()

		for sseStream.Next() {
			select {
			case <-ctx.Done():
				ch <- StreamEvent{Type: EventError, Err: ctx.Err()}
				return
			default:
			}

			chunk := sseStream.Current()
			if !stream.AddChunk(chunk) {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta

			if delta.Content != "" {
				ch <- StreamEvent{Type: EventText, Text: delta.Content}
			}
		}

		if err := sseStream.Err(); err != nil {
			ch <- StreamEvent{Type: EventError, Err: fmt.Errorf("stream error: %w", err)}
			return
		}

		event := StreamEvent{Type: EventDone}
		if len(stream.Choices) > 0 {
			event.FullContent = stream.Choices[0].Message.Content
			event.ToolCalls = stream.Choices[0].Message.ToolCalls
			event.FinishReason = stream.Choices[0].FinishReason
		}
		ch <- event
	}()

	return ch
}
