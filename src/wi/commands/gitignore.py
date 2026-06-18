from __future__ import annotations

from pathlib import Path

from rich.console import Console

_STACK_MARKERS: dict[str, list[str]] = {
    "python":     ["pyproject.toml", "setup.py", "setup.cfg", "requirements.txt"],
    "node":       ["package.json"],
    "go":         ["go.mod"],
    "rust":       ["Cargo.toml"],
    "java":       ["pom.xml", "build.gradle"],
    "ruby":       ["Gemfile"],
    "dotnet":     ["*.csproj", "*.sln"],
}

_ENTRIES: dict[str, list[str]] = {
    "python": [
        "__pycache__/", "*.py[cod]", ".venv/", "venv/", "dist/", "build/",
        "*.egg-info/", ".eggs/", ".pytest_cache/", ".mypy_cache/", ".ruff_cache/",
        ".coverage", "htmlcov/",
    ],
    "node": [
        "node_modules/", "dist/", "build/", ".next/", ".nuxt/", ".cache/",
        "*.log", "npm-debug.log*", "yarn-debug.log*",
    ],
    "go": [
        "*.exe", "*.exe~", "*.dll", "*.so", "*.dylib", "*.test", "*.out",
        "vendor/",
    ],
    "rust": [
        "target/", "Cargo.lock",
    ],
    "java": [
        "*.class", "*.jar", "*.war", "*.ear", "target/", ".gradle/", "build/",
        ".idea/", "*.iml",
    ],
    "ruby": [
        ".bundle/", "vendor/bundle/", "*.gem", "log/", "tmp/",
    ],
    "dotnet": [
        "bin/", "obj/", "*.user", ".vs/",
    ],
    "common": [
        ".env", ".env.*", "!.env.example", ".DS_Store", "Thumbs.db",
        "*.swp", "*.swo", "*~",
    ],
}


def _detect_stacks(cwd: Path) -> list[str]:
    stacks: list[str] = ["common"]
    files = {f.name for f in cwd.iterdir() if f.is_file()}
    for stack, markers in _STACK_MARKERS.items():
        if any(
            any(f == m or (m.startswith("*") and f.endswith(m[1:])) for f in files)
            for m in markers
        ):
            stacks.append(stack)
    return stacks


async def handle(args: str, console: Console) -> bool:
    cwd = Path.cwd()
    stacks = _detect_stacks(cwd)
    entries: list[str] = []
    for stack in stacks:
        entries.extend(_ENTRIES.get(stack, []))

    gitignore_path = cwd / ".gitignore"
    existing_lines = set(gitignore_path.read_text().splitlines()) if gitignore_path.exists() else set()
    new_entries = [e for e in entries if e not in existing_lines]

    if not new_entries:
        console.print("[dim].gitignore is already up to date[/dim]")
        return True

    with gitignore_path.open("a") as f:
        if gitignore_path.exists() and gitignore_path.stat().st_size > 0:
            f.write("\n")
        f.write("# wi/gitignore — detected: " + ", ".join(s for s in stacks if s != "common") + "\n")
        f.write("\n".join(new_entries) + "\n")

    detected = [s for s in stacks if s != "common"]
    console.print(
        f"[green]added {len(new_entries)} entries[/green] "
        f"[dim](detected: {', '.join(detected) or 'none'})[/dim]"
    )
    return True
