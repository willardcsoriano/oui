## Overview

`oui` is a provider-agnostic agentic coding shell for the terminal — the friction removed and a battle-tested workflow built in by default. You bring your own API keys (any major provider or local model) and `oui` treats them as interchangeable backends to the same agent loop. It is opinionated in three ways: it never interrupts you for permission, it ships with a full command suite out of the box, and it fans out independent work concurrently by default. Everything else — provider, model, project stack — is your call.

## Table of Contents

- [Overview](#overview)
- [The Problem](#the-problem)
- [Design Philosophy](#design-philosophy)
- [What `oui` Is Not](#what-oui-is-not)
- [Non-Goals](#non-goals)

## The Problem

AI coding agents have gotten good fast. But most of them ship with friction that shouldn't be there:

- **Constant permission prompts.** Every file write, every shell command triggers an approval gate. In practice the answer is always yes — so the gate adds latency without adding safety for the developer who knows what they're doing.
- **No built-in workflow.** There is no standard way to do a clean concern-scoped commit, open a CI-gated PR per branch, manage session handoffs, or scrub agent traces from a repo. You bolt these on yourself or you go without.
- **Provider lock-in.** Most tools are built around a single model. Switching providers means switching tools, losing config, starting over.
- **Sequential by default.** Independent subtasks don't fan out automatically. You wait for step one to finish before step two starts, even when they have no dependency on each other.

## Design Philosophy

These principles govern every decision in `oui`:

**The user is the architect, the agent is the implementer.** `oui` leads with design and trade-offs before writing code. When a direction wouldn't hold up in production, it pushes back and explains why — it doesn't just comply.

**Auto-mode is the right default.** Interruptions are friction. The developer steers through the prompt, not through approval dialogs. The only exceptions are a short list of genuinely destructive operations that should never run unattended.

**Parallelism is the right default.** Independent subtasks run concurrently. `oui` decomposes work into a dependency graph and fans out the independent branches as concurrent agents, serializing only what genuinely depends on a prior result.

**Speed requires verified facts.** AI training data lags. For anything time-sensitive — library versions, CLI flags, API surfaces, current best practices — `oui` fetches and verifies before acting. It never silently falls back to stale training data.

**Keys and secrets stay local.** API keys live in `~/.oui/config.toml` (mode `0600`) or in environment variables — never in logs, never echoed, never transmitted except to the configured model provider.

**Hooks are first-class.** Behavior should be extensible without forking the tool. `oui` supports a hooks system (PostToolUse, PreToolUse, and lifecycle events) so users can wire in their own scripts at any point in the agent loop.

For the concrete feature set, command reference, configuration interface, and provider list, see [`docs/FEATURES.md`](FEATURES.md).

## What `oui` Is Not

- Not a cloud product — runs entirely locally, your keys stay local
- Not a wrapper around Claude Code or Gemini CLI — its own implementation
- Not opinionated about your stack — works with any language, any project
- Not a replacement for an IDE — a coding agent that lives in the terminal

## Non-Goals

- GUI or web interface
- Managing, rotating, or proxying API keys
- Storing or syncing user code remotely
- Telemetry or usage analytics
