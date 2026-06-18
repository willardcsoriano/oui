from __future__ import annotations

import os
from collections.abc import Iterator
from typing import Any

import litellm

litellm.suppress_debug_info = True

TOOL_SCHEMAS: list[dict[str, Any]] = [
    {
        "type": "function",
        "function": {
            "name": "read_file",
            "description": "Read the contents of a file at the given path.",
            "parameters": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Absolute or relative file path"},
                },
                "required": ["path"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "write_file",
            "description": "Write content to a file, creating parent directories as needed.",
            "parameters": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Absolute or relative file path"},
                    "content": {"type": "string", "description": "Content to write"},
                },
                "required": ["path", "content"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "bash",
            "description": "Run a shell command and return its output.",
            "parameters": {
                "type": "object",
                "properties": {
                    "command": {"type": "string", "description": "Shell command to execute"},
                },
                "required": ["command"],
            },
        },
    },
]

_PROVIDER_DEFAULTS: dict[str, str] = {
    "anthropic": "anthropic/claude-sonnet-4-6",
    "openai": "openai/gpt-4o",
    "gemini": "gemini/gemini-2.0-flash",
    "deepseek": "deepseek/deepseek-chat",
    "ollama": "ollama/llama3.2",
}

_FALLBACK_MODEL = "anthropic/claude-sonnet-4-6"


def resolve_model(model: str | None = None) -> str:
    """Resolve the active model from explicit arg, env vars, or fallback default.

    Resolution order:
      1. Explicit ``model`` argument (e.g. from --model CLI flag)
      2. WI_MODEL if it is already a qualified string (contains '/')
      3. WI_PROVIDER/WI_MODEL if both are set and WI_MODEL is a bare name
      4. WI_PROVIDER default from _PROVIDER_DEFAULTS if only WI_PROVIDER is set
      5. _FALLBACK_MODEL
    """
    if model:
        return model

    wi_model = os.environ.get("WI_MODEL", "")
    wi_provider = os.environ.get("WI_PROVIDER", "")

    if wi_model and ("/" in wi_model or not wi_provider):
        return wi_model

    if wi_model and wi_provider:
        return f"{wi_provider}/{wi_model}"

    if wi_provider:
        if wi_provider not in _PROVIDER_DEFAULTS:
            raise ValueError(
                f"WI_PROVIDER={wi_provider!r} has no default model; "
                f"also set WI_MODEL (e.g. WI_MODEL=my-model-name)"
            )
        return _PROVIDER_DEFAULTS[wi_provider]

    return _FALLBACK_MODEL


def _extra_kwargs(model: str) -> dict[str, Any]:
    if model.startswith("ollama/"):
        base = (
            os.environ.get("WI_OLLAMA_BASE_URL")
            or os.environ.get("OLLAMA_API_BASE")
            or "http://localhost:11434"
        )
        return {"api_base": base}
    return {}


def stream_completion(
    messages: list[dict[str, Any]],
    model: str | None = None,
) -> Iterator[Any]:
    resolved = resolve_model(model)
    return litellm.completion(
        model=resolved,
        messages=messages,
        tools=TOOL_SCHEMAS,
        tool_choice="auto",
        stream=True,
        **_extra_kwargs(resolved),
    )
