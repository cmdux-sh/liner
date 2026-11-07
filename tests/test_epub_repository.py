from __future__ import annotations

import zipfile
from pathlib import Path

import httpx
import pytest

from liner.handlers.base import HandlerHardFailure
from liner.handlers.epub_extraction import MAX_UNPACKED_BYTES, _xml, extract_epub_text
from liner.handlers.repository import RepositoryHandler, parse_repository_url
from liner.tape import TapeValidationError, validate_tape
from liner.types import SourceSpec

CONTAINER = (
    '<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles>'
    '<rootfile full-path="OEBPS/book.opf" media-type="application/oebps-package+xml"/>'
    "</rootfiles></container>"
)
PACKAGE = (
    '<package xmlns="http://www.idpf.org/2007/opf"><manifest>'
    '<item id="b" href="ch%202.xhtml" media-type="application/xhtml+xml"/>'
    '<item id="a" href="text/ch1.xhtml" media-type="application/xhtml+xml"/>'
    '<item id="c" href="cover.png" media-type="image/png"/>'
    '</manifest><spine><itemref idref="a"/><itemref idref="c"/><itemref idref="b"/></spine>'
    "</package>"
)
PROSE = " Thumbs are imprecise, so targets should stay generous on every phone screen."


def chapter(text: str) -> str:
    paragraphs = "".join(f"<p>{text}{PROSE}</p>" for _ in range(3))
    return f"<html><body><article><h1>{text}</h1>{paragraphs}</article></body></html>"


def make_epub(path: Path, *, container: str = CONTAINER, package: str = PACKAGE) -> Path:
    with zipfile.ZipFile(path, "w") as archive:
        archive.writestr("mimetype", "application/epub+zip")
        archive.writestr("META-INF/container.xml", container)
        archive.writestr("OEBPS/book.opf", package)
        archive.writestr("OEBPS/text/ch1.xhtml", chapter("First chapter"))
        archive.writestr("OEBPS/ch 2.xhtml", chapter("Second chapter"))
    return path


def test_epub_reads_chapters_in_spine_order(tmp_path: Path) -> None:
    body = extract_epub_text(make_epub(tmp_path / "b.epub"), "id")

    assert body.index("First chapter") < body.index("Second chapter")


def test_epub_rejects_entity_declarations(tmp_path: Path) -> None:
    hostile = '<!DOCTYPE x [<!ENTITY a "aaaa">]>' + CONTAINER
    with pytest.raises(HandlerHardFailure, match="not a readable EPUB"):
        extract_epub_text(make_epub(tmp_path / "b.epub", container=hostile), "id")


def test_epub_rejects_entities_hidden_by_another_encoding(tmp_path: Path) -> None:
    hostile = '<?xml version="1.0" encoding="UTF-16"?><!DOCTYPE x [<!ENTITY a "a">]><a>&a;</a>'
    path = make_epub(tmp_path / "b.epub")
    with zipfile.ZipFile(path, "a") as archive:
        archive.writestr("OEBPS/hostile.opf", hostile.encode("utf-16"))
    with zipfile.ZipFile(path) as archive, pytest.raises(ValueError, match="document type"):
        _xml(archive, "OEBPS/hostile.opf")


def test_epub_rejects_a_file_that_is_not_one(tmp_path: Path) -> None:
    path = tmp_path / "b.epub"
    path.write_bytes(b"not a zip")
    with pytest.raises(HandlerHardFailure, match="not a readable EPUB"):
        extract_epub_text(path, "id")


def test_epub_with_no_text_says_why(tmp_path: Path) -> None:
    empty = PACKAGE.replace('<itemref idref="a"/>', "").replace('<itemref idref="b"/>', "")
    with pytest.raises(HandlerHardFailure, match="DRM"):
        extract_epub_text(make_epub(tmp_path / "b.epub", package=empty), "id")


