## Overview

`wi` is a provider-agnostic agentic coding shell for the terminal — the best parts of Claude Code and Gemini CLI combined, with the friction removed and a battle-tested workflow built in by default. You bring your own API keys (Claude, Gemini, DeepSeek, local Ollama, or any OpenAI-compatible endpoint) and `wi` treats them as interchangeable backends to the same agent loop. It is opinionated in three ways: it never interrupts you for permission, it ships with a full command suite out of the box, and it fans out independent work concurrently by default. Everything else — provider, model, project stack — is your call.

## Table of Contents

- [Overview](#overview)
- [The Problem](#the-problem)
- [Design Philosophy](#design-philosophy)
- [Feature Set](#feature-set)
  - [1. Auto-mode by default, with a hard deny list](#1-auto-mode-by-default-with-a-hard-deny-list)
  - [2. Built-in command suite](#2-built-in-command-suite)
  - [3. Any provider, any key](#3-any-provider-any-key)
  - [4. Hooks system](#4-hooks-system)
  - [5. Parallelism by default](#5-parallelism-by-default)
  - [6. Secrets and privacy by design](#6-secrets-and-privacy-by-design)
- [What `wi` Is Not](#what-wi-is-not)
- [Non-Goals](#non-goals)

## The Problem

Claude Code and Gemini CLI are both excellent tools. But using either means accepting friction that shouldn't exist:

- **Constant permission prompts.** Every file write, every shell command triggers an approval gate. In practice the answer is always yes — so the gate adds latency without adding safety for the developer who knows what they're doing.
- **No built-in workflow.** There is no standard way to do a clean concern-scoped commit, open a CI-gated PR per branch, manage session handoffs, or scrub agent traces from a repo. You bolt these on yourself or you go without.
- **Provider lock-in.** Claude Code only talks to Claude. Gemini CLI only talks to Gemini. Switching models means switching tools, losing config, starting over.
- **Sequential by default.** Neither tool fans out independent subtasks automatically. You wait for step one to finish before step two starts, even when they have no dependency on each other.

## Design Philosophy

These principles govern every decision in `wi`:

**The user is the architect, the agent is the implementer.** `wi` leads with design and trade-offs before writing code. When a direction wouldn't hold up in production, it pushes back and explains why — it doesn't just comply.

**Auto-mode is the right default.** Interruptions are friction. The developer steers through the prompt, not through approval dialogs. The only exceptions are a short list of genuinely destructive operations that should never run unattended.

**Parallelism is the right default.** Independent subtasks run concurrently. `wi` decomposes work into a dependency graph and fans out the independent branches as concurrent agents, serializing only what genuinely depends on a prior result.

**Speed requires verified facts.** AI training data lags. For anything time-sensitive — library versions, CLI flags, API surfaces, current best practices — `wi` fetches and verifies before acting. It never silently falls back to stale training data.

**Keys and secrets are session-scoped.** API keys never touch disk, never appear in logs, and die with the session. Code and context leave the machine only to reach the configured model provider — nowhere else.

**Hooks are first-class.** Behavior should be extensible without forking the tool. `wi` supports a hooks system (PostToolUse, PreToolUse, and lifecycle events) so users can wire in their own scripts at any point in the agent loop.

## Feature Set

### 1. Auto-mode by default, with a hard deny list

The agent runs uninterrupted — no permission prompts, no approval gates, no mid-task pauses. This is the default, not a flag.

Even in auto-mode, a short hard deny list applies unconditionally:

| Blocked operation | Reason |
|---|---|
| `git push --force` / `git push -f` | Rewrites shared history |
| `git reset --hard` | Destroys uncommitted work |
| `git clean -f` | Permanently deletes untracked files |
| `rm -rf` / `rm -fr` | Irreversible mass deletion |

Everything else runs without asking. The deny list is the floor, not a ceiling — users can extend it.

### 2. Built-in command suite

The following commands ship with `wi` out of the box:

| Command | What it does |
|---|---|
| `/commit` | Segregates working changes into the maximum number of independent concern-scoped branches, verifies each in isolation, commits cleanly — splits at file or hunk granularity |
| `/pr` | Pushes each branch from `/commit` and opens one CI-gated PR/MR per branch (forge-aware: GitHub, GitLab, Gitea) |
| `/merge` | Lands open PRs via forge auto-merge — merge-when-green by default; never force-merges or bypasses required checks |
| `/wrap` | End-of-session handoff — captures state, decisions, and next steps into memory and `HANDOFF.md` for cold resumption |
| `/unwrap` | Resumes a session cold — reads `HANDOFF.md` and memory to brief you on where you left off |
| `/bye` | Closes a session permanently — marks handoff complete, reconciles memory, commits the closure entry |
| `/privacy` | Scans the repo for AI coding-agent traces and removes them from tracked files, history, and untracked artifacts |
| `/gitignore` | Generates a stack-aware `.gitignore` from the detected tech stack, merged idempotently into the committed file |

### 3. Any provider, any key

`wi` is a harness, not a client. The model backend is a config line. Point it at:

- Anthropic Claude (any model)
- Google Gemini
- DeepSeek
- OpenAI / GPT
- Local models via Ollama or LM Studio
- Any OpenAI-compatible endpoint

The agent loop, tools, and commands are identical regardless of which provider is active. Switch providers mid-project without losing anything.

### 4. Hooks system

`wi` exposes a hooks interface modeled on what makes Claude Code's `settings.json` hooks useful but generalized:

- **PostToolUse** — runs after any tool call (e.g., auto-inject a TOC after writing a markdown file)
- **PreToolUse** — runs before any tool call (e.g., validate, log, or block)
- **Lifecycle hooks** — session start, session end, command entry/exit

Hooks are shell commands. No plugin API to learn — just wire a script.

### 5. Parallelism by default

When `wi` receives a multi-part task, it decomposes it into a dependency graph and fans out independent branches as concurrent sub-agents. Only genuinely dependent steps are serialized. This is not an opt-in mode — it is how `wi` works by default.

### 6. Secrets and privacy by design

- API keys are passed via environment variables, never written to config files or disk
- Keys are never echoed in logs, output, or memory
- User code and context are sent only to the configured model provider — no telemetry, no third-party calls
- PII is treated as ephemeral: used for the immediate task, never persisted

## What `wi` Is Not

- Not a cloud product — runs entirely locally, your keys stay local
- Not a wrapper around Claude Code or Gemini CLI — its own implementation
- Not opinionated about your stack — works with any language, any project
- Not a replacement for an IDE — a coding agent that lives in the terminal

## Non-Goals

- GUI or web interface
- Managing, rotating, or proxying API keys
- Storing or syncing user code remotely
- Telemetry or usage analytics
