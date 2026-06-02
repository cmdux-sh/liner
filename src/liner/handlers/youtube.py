from __future__ import annotations

import re
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlparse

from liner.config import FetchConfig
from liner.handlers.base import HandlerHardFailure
from liner.types import SourceContent, SourceSpec

VIDEO_ID_RE = re.compile(r"^[A-Za-z0-9_-]{11}$")


class YouTubeHandler:
    def __init__(self, config: FetchConfig) -> None:
        self._config = config

    def fetch(self, spec: SourceSpec) -> SourceContent:
        url = spec.url
        video_id = _extract_video_id(url)
        if video_id is None:
            raise HandlerHardFailure(
                f"Could not extract a YouTube video ID from {url!r}.", url
            )

        metadata = self._fetch_metadata(video_id, url)
        transcript_text, transcript_meta = self._fetch_transcript(video_id, url)

        fetched_at = datetime.now(UTC).isoformat()
        merged_metadata: dict[str, Any] = {**metadata.get("extra", {}), **transcript_meta}

        return SourceContent(
            title=metadata.get("title") or url,
            url=url,
            body=transcript_text,
            fetched_at=fetched_at,
            author=metadata.get("author"),
            published_at=metadata.get("published_at"),
            duration_seconds=metadata.get("duration_seconds"),
            metadata=merged_metadata,
        )

    def _fetch_metadata(self, video_id: str, url: str) -> dict[str, Any]:
        from yt_dlp import YoutubeDL
        from yt_dlp.utils import DownloadError

        opts: dict[str, Any] = {
            "quiet": True,
            "no_warnings": True,
            "skip_download": True,
            "extract_flat": False,
            "noprogress": True,
        }
        cookies = self._config.cookies_file
        if cookies is not None:
            opts["cookiefile"] = str(cookies)

        try:
            with YoutubeDL(opts) as ydl:
                info = ydl.extract_info(
                    f"https://www.youtube.com/watch?v={video_id}", download=False
                )
        except DownloadError as e:
            _raise_yt_dlp_error(e, url)
            raise  # unreachable

        upload_date = info.get("upload_date")  # YYYYMMDD
        published_at: str | None = None
        if upload_date and len(upload_date) == 8:
            published_at = f"{upload_date[0:4]}-{upload_date[4:6]}-{upload_date[6:8]}"

        return {
            "title": info.get("title"),
            "author": info.get("uploader") or info.get("channel"),
            "published_at": published_at,
            "duration_seconds": int(info["duration"]) if info.get("duration") else None,
            "extra": {
                "video_id": video_id,
                "channel_url": info.get("channel_url"),
                "view_count": info.get("view_count"),
            },
        }

    def _fetch_transcript(self, video_id: str, url: str) -> tuple[str, dict[str, Any]]:
        try:
            text, meta = _fetch_via_transcript_api(video_id, self._config.cookies_file)
            return _normalize_transcript(text), meta
        except _TranscriptUnavailable as e:
            primary_msg = str(e)
        except _RateLimitedError as e:
            raise HandlerHardFailure(
                f"YouTube is rate-limiting your IP ({e}). Wait and retry, "
                "use a different network, or provide --cookies.",
                url,
            ) from e

        try:
            text, meta = _fetch_via_yt_dlp_subs(video_id, self._config.cookies_file)
            meta.setdefault("transcript_fallback_reason", primary_msg)
            return _normalize_transcript(text), meta
        except _TranscriptUnavailable as e:
            raise HandlerHardFailure(
                f"Could not get transcript for {url}: {primary_msg}; yt-dlp fallback: {e}",
                url,
            ) from e


