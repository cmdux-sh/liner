"""Shared PDF text extraction for local and remote sources."""
from __future__ import annotations

from io import BytesIO
from pathlib import Path
from typing import BinaryIO

from liner.handlers.base import HandlerHardFailure


def extract_pdf_text(source: Path | bytes | BinaryIO, identifier: str) -> str:
    try:
        import pdfplumber
    except ImportError as e:  # pragma: no cover - pdfplumber is a core dep
        raise HandlerHardFailure(
            "pdfplumber is required for PDF extraction but is not installed.",
            identifier,
        ) from e

    pdf_source: Path | BinaryIO = BytesIO(source) if isinstance(source, bytes) else source

    parts: list[str] = []
    try:
        with pdfplumber.open(pdf_source) as pdf:
            for page in pdf.pages:
                text = page.extract_text() or ""
                if text.strip():
                    parts.append(text)
    except Exception as e:
        raise HandlerHardFailure(
            f"Failed to extract text from PDF {identifier}: {e}",
            identifier,
        ) from e

    body = "\n\n".join(parts).strip()
    if not body:
        raise HandlerHardFailure(
            f"PDF {identifier} produced no extractable text - it may be image-only or encrypted.",
            identifier,
        )
    return body


__all__ = ["extract_pdf_text"]
