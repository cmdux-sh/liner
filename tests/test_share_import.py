from __future__ import annotations

import zipfile
from pathlib import Path

import pytest

from liner.project import init_project
from liner.share import ShareOptions, pack, unpack


def _populated_project(tmp_path: Path, with_personal: bool = False) -> Path:
    project = init_project(tmp_path / "topic")
    project.synthesis_path.write_text("Synthesis prose.\n", encoding="utf-8")
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    (project.sources_dir / "01-example.md").write_text("# source 1\nbody", encoding="utf-8")
    project.mixtape_path.write_text("# MIXTAPE\n", encoding="utf-8")
    if with_personal:
        project.personal_dir.mkdir(parents=True, exist_ok=True)
        (project.personal_dir / "note.md").write_text("local note", encoding="utf-8")
    return project.path


def test_pack_default_includes_personal(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path, with_personal=True)
    result = pack(ProjectFolder(path))
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert "topic/personal/note.md" in names


def test_pack_no_personal_excludes_personal(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path, with_personal=True)
    result = pack(
        ProjectFolder(path),
        options=ShareOptions(include_personal=False),
    )
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert not any(n.startswith("topic/personal/") for n in names)
    assert "topic/tape.yaml" in names


def test_pack_and_unpack_personal_roundtrip(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path, with_personal=True)
    result = pack(ProjectFolder(path))
    dest = tmp_path / "extracted"
    project = unpack(result.archive_path, dest)
    assert (project.personal_dir / "note.md").exists()
    assert (project.personal_dir / "note.md").read_text(encoding="utf-8") == "local note"


def test_pack_default_includes_everything(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path)
    result = pack(ProjectFolder(path))

    assert result.archive_path.exists()
    assert result.archive_path.name == "topic.mixtape"
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert "topic/tape.yaml" in names
    assert "topic/synthesis.md" in names
    assert "topic/MIXTAPE.md" in names
    assert any(n.startswith("topic/sources/") for n in names)
    assert any(n.startswith("topic/working/") for n in names)


def test_pack_minimal_only_includes_tape(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path)
    result = pack(ProjectFolder(path), options=ShareOptions(minimal=True))
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert names == {"topic/tape.yaml"}


def test_pack_excludes_working_and_sources_via_flags(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path)
    result = pack(
        ProjectFolder(path),
        options=ShareOptions(include_working_notes=False, include_source_content=False),
    )
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert not any(n.startswith("topic/sources/") for n in names)
    assert not any(n.startswith("topic/working/") for n in names)
    assert "topic/tape.yaml" in names


def test_pack_then_unpack_roundtrip(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path)
    result = pack(ProjectFolder(path))

    dest = tmp_path / "extracted"
    project = unpack(result.archive_path, dest)

    assert project.path == dest / "topic"
    assert project.tape_path.exists()
    assert project.synthesis_path.exists()
    assert (project.sources_dir / "01-example.md").exists()


def test_unpack_rejects_archive_without_tape(tmp_path: Path) -> None:
    bogus = tmp_path / "bogus.mixtape"
    with zipfile.ZipFile(bogus, "w") as zf:
        zf.writestr("rootdir/notes.md", "no tape here")
    with pytest.raises(ValueError):
        unpack(bogus, tmp_path / "out")


def test_pack_custom_out_path(tmp_path: Path) -> None:
    from liner.project import ProjectFolder
    path = _populated_project(tmp_path)
    custom = tmp_path / "custom.mixtape"
    result = pack(ProjectFolder(path), out_path=custom)
    assert result.archive_path == custom
    assert custom.exists()
