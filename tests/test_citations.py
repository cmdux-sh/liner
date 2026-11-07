from __future__ import annotations

import json
import os
from pathlib import Path

import pytest
import yaml

from liner.citations import list_citations, parse_citations
from liner.library_view import build_library_view
from liner.project import ProjectFolder
from liner.research_checks import check_phase
from tests.test_library import make_liner
from tests.test_research_checks import KEPT, URL

SECTIONS = "## Generative rules\n\n## Stances this corpus takes\n"
SOURCE_FILE = f"# Guide\n\n**Source type:** web  \n**URL:** {URL}  \n**Fetched:** now  \n\nBody.\n"


def corpus_with(tmp_path: Path, synthesis: str, *, kept: int = 1) -> Path:
    corpus = tmp_path / "corpus"
    (corpus / "working").mkdir(parents=True)
    candidates = [{**KEPT, "url": f"{URL}/{index}" if index else URL} for index in range(kept)]
    candidates.append({"url": "https://example.com/dropped", "decision": "dropped"})
    (corpus / "working/03-evaluation.yaml").write_text(yaml.safe_dump({"candidates": candidates}))
    (corpus / "tape.yaml").write_text(yaml.safe_dump({"sources": []}))
    (corpus / "synthesis.md").write_text(synthesis, encoding="utf-8")
    return corpus


def failure(corpus: Path) -> str:
    result = check_phase(corpus, "synthesis")
    assert not result["ok"]
    return result["message"]


def test_parse_finds_targets_and_the_lines_that_cite_them() -> None:
    parsed = parse_citations(
        "One.[^a]\nTwo.[^a] Three.[^b]\n\n[^a]: https://a.test/x\n[^b]: local-sources/n.md\n"
    )

    assert parsed.targets == {"a": "https://a.test/x", "b": "local-sources/n.md"}
    assert parsed.references == {"a": [1, 2], "b": [2]}


@pytest.mark.parametrize(
    "definition", ["[^a]: <https://a.test/x>", '[^a]: https://a.test/x "A guide"']
)
def test_common_footnote_styles_still_name_the_source(definition: str) -> None:
    assert parse_citations(f"Claim.[^a]\n\n{definition}\n").targets == {"a": "https://a.test/x"}


def local_copy(citation: str) -> str:
    return (
        f"# Talk\n\n**Source type:** local_file  \n**Citation:** {citation}  \n"
        "**Local path:** local-sources/talk.md  \n**Fetched:** now  \n\nBody.\n"
    )


def test_a_local_copy_matches_the_address_in_its_citation(tmp_path: Path) -> None:
    project = ProjectFolder(
        make_liner(
            tmp_path,
            "t",
            synthesis=f"Claim.[^v]\n\n[^v]: {URL}\n",
            sources={"01-talk.md": local_copy(f"A talk. Transcript. {URL}.")},
        )
    )

    assert list_citations(project)["citations"][0]["file"].endswith("01-talk.md")


def test_a_footnote_with_trailing_punctuation_still_opens_its_source(tmp_path: Path) -> None:
    project = ProjectFolder(
        make_liner(
            tmp_path,
            "t",
            synthesis=f"Claim.[^g]\n\n[^g]: {URL}.\n",
            sources={"01-guide.md": SOURCE_FILE},
        )
    )

    assert list_citations(project)["citations"][0]["file"].endswith("01-guide.md")


def test_a_real_source_wins_over_a_citation_that_names_its_address(tmp_path: Path) -> None:
    project = ProjectFolder(
        make_liner(
            tmp_path,
            "t",
            synthesis=f"Claim.[^v]\n\n[^v]: {URL}\n",
            sources={"01-talk.md": local_copy(URL), "02-guide.md": SOURCE_FILE},
        )
    )

    assert list_citations(project)["citations"][0]["file"].endswith("02-guide.md")


def test_a_skill_source_matches_its_compiled_file(tmp_path: Path) -> None:
    source = "# Skill\n\n**Skill path/name:** skills/foo  \n**Fetched:** now  \n\nBody.\n"
    project = ProjectFolder(
        make_liner(
            tmp_path,
            "t",
            synthesis="Claim.[^s]\n\n[^s]: skills/foo\n",
            sources={"01-skill.md": source},
        )
    )

    assert list_citations(project)["citations"][0]["file"] == str(
        project.sources_dir / "01-skill.md"
    )


