from __future__ import annotations

import os
from collections.abc import Iterator
from typing import Any

import litellm
from litellm import ModelResponse

# Suppress litellm's noisy startup output.
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


def default_model() -> str:
    return os.environ.get("WI_MODEL", "anthropic/claude-sonnet-4-6")


def stream_completion(
    messages: list[dict[str, Any]],
    model: str | None = None,
) -> Iterator[Any]:
    return litellm.completion(
        model=model or default_model(),
        messages=messages,
        tools=TOOL_SCHEMAS,
        tool_choice="auto",
        stream=True,
    )
