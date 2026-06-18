from __future__ import annotations

import asyncio

import typer
from rich.console import Console

from wi import __version__
from wi.agent import loop

app = typer.Typer(add_completion=False, no_args_is_help=False)
console = Console()


@app.command()
def run(
    prompt: str | None = typer.Argument(default=None, help="Prompt to run; omit to enter REPL"),
    version: bool = typer.Option(False, "--version", "-v", help="Show version and exit", is_eager=True),
    model: str | None = typer.Option(None, "--model", "-m", help="Model override (e.g. anthropic/claude-opus-4-8)"),
) -> None:
    if version:
        console.print(f"wi {__version__}")
        raise typer.Exit()
    if prompt is None:
        asyncio.run(_async_repl(model))
    else:
        asyncio.run(loop.run(prompt, model))


async def _async_repl(model: str | None) -> None:
    console.print(f"[bold]wi[/bold] [dim]{__version__}[/dim]  [dim]ctrl-c or /bye to exit[/dim]")
    while True:
        try:
            text = console.input("[bold cyan]>[/bold cyan] ").strip()
        except (KeyboardInterrupt, EOFError):
            console.print("\n[dim]bye[/dim]")
            break
        if not text:
            continue
        if text in {"/bye", "exit", "quit"}:
            console.print("[dim]bye[/dim]")
            break
        await loop.run(text, model)
