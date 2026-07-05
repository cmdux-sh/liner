from __future__ import annotations

from pathlib import Path

import pytest

from liner.project import (
    ProjectFolder,
    init_project,
    read_liner_metadata,
    refresh_status_snapshot,
    slugify,
)


def test_slugify_basic() -> None:
    assert slugify("Hello World") == "hello-world"
    assert slugify("  --  edge  --") == "edge"
    assert slugify("Café Résumé") == "cafe-resume"
    assert slugify("") == "untitled"
    assert slugify("///") == "untitled"


def test_slugify_truncates_long_strings() -> None:
    long = "a" * 200
    out = slugify(long, max_length=30)
    assert len(out) <= 30


def test_init_project_creates_expected_structure(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    assert project.path == tmp_path / "demo"
    assert project.corpus_path == project.path / "mixtape"
    assert project.liner_metadata_path.exists()
    assert project.tape_path.exists()
    assert project.synthesis_path.exists()
    assert not project.local_sources_dir.exists()
    assert (project.working_dir / "01-jtbd-and-knowledge-map.md").exists()
    assert (project.working_dir / "02-candidate-longlist.md").exists()
    assert (project.working_dir / "03-evaluation.yaml").exists()
    assert (project.working_dir / "04-quality-checks.md").exists()
    metadata = read_liner_metadata(project)
    assert metadata["status"]["milestone"] == "started"
    assert metadata["status"]["corpus"]["evidence"] == "mixtape/MIXTAPE.md"
    assert metadata["project_skill"]["status"] == "missing"


def test_refresh_status_marks_mixtape_as_corpus_ready_not_complete(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.mixtape_path.write_text("# MIXTAPE\n\nReady corpus.\n", encoding="utf-8")

    snapshot = refresh_status_snapshot(project)
    metadata = read_liner_metadata(project)

    assert snapshot["milestone"] == "corpus_ready"
    assert snapshot["corpus"]["state"] == "ready"
    assert metadata["status"]["milestone"] == "corpus_ready"
    assert metadata["project_skill"]["status"] == "missing"


def test_init_project_refuses_overwrite_without_force(tmp_path: Path) -> None:
    init_project(tmp_path / "demo")
    with pytest.raises(FileExistsError):
        init_project(tmp_path / "demo")


def test_init_project_force_overwrites(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.tape_path.write_text("# clobbered", encoding="utf-8")
    init_project(tmp_path / "demo", force=True)
    # Force overwrites the tape back to the starter
    assert "starter tape" in project.tape_path.read_text(encoding="utf-8")


def test_project_folder_predicates(tmp_path: Path) -> None:
    project = ProjectFolder(tmp_path / "empty")
    assert not project.is_valid()
    assert not project.has_synthesis()

    init_project(tmp_path / "empty")
    assert project.is_valid()
    assert project.has_synthesis()


def test_project_folder_reads_legacy_root_layout(tmp_path: Path) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    (root / "tape.yaml").write_text("title: Legacy\nversion: 1\ncurator: A\ndescription: D\nsources: []\n", encoding="utf-8")
    (root / "synthesis.md").write_text("Legacy synthesis.\n", encoding="utf-8")

    project = ProjectFolder(root)

    assert project.corpus_path == root
    assert project.tape_path == root / "tape.yaml"
    assert project.is_valid()
    assert project.has_synthesis()


def test_project_folder_keeps_legacy_root_layout_after_liner_metadata(tmp_path: Path) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    (root / "liner.yaml").write_text("artifact: liner\nversion: 2\n", encoding="utf-8")
    (root / "tape.yaml").write_text(
        "title: Legacy\nversion: 1\ncurator: A\ndescription: D\nsources: []\n",
        encoding="utf-8",
    )
    (root / "synthesis.md").write_text("Legacy synthesis.\n", encoding="utf-8")

    project = ProjectFolder(root)

    assert project.corpus_path == root
    assert project.tape_path == root / "tape.yaml"
    assert project.is_valid()
