from __future__ import annotations

import json

from rich.console import Console
from rich.markup import escape

from oui.commands._utils import git_default_branch, run


async def handle(args: str, console: Console) -> bool:
    default = await git_default_branch()

    result = await run([
        "gh", "pr", "list",
        "--state", "open",
        "--base", default,
        "--json", "number,headRefName,isDraft",
    ])
    if result.returncode != 0:
        console.print(f"[red]gh pr list failed:[/red] {escape(result.stderr.strip())}")
        return True

    prs = json.loads(result.stdout or "[]")
    if not prs:
        console.print("[dim]no open PRs targeting[/dim] " + default)
        return True

    for pr in prs:
        number = pr["number"]
        branch = pr["headRefName"]
        if pr["isDraft"]:
            console.print(f"[dim]skipping draft PR #{number} ({branch})[/dim]")
            continue

        merge = await run([
            "gh", "pr", "merge", str(number),
            "--auto", "--squash", "--delete-branch",
        ])
        if merge.returncode != 0:
            console.print(f"[red]could not queue PR #{number}:[/red] {escape(merge.stderr.strip())}")
        else:
            console.print(f"[green]queued[/green] PR #{number} [dim]({branch})[/dim] — merge-when-green")

    return True
