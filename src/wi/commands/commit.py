from __future__ import annotations

from pathlib import Path

from rich.console import Console
from rich.markup import escape

from wi.commands._utils import run

_TYPE_MAP: dict[str, str] = {
    "tests":  "test",
    "test":   "test",
    "docs":   "docs",
    "doc":    "docs",
    ".github": "ci",
}

_PATH_TYPE_MAP: dict[str, str] = {
    "pyproject.toml": "build",
    "setup.py":       "build",
    "Makefile":       "build",
    "uv.lock":        "build",
    "Dockerfile":     "build",
}


def _concern_key(rel_path: str) -> str:
    parts = Path(rel_path).parts
    if not parts:
        return "root"
    top = parts[0]
    if top == "src" and len(parts) >= 3:
        return parts[2]
    return top


def _commit_type(files: list[str]) -> str:
    keys = {_concern_key(f) for f in files}
    for key in keys:
        if key in _TYPE_MAP:
            return _TYPE_MAP[key]
    for f in files:
        name = Path(f).name
        if name in _PATH_TYPE_MAP:
            return _PATH_TYPE_MAP[name]
    return "chore"


def _commit_scope(files: list[str]) -> str | None:
    keys = {_concern_key(f) for f in files}
    if len(keys) == 1:
        key = list(keys)[0]
        return None if key in {"root", "src"} else key
    return None


def _short_desc(files: list[str]) -> str:
    names = sorted({Path(f).name for f in files})
    if len(names) == 1:
        return f"update {names[0]}"
    if len(names) <= 3:
        return "update " + ", ".join(names)
    return f"update {len(files)} files"


async def handle(args: str, console: Console) -> bool:
    status = await run(["git", "status", "--porcelain"])
    if status.returncode != 0:
        console.print("[red]not a git repository[/red]")
        return True

    lines = [l for l in status.stdout.splitlines() if l.strip()]
    if not lines:
        console.print("[dim]nothing to commit — working tree clean[/dim]")
        return True

    # Collect unstaged and untracked files (not already staged)
    unstaged: list[str] = []
    for line in lines:
        xy, _, rel = line[:2], line[2], line[3:]
        state = xy.strip()
        if state and xy[1] != " ":
            unstaged.append(rel.strip())
        elif xy == "??":
            unstaged.append(rel.strip())

    if not unstaged:
        console.print("[dim]all changes already staged — run git commit manually[/dim]")
        return True

    # Group by concern key
    groups: dict[str, list[str]] = {}
    for f in unstaged:
        key = _concern_key(f)
        groups.setdefault(key, []).append(f)

    console.print(f"[dim]found {len(unstaged)} file(s) across {len(groups)} concern(s)[/dim]\n")

    committed = 0
    for key, files in groups.items():
        ctype = _commit_type(files)
        scope = _commit_scope(files)
        desc = _short_desc(files)
        scope_str = f"({scope})" if scope else ""
        message = f"{ctype}{scope_str}: {desc}"

        add = await run(["git", "add", "--"] + files)
        if add.returncode != 0:
            console.print(f"[red]git add failed for {key}:[/red] {escape(add.stderr.strip())}")
            continue

        commit = await run(["git", "commit", "-m", message])
        if commit.returncode != 0:
            console.print(f"[red]git commit failed for {key}:[/red] {escape(commit.stderr.strip())}")
            await run(["git", "reset", "HEAD"] + files)
            continue

        sha_line = commit.stdout.strip().splitlines()[0] if commit.stdout else ""
        console.print(f"[green]✓[/green] {escape(message)}  [dim]{escape(sha_line)}[/dim]")
        committed += 1

    if committed:
        console.print(f"\n[dim]{committed} commit(s) created[/dim]")
    return True
