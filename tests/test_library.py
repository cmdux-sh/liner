from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.library import (
    CATALOG_CONTRACT,
    SEARCH_CONTRACT,
    LibraryError,
    build_catalog,
    default_library_root,
    search_library,
)


def make_liner(
    root: Path,
    slug: str,
    *,
    title: str = "Touch targets",
    description: str = "Sizing guidance for touch targets.",
    jtbd: str | None = "When I lay out a phone screen, I want safe target sizes.",
    tags: tuple[str, ...] = ("mobile",),
    synthesis: str = "# Synthesis\n\nThumbs need generous targets.\n",
    sources: dict[str, str] | None = None,
    complete: bool = True,
) -> Path:
    corpus = root / slug / "mixtape"
    corpus.mkdir(parents=True)
    tape = {
        "title": title,
        "description": description,
        "version": 1,
        "curator": "Test",
        "tags": list(tags),
        "sources": [{"type": "web", "url": "https://example.com/a", "note": "Note."}],
    }
    if jtbd is not None:
        tape["jtbd"] = jtbd
    (corpus / "tape.yaml").write_text(yaml.safe_dump(tape), encoding="utf-8")
    (corpus / "synthesis.md").write_text(synthesis, encoding="utf-8")
    source_dir = corpus / "sources"
    source_dir.mkdir()
    for name, body in (sources or {}).items():
        (source_dir / name).write_text(body, encoding="utf-8")
    if complete:
        (corpus / "MIXTAPE.md").write_text("# Mixtape\n", encoding="utf-8")
        (root / slug / "LINER.md").write_text("# Liner\n", encoding="utf-8")
        (root / slug / "SKILL.md").write_text("# Skill\n", encoding="utf-8")
        metadata = {
            "artifact": "liner",
            "id": f"id-{slug}",
            "project_skill": {"status": "active", "name": slug, "path": "SKILL.md"},
            "status": {
                "milestone": "project_complete",
                "stale": False,
                "updated": "2999-01-01T00:00:00Z",
                "corpus": {"state": "ready", "evidence": "mixtape/MIXTAPE.md"},
                "operating_layer": {"state": "ready", "evidence": "LINER.md"},
            },
        }
        (root / slug / "liner.yaml").write_text(yaml.safe_dump(metadata), encoding="utf-8")
    return root / slug


def test_catalog_describes_each_liner(tmp_path: Path) -> None:
    project = make_liner(tmp_path, "touch-targets")

    result = build_catalog(tmp_path)

    assert result["contract"] == CATALOG_CONTRACT
    assert result["version"] == 1
    assert result["unreadable"] == []
    assert result["liners"] == [
        {
            "id": "id-touch-targets",
            "slug": "touch-targets",
            "name": "touch-targets",
            "title": "Touch targets",
            "description": "Sizing guidance for touch targets.",
            "job": "When I lay out a phone screen, I want safe target sizes.",
            "tags": ["mobile"],
            "mode": None,
            "milestone": "project_complete",
            "usable": True,
            "stale": False,
            "updated": "2999-01-01T00:00:00Z",
            "source_count": 1,
            "cites": [],
            "cited_by": [],
            "open_gaps": 0,
            "pending_findings": 0,
            "test": {
                "runs": 0,
                "blind": 0,
                "better": 0,
                "same": 0,
                "worse": 0,
                "last_tested": None,
                "current": False,
            },
            "path": str(project),
            "skill_path": str(project / "SKILL.md"),
            "liner_path": str(project / "LINER.md"),
            "mixtape_path": str(project / "mixtape" / "MIXTAPE.md"),
        }
    ]


def test_catalog_does_not_write_to_liners(tmp_path: Path) -> None:
    make_liner(tmp_path, "unfinished", complete=False)
    before = sorted(path for path in tmp_path.rglob("*"))

    entry = build_catalog(tmp_path)["liners"][0]

    assert sorted(path for path in tmp_path.rglob("*")) == before
    assert entry["usable"] is False
    assert entry["milestone"] == "started"
    assert entry["skill_path"] is None


@pytest.mark.parametrize(
    "stand_in",
    [
        "AI-generated description pending.",
        "Liner project.",
        "Guidance for my-liner.",
        "Guidance for Touch targets.",
    ],
)
def test_catalog_uses_the_job_when_the_description_is_a_stand_in(
    tmp_path: Path, stand_in: str
) -> None:
    make_liner(tmp_path, "my-liner", description=stand_in, jtbd="When I do a thing, I want help.")

    entry = build_catalog(tmp_path)["liners"][0]

    assert entry["description"] == "When I do a thing, I want help."