def _extract_video_id(url: str) -> str | None:
    parsed = urlparse(url)
    host = (parsed.netloc or "").lower()

    if host.endswith("youtu.be"):
        candidate = parsed.path.lstrip("/")
        return candidate if VIDEO_ID_RE.match(candidate) else None

    if "youtube.com" in host:
        path = parsed.path
        if path == "/watch":
            vals = parse_qs(parsed.query).get("v")
            if vals and VIDEO_ID_RE.match(vals[0]):
                return vals[0]
        for prefix in ("/embed/", "/shorts/", "/v/"):
            if path.startswith(prefix):
                candidate = path[len(prefix):].split("/", 1)[0]
                return candidate if VIDEO_ID_RE.match(candidate) else None

    if VIDEO_ID_RE.match(url):
        return url
    return None


class _TranscriptUnavailable(Exception):
    pass


class _RateLimitedError(Exception):
    pass


def _fetch_via_transcript_api(
    video_id: str, cookies: Path | None
) -> tuple[str, dict[str, Any]]:
    from youtube_transcript_api import (
        NoTranscriptFound,
        TranscriptsDisabled,
        YouTubeTranscriptApi,
    )
    from youtube_transcript_api._errors import (
        AgeRestricted,
        IpBlocked,
        RequestBlocked,
        VideoUnavailable,
        VideoUnplayable,
        YouTubeRequestFailed,
    )

    http_client = _session_with_cookies(cookies) if cookies is not None else None
    api = YouTubeTranscriptApi(http_client=http_client)

    try:
        transcript_list = api.list(video_id)
    except TranscriptsDisabled as e:
        raise _TranscriptUnavailable("captions are disabled for this video") from e
    except (VideoUnavailable, VideoUnplayable) as e:
        raise _TranscriptUnavailable(f"video unavailable: {e}") from e
    except AgeRestricted as e:
        raise _TranscriptUnavailable(
            f"age-restricted; provide --cookies to access: {e}"
        ) from e
    except (IpBlocked, RequestBlocked) as e:
        raise _RateLimitedError(str(e)) from e
    except YouTubeRequestFailed as e:
        raise _TranscriptUnavailable(f"YouTube request failed: {e}") from e

    chosen = None
    is_generated = False
    # Preference: manual English → any manual → English auto → any auto.
    try:
        chosen = transcript_list.find_manually_created_transcript(["en", "en-US", "en-GB"])
    except NoTranscriptFound:
        pass
    if chosen is None:
        for t in transcript_list:
            if not t.is_generated:
                chosen = t
                break
    if chosen is None:
        try:
            chosen = transcript_list.find_generated_transcript(["en", "en-US", "en-GB"])
            is_generated = True
        except NoTranscriptFound:
            pass
    if chosen is None:
        for t in transcript_list:
            chosen = t
            is_generated = t.is_generated
            break
    if chosen is None:
        raise _TranscriptUnavailable("no transcript tracks available")

    try:
        fetched = chosen.fetch()
    except (IpBlocked, RequestBlocked) as e:
        raise _RateLimitedError(str(e)) from e

    snippets = getattr(fetched, "snippets", fetched)  # FetchedTranscript or list
    text = " ".join(_snippet_text(s) for s in snippets if _snippet_text(s))
    meta = {
        "transcript_source": "youtube-transcript-api",
        "transcript_type": "auto" if (is_generated or chosen.is_generated) else "manual",
        "transcript_language": getattr(chosen, "language_code", None),
    }
    return text, meta


def _snippet_text(snippet: object) -> str:
    if isinstance(snippet, dict):
        return str(snippet.get("text", ""))
    return str(getattr(snippet, "text", ""))


def _session_with_cookies(cookies: Path) -> Any:
    import http.cookiejar

    import requests

    session = requests.Session()
    jar = http.cookiejar.MozillaCookieJar(str(cookies))
    jar.load(ignore_discard=True, ignore_expires=True)
    session.cookies = jar  # type: ignore[assignment]
    return session


