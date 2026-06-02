from __future__ import annotations

import json
import sqlite3
from dataclasses import asdict, dataclass
from datetime import UTC, datetime, timedelta
from pathlib import Path

from liner.types import SourceContent

CURRENT_SCHEMA_VERSION = 1


@dataclass(frozen=True, slots=True)
class CacheInfo:
    path: Path
    entry_count: int
    size_bytes: int


@dataclass(frozen=True, slots=True)
class CacheEntry:
    url: str
    source_type: str
    fetched_at: str
    expires_at: str


class CacheSchemaMismatchError(RuntimeError):
    pass


def _now() -> datetime:
    return datetime.now(UTC)


class SourceCache:
    def __init__(self, path: Path) -> None:
        self.path = path
        path.parent.mkdir(parents=True, exist_ok=True)
        # Store and compare timestamps as ISO 8601 strings — Python 3.12+ deprecated
        # the built-in TIMESTAMP converter, and ISO 8601 sorts lexicographically.
        self._conn = sqlite3.connect(str(path))
        self._conn.row_factory = sqlite3.Row
        self._init_schema()

    def _init_schema(self) -> None:
        cur = self._conn.cursor()
        cur.executescript(
            """
            CREATE TABLE IF NOT EXISTS schema_version (
                version INTEGER PRIMARY KEY
            );
            CREATE TABLE IF NOT EXISTS source_cache (
                url TEXT PRIMARY KEY,
                source_type TEXT NOT NULL,
                content_json TEXT NOT NULL,
                fetched_at TIMESTAMP NOT NULL,
                expires_at TIMESTAMP NOT NULL
            );
            CREATE INDEX IF NOT EXISTS idx_expires ON source_cache(expires_at);
            """
        )
        cur.execute("SELECT MAX(version) FROM schema_version")
        row = cur.fetchone()
        existing = row[0] if row else None
        if existing is None:
            cur.execute("INSERT INTO schema_version(version) VALUES (?)", (CURRENT_SCHEMA_VERSION,))
        elif existing > CURRENT_SCHEMA_VERSION:
            raise CacheSchemaMismatchError(
                f"Cache schema version {existing} is newer than this build's {CURRENT_SCHEMA_VERSION}. "
                "Run `liner cache clear` or upgrade liner."
            )
        # NOTE: when bumping CURRENT_SCHEMA_VERSION, branch here on `existing < CURRENT_SCHEMA_VERSION`
        # and run migration steps before continuing.
        self._conn.commit()

    def get(self, url: str) -> SourceContent | None:
        cur = self._conn.cursor()
        cur.execute(
            "SELECT content_json, expires_at FROM source_cache WHERE url = ? AND expires_at > ?",
            (url, _now().isoformat()),
        )
        row = cur.fetchone()
        if row is None:
            return None
        data = json.loads(row["content_json"])
        return SourceContent(**data)

    def put(self, content: SourceContent, source_type: str, ttl_days: int) -> None:
        fetched_at = _now()
        expires_at = fetched_at + timedelta(days=ttl_days)
        payload = json.dumps(asdict(content), ensure_ascii=False)
        self._conn.execute(
            """
            INSERT INTO source_cache (url, source_type, content_json, fetched_at, expires_at)
            VALUES (?, ?, ?, ?, ?)
            ON CONFLICT(url) DO UPDATE SET
                source_type = excluded.source_type,
                content_json = excluded.content_json,
                fetched_at = excluded.fetched_at,
                expires_at = excluded.expires_at
            """,
            (content.url, source_type, payload, fetched_at.isoformat(), expires_at.isoformat()),
        )
        self._conn.commit()

    def purge(self, url: str) -> bool:
        cur = self._conn.execute("DELETE FROM source_cache WHERE url = ?", (url,))
        self._conn.commit()
        return cur.rowcount > 0

    def clear(self) -> int:
        cur = self._conn.execute("DELETE FROM source_cache")
        self._conn.commit()
        return cur.rowcount

    def get_raw(self, url: str) -> SourceContent | None:
        """Like `get` but bypasses TTL expiry — for cache inspection commands."""
        cur = self._conn.cursor()
        cur.execute("SELECT content_json FROM source_cache WHERE url = ?", (url,))
        row = cur.fetchone()
        if row is None:
            return None
        data = json.loads(row["content_json"])
        return SourceContent(**data)

    def list(self, limit: int = 100, offset: int = 0) -> list[CacheEntry]:
        cur = self._conn.execute(
            """
            SELECT url, source_type, fetched_at, expires_at
            FROM source_cache
            ORDER BY fetched_at DESC
            LIMIT ? OFFSET ?
            """,
            (limit, offset),
        )
        return [
            CacheEntry(
                url=row["url"],
                source_type=row["source_type"],
                fetched_at=row["fetched_at"],
                expires_at=row["expires_at"],
            )
            for row in cur.fetchall()
        ]

    def info(self) -> CacheInfo:
        cur = self._conn.execute("SELECT COUNT(*) AS n FROM source_cache")
        count = int(cur.fetchone()["n"])
        size = self.path.stat().st_size if self.path.exists() else 0
        return CacheInfo(path=self.path, entry_count=count, size_bytes=size)

    def close(self) -> None:
        self._conn.close()

    def __enter__(self) -> SourceCache:
        return self

    def __exit__(self, *exc_info: object) -> None:
        self.close()
