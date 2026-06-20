from __future__ import annotations

import asyncio
import subprocess
from pathlib import Path
from typing import Any


async def run(
    cmd: list[str] | str,
    *,
    capture: bool = True,
    cwd: Path | None = None,
) -> subprocess.CompletedProcess:
    return await asyncio.to_thread(
        subprocess.run,
        cmd,
        capture_output=capture,
        text=True,
        cwd=cwd,
    )


async def git_default_branch() -> str:
    result = await run(["git", "remote", "show", "origin"])
    for line in result.stdout.splitlines():
        if "HEAD branch" in line:
            return line.split(":")[-1].strip()
    return "main"


async def git_current_branch() -> str:
    result = await run(["git", "branch", "--show-current"])
    return result.stdout.strip()


def check_git(cwd: Path = Path.cwd()) -> bool:
    return (cwd / ".git").exists() or any(
        (p / ".git").exists() for p in cwd.parents
    )