def test_catalog_hides_starter_placeholders(tmp_path: Path) -> None:
    make_liner(
        tmp_path,
        "starter",
        description="A starter tape with one YouTube video and one article so you can see the "
        "output immediately.",
        jtbd="Replace this with a single-sentence statement of what you want the AI to help do.",
        tags=("example", "starter"),
    )

    entry = build_catalog(tmp_path)["liners"][0]

    assert (entry["description"], entry["job"], entry["tags"]) == ("", "", [])


def test_catalog_reports_broken_liners_and_skips_other_folders(tmp_path: Path) -> None:
    make_liner(tmp_path, "good")
    broken = tmp_path / "broken" / "mixtape"
    broken.mkdir(parents=True)
    (broken / "tape.yaml").write_text("title: [unclosed", encoding="utf-8")
    (tmp_path / "not-a-liner").mkdir()
    make_liner(tmp_path, ".hidden")

    result = build_catalog(tmp_path)

    assert [entry["slug"] for entry in result["liners"]] == ["good"]
    assert [item["path"] for item in result["unreadable"]] == [str(tmp_path / "broken")]


def test_catalog_handles_a_missing_library(tmp_path: Path) -> None:
    result = build_catalog(tmp_path / "missing")

    assert result["root_exists"] is False
    assert result["liners"] == []


