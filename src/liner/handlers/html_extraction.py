"""Shared HTML → text extraction used by WebHandler, WebJsHandler, and LocalFileHandler."""

from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import UTC, datetime
from html import unescape
from html.parser import HTMLParser

import trafilatura

MIN_USEFUL_BODY_CHARS = 100

# Phrases indicating server-rendered HTML is just a noscript fallback for a
# JavaScript-only app. Surfacing this as the source body misleads the AI.
JS_REQUIRED_PATTERNS = (
    "this page requires javascript",
    "please enable javascript",
    "please turn on javascript",
    "you need to enable javascript to run this app",
    "javascript is required to view this site",
    "javascript is disabled in your browser",
)

# Substrings that show up in bot-detection / DDoS-mitigation interstitials.
# When a 4xx/5xx response body matches one of these, the right next move is
# a headless-browser retry (cookies, JS challenge solver) rather than
# treating the failure as terminal. Match against title or first KB of body —
# these phrases appear early in the document on every vendor we've checked.
BOT_CHALLENGE_PATTERNS = (
    "just a moment...",  # Cloudflare interstitial
    "checking your browser before accessing",  # Cloudflare older
    "cf-browser-verification",  # Cloudflare DOM class
    "/cdn-cgi/challenge-platform",  # Cloudflare asset path
    "access denied",  # generic Akamai / Imperva
    "request unsuccessful. incapsula",  # Imperva Incapsula
    "pardon our interruption",  # Distil / Imperva
    "ddos protection by",  # generic vendor banner
    "ray id:",  # Cloudflare error footer
    "please verify you are a human",  # PerimeterX / hCaptcha
    "press and hold",  # PerimeterX press-and-hold
    "/_px/",  # PerimeterX asset path
    "vercel security checkpoint",  # Vercel bot mitigation
    "we're verifying your browser",  # Vercel challenge copy
    "website owner? click here to fix",  # Vercel challenge footer
    "performing security verification",  # generic journal security interstitial
    "security service to protect against malicious bots",  # generic bot service copy
)

# Signals that an HTML response is only the application shell for a client-side
# app. These pages often have a title and a mountain of JS assets, but almost
# no body text. If we accept the tiny fallback body, users get a "successful"
# source that contains no useful source context.
SPA_SHELL_PATTERNS = (
    "data-beasties-container",  # Angular prerender shell
    "<app-root",
    "ng-version",
    'id="root"',
    "id='root'",
    'id="app"',
    "id='app'",
    "__next_data__",
    "window.__",
    "webpack",
    "vite",
)

# Cookie banners are common on JS-rendered sites. If extraction returns only
# the consent copy, the source is not usable even though the fetch "worked".
COOKIE_NOTICE_ACTIONS = (
    "ok, got it",
    "accept all",
    "accept cookies",
    "reject all",
    "manage options",
    "learn more",
)

DECLARED_TITLE_KEYS = (
    "og:title",
    "twitter:title",
    "citation_title",
    "dc.title",
    "dcterms.title",
)
DECLARED_AUTHOR_KEYS = (
    "citation_author",
    "author",
    "article:author",
    "dc.creator",
    "dcterms.creator",
)
DECLARED_PUBLISHED_KEYS = (
    "article:published_time",
    "citation_publication_date",
    "citation_date",
    "datepublished",
    "dc.date.issued",
    "dcterms.issued",
)
DECLARED_UPDATED_KEYS = (
    "article:modified_time",
    "datemodified",
    "last-modified",
    "dcterms.modified",
)
INVALID_TITLE_MARKERS = (
    "references-details-empty",
    "reference-details-empty",
    "vercel security checkpoint",
    "performing security verification",
    "just a moment...",
)


def looks_like_js_stub(body: str) -> bool:
    if not body:
        return False
    lowered = body.lower()
    return any(p in lowered for p in JS_REQUIRED_PATTERNS)


def looks_like_bot_challenge(body: str) -> bool:
    """True if `body` looks like a bot-detection / DDoS interstitial.

    These pages return as 4xx/5xx with HTML that's a challenge UI rather
    than the article. They are usually solvable by a headless-browser
    retry (cookies + JS) — which is the same recovery path as a JS stub.
    """
    if not body:
        return False
    # Scan a generous window so we catch banners that appear in body, not <head>.
    lowered = body[:4096].lower()
    return any(p in lowered for p in BOT_CHALLENGE_PATTERNS)


def looks_like_spa_shell(html: str, fallback_body: str) -> bool:
    """True when HTML looks like a JS app shell with no useful rendered text."""
    if len(fallback_body.strip()) >= MIN_USEFUL_BODY_CHARS:
        return False
    lowered = html[:200_000].lower()
    if "<script" not in lowered:
        return False
    return any(p in lowered for p in SPA_SHELL_PATTERNS)


def looks_like_cookie_notice_only(body: str) -> bool:
    """True when extracted text is just a short cookie-consent notice."""
    lowered = " ".join(body.lower().split())
    if not lowered or len(lowered) >= 500:
        return False
    if "cookie" not in lowered and "cookies" not in lowered:
        return False
    if (
        "uses cookies from google" in lowered
        and "deliver and enhance the quality of its services" in lowered
        and "analyze traffic" in lowered
    ):
        return True
    return any(p in lowered for p in COOKIE_NOTICE_ACTIONS)


