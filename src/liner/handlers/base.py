from __future__ import annotations

from typing import Protocol

from liner.types import SourceContent, SourceSpec


class HandlerHardFailure(Exception):
    def __init__(self, message: str, url: str) -> None:
        self.url = url
        super().__init__(message)


class JsRenderingRequired(HandlerHardFailure):
    """The site is JavaScript-rendered and the server-side handler can't read it.

    The compile loop catches this specifically: if a `web_js` handler is
    registered, it retries the fetch through Playwright. Otherwise it surfaces
    the standard "run `liner setup-js`" message.
    """


class HandlerSoftFailure(Exception):
    def __init__(self, content: SourceContent, message: str) -> None:
        self.content = content
        super().__init__(message)


class SourceHandler(Protocol):
    """Fetches source content given a `SourceSpec`.

    The full spec (rather than just a URL) is passed so handlers can read
    fields like `render` or `path` they need to do their job.
    """

    def fetch(self, spec: SourceSpec) -> SourceContent: ...
