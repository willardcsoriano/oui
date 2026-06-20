## Overview

This file is the single source of truth for where `wi` is right now — what has landed, what is in flight, what is next, and any decisions or blockers worth remembering. It is updated at the end of each work session or whenever a phase boundary is crossed. The roadmap in `docs/ROADMAP.md` defines the full plan; this file tracks execution against it. If this file and the roadmap disagree, this file reflects reality and the roadmap reflects intent.

---

## Table of Contents

- [Overview](#overview)
- [Current Status](#current-status)
- [What Has Landed (main)](#what-has-landed-main)
  - [Phase 0 — Project Scaffold (merged via PR #1)](#phase-0-project-scaffold-merged-via-pr-1)
  - [Phase 1 — Core Agent Loop (merged via PR #2)](#phase-1-core-agent-loop-merged-via-pr-2)
  - [Phase 2 — Multi-Provider Validation](#phase-2-multi-provider-validation)
  - [Phase 3 — Parallelism by Default](#phase-3-parallelism-by-default)
  - [Phase 4 — Hooks System](#phase-4-hooks-system)
  - [Phase 5 — Built-in Command Suite](#phase-5-built-in-command-suite)
  - [Phase 6 — Go Port](#phase-6-go-port)
- [What Is In Flight](#what-is-in-flight)
- [What Is Next](#what-is-next)
- [Key Decisions Made](#key-decisions-made)
- [Blockers / Watch Items](#blockers-watch-items)

## Current Status

**Active phase:** Phase 6 implemented — pending PR/merge

**Branch/PR state:**

| Branch | PR | State |
|---|---|---|
| `feat/phase-0-scaffold` | #1 | Merged to `main` |
| `feat/phase-1-agent-loop` | #2 | Merged to `main` |
| `feat/phase-2-multi-provider` | #3 | Merged to `main` |
| `feat/phase-3-parallelism` | #4 | Merged to `main` |
| `feat/phase-4-hooks` | #5 | Merged to `main` |
| `feat/phase-5-commands` | #6 | Merged to `main` |
| `feat/phase-6-go-port` | — | Local only, unpushed — ready to PR |

**Immediate next action:** `/pr` then `/merge`.

---

## What Has Landed (main)

### Phase 0 — Project Scaffold (merged via PR #1)
- `pyproject.toml` with `uv`, `typer`, `rich`, `litellm`
- `src/wi/` directory layout: `agent/`, `providers/`, `tools/`, `commands/`, `hooks/`
- `wi` CLI entry point: REPL on `wi`, one-shot on `wi "<prompt>"`
- `.gitignore` covering pycache, venv, dist, `.env`
- Docs moved to `docs/`

### Phase 1 — Core Agent Loop (merged via PR #2)
- `tools/executor.py`: tool executor + hard deny list (`git push --force`, `git reset --hard`, `git clean -f`, `rm -rf`)
- `providers/client.py`: LiteLLM wrapper, tool schemas for `read_file`, `write_file`, `bash`
- `agent/loop.py`: streaming turn loop — prompt → stream → parse tool calls → execute → next turn
- `cli.py`: loop wired into REPL and one-shot; `--model` flag for runtime override

### Phase 2 — Multi-Provider Validation
- `providers/client.py`: replaced `default_model()` with `resolve_model()` — composes `WI_PROVIDER` + `WI_MODEL` env vars; `_PROVIDER_DEFAULTS` table for Anthropic, OpenAI, Gemini, DeepSeek, Ollama; `_extra_kwargs()` injects `api_base` for Ollama automatically
- `agent/loop.py`: catches `litellm.AuthenticationError` and `BadRequestError` with actionable messages; `ValueError` from bad `WI_PROVIDER` surfaced cleanly in REPL
- `pyproject.toml`: added `pytest>=8.0` dev dependency group
- `tests/providers/test_client.py`: 8 unit tests covering all `resolve_model()` branches — no API calls required

### Phase 3 — Parallelism by Default
- `agent/loop.py`: `run()` and `_execute()` are now async; tool calls within a turn fan out via `asyncio.gather()` — independent tools run concurrently in a thread pool via `asyncio.to_thread()`
- `cli.py`: entry point updated to `asyncio.run()`; REPL inner loop is now `async def _async_repl()` so one event loop lives for the whole session
- `tests/agent/test_loop.py`: 5 async unit tests covering `_execute` — normal result, malformed args, denied command, concurrent ordering, unexpected exception

### Phase 4 — Hooks System
- `hooks/loader.py`: `HooksConfig` dataclass + `load(path)` — reads `~/.wi/hooks.toml`; supports `[[pre_tool_use]]`, `[[post_tool_use]]` arrays and `[session]` start/end as string or list; returns empty config if file absent
- `hooks/runner.py`: `run_pre_tool_use` (blocks on non-zero exit, exposes `WI_TOOL_NAME`/`WI_TOOL_ARGS` env), `run_post_tool_use` (silent on failure, adds `WI_TOOL_RESULT`), `run_lifecycle` (sequential commands)
- `agent/loop.py`: loads config once per `run()` call; wraps hook calls in `asyncio.to_thread()`; PreToolUse blocks before tool dispatch, PostToolUse fires after result
- `cli.py`: extracted `_async_oneshot()`; lifecycle hooks fire via `asyncio.to_thread()` at session start/end in both REPL and one-shot paths, with `finally` guaranteeing session.end
- `tests/hooks/test_loader.py` + `tests/hooks/test_runner.py`: 21 tests covering all loader branches and runner behaviour (blocking, matching, env vars, lifecycle ordering)

### Phase 5 — Built-in Command Suite
- `commands/_utils.py`: shared async subprocess helper (`run()`), `git_default_branch()`, `git_current_branch()`, `check_git()`
- `commands/registry.py`: command dispatcher — routes `/`-prefixed REPL input to handlers; returns `False` to exit
- `commands/gitignore.py`: detects stack from files present (Python, Node, Go, Rust, Java, Ruby, .NET), merges entries idempotently into `.gitignore`
- `commands/wrap.py`: writes `HANDOFF.md` with branch, recent commits, diff stat, working tree status, and template sections for decisions/next-steps/blockers
- `commands/unwrap.py`: reads and renders `HANDOFF.md` via Rich Markdown
- `commands/bye.py`: calls `/wrap` then returns `False` to exit REPL
- `commands/privacy.py`: scans git-tracked files for AI attribution patterns (Co-Authored-By, "Generated with", etc.), reports with file and line number
- `commands/pr.py`: pushes current branch, creates PR via `gh pr create --fill`
- `commands/merge.py`: lists open PRs, enables auto-merge with squash on each (skips drafts)
- `commands/commit.py`: groups unstaged files by concern key (directory heuristic), stages and commits each group with a generated conventional message
- `cli.py`: REPL now routes `/`-prefixed input to `cmd_registry.dispatch()`; `exit`/`quit` still work as bare aliases

---

### Phase 6 — Go Port
- `go.mod` at repo root: module `github.com/willardcsoriano/wi`, Go 1.24, deps cobra/bubbletea/lipgloss/bubbles/toml
- `internal/providers/types.go`: shared `Message`, `ToolCall`, `ToolSchema`, `TurnResult`, `Provider` interface
- `internal/providers/anthropic.go`: direct Anthropic SSE streaming client — translates internal message format (batching tool-result messages into Anthropic's grouped user turn), parses SSE events by type, accumulates text + tool calls
- `internal/providers/openai.go`: OpenAI-compatible SSE streaming client — covers OpenAI, Gemini, DeepSeek, Ollama via base URL injection
- `internal/providers/client.go`: `ResolveModel()` mirrors Python logic (WI_PROVIDER + WI_MODEL composition); `New()` factory dispatching to provider implementations
- `internal/tools/executor.go`: `read_file`, `write_file`, `bash` tools with the same deny list as the Python port
- `internal/hooks/loader.go`: TOML loader via BurntSushi/toml — same `~/.wi/hooks.toml` schema, string-or-slice lifecycle commands
- `internal/hooks/runner.go`: `RunPreToolUse`, `RunPostToolUse`, `RunLifecycle` — subprocess hook execution with `WI_TOOL_*` env vars
- `internal/agent/loop.go`: streaming turn loop with goroutine-based parallel tool execution (`sync.WaitGroup`)
- `internal/commands/*.go`: all 8 commands — `/gitignore`, `/wrap`, `/unwrap`, `/bye`, `/privacy`, `/pr`, `/merge`, `/commit`
- `internal/tui/repl.go`: bubbletea REPL — textinput prompt, tea.Cmd goroutines for agent/command dispatch, `progWriter` buffers to line boundaries for `prog.Println`
- `cmd/wi/main.go`: cobra CLI — `wi [prompt]` (one-shot vs REPL), `--model` flag, lifecycle hooks
- `.github/workflows/release.yml`: matrix build for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 — publishes to GitHub release on `v*` tag
- All targets cross-compile clean (~7–8 MB each, CGO_ENABLED=0)

---

## What Is In Flight

Phase 6 on `feat/phase-6-go-port` — built, compiled, cross-compiled clean. Awaiting PR and merge.

---

## What Is Next

Nothing — all planned phases implemented. Future work: live provider smoke tests, `/version` in REPL, shell completions.

---

## Key Decisions Made

| Decision | Rationale |
|---|---|
| Python prototype first, Go port later | Validate agent loop design cheaply before committing to lower-level implementation |
| LiteLLM for provider abstraction | Avoids writing per-provider HTTP clients; unified interface over 100+ providers including Ollama |
| No framework (no LangChain/LangGraph) | The agent loop is the product — frameworks hide exactly the decisions we need to make |
| Commands are direct handlers, not agentic tools | More predictable, easier to iterate; can make agentic later once loop is proven |
| Hard deny list in `executor.py`, not user-configurable | Safety floor must be unconditional; users can extend it but not shrink it |
| Squash merge strategy | One commit per concern on `main`; keeps history readable |
| Go port: direct HTTP + SSE, no LiteLLM | LiteLLM has no Go SDK; direct Anthropic and OpenAI-compatible clients cover all providers via base URL injection |
| bubbletea `sharedProg` pattern | `*tea.Program` must be set before `Run()` starts the event loop; storing it in a shared pointer avoids circular initialization |

---

## Blockers / Watch Items

- No live provider smoke tests in either Python or Go — end-to-end validation requires API keys and is manual for now
