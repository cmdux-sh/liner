from __future__ import annotations

import re
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
FAILURE_BODY_PREVIEW_BYTES = 500


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
            raise HandlerHardFailure(_network_failure_message(url, e), url) from e

        if response.status_code >= 400:
            # Bot-detection vendors (Cloudflare, Akamai, Imperva, PerimeterX)
            # serve a challenge UI on 4xx/5xx instead of the real article. The
            # right recovery is the same as for a JS stub: retry through the
            # headless browser, which can carry cookies + execute the challenge.
            # Falling through to HandlerHardFailure would have dead-ended a
            # case the JS handler can sometimes solve.
            if looks_like_bot_challenge(response.text):
                raise JsRenderingRequired(
                    _response_failure_message(
                        url,
                        response,
                        category="js_required",
                        lead="bot-detection interstitial; retrying via render: js",
                    ),
                    url,
                )
            raise HandlerHardFailure(
                _response_failure_message(url, response),
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


def _network_failure_message(url: str, error: httpx.HTTPError) -> str:
    category = "tls" if _looks_like_tls_error(error) else "network"
    return (
        f"Failed to fetch {url} — category: {category}; "
        f"error: {type(error).__name__}: {error}"
    )


def _response_failure_message(
    url: str,
    response: httpx.Response,
    *,
    category: str | None = None,
    lead: str | None = None,
) -> str:
    category = category or _response_failure_category(response)
    content_type = _response_content_type(response)
    preview = _response_body_preview(response)
    parts = [
        f"Failed to fetch {url}",
        f"category: {category}",
        f"status: HTTP {response.status_code}",
        f"content-type: {content_type}",
    ]
    if lead:
        parts.append(lead)
    if preview:
        parts.append(f"body preview: {preview}")
    return " — ".join([parts[0], "; ".join(parts[1:])])


def _response_failure_category(response: httpx.Response) -> str:
    text = _safe_response_text(response).lower()
    if looks_like_js_stub(text) or looks_like_cookie_notice_only(text):
        return "js_required"
    if _looks_like_paywall(text):
        return "paywall"
    if response.status_code == 404:
        return "not_found"
    if response.status_code in {401, 403}:
        return "forbidden"
    return "unknown"


def _response_content_type(response: httpx.Response) -> str:
    content_type = response.headers.get("content-type", "").strip()
    if not content_type:
        return "unknown"
    return content_type.split(";", 1)[0].strip().lower() or "unknown"


def _response_body_preview(response: httpx.Response) -> str:
    raw = response.content or b""
    if not raw:
        return ""
    truncated = len(raw) > FAILURE_BODY_PREVIEW_BYTES
    head = raw[:FAILURE_BODY_PREVIEW_BYTES]
    preview = head.decode(response.encoding or "utf-8", errors="replace")
    preview = re.sub(r"\s+", " ", preview).strip()
    if truncated:
        preview += " ... [truncated]"
    return preview


def _safe_response_text(response: httpx.Response) -> str:
    try:
        return response.text
    except UnicodeDecodeError:
        return response.content.decode("utf-8", errors="replace")


def _looks_like_paywall(text: str) -> bool:
    return any(
        phrase in text
        for phrase in (
            "subscribe to continue",
            "subscribe to read",
            "subscribers only",
            "subscriber-only",
            "members only",
            "member-only",
            "sign in to continue",
            "sign in to read",
            "paywall",
        )
    )


def _looks_like_tls_error(error: httpx.HTTPError) -> bool:
    message = str(error).lower()
    return any(term in message for term in ("tls", "ssl", "certificate", "cert verify"))


__all__ = ["WebHandler"]
