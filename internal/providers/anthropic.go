package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const anthropicBaseURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"
const anthropicMaxTokens = 8192

type anthropicProvider struct {
	model  string
	apiKey string
}

// ── Request types ──────────────────────────────────────────────────────────────

type anthropicRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"` // extracted from role:system messages
	Messages  []anthropicMsg  `json:"messages"`
	Tools     []anthropicTool `json:"tools,omitempty"`
	Stream    bool            `json:"stream"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []anthropicContent
}

type anthropicContent struct {
	Type string `json:"type"`
	// text
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
	// tool_result
	ToolUseID   string `json:"tool_use_id,omitempty"`
	ToolContent string `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

// ── SSE event types ───────────────────────────────────────────────────────────

type anthropicEvent struct {
	Type         string                 `json:"type"`
	Index        int                    `json:"index"`
	ContentBlock *anthropicBlock        `json:"content_block"`
	Delta        *anthropicDelta        `json:"delta"`
}

type anthropicBlock struct {
	Type string `json:"type"` // "text" or "tool_use"
	ID   string `json:"id"`
	Name string `json:"name"`
}

type anthropicDelta struct {
	Type        string `json:"type"` // "text_delta" or "input_json_delta"
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
}

// ── Provider implementation ───────────────────────────────────────────────────

func (p *anthropicProvider) Stream(ctx context.Context, messages []Message, tools []ToolSchema, w io.Writer) (TurnResult, error) {
	req := anthropicRequest{
		Model:     p.model,
		MaxTokens: anthropicMaxTokens,
		System:    extractSystemMsg(messages),
		Messages:  toAnthropicMessages(messages),
		Tools:     toAnthropicTools(tools),
		Stream:    true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return TurnResult{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicBaseURL, bytes.NewReader(body))
	if err != nil {
		return TurnResult{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return TurnResult{}, fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return TurnResult{}, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, errBody)
	}

	return parseAnthropicSSE(resp.Body, w)
}

func parseAnthropicSSE(r io.Reader, w io.Writer) (TurnResult, error) {
	// Track state per content block index.
	type blockState struct {
		isToolUse bool
		id        string
		name      string
		argsBuf   strings.Builder
	}

	blocks := map[int]*blockState{}
	var textBuf strings.Builder
	var result TurnResult

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var ev anthropicEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "content_block_start":
			if ev.ContentBlock == nil {
				continue
			}
			b := &blockState{}
			if ev.ContentBlock.Type == "tool_use" {
				b.isToolUse = true
				b.id = ev.ContentBlock.ID
				b.name = ev.ContentBlock.Name
			}
			blocks[ev.Index] = b

		case "content_block_delta":
			if ev.Delta == nil {
				continue
			}
			b := blocks[ev.Index]
			if b == nil {
				continue
			}
			switch ev.Delta.Type {
			case "text_delta":
				textBuf.WriteString(ev.Delta.Text)
				fmt.Fprint(w, ev.Delta.Text)
			case "input_json_delta":
				if b.isToolUse {
					b.argsBuf.WriteString(ev.Delta.PartialJSON)
				}
			}

		case "message_stop":
			// Flush remaining text.
			if textBuf.Len() > 0 {
				fmt.Fprintln(w)
			}
			result.Text = textBuf.String()
			// Collect tool calls.
			for _, b := range blocks {
				if !b.isToolUse {
					continue
				}
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					ID:   b.id,
					Type: "function",
					Function: ToolFunction{
						Name:      b.name,
						Arguments: b.argsBuf.String(),
					},
				})
			}
			return result, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return TurnResult{}, fmt.Errorf("anthropic: read SSE: %w", err)
	}

	// Reached EOF without message_stop — treat accumulated state as result.
	if textBuf.Len() > 0 {
		fmt.Fprintln(w)
	}
	result.Text = textBuf.String()
	for _, b := range blocks {
		if !b.isToolUse {
			continue
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   b.id,
			Type: "function",
			Function: ToolFunction{
				Name:      b.name,
				Arguments: b.argsBuf.String(),
			},
		})
	}
	return result, nil
}

// ── Format converters ─────────────────────────────────────────────────────────

// extractSystemMsg returns the content of the first system-role message.
// Anthropic uses a top-level "system" field rather than a messages entry.
func extractSystemMsg(msgs []Message) string {
	for _, m := range msgs {
		if m.Role == "system" {
			return m.Content
		}
	}
	return ""
}

func toAnthropicMessages(msgs []Message) []anthropicMsg {
	var result []anthropicMsg
	i := 0
	for i < len(msgs) {
		m := msgs[i]
		switch m.Role {
		case "user":
			result = append(result, anthropicMsg{Role: "user", Content: m.Content})
			i++

		case "assistant":
			var content []anthropicContent
			if m.Content != "" {
				content = append(content, anthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			result = append(result, anthropicMsg{Role: "assistant", Content: content})
			i++

		case "tool":
			// Batch consecutive tool-result messages into one user message.
			var toolResults []anthropicContent
			for i < len(msgs) && msgs[i].Role == "tool" {
				toolResults = append(toolResults, anthropicContent{
					Type:        "tool_result",
					ToolUseID:   msgs[i].ToolCallID,
					ToolContent: msgs[i].Content,
				})
				i++
			}
			result = append(result, anthropicMsg{Role: "user", Content: toolResults})

		default:
			i++
		}
	}
	return result
}

func toAnthropicTools(schemas []ToolSchema) []anthropicTool {
	tools := make([]anthropicTool, len(schemas))
	for i, s := range schemas {
		tools[i] = anthropicTool{
			Name:        s.Name,
			Description: s.Description,
			InputSchema: s.Parameters,
		}
	}
	return tools
}

func newAnthropicProvider(model string) (*anthropicProvider, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	return &anthropicProvider{model: model, apiKey: key}, nil
}
