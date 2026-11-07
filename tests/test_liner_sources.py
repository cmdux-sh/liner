from __future__ import annotations

from pathlib import Path
from uuid import uuid4

import pytest
import yaml

from liner.compile import compile_project
from liner.config import Config
from liner.handlers.base import HandlerHardFailure
from liner.handlers.liner import LinerHandler
from liner.library import build_catalog, search_library
from liner.maintenance import apply_change_set, plan_source_add, plan_synthesis_review
from liner.project import SYNTHESIS_PLACEHOLDER, ProjectFolder, read_liner_metadata
from liner.refresh import check_sources
from liner.tape import TapeValidationError, validate_tape
from liner.types import SourceSpec
from tests.test_library import make_liner


def identified(root: Path, slug: str, synthesis: str) -> ProjectFolder:
    folder = ProjectFolder(make_liner(root, slug, synthesis=synthesis))
    metadata = read_liner_metadata(folder)
    metadata.update({"version": 2, "id": str(uuid4()), "mixtape": "mixtape"})
    folder.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    tape = yaml.safe_load(folder.tape_path.read_text(encoding="utf-8"))
    tape["sources"] = []
    folder.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    return folder


@pytest.fixture
def pair(tmp_path: Path) -> tuple[ProjectFolder, ProjectFolder]:
    cited = identified(tmp_path, "typography", "# Synthesis\n\nBody text starts at 16 px.\n")
    citing = identified(tmp_path, "forms", "# Synthesis\n\nForms need clear labels.\n")
    source = {
        "type": "liner",
        "url": "liner://typography",
        "note": "Type sizes.",
        "section": "Foundations",
        "kind": "reference",
    }
    apply_change_set(citing.path, plan_source_add(citing.path, source), approved=True)
    review = plan_synthesis_review(citing.path, "still_current")
    apply_change_set(citing.path, review, approved=True)
    return citing, cited


def compile_citing(project: ProjectFolder) -> None:
    compile_project(project, cache=None, handlers={}, config=Config(), no_cache=True)


def test_a_cited_synthesis_compiles_and_is_searchable(
    pair: tuple[ProjectFolder, ProjectFolder], tmp_path: Path
) -> None:
    citing, _ = pair

    compile_citing(citing)

    results = search_library("16 px", root=tmp_path, liners=["forms"])["results"]
    assert any(result["source_url"] == "liner://typography" for result in results)


def test_the_catalog_shows_both_directions(
    pair: tuple[ProjectFolder, ProjectFolder], tmp_path: Path
) -> None:
    entries = {entry["slug"]: entry for entry in build_catalog(tmp_path)["liners"]}

    assert entries["forms"]["cites"] == ["typography"]
    assert entries["forms"]["cited_by"] == []
    assert entries["typography"]["cited_by"] == ["forms"]


def test_refresh_notices_when_a_cited_liner_changes(
    pair: tuple[ProjectFolder, ProjectFolder],
) -> None:
    citing, cited = pair
    compile_citing(citing)
    assert check_sources(citing, handlers={}, config=Config())["counts"]["unchanged"] == 1

    cited.synthesis_path.write_text("# Synthesis\n\nBody text starts at 18 px.\n", encoding="utf-8")

    assert check_sources(citing, handlers={}, config=Config())["counts"]["changed"] == 1


@pytest.mark.parametrize(
    "url", ["liner://../secret", "liner://a/b", "liner://", "liner://café", "https://example.com"]
)
def test_tape_rejects_anything_but_a_slug(url: str) -> None:
    raw = {
        "title": "T",
        "description": "D",
        "version": 1,
        "curator": "C",
        "sources": [{"type": "liner", "url": url}],
    }
    with pytest.raises(TapeValidationError):
        validate_tape(raw)


def test_a_liner_cannot_cite_itself(pair: tuple[ProjectFolder, ProjectFolder]) -> None:
    citing, _ = pair
    with pytest.raises(HandlerHardFailure, match="itself"):
        LinerHandler(citing).fetch(SourceSpec(type="liner", url="liner://forms"))


def test_a_placeholder_synthesis_cannot_be_cited(
    pair: tuple[ProjectFolder, ProjectFolder],
) -> None:
    citing, cited = pair
    cited.synthesis_path.write_text(SYNTHESIS_PLACEHOLDER, encoding="utf-8")
    with pytest.raises(HandlerHardFailure, match="no synthesis yet"):
        LinerHandler(citing).fetch(SourceSpec(type="liner", url="liner://typography"))


