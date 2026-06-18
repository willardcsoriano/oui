from __future__ import annotations

from rich.console import Console

from wi.commands import wrap


async def handle(args: str, console: Console) -> bool:
    await wrap.handle("", console)
    console.print("[dim]bye[/dim]")
    return False
