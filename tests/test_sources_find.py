from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.library import LibraryError, find_sources

URL = "https://example.org/portfolio-guide"
VIDEO = "https://www.youtube.com/watch?v=DUejFPtYll0"


@pytest.fixture
def library(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    monkeypatch.setenv("LINER_DIR", str(tmp_path))
    result = CliRunner().invoke(app, ["research", "start", "Write case studies", "--title", "Case"])
    assert result.exit_code == 0, result.output
    corpus = tmp_path / "case" / "mixtape"
    tape = yaml.safe_load((corpus / "tape.yaml").read_text(encoding="utf-8"))
    tape["sources"] = [
        {"type": "web", "url": URL, "section": "core"},
        {
            "type": "local_file",
            "path": "local-sources/yt.md",
            "citation": f"Staff portfolio talk. {VIDEO}",
            "section": "core",
        },
    ]
    (corpus / "tape.yaml").write_text(yaml.safe_dump(tape), encoding="utf-8")
    candidates = [
        {"url": URL, "title": "Portfolio guide", "decision": "kept", "rationale": "Direct."},
        {"url": VIDEO, "title": "Staff talk", "decision": "kept", "rationale": "A talk."},
        {
            "url": "https://example.org/listicle",
            "title": "Portfolio listicle",
            "decision": "dropped",
            "rationale": "Generic SEO filler.",
        },
    ]
    (corpus / "working" / "03-evaluation.yaml").write_text(
        yaml.safe_dump({"candidates": candidates}), encoding="utf-8"
    )
    return tmp_path


def test_rows_show_the_liner_and_the_earlier_verdict(library: Path) -> None:
    result = find_sources("portfolio", root=library)

    assert result["total"] == 2
    by_title = {row["title"]: row for row in result["sources"]}
    assert by_title["Portfolio guide"]["liner"] == "case"
    assert by_title["Portfolio listicle"]["decision"] == "dropped"
    assert by_title["Portfolio listicle"]["reason"] == "Generic SEO filler."


def test_a_local_source_matches_its_video_once(library: Path) -> None:
    rows = find_sources("staff talk", root=library)["sources"]

    assert len(rows) == 1
    assert rows[0]["local"] is True
    assert rows[0]["url"] == VIDEO


def test_output_is_capped_but_reports_the_total(library: Path) -> None:
    result = find_sources("portfolio", root=library, limit=1)

    assert len(result["sources"]) == 1
    assert result["total"] == 2


def test_an_address_written_differently_keeps_its_verdict(library: Path) -> None:
    evaluation = library / "case/mixtape/working/03-evaluation.yaml"
    data = yaml.safe_load(evaluation.read_text(encoding="utf-8"))
    data["candidates"][0]["url"] = "http://www.example.org/portfolio-guide/"
    data["candidates"][0]["decision"] = "trim"
    evaluation.write_text(yaml.safe_dump(data), encoding="utf-8")

    rows = find_sources("guide", root=library)["sources"]

    assert [row["decision"] for row in rows] == ["trim"]


def test_a_similar_address_does_not_take_a_verdict(library: Path) -> None:
    tape = library / "case/mixtape/tape.yaml"
    data = yaml.safe_load(tape.read_text(encoding="utf-8"))
    data["sources"][1]["citation"] = f"Other page. {URL}-two"
    tape.write_text(yaml.safe_dump(data), encoding="utf-8")

    rows = {row["url"]: row for row in find_sources("portfolio", root=library)["sources"]}

    assert rows[f"{URL}-two"]["reason"] == ""


def test_results_are_marked_as_evidence(library: Path) -> None:
    assert "evidence" in find_sources("portfolio", root=library)["notice"].lower()


def test_a_broken_tape_is_reported_not_hidden(library: Path) -> None:
    (library / "case/mixtape/tape.yaml").write_text("sources: [", encoding="utf-8")

    assert find_sources("portfolio", root=library)["unreadable"] == ["case"]


def test_a_query_needs_a_word(library: Path) -> None:
    with pytest.raises(LibraryError):
        find_sources("  ", root=library)


def test_cli_returns_versioned_json(library: Path) -> None:
    result = CliRunner().invoke(app, ["find-sources", "portfolio", "--json"])

    assert result.exit_code == 0, result.output
    assert json.loads(result.stdout)["contract"] == "liner.source_list"


def test_the_sources_step_checks_the_library_first() -> None:
    text = Path("src/liner/bundled/research/phases/candidates.md").read_text(encoding="utf-8")

    assert 'liner find-sources "<topic>" --json' in text
