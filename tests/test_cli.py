from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path

import pytest
from typer.testing import CliRunner

from liner.cli import _source_output_summary, app
from liner.project import ProjectFolder, read_liner_metadata
from liner.tape import load_tape
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
    project = ProjectFolder(target)
    assert project.liner_metadata_path.exists()
    assert project.tape_path.exists()
    assert project.synthesis_path.exists()
    assert (project.working_dir / "01-jtbd-and-knowledge-map.md").exists()


def test_init_refuses_overwrite(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    result = runner.invoke(app, ["init", str(target)])
    assert result.exit_code == 1


def test_init_accepts_metadata_flags(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    jtbd = "When creating a project from automation, I want metadata flags, so setup is one command."

    result = runner.invoke(
        app,
        [
            "init",
            str(target),
            "--mode",
            "methodology",
            "--jtbd",
            jtbd,
            "--title",
            "Design Engineering",
            "--description",
            "A working corpus for design engineering.",
            "--curator",
            "Arturo",
        ],
    )

    assert result.exit_code == 0, result.output
    project = ProjectFolder(target)
    tape = load_tape(project.tape_path)
    assert tape.mode == "methodology"
    assert tape.jtbd == jtbd
    assert tape.title == "Design Engineering"
    assert tape.description == "A working corpus for design engineering."
    assert tape.curator == "Arturo"
    working = (project.working_dir / "01-jtbd-and-knowledge-map.md").read_text(encoding="utf-8")
    assert jtbd in working
    assert "TODO — a single specific Job Story" not in working


def test_init_rejects_invalid_mode_before_scaffolding(tmp_path: Path) -> None:
    target = tmp_path / "demo"

    result = runner.invoke(app, ["init", str(target), "--mode", "slow"])

    assert result.exit_code == 1
    assert "--mode must be quick or methodology" in result.output
    assert not target.exists()


def test_compile_requires_synthesis(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    ProjectFolder(target).synthesis_path.unlink()
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


def test_list_keeps_legacy_root_project_after_liner_metadata(tmp_path: Path) -> None:
    project = tmp_path / "legacy"
    project.mkdir()
    (project / "liner.yaml").write_text("artifact: liner\nversion: 2\n", encoding="utf-8")
    (project / "tape.yaml").write_text(
        "title: Legacy\nversion: 1\ncurator: A\ndescription: D\nsources: []\n",
        encoding="utf-8",
    )

    result = runner.invoke(app, ["list", "--dir", str(tmp_path), "--json"])

    assert result.exit_code == 0
    assert "legacy" in result.stdout
    assert "Legacy" in result.stdout


def test_status_json_includes_phase_report(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    project = ProjectFolder(target)
    _write_tape(target, jtbd="When testing status, I want phase JSON, so the TUI can read it.")
    (project.working_dir / "01-jtbd-and-knowledge-map.md").write_text(
        "# JTBD and knowledge map\n\n## Job-to-be-done\n\nSet.\n\n## Knowledge map\n\n- Foundations\n",
        encoding="utf-8",
    )
    (project.corpus_path / ".liner-progress.json").write_text(
        json.dumps({"step": 2, "lastTouched": "2026-06-14T19:00:00Z"}),
        encoding="utf-8",
    )
    (project.corpus_path / ".liner-gates.json").write_text(
        json.dumps({"gate0Accepted": True}),
        encoding="utf-8",
    )
    run_dir = project.corpus_path / ".liner-runs" / "framing"
    run_dir.mkdir(parents=True)
    (run_dir / "run.jsonl").write_text(
        "\n".join(
            [
                json.dumps({"type": "_liner_meta", "agent": "codex", "taskLabel": "framing"}),
                json.dumps({"type": "_liner_close", "exitCode": 0}),
                json.dumps({"type": "result", "subtype": "success", "is_error": False}),
            ]
        )
        + "\n",
        encoding="utf-8",
    )

    result = runner.invoke(app, ["status", str(target), "--json"])

    assert result.exit_code == 0, result.output
    payload = json.loads(result.stdout)
    assert "totals" in payload
    assert payload["progress"]["step"] == 2
    assert payload["progress"]["source"] == "file"
    assert payload["progress"]["current_phase"] == "candidates"
    assert payload["status_snapshot"]["milestone"] == "started"
    assert payload["project_skill"]["status"] == "missing"
    phases = {phase["id"]: phase for phase in payload["phases"]}
    assert phases["framing"]["status"] == "complete"
    assert phases["framing"]["artifact"]["has_real_content"] is True
    assert phases["framing"]["runs"]["count"] == 1
    assert phases["framing"]["runs"]["latest_exit_code"] == 0
    assert phases["gate0"]["status"] == "complete"
    assert phases["gate0"]["gate"]["accepted"] is True
    assert phases["candidates"]["status"] == "in_progress"


def test_status_json_infers_phase_progress_when_cursor_is_missing(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    project = ProjectFolder(target)
    _write_tape(target, jtbd="When opening an older project, I want status inferred, so it is usable.")
    (project.working_dir / "01-jtbd-and-knowledge-map.md").write_text(
        "# JTBD and knowledge map\n\n## Job-to-be-done\n\nSet.\n\n## Knowledge map\n\n- Foundations\n",
        encoding="utf-8",
    )

    result = runner.invoke(app, ["status", str(target), "--json"])

    assert result.exit_code == 0, result.output
    payload = json.loads(result.stdout)
    assert payload["progress"]["source"] == "inferred"
    assert payload["progress"]["step"] == 1
    assert payload["progress"]["current_phase"] == "gate0"
    phases = {phase["id"]: phase for phase in payload["phases"]}
    assert phases["framing"]["status"] == "complete"
    assert phases["gate0"]["status"] == "in_progress"


def test_status_json_no_write_refreshes_without_process_manifest(tmp_path: Path) -> None:
    target = tmp_path / "demo"
    runner.invoke(app, ["init", str(target)])
    project = ProjectFolder(target)
    before = read_liner_metadata(project)
    _write_tape(target, jtbd="When opening from the TUI, I want status without filesystem churn.")
    run_dir = project.corpus_path / ".liner-runs" / "framing"
    run_dir.mkdir(parents=True)
    (run_dir / "run.jsonl").write_text(
        "\n".join(
            [
                json.dumps({"type": "_liner_meta", "agent": "codex", "taskLabel": "framing"}),
                json.dumps({"type": "_liner_close", "exitCode": 0}),
            ]
        )
        + "\n",
        encoding="utf-8",
    )

    result = runner.invoke(app, ["status", str(target), "--json", "--no-write"])

    assert result.exit_code == 0, result.output
    payload = json.loads(result.stdout)
    phases = {phase["id"]: phase for phase in payload["phases"]}
    assert phases["framing"]["runs"]["count"] == 1
    assert not (project.corpus_path / "process.json").exists()
    after = read_liner_metadata(project)
    assert after["status"]["updated"] == before["status"]["updated"]


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
    assert ProjectFolder(dest / "demo").tape_path.exists()


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


def _write_tape(folder: Path, *, jtbd: str) -> None:
    ProjectFolder(folder).tape_path.write_text(
        "\n".join(
            [
                "title: Demo",
                "description: Status fixture",
                "version: 1",
                "curator: Arturo",
                "mode: methodology",
                f"jtbd: {jtbd!r}",
                "sources: []",
                "",
            ]
        ),
        encoding="utf-8",
    )
