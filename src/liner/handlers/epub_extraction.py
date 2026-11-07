"""Read the text of an EPUB in reading order, without trusting the archive."""

from __future__ import annotations

import posixpath
import zipfile
from pathlib import Path
from urllib.parse import unquote
from xml.etree import ElementTree
from xml.parsers import expat

from liner.handlers.base import HandlerHardFailure
from liner.handlers.html_extraction import extract_html_text

MAX_UNPACKED_BYTES = 100 * 1024 * 1024

_XHTML_TYPES = {"application/xhtml+xml", "text/html"}
_CONTAINER = "META-INF/container.xml"


def extract_epub_text(path: Path, identifier: str) -> str:
    try:
        with zipfile.ZipFile(path) as archive:
            parts = _chapters(archive, identifier)
    except (
        zipfile.BadZipFile,
        ElementTree.ParseError,
        expat.ExpatError,
        KeyError,
        ValueError,
        OSError,
    ) as error:
        raise HandlerHardFailure(f"{path} is not a readable EPUB: {error}", identifier) from error
    body = "\n\n".join(part for part in parts if part).strip()
    if not body:
        raise HandlerHardFailure(
            f"{path} produced no text. It may be DRM-protected or image-only.", identifier
        )
    return body


def _chapters(archive: zipfile.ZipFile, identifier: str) -> list[str]:
    if sum(item.file_size for item in archive.infolist()) > MAX_UNPACKED_BYTES:
        raise HandlerHardFailure(
            f"EPUB unpacks to more than {MAX_UNPACKED_BYTES} bytes.", identifier
        )
    rootfile = next(
        (
            element.get("full-path")
            for element in _xml(archive, _CONTAINER).iter()
            if _tag(element) == "rootfile"
        ),
        None,
    )
    if not rootfile:
        raise ValueError(f"{_CONTAINER} names no package file")
    package = _xml(archive, rootfile)
    manifest = {
        element.get("id"): element
        for element in package.iter()
        if _tag(element) == "item" and element.get("href")
    }
    folder = posixpath.dirname(rootfile)
    parts: list[str] = []
    for itemref in package.iter():
        if _tag(itemref) != "itemref":
            continue
        item = manifest.get(itemref.get("idref"))
        if item is None or item.get("media-type") not in _XHTML_TYPES:
            continue
        name = posixpath.normpath(posixpath.join(folder, unquote(item.get("href", ""))))
        if name in archive.namelist():
            parts.append(_chapter_text(archive.read(name).decode("utf-8", errors="replace")))
    return parts


def _chapter_text(html: str) -> str:
    extraction = extract_html_text(html)
    return (extraction.body or extraction.fallback_body or "").strip()


def _xml(archive: zipfile.ZipFile, name: str) -> ElementTree.Element:
    data = archive.read(name)

    def refuse(*_: object) -> None:  # Entity expansion could exhaust memory.
        raise ValueError(f"{name} declares a document type")

    # Parsing, not a byte search, so another encoding can't hide the declaration.
    checker = expat.ParserCreate()
    checker.StartDoctypeDeclHandler = refuse
    checker.Parse(data, True)
    return ElementTree.fromstring(data)


def _tag(element: ElementTree.Element) -> str:
    return str(element.tag).rpartition("}")[2]