def test_epub_refuses_a_zip_bomb(tmp_path: Path) -> None:
    path = make_epub(tmp_path / "b.epub")
    with zipfile.ZipFile(path, "a", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("OEBPS/big.bin", b"\0" * (MAX_UNPACKED_BYTES + 1))
    with pytest.raises(HandlerHardFailure, match="unpacks to more"):
        extract_epub_text(path, "id")


@pytest.mark.parametrize(
    ("url", "expected"),
    [
        ("https://github.com/o/r", ("o", "r", None, "")),
        ("https://github.com/o/r.git", ("o", "r", None, "")),
        ("https://github.com/o/r/tree/v2/docs/api", ("o", "r", "v2", "docs/api")),
    ],
)
def test_parse_repository_url(url: str, expected: tuple[str, str, str | None, str]) -> None:
    assert parse_repository_url(url) == expected


@pytest.mark.parametrize(
    "url", ["https://github.com/o", "https://github.com/o/", "https://gitlab.com/o/r"]
)
def test_tape_rejects_other_repository_urls(url: str) -> None:
    raw = {
        "title": "T",
        "description": "D",
        "version": 1,
        "curator": "C",
        "sources": [{"type": "repository", "url": url}],
    }
    with pytest.raises(TapeValidationError):
        validate_tape(raw)


TREE = [
    {"path": "docs/guide.md", "type": "blob", "size": 10},
    {"path": "README.md", "type": "blob", "size": 10},
    {"path": "docs/huge.md", "type": "blob", "size": 10**7},
    {"path": "src/app.py", "type": "blob", "size": 10},
    {"path": "node_modules/x/README.md", "type": "blob", "size": 10},
    {"path": ".github/SECURITY.md", "type": "blob", "size": 10},
    {"path": "docs", "type": "tree"},
]


def github(request: httpx.Request) -> httpx.Response:
    url = str(request.url)
    if url.endswith("/repos/o/r"):
        return httpx.Response(200, json={"default_branch": "trunk"})
    if url.endswith("/commits/trunk"):
        return httpx.Response(200, json={"sha": "abc123"})
    if "/git/trees/abc123" in url:
        return httpx.Response(200, json={"tree": TREE})
    if url.startswith("https://raw.githubusercontent.com/o/r/abc123/"):
        return httpx.Response(200, text=f"Text of {url.rsplit('/', 1)[1]}\n")
    return httpx.Response(404)


def handler_with(transport: httpx.MockTransport) -> RepositoryHandler:
    handler = RepositoryHandler()
    handler._client = httpx.Client(transport=transport)
    return handler


def test_repository_snapshots_documents_at_one_commit() -> None:
    spec = SourceSpec(type="repository", url="https://github.com/o/r")

    content = handler_with(httpx.MockTransport(github)).fetch(spec)

    assert content.title == "o/r"
    assert content.metadata["commit"] == "abc123"
    assert content.metadata["documents"] == "2 of 3"
    assert content.body.index("# README.md") < content.body.index("# docs/guide.md")
    assert "app.py" not in content.body and "SECURITY" not in content.body


def test_a_cut_short_listing_is_recorded() -> None:
    def truncated(request: httpx.Request) -> httpx.Response:
        if "/git/trees/" in str(request.url):
            return httpx.Response(200, json={"tree": TREE, "truncated": True})
        return github(request)

    spec = SourceSpec(type="repository", url="https://github.com/o/r")

    content = handler_with(httpx.MockTransport(truncated)).fetch(spec)

    assert content.metadata["documents"] == "2 of at least 3"


def test_a_folder_url_reads_only_that_folder() -> None:
    def pinned(request: httpx.Request) -> httpx.Response:
        if str(request.url).endswith("/commits/v2"):
            return httpx.Response(200, json={"sha": "abc123"})
        return github(request)

    spec = SourceSpec(type="repository", url="https://github.com/o/r/tree/v2/docs")

    content = handler_with(httpx.MockTransport(pinned)).fetch(spec)

    assert "README" not in content.body and "guide.md" in content.body


def test_the_rate_limit_is_explained() -> None:
    spec = SourceSpec(type="repository", url="https://github.com/o/r")
    handler = handler_with(httpx.MockTransport(lambda _: httpx.Response(403)))

    with pytest.raises(HandlerHardFailure, match="hourly limit"):
        handler.fetch(spec)