def test_an_unfinished_or_missing_liner_cannot_be_cited(
    pair: tuple[ProjectFolder, ProjectFolder],
) -> None:
    citing, cited = pair
    cited.mixtape_path.unlink()
    for url in ("liner://typography", "liner://nowhere"):
        with pytest.raises(HandlerHardFailure, match="No finished liner"):
            LinerHandler(citing).fetch(SourceSpec(type="liner", url=url))


class _Page:
    def fetch(self, spec: SourceSpec) -> object:
        from datetime import UTC, datetime

        from liner.types import SourceContent

        return SourceContent(
            title="Type scale guide",
            url=spec.url,
            body="Body text starts at 16 px. Headings step up by a ratio of 1.25.",
            fetched_at=datetime.now(UTC).isoformat(),
        )


@pytest.fixture
def reuse(tmp_path: Path) -> tuple[ProjectFolder, ProjectFolder, str]:
    """A typography liner with one compiled web page, and a forms liner reusing it."""
    cited = identified(tmp_path, "typography", "# Synthesis\n\nBody text starts at 16 px.\n")
    page = {"type": "web", "url": "https://example.com/type-scale", "note": "Sizes."}
    apply_change_set(cited.path, plan_source_add(cited.path, page), approved=True)
    apply_change_set(cited.path, plan_synthesis_review(cited.path, "still_current"), approved=True)
    compile_project(cited, cache=None, handlers={"web": _Page()}, config=Config(), no_cache=True)
    source_id = yaml.safe_load(cited.tape_path.read_text(encoding="utf-8"))["sources"][0]["id"]
    citing = identified(tmp_path, "forms", "# Synthesis\n\nForms need clear labels.\n")
    reused = {
        "type": "liner",
        "url": f"liner://typography/{source_id}",
        "note": "Type sizes for form labels.",
    }
    apply_change_set(citing.path, plan_source_add(citing.path, reused), approved=True)
    apply_change_set(
        citing.path, plan_synthesis_review(citing.path, "still_current"), approved=True
    )
    return citing, cited, source_id