def _fetch_via_yt_dlp_subs(
    video_id: str, cookies: Path | None
) -> tuple[str, dict[str, Any]]:
    import tempfile

    from yt_dlp import YoutubeDL
    from yt_dlp.utils import DownloadError

    with tempfile.TemporaryDirectory() as tmp:
        outtmpl = str(Path(tmp) / "%(id)s.%(ext)s")
        opts: dict[str, Any] = {
            "quiet": True,
            "no_warnings": True,
            "skip_download": True,
            "writesubtitles": True,
            "writeautomaticsub": True,
            "subtitlesformat": "vtt",
            "subtitleslangs": ["en", "en-US", "en-orig", "en.*"],
            "outtmpl": outtmpl,
            "noprogress": True,
        }
        if cookies is not None:
            opts["cookiefile"] = str(cookies)

        try:
            with YoutubeDL(opts) as ydl:
                ydl.download([f"https://www.youtube.com/watch?v={video_id}"])
        except DownloadError as e:
            _raise_yt_dlp_error(e, f"https://www.youtube.com/watch?v={video_id}")
            raise _TranscriptUnavailable(str(e)) from e  # unreachable

        vtt_files = sorted(Path(tmp).glob(f"{video_id}*.vtt"))
        if not vtt_files:
            raise _TranscriptUnavailable("yt-dlp produced no subtitle files")

        # Prefer manual over auto: yt-dlp names auto-captions like "<id>.en.vtt"
        # vs. manual same naming. Just pick the first deterministically.
        text = _parse_vtt(vtt_files[0].read_text(encoding="utf-8"))
        return text, {
            "transcript_source": "yt-dlp",
            "transcript_type": "auto",
            "transcript_language": _lang_from_filename(vtt_files[0].name, video_id),
        }


def _lang_from_filename(name: str, video_id: str) -> str | None:
    # e.g. "abc12345678.en.vtt" → "en"
    stem = name.removeprefix(video_id).removesuffix(".vtt").strip(".")
    return stem or None


def _parse_vtt(content: str) -> str:
    lines: list[str] = []
    for raw in content.splitlines():
        line = raw.strip()
        if not line:
            continue
        if line.startswith("WEBVTT"):
            continue
        if "-->" in line:
            continue
        if line.isdigit():
            continue
        if line.startswith(("NOTE", "STYLE", "Kind:", "Language:")):
            continue
        cleaned = re.sub(r"<[^>]+>", "", line)
        if cleaned:
            lines.append(cleaned)
    # Dedupe consecutive duplicate lines (auto-captions repeat).
    deduped: list[str] = []
    for line in lines:
        if not deduped or deduped[-1] != line:
            deduped.append(line)
    return " ".join(deduped)


def _normalize_transcript(text: str) -> str:
    # Strip bracketed annotations like [Music], [Applause].
    text = re.sub(r"\[[^\]]+\]", " ", text)
    # Collapse whitespace.
    text = re.sub(r"\s+", " ", text).strip()
    # Split on sentence boundaries to make paragraphs more readable.
    sentences = re.split(r"(?<=[.!?])\s+(?=[A-Z])", text)
    paragraphs: list[str] = []
    buf: list[str] = []
    for sentence in sentences:
        buf.append(sentence)
        if len(" ".join(buf)) > 400:
            paragraphs.append(" ".join(buf))
            buf = []
    if buf:
        paragraphs.append(" ".join(buf))
    return "\n\n".join(paragraphs)


def _raise_yt_dlp_error(e: Exception, url: str) -> None:
    msg = str(e).lower()
    if "sign in to confirm your age" in msg:
        raise HandlerHardFailure(
            f"Video {url} is age-restricted. Provide cookies via --cookies "
            "or [fetch].cookies_file.",
            url,
        ) from e
    if "private video" in msg:
        raise HandlerHardFailure(f"Video {url} is private.", url) from e
    if "video unavailable" in msg or "removed" in msg:
        raise HandlerHardFailure(f"Video {url} is unavailable or removed.", url) from e
    if "http error 429" in msg or "too many requests" in msg:
        raise HandlerHardFailure(
            f"YouTube is rate-limiting your IP when fetching {url}. Wait or use cookies.",
            url,
        ) from e
    raise HandlerHardFailure(f"yt-dlp failed for {url}: {e}", url) from e


__all__ = ["YouTubeHandler"]
