from __future__ import annotations

import zipfile
from collections.abc import Iterator
from dataclasses import dataclass
from pathlib import Path, PurePosixPath, PureWindowsPath

from liner.project import ProjectFolder

MAX_ARCHIVE_ENTRIES = 20_000
MAX_ARCHIVE_BYTES = 1024 * 1024 * 1024


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


def unpack(archive: Path, destination: Path, *, overwrite: bool = False) -> ProjectFolder:
    """Extract a `.mixtape` archive into `destination` and return the project folder.

    The archive must contain exactly one top-level directory holding a tape.yaml.
    An archive can come from anyone, so it can't escape `destination`, replace an
    existing folder unless asked, or bring agent transcripts that compile would trust.
    """
    if not archive.exists():
        raise FileNotFoundError(f"Archive not found: {archive}")

    destination.mkdir(parents=True, exist_ok=True)

    with zipfile.ZipFile(archive, "r") as zf:
        names = zf.namelist()
        _check_archive(archive, zf)
        top_levels = {n.split("/", 1)[0] for n in names if n}
        if len(top_levels) != 1:
            raise ValueError(
                f"Expected a single top-level directory in {archive}; got {sorted(top_levels)}"
            )
        top = top_levels.pop()
        if (destination / top).exists() and not overwrite:
            raise FileExistsError(
                f"{destination / top} already exists. Move it, or pass --overwrite to replace "
                "matching files."
            )
        zf.extractall(destination, members=[n for n in names if not _is_transcript(n)])

    project_path = destination / top
    project = ProjectFolder(project_path)
    if not project.tape_path.exists():
        raise ValueError(f"No tape.yaml found inside extracted folder {project_path}")
    return project


def _check_archive(archive: Path, zf: zipfile.ZipFile) -> None:
    infos = zf.infolist()
    if len(infos) > MAX_ARCHIVE_ENTRIES:
        raise ValueError(f"{archive} has too many files to import.")
    if sum(info.file_size for info in infos) > MAX_ARCHIVE_BYTES:
        raise ValueError(f"{archive} is too large to import.")
    for info in infos:
        name = info.filename
        path = PurePosixPath(name)
        windows = PureWindowsPath(name)
        if path.is_absolute() or windows.is_absolute() or windows.drive or ".." in windows.parts:
            raise ValueError(f"{archive} has an unsafe path: {name!r}")


def _is_transcript(name: str) -> bool:
    """Agent logs are private, and compile reads them as source text, so they never travel.

    Build prompts and review drafts stay home too: prompts hold the person's answers, and
    a draft `SKILL.md` in a shared liner could be mistaken for the real one.
    """
    path = PurePosixPath(name)
    parts = [part.casefold() for part in path.parts]
    if ".liner-runs" not in parts:
        return False
    after = parts[parts.index(".liner-runs") + 1 :]
    return path.suffix.casefold() == ".jsonl" or bool(after and after[0] in {"steps", "review"})


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
        if _is_transcript(rel.as_posix()) or rel.name == "empirical-tests-pending.yaml":
            continue
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
