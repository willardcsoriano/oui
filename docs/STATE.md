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
- [What Is In Flight](#what-is-in-flight)
- [What Is Next](#what-is-next)
  - [Phase 3 — Parallelism by Default](#phase-3-parallelism-by-default)
  - [Phase 4 — Hooks System](#phase-4-hooks-system)
  - [Phase 5 — Built-in Command Suite](#phase-5-built-in-command-suite)
  - [Phase 6 — Go Port](#phase-6-go-port)
- [Key Decisions Made](#key-decisions-made)
- [Blockers / Watch Items](#blockers-watch-items)

## Current Status

**Active phase:** Phase 2 complete, Phase 3 not started

**Branch/PR state:**

| Branch | PR | State |
|---|---|---|
| `feat/phase-0-scaffold` | #1 | Merged to `main` |
| `feat/phase-1-agent-loop` | #2 | Merged to `main` |

**Immediate next action:** Start Phase 3 (asyncio fan-out for parallel tool calls).

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

---

## What Is In Flight

Nothing — all completed phases are on `main`.

---

## What Is Next

### Phase 3 — Parallelism by Default
asyncio fan-out for independent tool calls in a single turn. Sub-agent spawning for independent subtasks.

### Phase 4 — Hooks System
`PreToolUse`, `PostToolUse`, and lifecycle hooks (`session.start`, `session.end`) via `~/.wi/hooks.toml`.

### Phase 5 — Built-in Command Suite
In order: `/gitignore`, `/wrap`, `/unwrap`, `/bye`, `/privacy`, `/pr`, `/merge`, `/commit`.

### Phase 6 — Go Port
Translate validated Python design to Go for single-binary distribution.

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

---

## Blockers / Watch Items

- No live provider smoke tests — `resolve_model()` is unit-tested but end-to-end validation against OpenAI/Gemini/DeepSeek/Ollama requires API keys and is manual for now
