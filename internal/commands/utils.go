package commands

import (
	"os/exec"
	"strings"
)

type runResult struct {
	Stdout     string
	Stderr     string
	ReturnCode int
}

func runCmd(args ...string) runResult {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	rc := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			rc = exitErr.ExitCode()
		} else {
			rc = 1
		}
	}
	return runResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ReturnCode: rc,
	}
}

func gitCurrentBranch() string {
	r := runCmd("git", "rev-parse", "--abbrev-ref", "HEAD")
	if r.ReturnCode != 0 {
		return ""
	}
	return strings.TrimSpace(r.Stdout)
}

func gitDefaultBranch() string {
	r := runCmd("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	if r.ReturnCode == 0 {
		parts := strings.SplitN(strings.TrimSpace(r.Stdout), "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	// Fall back to checking known default names.
	for _, name := range []string{"main", "master"} {
		r2 := runCmd("git", "rev-parse", "--verify", name)
		if r2.ReturnCode == 0 {
			return name
		}
	}
	return "main"
}
