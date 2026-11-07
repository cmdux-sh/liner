"""Cite another liner in the same library.

`liner://<slug>` compiles its reviewed synthesis. `liner://<slug>/<source-id>` copies
one of its sources' saved text, with the original address and where it came from.
"""

from __future__ import annotations

from datetime import UTC, datetime

from liner.citations import compiled_source_files
from liner.handlers.base import HandlerHardFailure
from liner.output.mixtape import compiled_source_body
from liner.project import SYNTHESIS_PLACEHOLDER, ProjectFolder, read_liner_metadata
from liner.tape import TapeValidationError, liner_locator, load_tape
from liner.types import SourceContent, SourceSpec

LOCAL_ORIGINS = {"local_file", "skill"}


class LinerHandler:
    def __init__(self, project: ProjectFolder) -> None:
        self._project = project

    def fetch(self, spec: SourceSpec) -> SourceContent:
        slug, source_id = liner_locator(spec.url)
        cited = ProjectFolder(self._project.path.parent / slug)
        if cited.path.resolve() == self._project.path.resolve():
            raise HandlerHardFailure("A liner can't cite itself.", spec.url)
        if source_id is not None:
            return _reused_source(cited, slug, source_id, spec)
        if not cited.mixtape_path.is_file() or not cited.synthesis_path.is_file():
            raise HandlerHardFailure(
                f"No finished liner named {slug} next to {self._project.path.name}.", spec.url
            )
        try:
            title = load_tape(cited.tape_path).title
            body = cited.synthesis_path.read_text(encoding="utf-8", errors="replace").strip()
        except (OSError, TapeValidationError) as error:
            raise HandlerHardFailure(f"Could not read liner {slug}: {error}", spec.url) from error
        if not body or body == SYNTHESIS_PLACEHOLDER.strip() or "Replace this placeholder" in body:
            raise HandlerHardFailure(f"Liner {slug} has no synthesis yet.", spec.url)
        return SourceContent(
            title=title,
            url=spec.url,
            body=body,
            fetched_at=datetime.now(UTC).isoformat(),
            metadata={
                "extraction": "liner",
                "liner_id": read_liner_metadata(cited).get("id"),
            },
        )


def _reused_source(
    cited: ProjectFolder, slug: str, source_id: str, spec: SourceSpec
) -> SourceContent:
    """One source's saved text from another liner, without fetching it again."""
    try:
        sources = load_tape(cited.tape_path).sources
    except (OSError, TapeValidationError) as error:
        raise HandlerHardFailure(f"Could not read liner {slug}: {error}", spec.url) from error
    origin = next((source for source in sources if source.source_id == source_id), None)
    if origin is None:
        raise HandlerHardFailure(
            f"Liner {slug} has no source {source_id}. Run `liner find-sources` for current "
            "reuse addresses.",
            spec.url,
        )
    if origin.type == "liner":
        raise HandlerHardFailure(
            f"That source in {slug} is itself another liner; cite {origin.url} instead.", spec.url
        )
    locator = origin.url or origin.path or ""
    compiled = compiled_source_files(cited).get(locator)
    try:
        text = compiled.read_text(encoding="utf-8", errors="replace") if compiled else None
    except OSError as error:
        raise HandlerHardFailure(
            f"Could not read that source in {slug}: {error}", spec.url
        ) from error
    body = compiled_source_body(text, origin) if text is not None else None
    if not body:
        raise HandlerHardFailure(
            f"Liner {slug} has no full saved text for that source, so it can't be reused. "
            "Add its address instead, or compile that liner first.",
            spec.url,
        )
    title = (
        next(
            (
                line.removeprefix("# ").strip()
                for line in text.splitlines()
                if line.startswith("# ")
            ),
            locator,
        )
        if text
        else locator
    )
    local = origin.type in LOCAL_ORIGINS
    return SourceContent(
        title=title,
        url=spec.url,
        body=body,
        fetched_at=datetime.now(UTC).isoformat(),
        metadata={
            "extraction": "liner_source",
            "origin_liner": slug,
            "origin_source_id": source_id,
            "origin_type": origin.type,
            "origin_url": origin.url or None,
            "origin_citation": origin.url or origin.citation or None,
            "local_origin": local,
        },
    )


def reused_local_sources(project: ProjectFolder, sources: tuple[SourceSpec, ...]) -> list[str]:
    """Reuse addresses whose original is a local file or skill in the other liner.

    The compiled copy records this, so it holds even after the other liner moves.
    """
    recorded = _recorded_local_origins(project)
    local: list[str] = []
    for spec in sources:
        if spec.type != "liner":
            continue
        slug, source_id = liner_locator(spec.url)
        if source_id is None:
            continue
        if spec.url in recorded:
            local.append(spec.url)
            continue
        try:
            origins = load_tape(ProjectFolder(project.path.parent / slug).tape_path).sources
        except (OSError, TapeValidationError):
            continue
        if any(o.source_id == source_id and o.type in LOCAL_ORIGINS for o in origins):
            local.append(spec.url)
    return local


def _recorded_local_origins(project: ProjectFolder) -> set[str]:
    found: set[str] = set()
    for locator, path in compiled_source_files(project).items():
        if not locator.startswith("liner://"):
            continue
        try:
            header = path.read_text(encoding="utf-8", errors="replace").split("**Fetched:** ")[0]
        except OSError:
            continue
        if "**Local origin:** yes" in header:
            found.add(locator)
    return found
