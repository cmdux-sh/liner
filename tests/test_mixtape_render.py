from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

from liner.output.mixtape import write_mixtape, written_source_paths
from liner.project import ProjectFolder, init_project
from liner.types import (
    CompiledSource,
    CompileResult,
    CompileWarning,
    SourceContent,
    SourceSpec,
    Tape,
)


def _result() -> CompileResult:
    tape = Tape(
        title="My Tape",
        description="for tests",
        version=1,
        curator="me",
        sources=(),
        mode="quick",
        jtbd="Test how the renderer formats master and source files.",
    )
    s1 = SourceSpec(type="youtube", url="https://yt/1", note="watch first", section="intro")
    s2 = SourceSpec(type="web", url="https://ex/a", note="bg reading", section="intro")
    s3 = SourceSpec(type="web", url="https://ex/b", note="loose end")  # no section
    c1 = SourceContent(
        title="Vid 1",
        url=s1.url,
        body="A short transcript.",
        fetched_at="2026-05-16T00:00:00+00:00",
        author="Channel A",
        duration_seconds=125,
        metadata={"transcript_type": "manual"},
    )
    c2 = SourceContent(
        title="Article A",
        url=s2.url,
        body="Article body.",
        fetched_at="2026-05-16T00:00:00+00:00",
        author="Writer",
        published_at="2026-05-01",
        metadata={},
    )
    return CompileResult(
        tape=tape,
        compiled_at=datetime(2026, 5, 16, 12, 0, 0, tzinfo=UTC),
        sources=(
            CompiledSource(spec=s1, content=c1),
            CompiledSource(spec=s2, content=c2),
            CompiledSource(spec=s3, content=None),
        ),
        warnings=(CompileWarning(url=s3.url, message="404", severity="error"),),
    )


def _project_with_synthesis(tmp_path: Path) -> ProjectFolder:
    project = init_project(tmp_path / "demo")
    project.synthesis_path.write_text("My synthesis prose here.\n", encoding="utf-8")
    return project


def test_write_mixtape_writes_master_and_source_files(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())

    assert project.mixtape_path.exists()
    files = sorted(p.name for p in project.sources_dir.iterdir())
    assert files == ["01-vid-1.md", "02-article-a.md", "03-https-ex-b.md"]


def test_master_file_has_synthesis_and_index(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())

    text = project.mixtape_path.read_text(encoding="utf-8")
    assert "# My Tape" in text
    assert "**Mode:** quick" in text
    assert "**JTBD:** Test how the renderer formats" in text
    assert "## How to use this mixtape" in text
    assert "## Synthesis" in text
    assert "My synthesis prose here." in text
    assert "### intro" in text
    assert "### Ungrouped" in text
    assert "#### Source 1: Vid 1" in text
    assert "#### Source 2: Article A" in text
    assert "[./sources/01-vid-1.md]" in text
    # Sections rendered in first-appearance order.
    assert text.index("### intro") < text.index("### Ungrouped")


def test_master_file_includes_warnings(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())
    text = project.mixtape_path.read_text(encoding="utf-8")
    assert "Compilation notes" in text
    assert "404" in text


def test_source_file_format(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())
    body = (project.sources_dir / "01-vid-1.md").read_text(encoding="utf-8")
    assert body.startswith("# Vid 1\n")
    assert "**Source type:** youtube" in body
    assert "**URL:** https://yt/1" in body
    assert "**Author:** Channel A" in body
    assert "**Duration:** 2:05" in body
    assert "**Fetched:** 2026-05-16" in body
    assert "watch first" in body
    assert "A short transcript." in body


def test_failed_source_file_notes_unavailability(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())
    body = (project.sources_dir / "03-https-ex-b.md").read_text(encoding="utf-8")
    assert "Source unavailable" in body


def test_written_source_paths_matches_files(tmp_path: Path) -> None:
    project = _project_with_synthesis(tmp_path)
    write_mixtape(project, _result())
    plan = written_source_paths(project, _result())
    on_disk = sorted(p.name for p in project.sources_dir.iterdir())
    assert sorted(entry["filename"] for entry in plan) == on_disk
    assert plan[0]["succeeded"] is True
    assert plan[2]["succeeded"] is False


def test_orphan_compile_output_is_cleaned_up(tmp_path: Path) -> None:
    """A file from a prior compile whose slug no longer matches should be
    deleted on the next compile. Reproduces the real Medium-rescue case:
    first compile failed → slug derived from URL; second compile recovered →
    slug derived from the rescued title."""
    project = _project_with_synthesis(tmp_path)
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    # Seed an orphan that matches the compile-output signature.
    orphan = project.sources_dir / "03-https-ex-b-old-slug.md"
    orphan.write_text("stale", encoding="utf-8")

    write_mixtape(project, _result())

    assert not orphan.exists(), "Orphan compile output should be removed"
    # The real planned files for source 3 are still there.
    assert (project.sources_dir / "03-https-ex-b.md").exists()


def test_curator_added_files_are_preserved(tmp_path: Path) -> None:
    """Cleanup MUST NOT touch files the curator put in sources/ by hand —
    they don't carry the NN- index-prefix signature this code writes."""
    project = _project_with_synthesis(tmp_path)
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    notes = project.sources_dir / "notes.md"
    notes.write_text("my notes", encoding="utf-8")
    attachment = project.sources_dir / "extra-research.txt"
    attachment.write_text("hand-added", encoding="utf-8")
    single_digit = project.sources_dir / "1-not-our-format.md"
    single_digit.write_text("not a compile output", encoding="utf-8")

    write_mixtape(project, _result())

    assert notes.exists()
    assert attachment.exists()
    assert single_digit.exists()
