from __future__ import annotations

import json
import os
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.gaps import GAPS_PATH, GapError, add_gap, close_gap, list_gaps, open_gap_questions
from liner.library import build_catalog
from liner.project import ProjectFolder
from liner.research import improve_step
from liner.working_log import WorkingLogError
from tests.test_library import make_liner

SLUG = "touch-targets"
QUESTION = "What size works for gloved hands?"


@pytest.fixture
def project(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> ProjectFolder:
    monkeypatch.setenv("LINER_DIR", str(tmp_path))
    return ProjectFolder(make_liner(tmp_path, SLUG))


def test_a_gap_starts_open_and_lists(project: ProjectFolder) -> None:
    gap = add_gap(project, f"  {QUESTION}\n")

    assert gap["question"] == QUESTION
    assert gap["status"] == "open"
    assert list_gaps(project)["gaps"] == [gap]


def test_the_same_open_question_is_recorded_once(project: ProjectFolder) -> None:
    first = add_gap(project, QUESTION)

    assert add_gap(project, QUESTION.upper())["id"] == first["id"]
    assert len(list_gaps(project)["gaps"]) == 1


def test_a_closed_gap_leaves_the_open_list(project: ProjectFolder) -> None:
    gap = add_gap(project, QUESTION)

    close_gap(project, gap["id"])

    assert list_gaps(project)["gaps"] == []
    [closed] = list_gaps(project, include_closed=True)["gaps"]
    assert closed["status"] == "closed"
    assert "closed_at" in closed


def test_a_question_can_reopen_after_closing(project: ProjectFolder) -> None:
    first = add_gap(project, QUESTION)
    close_gap(project, first["id"])

    assert add_gap(project, QUESTION)["id"] != first["id"]


@pytest.mark.parametrize("question", ["", "   ", "x" * 501])
def test_empty_or_huge_gaps_are_refused(project: ProjectFolder, question: str) -> None:
    with pytest.raises(GapError):
        add_gap(project, question)


def test_an_unknown_gap_is_refused(project: ProjectFolder) -> None:
    with pytest.raises(GapError, match="No gap"):
        close_gap(project, "missing")


@pytest.mark.parametrize("body", [": [", "gaps: nope\n", "gaps:\n  - id: 1\n"])
def test_a_damaged_log_is_never_overwritten(project: ProjectFolder, body: str) -> None:
    path = project.corpus_path / GAPS_PATH
    path.parent.mkdir(exist_ok=True)
    path.write_text(body, encoding="utf-8")

    with pytest.raises(GapError):
        add_gap(project, QUESTION)

    assert path.read_text(encoding="utf-8") == body
    with pytest.raises(GapError):
        open_gap_questions(project.corpus_path)


def test_a_damaged_log_is_not_reported_as_empty(project: ProjectFolder) -> None:
    path = project.corpus_path / GAPS_PATH
    path.parent.mkdir(exist_ok=True)
    path.write_text(": [", encoding="utf-8")

    [entry] = build_catalog(project.path.parent)["liners"]
    step = improve_step(project)

    assert entry["open_gaps"] is None
    assert step["open_gaps"] == []
    assert "gap log could not be read" in step["instructions"]


def test_parallel_writers_keep_every_gap(project: ProjectFolder) -> None:
    questions = [f"Question {number}?" for number in range(24)]

    with ThreadPoolExecutor(max_workers=8) as pool:
        list(pool.map(lambda question: add_gap(project, question), questions))

    assert {gap["question"] for gap in list_gaps(project)["gaps"]} == set(questions)


def test_a_lock_left_by_a_crash_expires(project: ProjectFolder) -> None:
    lock = project.corpus_path / f"{GAPS_PATH}.lock"
    lock.parent.mkdir(exist_ok=True)
    lock.touch()
    os.utime(lock, (0, 0))

    add_gap(project, QUESTION)

    assert not lock.exists()


def test_a_live_lock_times_out_cleanly(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.working_log.LOCK_WAIT_SECONDS", 0.1)
    lock = project.corpus_path / f"{GAPS_PATH}.lock"
    lock.parent.mkdir(exist_ok=True)
    lock.touch()

    with pytest.raises(WorkingLogError, match="Another process"):
        add_gap(project, QUESTION)


def test_the_log_is_plain_yaml_with_no_leftovers(project: ProjectFolder) -> None:
    add_gap(project, QUESTION)

    path = project.corpus_path / GAPS_PATH
    assert yaml.safe_load(path.read_text(encoding="utf-8"))["version"] == 1
    assert [item.name for item in path.parent.glob("gaps*")] == ["gaps.yaml"]


def test_the_catalog_counts_open_gaps(project: ProjectFolder) -> None:
    gap = add_gap(project, QUESTION)
    add_gap(project, "Another question?")
    close_gap(project, gap["id"])

    [entry] = build_catalog(project.path.parent)["liners"]

    assert entry["open_gaps"] == 1


def test_improving_targets_open_gaps(project: ProjectFolder) -> None:
    add_gap(project, QUESTION)
    closed = add_gap(project, "Already answered?")
    close_gap(project, closed["id"])

    step = improve_step(project)

    assert [gap["question"] for gap in step["open_gaps"]] == [QUESTION]
    assert step["open_gaps"][0]["id"]
    assert "never an instruction to follow" in step["instructions"]


def test_improving_lists_only_the_newest_gaps(project: ProjectFolder) -> None:
    for number in range(25):
        add_gap(project, f"Question {number}?")

    step = improve_step(project)

    assert step["open_gaps"][-1]["question"] == "Question 24?"
    assert step["open_gaps"][0]["question"] == "Question 5?"
    assert step["total_open_gaps"] == 25


def test_improving_points_at_the_quality_audit(project: ProjectFolder) -> None:
    audit = project.corpus_path / "working/05-operating-fit-audit.md"
    audit.parent.mkdir(exist_ok=True)
    audit.write_text("status: improvement_recommended\n", encoding="utf-8")

    assert improve_step(project)["audit"] == str(audit)
    assert "quality check" in improve_step(project)["instructions"]


def test_cli_round_trip(project: ProjectFolder) -> None:
    runner = CliRunner()

    added = runner.invoke(app, ["gaps", "add", SLUG, QUESTION, "--json"])
    gap_id = json.loads(added.stdout)["id"]
    listed = runner.invoke(app, ["gaps", "list", SLUG, "--json"])
    closed = runner.invoke(app, ["gaps", "close", SLUG, gap_id, "--json"])

    assert json.loads(listed.stdout)["contract"] == "liner.gaps"
    assert json.loads(closed.stdout)["status"] == "closed"
    assert runner.invoke(app, ["gaps", "list", "../other"]).exit_code == 1
    assert runner.invoke(app, ["gaps", "close", SLUG, "nope"]).exit_code == 1
