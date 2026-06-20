from __future__ import annotations

import re
import shlex
import subprocess
from pathlib import Path
from typing import Any

# Operations that must never run unattended, regardless of user config.
_DENY_PATTERNS: list[re.Pattern[str]] = [
    re.compile(r"\bgit\s+push\s+(-f|--force)\b"),
    re.compile(r"\bgit\s+reset\s+--hard\b"),
    re.compile(r"\bgit\s+clean\s+.*-[a-zA-Z]*f"),
    re.compile(r"\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f\b"),
    re.compile(r"\brm\s+.*-[a-zA-Z]*f[a-zA-Z]*r\b"),
]


class DeniedCommandError(Exception):
    pass


def _check_deny_list(command: str) -> None:
    for pattern in _DENY_PATTERNS:
        if pattern.search(command):
            raise DeniedCommandError(f"blocked by deny list: {command!r}")


def execute_tool(name: str, args: dict[str, Any]) -> str:
    match name:
        case "read_file":
            return _read_file(args["path"])
        case "write_file":
            return _write_file(args["path"], args["content"])
        case "bash":
            return _bash(args["command"])
        case _:
            return f"unknown tool: {name}"


def _read_file(path: str) -> str:
    try:
        return Path(path).read_text()
    except FileNotFoundError:
        return f"error: file not found: {path}"
    except PermissionError:
        return f"error: permission denied: {path}"


def _write_file(path: str, content: str) -> str:
    try:
        p = Path(path)
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)
        return f"wrote {len(content)} bytes to {path}"
    except PermissionError:
        return f"error: permission denied: {path}"


def _bash(command: str) -> str:
    _check_deny_list(command)
    result = subprocess.run(
        command,
        shell=True,
        capture_output=True,
        text=True,
        timeout=60,
    )
    output = result.stdout
    if result.stderr:
        output += result.stderr
    return output.rstrip() or "(no output)"
