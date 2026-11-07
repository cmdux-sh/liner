from __future__ import annotations

import json
import re
from pathlib import Path
from uuid import uuid4

import typer.main
import yaml

from liner.cli import app
from liner.compile import compile_project
from liner.config import Config
from liner.findings import add_finding
from liner.gaps import add_gap
from liner.library_view import build_library_view
from liner.project import ProjectFolder, read_liner_metadata
from liner.refresh import check_sources
from liner.research import STATE_FILENAME
from tests.test_library import make_liner
from tests.test_refresh import PAGE, StubWeb

URL = "https://example.com/a"


def liner(root: Path, slug: str) -> ProjectFolder:
    folder = ProjectFolder(make_liner(root, slug))
    metadata = read_liner_metadata(folder)
    metadata.update({"version": 2, "id": str(uuid4()), "mixtape": "mixtape"})
    folder.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    tape = yaml.safe_load(folder.tape_path.read_text(encoding="utf-8"))
    tape["sources"][0]["id"] = str(uuid4())
    folder.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    return folder


def view_of(root: Path, slug: str = "a") -> dict:
    return next(item for item in build_library_view(root)["liners"] if item["slug"] == slug)


def test_a_ready_liner_reports_what_it_holds(tmp_path: Path) -> None:
    liner(tmp_path, "a")

    item = view_of(tmp_path)

    assert item["state"] == "ready"
    assert item["sources"][0]["locator"] == URL
    assert item["sections"] == [{"name": "Other", "sources": 1}]
    assert item["synthesis"]["lines"] == 3


def test_attention_collects_what_needs_the_person(tmp_path: Path) -> None:
    folder = liner(tmp_path, "a")
    add_gap(folder, "Is this right?")
    add_finding(folder, "A finding.")

    attention = build_library_view(tmp_path)["attention"]
    titles = {item["title"]: item for item in attention}

    assert "1 finding to accept or reject" in titles
    assert "1 question it couldn't answer" in titles
    assert "Never tested" in titles


def test_a_saved_refresh_change_lasts_until_a_compile_takes_it_in(tmp_path: Path) -> None:
    folder = liner(tmp_path, "a")
    web = StubWeb()
    compile_project(folder, cache=None, handlers={"web": web}, config=Config(), no_cache=True)
    web.body = PAGE.replace("44 pt", "48 pt")
    check_sources(folder, handlers={"web": web}, config=Config())

    assert view_of(tmp_path)["state"] == "sources moved"
    assert view_of(tmp_path)["refreshed_at"]

    compile_project(folder, cache=None, handlers={"web": web}, config=Config(), no_cache=True)

    assert view_of(tmp_path)["state"] == "ready"


def test_a_shared_source_merges_and_a_citation_shows(tmp_path: Path) -> None:
    liner(tmp_path, "a")
    other = liner(tmp_path, "b")
    tape = yaml.safe_load(other.tape_path.read_text(encoding="utf-8"))
    tape["sources"].append({"type": "liner", "url": "liner://a", "id": str(uuid4())})
    other.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    view = build_library_view(tmp_path)

    assert view["sources"][0]["liners"] == ["a", "b"]
    assert view["connections"]["citations"][0]["from"] == "b"
    assert view["connections"]["citations"][0]["to"] == "a"


def test_a_build_in_progress_is_building(tmp_path: Path) -> None:
    folder = liner(tmp_path, "a")
    state = {"contract": "liner.research_state", "version": 1, "completed": [], "approvals": {}}
    (folder.corpus_path / STATE_FILENAME).write_text(json.dumps(state), encoding="utf-8")

    assert view_of(tmp_path)["state"] == "building"


def test_the_npm_launcher_forwards_every_core_command() -> None:
    launcher = Path("packages/tui/bin/liner.js").read_text(encoding="utf-8")
    listed = set(re.findall(r'name: "([\w-]+)",?\s*(?:description: "[^"]*",\s*)?', launcher))

    assert set(typer.main.get_command(app).commands) <= listed
