from __future__ import annotations

from wi.hooks.loader import HookEntry, HooksConfig
from wi.hooks.runner import run_lifecycle, run_post_tool_use, run_pre_tool_use


def _config(**kwargs) -> HooksConfig:
    return HooksConfig(**kwargs)


# --- run_pre_tool_use ---

def test_pre_tool_use_no_hooks_returns_none():
    result = run_pre_tool_use(_config(), "bash", {"command": "ls"})
    assert result is None


def test_pre_tool_use_passing_hook_returns_none():
    config = _config(pre_tool_use=[HookEntry(command="exit 0", match="bash")])
    result = run_pre_tool_use(config, "bash", {"command": "ls"})
    assert result is None


def test_pre_tool_use_blocking_hook_returns_message():
    config = _config(pre_tool_use=[HookEntry(command="echo 'denied'; exit 1", match="bash")])
    result = run_pre_tool_use(config, "bash", {"command": "rm -rf /"})
    assert result is not None
    assert "denied" in result


def test_pre_tool_use_non_matching_hook_is_skipped():
    config = _config(pre_tool_use=[HookEntry(command="exit 1", match="bash")])
    result = run_pre_tool_use(config, "read_file", {"path": "/etc/passwd"})
    assert result is None


def test_pre_tool_use_wildcard_match_fires_on_all_tools():
    config = _config(pre_tool_use=[HookEntry(command="exit 1", match="*")])
    assert run_pre_tool_use(config, "bash", {}) is not None
    assert run_pre_tool_use(config, "read_file", {}) is not None


def test_pre_tool_use_blocking_hook_with_no_output_returns_default_message():
    config = _config(pre_tool_use=[HookEntry(command="exit 1", match="*")])
    result = run_pre_tool_use(config, "bash", {})
    assert result == "blocked by PreToolUse hook"


def test_pre_tool_use_env_vars_are_passed(tmp_path):
    out = tmp_path / "out.txt"
    config = _config(pre_tool_use=[HookEntry(command=f'echo "$WI_TOOL_NAME" > {out}', match="*")])
    run_pre_tool_use(config, "bash", {})
    assert out.read_text().strip() == "bash"


# --- run_post_tool_use ---

def test_post_tool_use_no_hooks_is_silent():
    run_post_tool_use(_config(), "bash", {}, "ok")  # must not raise


def test_post_tool_use_fires_matching_hook(tmp_path):
    out = tmp_path / "out.txt"
    config = _config(post_tool_use=[HookEntry(command=f'echo "$WI_TOOL_RESULT" > {out}', match="bash")])
    run_post_tool_use(config, "bash", {}, "hello")
    assert out.read_text().strip() == "hello"


def test_post_tool_use_skips_non_matching_hook(tmp_path):
    out = tmp_path / "out.txt"
    config = _config(post_tool_use=[HookEntry(command=f'touch {out}', match="bash")])
    run_post_tool_use(config, "read_file", {}, "content")
    assert not out.exists()


# --- run_lifecycle ---

def test_run_lifecycle_empty_list_is_silent():
    run_lifecycle([])  # must not raise


def test_run_lifecycle_runs_commands(tmp_path):
    out = tmp_path / "out.txt"
    run_lifecycle([f"echo done > {out}"])
    assert out.read_text().strip() == "done"


def test_run_lifecycle_runs_multiple_commands_in_order(tmp_path):
    out = tmp_path / "out.txt"
    run_lifecycle([f"echo first > {out}", f"echo second >> {out}"])
    lines = out.read_text().strip().splitlines()
    assert lines == ["first", "second"]
