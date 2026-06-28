from __future__ import annotations

import zipfile
from collections.abc import Iterator
from dataclasses import dataclass
from pathlib import Path

from liner.project import ProjectFolder


@dataclass(frozen=True, slots=True)
class ShareOptions:
    include_working_notes: bool = True
    include_source_content: bool = True
    include_personal: bool = True
    minimal: bool = False  # tape.yaml only


@dataclass(frozen=True, slots=True)
class ShareResult:
    archive_path: Path
    entry_count: int


_DEFAULT_OPTIONS = ShareOptions()


def pack(
    project: ProjectFolder,
    *,
    out_path: Path | None = None,
    options: ShareOptions = _DEFAULT_OPTIONS,
) -> ShareResult:
    """Zip a project folder into a `.mixtape` archive.

    The archive contains the project folder at its root (entries are prefixed
    with the folder name) so unzipping yields a self-contained project folder.
    """
    if not project.path.is_dir():
        raise FileNotFoundError(f"Project folder does not exist: {project.path}")
    if not project.tape_path.exists():
        raise FileNotFoundError(
            f"No tape.yaml in {project.path}; cannot share. Run `liner init` first."
        )

    if out_path is not None:
        archive_path = out_path
    else:
        archive_path = project.path.parent / f"{project.path.name}.mixtape"

    entries = list(_iter_entries(project, options))

    archive_path.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(archive_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for source, arcname in entries:
            zf.write(source, arcname)

    return ShareResult(archive_path=archive_path, entry_count=len(entries))


def unpack(archive: Path, destination: Path) -> ProjectFolder:
    """Extract a `.mixtape` archive into `destination` and return the project folder.

    The archive must contain exactly one top-level directory holding a tape.yaml.
    """
    if not archive.exists():
        raise FileNotFoundError(f"Archive not found: {archive}")

    destination.mkdir(parents=True, exist_ok=True)

    with zipfile.ZipFile(archive, "r") as zf:
        names = zf.namelist()
        top_levels = {n.split("/", 1)[0] for n in names if n}
        if len(top_levels) != 1:
            raise ValueError(
                f"Expected a single top-level directory in {archive}; got {sorted(top_levels)}"
            )
        zf.extractall(destination)

    top = top_levels.pop()
    project_path = destination / top
    project = ProjectFolder(project_path)
    if not project.tape_path.exists():
        raise ValueError(f"No tape.yaml found inside extracted folder {project_path}")
    return project


def _iter_entries(
    project: ProjectFolder, options: ShareOptions
) -> Iterator[tuple[Path, str]]:
    root = project.path
    root_name = root.name

    if options.minimal:
        yield project.tape_path, f"{root_name}/{project.tape_path.relative_to(root).as_posix()}"
        return

    for path in sorted(root.rglob("*")):
        if not path.is_file():
            continue
        rel = path.relative_to(root)
        first = rel.parts[0]
        if _is_sources_entry(rel) and not options.include_source_content:
            continue
        if _is_working_entry(rel) and not options.include_working_notes:
            continue
        if _is_personal_entry(rel) and not options.include_personal:
            continue
        arcname = f"{root_name}/{rel.as_posix()}"
        yield path, arcname


def _is_sources_entry(rel: Path) -> bool:
    parts = rel.parts
    return parts[:1] == ("sources",) or parts[:2] == ("mixtape", "sources")


def _is_working_entry(rel: Path) -> bool:
    parts = rel.parts
    return parts[:1] == ("working",) or parts[:2] == ("mixtape", "working")


def _is_personal_entry(rel: Path) -> bool:
    parts = rel.parts
    return (
        parts[:1] in {("personal",), ("local-sources",)}
        or parts[:2] in {("mixtape", "personal"), ("mixtape", "local-sources")}
    )
