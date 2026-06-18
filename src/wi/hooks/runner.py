from __future__ import annotations

import fnmatch
import json
import os
import subprocess
from typing import Any

from wi.hooks.loader import HooksConfig


def run_pre_tool_use(
    config: HooksConfig,
    tool_name: str,
    args: dict[str, Any],
) -> str | None:
    """Run matching PreToolUse hooks.

    Returns an error string if any hook exits non-zero (blocking the tool call),
    or None if all hooks pass.
    """
    env = {**os.environ, "WI_TOOL_NAME": tool_name, "WI_TOOL_ARGS": json.dumps(args)}
    for hook in config.pre_tool_use:
        if not fnmatch.fnmatch(tool_name, hook.match):
            continue
        result = subprocess.run(
            hook.command, shell=True, capture_output=True, text=True, env=env
        )
        if result.returncode != 0:
            msg = (result.stdout + result.stderr).strip()
            return msg or "blocked by PreToolUse hook"
    return None


def run_post_tool_use(
    config: HooksConfig,
    tool_name: str,
    args: dict[str, Any],
    result: str,
) -> None:
    """Run matching PostToolUse hooks. Hook failures are silent."""
    env = {
        **os.environ,
        "WI_TOOL_NAME": tool_name,
        "WI_TOOL_ARGS": json.dumps(args),
        "WI_TOOL_RESULT": result,
    }
    for hook in config.post_tool_use:
        if not fnmatch.fnmatch(tool_name, hook.match):
            continue
        subprocess.run(hook.command, shell=True, capture_output=True, text=True, env=env)


def run_lifecycle(commands: list[str]) -> None:
    """Run lifecycle hook commands sequentially. Failures are silent."""
    for cmd in commands:
        subprocess.run(cmd, shell=True)
