package agent

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/willardcsoriano/oui/internal/hooks"
	"github.com/willardcsoriano/oui/internal/providers"
	"github.com/willardcsoriano/oui/internal/tools"
)

// Run executes the agentic loop for a single user prompt.
// All output — streamed text and tool results — is written to w in real time.
func Run(ctx context.Context, prompt string, p providers.Provider, hooksCfg hooks.Config, w io.Writer) error {
	msgs := []providers.Message{{Role: "user", Content: prompt}}

	for {
		result, err := p.Stream(ctx, msgs, tools.Schemas, w)
		if err != nil {
			return fmt.Errorf("model: %w", err)
		}

		msgs = append(msgs, providers.Message{
			Role:      "assistant",
			Content:   result.Text,
			ToolCalls: result.ToolCalls,
		})

		if len(result.ToolCalls) == 0 {
			break
		}

		// Execute all tool calls in parallel, preserving order in the result slice.
		toolResults := executeParallel(ctx, result.ToolCalls, hooksCfg, w)
		for _, tr := range toolResults {
			msgs = append(msgs, providers.Message{
				Role:       "tool",
				ToolCallID: tr.id,
				Content:    tr.result,
			})
		}
	}

	return nil
}

type toolResult struct {
	id     string
	result string
}

func executeParallel(ctx context.Context, calls []providers.ToolCall, hooksCfg hooks.Config, w io.Writer) []toolResult {
	results := make([]toolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(i int, call providers.ToolCall) {
			defer wg.Done()
			results[i] = executeSingle(call, hooksCfg, w)
		}(i, call)
	}

	wg.Wait()
	return results
}

func executeSingle(call providers.ToolCall, hooksCfg hooks.Config, w io.Writer) toolResult {
	argsJSON := call.Function.Arguments
	toolName := call.Function.Name

	// PreToolUse hook — may block execution.
	if msg := hooks.RunPreToolUse(hooksCfg, toolName, argsJSON); msg != "" {
		fmt.Fprintf(w, "\n[hook blocked] %s\n", msg)
		return toolResult{id: call.ID, result: "[blocked by hook] " + msg}
	}

	fmt.Fprintf(w, "\n\033[33m⚙\033[0m \033[33m%s\033[0m\033[2m(%s)\033[0m\n", toolName, argsJSON)

	output, err := tools.Execute(toolName, argsJSON)
	if err != nil {
		output = fmt.Sprintf("error: %s", err)
	}

	hooks.RunPostToolUse(hooksCfg, toolName, argsJSON, output)

	// Show a brief preview of the result (first 200 chars).
	preview := output
	if len(preview) > 200 {
		preview = preview[:200] + "…"
	}
	fmt.Fprintf(w, "\033[2m  → %s\033[0m\n", preview)

	return toolResult{id: call.ID, result: output}
}
