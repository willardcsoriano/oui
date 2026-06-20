package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func handleCommit(_ string, w io.Writer) (bool, error) {
	r := runCmd("git", "diff", "--name-only")
	if r.ReturnCode != 0 {
		fmt.Fprintln(w, "error: not in a git repository")
		return true, nil
	}

	// Also pick up new untracked files.
	untrackedR := runCmd("git", "ls-files", "--others", "--exclude-standard")
	allFiles := strings.Fields(r.Stdout + " " + untrackedR.Stdout)
	if len(allFiles) == 0 {
		fmt.Fprintln(w, "nothing to commit")
		return true, nil
	}

	// Group by concern key.
	groups := map[string][]string{}
	for _, f := range allFiles {
		key := concernKey(f)
		groups[key] = append(groups[key], f)
	}

	for key, files := range groups {
		// Stage this group.
		stageArgs := append([]string{"git", "add"}, files...)
		stageR := runCmd(stageArgs...)
		if stageR.ReturnCode != 0 {
			fmt.Fprintf(w, "error: git add failed for %s: %s\n", key, strings.TrimSpace(stageR.Stderr))
			continue
		}

		msg := buildCommitMessage(key, files)
		commitR := runCmd("git", "commit", "-m", msg)
		if commitR.ReturnCode != 0 {
			fmt.Fprintf(w, "error: commit failed for %s: %s\n", key, strings.TrimSpace(commitR.Stderr))
		} else {
			fmt.Fprintf(w, "committed (%s): %s\n", key, msg)
		}
	}

	return true, nil
}

// concernKey maps a file path to a concern name used for grouping commits.
// Files under src/wi/X/ → X; top-level files → their name; otherwise top-level dir.
func concernKey(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) == 1 {
		return stripExt(parts[0])
	}
	// src/wi/<module>/... → <module>
	if len(parts) >= 3 && parts[0] == "src" {
		return parts[2]
	}
	return parts[0]
}

func stripExt(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext)
}

func buildCommitMessage(concern string, files []string) string {
	t := commitType(files)
	scope := commitScope(concern, files)
	desc := shortDesc(concern, files)

	if scope != "" {
		return fmt.Sprintf("%s(%s): %s", t, scope, desc)
	}
	return fmt.Sprintf("%s: %s", t, desc)
}

func commitType(files []string) string {
	for _, f := range files {
		if strings.HasPrefix(f, "test") || strings.Contains(f, "/test") {
			return "test"
		}
		if strings.HasSuffix(f, ".md") {
			return "docs"
		}
		if strings.Contains(f, "workflow") || strings.Contains(f, ".github") {
			return "ci"
		}
	}
	return "feat"
}

func commitScope(concern string, files []string) string {
	// Use concern as scope only when it's a meaningful sub-package name.
	if concern == "." || concern == "src" || concern == "tests" || concern == "docs" {
		return ""
	}
	return concern
}

func shortDesc(concern string, files []string) string {
	if len(files) == 1 {
		base := filepath.Base(files[0])
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return fmt.Sprintf("%s (%d files)", concern, len(files))
}
