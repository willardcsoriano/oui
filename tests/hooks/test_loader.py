from __future__ import annotations

from pathlib import Path

import pytest

from wi.hooks.loader import HookEntry, HooksConfig, load


def test_load_returns_empty_config_when_file_absent(tmp_path):
    config = load(tmp_path / "nonexistent.toml")
    assert config == HooksConfig()


def test_load_pre_tool_use(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[[pre_tool_use]]\ncommand = "echo hi"\nmatch = "bash"\n')
    config = load(p)
    assert config.pre_tool_use == [HookEntry(command="echo hi", match="bash")]


def test_load_pre_tool_use_default_match(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[[pre_tool_use]]\ncommand = "check"\n')
    config = load(p)
    assert config.pre_tool_use[0].match == "*"


def test_load_post_tool_use(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[[post_tool_use]]\ncommand = "log"\nmatch = "*"\n')
    config = load(p)
    assert config.post_tool_use == [HookEntry(command="log", match="*")]


def test_load_session_start_string(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[session]\nstart = "notify start"\n')
    config = load(p)
    assert config.session_start == ["notify start"]


def test_load_session_end_string(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[session]\nend = "cleanup"\n')
    config = load(p)
    assert config.session_end == ["cleanup"]


def test_load_session_start_array(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text('[session]\nstart = ["cmd1", "cmd2"]\n')
    config = load(p)
    assert config.session_start == ["cmd1", "cmd2"]


def test_load_multiple_hooks(tmp_path):
    p = tmp_path / "hooks.toml"
    p.write_text(
        '[[pre_tool_use]]\ncommand = "first"\n'
        '[[pre_tool_use]]\ncommand = "second"\nmatch = "bash"\n'
    )
    config = load(p)
    assert len(config.pre_tool_use) == 2
    assert config.pre_tool_use[1].match == "bash"
