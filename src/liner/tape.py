from __future__ import annotations

from pathlib import Path
from typing import Any
from urllib.parse import urlparse

import yaml

from liner.types import JtbdClarification, SourceSpec, Tape

ALLOWED_SOURCE_TYPES = {"youtube", "web", "local_file"}
ALLOWED_PRIORITIES = {"required", "optional"}
ALLOWED_MODES = {"quick", "methodology"}
ALLOWED_RENDER = {"server", "js"}
ALLOWED_KINDS = {"reference", "principle", "prescription", "example"}
ALLOWED_LOCAL_FILE_EXTENSIONS = {".md", ".txt", ".html", ".htm", ".pdf"}
SUPPORTED_TAPE_VERSION = 1


class TapeValidationError(ValueError):
    def __init__(self, field_path: str, message: str) -> None:
        self.field_path = field_path
        super().__init__(f"{field_path}: {message}")


def load_tape(path: Path) -> Tape:
    with path.open("r", encoding="utf-8") as f:
        raw = yaml.safe_load(f)
    if not isinstance(raw, dict):
        raise TapeValidationError("<root>", "tape file must be a YAML mapping")
    return validate_tape(raw)


def validate_tape(raw: dict[str, Any]) -> Tape:
    for required in ("title", "description", "version", "curator", "sources"):
        if required not in raw:
            raise TapeValidationError(required, "required field missing")

    version = raw["version"]
    if not isinstance(version, int):
        raise TapeValidationError("version", "must be an integer")
    if version != SUPPORTED_TAPE_VERSION:
        raise TapeValidationError(
            "version",
            f"unsupported tape version {version}; this build supports version {SUPPORTED_TAPE_VERSION}",
        )

    sources_raw = raw["sources"]
    if not isinstance(sources_raw, list):
        raise TapeValidationError("sources", "must be a list")
    # Empty `sources:` is a legitimate in-progress state — a freshly-replayed
    # folder, a `liner init`-scaffolded project before Phase 7 runs, a tape
    # the curator just opened to edit. Don't reject it at parse time; the
    # non-empty invariant is enforced at compile/share, where it surfaces as
    # an actionable "add sources before compiling" message instead of
    # silently dropping the folder from `liner list`.

    sources: list[SourceSpec] = []
    for i, src in enumerate(sources_raw):
        sources.append(_validate_source(src, f"sources[{i}]"))

    tags_raw = raw.get("tags", [])
    if not isinstance(tags_raw, list) or not all(isinstance(t, str) for t in tags_raw):
        raise TapeValidationError("tags", "must be a list of strings if present")

    mode = raw.get("mode")
    if mode is not None and (not isinstance(mode, str) or mode not in ALLOWED_MODES):
        raise TapeValidationError(
            "mode",
            f"must be one of {sorted(ALLOWED_MODES)} if present, got {mode!r}",
        )

    for optional_str_field in ("jtbd", "methodology_version", "parent"):
        value = raw.get(optional_str_field)
        if value is not None and not isinstance(value, str):
            raise TapeValidationError(optional_str_field, "must be a string if present")

    # jtbd_clarifications is an optional list of {question, answer} pairs from
    # the wizard's elicitation step. Permissive parse: skip malformed entries,
    # collect well-formed ones. Empty (or all-malformed) is the same as absent.
    clarifications_raw = raw.get("jtbd_clarifications")
    clarifications: list[JtbdClarification] = []
    if clarifications_raw is not None:
        if not isinstance(clarifications_raw, list):
            raise TapeValidationError(
                "jtbd_clarifications", "must be a list of {question, answer} mappings"
            )
        for i, entry in enumerate(clarifications_raw):
            if not isinstance(entry, dict):
                raise TapeValidationError(
                    f"jtbd_clarifications[{i}]", "must be a mapping"
                )
            q = entry.get("question")
            a = entry.get("answer")
            if not isinstance(q, str) or not isinstance(a, str):
                raise TapeValidationError(
                    f"jtbd_clarifications[{i}]",
                    "must have string 'question' and 'answer' fields",
                )
            clarifications.append(JtbdClarification(question=q, answer=a))

    return Tape(
        title=str(raw["title"]),
        description=str(raw["description"]),
        version=version,
        curator=str(raw["curator"]),
        sources=tuple(sources),
        tags=tuple(tags_raw),
        created=_optional_str(raw, "created"),
        updated=_optional_str(raw, "updated"),
        license=_optional_str(raw, "license"),
        homepage=_optional_str(raw, "homepage"),
        mode=mode,  # type: ignore[arg-type]
        jtbd=_optional_str(raw, "jtbd"),
        jtbd_clarifications=tuple(clarifications),
        methodology_version=_optional_str(raw, "methodology_version"),
        parent=_optional_str(raw, "parent"),
    )


def _validate_source(src: Any, path_prefix: str) -> SourceSpec:
    if not isinstance(src, dict):
        raise TapeValidationError(path_prefix, "must be a mapping")

    if "type" not in src:
        raise TapeValidationError(f"{path_prefix}.type", "required field missing")
    s_type = src["type"]
    if s_type not in ALLOWED_SOURCE_TYPES:
        raise TapeValidationError(
            f"{path_prefix}.type",
            f"must be one of {sorted(ALLOWED_SOURCE_TYPES)}, got {s_type!r}",
        )

    # Shared optional fields
    priority = src.get("priority", "required")
    if priority not in ALLOWED_PRIORITIES:
        raise TapeValidationError(
            f"{path_prefix}.priority",
            f"must be one of {sorted(ALLOWED_PRIORITIES)}, got {priority!r}",
        )

    note = src.get("note")
    if note is not None and not isinstance(note, str):
        raise TapeValidationError(f"{path_prefix}.note", "must be a string if present")

    section = src.get("section")
    if section is not None and not isinstance(section, str):
        raise TapeValidationError(f"{path_prefix}.section", "must be a string if present")

    kind = src.get("kind")
    if kind is not None:
        if not isinstance(kind, str) or kind not in ALLOWED_KINDS:
            raise TapeValidationError(
                f"{path_prefix}.kind",
                f"must be one of {sorted(ALLOWED_KINDS)} if present, got {kind!r}",
            )

    if s_type == "local_file":
        return _validate_local_file_source(src, path_prefix, note, section, priority, kind)
    return _validate_url_source(src, path_prefix, s_type, note, section, priority, kind)


