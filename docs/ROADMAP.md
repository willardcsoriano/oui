## Overview

This roadmap covers the phased implementation of `oui` — starting with a Python prototype to validate the core design decisions (agent loop, provider abstraction, parallelism model, hooks interface) before porting to Go for production distribution. The prototype is not throwaway: it is the design lab where the hardest architectural decisions get made cheaply. Every phase has a concrete test condition so progress is unambiguous. Phases are ordered by dependency — each one unblocks the next — with the riskiest unknowns front-loaded. The Go port begins only after the Python design is stable and validated against real API keys across multiple providers.

---

## Table of Contents

- [Overview](#overview)
- [Phase 0: Project Scaffold](#phase-0-project-scaffold)
- [Phase 1: Core Agent Loop](#phase-1-core-agent-loop)
- [Phase 2: Multi-Provider Validation](#phase-2-multi-provider-validation)
- [Phase 3: Parallelism by Default](#phase-3-parallelism-by-default)
- [Phase 4: Hooks System](#phase-4-hooks-system)
- [Phase 5: Built-in Command Suite](#phase-5-built-in-command-suite)
- [Phase 6: Go Port](#phase-6-go-port)

## Phase 0: Project Scaffold

**Goal:** Working project skeleton — installable, runnable, nothing more.

- `pyproject.toml` with `uv`, `typer`, `rich`, `litellm`
- `src/oui/` directory layout matching the target architecture
- `oui` CLI entry point: `oui` drops into REPL, `oui "<prompt>"` runs one-shot
- LiteLLM wired up but not yet calling anything

**Done when:** `uv run oui` launches without error and prints a version string.

---

## Phase 1: Core Agent Loop

**Goal:** A working agent REPL against a single provider (Anthropic via LiteLLM), streaming, with tool execution.

- Streaming response loop: prompt → stream tokens → parse tool calls → execute → next turn
- Three built-in tools: `read_file`, `write_file`, `bash`
- Deny list enforced in `executor.py` from day one — never retrofitted
- `rich` output: streamed text, tool call display, result display

**Done when:** You can prompt `oui` to read a file, modify it, and write it back in one conversation, streaming live.

---

## Phase 2: Multi-Provider Validation

**Goal:** Confirm the provider abstraction holds — swap providers without touching the agent loop.

- Point LiteLLM at OpenAI, Gemini, DeepSeek, and a local Ollama instance
- Validate that tool call schema differences are absorbed by LiteLLM, not leaked into the loop
- Provider set via env var (`OUI_PROVIDER`, `OUI_MODEL`) — no config file

**Done when:** The same agent loop runs identically against Anthropic and at least one other provider by changing one env var.

---

## Phase 3: Parallelism by Default

**Goal:** Independent tool calls fan out as concurrent asyncio tasks; independent sub-tasks spawn concurrent sub-agents.

- When the model returns multiple independent tool calls in one turn, execute them concurrently
- Sub-agent spawning: `oui` can delegate a subtask to a child instance and await the result
- Dependency graph: if subtask B depends on subtask A's output, serialize; otherwise fan out

**Done when:** A multi-file read task executes file reads concurrently, with measurable wall-clock improvement over sequential.

---

## Phase 4: Hooks System

**Goal:** Users can wire shell scripts into the agent lifecycle without modifying `oui`.

- `PreToolUse`: runs before any tool call; non-zero exit blocks execution
- `PostToolUse`: runs after any tool call
- Lifecycle hooks: `session.start`, `session.end`
- Hook config: `~/.oui/hooks.toml` — shell commands keyed by event name

**Done when:** A `PostToolUse` hook that appends a log line after every `bash` call works end-to-end.

---

## Phase 5: Built-in Command Suite

Implement in this order (simplest to hardest):

| Command | What it does | Complexity |
|---|---|---|
| `/gitignore` | Stack-aware `.gitignore` generation | Low |
| `/wrap` | End-of-session handoff to `HANDOFF.md` + memory | Medium |
| `/unwrap` | Cold-resume from `HANDOFF.md` | Low |
| `/bye` | Close session, reconcile memory | Low |
| `/privacy` | Scan and remove AI agent traces | Medium |
| `/pr` | Push branch, open CI-gated PR per forge | High |
| `/merge` | Land open PRs via forge auto-merge | Medium |
| `/commit` | Hunk-level concern-scoped commit splitting | Highest |

**Done when:** `/wrap` → cold exit → `/unwrap` fully restores session context.

---

## Phase 6: Go Port

**Goal:** Translate the validated Python design to Go for single-binary distribution.

- All architectural decisions are frozen — translation, not redesign
- Provider HTTP layer written directly (no LiteLLM equivalent in Go; it's a one-time cost)
- `bubbletea` + `lipgloss` for TUI, `cobra` for CLI
- Cross-compile for Linux, macOS, and Windows in CI

**Done when:** `oui` ships as a single binary with no runtime dependency and behavior identical to the Python prototype.
