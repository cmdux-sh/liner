"""Keep source fetches on the public internet and within a size limit.

A tape can come from a stranger, so its URLs must not reach this machine's own
services, its local network, or a cloud metadata address. People who keep
sources on an intranet opt in with `allow_private_network` in `config.toml`.
"""

from __future__ import annotations

import ipaddress
import socket
from typing import Any
from urllib.parse import urlparse

import httpx

MAX_RESPONSE_BYTES = 50 * 1024 * 1024
_NAT64 = ipaddress.ip_network("64:ff9b::/96")
_REMOVED_AFTER_DECODING = {"content-encoding", "content-length", "transfer-encoding"}


class BlockedAddressError(httpx.RequestError):
    """The URL points at a private address, and private fetching is off."""


class ResponseTooLargeError(httpx.RequestError):
    """The response passed the size limit, so it was dropped."""


def blocked_reason(url: str) -> str | None:
    """Explain why this URL can't be fetched, or return None if it can.

    A name that doesn't resolve isn't blocked here, so the fetch reports it the
    usual way. Other schemes are left to the caller.
    """
    parsed = urlparse(url)
    if parsed.scheme not in {"http", "https", "ws", "wss"} or not parsed.hostname:
        return None
    try:
        results = socket.getaddrinfo(parsed.hostname, None, type=socket.SOCK_STREAM)
    except (OSError, UnicodeError):
        return None
    for result in results:
        address = str(result[4][0])
        if _is_private(address):
            return (
                f"{parsed.hostname} points to a private address ({address}), so Liner won't "
                "fetch it. To allow this, set allow_private_network = true under [fetch] in "
                "~/.liner/config.toml."
            )
    return None


def guarded_client(*, allow_private_network: bool = False, **options: Any) -> httpx.Client:
    """An httpx client that checks the first request and every redirect."""
    if allow_private_network:
        return httpx.Client(**options)
    return httpx.Client(event_hooks={"request": [_refuse_private]}, **options)


def get_bounded(
    client: httpx.Client,
    url: str,
    *,
    max_bytes: int = MAX_RESPONSE_BYTES,
    **options: Any,
) -> httpx.Response:
    """GET a URL, stopping as soon as the decoded body passes `max_bytes`."""
    with client.stream("GET", url, **options) as response:
        chunks: list[bytes] = []
        size = 0
        for chunk in response.iter_bytes():
            size += len(chunk)
            if size > max_bytes:
                raise ResponseTooLargeError(
                    f"{url} is larger than {max_bytes // (1024 * 1024)} MB, so it was not read.",
                    request=response.request,
                )
            chunks.append(chunk)
    # The chunks are already decoded, so a second pass must not decode them again.
    # Raw bytes, because a header value may hold non-ASCII text, such as an emoji.
    headers = [
        (name, value)
        for name, value in response.headers.raw
        if name.decode("latin-1").lower() not in _REMOVED_AFTER_DECODING
    ]
    return httpx.Response(
        response.status_code,
        headers=headers,
        content=b"".join(chunks),
        request=response.request,
        history=response.history,
    )


def _refuse_private(request: httpx.Request) -> None:
    reason = blocked_reason(str(request.url))
    if reason is not None:
        raise BlockedAddressError(reason, request=request)


def _is_private(address: str) -> bool:
    ip = ipaddress.ip_address(address.split("%", 1)[0])
    if isinstance(ip, ipaddress.IPv6Address):
        ip = _embedded_ipv4(ip) or ip
    return not ip.is_global or ip.is_multicast or ip.is_reserved or ip.is_unspecified


def _embedded_ipv4(ip: ipaddress.IPv6Address) -> ipaddress.IPv4Address | None:
    if ip in _NAT64:
        return ipaddress.IPv4Address(int(ip) & 0xFFFFFFFF)
    return ip.ipv4_mapped or ip.sixtofour


__all__ = [
    "MAX_RESPONSE_BYTES",
    "BlockedAddressError",
    "ResponseTooLargeError",
    "blocked_reason",
    "get_bounded",
    "guarded_client",
]
