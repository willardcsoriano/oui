## Overview

`wi` is a provider-agnostic agentic coding shell for the terminal. The core premise is simple: the best parts of Claude Code and Gemini CLI, combined into one tool, with the friction removed and the workflow built in. You bring your own API keys — Claude, Gemini, DeepSeek, or anything else — and `wi` treats them as interchangeable backends to the same agent loop. It is opinionated in exactly two ways: it never interrupts you for permission, and it ships with a battle-tested command suite baked in by default. Everything else is as open and provider-neutral as possible.

## Table of Contents

- [Overview](#overview)
- [The Problem](#the-problem)
- [The Vision](#the-vision)
  - [1. Auto-mode by default](#1-auto-mode-by-default)
  - [2. Built-in command suite](#2-built-in-command-suite)
  - [3. Any provider, any key](#3-any-provider-any-key)
- [What `wi` Is Not](#what-wi-is-not)
- [Non-Goals](#non-goals)

## The Problem

Claude Code and Gemini CLI are both excellent. But using either one means accepting their friction:

- **Constant permission prompts.** Every file write, every shell command, every tool call triggers an approval gate. In practice, the answer is always yes — so the gate adds latency without adding safety for the developer who knows what they're doing.
- **No built-in workflow commands.** There is no standard way to do a clean concern-scoped commit, open a CI-gated PR per branch, manage session handoffs, or scrub agent traces from a repo. You bolt these on yourself or you don't have them.
- **Provider lock-in.** Claude Code only talks to Claude. Gemini CLI only talks to Gemini. Switching models means switching tools, losing config, starting over.

## The Vision

`wi` fixes all three:

### 1. Auto-mode by default

The agent runs uninterrupted. No permission prompts, no approval gates, no mid-task pauses. This is the default, not an opt-in flag. The developer is in control by steering the prompt, not by clicking through dialogs.

### 2. Built-in command suite

The following commands ship with `wi` out of the box, ported from a battle-tested Claude Code configuration:

| Command | What it does |
|---|---|
| `/commit` | Segregates working changes into the maximum number of independent concern-scoped branches, verifies each in isolation, and commits cleanly |
| `/pr` | Pushes each branch from `/commit` and opens one CI-gated PR/MR per branch |
| `/merge` | Lands open PRs via forge auto-merge (merge-when-green by default) |
| `/wrap` | End-of-session handoff — captures state, decisions, and next steps into memory and `HANDOFF.md` |
| `/unwrap` | Resume a session cold — reads `HANDOFF.md` and memory to brief you on where you left off |
| `/bye` | Closes a session permanently — marks handoff complete, reconciles memory, commits the closure |
| `/privacy` | Scans the repo for AI coding-agent traces and removes them from tracked files, history, and untracked artifacts |
| `/gitignore` | Generates a stack-aware `.gitignore` from the detected tech stack, merged idempotently |

### 3. Any provider, any key

`wi` is a harness, not a client. The model backend is a configuration line. Point it at Claude, Gemini, DeepSeek, a local Ollama instance, or any OpenAI-compatible endpoint — the agent loop, tools, and commands stay identical regardless.

## What `wi` Is Not

- Not a cloud product. It runs locally, your keys stay local.
- Not a wrapper around Claude Code or Gemini CLI. It is its own implementation.
- Not opinionated about your stack. It works with any language, any project.

## Non-Goals

- GUI or web interface — terminal only.
- Managing or proxying API keys — you supply them, `wi` uses them.
- Replacing a full IDE — `wi` is a coding agent, not an editor.
