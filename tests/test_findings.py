from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path
from uuid import uuid4

import pytest
import yaml
from typer.testing import CliRunner

from liner import findings as findings_module
from liner.cli import app
from liner.compile import compile_project
from liner.config import Config
from liner.findings import (
    FINDINGS_PATH,
    FindingError,
    accept_finding,
    add_finding,
    list_findings,
    reject_finding,
)
from liner.gaps import add_gap, list_gaps
from liner.library import build_catalog, search_library
from liner.maintenance import apply_change_set, inspect_project, plan_synthesis_review
from liner.project import ProjectFolder, read_liner_metadata
from liner.types import SourceContent, SourceSpec
from tests.test_library import make_liner

SLUG = "touch-targets"
TEXT = "Gloved users need at least 60 pt targets."


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


def tape_sources(project: ProjectFolder) -> list[dict[str, object]]:
    return yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))["sources"]


def test_a_finding_waits_without_changing_the_liner(project: ProjectFolder) -> None:
    sources = tape_sources(project)
    revision = inspect_project(project.path).lifecycle

    finding = add_finding(project, TEXT, evidence="Field test, ward 4")

    assert finding["status"] == "pending"
    assert tape_sources(project) == sources
    assert inspect_project(project.path).lifecycle == revision
    assert list_findings(project)["findings"] == [finding]


def test_the_same_pending_finding_is_recorded_once(project: ProjectFolder) -> None:
    assert add_finding(project, TEXT)["id"] == add_finding(project, TEXT)["id"]


@pytest.mark.parametrize(("text", "evidence"), [("  ", ""), ("x" * 2001, ""), ("fine", "y" * 501)])
def test_empty_or_huge_findings_are_refused(
    project: ProjectFolder, text: str, evidence: str
) -> None:
    with pytest.raises(FindingError):
        add_finding(project, text, evidence=evidence)


def test_a_finding_can_only_close_an_open_gap(project: ProjectFolder) -> None:
    with pytest.raises(FindingError, match="not an open gap"):
        add_finding(project, TEXT, closes="missing")


def test_accepting_adds_a_source_and_starts_a_refresh(project: ProjectFolder) -> None:
    finding = add_finding(project, TEXT, evidence="Field test, ward 4")
    before = len(tape_sources(project))

    result = accept_finding(project, finding["id"])

    sources = tape_sources(project)
    assert len(sources) == before + 1
    added = sources[-1]
    assert added["type"] == "local_file"
    assert added["section"] == "Findings from real work"
    assert added["id"]
    note = (project.corpus_path / str(added["path"])).read_text(encoding="utf-8")
    assert TEXT in note
    assert "Field test, ward 4" in note
    assert "review the synthesis" in result["next"]
    assert inspect_project(project.path).lifecycle["refresh"]["state"] == "required"
    assert result["receipt_id"]
    assert list_findings(project)["findings"] == []


def test_accepting_leaves_operating_files_alone(project: ProjectFolder) -> None:
    project.liner_path.write_text("# Reviewed\n", encoding="utf-8")
    finding = add_finding(project, TEXT)

    accept_finding(project, finding["id"])

    assert project.liner_path.read_text(encoding="utf-8") == "# Reviewed\n"


def test_accepting_closes_the_linked_gap(project: ProjectFolder) -> None:
    gap = add_gap(project, "What size for gloves?")
    finding = add_finding(project, TEXT, closes=gap["id"])

    result = accept_finding(project, finding["id"])

    assert result["closed_gap"] == gap["id"]
    assert list_gaps(project)["gaps"] == []


def test_rejecting_keeps_the_record_and_the_gap(project: ProjectFolder) -> None:
    gap = add_gap(project, "What size for gloves?")
    finding = add_finding(project, TEXT, closes=gap["id"])
    sources = tape_sources(project)

    rejected = reject_finding(project, finding["id"])

    assert rejected["status"] == "rejected"
    assert tape_sources(project) == sources
    assert len(list_gaps(project)["gaps"]) == 1
    [kept] = list_findings(project, include_reviewed=True)["findings"]
    assert kept["id"] == finding["id"]


def test_a_finding_is_reviewed_only_once(project: ProjectFolder) -> None:
    finding = add_finding(project, TEXT)
    accept_finding(project, finding["id"])

    with pytest.raises(FindingError, match="already accepted"):
        reject_finding(project, finding["id"])
    with pytest.raises(FindingError, match="No finding"):
        accept_finding(project, "missing")


def test_a_rejected_finding_is_not_raised_again(project: ProjectFolder) -> None:
    finding = add_finding(project, TEXT)
    reject_finding(project, finding["id"])

    again = add_finding(project, TEXT)

    assert again["id"] == finding["id"]
    assert again["status"] == "rejected"
    assert list_findings(project)["findings"] == []


