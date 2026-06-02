"""Tests for the Playwright-backed WebJsHandler.

The handler module is importable without Playwright installed, but the
behavioral tests are skipped when the optional dependency is missing.
"""
from __future__ import annotations

import pytest

from liner.config import FetchConfig
from liner.handlers.web_js import (
    PLAYWRIGHT_AVAILABLE,
    MissingExtraError,
    WebJsHandler,
)
from liner.types import SourceSpec


def test_missing_extra_error_raised_without_playwright(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """When the extra isn't installed, constructor surfaces a clear install hint."""
    import liner.handlers.web_js as web_js_mod

    monkeypatch.setattr(web_js_mod, "PLAYWRIGHT_AVAILABLE", False)
    with pytest.raises(MissingExtraError) as exc:
        WebJsHandler(FetchConfig())
    assert "liner setup-js" in str(exc.value)


def test_missing_browser_error_helper_recognizes_playwright_text() -> None:
    from liner.handlers.web_js import _is_missing_browser_error

    msg1 = "Executable doesn't exist at /Users/x/Library/Caches/ms-playwright/chromium-1234/chrome"
    msg2 = "Looks like Playwright was just installed. Please run: playwright install"
    msg3 = "some random network error"
    assert _is_missing_browser_error(RuntimeError(msg1))
    assert _is_missing_browser_error(RuntimeError(msg2))
    assert not _is_missing_browser_error(RuntimeError(msg3))


def test_pdf_url_helper_handles_query_strings() -> None:
    from liner.handlers.web_js import _looks_like_pdf_url

    assert _looks_like_pdf_url("https://example.com/report.pdf")
    assert _looks_like_pdf_url("https://example.com/report.pdf?type=standard")
    assert not _looks_like_pdf_url("https://example.com/report")


def test_download_starting_helper_recognizes_playwright_text() -> None:
    from liner.handlers.web_js import _is_download_starting_error

    assert _is_download_starting_error(RuntimeError("Page.goto: Download is starting"))
    assert not _is_download_starting_error(RuntimeError("Navigation timeout"))


def test_content_from_pdf_download_extracts_downloaded_pdf(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path,
) -> None:
    from liner.handlers.web_js import _content_from_pdf_download

    class FakeDownload:
        suggested_filename = "resume-guide.pdf"

        def __init__(self, path) -> None:
            self._path = path

        def path(self) -> str:
            return str(self._path)

    pdf_path = tmp_path / "download.pdf"
    pdf_path.write_bytes(b"%PDF-1.7 fake")
    body = "Downloaded PDF content. " * 20
    monkeypatch.setattr("liner.handlers.web_js.extract_pdf_text", lambda *_: body)

    content = _content_from_pdf_download(
        SourceSpec(
            type="web",
            url="https://example.com/resume-guide.pdf?type=standard",
            render="js",
        ),
        FakeDownload(pdf_path),
    )

    assert content.title == "resume-guide.pdf"
    assert content.body == body
    assert content.metadata["extraction"] == "playwright_pdf_download"
    assert content.metadata["suggested_filename"] == "resume-guide.pdf"
    assert content.metadata["size_bytes"] == len(b"%PDF-1.7 fake")


@pytest.mark.skipif(
    not PLAYWRIGHT_AVAILABLE,
    reason="Playwright not installed; install with `pipx install 'linersh[js]'`",
)
def test_real_playwright_extraction() -> None:  # pragma: no cover - opt-in
    """Smoke test against a static page if Playwright is actually available."""
    from liner.types import SourceSpec

    with WebJsHandler(FetchConfig()) as handler:
        spec = SourceSpec(
            type="web",
            url="https://example.com",
            render="js",
        )
        content = handler.fetch(spec)
        assert content.body
        assert "Example Domain" in content.body or len(content.body) > 0
