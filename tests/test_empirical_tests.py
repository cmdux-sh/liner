from __future__ import annotations

import json
import zipfile
from pathlib import Path
from uuid import uuid4

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.empirical_tests import (
    TESTS_PATH,
    EmpiricalTestError,
    add_test,
    judge_blind_test,
    list_tests,
    start_blind_test,
    summarize_tests,
)
from liner.library import build_catalog
from liner.project import ProjectFolder, read_liner_metadata
from tests.test_library import make_liner

SLUG = "touch-targets"


@pytest.fixture
def project(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> ProjectFolder:
    monkeypatch.setenv("LINER_DIR", str(tmp_path))
    folder = ProjectFolder(make_liner(tmp_path, SLUG))
    metadata = read_liner_metadata(folder)
    metadata.update({"version": 2, "id": str(uuid4()), "mixtape": "mixtape"})
    folder.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    tape = yaml.safe_load(folder.tape_path.read_text(encoding="utf-8"))
    tape["sources"][0]["id"] = str(uuid4())
    folder.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    return folder


def record(project: ProjectFolder, verdict: str = "better") -> dict:
    return add_test(
        project, "How big?", without_answer="Big.", with_answer="44 pt.", verdict=verdict
    )


def entry(project: ProjectFolder) -> dict:
    return build_catalog(project.path.parent)["liners"][0]


def test_catalog_shows_verdicts_and_when_they_are_out_of_date(project: ProjectFolder) -> None:
    assert entry(project)["test"]["runs"] == 0
    assert entry(project)["test"]["current"] is False

    record(project)
    record(project, "same")

    result = entry(project)["test"]
    assert (result["runs"], result["better"], result["same"], result["worse"]) == (2, 1, 1, 0)
    assert result["current"] is True

    project.synthesis_path.write_text("# Synthesis\n\nChanged.\n", encoding="utf-8")
    assert entry(project)["test"]["current"] is False


def test_a_liner_of_unknown_revision_is_never_current(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("liner.empirical_tests._revision", lambda _: None)
    record(project)

    assert summarize_tests(project)["current"] is False


def test_answers_stay_out_of_the_listing(project: ProjectFolder) -> None:
    record(project)

    listing = list_tests(project)

    assert "44 pt." not in json.dumps(listing)
    assert "44 pt." in Path(listing["path"]).read_text(encoding="utf-8")


@pytest.mark.parametrize(
    ("question", "answer", "verdict"),
    [("Q", "A", "great"), (" ", "A", "better"), ("Q", " ", "better"), ("Q", "A" * 20001, "better")],
)
def test_bad_input_is_refused(
    project: ProjectFolder, question: str, answer: str, verdict: str
) -> None:
    with pytest.raises(EmpiricalTestError):
        add_test(project, question, without_answer="A", with_answer=answer, verdict=verdict)


def test_an_uncompiled_liner_cannot_be_tested(project: ProjectFolder) -> None:
    project.mixtape_path.unlink()
    with pytest.raises(EmpiricalTestError, match="no compiled liner"):
        record(project)


def test_a_damaged_log_reads_as_unknown_and_blocks_writes(project: ProjectFolder) -> None:
    log = project.corpus_path / TESTS_PATH
    log.parent.mkdir(parents=True, exist_ok=True)
    log.write_text("tests: nonsense\n", encoding="utf-8")

    assert entry(project)["test"] is None
    with pytest.raises(EmpiricalTestError):
        record(project)
    assert log.read_text(encoding="utf-8") == "tests: nonsense\n"
    with pytest.raises(EmpiricalTestError):
        summarize_tests(project)


def test_cli_records_and_lists(project: ProjectFolder, tmp_path: Path) -> None:
    without = tmp_path / "without.txt"
    with_liner = tmp_path / "with.txt"
    without.write_text("Big.", encoding="utf-8")
    with_liner.write_text("44 pt.", encoding="utf-8")
    runner = CliRunner()

    added = runner.invoke(
        app,
        [
            "test", "add", SLUG, "How big?",
            "--without-file", str(without), "--with-file", str(with_liner),
            "--verdict", "better", "--json",
        ],
    )  # fmt: skip
    listed = runner.invoke(app, ["test", "list", SLUG, "--json"])
    catalog = runner.invoke(app, ["catalog"])

    assert added.exit_code == 0, added.output
    assert json.loads(listed.stdout)["summary"]["better"] == 1
    assert "better in 1 of 1 tests" in catalog.stdout
    missing = runner.invoke(
        app,
        [
            "test", "add", SLUG, "Q", "--without-file", "/nonexistent",
            "--with-file", str(with_liner), "--verdict", "better",
        ],
    )  # fmt: skip
    assert missing.exit_code == 1


def blind(project: ProjectFolder) -> dict:
    return start_blind_test(
        project, "How big?", without_answer="Big.", with_answer="44 pt."
    )


def test_a_blind_sheet_shows_no_labels(project: ProjectFolder) -> None:
    sheet = Path(blind(project)["path"]).read_text(encoding="utf-8")
    assert "Answer A" in sheet and "Answer B" in sheet
    assert "with" not in sheet.lower()


@pytest.mark.parametrize("with_is", ["a", "b"])
def test_a_blind_choice_becomes_the_right_verdict(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, with_is: str
) -> None:
    monkeypatch.setattr("liner.empirical_tests.secrets.choice", lambda _: with_is)
    started = blind(project)
    other = "b" if with_is == "a" else "a"
    assert judge_blind_test(project, started["id"], with_is)["verdict"] == "better"
    second = blind(project)
    result = judge_blind_test(project, second["id"], other)
    assert result["verdict"] == "worse"
    assert result["with_liner_was"] == with_is.upper()
    assert result["blind"] is True
    assert not Path(second["path"]).exists()
    summary = entry(project)["test"]
    assert (summary["runs"], summary["blind"], summary["better"], summary["worse"]) == (2, 2, 1, 1)


@pytest.mark.parametrize(
    "tell",
    [
        "See https://developer.apple.com/app-store/product-page.",
        "Apple caps it at 30 characters.[^appinfo]",
        "**Sources:** Apple's App information page.",
        "## References\n\nApple's App information page.",
        "Per the liner, keep it short.",
    ],
)
def test_an_answer_that_shows_its_sources_is_refused(project: ProjectFolder, tell: str) -> None:
    with pytest.raises(EmpiricalTestError, match="show which answer is which"):
        start_blind_test(
            project, "How big?", without_answer="Big.", with_answer=f"44 pt.\n\n{tell}"
        )
    assert not (project.corpus_path / "working" / "blind-tests").exists()


def test_an_answer_that_only_talks_about_sources_is_kept(project: ProjectFolder) -> None:
    answer = "Sources of truth differ: drink 8 glasses a day."
    started = start_blind_test(project, "How much?", without_answer=answer, with_answer="2 L.")
    assert Path(started["path"]).exists()


def test_order_varies(project: ProjectFolder) -> None:
    seen = set()
    for _ in range(40):
        test = blind(project)
        sheet = Path(test["path"]).read_text(encoding="utf-8")
        seen.add(sheet.index("Big.") < sheet.index("44 pt."))
    assert seen == {True, False}


def test_same_is_same_and_a_test_is_judged_once(project: ProjectFolder) -> None:
    started = blind(project)
    assert judge_blind_test(project, started["id"], "same")["verdict"] == "same"
    with pytest.raises(EmpiricalTestError, match="No blind test"):
        judge_blind_test(project, started["id"], "a")


def test_a_bad_choice_is_refused(project: ProjectFolder) -> None:
    started = blind(project)
    with pytest.raises(EmpiricalTestError, match="must be one of"):
        judge_blind_test(project, started["id"], "better")


def test_blind_commands_work_from_the_cli(project: ProjectFolder, tmp_path: Path) -> None:
    (tmp_path / "w.txt").write_text("44 pt.", encoding="utf-8")
    (tmp_path / "o.txt").write_text("Big.", encoding="utf-8")
    runner = CliRunner()
    started = runner.invoke(
        app,
        [
            "test", "start", SLUG, "How big?",
            "--without-file", str(tmp_path / "o.txt"),
            "--with-file", str(tmp_path / "w.txt"),
            "--json",
        ],
    )
    assert started.exit_code == 0, started.output
    test_id = json.loads(started.output)["id"]
    judged = runner.invoke(app, ["test", "judge", SLUG, test_id, "--preferred", "same"])
    assert judged.exit_code == 0, judged.output
    assert "same" in judged.output


def test_a_failed_judgment_leaves_the_test_waiting(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    started = blind(project)

    def fail(*_: object, **__: object) -> None:
        raise EmpiricalTestError("log is damaged")

    monkeypatch.setattr("liner.empirical_tests._record", fail)
    with pytest.raises(EmpiricalTestError, match="damaged"):
        judge_blind_test(project, started["id"], "a")
    monkeypatch.undo()
    assert Path(started["path"]).exists()
    assert judge_blind_test(project, started["id"], "same")["id"] == started["id"]


def test_mcp_and_share_keep_the_answer_key_back(project: ProjectFolder) -> None:
    from liner.mcp_server import ToolError, read_file
    from liner.share import pack

    blind(project)
    relative = (project.corpus_path / "working/empirical-tests-pending.yaml").relative_to(
        project.path
    )
    with pytest.raises(ToolError, match="answer key"):
        read_file(project.path.parent, SLUG, str(relative), None)
    with zipfile.ZipFile(pack(project).archive_path) as archive:
        assert not [name for name in archive.namelist() if "pending" in name]


def test_judging_in_plain_text_reveals_the_answer(
    project: ProjectFolder, tmp_path: Path
) -> None:
    started = blind(project)
    result = CliRunner().invoke(app, ["test", "judge", SLUG, started["id"], "--preferred", "a"])
    assert "used the liner was" in result.output
