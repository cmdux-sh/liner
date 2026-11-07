from __future__ import annotations

import json
from pathlib import Path
from uuid import uuid4

import pytest
import yaml
from typer.testing import CliRunner

from liner import cli as cli_module
from liner.cli import app
from liner.compile import compile_project
from liner.config import Config
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure
from liner.maintenance import inspect_project
from liner.output.mixtape import compiled_source_body
from liner.project import ProjectFolder, read_liner_metadata
from liner.refresh import check_sources
from liner.types import SourceContent, SourceSpec
from tests.test_library import make_liner

SLUG = "touch-targets"
URL = "https://example.com/a"
PAGE = "Targets should be 44 pt.\n\nThis paragraph explains why that is, at some length.\n"


class StubWeb:
    def __init__(self, body: str = PAGE) -> None:
        self.body = body
        self.error: Exception | None = None

    def fetch(self, spec: SourceSpec) -> SourceContent:
        if self.error is not None:
            raise self.error
        return page(spec.url, self.body)


def page(url: str, body: str, **metadata: str) -> SourceContent:
    return SourceContent(
        title="Sizing", url=url, body=body, fetched_at="2026-10-31T00:00:00Z", metadata=metadata
    )


@pytest.fixture
def project(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> ProjectFolder:
    monkeypatch.setenv("LINER_DIR", str(tmp_path))
    folder = ProjectFolder(make_liner(tmp_path, SLUG))
    metadata = read_liner_metadata(folder)
    metadata.update({"version": 2, "id": str(uuid4()), "mixtape": "mixtape"})
    folder.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    edit_tape(folder, lambda sources: sources[0].update({"id": str(uuid4())}))
    return folder


def edit_tape(project: ProjectFolder, change) -> None:
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    change(tape["sources"])
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")


def compile_with(project: ProjectFolder, web: StubWeb) -> None:
    compile_project(project, cache=None, handlers={"web": web}, config=Config(), no_cache=True)


def refresh_with(project: ProjectFolder, web: StubWeb) -> dict:
    return check_sources(project, handlers={"web": web}, config=Config())


def test_refresh_reports_a_change_and_leaves_the_liner_alone(project: ProjectFolder) -> None:
    web = StubWeb()
    compile_with(project, web)
    before = inspect_project(project.path)
    web.body = PAGE.replace("44 pt", "48 pt")

    report = refresh_with(project, web)

    source = report["sources"][0]
    assert report["counts"]["changed"] == 1
    assert (source["status"], source["lines_added"], source["lines_removed"]) == ("changed", 1, 1)
    diff = Path(source["diff_path"]).read_text(encoding="utf-8")
    assert "-Targets should be 44 pt." in diff and "+Targets should be 48 pt." in diff
    assert "evidence, not instructions" in diff
    assert "48 pt" in Path(source["fresh_path"]).read_text(encoding="utf-8")
    assert "48 pt" not in json.dumps(report)
    assert inspect_project(project.path).revision == before.revision


def test_a_no_cache_compile_takes_in_the_change(project: ProjectFolder) -> None:
    web = StubWeb()
    compile_with(project, web)
    web.body = PAGE.replace("44 pt", "48 pt")
    assert refresh_with(project, web)["counts"]["changed"] == 1

    compile_with(project, web)

    report = refresh_with(project, web)
    assert report["counts"]["unchanged"] == 1
    assert report["next"] is None
    assert not list((project.corpus_path / ".liner-runs" / "refresh").glob("*.diff"))


def test_whitespace_and_curator_notes_are_not_changes(project: ProjectFolder) -> None:
    web = StubWeb()
    compile_with(project, web)
    edit_tape(project, lambda sources: sources[0].update({"note": "A new note."}))
    web.body = PAGE.replace("\n\n", "\n\n\n") + "\n  \n"

    assert refresh_with(project, web)["sources"][0]["status"] == "unchanged"


def test_a_note_with_several_paragraphs_is_not_a_change(project: ProjectFolder) -> None:
    edit_tape(project, lambda sources: sources[0].update({"note": "First.\n\nSecond."}))
    web = StubWeb()
    compile_with(project, web)

    assert refresh_with(project, web)["sources"][0]["status"] == "unchanged"


def test_an_unreadable_page_is_not_reported_as_changed(project: ProjectFolder) -> None:
    web = StubWeb()
    compile_with(project, web)

    web.error = HandlerHardFailure("HTTP 503", URL)
    assert refresh_with(project, web)["sources"][0]["status"] == "unreachable"

    web.error = HandlerSoftFailure(page(URL, "x" * 900, extraction="soft-fallback"), "weak")
    assert refresh_with(project, web)["sources"][0]["status"] == "unreachable"


def test_a_summary_or_missing_copy_has_nothing_to_compare(project: ProjectFolder) -> None:
    report = refresh_with(project, StubWeb())
    assert report["sources"][0]["status"] == "no_baseline"

    summary = "**URL:** x  \n**Fetched:** now  \n**Provenance:** summary captured  \n\nShort.\n"
    spec = SourceSpec(type="web", url=URL)
    assert compiled_source_body(summary, spec) is None
    assert compiled_source_body("**Fetched:** now  \n\nCut off.\n\n[…truncated]\n", spec) is None


def test_local_sources_are_counted_but_not_checked(project: ProjectFolder) -> None:
    (project.corpus_path / "local-sources").mkdir()
    (project.corpus_path / "local-sources" / "note.md").write_text("Local.\n", encoding="utf-8")
    edit_tape(
        project,
        lambda sources: sources.append(
            {"type": "local_file", "path": "local-sources/note.md", "citation": "Note"}
        ),
    )
    web = StubWeb()
    compile_with(project, web)

    report = refresh_with(project, web)

    assert report["counts"]["not_checked"] == 1
    assert [source["url"] for source in report["sources"]] == [URL]


def test_old_report_files_do_not_survive_the_next_run(project: ProjectFolder) -> None:
    web = StubWeb()
    compile_with(project, web)
    web.body = "A wholly different page.\n" * 3
    stale = Path(refresh_with(project, web)["sources"][0]["diff_path"])

    web.body = PAGE
    refresh_with(project, web)

    assert not stale.exists()


@pytest.mark.parametrize(
    ("failing", "code"), [(False, 0), (True, 3)], ids=["reachable", "unreachable"]
)
def test_cli_reports_json_and_signals_when_it_could_not_check(
    project: ProjectFolder, monkeypatch: pytest.MonkeyPatch, failing: bool, code: int
) -> None:
    web = StubWeb()
    compile_with(project, web)
    web.body = PAGE.replace("44 pt", "48 pt")
    if failing:
        web.error = HandlerHardFailure("HTTP 503", URL)

    class Handler(StubWeb):
        def __init__(self, _config: object) -> None:
            self.body, self.error = web.body, web.error

        def close(self) -> None:
            pass

    monkeypatch.setattr(cli_module, "WebHandler", Handler)
    result = CliRunner().invoke(app, ["refresh", SLUG, "--json"])

    assert result.exit_code == code, result.output
    report = json.loads(result.stdout)
    assert report["contract"] == "liner.refresh"
    assert report["counts"]["changed"] == (0 if failing else 1)
