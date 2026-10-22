# Commands

## Overview

This is the reference for oui's built-in slash commands, typed at the REPL prompt as `/command`. Most of them automate the git and GitHub chores that come around every coding session: grouping and committing changes by concern, opening and merging pull requests, writing and reading a `HANDOFF.md` between sessions, checking tracked files for AI attribution, and generating a `.gitignore` for the detected stack. `/provider` and `/model` switch the active provider or model for the next turn without restarting. Any input that does not start with `/` goes to the agent. The GitHub commands shell out to the `gh` CLI, so it must be installed and authenticated.

## Reference

| Command | What it does |
|---|---|
| `/commit` | Groups unstaged changes by concern (directory heuristic), stages and commits each group with a generated Conventional Commits message |
| `/pr` | Pushes the current branch and opens a PR via `gh pr create --fill` |
| `/merge` | Lists open PRs targeting the default branch and enables auto-merge (squash, delete branch) on each; skips drafts |
| `/wrap` | Writes `HANDOFF.md` with branch, recent commits, diff stat, working tree status, and template sections for decisions/next-steps/blockers |
| `/unwrap` | Prints `HANDOFF.md` as written, for cold session resumption |
| `/bye` | Runs `/wrap` then exits the REPL |
| `/privacy` | Scans git-tracked files for AI attribution patterns (`Co-Authored-By`, "Generated with", etc.) and reports file + line number |
| `/gitignore` | Detects the project stack (Python, Node, Go, Rust, Java, Ruby, .NET) from files present and merges appropriate entries into `.gitignore` idempotently |
| `/provider [name]` | Prints the active provider when called with no arg; switches to `name` and resets the model to its configured default when an arg is provided |
| `/model [name]` | Prints the active model when called with no arg; switches to `name` within the current provider when an arg is provided |
