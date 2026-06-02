from __future__ import annotations

from datetime import UTC, datetime
from urllib.parse import unquote, urlparse

import httpx

from liner.config import FetchConfig
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure, JsRenderingRequired
from liner.handlers.html_extraction import (
    MIN_USEFUL_BODY_CHARS,
    extract_html_text,
    looks_like_bot_challenge,
    looks_like_cookie_notice_only,
    looks_like_js_stub,
    looks_like_spa_shell,
)
from liner.handlers.pdf_extraction import extract_pdf_text
from liner.types import SourceContent, SourceSpec

# Backward-compat re-export — older imports may still reference this helper.
_looks_like_js_stub = looks_like_js_stub


class WebHandler:
    def __init__(self, config: FetchConfig) -> None:
        self._config = config
        self._client = httpx.Client(
            headers={"User-Agent": config.user_agent},
            timeout=config.timeout_seconds,
            follow_redirects=True,
        )

    def fetch(self, spec: SourceSpec) -> SourceContent:
        url = spec.url
        try:
            response = self._client.get(url)
        except httpx.HTTPError as e:
            raise HandlerHardFailure(f"Failed to fetch {url} — network error: {e}", url) from e

        if response.status_code >= 400:
            # Bot-detection vendors (Cloudflare, Akamai, Imperva, PerimeterX)
            # serve a challenge UI on 4xx/5xx instead of the real article. The
            # right recovery is the same as for a JS stub: retry through the
            # headless browser, which can carry cookies + execute the challenge.
            # Falling through to HandlerHardFailure would have dead-ended a
            # case the JS handler can sometimes solve.
            if looks_like_bot_challenge(response.text):
                raise JsRenderingRequired(
                    f"{url} returned HTTP {response.status_code} with a bot-detection "
                    "interstitial — retrying via render: js.",
                    url,
                )
            raise HandlerHardFailure(
                f"Failed to fetch {url} — got HTTP {response.status_code}. "
                "The site may be blocking automated requests.",
                url,
            )

        if _looks_like_pdf_response(response):
            return self._content_from_pdf_response(spec, response)

        extraction = extract_html_text(response.text)
        title = extraction.title or url
        fetched_at = datetime.now(UTC).isoformat()

        if extraction.body is not None and looks_like_js_stub(extraction.body):
            raise JsRenderingRequired(
                f"{url} is JavaScript-rendered — server-rendered HTML is just a noscript stub.",
                url,
            )
        if extraction.body is not None and looks_like_cookie_notice_only(extraction.body):
            raise JsRenderingRequired(
                f"{url} returned only a cookie consent notice — retrying via render: js.",
                url,
            )

        body = extraction.body
        if body is None or len(body) < MIN_USEFUL_BODY_CHARS:
            fallback = extraction.fallback_body
            if looks_like_js_stub(fallback):
                raise JsRenderingRequired(
                    f"{url} is JavaScript-rendered — server-rendered HTML is just a noscript stub.",
                    url,
                )
            if looks_like_spa_shell(response.text, fallback):
                raise JsRenderingRequired(
                    f"{url} looks like a JavaScript app shell — server-rendered HTML "
                    "contains almost no readable content.",
                    url,
                )
            if looks_like_cookie_notice_only(fallback):
                raise JsRenderingRequired(
                    f"{url} returned only a cookie consent notice — retrying via render: js.",
                    url,
                )
            soft = SourceContent(
                title=str(title),
                url=url,
                body=fallback,
                fetched_at=fetched_at,
                author=str(extraction.author) if extraction.author else None,
                published_at=str(extraction.published_at) if extraction.published_at else None,
                metadata={
                    "extraction": "soft-fallback",
                    **(
                        {"trafilatura_error": extraction.extraction_error}
                        if extraction.extraction_error
                        else {}
                    ),
                },
            )
            if extraction.extraction_error and len(fallback) >= MIN_USEFUL_BODY_CHARS:
                raise HandlerSoftFailure(
                    soft,
                    f"Primary HTML extraction failed for {url}; used fallback text extraction.",
                )
            raise HandlerSoftFailure(
                soft,
                f"Extracted content from {url} was empty or very short ({len(fallback)} chars).",
            )

        return SourceContent(
            title=str(title),
            url=url,
            body=body,
            fetched_at=fetched_at,
            author=str(extraction.author) if extraction.author else None,
            published_at=str(extraction.published_at) if extraction.published_at else None,
            metadata={"extraction": "trafilatura"},
        )

    def close(self) -> None:
        self._client.close()

    def _content_from_pdf_response(
        self, spec: SourceSpec, response: httpx.Response
    ) -> SourceContent:
        url = spec.url
        body = extract_pdf_text(response.content, url)
        title = _pdf_title(url)
        content = SourceContent(
            title=title,
            url=url,
            body=body,
            fetched_at=datetime.now(UTC).isoformat(),
            author=None,
            published_at=None,
            metadata={
                "extraction": "remote_pdf",
                "content_type": response.headers.get("content-type", ""),
                "size_bytes": len(response.content),
            },
        )
        if len(body) < MIN_USEFUL_BODY_CHARS:
            raise HandlerSoftFailure(
                content,
                f"Extracted PDF text from {url} was empty or very short ({len(body)} chars).",
            )
        return content


def _looks_like_pdf_response(response: httpx.Response) -> bool:
    content_type = response.headers.get("content-type", "").lower()
    if "application/pdf" in content_type:
        return True
    if response.content.startswith(b"%PDF-"):
        return True
    path = urlparse(str(response.url)).path.lower()
    return path.endswith(".pdf")


def _pdf_title(url: str) -> str:
    path = unquote(urlparse(url).path.rstrip("/"))
    name = path.rsplit("/", 1)[-1]
    return name or url


__all__ = ["WebHandler"]
