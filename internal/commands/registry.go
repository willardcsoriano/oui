package commands

import (
	"fmt"
	"io"
	"strings"
)

// Handler is a command function. Returns (continueREPL bool, err error).
type Handler func(args string, w io.Writer) (bool, error)

var registry = map[string]Handler{
	"/gitignore": handleGitignore,
	"/wrap":      handleWrap,
	"/unwrap":    handleUnwrap,
	"/bye":       handleBye,
	"/privacy":   handlePrivacy,
	"/pr":        handlePR,
	"/merge":     handleMerge,
	"/commit":    handleCommit,
}

// Dispatch routes a slash command to its handler. Returns false if the REPL
// should exit, true to continue. Unknown commands print a usage hint.
func Dispatch(text string, w io.Writer) (bool, error) {
	parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
	cmd := parts[0]
	args := ""
	if len(parts) == 2 {
		args = parts[1]
	}

	handler, ok := registry[cmd]
	if !ok {
		fmt.Fprintf(w, "unknown command: %s\navailable: %s\n", cmd, knownCommands())
		return true, nil
	}
	return handler(args, w)
}

func knownCommands() string {
	cmds := make([]string, 0, len(registry))
	for k := range registry {
		cmds = append(cmds, k)
	}
	return strings.Join(cmds, "  ")
}
