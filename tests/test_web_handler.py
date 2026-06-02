from __future__ import annotations

import httpx
import pytest

from liner.config import FetchConfig
from liner.handlers import html_extraction
from liner.handlers.base import HandlerHardFailure, HandlerSoftFailure, JsRenderingRequired
from liner.handlers.html_extraction import (
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
    # Imperva Incapsula.
    assert looks_like_bot_challenge("Request unsuccessful. Incapsula incident ID: 1234")
    # Distil / Imperva "Pardon our interruption" page.
    assert looks_like_bot_challenge("<h1>Pardon Our Interruption</h1>")
    # PerimeterX press-and-hold challenge.
    assert looks_like_bot_challenge("<p>Please verify you are a human</p>")


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
            text="<html><body><h1>Forbidden</h1><p>You do not have permission.</p></body></html>",
        )

    handler._client = httpx.Client(transport=httpx.MockTransport(fake_transport_clean))

    with pytest.raises(HandlerHardFailure) as exc_info:
        handler.fetch(SourceSpec(type="web", url="https://example.com/private"))
    # Specifically, not JsRenderingRequired (that's a subclass — check exact type).
    assert not isinstance(exc_info.value, JsRenderingRequired)
    assert "HTTP 403" in str(exc_info.value)
    handler.close()
