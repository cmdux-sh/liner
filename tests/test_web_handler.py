from __future__ import annotations

import httpx
import pytest

from liner.config import FetchConfig
from liner.handlers import html_extraction
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure, JsRenderingRequired
from liner.handlers.html_extraction import (
    extract_html_text,
    looks_like_bot_challenge,
    looks_like_cookie_notice_only,
    looks_like_spa_shell,
)
from liner.handlers.web import WebHandler, _looks_like_js_stub
from liner.types import SourceSpec


def test_looks_like_js_stub_recognizes_common_phrases() -> None:
    assert _looks_like_js_stub("This page requires JavaScript to display.")
    assert _looks_like_js_stub("please enable JavaScript and reload")
    assert _looks_like_js_stub("Please turn on JavaScript in your browser")
    assert _looks_like_js_stub("You need to enable JavaScript to run this app.")


def test_looks_like_js_stub_ignores_normal_content() -> None:
    assert not _looks_like_js_stub("")
    assert not _looks_like_js_stub("A normal article body discussing many topics.")
    # An article that merely mentions JavaScript shouldn't trip the check
    # — only the literal "requires/enable JavaScript" phrasings should.
    assert not _looks_like_js_stub(
        "This guide compares JavaScript and TypeScript for large frontends."
    )


def test_web_handler_raises_hard_failure_for_js_rendered_pages() -> None:
    js_stub_html = (
        "<html><head><title>SPA App</title></head>"
        "<body><noscript><h1>This page requires JavaScript.</h1>"
        "<p>Please turn on JavaScript in your browser and refresh the page to view its content.</p>"
        "</noscript><div id='root'></div></body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=js_stub_html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/spa"))
    assert "JavaScript" in str(exc_info.value)
    handler.close()


def test_web_handler_accepts_real_html() -> None:
    html = (
        "<html><head><title>Real Article</title></head><body>"
        + "<article><h1>The Article</h1>"
        + "<p>"
        + ("This is a substantive paragraph of content. " * 20)
        + "</p>"
        + "</article></body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    content = handler.fetch(SourceSpec(type="web", url="https://example.com/article"))
    assert "substantive paragraph" in content.body
    assert content.metadata.get("extraction") == "trafilatura"
    handler.close()


def test_html_metadata_uses_declared_publication_and_update_fields() -> None:
    html = """
    <html><head>
      <meta property="og:title" content="Reliable article title">
      <meta name="citation_author" content="Ada Lovelace">
      <meta name="citation_author" content="Grace Hopper">
      <meta property="article:published_time" content="2022-06-01T09:00:00Z">
      <meta property="article:modified_time" content="2024-04-12T10:30:00Z">
    </head><body><article>Substantive body.</article></body></html>
    """

    extraction = extract_html_text(html)

    assert extraction.title == "Reliable article title"
    assert extraction.author == "Ada Lovelace; Grace Hopper"
    assert extraction.published_at == "2022-06-01T09:00:00Z"
    assert extraction.updated_at == "2024-04-12T10:30:00Z"


def test_html_metadata_omits_placeholder_and_inferred_navigation_values(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    class BadMetadata:
        def as_dict(self) -> dict[str, str]:
            return {
                "title": "references-details-empty",
                "author": "Thoroughly Review the Authorization Logic; Technologies",
                "date": "1997-03-18",
            }

    monkeypatch.setattr(html_extraction.trafilatura, "extract_metadata", lambda *_: BadMetadata())
    html = "<html><head><title>references-details-empty</title></head><body><article>Useful content.</article></body></html>"

    extraction = extract_html_text(html)

    assert extraction.title is None
    assert extraction.author is None
    assert extraction.published_at is None
    assert extraction.updated_at is None


def test_web_handler_extracts_remote_pdf_before_html_pipeline(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    pdf_bytes = b"%PDF-1.7 fake bytes"

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "application/pdf"},
            content=pdf_bytes,
            request=request,
        )

    def fake_pdf_extract(source: bytes, identifier: str) -> str:
        assert source == pdf_bytes
        assert identifier == "https://example.com/report.pdf"
        return "Extracted PDF text " * 20

    monkeypatch.setattr("liner.handlers.web.extract_pdf_text", fake_pdf_extract)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    content = handler.fetch(SourceSpec(type="web", url="https://example.com/report.pdf"))
    assert "Extracted PDF text" in content.body
    assert content.title == "report.pdf"
    assert content.metadata.get("extraction") == "remote_pdf"
    assert content.metadata.get("size_bytes") == len(pdf_bytes)
    handler.close()


def test_web_handler_warns_for_short_remote_pdf_text(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "application/pdf"},
            content=b"%PDF-1.7 fake bytes",
            request=request,
        )

    monkeypatch.setattr("liner.handlers.web.extract_pdf_text", lambda *_: "short")

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerSoftFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/report.pdf"))
    assert exc_info.value.content.metadata.get("extraction") == "remote_pdf"
    assert "PDF text" in str(exc_info.value)
    handler.close()


