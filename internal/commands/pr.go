package commands

import (
	"fmt"
	"io"
	"strings"
)

func handlePR(_ string, w io.Writer) (bool, error) {
	branch := gitCurrentBranch()
	def := gitDefaultBranch()

	if branch == "" {
		fmt.Fprintln(w, "error: not on a branch (detached HEAD?)")
		return true, nil
	}
	if branch == def {
		fmt.Fprintf(w, "error: already on default branch (%s) — create a feature branch first\n", def)
		return true, nil
	}

	statusR := runCmd("git", "status", "--short")
	if strings.TrimSpace(statusR.Stdout) != "" {
		fmt.Fprintln(w, "warning: uncommitted changes present — commit or stash before opening a PR")
	}

	fmt.Fprintf(w, "pushing %s…\n", branch)
	pushR := runCmd("git", "push", "-u", "origin", branch)
	if pushR.ReturnCode != 0 {
		fmt.Fprintf(w, "error: push failed: %s\n", strings.TrimSpace(pushR.Stderr))
		return true, nil
	}

	fmt.Fprintln(w, "creating PR…")
	prR := runCmd("gh", "pr", "create", "--fill", "--base", def)
	if prR.ReturnCode != 0 {
		fmt.Fprintf(w, "error: gh pr create failed: %s\n", strings.TrimSpace(prR.Stderr))
		return true, nil
	}

	fmt.Fprintf(w, "PR opened: %s\n", strings.TrimSpace(prR.Stdout))
	return true, nil
}
