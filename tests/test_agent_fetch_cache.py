"""Tests for the agent WebFetch summary recovery used as a compile-time fallback.

When the Python fetcher can't reach a URL (e.g. Cloudflare wall), compile
falls back to whatever summary the agent captured during research. The cache
is derived live from the `.liner-runs/*.jsonl` transcripts — no separate
persistence — and these tests cover the parsing + lookup logic.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from liner.agent_fetch_cache import (
    MIN_USABLE_SUMMARY_CHARS,
    build_agent_fetch_cache,
    title_from_summary,
)


def _write_run(
    folder: Path,
    task: str,
    *,
    started_at: str,
    fetches: list[tuple[str, str, str]],  # (call_id, url, result_body)
) -> Path:
    """Write a synthetic JSONL run log with WebFetch tool_use/tool_result pairs."""
    run_dir = folder / ".liner-runs" / task
    run_dir.mkdir(parents=True, exist_ok=True)
    safe_stamp = started_at.replace(":", "-")
    path = run_dir / f"{safe_stamp}.jsonl"

    lines: list[dict] = [
        {
            "type": "_liner_meta",
            "agent": "claude",
            "resume": False,
            "taskLabel": task,
            "startedAt": started_at,
        }
    ]
    # One assistant message with every tool_use, then one user message per
    # tool_result. Matches the shape Claude's SDK actually emits.
    if fetches:
        lines.append(
            {
                "type": "assistant",
                "message": {
                    "model": "claude-opus-4-7",
                    "content": [
                        {"type": "tool_use", "id": cid, "name": "WebFetch", "input": {"url": url}}
                        for cid, url, _ in fetches
                    ],
                },
            }
        )
        for cid, _, body in fetches:
            lines.append(
                {
                    "type": "user",
                    "message": {
                        "content": [
                            {
                                "type": "tool_result",
                                "tool_use_id": cid,
                                "content": [{"type": "text", "text": body}],
                            }
                        ]
                    },
                }
            )
    lines.append({"type": "_liner_close", "exitCode": 0, "endedAt": started_at})
    path.write_text("\n".join(json.dumps(line) for line in lines) + "\n", encoding="utf-8")
    return path


@pytest.fixture
def folder(tmp_path: Path) -> Path:
    return tmp_path


def test_empty_folder_returns_empty_cache(folder: Path) -> None:
    assert build_agent_fetch_cache(folder) == {}


def test_pairs_tool_use_with_tool_result(folder: Path) -> None:
    body = "# Article Summary\n**Title:** Designing CLIs\n**Author:** Test\n\nLong enough summary body."
    _write_run(
        folder,
        "candidates",
        started_at="2026-05-22T10:00:00Z",
        fetches=[("toolu_abc", "https://example.com/article", body)],
    )
    cache = build_agent_fetch_cache(folder)
    assert "https://example.com/article" in cache
    assert cache["https://example.com/article"].body == body
    assert cache["https://example.com/article"].captured_at == "2026-05-22T10:00:00Z"


def test_skips_summaries_shorter_than_threshold(folder: Path) -> None:
    """An error envelope or one-line result isn't usable as fallback content."""
    short = "Failed"  # well under MIN_USABLE_SUMMARY_CHARS
    assert len(short) < MIN_USABLE_SUMMARY_CHARS
    _write_run(
        folder,
        "candidates",
        started_at="2026-05-22T10:00:00Z",
        fetches=[("toolu_abc", "https://example.com/article", short)],
    )
    assert build_agent_fetch_cache(folder) == {}


def test_latest_fetch_wins_when_multiple_runs_touched_same_url(folder: Path) -> None:
    old_body = "OLD summary — captured first." + " padding" * 20
    new_body = "NEW summary — captured later." + " padding" * 20
    _write_run(
        folder,
        "candidates",
        started_at="2026-05-22T10:00:00Z",
        fetches=[("toolu_a", "https://example.com/x", old_body)],
    )
    _write_run(
        folder,
        "evaluation",
        started_at="2026-05-22T12:00:00Z",
        fetches=[("toolu_b", "https://example.com/x", new_body)],
    )
    cache = build_agent_fetch_cache(folder)
    assert cache["https://example.com/x"].body == new_body
    assert cache["https://example.com/x"].captured_at == "2026-05-22T12:00:00Z"


def test_multiple_urls_in_one_run(folder: Path) -> None:
    body_a = "Summary A " * 20
    body_b = "Summary B " * 20
    _write_run(
        folder,
        "candidates",
        started_at="2026-05-22T10:00:00Z",
        fetches=[
            ("toolu_a", "https://example.com/a", body_a),
            ("toolu_b", "https://example.com/b", body_b),
        ],
    )
    cache = build_agent_fetch_cache(folder)
    assert set(cache) == {"https://example.com/a", "https://example.com/b"}


def test_non_webfetch_tool_uses_are_ignored(folder: Path) -> None:
    """Read / Grep / etc. tool_use events should not appear as fetches."""
    run_dir = folder / ".liner-runs" / "candidates"
    run_dir.mkdir(parents=True, exist_ok=True)
    path = run_dir / "2026-05-22T10-00-00.jsonl"
    lines = [
        {"type": "_liner_meta", "agent": "claude", "taskLabel": "candidates", "startedAt": "2026-05-22T10:00:00Z"},
        {
            "type": "assistant",
            "message": {
                "content": [
                    {"type": "tool_use", "id": "toolu_r", "name": "Read", "input": {"file_path": "/tmp/x"}},
                ],
            },
        },
        {
            "type": "user",
            "message": {
                "content": [
                    {"type": "tool_result", "tool_use_id": "toolu_r", "content": [{"type": "text", "text": "file body" * 50}]},
                ]
            },
        },
        {"type": "_liner_close", "exitCode": 0, "endedAt": "2026-05-22T10:00:01Z"},
    ]
    path.write_text("\n".join(json.dumps(line) for line in lines) + "\n", encoding="utf-8")
    assert build_agent_fetch_cache(folder) == {}


def test_malformed_lines_are_skipped(folder: Path) -> None:
    body = "Valid summary content " * 20
    log = _write_run(
        folder,
        "candidates",
        started_at="2026-05-22T10:00:00Z",
        fetches=[("toolu_a", "https://example.com/x", body)],
    )
    with log.open("a", encoding="utf-8") as fp:
        fp.write('{"type":"assistant","message":{"content":[{"type":"tool_u\n')
    cache = build_agent_fetch_cache(folder)
    assert "https://example.com/x" in cache


def test_title_from_summary_prefers_explicit_title_line() -> None:
    summary = "# Article Summary\n\n**Title:** How CLIs Should Behave\n\n**Author:** Jane Doe"
    assert title_from_summary(summary, fallback="x") == "How CLIs Should Behave"


def test_title_from_summary_falls_back_to_first_line() -> None:
    summary = "An interesting take on terminal UX patterns and design.\n\nMore text follows."
    assert title_from_summary(summary, fallback="x") == "An interesting take on terminal UX patterns and design."


def test_title_from_summary_falls_back_to_url_when_unparseable() -> None:
    assert title_from_summary("", fallback="https://example.com") == "https://example.com"
