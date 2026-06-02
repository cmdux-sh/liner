from __future__ import annotations

from datetime import UTC, datetime, timedelta
from pathlib import Path

import pytest

from liner import cache as cache_mod
from liner.cache import SourceCache
from liner.types import SourceContent


def _make_content(url: str = "https://example.com/x", body: str = "hello") -> SourceContent:
    return SourceContent(
        title="t",
        url=url,
        body=body,
        fetched_at="2026-05-16T00:00:00+00:00",
        author=None,
        published_at=None,
        duration_seconds=None,
        metadata={"k": "v"},
    )


def test_put_and_get_round_trip(tmp_path: Path) -> None:
    cache = SourceCache(tmp_path / "c.db")
    content = _make_content()
    cache.put(content, "web", ttl_days=7)
    got = cache.get(content.url)
    assert got is not None
    assert got.url == content.url
    assert got.body == "hello"
    assert got.metadata == {"k": "v"}


def test_get_returns_none_when_expired(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    cache = SourceCache(tmp_path / "c.db")
    content = _make_content()

    fake_now = datetime(2026, 1, 1, tzinfo=UTC)
    monkeypatch.setattr(cache_mod, "_now", lambda: fake_now)
    cache.put(content, "web", ttl_days=1)

    monkeypatch.setattr(cache_mod, "_now", lambda: fake_now + timedelta(days=2))
    assert cache.get(content.url) is None


def test_purge_and_clear(tmp_path: Path) -> None:
    cache = SourceCache(tmp_path / "c.db")
    cache.put(_make_content("https://example.com/a"), "web", 7)
    cache.put(_make_content("https://example.com/b"), "web", 7)
    assert cache.info().entry_count == 2
    assert cache.purge("https://example.com/a") is True
    assert cache.info().entry_count == 1
    cache.clear()
    assert cache.info().entry_count == 0


def test_info_reports_size(tmp_path: Path) -> None:
    cache = SourceCache(tmp_path / "c.db")
    cache.put(_make_content(), "web", 7)
    info = cache.info()
    assert info.path == tmp_path / "c.db"
    assert info.entry_count == 1
    assert info.size_bytes > 0