def test_a_cited_kept_source_passes(tmp_path: Path) -> None:
    corpus = corpus_with(tmp_path, f"Claim.[^g]\n\n{SECTIONS}\n[^g]: {URL}\n")

    assert check_phase(corpus, "synthesis")["ok"]


@pytest.mark.parametrize(
    ("body", "expected"),
    [
        ("Claim.[^g]\n", "without defining"),
        (f"Claim.\n\n[^g]: {URL}\n", "never cites"),
        ("Claim.[^g]\n\n[^g]: https://example.com/dropped\n", "not a kept source"),
        ("No citations.\n", "must cite at least 1"),
    ],
)
def test_bad_citations_are_explained(tmp_path: Path, body: str, expected: str) -> None:
    assert expected in failure(corpus_with(tmp_path, f"{body}\n{SECTIONS}"))


def test_three_sources_are_needed_when_more_were_kept(tmp_path: Path) -> None:
    corpus = corpus_with(tmp_path, f"Claim.[^g]\n\n{SECTIONS}\n[^g]: {URL}\n", kept=4)

    assert "must cite at least 3" in failure(corpus)


def test_listing_matches_compiled_files_and_reports_gaps(tmp_path: Path) -> None:
    synthesis = f"Claim.[^g] Other.[^n]\n\n[^g]: {URL}\n[^n]: https://example.com/not-compiled\n"
    project = ProjectFolder(
        make_liner(tmp_path, "t", synthesis=synthesis, sources={"01-guide.md": SOURCE_FILE})
    )

    citations = list_citations(project)["citations"]

    assert citations[0] == {
        "label": "g",
        "source": URL,
        "lines": [1],
        "file": str(project.sources_dir / "01-guide.md"),
    }
    assert citations[1]["file"] is None


def test_an_older_liner_has_nothing_to_list(tmp_path: Path) -> None:
    project = ProjectFolder(make_liner(tmp_path, "t"))

    assert list_citations(project)["citations"] == []


FIXTURE_LIBRARY = Path(__file__).parent / "fixtures" / "library"
NAVIGATOR_REPORT = (
    Path(__file__).parents[1] / "packages/go-tui/internal/navigator/testdata/mixed-library.json"
)


def navigator_report() -> dict:
    """The library report for the shared liner, with its folder written as FIXTURE."""
    text = json.dumps(build_library_view(FIXTURE_LIBRARY.resolve()), indent=1, sort_keys=True)
    return json.loads(text.replace(str(FIXTURE_LIBRARY.resolve()), "FIXTURE"))


def citation_lookup(report: dict) -> dict:
    liner = report["liners"][0]
    return {
        "files": liner["synthesis"]["files"],
        "sources": [(source["locator"], source["file"]) for source in liner["sources"]],
    }


def test_every_kind_of_source_in_the_shared_liner_opens() -> None:
    result = list_citations(ProjectFolder(FIXTURE_LIBRARY / "mixed-sources"))

    assert [citation["label"] for citation in result["citations"]] == ["web", "talk", "skill"]
    assert all(citation["file"] for citation in result["citations"])


def test_the_navigator_tests_read_the_report_core_writes_today() -> None:
    """The Go navigator tests load this saved report. Set LINER_UPDATE_NAVIGATOR_REPORT=1
    to save it again after the library report changes."""
    report = navigator_report()
    if os.environ.get("LINER_UPDATE_NAVIGATOR_REPORT") == "1":
        NAVIGATOR_REPORT.write_text(json.dumps(report, indent=1, sort_keys=True) + "\n")
    saved = json.loads(NAVIGATOR_REPORT.read_text(encoding="utf-8"))

    assert citation_lookup(saved) == citation_lookup(report)
    assert None not in citation_lookup(report)["files"].values()


def test_a_build_stops_when_a_citation_opens_nothing(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    from liner import cli
    from liner.research import ResearchError

    project = ProjectFolder(
        make_liner(
            tmp_path,
            "t",
            synthesis=f"Claim.[^g] Other.[^gone]\n\n[^g]: {URL}\n[^gone]: https://gone.test/x\n",
            sources={"01-guide.md": SOURCE_FILE},
        )
    )
    monkeypatch.setattr(cli, "compile_project", lambda *args, **kwargs: None)

    with pytest.raises(ResearchError, match=r"\[\^gone\]"):
        cli._compile_research_corpus(project)