def test_web_handler_falls_back_when_primary_html_extraction_crashes(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    html = (
        "<html><head><title>Fallback Article</title></head><body>"
        + "<article><h1>Fallback Article</h1>"
        + "<p>"
        + ("Useful fallback content survives extraction crashes. " * 20)
        + "</p>"
        + "</article></body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=html)

    def broken_extract(*args: object, **kwargs: object) -> str | None:
        raise TypeError("'NoneType' object is not subscriptable")

    monkeypatch.setattr(html_extraction.trafilatura, "extract", broken_extract)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerSoftFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/fallback"))
    assert "used fallback text extraction" in str(exc_info.value)
    assert "Useful fallback content" in exc_info.value.content.body
    assert exc_info.value.content.metadata.get("trafilatura_error")
    handler.close()


# --- Fix A: bot-challenge → render: js auto-fallback -----------------------


def test_looks_like_bot_challenge_recognizes_vendor_signatures() -> None:
    # Cloudflare interstitial — the exact substring we observed live on Medium.
    cf = "<!DOCTYPE html><html><head><title>Just a moment...</title></head>"
    assert looks_like_bot_challenge(cf)
    # Cloudflare older "checking your browser" wording.
    assert looks_like_bot_challenge(
        "<html>Checking your browser before accessing example.com…</html>"
    )
    # Generic security-service interstitials observed on Hindawi and SAGE.
    assert looks_like_bot_challenge(
        "Performing security verification. This website uses a security service "
        "to protect against malicious bots."
    )


def test_web_handler_retries_200_security_verification_page_via_js() -> None:
    html = (
        "<html><head><title>Performing security verification</title></head><body>"
        "<h1>Performing security verification</h1>"
        "<p>This website uses a security service to protect against malicious bots.</p>"
        "</body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(JsRenderingRequired) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://journals.example/article"))
    assert "security challenge" in str(exc_info.value)
    handler.close()
    # Imperva Incapsula.
    assert looks_like_bot_challenge("Request unsuccessful. Incapsula incident ID: 1234")
    # Distil / Imperva "Pardon our interruption" page.
    assert looks_like_bot_challenge("<h1>Pardon Our Interruption</h1>")
    # PerimeterX press-and-hold challenge.
    assert looks_like_bot_challenge("<p>Please verify you are a human</p>")
    # Vercel Security Checkpoint, seen on Brian Lovin article fetches.
    assert looks_like_bot_challenge(
        "<html><head><title>Vercel Security Checkpoint</title></head>"
        "<body><p>We're verifying your browser</p>"
        "<a>Website owner? Click here to fix</a></body></html>"
    )


def test_looks_like_bot_challenge_ignores_normal_content() -> None:
    assert not looks_like_bot_challenge("")
    assert not looks_like_bot_challenge("A normal article about web scraping.")
    # Article mentioning Cloudflare in passing shouldn't trip — we only match
    # against signatures unique to challenge pages.
    assert not looks_like_bot_challenge(
        "Our team migrated to Cloudflare last quarter to improve TLS handshake latency."
    )


def test_looks_like_spa_shell_recognizes_empty_client_app_shell() -> None:
    html = (
        "<!doctype html><html data-beasties-container><head><title>PAIR Guidebook</title>"
        '<script src="main.js"></script></head><body><app-root></app-root></body></html>'
    )
    assert looks_like_spa_shell(html, "PAIR Guidebook")


def test_looks_like_cookie_notice_only_recognizes_short_consent_text() -> None:
    assert looks_like_cookie_notice_only(
        "This site uses cookies from Google to deliver and enhance the quality "
        "of its services and to analyze traffic. Learn more OK, got it"
    )
    assert not looks_like_cookie_notice_only(
        "This article explains how product teams can use cookies for analytics. " * 30
    )


def test_web_handler_raises_js_required_for_cookie_notice_only() -> None:
    html = (
        "<html><head><title>PAIR Guidebook</title></head><body>"
        "<p>This site uses cookies from Google to deliver and enhance the quality "
        "of its services and to analyze traffic.</p><a>Learn more</a><button>OK, got it</button>"
        "</body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(JsRenderingRequired) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://pair.withgoogle.com/guidebook/"))
    assert "cookie consent" in str(exc_info.value)
    handler.close()


