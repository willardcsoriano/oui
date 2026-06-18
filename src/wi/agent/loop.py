from __future__ import annotations

import asyncio
import json
from typing import Any

import litellm
from rich.console import Console
from rich.live import Live
from rich.markup import escape
from rich.text import Text

from wi.hooks import loader as hooks_loader
from wi.hooks import runner as hooks_runner
from wi.hooks.loader import HooksConfig
from wi.providers.client import stream_completion
from wi.tools.executor import DeniedCommandError, execute_tool

console = Console()


async def run(prompt: str, model: str | None = None) -> None:
    hooks_config = hooks_loader.load()
    messages: list[dict[str, Any]] = [{"role": "user", "content": prompt}]
    while True:
        try:
            assistant_msg, tool_calls = _stream_turn(messages, model)
        except litellm.AuthenticationError:
            console.print(
                "[bold red]authentication error[/bold red] — "
                "check your API key env var (e.g. ANTHROPIC_API_KEY, OPENAI_API_KEY)"
            )
            return
        except litellm.BadRequestError as e:
            console.print(f"[bold red]bad request:[/bold red] {escape(str(e))}")
            return
        except ValueError as e:
            console.print(f"[bold red]config error:[/bold red] {escape(str(e))}")
            return
        messages.append(assistant_msg)
        if not tool_calls:
            break
        results = await asyncio.gather(*[_execute(call, hooks_config) for call in tool_calls])
        for call, result in zip(tool_calls, results):
            messages.append({
                "role": "tool",
                "tool_call_id": call["id"],
                "content": result,
            })


def _stream_turn(
    messages: list[dict[str, Any]],
    model: str | None,
) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    text_buf = ""
    tool_call_bufs: dict[int, dict[str, Any]] = {}

    stream = stream_completion(messages, model)

    with Live("", console=console, refresh_per_second=20) as live:
        for chunk in stream:
            delta = chunk.choices[0].delta

            if delta.content:
                text_buf += delta.content
                live.update(Text(text_buf))

            if delta.tool_calls:
                for tc in delta.tool_calls:
                    idx = tc.index
                    if idx not in tool_call_bufs:
                        tool_call_bufs[idx] = {
                            "id": tc.id or "",
                            "type": "function",
                            "function": {"name": "", "arguments": ""},
                        }
                    buf = tool_call_bufs[idx]
                    if tc.id:
                        buf["id"] = tc.id
                    if tc.function:
                        if tc.function.name:
                            buf["function"]["name"] += tc.function.name
                        if tc.function.arguments:
                            buf["function"]["arguments"] += tc.function.arguments

    if text_buf:
        console.print()

    tool_calls = list(tool_call_bufs.values())

    assistant_msg: dict[str, Any] = {"role": "assistant", "content": text_buf or None}
    if tool_calls:
        assistant_msg["tool_calls"] = tool_calls

    return assistant_msg, tool_calls


async def _execute(call: dict[str, Any], hooks_config: HooksConfig) -> str:
    name = call["function"]["name"]
    try:
        args = json.loads(call["function"]["arguments"])
    except json.JSONDecodeError:
        return "error: malformed tool arguments"

    blocked = await asyncio.to_thread(hooks_runner.run_pre_tool_use, hooks_config, name, args)
    if blocked:
        console.print(f"[bold red]✗[/bold red] [red]hook blocked[/red] [yellow]{name}[/yellow]: {escape(blocked)}")
        return f"error: {blocked}"

    console.print(f"[bold yellow]⚙[/bold yellow] [yellow]{name}[/yellow]([dim]{escape(call['function']['arguments'])}[/dim])")

    try:
        result = await asyncio.to_thread(execute_tool, name, args)
    except DeniedCommandError as e:
        result = f"error: {e}"
        console.print(f"[bold red]✗[/bold red] [red]blocked:[/red] {escape(str(e))}")
        return result
    except Exception as e:
        result = f"error: {e}"

    await asyncio.to_thread(hooks_runner.run_post_tool_use, hooks_config, name, args, result)

    preview = result if len(result) <= 120 else result[:120] + "…"
    console.print(f"[dim]  → {escape(preview)}[/dim]")
    return result
