from __future__ import annotations

import gzip
import socket
import threading
from collections.abc import Iterator
from dataclasses import replace
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

import httpx
import pytest
from typer.testing import CliRunner

from liner.cli import app
from liner.config import FetchConfig, load_config
from liner.handlers import fetch_guard
from liner.handlers.base import HandlerHardFailure
from liner.handlers.fetch_guard import (
    BlockedAddressError,
    ResponseTooLargeError,
    blocked_reason,
    get_bounded,
    guarded_client,
)
from liner.handlers.web import WebHandler
from liner.handlers.web_js import _content_from_pdf_download
from liner.types import SourceSpec

ARTICLE = "<html><title>Local</title><body><p>" + "Readable words here. " * 40 + "</p></body></html>"


class _Local(BaseHTTPRequestHandler):
    hits = 0

    def do_GET(self) -> None:  # noqa: N802
        type(self).hits += 1
        body = ARTICLE.encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_: object) -> None:
        return


@pytest.fixture
def local_server() -> Iterator[str]:
    _Local.hits = 0
    server = HTTPServer(("127.0.0.1", 0), _Local)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    yield f"http://127.0.0.1:{server.server_port}/secret"
    server.shutdown()
    server.server_close()


def _resolves_to(monkeypatch: pytest.MonkeyPatch, *addresses: str) -> None:
    def fake(*_: object, **__: object) -> list[tuple[object, ...]]:
        return [(socket.AF_INET, socket.SOCK_STREAM, 6, "", (a, 443)) for a in addresses]

    monkeypatch.setattr(fetch_guard.socket, "getaddrinfo", fake)


@pytest.mark.parametrize(
    "address",
    [
        "127.0.0.1",
        "10.1.2.3",
        "172.16.0.1",
        "192.168.1.1",
        "169.254.169.254",
        "100.100.100.100",
        "0.0.0.0",
        "224.0.0.1",
        "::1",
        "fe80::1",
        "fd00::1",
        "::ffff:127.0.0.1",
        "64:ff9b::7f00:1",
        "2002:7f00:1::",
    ],
)
def test_private_addresses_are_blocked(monkeypatch: pytest.MonkeyPatch, address: str) -> None:
    _resolves_to(monkeypatch, address)
    assert address in (blocked_reason("https://example.com/") or "")


def test_a_public_address_is_allowed(monkeypatch: pytest.MonkeyPatch) -> None:
    _resolves_to(monkeypatch, "93.184.216.34")
    assert blocked_reason("https://example.com/") is None


def test_one_private_answer_among_public_ones_is_blocked(monkeypatch: pytest.MonkeyPatch) -> None:
    _resolves_to(monkeypatch, "93.184.216.34", "127.0.0.1")
    assert blocked_reason("https://example.com/") is not None


def test_a_number_written_as_a_host_is_resolved_first() -> None:
    assert blocked_reason("http://2130706433/") is not None


def test_a_name_that_does_not_resolve_is_left_to_the_fetch(monkeypatch: pytest.MonkeyPatch) -> None:
    def fail(*_: object, **__: object) -> None:
        raise socket.gaierror("no such host")

    monkeypatch.setattr(fetch_guard.socket, "getaddrinfo", fail)
    assert blocked_reason("https://nowhere.invalid/") is None


def test_web_handler_refuses_a_private_address(local_server: str) -> None:
    handler = WebHandler(FetchConfig())
    with pytest.raises(HandlerHardFailure, match="private address"):
        handler.fetch(SourceSpec(type="web", url=local_server))
    assert _Local.hits == 0


def test_a_person_can_allow_private_addresses(local_server: str) -> None:
    handler = WebHandler(FetchConfig(allow_private_network=True))
    content = handler.fetch(SourceSpec(type="web", url=local_server))
    assert "Readable words" in content.body
    assert _Local.hits == 1


