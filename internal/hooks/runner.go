package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunPreToolUse runs matching pre-tool-use hooks. Returns a non-empty block message
// if any hook exits non-zero (tool execution should be skipped).
func RunPreToolUse(cfg Config, toolName, argsJSON string) string {
	for _, h := range cfg.PreToolUse {
		if !matchGlob(h.Match, toolName) {
			continue
		}
		cmd := exec.Command("bash", "-c", h.Command)
		cmd.Env = append(os.Environ(),
			"WI_TOOL_NAME="+toolName,
			"WI_TOOL_ARGS="+argsJSON,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("hook blocked tool %q: %s", toolName, string(out))
		}
	}
	return ""
}

// RunPostToolUse fires matching post-tool-use hooks. Failures are silently ignored.
func RunPostToolUse(cfg Config, toolName, argsJSON, result string) {
	for _, h := range cfg.PostToolUse {
		if !matchGlob(h.Match, toolName) {
			continue
		}
		cmd := exec.Command("bash", "-c", h.Command)
		cmd.Env = append(os.Environ(),
			"WI_TOOL_NAME="+toolName,
			"WI_TOOL_ARGS="+argsJSON,
			"WI_TOOL_RESULT="+result,
		)
		_ = cmd.Run()
	}
}

// RunLifecycle executes lifecycle commands sequentially.
func RunLifecycle(commands []string) {
	for _, c := range commands {
		cmd := exec.Command("bash", "-c", c)
		cmd.Env = os.Environ()
		_ = cmd.Run()
	}
}

func matchGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

func argsToJSON(args map[string]any) string {
	b, _ := json.Marshal(args)
	return string(b)
}
