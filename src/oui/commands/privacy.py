from __future__ import annotations

import re
from pathlib import Path

from rich.console import Console
from rich.markup import escape

from oui.commands._utils import run

_PATTERNS: list[re.Pattern] = [
    re.compile(r"Co-Authored-By:.*[Cc]laude", re.IGNORECASE),
    re.compile(r"Generated with.*[Cc]laude", re.IGNORECASE),
    re.compile(r"made with.*[Cc]laude [Cc]ode", re.IGNORECASE),
    re.compile(r"🤖\s*(Generated|made) with", re.IGNORECASE),
    re.compile(r"<noreply@anthropic\.com>"),
]

_BINARY_EXTENSIONS = {
    ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2",
    ".ttf", ".eot", ".otf", ".pdf", ".zip", ".tar", ".gz", ".lock",
}


async def handle(args: str, console: Console) -> bool:
    result = await run(["git", "ls-files"])
    if result.returncode != 0:
        console.print("[red]not a git repository[/red]")
        return True

    tracked = [p for p in result.stdout.splitlines() if p]
    findings: list[tuple[str, int, str]] = []

    for rel_path in tracked:
        path = Path(rel_path)
        if path.suffix.lower() in _BINARY_EXTENSIONS:
            continue
        try:
            text = path.read_text(errors="replace")
        except OSError:
            continue
        for pattern in _PATTERNS:
            for match in pattern.finditer(text):
                line_no = text[: match.start()].count("\n") + 1
                findings.append((rel_path, line_no, match.group()))

    if not findings:
        console.print("[green]no AI attribution traces found[/green]")
        return True

    console.print(f"[yellow]found {len(findings)} AI trace(s):[/yellow]\n")
    for path, line, snippet in findings:
        console.print(f"  [bold]{escape(path)}[/bold]:{line}  [dim]{escape(snippet)}[/dim]")

    console.print(
        f"\n[dim]Review and remove manually, or amend the relevant commits. "
        f"Re-run /privacy to verify.[/dim]"
    )
    return True
