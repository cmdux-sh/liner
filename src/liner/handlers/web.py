from __future__ import annotations

import re
from datetime import UTC, datetime
from html import unescape
from urllib.parse import unquote, urljoin, urlparse

import httpx

from liner.config import FetchConfig
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure, JsRenderingRequired
from liner.handlers.fetch_guard import get_bounded, guarded_client
from liner.handlers.html_extraction import (
    EXTRACTOR_VERSION,
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
        self._client = guarded_client(
            allow_private_network=config.allow_private_network,
            headers={"User-Agent": config.user_agent},
            timeout=config.timeout_seconds,
            follow_redirects=True,
        )

    def _fetch_markdown_alternate(self, url: str, alternate: str) -> SourceContent | None:
        try:
            response = get_bounded(self._client, alternate)
        except httpx.HTTPError:
            return None
        if response.status_code >= 400 or not _looks_like_text_response(response):
            return None
        if _origin(str(response.url)) != _origin(url):  # A redirect left the site.
            return None
        if len(response.text.strip()) < MIN_USEFUL_BODY_CHARS:
            return None
        return _content_from_text(
            url, response.text, extraction="markdown-alternate", markdown_url=alternate
        )

    def fetch(self, spec: SourceSpec) -> SourceContent:
        url = spec.url
        try:
            response = get_bounded(self._client, url)
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
        if _looks_like_text_response(response):
            return _content_from_text(url, response.text, extraction="markdown")
        try:
            return self._content_from_html(url, response)
        except (JsRenderingRequired, HandlerSoftFailure):
            # Some pages that need a browser, such as Apple's Human Interface
            # Guidelines, declare a Markdown copy of themselves for agents.
            alternate = _markdown_alternate(response)
            content = self._fetch_markdown_alternate(url, alternate) if alternate else None
            if content is None:
                raise
            return content

    def _content_from_html(self, url: str, response: httpx.Response) -> SourceContent:
        extraction = extract_html_text(response.text)
        title = extraction.title or url
        fetched_at = datetime.now(UTC).isoformat()

        if looks_like_bot_challenge(response.text):
            raise JsRenderingRequired(
                f"{url} returned a security challenge instead of source content — "
                "retrying via render: js.",
                url,
            )

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
                updated_at=str(extraction.updated_at) if extraction.updated_at else None,
                metadata={
                    "extraction": "soft-fallback",
                    "extractor_version": EXTRACTOR_VERSION,
                    **(
                        {"metadata_source": extraction.metadata_source}
                        if extraction.metadata_source
                        else {}
                    ),
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
            updated_at=str(extraction.updated_at) if extraction.updated_at else None,
            metadata={
                "extraction": "trafilatura",
                "extractor_version": EXTRACTOR_VERSION,
                **(
                    {"metadata_source": extraction.metadata_source}
                    if extraction.metadata_source
                    else {}
                ),
            },
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


_MARKDOWN_ALTERNATE = re.compile(r"<link\b[^>]*>", re.IGNORECASE)
_HTML_COMMENT = re.compile(r"<!--.*?-->", re.DOTALL)
_HTML_START = re.compile(r"<(!doctype|html|head|body)\b", re.IGNORECASE)
_ATTRIBUTE = re.compile(r'([a-z-]+)\s*=\s*["\']([^"\']*)["\']', re.IGNORECASE)


def _looks_like_text_response(response: httpx.Response) -> bool:
    content_type = response.headers.get("content-type", "").lower()
    if not content_type.startswith(("text/markdown", "text/x-markdown", "text/plain")):
        return False
    # HTML served as plain text is still HTML. Markdown may open with a comment.
    start = _HTML_COMMENT.sub("", response.text.lstrip("\ufeff \t\r\n"), count=1)
    return not _HTML_START.match(start.lstrip())


def _markdown_alternate(response: httpx.Response) -> str | None:
    """The page's own Markdown copy, as Apple's documentation declares for agents.

    Only a `<link>` in the page's head counts, and only a copy on the same origin, so
    the page's body can't pick it and the page can't send Liner to another site.
    """
    text = response.text
    end = text.lower().find("</head")
    head = _HTML_COMMENT.sub(" ", text[: end if end >= 0 else 200_000])
    for tag in _MARKDOWN_ALTERNATE.findall(head):
        attributes = {name.lower(): value for name, value in _ATTRIBUTE.findall(tag)}
        if "alternate" not in attributes.get("rel", "").lower().split():
            continue
        if attributes.get("type", "").split(";")[0].strip().lower() != "text/markdown":
            continue
        href = urljoin(str(response.url), unescape(attributes.get("href", "")))
        if _origin(href) == _origin(str(response.url)):
            return href
    return None


def _origin(url: str) -> tuple[str, str | None, int | None]:
    parts = urlparse(url)
    try:
        port = parts.port or {"http": 80, "https": 443}.get(parts.scheme)
    except ValueError:
        port = None
    return parts.scheme, parts.hostname, port


def _content_from_text(url: str, text: str, *, extraction: str, **extra: str) -> SourceContent:
    body = text.lstrip("\ufeff").strip()
    if body.startswith("<!--"):  # Front matter, such as the page metadata Apple puts first.
        body = _HTML_COMMENT.sub("", body, count=1).strip()
    heading = next((line[2:].strip() for line in body.splitlines() if line.startswith("# ")), None)
    content = SourceContent(
        title=heading or url,
        url=url,
        body=body,
        fetched_at=datetime.now(UTC).isoformat(),
        metadata={"extraction": extraction, **extra},
    )
    if len(body) < MIN_USEFUL_BODY_CHARS:
        raise HandlerSoftFailure(content, f"{url} returned almost no text ({len(body)} chars).")
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
    return f"Failed to fetch {url} — category: {category}; error: {type(error).__name__}: {error}"


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
