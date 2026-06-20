package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type openAIProvider struct {
	model   string
	apiKey  string
	baseURL string // e.g. http://localhost:11434/v1 for Ollama
}

// ── Request types ──────────────────────────────────────────────────────────────

type openAIRequest struct {
	Model    string         `json:"model"`
	Messages []openAIMsg    `json:"messages"`
	Tools    []openAITool   `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
}

type openAIMsg struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"` // string or null
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function openAIFuncCall   `json:"function"`
}

type openAIFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string           `json:"type"`
	Function openAIFuncSchema `json:"function"`
}

type openAIFuncSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// ── SSE chunk types ───────────────────────────────────────────────────────────

type openAIChunk struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Delta        openAIDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type openAIDelta struct {
	Content   string              `json:"content"`
	ToolCalls []openAIToolDelta   `json:"tool_calls"`
}

type openAIToolDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ── Provider implementation ───────────────────────────────────────────────────

func (p *openAIProvider) Stream(ctx context.Context, messages []Message, tools []ToolSchema, w io.Writer) (TurnResult, error) {
	oaiMsgs := toOpenAIMessages(messages)
	req := openAIRequest{
		Model:    p.model,
		Messages: oaiMsgs,
		Tools:    toOpenAITools(tools),
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return TurnResult{}, fmt.Errorf("openai: marshal request: %w", err)
	}

	url := p.baseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	} else {
		url = strings.TrimRight(url, "/") + "/chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return TurnResult{}, fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return TurnResult{}, fmt.Errorf("openai: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return TurnResult{}, fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, errBody)
	}

	return parseOpenAISSE(resp.Body, w)
}

func parseOpenAISSE(r io.Reader, w io.Writer) (TurnResult, error) {
	type toolAccum struct {
		id        string
		name      string
		argsBuf   strings.Builder
	}

	toolMap := map[int]*toolAccum{}
	var textBuf strings.Builder

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

		var chunk openAIChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			textBuf.WriteString(delta.Content)
			fmt.Fprint(w, delta.Content)
		}

		for _, tc := range delta.ToolCalls {
			acc := toolMap[tc.Index]
			if acc == nil {
				acc = &toolAccum{}
				toolMap[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			acc.argsBuf.WriteString(tc.Function.Arguments)
		}
	}

	if err := scanner.Err(); err != nil {
		return TurnResult{}, fmt.Errorf("openai: read SSE: %w", err)
	}

	if textBuf.Len() > 0 {
		fmt.Fprintln(w)
	}

	var result TurnResult
	result.Text = textBuf.String()
	for _, acc := range toolMap {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   acc.id,
			Type: "function",
			Function: ToolFunction{
				Name:      acc.name,
				Arguments: acc.argsBuf.String(),
			},
		})
	}
	return result, nil
}

// ── Format converters ─────────────────────────────────────────────────────────

func toOpenAIMessages(msgs []Message) []openAIMsg {
	result := make([]openAIMsg, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			result = append(result, openAIMsg{Role: "user", Content: m.Content})
		case "assistant":
			msg := openAIMsg{Role: "assistant"}
			if m.Content != "" {
				msg.Content = m.Content
			}
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, openAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFuncCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			result = append(result, msg)
		case "tool":
			result = append(result, openAIMsg{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		}
	}
	return result
}

func toOpenAITools(schemas []ToolSchema) []openAITool {
	tools := make([]openAITool, len(schemas))
	for i, s := range schemas {
		tools[i] = openAITool{
			Type: "function",
			Function: openAIFuncSchema{
				Name:        s.Name,
				Description: s.Description,
				Parameters:  s.Parameters,
			},
		}
	}
	return tools
}
