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
    assert "topic/mixtape/personal/note.md" in names


def test_pack_no_personal_excludes_personal(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    path = _populated_project(tmp_path, with_personal=True)
    result = pack(
        ProjectFolder(path),
        options=ShareOptions(include_personal=False),
    )
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert not any(n.startswith("topic/mixtape/personal/") for n in names)
    assert "topic/mixtape/tape.yaml" in names


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
    assert "topic/liner.yaml" in names
    assert "topic/mixtape/tape.yaml" in names
    assert "topic/mixtape/synthesis.md" in names
    assert "topic/mixtape/MIXTAPE.md" in names
    assert any(n.startswith("topic/mixtape/sources/") for n in names)
    assert any(n.startswith("topic/mixtape/working/") for n in names)


def test_pack_minimal_only_includes_tape(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    path = _populated_project(tmp_path)
    result = pack(ProjectFolder(path), options=ShareOptions(minimal=True))
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert names == {"topic/mixtape/tape.yaml"}


def test_pack_excludes_working_and_sources_via_flags(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    path = _populated_project(tmp_path)
    result = pack(
        ProjectFolder(path),
        options=ShareOptions(include_working_notes=False, include_source_content=False),
    )
    with zipfile.ZipFile(result.archive_path) as zf:
        names = set(zf.namelist())
    assert not any(n.startswith("topic/mixtape/sources/") for n in names)
    assert not any(n.startswith("topic/mixtape/working/") for n in names)
    assert "topic/mixtape/tape.yaml" in names


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


def _archive(path: Path, entries: dict[str, str]) -> Path:
    with zipfile.ZipFile(path, "w") as zf:
        for name, text in entries.items():
            zf.writestr(name, text)
    return path


TAPE = "liner/mixtape/tape.yaml"


def test_import_refuses_to_replace_an_existing_folder(tmp_path: Path) -> None:
    archive = _archive(tmp_path / "a.mixtape", {TAPE: "new"})
    tape = tmp_path / "out" / TAPE
    tape.parent.mkdir(parents=True)
    tape.write_text("mine", encoding="utf-8")
    with pytest.raises(FileExistsError, match="--overwrite"):
        unpack(archive, tmp_path / "out")
    assert tape.read_text(encoding="utf-8") == "mine"
    unpack(archive, tmp_path / "out", overwrite=True)
    assert tape.read_text(encoding="utf-8") == "new"


@pytest.mark.parametrize(
    "name",
    ["liner/../../evil.txt", "/etc/evil.txt", "liner\\..\\evil.txt", "C:/evil.txt"],
)
def test_import_refuses_unsafe_paths(tmp_path: Path, name: str) -> None:
    archive = _archive(tmp_path / "a.mixtape", {TAPE: "t", name: "x"})
    with pytest.raises(ValueError, match="unsafe path|single top-level"):
        unpack(archive, tmp_path / "out")
    assert not (tmp_path / "out" / "liner").exists()


def test_import_refuses_an_oversized_archive(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setattr("liner.share.MAX_ARCHIVE_BYTES", 3)
    archive = _archive(tmp_path / "a.mixtape", {TAPE: "too long"})
    with pytest.raises(ValueError, match="too large"):
        unpack(archive, tmp_path / "out")


def test_import_drops_transcripts_but_keeps_receipts(tmp_path: Path) -> None:
    archive = _archive(
        tmp_path / "a.mixtape",
        {
            TAPE: "t",
            "liner/mixtape/.liner-runs/run/agent.JSONL": "{}",
            "liner/mixtape/.LINER-RUNS/run/other.jsonl": "{}",
            "liner/mixtape/.liner-runs/maintenance/receipt.json": "{}",
        },
    )
    unpack(archive, tmp_path / "out")
    runs = tmp_path / "out/liner/mixtape/.liner-runs"
    assert not (runs / "run/agent.JSONL").exists()
    assert not list((tmp_path / "out").rglob("*.jsonl"))
    assert not list((tmp_path / "out").rglob("*.JSONL"))
    assert (runs / "maintenance/receipt.json").exists()


def test_share_leaves_transcripts_behind(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    folder = tmp_path / "liner"
    init_project(folder)
    project = ProjectFolder(folder)
    log = project.corpus_path / ".liner-runs" / "run" / "agent.jsonl"
    log.parent.mkdir(parents=True)
    log.write_text("{}", encoding="utf-8")
    with zipfile.ZipFile(pack(project).archive_path) as zf:
        assert not [name for name in zf.namelist() if name.endswith(".jsonl")]


def test_import_accepts_a_colon_in_a_folder_name(tmp_path: Path) -> None:
    archive = _archive(tmp_path / "a.mixtape", {"notes: v2/mixtape/tape.yaml": "t"})
    assert unpack(archive, tmp_path / "out").path.name == "notes: v2"


def test_share_leaves_build_prompts_and_review_drafts_behind(tmp_path: Path) -> None:
    from liner.project import ProjectFolder

    folder = tmp_path / "liner"
    init_project(folder)
    project = ProjectFolder(folder)
    for relative in ("steps/framing.md", "review/SKILL.md", "fetched/page.md"):
        path = project.corpus_path / ".liner-runs" / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("text", encoding="utf-8")
    with zipfile.ZipFile(pack(project).archive_path) as zf:
        names = zf.namelist()
    assert not [name for name in names if "/.liner-runs/steps/" in name]
    assert not [name for name in names if "/.liner-runs/review/" in name]