def test_one_source_is_reused_with_its_original_address(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    from liner.citations import compiled_source_files

    citing, _, source_id = reuse
    compile_citing(citing)

    files = compiled_source_files(citing)
    compiled = files[f"liner://typography/{source_id}"]
    text = compiled.read_text(encoding="utf-8")
    assert "Headings step up by a ratio of 1.25." in text
    assert "**Reused from:** liner `typography`" in text
    # A citation of the original page opens the reused copy.
    assert files["https://example.com/type-scale"] == compiled


def test_find_sources_gives_the_reuse_address(
    reuse: tuple[ProjectFolder, ProjectFolder, str], tmp_path: Path
) -> None:
    from liner.library import find_sources

    _, _, source_id = reuse
    rows = find_sources("type scale", root=tmp_path)["sources"]

    assert {row["reuse"] for row in rows if row["liner"] == "typography"} == {
        f"liner://typography/{source_id}"
    }


def test_catalog_counts_a_reused_source_as_citing_its_liner(
    reuse: tuple[ProjectFolder, ProjectFolder, str], tmp_path: Path
) -> None:
    entries = {entry["slug"]: entry for entry in build_catalog(tmp_path)["liners"]}

    assert entries["forms"]["cites"] == ["typography"]


def test_refresh_says_where_a_reused_page_is_checked(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    citing, _, _ = reuse
    compile_citing(citing)

    report = check_sources(citing, handlers={}, config=Config())

    assert report["counts"]["unchanged"] == 1
    assert "liner refresh typography" in report["reused"][0]["reason"]


def test_an_unknown_source_id_names_the_fix(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    citing, _, _ = reuse
    missing = f"liner://typography/{uuid4()}"

    with pytest.raises(HandlerHardFailure, match="liner find-sources"):
        LinerHandler(citing).fetch(SourceSpec(type="liner", url=missing))


def test_share_and_refresh_flag_a_reused_local_file(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    from liner.handlers.liner import reused_local_sources
    from liner.tape import load_tape

    citing, cited, source_id = reuse
    tape = yaml.safe_load(cited.tape_path.read_text(encoding="utf-8"))
    tape["sources"][0] = {
        "id": source_id,
        "type": "local_file",
        "path": "local-sources/notes.md",
        "citation": "Team notes, 2026",
    }
    cited.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    sources = load_tape(citing.tape_path).sources
    assert reused_local_sources(citing, sources) == [f"liner://typography/{source_id}"]
    report = check_sources(citing, handlers={}, config=Config())
    assert "no live page" in report["reused"][0]["reason"]


def test_tape_accepts_a_reuse_address() -> None:
    raw = {
        "title": "T",
        "description": "D",
        "version": 1,
        "curator": "C",
        "sources": [{"type": "liner", "url": f"liner://typography/{uuid4()}"}],
    }
    assert validate_tape(raw).sources[0].url.startswith("liner://typography/")


def test_share_still_warns_after_the_other_liner_moves(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    import shutil

    from liner.handlers.liner import reused_local_sources
    from liner.tape import load_tape

    citing, cited, source_id = reuse
    tape = yaml.safe_load(cited.tape_path.read_text(encoding="utf-8"))
    page = tape["sources"][0]
    tape["sources"][0] = {
        "id": source_id,
        "type": "local_file",
        "path": "local-sources/notes.md",
        "citation": "Team notes, 2026",
    }
    cited.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    compiled = next(cited.sources_dir.glob("*.md"))
    text = compiled.read_text(encoding="utf-8").replace(
        f"**URL:** {page['url']}", "**Local path:** local-sources/notes.md"
    )
    compiled.write_text(text, encoding="utf-8")
    compile_citing(citing)
    shutil.rmtree(cited.path)

    sources = load_tape(citing.tape_path).sources
    assert reused_local_sources(citing, sources) == [f"liner://typography/{source_id}"]


def test_find_sources_offers_no_reuse_without_saved_text(
    reuse: tuple[ProjectFolder, ProjectFolder, str], tmp_path: Path
) -> None:
    from liner.library import find_sources

    _, cited, _ = reuse
    for path in cited.sources_dir.glob("*.md"):
        path.unlink()

    rows = find_sources("type scale", root=tmp_path)["sources"]
    assert all(row["reuse"] is None for row in rows if row["liner"] == "typography")


def test_an_unvalidated_address_fails_closed() -> None:
    from liner.tape import liner_locator

    with pytest.raises(ValueError):
        liner_locator("liner://../secret")


def test_fetch_saves_a_reused_source_for_evaluation(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    import json

    from typer.testing import CliRunner

    from liner.cli import app

    citing, _, source_id = reuse
    root = str(citing.path.parent)
    address = f"liner://typography/{source_id}"

    result = CliRunner().invoke(app, ["fetch", "forms", address, "--root", root, "--json"])

    assert result.exit_code == 0, result.output
    saved = Path(json.loads(result.stdout)["path"]).read_text(encoding="utf-8")
    assert f"Fetched from {address}" in saved
    assert "Headings step up by a ratio of 1.25." in saved


def test_fetch_names_a_reuse_address_it_cannot_read(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    from typer.testing import CliRunner

    from liner.cli import app

    citing, _, _ = reuse
    address = "liner://typography/00000000-0000-0000-0000-000000000000"

    result = CliRunner().invoke(
        app, ["fetch", "forms", address, "--root", str(citing.path.parent)]
    )

    assert result.exit_code == 1
    assert "find-sources" in " ".join(result.output.split())


def test_a_reusable_copy_ranks_above_a_failed_read_of_the_same_page(
    reuse: tuple[ProjectFolder, ProjectFolder, str], tmp_path: Path
) -> None:
    from liner.library import find_sources

    citing, _, source_id = reuse
    working = citing.corpus_path / "working"
    working.mkdir(parents=True, exist_ok=True)
    (working / "03-evaluation.yaml").write_text(
        yaml.safe_dump(
            {
                "candidates": [
                    {
                        "url": "https://example.com/type-scale",
                        "title": "Type scale",
                        "decision": "dropped",
                        "rationale": "Stub: the type scale page came back empty.",
                    }
                ]
            }
        ),
        encoding="utf-8",
    )

    # The failed read matches more of the query than the reusable copy does.
    rows = find_sources("type scale page empty", root=tmp_path)["sources"]
    same_page = [row for row in rows if row["url"] == "https://example.com/type-scale"]

    assert [row["liner"] for row in same_page][0] == "typography"
    assert same_page[0]["reuse"] == f"liner://typography/{source_id}"


def test_fetch_refuses_a_liner_reusing_itself(
    reuse: tuple[ProjectFolder, ProjectFolder, str],
) -> None:
    from typer.testing import CliRunner

    from liner.cli import app

    citing, _, _ = reuse
    root = str(citing.path.parent)

    result = CliRunner().invoke(app, ["fetch", "forms", "liner://forms", "--root", root])

    assert result.exit_code == 1
    assert "can't cite itself" in " ".join(result.output.split())

