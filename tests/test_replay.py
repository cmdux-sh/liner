"""Tests for the `liner replay` CLI command.

The command clones the curator-facing inputs (JTBD, clarifications, mode,
title, description, curator) from an existing tape.yaml into a freshly-
scaffolded project folder. Everything that's a *result* of the pipeline
(sources list, working/, synthesis.md, MIXTAPE.md, sources/) is intentionally
NOT carried over — the whole point is to re-run the pipeline against the
same input.

These tests cover:
  - the happy path (clone + parent set)
  - clarifications round-trip through the replay
  - explicit --out destinations
  - the destination-exists guard (with --force override)
  - the input tape's sources are dropped (replay starts empty)
"""

from __future__ import annotations

from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app

runner = CliRunner()


def _write_source_tape(folder: Path, **overrides: object) -> None:
    """Scaffold a minimal source project folder with a tape.yaml the replay
    command can read. Optional overrides extend / replace the base fields.
    """
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "synthesis.md").write_text("# Synthesis\n\nReal content.\n")
    (folder / "working").mkdir(exist_ok=True)
    (folder / "working" / "01-jtbd-and-knowledge-map.md").write_text(
        "# Knowledge map\n\nSomething.\n"
    )
    tape: dict[str, object] = {
        "title": "Source Tape",
        "description": "Original",
        "version": 1,
        "curator": "tester",
        "mode": "quick",
        "jtbd": "Write good CLI prose.",
        "sources": [
            {"type": "web", "url": "https://example.com/one"},
            {"type": "web", "url": "https://example.com/two"},
        ],
    }
    tape.update(overrides)
    (folder / "tape.yaml").write_text(
        yaml.safe_dump(tape, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )


def _load(path: Path) -> dict[str, object]:
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def test_replay_clones_inputs_into_default_destination(tmp_path: Path) -> None:
    src = tmp_path / "original"
    _write_source_tape(src)

    result = runner.invoke(app, ["replay", str(src)])
    assert result.exit_code == 0, result.stdout

    dest = tmp_path / "original-replay"
    assert dest.exists(), "default destination should be <source>-replay"
    tape = _load(dest / "tape.yaml")

    # Curator-facing inputs cloned.
    assert tape["jtbd"] == "Write good CLI prose."
    assert tape["mode"] == "quick"
    assert tape["curator"] == "tester"
    # Parent is recorded so Phase 8 can run a comparison test.
    assert tape["parent"] == str(src.resolve())
    # Sources reset — the replay regenerates the pipeline from scratch.
    assert tape["sources"] == []


def test_replay_carries_jtbd_clarifications(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write_source_tape(
        src,
        jtbd_clarifications=[
            {"question": "How weighted are X and Y?", "answer": "Equally."},
            {"question": "Quality anchors?", "answer": "ripgrep, fzf"},
        ],
    )

    result = runner.invoke(app, ["replay", str(src)])
    assert result.exit_code == 0, result.stdout

    dest_tape = _load(tmp_path / "src-replay" / "tape.yaml")
    clars = dest_tape.get("jtbd_clarifications")
    assert isinstance(clars, list) and len(clars) == 2
    assert clars[0]["answer"] == "Equally."


def test_replay_honors_explicit_out(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write_source_tape(src)
    out = tmp_path / "custom-destination"

    result = runner.invoke(app, ["replay", str(src), "--out", str(out)])
    assert result.exit_code == 0, result.stdout
    assert out.exists()
    assert (out / "tape.yaml").exists()


def test_replay_refuses_existing_destination_without_force(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write_source_tape(src)
    dest = tmp_path / "already-here"
    dest.mkdir()
    (dest / "tape.yaml").write_text("existing\n")

    result = runner.invoke(app, ["replay", str(src), "--out", str(dest)])
    assert result.exit_code != 0
    # Existing file should still be in place — not clobbered.
    assert (dest / "tape.yaml").read_text() == "existing\n"


def test_replay_force_overwrites_destination(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write_source_tape(src)
    dest = tmp_path / "overwrite-me"
    dest.mkdir()
    (dest / "tape.yaml").write_text("existing\n")

    result = runner.invoke(
        app, ["replay", str(src), "--out", str(dest), "--force"]
    )
    assert result.exit_code == 0, result.stdout
    # New tape replaces the old one.
    tape = _load(dest / "tape.yaml")
    assert tape["jtbd"] == "Write good CLI prose."


def test_replay_does_not_copy_synthesis_or_working(tmp_path: Path) -> None:
    src = tmp_path / "src"
    _write_source_tape(src)
    # Mark the source synthesis with content the new project must NOT have.
    (src / "synthesis.md").write_text(
        "# Synthesis from original\n\nSpecific text only present in the source.\n"
    )

    result = runner.invoke(app, ["replay", str(src)])
    assert result.exit_code == 0, result.stdout

    dest = tmp_path / "src-replay"
    syn = (dest / "synthesis.md").read_text(encoding="utf-8")
    assert "Specific text only present in the source" not in syn

    # Working artifacts from the source must not appear in the destination.
    src_working_file = src / "working" / "01-jtbd-and-knowledge-map.md"
    src_content = src_working_file.read_text(encoding="utf-8")
    dest_working_file = dest / "working" / "01-jtbd-and-knowledge-map.md"
    if dest_working_file.exists():
        # Scaffolding may produce a placeholder file; it must be a placeholder,
        # not a copy of the source's content.
        assert dest_working_file.read_text(encoding="utf-8") != src_content


@pytest.mark.parametrize(
    "name,expected_dirname",
    [
        ("v2-attempt", "v2-attempt"),
        ("cli-writer-iteration-2", "cli-writer-iteration-2"),
    ],
)
def test_replay_honors_name_override(
    tmp_path: Path, name: str, expected_dirname: str
) -> None:
    src = tmp_path / "src"
    _write_source_tape(src)

    result = runner.invoke(app, ["replay", str(src), "--name", name])
    assert result.exit_code == 0, result.stdout
    assert (tmp_path / expected_dirname).exists()
