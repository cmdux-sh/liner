"""Local-file source handler.

Resolves `local_file` source paths against the project's `personal/` directory
and extracts text by extension: `.md`/`.txt` as-is, `.html`/`.htm` via the
shared trafilatura pipeline, `.pdf` via pdfplumber.
"""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

from liner.handlers.base import HandlerHardFailure
from liner.handlers.html_extraction import (
    extract_html_text,
    looks_like_js_stub,
)
from liner.handlers.pdf_extraction import extract_pdf_text
from liner.project import ProjectFolder
from liner.types import SourceContent, SourceSpec

# 10MB cap; keep docs/mixtape-format.md in sync.
MAX_FILE_BYTES = 10 * 1024 * 1024

ALLOWED_EXTENSIONS = {".md", ".txt", ".html", ".htm", ".pdf"}


class LocalFileHandler:
    """Reads a local file from `<project>/personal/<path>` and extracts text."""

    def __init__(self, project: ProjectFolder) -> None:
        self._project = project

    def fetch(self, spec: SourceSpec) -> SourceContent:
        if spec.type != "local_file":
            raise HandlerHardFailure(
                f"LocalFileHandler can only handle local_file sources, got {spec.type!r}",
                spec.url or spec.path or "<unknown>",
            )
        if spec.path is None or not spec.path.strip():
            raise HandlerHardFailure(
                "local_file source is missing `path`",
                spec.url or "<missing path>",
            )
        if spec.citation is None or not spec.citation.strip():
            raise HandlerHardFailure(
                "local_file source is missing `citation`",
                spec.path or "<missing citation>",
            )

        resolved = self._resolve_path(spec.path)
        file_url = f"file://{resolved}"

        if not resolved.exists():
            raise HandlerHardFailure(f"local_file path does not exist: {resolved}", file_url)
        if not resolved.is_file():
            raise HandlerHardFailure(f"local_file path is not a regular file: {resolved}", file_url)

        size = resolved.stat().st_size
        if size > MAX_FILE_BYTES:
            raise HandlerHardFailure(
                f"{resolved} is {size} bytes; max allowed is {MAX_FILE_BYTES}.",
                file_url,
            )

        ext = resolved.suffix.lower()
        if ext not in ALLOWED_EXTENSIONS:
            raise HandlerHardFailure(
                f"Unsupported local_file extension {ext!r}. Allowed: {sorted(ALLOWED_EXTENSIONS)}",
                file_url,
            )

        body = self._extract(resolved, ext, file_url)

        return SourceContent(
            title=spec.citation,
            url=file_url,
            body=body,
            fetched_at=datetime.now(UTC).isoformat(),
            author=None,
            published_at=None,
            metadata={"extraction": "local_file", "kind": ext.lstrip("."), "size_bytes": size},
        )

    # --- Internals ---------------------------------------------------------

    def _resolve_path(self, path_value: str) -> Path:
        # Validation already enforced that path starts with personal/ and has
        # no `..` segments. Double-check the resolved path stays inside
        # personal_dir as defense in depth.
        candidate = (self._project.path / path_value).resolve()
        personal_root = self._project.personal_dir.resolve()
        try:
            candidate.relative_to(personal_root)
        except ValueError as e:
            raise HandlerHardFailure(
                f"local_file path {path_value!r} resolves outside personal/: {candidate}",
                f"file://{candidate}",
            ) from e
        return candidate

    def _extract(self, path: Path, ext: str, file_url: str) -> str:
        if ext in (".md", ".txt"):
            return path.read_text(encoding="utf-8", errors="replace").rstrip()
        if ext in (".html", ".htm"):
            html = path.read_text(encoding="utf-8", errors="replace")
            extraction = extract_html_text(html)
            body = extraction.body
            if body is not None and looks_like_js_stub(body):
                raise HandlerHardFailure(
                    f"{path} appears to be a JS-required noscript stub. Save the page from "
                    "Reader Mode or as a single-file HTML after the page has rendered.",
                    file_url,
                )
            if body is None or not body.strip():
                fallback = extraction.fallback_body
                if not fallback.strip():
                    raise HandlerHardFailure(f"Could not extract any text from {path}.", file_url)
                return fallback
            return body
        if ext == ".pdf":
            return extract_pdf_text(path, file_url)
        # Should be unreachable — guard above filters extensions.
        raise HandlerHardFailure(f"Unsupported extension {ext!r}", file_url)


__all__ = ["LocalFileHandler", "ALLOWED_EXTENSIONS", "MAX_FILE_BYTES"]
