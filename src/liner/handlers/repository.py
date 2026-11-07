"""Snapshot the documentation of a public GitHub repository at one commit."""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import PurePosixPath
from urllib.parse import quote, urlparse

import httpx

from liner.handlers.base import HandlerHardFailure
from liner.handlers.fetch_guard import get_bounded, guarded_client
from liner.types import SourceContent, SourceSpec

DOCUMENT_EXTENSIONS = {".md", ".mdx", ".markdown", ".rst", ".txt", ".adoc"}
MAX_FILES = 60
MAX_FILE_BYTES = 512 * 1024
MAX_TOTAL_BYTES = 2 * 1024 * 1024

_API = "https://api.github.com/repos"
_RAW = "https://raw.githubusercontent.com"
_SKIPPED_FOLDERS = {"node_modules", "vendor", "dist", "build", "target", "__pycache__"}


def parse_repository_url(url: str) -> tuple[str, str, str | None, str]:
    """Return owner, repository, ref, and folder from a github.com URL."""
    parsed = urlparse(url)
    parts = [part for part in parsed.path.split("/") if part]
    if parsed.netloc.lower() != "github.com" or len(parts) < 2:
        raise ValueError(f"{url} is not a GitHub repository URL")
    owner, repository = parts[0], parts[1].removesuffix(".git")
    if len(parts) == 2:
        return owner, repository, None, ""
    if parts[2] == "tree" and len(parts) >= 4:
        return owner, repository, parts[3], "/".join(parts[4:])
    raise ValueError(f"{url} must be a repository or a /tree/<ref>/<folder> URL")


class RepositoryHandler:
    def __init__(self) -> None:
        self._client = guarded_client(timeout=20.0, follow_redirects=True)

    def fetch(self, spec: SourceSpec) -> SourceContent:
        try:
            owner, repository, ref, folder = parse_repository_url(spec.url)
            slug = f"{owner}/{repository}"
            commit = self._commit(slug, ref)
            chosen, available = self._documents(slug, commit, folder)
            files = [(path, self._text(slug, commit, path)) for path in chosen]
        except (httpx.HTTPError, ValueError, KeyError, TypeError) as error:
            raise HandlerHardFailure(_failure(spec.url, error), spec.url) from error
        if not files:
            raise HandlerHardFailure(f"{spec.url} has no documentation files.", spec.url)
        title = f"{slug}/{folder}" if folder else slug
        return SourceContent(
            title=title,
            url=spec.url,
            body="\n\n".join(f"# {path}\n\n{text.strip()}" for path, text in files),
            fetched_at=datetime.now(UTC).isoformat(),
            metadata={
                "extraction": "repository",
                "commit": commit,
                "documents": f"{len(files)} of {available}",
            },
        )

    def close(self) -> None:
        self._client.close()

    def _get(self, url: str) -> httpx.Response:
        response = get_bounded(self._client, url)
        response.raise_for_status()
        return response

    def _commit(self, slug: str, ref: str | None) -> str:
        if ref is None:
            ref = str(self._get(f"{_API}/{slug}").json()["default_branch"])
        return str(self._get(f"{_API}/{slug}/commits/{quote(ref, safe='')}").json()["sha"])

    def _documents(self, slug: str, commit: str, folder: str) -> tuple[list[str], str]:
        listing = self._get(f"{_API}/{slug}/git/trees/{commit}?recursive=1").json()
        tree = listing["tree"]
        prefix = f"{folder.rstrip('/')}/" if folder else ""
        candidates = [
            item
            for item in tree
            if item.get("type") == "blob" and _is_document(item["path"], prefix)
        ]
        candidates.sort(key=lambda item: _reading_order(item["path"]))
        chosen: list[str] = []
        total = 0
        for item in candidates:
            size = int(item.get("size", 0))
            if size > MAX_FILE_BYTES or total + size > MAX_TOTAL_BYTES or len(chosen) == MAX_FILES:
                continue
            chosen.append(item["path"])
            total += size
        # GitHub cuts the listing of very large repositories short.
        at_least = "at least " if listing.get("truncated") else ""
        return chosen, f"{at_least}{len(candidates)}"

    def _text(self, slug: str, commit: str, path: str) -> str:
        return self._get(f"{_RAW}/{slug}/{commit}/{quote(path)}").text


def _is_document(path: str, prefix: str) -> bool:
    if not path.startswith(prefix):
        return False
    pure = PurePosixPath(path)
    return pure.suffix.lower() in DOCUMENT_EXTENSIONS and not any(
        part.startswith(".") or part in _SKIPPED_FOLDERS for part in pure.parts
    )


def _reading_order(path: str) -> tuple[bool, int, str]:
    pure = PurePosixPath(path)
    return (pure.stem.lower() != "readme", len(pure.parts), path.lower())


def _failure(url: str, error: Exception) -> str:
    if isinstance(error, httpx.HTTPStatusError) and error.response.status_code in {403, 429}:
        return f"GitHub refused {url}, probably its hourly limit. Try again later."
    return f"Could not read {url}: {error}"
