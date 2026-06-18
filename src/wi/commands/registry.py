from __future__ import annotations

from collections.abc import Awaitable, Callable
from typing import Any

from rich.console import Console
from rich.markup import escape

from wi.commands import bye, commit, gitignore, merge, pr, privacy, unwrap, wrap

Handler = Callable[[str, Console], Awaitable[bool]]

_COMMANDS: dict[str, Handler] = {
    "/gitignore": gitignore.handle,
    "/wrap":      wrap.handle,
    "/unwrap":    unwrap.handle,
    "/bye":       bye.handle,
    "/privacy":   privacy.handle,
    "/pr":        pr.handle,
    "/merge":     merge.handle,
    "/commit":    commit.handle,
}


async def dispatch(text: str, console: Console) -> bool:
    """Route a slash command to its handler. Returns False to exit the REPL."""
    parts = text.split(maxsplit=1)
    name = parts[0].lower()
    args = parts[1].strip() if len(parts) > 1 else ""

    handler = _COMMANDS.get(name)
    if handler is None:
        available = "  ".join(sorted(_COMMANDS))
        console.print(f"[red]unknown command:[/red] {escape(name)}\n[dim]available: {available}[/dim]")
        return True

    return await handler(args, console)
