from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

import pytest
from typer.testing import CliRunner

from liner.cli import _source_output_summary, app
from liner.types import CompiledSource, CompileResult, CompileWarning, SourceSpec, Tape

runner = CliRunner()


@pytest.fixture(autouse=True)
def _isolated_home(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """Keep cache and config writes inside tmp_path."""
    monkeypatch.setenv("HOME", str(tmp_path))


def test_init_creates_project_folder(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    result = runner.invoke(app, ["init", str(target)])
    assert result.exit_code == 0, result.output
    assert (target / "tape.yaml").exists()
    assert (target / "synthesis.md").exists()
    assert (target / "working" / "01-jtbd-and-knowledge-map.md").exists()


def test_init_refuses_overwrite(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    result = runner.invoke(app, ["init", str(target)])
    assert result.exit_code == 1


def test_compile_requires_synthesis(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    (target / "synthesis.md").unlink()
    result = runner.invoke(app, ["compile", str(target)])
    assert result.exit_code == 1
    assert "synthesis.md" in result.output


def test_compile_fails_without_tape(tmp_path: Path) -> None:
    empty = tmp_path / "empty"
    empty.mkdir()
    result = runner.invoke(app, ["compile", str(empty)])
    assert result.exit_code == 1


def test_list_finds_project_folders(tmp_path: Path) -> None:
    runner.invoke(app, ["init", str(tmp_path / "one")])
    runner.invoke(app, ["init", str(tmp_path / "two")])
    result = runner.invoke(app, ["list", "--dir", str(tmp_path), "--json"])
    assert result.exit_code == 0
    assert "one" in result.stdout
    assert "two" in result.stdout


def test_share_requires_existing_folder(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    result = runner.invoke(app, ["share", str(target)])
    assert result.exit_code == 0
    assert (tmp_path / "demo.mixtape").exists()


def test_import_extracts_and_runs_without_refetch(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    runner.invoke(app, ["share", str(target)])

    archive = tmp_path / "demo.mixtape"
    dest = tmp_path / "extracted"
    result = runner.invoke(app, ["import", str(archive), str(dest), "--no-refetch"])
    assert result.exit_code == 0
    assert (dest / "demo" / "tape.yaml").exists()


def test_version_flag() -> None:
    result = runner.invoke(app, ["--version"])
    assert result.exit_code == 0
    assert "liner" in result.stdout


def test_source_output_summary_names_unavailable_placeholders() -> None:
    result = CompileResult(
        tape=Tape(title="T", description="d", version=1, curator="c", sources=()),
        compiled_at=datetime.now(UTC),
        sources=(
            CompiledSource(spec=SourceSpec(type="web", url="https://ok"), content=None),
            CompiledSource(spec=SourceSpec(type="web", url="https://nope"), content=None),
        ),
        warnings=(CompileWarning(url="https://nope", message="failed"),),
    )

    assert _source_output_summary(result) == ("(0/2 usable sources, 2 unavailable placeholders)")