def _validate_url_source(
    src: dict[str, Any],
    path_prefix: str,
    s_type: str,
    note: str | None,
    section: str | None,
    priority: str,
    kind: str | None,
) -> SourceSpec:
    if "url" not in src:
        raise TapeValidationError(f"{path_prefix}.url", "required field missing")
    url = src["url"]
    if not isinstance(url, str) or not url.strip():
        raise TapeValidationError(f"{path_prefix}.url", "must be a non-empty string")
    parsed = urlparse(url)
    if not parsed.scheme or not parsed.netloc:
        raise TapeValidationError(f"{path_prefix}.url", f"not a valid URL: {url!r}")

    for forbidden in ("path", "citation"):
        if forbidden in src:
            raise TapeValidationError(
                f"{path_prefix}.{forbidden}",
                f"only valid on local_file sources, not {s_type!r}",
            )

    render = src.get("render")
    if render is not None:
        if s_type != "web":
            raise TapeValidationError(
                f"{path_prefix}.render",
                f"only valid on web sources, not {s_type!r}",
            )
        if not isinstance(render, str) or render not in ALLOWED_RENDER:
            raise TapeValidationError(
                f"{path_prefix}.render",
                f"must be one of {sorted(ALLOWED_RENDER)} if present, got {render!r}",
            )

    return SourceSpec(
        type=s_type,  # type: ignore[arg-type]
        url=url,
        note=note,
        section=section,
        priority=priority,  # type: ignore[arg-type]
        render=render,  # type: ignore[arg-type]
        kind=kind,  # type: ignore[arg-type]
    )


def _validate_local_file_source(
    src: dict[str, Any],
    path_prefix: str,
    note: str | None,
    section: str | None,
    priority: str,
    kind: str | None,
) -> SourceSpec:
    if "url" in src:
        raise TapeValidationError(
            f"{path_prefix}.url",
            "local_file sources must use `path`, not `url`",
        )
    if "render" in src:
        raise TapeValidationError(
            f"{path_prefix}.render",
            "render is only valid on web sources",
        )

    for required in ("path", "citation"):
        if required not in src or src[required] is None:
            raise TapeValidationError(
                f"{path_prefix}.{required}", "required field missing for local_file"
            )

    raw_path = src["path"]
    if not isinstance(raw_path, str) or not raw_path.strip():
        raise TapeValidationError(
            f"{path_prefix}.path", "must be a non-empty string"
        )
    path_value = raw_path.strip()

    citation = src["citation"]
    if not isinstance(citation, str) or not citation.strip():
        raise TapeValidationError(
            f"{path_prefix}.citation", "must be a non-empty string"
        )

    # Validate path shape only — file existence is checked at compile time.
    if Path(path_value).is_absolute():
        raise TapeValidationError(
            f"{path_prefix}.path",
            f"must be a relative path under personal/, got absolute path {path_value!r}",
        )
    parts = path_value.replace("\\", "/").split("/")
    if parts[0] != "personal":
        raise TapeValidationError(
            f"{path_prefix}.path",
            f"must start with 'personal/', got {path_value!r}",
        )
    if any(part == ".." for part in parts):
        raise TapeValidationError(
            f"{path_prefix}.path", f"must not contain '..' segments: {path_value!r}"
        )

    ext = Path(path_value).suffix.lower()
    if ext not in ALLOWED_LOCAL_FILE_EXTENSIONS:
        raise TapeValidationError(
            f"{path_prefix}.path",
            f"unsupported extension {ext!r}; allowed: {sorted(ALLOWED_LOCAL_FILE_EXTENSIONS)}",
        )

    return SourceSpec(
        type="local_file",
        url="",
        note=note,
        section=section,
        priority=priority,  # type: ignore[arg-type]
        path=path_value,
        citation=citation.strip(),
        kind=kind,  # type: ignore[arg-type]
    )


def _optional_str(raw: dict[str, Any], key: str) -> str | None:
    if key not in raw or raw[key] is None:
        return None
    return str(raw[key])


STARTER_TAPE = """# liner starter tape — edit the sources and curator notes, then run
# `liner compile <project-folder>` to build the mixtape.
title: My First Mixtape
description: A starter tape with one YouTube video and one article so you can see the output immediately.
version: 1
curator: You
mode: quick
jtbd: Replace this with a single-sentence statement of what you want the AI to help you do.
tags: [example, starter]

sources:
  - type: youtube
    url: https://www.youtube.com/watch?v=aircAruvnKk
    note: "3Blue1Brown intro to neural networks. Famously clear visual explanation."
    section: intro

  - type: web
    url: https://en.wikipedia.org/wiki/Mixtape
    note: "Background on the mixtape as a curation format — liner's namesake."
    section: intro
"""


def write_starter_tape(path: Path, force: bool = False) -> None:
    if path.exists() and not force:
        raise FileExistsError(f"{path} already exists. Use --force to overwrite.")
    path.write_text(STARTER_TAPE, encoding="utf-8")
