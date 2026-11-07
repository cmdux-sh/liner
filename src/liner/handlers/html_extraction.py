"""Shared HTML → text extraction used by WebHandler, WebJsHandler, and LocalFileHandler."""

from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import UTC, datetime
from html import unescape
from html.parser import HTMLParser

import trafilatura

MIN_USEFUL_BODY_CHARS = 100
# Bump when extraction gets better at a kind of page, so cached pages saved by the
# older extractor are fetched again instead of served. 2: pages trafilatura cut short
# are read again in recall mode (2026-11-05). 3: table cells holding only an icon keep
# its label, such as Apple's check marks (2026-11-06). 4: when recall mode stops short
# too, the fallbacks are tried alone, which reads Apple's product page guide (2026-11-06).
# 5: a page that needs a browser is read from its declared Markdown copy (2026-11-06).
EXTRACTOR_VERSION = 5

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
    labeled = _label_icon_cells(html)
    icon_tables = labeled != html  # Its tables hold marks that only a table read keeps.
    html = labeled
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
    if body and _keeps_too_little(body, html):
        recalled = _read_again(html)
        if recalled is not None and len(recalled.split()) > 2 * len(body.split()):
            body = recalled
    elif body and icon_tables:
        recalled = _recall_extract(html)
        if recalled is not None and len(recalled.split()) >= len(body.split()):
            body = recalled

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


# Article pages keep two thirds or more of their visible words. A page that keeps
# under a quarter, such as Apple's App Review Guidelines (1%), lost its body to the
# precise extractor, so try again favoring recall.
_MIN_KEPT_SHARE = 0.25
_MIN_PAGE_WORDS_FOR_RETRY = 300
_CHROME_TAGS = ("script", "style", "noscript", "svg", "template", "nav", "header", "footer")


_CELL = re.compile(r"<(td|th)\b([^>]*)>(.*?)</\1\s*>", re.IGNORECASE | re.DOTALL)
_LABEL = re.compile(r"""\b(?:alt|aria-label|title)\s*=\s*(?:"([^"]*)"|'([^']*)')""", re.IGNORECASE)


def _label_icon_cells(html: str) -> str:
    """Give a table cell that holds only an icon its label, so a check mark isn't lost.

    Apple's property tables mark each cell with an empty `<figure alt="Localized">`,
    which extraction would drop as an empty cell.
    """
    lowered = html.lower()
    if "<td" not in lowered and "<th" not in lowered:
        return html

    def label(match: re.Match[str]) -> str:
        tag, attributes, inner = match.groups()
        if "<td" in inner.lower() or re.sub(r"<[^>]*>|&nbsp;|\s", "", inner):
            return match.group(0)
        labels = [unescape(a or b).strip() for a, b in _LABEL.findall(inner)]
        labels = [text for text in labels if text]
        if not labels:
            return match.group(0)
        text = ", ".join(dict.fromkeys(labels)).replace("&", "&amp;").replace("<", "&lt;")
        return f"<{tag}{attributes}>{text}</{tag}>"

    return _CELL.sub(label, html)


def _keeps_too_little(body: str, html: str) -> bool:
    # A stub, cookie wall, or bot check is a short body on purpose: the fetch handlers
    # detect those from it, so a retry must not replace it with the page's chrome.
    if looks_like_js_stub(body) or looks_like_cookie_notice_only(body):
        return False
    if looks_like_bot_challenge(body):
        return False
    # Most article pages are well over a tenth text, so skip the page count for them.
    if len(body) * 10 >= len(html):
        return False
    page_words = _visible_word_count(html)
    return (
        page_words >= _MIN_PAGE_WORDS_FOR_RETRY and len(body.split()) < _MIN_KEPT_SHARE * page_words
    )


def _visible_word_count(html: str) -> int:
    """Words a reader sees, without navigation, headers, footers, or code."""
    text = re.sub(r"<!--.*?-->", " ", html, flags=re.DOTALL)
    for tag in _CHROME_TAGS:
        text = re.sub(rf"<{tag}\b[^>]*>.*?</{tag}\s*>", " ", text, flags=re.IGNORECASE | re.DOTALL)
    return len(re.sub(r"<[^>]+>", " ", text).split())


def _recall_extract(html: str) -> str | None:
    """A looser extraction for a page whose precise extract kept too little.

    It keeps trafilatura's fallback extractors and tables on purpose: Apple's
    guidelines need the fallbacks, and its field reference is a table.
    """
    return _extract_with_fallbacks(html, favor_recall=True)


def _read_again(html: str) -> str | None:
    """Recall mode, then the fallbacks alone if recall also stopped short.

    Recall mode can stop at a page's intro, as on Apple's product page guide, while
    the fallbacks without it read the whole page. They can also pick up more page
    chrome, so they're used only when the recall read still keeps too little.
    """
    recalled = _recall_extract(html)
    if recalled is not None and not _keeps_too_little(recalled, html):
        return recalled
    other = _extract_with_fallbacks(html, favor_recall=False)
    if other is not None and (recalled is None or len(other.split()) > len(recalled.split())):
        return other
    return recalled


def _extract_with_fallbacks(html: str, *, favor_recall: bool) -> str | None:
    try:
        text = trafilatura.extract(
            html,
            output_format="txt",
            include_comments=False,
            include_tables=True,
            favor_recall=favor_recall,
        )
    except Exception:
        return None
    if not text:
        return None
    lines = (" ".join(line.split()) for line in text.splitlines())
    return "\n".join(line for line in lines if line) or None


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
