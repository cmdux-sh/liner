from __future__ import annotations

from pathlib import Path

import pytest

from liner.handlers.base import HandlerHardFailure
from liner.handlers.local_file import MAX_FILE_BYTES, LocalFileHandler
from liner.project import init_project
from liner.types import SourceSpec


def _spec(path: str, citation: str = "Test citation") -> SourceSpec:
    return SourceSpec(type="local_file", url="", path=path, citation=citation)


def _setup_project(tmp_path: Path) -> tuple[Path, LocalFileHandler]:
    project = init_project(tmp_path / "p")
    project.personal_dir.mkdir(parents=True, exist_ok=True)
    return project.path, LocalFileHandler(project)


def test_md_file_extraction(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    (path / "personal" / "note.md").write_text(
        "# Hello\n\nThis is a markdown note.\n", encoding="utf-8"
    )
    content = handler.fetch(_spec("personal/note.md"))
    assert content.title == "Test citation"
    assert "markdown note" in content.body
    assert content.metadata.get("kind") == "md"


def test_txt_file_extraction(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    (path / "personal" / "n.txt").write_text("plain text content", encoding="utf-8")
    content = handler.fetch(_spec("personal/n.txt"))
    assert content.body == "plain text content"


def test_html_file_extraction(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    html = (
        "<html><head><title>Saved Page</title></head><body>"
        "<article><h1>Saved</h1>"
        + "<p>"
        + ("Substantial article body. " * 20)
        + "</p></article></body></html>"
    )
    (path / "personal" / "saved.html").write_text(html, encoding="utf-8")
    content = handler.fetch(_spec("personal/saved.html"))
    assert "Substantial article body" in content.body


def test_html_js_stub_rejected(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    stub_html = (
        "<html><body><noscript><h1>This page requires JavaScript.</h1>"
        "<p>Please turn on JavaScript and refresh.</p></noscript>"
        "<div id='root'></div></body></html>"
    )
    (path / "personal" / "stub.html").write_text(stub_html, encoding="utf-8")
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(_spec("personal/stub.html"))
    assert "JavaScript" in str(exc.value) or "noscript" in str(exc.value).lower()


def test_unsupported_extension_rejected(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    (path / "personal" / "bin.exe").write_bytes(b"\x00\x01\x02")
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(_spec("personal/bin.exe"))
    assert "Unsupported" in str(exc.value)


def test_missing_file_hard_fails(tmp_path: Path) -> None:
    _, handler = _setup_project(tmp_path)
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(_spec("personal/missing.md"))
    assert "does not exist" in str(exc.value)


def test_file_too_large_rejected(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    big = path / "personal" / "big.txt"
    big.write_bytes(b"x" * (MAX_FILE_BYTES + 1))
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(_spec("personal/big.txt"))
    assert "bytes" in str(exc.value)


def test_path_outside_personal_rejected(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    # Create a file outside personal/
    outside = path / "synthesis.md"  # synthesis.md exists from init
    assert outside.exists()
    # Use path traversal — validation normally catches this but exercise the
    # handler's defense-in-depth via a path that bypasses validation by going
    # through a non-existent symlink-like trick. The simplest test: a path
    # that would resolve outside personal_dir.
    spec = SourceSpec(
        type="local_file",
        url="",
        path="personal/../synthesis.md",
        citation="bypass",
    )
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(spec)
    assert "outside" in str(exc.value)


def test_missing_citation_rejected(tmp_path: Path) -> None:
    path, handler = _setup_project(tmp_path)
    (path / "personal" / "note.md").write_text("hi", encoding="utf-8")
    spec = SourceSpec(type="local_file", url="", path="personal/note.md", citation="")
    with pytest.raises(HandlerHardFailure) as exc:
        handler.fetch(spec)
    assert "citation" in str(exc.value)
