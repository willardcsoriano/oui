from __future__ import annotations

from pathlib import Path

from rich.console import Console
from rich.markdown import Markdown

from wi.commands.wrap import HANDOFF_PATH


async def handle(args: str, console: Console) -> bool:
    if not HANDOFF_PATH.exists():
        console.print(f"[red]no {HANDOFF_PATH} found[/red] — run [bold]/wrap[/bold] first")
        return True

    content = HANDOFF_PATH.read_text()
    console.print(Markdown(content))
    return True
