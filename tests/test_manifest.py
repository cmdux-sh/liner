"""Tests for the mixtape process-manifest aggregator.

The aggregator walks `.liner-runs/<task>/<ts>.jsonl` files and rolls them
into a single `process.json`. These tests use synthetic JSONL fixtures
that mimic the real Claude-agent envelope shape — the parser only cares
about three line types (`_liner_meta`, `assistant`, `result`, `_liner_close`)
so the fixtures stay small and focused.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from liner.manifest import (
    MANIFEST_FILENAME,
    build_manifest,
    parse_run_log,
    read_manifest,
    write_manifest,
)


def _write_run_log(
    folder: Path,
    task: str,
    *,
    started_at: str,
    ended_at: str,
    agent: str = "claude",
    model: str = "claude-opus-4-7",
    tool_calls: list[dict] | None = None,
    usage: dict | None = None,
    cost_usd: float | None = 0.5,
    num_turns: int = 5,
    exit_code: int = 0,
) -> Path:
    """Write a synthetic .liner-runs JSONL file mimicking Claude's stream."""
    run_dir = folder / ".liner-runs" / task
    run_dir.mkdir(parents=True, exist_ok=True)
    safe_stamp = started_at.replace(":", "-")
    path = run_dir / f"{safe_stamp}.jsonl"

    lines: list[dict] = []
    lines.append(
        {
            "type": "_liner_meta",
            "agent": agent,
            "resume": False,
            "taskLabel": task,
            "startedAt": started_at,
        }
    )
    if tool_calls:
        lines.append(
            {
                "type": "assistant",
                "message": {
                    "model": model,
                    "content": [
                        {"type": "tool_use", "name": tc["name"], "input": tc.get("input") or {}}
                        for tc in tool_calls
                    ],
                },
            }
        )
    lines.append(
        {
            "type": "result",
            "subtype": "success",
            "is_error": False,
            "duration_ms": 1234,
            "num_turns": num_turns,
            "total_cost_usd": cost_usd,
            "usage": usage
            or {
                "input_tokens": 100,
                "output_tokens": 50,
                "cache_read_input_tokens": 1000,
                "cache_creation_input_tokens": 200,
            },
            "modelUsage": {model: {"inputTokens": 100, "outputTokens": 50}},
        }
    )
    lines.append({"type": "_liner_close", "exitCode": exit_code, "endedAt": ended_at})

    path.write_text("\n".join(json.dumps(line) for line in lines) + "\n", encoding="utf-8")
    return path


@pytest.fixture
def folder(tmp_path: Path) -> Path:
    """A minimal mixtape folder with a tape.yaml so manifest can read metadata."""
    (tmp_path / "tape.yaml").write_text(
        "title: Test Mixtape\n"
        "description: A fixture\n"
        "version: 1\n"
        "curator: tester\n"
        "jtbd: prove the aggregator works\n"
        "mode: methodology\n"
        "sources:\n"
        "  - type: web\n"
        "    url: https://example.com\n",
        encoding="utf-8",
    )
    return tmp_path


def test_empty_folder_yields_empty_manifest(folder: Path) -> None:
    m = build_manifest(folder).to_dict()
    assert m["totals"]["runs"] == 0
    assert m["totals"]["tool_calls"] == 0
    assert m["totals"]["fetches"] == 0
    assert m["totals"]["cost_usd"] is None  # no runs → unknown, not 0
    assert m["runs"] == []
    assert m["domains"] == []
    assert m["agents_used"] == []
    assert m["models_used"] == []
    # Mixtape metadata still populated from tape.yaml.
    assert m["mixtape"]["title"] == "Test Mixtape"


def test_single_run_rollup(folder: Path) -> None:
    _write_run_log(
        folder,
        "framing",
        started_at="2026-05-22T10:00:00Z",
        ended_at="2026-05-22T10:00:30Z",
        tool_calls=[
            {"name": "Read"},
            {"name": "Read"},
            {"name": "Write"},
        ],
    )
    m = build_manifest(folder).to_dict()
    assert m["totals"]["runs"] == 1
    assert m["totals"]["tool_calls"] == 3
    assert m["totals"]["fetches"] == 0
    assert m["totals"]["cost_usd"] == pytest.approx(0.5)
    assert m["agents_used"] == ["claude"]
    assert m["models_used"] == ["claude-opus-4-7"]
    run = m["runs"][0]
    assert run["task_label"] == "framing"
    assert run["tools"] == {"Read": 2, "Write": 1}
    assert run["duration_s"] == pytest.approx(30.0)


def test_fetches_and_domain_dedup(folder: Path) -> None:
    _write_run_log(
        folder,
        "candidates",
        started_at="2026-05-22T11:00:00Z",
        ended_at="2026-05-22T11:00:10Z",
        tool_calls=[
            {"name": "WebFetch", "input": {"url": "https://www.youtube.com/watch?v=a"}},
            {"name": "WebFetch", "input": {"url": "https://youtube.com/watch?v=b"}},
            {"name": "WebFetch", "input": {"url": "https://brandur.org/interfaces"}},
            {"name": "Read"},
        ],
    )
    m = build_manifest(folder).to_dict()
    assert m["totals"]["fetches"] == 3
    # www. prefix dropped → both YouTube fetches collapse to one domain.
    domain_map = {d["domain"]: d["count"] for d in m["domains"]}
    assert domain_map == {"youtube.com": 2, "brandur.org": 1}
    # Domains sorted by descending count.
    assert m["domains"][0]["domain"] == "youtube.com"
    # Flat fetch list preserves order and metadata.
    assert m["fetches"][0]["url"] == "https://www.youtube.com/watch?v=a"
    assert m["fetches"][0]["task_label"] == "candidates"


