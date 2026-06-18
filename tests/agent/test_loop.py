from __future__ import annotations

import asyncio
from unittest.mock import patch

import pytest

from wi.agent.loop import _execute
from wi.hooks.loader import HooksConfig
from wi.tools.executor import DeniedCommandError

_EMPTY_HOOKS = HooksConfig()


def _call(name: str, args: str, call_id: str = "call_1") -> dict:
    return {"id": call_id, "function": {"name": name, "arguments": args}}


@pytest.mark.anyio
async def test_execute_returns_tool_result():
    call = _call("bash", '{"command": "echo hi"}')
    with patch("wi.agent.loop.execute_tool", return_value="hi"):
        result = await _execute(call, _EMPTY_HOOKS)
    assert result == "hi"


@pytest.mark.anyio
async def test_execute_malformed_args_returns_error():
    call = _call("bash", "not-json")
    result = await _execute(call, _EMPTY_HOOKS)
    assert result == "error: malformed tool arguments"


@pytest.mark.anyio
async def test_execute_denied_command_returns_error():
    call = _call("bash", '{"command": "rm -rf /"}')
    with patch("wi.agent.loop.execute_tool", side_effect=DeniedCommandError("blocked")):
        result = await _execute(call, _EMPTY_HOOKS)
    assert "blocked" in result


@pytest.mark.anyio
async def test_execute_concurrent_results_in_call_order():
    calls = [
        _call("bash", '{"command": "echo first"}', call_id="c1"),
        _call("bash", '{"command": "echo second"}', call_id="c2"),
    ]
    with patch("wi.agent.loop.execute_tool", side_effect=["first", "second"]):
        results = await asyncio.gather(*[_execute(c, _EMPTY_HOOKS) for c in calls])
    assert results == ["first", "second"]


@pytest.mark.anyio
async def test_execute_unexpected_error_returns_error_string():
    call = _call("bash", '{"command": "boom"}')
    with patch("wi.agent.loop.execute_tool", side_effect=RuntimeError("oops")):
        result = await _execute(call, _EMPTY_HOOKS)
    assert "oops" in result