def test_a_retry_after_a_crash_does_not_add_the_source_twice(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    finding = add_finding(project, TEXT)
    real_write = findings_module.write_yaml

    def crash(*_: object) -> None:
        raise OSError("crashed before recording")

    monkeypatch.setattr("liner.findings.write_yaml", crash)
    with pytest.raises(OSError):
        accept_finding(project, finding["id"])
    monkeypatch.setattr("liner.findings.write_yaml", real_write)
    count = len(tape_sources(project))

    result = accept_finding(project, finding["id"])

    assert result["status"] == "accepted"
    assert len(tape_sources(project)) == count


def test_a_write_failure_is_a_clean_error(project: ProjectFolder) -> None:
    finding = add_finding(project, TEXT)
    (project.corpus_path / "local-sources").mkdir(exist_ok=True)
    (project.corpus_path / "local-sources/findings").write_text("a file\n", encoding="utf-8")

    with pytest.raises(FindingError, match="Core could not add"):
        accept_finding(project, finding["id"])

    assert list_findings(project)["findings"][0]["status"] == "pending"


def test_an_entry_missing_fields_is_refused(project: ProjectFolder) -> None:
    path = project.corpus_path / FINDINGS_PATH
    path.parent.mkdir(exist_ok=True)
    entry = {"id": "a", "finding": "x", "evidence": "", "status": "pending"}
    path.write_text(yaml.safe_dump({"findings": [entry]}), encoding="utf-8")

    with pytest.raises(FindingError, match="not a findings log"):
        accept_finding(project, "a")


def test_a_failed_accept_leaves_no_note_behind(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch
) -> None:
    finding = add_finding(project, TEXT)
    sources = tape_sources(project)

    def refuse(*_: object, **__: object) -> None:
        from liner.maintenance import ProjectInspectionError

        raise ProjectInspectionError("Project changed.")

    monkeypatch.setattr("liner.findings.plan_source_add", refuse)
    with pytest.raises(FindingError, match="Project changed"):
        accept_finding(project, finding["id"])

    assert tape_sources(project) == sources
    assert not list((project.corpus_path / "local-sources/findings").glob("*.md"))
    assert list_findings(project)["findings"][0]["status"] == "pending"


def test_a_damaged_log_is_never_overwritten(project: ProjectFolder) -> None:
    path = project.corpus_path / FINDINGS_PATH
    path.parent.mkdir(exist_ok=True)
    path.write_text("findings: nope\n", encoding="utf-8")

    with pytest.raises(FindingError):
        add_finding(project, TEXT)

    assert path.read_text(encoding="utf-8") == "findings: nope\n"
    [entry] = build_catalog(project.path.parent)["liners"]
    assert entry["pending_findings"] is None


def test_the_catalog_counts_pending_findings(project: ProjectFolder) -> None:
    add_finding(project, TEXT)
    reject_finding(project, add_finding(project, "Declined.")["id"])

    [entry] = build_catalog(project.path.parent)["liners"]

    assert entry["pending_findings"] == 1


def test_cli_requires_yes_to_review(project: ProjectFolder) -> None:
    runner = CliRunner()
    added = runner.invoke(app, ["findings", "add", SLUG, TEXT, "--json"])
    finding_id = json.loads(added.stdout)["id"]
    sources = tape_sources(project)

    refused = runner.invoke(app, ["findings", "accept", SLUG, finding_id])
    assert refused.exit_code == 2
    assert tape_sources(project) == sources

    listed = runner.invoke(app, ["findings", "list", SLUG, "--json"])
    assert json.loads(listed.stdout)["contract"] == "liner.findings"

    accepted = runner.invoke(app, ["findings", "accept", SLUG, finding_id, "--yes", "--json"])
    assert json.loads(accepted.stdout)["status"] == "accepted"
    assert runner.invoke(app, ["findings", "reject", SLUG, finding_id, "--yes"]).exit_code == 1


class StubHandler:
    def fetch(self, spec: SourceSpec) -> SourceContent:
        return SourceContent(
            title="Guide",
            url=spec.url or "",
            body="Guide body",
            fetched_at=datetime.now(UTC).isoformat(),
        )


def test_an_accepted_finding_reaches_the_compiled_liner(project: ProjectFolder) -> None:
    accept_finding(project, add_finding(project, TEXT)["id"])

    review = plan_synthesis_review(project.path, "still_current")
    apply_change_set(project.path, review, approved=True)
    handlers = {"web": StubHandler(), "youtube": StubHandler()}
    compile_project(project, cache=None, handlers=handlers, config=Config())

    assert TEXT in "".join(
        path.read_text(encoding="utf-8") for path in (project.corpus_path / "sources").glob("*.md")
    )
    results = search_library("gloved", root=project.path.parent)["results"]
    assert any("sources/" in result["path"] for result in results)