def test_a_redirect_to_a_private_address_is_refused(monkeypatch: pytest.MonkeyPatch) -> None:
    requested: list[str] = []

    def transport(request: httpx.Request) -> httpx.Response:
        requested.append(str(request.url))
        return httpx.Response(302, headers={"location": "http://169.254.169.254/latest"})

    real = fetch_guard.blocked_reason
    monkeypatch.setattr(
        fetch_guard,
        "blocked_reason",
        lambda url: None if "public.example" in url else real(url),
    )
    client = guarded_client(transport=httpx.MockTransport(transport), follow_redirects=True)
    with pytest.raises(BlockedAddressError):
        get_bounded(client, "https://public.example/")
    assert requested == ["https://public.example/"]


def test_an_oversized_body_stops_early() -> None:
    sent: list[int] = []

    def endless() -> Iterator[bytes]:
        for index in range(1000):
            sent.append(index)
            yield b"x" * 1024

    client = httpx.Client(
        transport=httpx.MockTransport(lambda _: httpx.Response(200, content=endless()))
    )
    with pytest.raises(ResponseTooLargeError):
        get_bounded(client, "https://example.com/", max_bytes=4096)
    assert len(sent) < 10


def test_a_compressed_body_is_limited_by_its_decoded_size() -> None:
    bomb = gzip.compress(b"\0" * 1_000_000)
    client = httpx.Client(
        transport=httpx.MockTransport(
            lambda _: httpx.Response(200, headers={"content-encoding": "gzip"}, content=bomb)
        )
    )
    with pytest.raises(ResponseTooLargeError):
        get_bounded(client, "https://example.com/", max_bytes=100_000)


def test_a_compressed_body_within_the_limit_reads_normally() -> None:
    client = httpx.Client(
        transport=httpx.MockTransport(
            lambda _: httpx.Response(
                200,
                headers={"content-encoding": "gzip", "content-type": "text/plain; charset=utf-8"},
                content=gzip.compress("héllo".encode()),
            )
        )
    )
    response = get_bounded(client, "https://example.com/")
    assert response.text == "héllo"
    assert str(response.url) == "https://example.com/"
    assert response.headers["content-type"].startswith("text/plain")


