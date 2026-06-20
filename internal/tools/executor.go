package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/willardcsoriano/wi/internal/providers"
)

var Schemas = []providers.ToolSchema{
	{
		Name:        "read_file",
		Description: "Read the contents of a file at the given path.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Relative or absolute file path to read.",
				},
			},
			"required": []string{"path"},
		},
	},
	{
		Name:        "write_file",
		Description: "Write content to a file, creating it or overwriting it.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to write.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file.",
				},
			},
			"required": []string{"path", "content"},
		},
	},
	{
		Name:        "bash",
		Description: "Execute a shell command and return combined stdout and stderr.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to run.",
				},
			},
			"required": []string{"command"},
		},
	},
}

// denyPatterns are unconditional blocks — not user-configurable.
var denyPatterns = []string{
	"git push --force",
	"git push -f",
	"git reset --hard",
	"git clean -f",
	"rm -rf",
	":(){ :|:& };:", // fork bomb
	"dd if=",
	"mkfs",
	"> /dev/",
}

func checkDenyList(command string) error {
	lower := strings.ToLower(command)
	for _, pat := range denyPatterns {
		if strings.Contains(lower, pat) {
			return fmt.Errorf("command blocked by deny list: %q matches pattern %q", command, pat)
		}
	}
	return nil
}

// Execute dispatches a tool call by name with JSON-encoded args.
func Execute(name, argsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed tool args: %w", err)
	}

	switch name {
	case "read_file":
		return readFile(args)
	case "write_file":
		return writeFile(args)
	case "bash":
		return bash(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func readFile(args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("read_file: path must be a non-empty string")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	return string(data), nil
}

func writeFile(args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("write_file: path must be a non-empty string")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("write_file: content must be a string")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write_file: %w", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}

func bash(args map[string]any) (string, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("bash: command must be a non-empty string")
	}
	if err := checkDenyList(command); err != nil {
		return "", err
	}
	cmd := exec.Command("bash", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Return output even on non-zero exit — the model needs to see stderr.
		return string(out), nil
	}
	return string(out), nil
}
