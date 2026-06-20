package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type prEntry struct {
	Number      int    `json:"number"`
	HeadRefName string `json:"headRefName"`
	IsDraft     bool   `json:"isDraft"`
}

func handleMerge(_ string, w io.Writer) (bool, error) {
	def := gitDefaultBranch()

	r := runCmd("gh", "pr", "list",
		"--state", "open",
		"--base", def,
		"--json", "number,headRefName,isDraft",
	)
	if r.ReturnCode != 0 {
		fmt.Fprintf(w, "error: gh pr list failed: %s\n", strings.TrimSpace(r.Stderr))
		return true, nil
	}

	var prs []prEntry
	if err := json.Unmarshal([]byte(r.Stdout), &prs); err != nil {
		fmt.Fprintf(w, "error: parse pr list: %s\n", err)
		return true, nil
	}

	if len(prs) == 0 {
		fmt.Fprintf(w, "no open PRs targeting %s\n", def)
		return true, nil
	}

	for _, pr := range prs {
		if pr.IsDraft {
			fmt.Fprintf(w, "skipping draft PR #%d (%s)\n", pr.Number, pr.HeadRefName)
			continue
		}
		mergeR := runCmd("gh", "pr", "merge", fmt.Sprintf("%d", pr.Number),
			"--auto", "--squash", "--delete-branch",
		)
		if mergeR.ReturnCode != 0 {
			fmt.Fprintf(w, "error: could not queue PR #%d: %s\n", pr.Number, strings.TrimSpace(mergeR.Stderr))
		} else {
			fmt.Fprintf(w, "queued PR #%d (%s) — merge-when-green\n", pr.Number, pr.HeadRefName)
		}
	}

	return true, nil
}