def test_web_handler_raises_js_required_for_empty_spa_shell() -> None:
    html = (
        "<!doctype html><html data-beasties-container><head><title>PAIR Guidebook</title>"
        '<script src="main.js"></script></head><body><app-root></app-root></body></html>'
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text=html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(JsRenderingRequired) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://pair.withgoogle.com/guidebook/"))
    assert "JavaScript app shell" in str(exc_info.value)
    handler.close()


def test_web_handler_raises_js_required_for_cloudflare_403() -> None:
    """The exact failure mode the user hit on Medium: 403 + Cloudflare interstitial.
    Before Fix A this raised HandlerHardFailure (dead end). Now it raises
    JsRenderingRequired so compile's existing auto-fallback retries with web_js."""
    cf_html = (
        '<!DOCTYPE html><html lang="en-US"><head><title>Just a moment...</title>'
        '<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">'
        '</head><body><div class="cf-browser-verification">…</div></body></html>'
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(403, text=cf_html)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(JsRenderingRequired) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://medium.com/some/article"))
    assert "bot-detection" in str(exc_info.value)
    handler.close()


def test_web_handler_raises_js_required_for_vercel_429() -> None:
    vercel_html = (
        '<!DOCTYPE html><html lang="en" data-astro-cid-nbv56vs3>'
        "<head><title>Vercel Security Checkpoint</title></head>"
        "<body><p>We're verifying your browser</p>"
        "<a>Website owner? Click here to fix</a></body></html>"
    )

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            429,
            headers={
                "content-type": "text/html; charset=utf-8",
                "x-vercel-mitigated": "challenge",
            },
            text=vercel_html,
        )

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(JsRenderingRequired) as exc_info:
        handler.fetch(
            SourceSpec(
                type="web",
                url="https://brianlovin.com/writing/the-meta-skills-of-product-design-dm5y2kl",
            )
        )
    message = str(exc_info.value)
    assert "category: js_required" in message
    assert "HTTP 429" in message
    assert "bot-detection" in message
    handler.close()


def test_web_handler_still_raises_hard_failure_for_plain_403() -> None:
    """A 403 without bot-challenge markers stays a hard failure — we don't
    want to fire up Playwright for legitimate access-denied responses where
    JS rendering wouldn't help."""

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            403,
            text="<html><body><h1>Forbidden</h1><p>Access denied for this resource.</p></body></html>",
        )

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    # "Access denied" IS in the bot-challenge list (Akamai variant) so this
    # should actually trigger JsRequired. The test here is the opposite case:
    # a 403 with a clean Forbidden body that has nothing vendor-specific.
    def fake_transport_clean(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            403,
            headers={"content-type": "text/html; charset=utf-8"},
            text="<html><body><h1>Forbidden</h1><p>You do not have permission.</p></body></html>",
        )

    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport_clean))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/private"))
    # Specifically, not JsRenderingRequired (that's a subclass — check exact type).
    assert not isinstance(exc_info.value, JsRenderingRequired)
    message = str(exc_info.value)
    assert "HTTP 403" in message
    assert "category: forbidden" in message
    assert "content-type: text/html" in message
    assert "body preview:" in message
    assert "You do not have permission" in message
    handler.close()


def test_web_handler_failure_detail_truncates_response_preview() -> None:
    body = "<html><body>" + ("private failure detail " * 80) + "tail-marker</body></html>"

    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            404,
            headers={"content-type": "text/html"},
            text=body,
        )

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/missing"))
    message = str(exc_info.value)
    assert "category: not_found" in message
    assert "status: HTTP 404" in message
    assert "body preview:" in message
    assert "... [truncated]" in message
    assert "tail-marker" not in message
    handler.close()


def test_web_handler_failure_detail_classifies_paywall() -> None:
    def fake_transport(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            403,
            headers={"content-type": "text/html"},
            text="<html><body><h1>Subscribe to continue</h1><p>Sign in to read this story.</p></body></html>",
        )

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/paywalled"))
    message = str(exc_info.value)
    assert "category: paywall" in message
    assert "Subscribe to continue" in message
    handler.close()


def test_web_handler_network_errors_are_categorized() -> None:
    def fake_transport(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("TLS certificate verify failed", request=request)

    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/tls"))
    message = str(exc_info.value)
    assert "category: tls" in message
    assert "ConnectError" in message
    handler.close()
