from __future__ import annotations

from pathlib import Path

from rich.console import Console

from oui.commands._utils import git_current_branch, run

HANDOFF_PATH = Path("HANDOFF.md")


async def handle(args: str, console: Console) -> bool:
    branch = await git_current_branch()
    log = (await run(["git", "log", "--oneline", "-10"])).stdout.strip()
    diff_stat = (await run(["git", "diff", "--stat"])).stdout.strip()
    status = (await run(["git", "status", "--short"])).stdout.strip()

    content = f"""\
## Overview

[Fill in: what you were working on, the goal, and any context a cold reader needs.]

---

## Session State

**Branch:** `{branch}`

**Recent commits:**

```
{log or "(none)"}
```

**Uncommitted changes:**

```
{diff_stat or "(none)"}
```

**Working tree status:**

```
{status or "(clean)"}
```

---

## Key Decisions

- [Fill in: decisions made during this session and the reasoning]

---

## Next Steps

- [Fill in: what to do next, in order]

---

## Blockers

- [Fill in: anything blocking progress]
"""

    HANDOFF_PATH.write_text(content)
    console.print(f"[green]wrote[/green] {HANDOFF_PATH}  [dim](edit the placeholder sections before resuming)[/dim]")
    return True
