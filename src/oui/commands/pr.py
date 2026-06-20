from __future__ import annotations

from rich.console import Console
from rich.markup import escape

from oui.commands._utils import git_current_branch, git_default_branch, run


async def handle(args: str, console: Console) -> bool:
    branch = await git_current_branch()
    default = await git_default_branch()

    if not branch:
        console.print("[red]not on a branch (detached HEAD?)[/red]")
        return True
    if branch == default:
        console.print(f"[red]already on default branch ({default})[/red] — create a feature branch first")
        return True

    status = await run(["git", "status", "--short"])
    if status.stdout.strip():
        console.print("[yellow]warning:[/yellow] uncommitted changes present — commit or stash before opening a PR")

    console.print(f"[dim]pushing {branch}…[/dim]")
    push = await run(["git", "push", "-u", "origin", branch])
    if push.returncode != 0:
        console.print(f"[red]push failed:[/red] {escape(push.stderr.strip())}")
        return True

    console.print("[dim]creating PR…[/dim]")
    pr = await run(["gh", "pr", "create", "--fill", "--base", default])
    if pr.returncode != 0:
        console.print(f"[red]gh pr create failed:[/red] {escape(pr.stderr.strip())}")
        return True

    url = pr.stdout.strip()
    console.print(f"[green]PR opened:[/green] {escape(url)}")
    return True