def test_multiple_runs_aggregate_tokens_and_cost(folder: Path) -> None:
    _write_run_log(
        folder,
        "framing",
        started_at="2026-05-22T10:00:00Z",
        ended_at="2026-05-22T10:00:30Z",
        usage={"input_tokens": 100, "output_tokens": 50, "cache_read_input_tokens": 1000, "cache_creation_input_tokens": 200},
        cost_usd=0.25,
    )
    _write_run_log(
        folder,
        "candidates",
        started_at="2026-05-22T11:00:00Z",
        ended_at="2026-05-22T11:00:10Z",
        usage={"input_tokens": 200, "output_tokens": 75, "cache_read_input_tokens": 5000, "cache_creation_input_tokens": 0},
        cost_usd=1.5,
    )
    m = build_manifest(folder).to_dict()
    assert m["totals"]["runs"] == 2
    assert m["totals"]["cost_usd"] == pytest.approx(1.75)
    assert m["totals"]["tokens"] == {
        "input": 300,
        "output": 125,
        "cache_read": 6000,
        "cache_create": 200,
    }


def test_runs_sorted_chronologically(folder: Path) -> None:
    _write_run_log(folder, "later", started_at="2026-05-22T12:00:00Z", ended_at="2026-05-22T12:00:05Z")
    _write_run_log(folder, "earlier", started_at="2026-05-22T08:00:00Z", ended_at="2026-05-22T08:00:05Z")
    m = build_manifest(folder).to_dict()
    assert [r["task_label"] for r in m["runs"]] == ["earlier", "later"]


def test_missing_cost_yields_unknown_total(folder: Path) -> None:
    """Codex / older agents may not include total_cost_usd. Aggregate should
    flag the total as None rather than silently treating missing as 0."""
    _write_run_log(
        folder,
        "framing",
        started_at="2026-05-22T10:00:00Z",
        ended_at="2026-05-22T10:00:05Z",
        cost_usd=None,
    )
    m = build_manifest(folder).to_dict()
    assert m["totals"]["cost_usd"] is None


def test_malformed_lines_are_skipped(folder: Path) -> None:
    """A truncated JSON line in the middle of a stream shouldn't crash the
    parser — we'd rather report a partial run than fail the whole manifest."""
    _write_run_log(
        folder,
        "framing",
        started_at="2026-05-22T10:00:00Z",
        ended_at="2026-05-22T10:00:05Z",
    )
    log_path = next((folder / ".liner-runs" / "framing").glob("*.jsonl"))
    # Append a deliberately broken line.
    with log_path.open("a", encoding="utf-8") as fp:
        fp.write('{"type":"assistant","message":{"model":"claude-opus-4-7","content":[{"type":"tool_u\n')
    # Should still parse the rest and yield one valid run.
    m = build_manifest(folder).to_dict()
    assert m["totals"]["runs"] == 1


def test_legacy_phase_id_meta_is_accepted(folder: Path) -> None:
    """Older runs used `phaseId` in the meta envelope before it was renamed
    to `taskLabel`. Parser should accept either."""
    run_dir = folder / ".liner-runs" / "framing"
    run_dir.mkdir(parents=True, exist_ok=True)
    path = run_dir / "2026-05-22T10-00-00.jsonl"
    path.write_text(
        json.dumps(
            {"type": "_liner_meta", "agent": "claude", "phaseId": "framing", "startedAt": "2026-05-22T10:00:00Z"}
        )
        + "\n"
        + json.dumps(
            {"type": "result", "num_turns": 1, "total_cost_usd": 0.1, "usage": {"input_tokens": 1, "output_tokens": 1}}
        )
        + "\n"
        + json.dumps({"type": "_liner_close", "exitCode": 0, "endedAt": "2026-05-22T10:00:01Z"})
        + "\n",
        encoding="utf-8",
    )
    summary = parse_run_log(path)
    assert summary is not None
    assert summary.task_label == "framing"


def test_missing_meta_returns_none(folder: Path) -> None:
    run_dir = folder / ".liner-runs" / "broken"
    run_dir.mkdir(parents=True, exist_ok=True)
    path = run_dir / "garbage.jsonl"
    path.write_text('{"type":"result"}\n', encoding="utf-8")
    assert parse_run_log(path) is None


def test_write_and_read_roundtrip(folder: Path) -> None:
    _write_run_log(folder, "framing", started_at="2026-05-22T10:00:00Z", ended_at="2026-05-22T10:00:05Z")
    m = build_manifest(folder)
    target = write_manifest(folder, m)
    assert target.name == MANIFEST_FILENAME
    loaded = read_manifest(folder)
    assert loaded is not None
    assert loaded["totals"]["runs"] == 1
    assert loaded["mixtape"]["title"] == "Test Mixtape"


def test_read_manifest_missing_returns_none(folder: Path) -> None:
    assert read_manifest(folder) is None
