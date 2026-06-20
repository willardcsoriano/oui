## Overview

This document is the authoritative reference for what `oui` actually does — concrete feature behavior, command syntax, configuration interface, and behavioral guarantees. Every entry here is shipped and working in the Go implementation. Design rationale and philosophy live in `docs/VISION.md`; this file answers "what does it do and how do I use it?" Features are organized by area: the agent loop and auto-mode behavior, the built-in command suite, provider and model configuration, the hooks system, and the parallelism model. Configuration is via `~/.oui/config.toml` (auto-created on first run) with environment variable fallback for all API keys. The roadmap for planned work lives in `docs/ROADMAP.md`; the current implementation state is in `docs/STATE.md`.

---

## Table of Contents

- [Overview](#overview)
- [Auto-Mode and the Deny List](#auto-mode-and-the-deny-list)
- [Built-in Command Suite](#built-in-command-suite)
- [Provider and Model Configuration](#provider-and-model-configuration)
  - [Config file](#config-file)
  - [Environment variables](#environment-variables)
  - [Runtime switching](#runtime-switching)
  - [Supported providers](#supported-providers)
- [Hooks System](#hooks-system)
- [Parallelism](#parallelism)
- [Secrets and Privacy](#secrets-and-privacy)

## Auto-Mode and the Deny List

The agent runs uninterrupted — no permission prompts, no approval gates, no mid-task pauses. This is the default, not a flag.

A hard deny list applies unconditionally regardless of mode:

| Blocked operation | Reason |
|---|---|
| `git push --force` / `git push -f` | Rewrites shared history |
| `git reset --hard` | Destroys uncommitted work |
| `git clean -f` | Permanently deletes untracked files |
| `rm -rf` / `rm -fr` | Irreversible mass deletion |

Everything else runs without asking. The deny list is a floor, not a ceiling — users can extend it via hooks.

---

## Built-in Command Suite

All commands are typed at the REPL prompt as `/command`.

| Command | What it does |
|---|---|
| `/commit` | Groups unstaged changes by concern (directory heuristic), stages and commits each group with a generated Conventional Commits message |
| `/pr` | Pushes the current branch and opens a PR via `gh pr create --fill` |
| `/merge` | Lists open PRs and enables auto-merge (squash) on each; skips drafts |
| `/wrap` | Writes `HANDOFF.md` with branch, recent commits, diff stat, working tree status, and template sections for decisions/next-steps/blockers |
| `/unwrap` | Reads and renders `HANDOFF.md` via Rich Markdown for cold session resumption |
| `/bye` | Runs `/wrap` then exits the REPL |
| `/privacy` | Scans git-tracked files for AI attribution patterns (`Co-Authored-By`, "Generated with", etc.) and reports file + line number |
| `/gitignore` | Detects the project stack (Python, Node, Go, Rust, Java, Ruby, .NET) from files present and merges appropriate entries into `.gitignore` idempotently |
| `/provider [name]` | Prints the active provider when called with no arg; switches to `name` and resets the model to its configured default when an arg is provided |
| `/model [name]` | Prints the active model when called with no arg; switches to `name` within the current provider when an arg is provided |

---

## Provider and Model Configuration

### Config file

`~/.oui/config.toml` is created automatically on first run with defaults for all supported providers. It is written at mode `0600`.

```toml
active_provider = "anthropic"
active_model = ""          # empty = use provider's default_model

[providers.anthropic]
api_key = "sk-ant-..."
default_model = "claude-sonnet-4-6"

[providers.openai]
api_key = "sk-..."
default_model = "gpt-4o"

[providers.ollama]
base_url = "http://localhost:11434/v1"
default_model = "llama3.2"
```

### Environment variables

If `api_key` is not set in the config file, `oui` falls back to the conventional env var for each provider:

| Provider | Env var |
|---|---|
| anthropic | `ANTHROPIC_API_KEY` |
| openai | `OPENAI_API_KEY` |
| gemini | `GEMINI_API_KEY` |
| deepseek | `DEEPSEEK_API_KEY` |
| groq | `GROQ_API_KEY` |
| mistral | `MISTRAL_API_KEY` |
| cohere | `COHERE_API_KEY` |
| xai | `XAI_API_KEY` |
| together | `TOGETHER_API_KEY` |
| fireworks | `FIREWORKS_API_KEY` |
| perplexity | `PERPLEXITY_API_KEY` |

`OUI_PROVIDER` and `OUI_MODEL` override `active_provider` and `active_model` at startup.

### Runtime switching

Use `/provider` and `/model` in the REPL to hot-swap without restarting. The new provider/model takes effect immediately for the next turn; the change is not persisted back to `config.toml` automatically.

### Supported providers

| Provider | Notes |
|---|---|
| `anthropic` | Direct SSE client — supports all Claude models |
| `openai` | OpenAI-compatible client |
| `gemini` | OpenAI-compatible endpoint |
| `deepseek` | OpenAI-compatible endpoint |
| `groq` | OpenAI-compatible; base URL pre-configured |
| `mistral` | OpenAI-compatible; base URL pre-configured |
| `cohere` | OpenAI-compatible compatibility endpoint |
| `xai` | OpenAI-compatible; base URL pre-configured |
| `together` | OpenAI-compatible; base URL pre-configured |
| `fireworks` | OpenAI-compatible; base URL pre-configured |
| `perplexity` | OpenAI-compatible; base URL pre-configured |
| `ollama` | Local; no API key required; base URL defaults to `http://localhost:11434/v1` |

Any additional OpenAI-compatible endpoint can be used by adding a provider entry with a custom `base_url`.

---

## Hooks System

Hooks are shell commands configured in `~/.oui/hooks.toml`. No plugin API — just wire a script.

```toml
[[pre_tool_use]]
command = "echo 'about to run $OUI_TOOL_NAME'"

[[post_tool_use]]
command = "my-logger $OUI_TOOL_NAME $OUI_TOOL_RESULT"

[session]
start = "echo session started"
end = ["echo session ended", "cleanup.sh"]
```

| Hook | When it runs | On non-zero exit |
|---|---|---|
| `pre_tool_use` | Before any tool call | Blocks the tool call |
| `post_tool_use` | After any tool call | Silent — logged, not fatal |
| `session.start` | At REPL or one-shot startup | — |
| `session.end` | At exit (guaranteed via `finally`) | — |

Available env vars in hook commands: `OUI_TOOL_NAME`, `OUI_TOOL_ARGS` (pre), `OUI_TOOL_RESULT` (post).

---

## Parallelism

When the model returns multiple independent tool calls in a single turn, `oui` executes them concurrently via goroutines (`sync.WaitGroup`). Tool calls within a turn that have no dependency on each other fan out automatically — no configuration required.

---

## Secrets and Privacy

- API keys in `~/.oui/config.toml` are stored at mode `0600` (owner read/write only)
- Keys resolved from env vars are never written to disk
- Keys are never echoed in logs, output, memory, or REPL display
- User code and context are sent only to the configured model provider — no telemetry, no third-party calls
- `/privacy` scans tracked files for AI agent attribution patterns and reports them for removal
