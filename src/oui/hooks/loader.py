from __future__ import annotations

import tomllib
from dataclasses import dataclass, field
from pathlib import Path

DEFAULT_CONFIG_PATH = Path.home() / ".wi" / "hooks.toml"


@dataclass
class HookEntry:
    command: str
    match: str = "*"


@dataclass
class HooksConfig:
    pre_tool_use: list[HookEntry] = field(default_factory=list)
    post_tool_use: list[HookEntry] = field(default_factory=list)
    session_start: list[str] = field(default_factory=list)
    session_end: list[str] = field(default_factory=list)


def load(path: Path = DEFAULT_CONFIG_PATH) -> HooksConfig:
    if not path.exists():
        return HooksConfig()

    with path.open("rb") as f:
        data = tomllib.load(f)

    pre = [HookEntry(**e) for e in data.get("pre_tool_use", [])]
    post = [HookEntry(**e) for e in data.get("post_tool_use", [])]

    session = data.get("session", {})
    start_raw = session.get("start", [])
    end_raw = session.get("end", [])

    return HooksConfig(
        pre_tool_use=pre,
        post_tool_use=post,
        session_start=[start_raw] if isinstance(start_raw, str) else list(start_raw),
        session_end=[end_raw] if isinstance(end_raw, str) else list(end_raw),
    )
