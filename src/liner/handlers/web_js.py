"""Playwright-backed handler for JavaScript-rendered web pages.

Opt in with `liner setup-js`. When Playwright is not installed, importing
this module still works — the constructor raises a clear `MissingExtraError`
when actually used.
"""

from __future__ import annotations

import contextlib
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlparse

from liner.config import FetchConfig
from liner.handlers.base import HandlerHardFailure
from liner.handlers.html_extraction import (
    MIN_USEFUL_BODY_CHARS,
    extract_html_text,
    looks_like_bot_challenge,
    looks_like_cookie_notice_only,
    looks_like_js_stub,
)
from liner.handlers.pdf_extraction import extract_pdf_text
from liner.playwright_env import configure_frozen_playwright_cache
from liner.types import SourceContent, SourceSpec

configure_frozen_playwright_cache()

try:
    from playwright.sync_api import Error as PlaywrightError
    from playwright.sync_api import sync_playwright

    PLAYWRIGHT_AVAILABLE = True
except ImportError:  # pragma: no cover — exercised only without the extra
    PLAYWRIGHT_AVAILABLE = False
    sync_playwright = None  # type: ignore[assignment,unused-ignore]
    PlaywrightError = Exception  # type: ignore[assignment,misc,unused-ignore]


class MissingExtraError(RuntimeError):
    """Raised when `render: js` is used before `liner setup-js` has installed support."""


# Time to wait after `domcontentloaded` for the SPA to settle. Tunable later.
DEFAULT_SETTLE_MS = 1500


def _is_missing_browser_error(exc: BaseException) -> bool:
    """Identify Playwright's 'Chromium binary not downloaded' error.

    Playwright's message looks like:
        Executable doesn't exist at /path/to/headless_shell
        ╔════════════════════════════════════════════════════════════╗
        ║ Looks like Playwright was just installed or updated.       ║
        ║ Please run the following command to download new browsers: ║
        ║                                                            ║
        ║     playwright install                                     ║
        ╚════════════════════════════════════════════════════════════╝
    """
    text = str(exc).lower()
    return "executable doesn't exist" in text or "playwright install" in text


def _is_download_starting_error(exc: BaseException) -> bool:
    return "download is starting" in str(exc).lower()


def _looks_like_pdf_url(url: str) -> bool:
    return urlparse(url).path.lower().endswith(".pdf")