@dataclass(frozen=True, slots=True)
class HtmlExtraction:
    """Result of pulling text + metadata out of an HTML document."""

    title: str | None
    author: str | None
    published_at: str | None
    updated_at: str | None
    metadata_source: str | None
    # The cleaned article body from trafilatura. May be None or below the
    # useful-length threshold; callers decide what to do with that.
    body: str | None
    # A crude tag-stripped fallback string. Always present.
    fallback_body: str
    # Best-effort diagnostic when the primary extractor crashed. Callers can
    # still use fallback_body instead of failing the whole source.
    extraction_error: str | None = None


def extract_html_text(html: str) -> HtmlExtraction:
    """Run trafilatura against `html` and pull metadata + crude fallback text."""
    extraction_error: str | None = None
    try:
        body = trafilatura.extract(
            html,
            output_format="txt",
            include_comments=False,
            include_tables=False,
            no_fallback=True,
        )
    except Exception as e:
        body = None
        extraction_error = f"{type(e).__name__}: {e}"

    title: str | None = None
    extracted_title: str | None = None
    try:
        metadata_json = trafilatura.extract_metadata(html)
        if metadata_json is not None:
            md = metadata_json.as_dict() if hasattr(metadata_json, "as_dict") else {}
            extracted_title = md.get("title") or None
    except Exception as e:
        if extraction_error is None:
            extraction_error = f"{type(e).__name__}: {e}"

    declared = _declared_metadata(html)
    title = sanitize_source_title(declared.first(DECLARED_TITLE_KEYS))
    if title is None:
        title = sanitize_source_title(extracted_title)
    if title is None:
        title = sanitize_source_title(_fallback_title(html))
    author = _declared_authors(declared)
    published_at = _sanitize_declared_date(declared.first(DECLARED_PUBLISHED_KEYS))
    updated_at = _sanitize_declared_date(declared.first(DECLARED_UPDATED_KEYS))
    metadata_source = "declared_html" if declared.values else None

    fallback = body or _html_fallback_text(html)
    return HtmlExtraction(
        title=title,
        author=author,
        published_at=published_at,
        updated_at=updated_at,
        metadata_source=metadata_source,
        body=body,
        fallback_body=fallback,
        extraction_error=extraction_error,
    )


class _DeclaredMetadataParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.values: dict[str, list[str]] = {}

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        attributes = {key.lower(): value for key, value in attrs if value is not None}
        if tag.lower() == "meta":
            key = attributes.get("property") or attributes.get("name") or attributes.get("itemprop")
            value = attributes.get("content")
            self._record(key, value)
            return
        if tag.lower() == "time":
            self._record(attributes.get("itemprop"), attributes.get("datetime"))

    def _record(self, key: str | None, value: str | None) -> None:
        normalized_key = str(key or "").strip().lower()
        normalized_value = _clean_metadata_text(value)
        if not normalized_key or not normalized_value:
            return
        self.values.setdefault(normalized_key, []).append(normalized_value)

    def first(self, keys: tuple[str, ...]) -> str | None:
        for key in keys:
            values = self.values.get(key)
            if values:
                return values[0]
        return None


def _declared_metadata(html: str) -> _DeclaredMetadataParser:
    parser = _DeclaredMetadataParser()
    try:
        parser.feed(html)
    except Exception:
        return _DeclaredMetadataParser()
    return parser


def _declared_authors(metadata: _DeclaredMetadataParser) -> str | None:
    authors: list[str] = []
    for key in DECLARED_AUTHOR_KEYS:
        for value in metadata.values.get(key, []):
            if value not in authors:
                authors.append(value)
    return "; ".join(authors) or None


def _clean_metadata_text(value: object) -> str:
    return " ".join(unescape(str(value or "")).split()).strip()


def sanitize_source_title(value: object) -> str | None:
    cleaned = _clean_metadata_text(value)
    lowered = cleaned.lower()
    if not cleaned or any(marker in lowered for marker in INVALID_TITLE_MARKERS):
        return None
    if lowered in {"untitled", "document", "home", "loading"}:
        return None
    return cleaned


def _sanitize_declared_date(value: object) -> str | None:
    cleaned = _clean_metadata_text(value)
    if not cleaned:
        return None
    year_match = re.search(r"(?<!\d)(\d{4})(?!\d)", cleaned)
    if year_match is None:
        return None
    year = int(year_match.group(1))
    if year < 1450 or year > datetime.now(UTC).year + 1:
        return None
    return cleaned


def _fallback_title(html: str) -> str | None:
    m = re.search(r"<title>(.*?)</title>", html, re.IGNORECASE | re.DOTALL)
    if not m:
        return None
    return " ".join(m.group(1).split()).strip() or None


def _html_fallback_text(html: str) -> str:
    text = re.sub(r"<script.*?</script>", " ", html, flags=re.IGNORECASE | re.DOTALL)
    text = re.sub(r"<style.*?</style>", " ", text, flags=re.IGNORECASE | re.DOTALL)
    text = re.sub(r"<[^>]+>", " ", text)
    return " ".join(text.split())


__all__ = [
    "HtmlExtraction",
    "MIN_USEFUL_BODY_CHARS",
    "extract_html_text",
    "looks_like_bot_challenge",
    "looks_like_cookie_notice_only",
    "looks_like_js_stub",
    "looks_like_spa_shell",
    "sanitize_source_title",
]
