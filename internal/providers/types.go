package providers

import (
	"context"
	"io"
)

type Message struct {
	Role       string
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}

type ToolCall struct {
	ID       string
	Type     string
	Function ToolFunction
}

type ToolFunction struct {
	Name      string
	Arguments string // JSON-encoded
}

type ToolSchema struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type TurnResult struct {
	Text      string
	ToolCalls []ToolCall
}

type Provider interface {
	// Stream sends messages to the model, writes text chunks to w as they arrive,
	// and returns the accumulated text and tool calls when the turn is complete.
	Stream(ctx context.Context, messages []Message, tools []ToolSchema, w io.Writer) (TurnResult, error)
}