def test_web_handler_reports_an_oversized_page(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(fetch_guard, "MAX_RESPONSE_BYTES", 10)
    monkeypatch.setattr(get_bounded, "__kwdefaults__", {"max_bytes": 10})
    handler = WebHandler(FetchConfig())
    handler._client = httpx.Client(
        transport=httpx.MockTransport(lambda _: httpx.Response(200, content=b"x" * 100))
    )
    with pytest.raises(HandlerHardFailure, match="larger than"):
        handler.fetch(SourceSpec(type="web", url="https://example.com/"))


def test_config_reads_allow_private_network(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    path.write_text("[fetch]\nallow_private_network = true\n", encoding="utf-8")
    assert load_config(path).fetch.allow_private_network is True
    assert load_config(tmp_path / "missing.toml").fetch.allow_private_network is False


def test_config_requires_a_real_true(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    path.write_text('[fetch]\nallow_private_network = "false"\n', encoding="utf-8")
    assert load_config(path).fetch.allow_private_network is False


def test_clone_refuses_a_private_address(local_server: str, tmp_path: Path) -> None:
    target = tmp_path / "tape.yaml"
    result = CliRunner().invoke(app, ["clone", local_server, str(target)])
    assert result.exit_code == 1
    assert not target.exists()
    assert _Local.hits == 0


def test_js_handler_blocks_private_requests_and_allows_public_ones(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    from liner.handlers import web_js

    class Route:
        def __init__(self, url: str) -> None:
            self.request = type("Request", (), {"url": url})()
            self.outcome = ""

        def abort(self, _: str) -> None:
            self.outcome = "aborted"

        def continue_(self) -> None:
            self.outcome = "continued"

    monkeypatch.setattr(web_js, "PLAYWRIGHT_AVAILABLE", True)
    monkeypatch.setattr(
        web_js, "blocked_reason", lambda url: "private" if "internal" in url else None
    )
    handler = web_js.WebJsHandler(replace(FetchConfig()))
    private, public = Route("http://internal.test/a"), Route("https://example.com/a")
    handler._guard_route(private)
    handler._guard_route(public)
    assert (private.outcome, public.outcome) == ("aborted", "continued")


def test_js_handler_refuses_a_private_url_before_starting_a_browser(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    from liner.handlers import web_js

    monkeypatch.setattr(web_js, "PLAYWRIGHT_AVAILABLE", True)
    monkeypatch.setattr(web_js, "blocked_reason", lambda _: "private address")
    handler = web_js.WebJsHandler(FetchConfig())
    monkeypatch.setattr(handler, "_ensure_browser", lambda: pytest.fail("browser started"))
    with pytest.raises(HandlerHardFailure, match="private address"):
        handler.fetch(SourceSpec(type="web", url="http://internal.test/"))


def test_an_oversized_pdf_download_is_refused(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    from liner.handlers import web_js

    pdf = tmp_path / "big.pdf"
    pdf.write_bytes(b"x" * 100)
    monkeypatch.setattr(web_js, "MAX_RESPONSE_BYTES", 10)
    download = type("Download", (), {"path": lambda _: str(pdf)})()
    with pytest.raises(HandlerHardFailure, match="too large"):
        _content_from_pdf_download(SourceSpec(type="web", url="https://e.com/a.pdf"), download)


def test_js_handler_walks_redirects_before_the_browser_starts(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    from liner.handlers import web_js

    monkeypatch.setattr(web_js, "PLAYWRIGHT_AVAILABLE", True)
    monkeypatch.setattr(web_js, "blocked_reason", lambda _: None)

    def refuse(*_: object, **__: object) -> httpx.Client:
        def hook(request: httpx.Request) -> None:
            raise BlockedAddressError("points to a private address", request=request)

        return httpx.Client(
            event_hooks={"request": [hook]},
            transport=httpx.MockTransport(lambda _: httpx.Response(200)),
        )

    monkeypatch.setattr(web_js, "guarded_client", refuse)
    handler = web_js.WebJsHandler(FetchConfig())
    monkeypatch.setattr(handler, "_ensure_browser", lambda: pytest.fail("browser started"))
    with pytest.raises(HandlerHardFailure, match="private address"):
        handler.fetch(SourceSpec(type="web", url="https://example.com/"))


def test_a_bad_port_is_left_to_the_fetch(monkeypatch: pytest.MonkeyPatch) -> None:
    _resolves_to(monkeypatch, "93.184.216.34")
    assert blocked_reason("http://example.com:99999/") is None


def test_js_handler_closes_private_web_sockets(monkeypatch: pytest.MonkeyPatch) -> None:
    from liner.handlers import web_js

    class Socket:
        def __init__(self, url: str) -> None:
            self.url = url
            self.outcome = ""

        def close(self) -> None:
            self.outcome = "closed"

        def connect_to_server(self) -> None:
            self.outcome = "connected"

    monkeypatch.setattr(web_js, "PLAYWRIGHT_AVAILABLE", True)
    monkeypatch.setattr(
        web_js, "blocked_reason", lambda url: "private" if "internal" in url else None
    )
    handler = web_js.WebJsHandler(FetchConfig())
    private, public = Socket("ws://internal.test/a"), Socket("wss://example.com/a")
    handler._guard_web_socket(private)
    handler._guard_web_socket(public)
    assert (private.outcome, public.outcome) == ("closed", "connected")


def test_a_header_with_non_ascii_text_is_kept() -> None:
    from liner.handlers.fetch_guard import get_bounded

    snowman = "⛄ winter".encode()
    client = httpx.Client(
        transport=httpx.MockTransport(
            lambda request: httpx.Response(
                200, headers=[(b"x-note", snowman)], content=b"<html>ok</html>"
            )
        )
    )

    response = get_bounded(client, "https://example.com/")

    assert response.headers.raw[0] == (b"x-note", snowman)
    assert response.text == "<html>ok</html>"