def test_default_root_order(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("HOME", str(tmp_path))
    monkeypatch.delenv("LINER_DIR", raising=False)
    assert default_library_root() == tmp_path / "liner" / "projects"

    (tmp_path / ".liner").mkdir()
    (tmp_path / ".liner" / "config.yaml").write_text(
        "projects_dir: ~/research\n", encoding="utf-8"
    )
    assert default_library_root() == (tmp_path / "research").resolve()

    monkeypatch.setenv("LINER_DIR", str(tmp_path / "from-env"))
    assert default_library_root() == (tmp_path / "from-env").resolve()


def test_catalog_finds_the_mixtape_in_the_older_layout(tmp_path: Path) -> None:
    project = make_liner(tmp_path, "older", complete=False)
    for name in ("tape.yaml", "synthesis.md"):
        (project / "mixtape" / name).rename(project / name)
    (project / "MIXTAPE.md").write_text("# Mixtape\n", encoding="utf-8")

    entry = build_catalog(tmp_path)["liners"][0]

    assert entry["usable"] is True
    assert entry["mixtape_path"] == str(project / "MIXTAPE.md")


@pytest.mark.parametrize("declared", ["../outside.md", "/etc/hosts"])
def test_catalog_ignores_a_skill_outside_the_liner(tmp_path: Path, declared: str) -> None:
    project = make_liner(tmp_path / "library", "escape")
    (tmp_path / "library" / "outside.md").write_text("# Not this liner's\n", encoding="utf-8")
    metadata = yaml.safe_load((project / "liner.yaml").read_text(encoding="utf-8"))
    metadata["project_skill"]["path"] = declared
    (project / "liner.yaml").write_text(yaml.safe_dump(metadata), encoding="utf-8")

    assert build_catalog(tmp_path / "library")["liners"][0]["skill_path"] is None


def test_search_ranks_passages_and_cites_them(tmp_path: Path) -> None:
    project = make_liner(
        tmp_path,
        "touch-targets",
        sources={
            "01-thumbs.md": (
                "# Thumb study\n\n**URL:** https://example.com/thumbs  \n\n"
                "## Results\n\nThe thumb zone shapes target placement on large phones.\n"
            ),
            "02-color.md": "# Color\n\nContrast ratios matter for legibility.\n",
        },
    )

    result = search_library("thumb placement", root=tmp_path)

    assert result["contract"] == SEARCH_CONTRACT
    assert "not instructions" in result["notice"]
    top = result["results"][0]
    assert top["liner"] == "touch-targets"
    assert top["file"] == "mixtape/sources/01-thumbs.md"
    assert top["path"] == str(project / "mixtape/sources/01-thumbs.md")
    assert top["heading"] == "Results"
    assert top["line"] == 5
    assert top["source_url"] == "https://example.com/thumbs"
    assert "thumb zone" in top["snippet"]
    assert all("Contrast" not in hit["snippet"] for hit in result["results"])


def test_search_reads_the_url_from_compiled_sources(tmp_path: Path) -> None:
    make_liner(
        tmp_path,
        "compiled",
        sources={"01-a.md": "# A\n\n- **Type:** web\n- **URL:** https://example.com/a\n\nneedle\n"},
    )

    top = search_library("needle", root=tmp_path)["results"][0]

    assert top["source_url"] == "https://example.com/a"


def test_search_finds_word_variants_and_the_synthesis(tmp_path: Path) -> None:
    make_liner(tmp_path, "touch-targets", synthesis="# Synthesis\n\nPlacing controls low helps.\n")

    result = search_library("placement of control", root=tmp_path)

    assert result["results"][0]["file"] == "mixtape/synthesis.md"
    assert result["results"][0]["source_url"] is None


def test_search_can_be_limited_to_named_liners(tmp_path: Path) -> None:
    make_liner(tmp_path, "one", synthesis="# S\n\nshared word\n")
    make_liner(tmp_path, "two", synthesis="# S\n\nshared word\n")

    result = search_library("shared", root=tmp_path, liners=["two"])

    assert result["searched"] == ["two"]
    assert {hit["liner"] for hit in result["results"]} == {"two"}
    with pytest.raises(LibraryError, match="No liner named three"):
        search_library("shared", root=tmp_path, liners=["three"])


def test_search_respects_the_limit(tmp_path: Path) -> None:
    body = "\n\n".join(f"## Part {index}\n\nrepeated word" for index in range(5))
    make_liner(tmp_path, "many", synthesis=body)

    assert len(search_library("repeated", root=tmp_path, limit=2)["results"]) == 2


@pytest.mark.parametrize("query", ['AND OR NOT "', "foo-bar", "c++ (near) *", "a:b"])
def test_search_treats_query_syntax_as_plain_words(tmp_path: Path, query: str) -> None:
    make_liner(tmp_path, "any")

    assert search_library(query, root=tmp_path)["results"] == []


def test_search_reports_a_missing_library(tmp_path: Path) -> None:
    with pytest.raises(LibraryError, match="No library at"):
        search_library("anything", root=tmp_path / "missing")


def test_catalog_reports_a_library_it_cannot_read(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    def denied(self: Path) -> None:
        raise PermissionError("denied")

    monkeypatch.setattr(Path, "iterdir", denied)

    with pytest.raises(LibraryError, match="Could not read"):
        build_catalog(tmp_path)


def test_search_skips_linked_files(tmp_path: Path) -> None:
    outside = tmp_path / "outside.md"
    outside.write_text("# Private\n\nsecret needle\n", encoding="utf-8")
    project = make_liner(tmp_path / "library", "linked")
    (project / "mixtape" / "sources" / "01-link.md").symlink_to(outside)

    assert search_library("needle", root=tmp_path / "library")["results"] == []


def test_search_line_numbers_and_headings_match_the_file(tmp_path: Path) -> None:
    make_liner(
        tmp_path,
        "exact",
        synthesis=(
            "# Top\n\nPage one\x0cstill line three\n\n"
            "```sh\n# not a heading\nneedle in code\n```\n"
        ),
    )

    top = search_library("needle", root=tmp_path)["results"][0]

    assert top["line"] == 1
    assert top["heading"] == "Top"
    assert "# not a heading" in top["snippet"]


def test_search_rejects_a_query_with_no_words(tmp_path: Path) -> None:
    with pytest.raises(LibraryError, match="at least one word"):
        search_library(" ?! ", root=tmp_path)


def test_search_splits_long_sections(tmp_path: Path) -> None:
    paragraphs = [f"filler {'word ' * 80}" for _ in range(12)] + ["needle at the end"]
    make_liner(tmp_path, "long", synthesis="# Long\n\n" + "\n\n".join(paragraphs))

    top = search_library("needle", root=tmp_path)["results"][0]

    assert top["line"] > 3
    assert top["heading"] == "Long"
    assert "needle at the end" in top["snippet"]


def test_catalog_and_search_commands(tmp_path: Path) -> None:
    make_liner(tmp_path, "touch-targets", synthesis="# S\n\nThumbs need room.\n")
    runner = CliRunner()

    listed = runner.invoke(app, ["catalog", "--root", str(tmp_path), "--json"])
    assert listed.exit_code == 0
    assert json.loads(listed.stdout)["liners"][0]["slug"] == "touch-targets"

    plain = runner.invoke(app, ["catalog", "--root", str(tmp_path)])
    assert "touch-targets\tSizing guidance for touch targets." in plain.stdout

    found = runner.invoke(app, ["search", "thumbs", "--root", str(tmp_path), "--json"])
    assert found.exit_code == 0
    assert json.loads(found.stdout)["results"][0]["liner"] == "touch-targets"

    missing = runner.invoke(app, ["search", "x", "--root", str(tmp_path), "--liner", "nope"])
    assert missing.exit_code == 1