class WebJsHandler:
    """Fetches a URL via headless Chromium and runs the standard text-extraction pipeline."""

    def __init__(self, config: FetchConfig) -> None:
        if not PLAYWRIGHT_AVAILABLE:
            raise MissingExtraError("render: js needs Playwright. Run: liner setup-js")
        self._config = config
        self._playwright: Any | None = None
        self._browser: Any | None = None

    def _ensure_browser(self) -> Any:
        if self._browser is not None:
            return self._browser
        self._playwright = sync_playwright().start()
        try:
            self._browser = self._playwright.chromium.launch(headless=True)
        except PlaywrightError as e:
            # If the browser binary hasn't been downloaded, Playwright raises
            # an error whose message contains "Executable doesn't exist". The
            # default surface for this is a wall of text — translate it into a
            # clear instruction that names the right command.
            if _is_missing_browser_error(e):
                # Tear down the started runtime before re-raising so a stuck
                # process isn't left around.
                with contextlib.suppress(Exception):
                    self._playwright.stop()
                self._playwright = None
                raise HandlerHardFailure(
                    "Playwright Chromium isn't installed yet. The Python `[js]` extra "
                    "is in place, but the actual browser binary is a separate ~150MB "
                    "download. Run:\n"
                    "  liner setup-js",
                    "playwright://chromium",
                ) from e
            raise
        return self._browser

    def fetch(self, spec: SourceSpec) -> SourceContent:
        url = spec.url
        browser = self._ensure_browser()
        context = browser.new_context(user_agent=self._config.user_agent)
        # Best-effort cookie loading. Format conversion from Netscape is left
        # for a follow-up — Playwright wants explicit `name/value/domain/path`
        # entries and we'd need a small parser. For now, if cookies_file is
        # set, surface a warning via the metadata only.
        try:
            page = context.new_page()
            page.set_default_timeout(int(self._config.timeout_seconds * 1000))
            if _looks_like_pdf_url(url):
                try:
                    with page.expect_download() as download_info:
                        try:
                            page.goto(url, wait_until="domcontentloaded")
                        except PlaywrightError as e:
                            if not _is_download_starting_error(e):
                                raise
                    return _content_from_pdf_download(spec, download_info.value)
                except PlaywrightError as e:
                    raise HandlerHardFailure(
                        f"Failed to download {url} via headless browser: {e}", url
                    ) from e

            try:
                page.goto(url, wait_until="domcontentloaded")
                page.wait_for_timeout(DEFAULT_SETTLE_MS)
            except PlaywrightError as e:
                raise HandlerHardFailure(
                    f"Failed to load {url} via headless browser: {e}", url
                ) from e

            html = page.content()
        finally:
            context.close()

        extraction = extract_html_text(html)
        body = extraction.body
        fetched_at = datetime.now(UTC).isoformat()

        if looks_like_bot_challenge(html):
            raise HandlerHardFailure(
                f"{url} still returned a security challenge after JS rendering. "
                "Save an authenticated/rendered copy as a local_file source or replace it.",
                url,
            )

        if body is not None and looks_like_js_stub(body):
            raise HandlerHardFailure(
                f"{url} still looks like a noscript stub even after JS execution. "
                "The site may be detecting headless browsers or requires authentication.",
                url,
            )
        if body is not None and looks_like_cookie_notice_only(body):
            raise HandlerHardFailure(
                f"{url} produced only a cookie consent notice after JS rendering. "
                "Try saving the rendered page as a local_file source.",
                url,
            )

        if body is None or len(body) < MIN_USEFUL_BODY_CHARS:
            fallback = extraction.fallback_body
            if looks_like_js_stub(fallback):
                raise HandlerHardFailure(
                    f"{url} produced only a JS-required stub even with headless rendering.",
                    url,
                )
            if looks_like_cookie_notice_only(fallback):
                raise HandlerHardFailure(
                    f"{url} produced only a cookie consent notice after JS rendering. "
                    "Try saving the rendered page as a local_file source.",
                    url,
                )
            if extraction.extraction_error and len(fallback) >= MIN_USEFUL_BODY_CHARS:
                return SourceContent(
                    title=str(extraction.title or url),
                    url=url,
                    body=fallback,
                    fetched_at=fetched_at,
                    author=str(extraction.author) if extraction.author else None,
                    published_at=str(extraction.published_at) if extraction.published_at else None,
                    updated_at=str(extraction.updated_at) if extraction.updated_at else None,
                    metadata={
                        "extraction": "playwright-fallback",
                        **(
                            {"metadata_source": extraction.metadata_source}
                            if extraction.metadata_source
                            else {}
                        ),
                        "trafilatura_error": extraction.extraction_error,
                    },
                )
            raise HandlerHardFailure(
                f"Extracted content from {url} was empty or very short "
                f"({len(fallback)} chars) even after JS rendering. "
                "Consider saving the page locally and using a `local_file` source.",
                url,
            )

        title = extraction.title or url
        return SourceContent(
            title=str(title),
            url=url,
            body=body,
            fetched_at=fetched_at,
            author=str(extraction.author) if extraction.author else None,
            published_at=str(extraction.published_at) if extraction.published_at else None,
            updated_at=str(extraction.updated_at) if extraction.updated_at else None,
            metadata={
                "extraction": "playwright",
                **(
                    {"metadata_source": extraction.metadata_source}
                    if extraction.metadata_source
                    else {}
                ),
            },
        )

    def close(self) -> None:
        if self._browser is not None:
            with contextlib.suppress(Exception):
                self._browser.close()
            self._browser = None
        if self._playwright is not None:
            with contextlib.suppress(Exception):
                self._playwright.stop()
            self._playwright = None

    def __enter__(self) -> WebJsHandler:
        return self

    def __exit__(self, *exc_info: object) -> None:
        self.close()


def _content_from_pdf_download(spec: SourceSpec, download: Any) -> SourceContent:
    url = spec.url
    download_path = Path(download.path())
    body = extract_pdf_text(download_path, url)
    title = _pdf_download_title(url, download)
    content = SourceContent(
        title=title,
        url=url,
        body=body,
        fetched_at=datetime.now(UTC).isoformat(),
        author=None,
        published_at=None,
        metadata={
            "extraction": "playwright_pdf_download",
            "suggested_filename": str(getattr(download, "suggested_filename", "") or ""),
            "size_bytes": download_path.stat().st_size if download_path.exists() else 0,
        },
    )
    if len(body) < MIN_USEFUL_BODY_CHARS:
        raise HandlerHardFailure(
            f"Extracted PDF text from {url} was empty or very short ({len(body)} chars) "
            "after headless-browser download.",
            url,
        )
    return content


def _pdf_download_title(url: str, download: Any) -> str:
    suggested = str(getattr(download, "suggested_filename", "") or "").strip()
    if suggested:
        return suggested
    path = unquote(urlparse(url).path.rstrip("/"))
    name = path.rsplit("/", 1)[-1]
    return name or url


__all__ = ["WebJsHandler", "MissingExtraError", "PLAYWRIGHT_AVAILABLE"]
