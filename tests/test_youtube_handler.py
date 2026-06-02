"""Unit tests for the YouTube source handler.

The handler lazy-imports `youtube_transcript_api` and `yt_dlp` inside its
methods, so we patch attributes on those modules with fake classes/functions.
Every code path in `YouTubeHandler.fetch` has at least one test here — these
are the "must pass every time" guarantees that the handler keeps working
across refactors and dependency upgrades.

Real-network smoke testing lives in `test_youtube_handler_integration.py` and
is opt-in via the `network` pytest marker.
"""

from __future__ import annotations

from typing import Any

import pytest

from liner.config import FetchConfig
from liner.handlers.base import HandlerHardFailure
from liner.handlers.youtube import (
    YouTubeHandler,
    _extract_video_id,
    _normalize_transcript,
    _parse_vtt,
)
from liner.types import SourceSpec


# ---------------------------------------------------------------------------
# Pure helpers — no external services involved
# ---------------------------------------------------------------------------


class TestExtractVideoId:
    """Every URL shape we should accept, plus a few we shouldn't."""

    @pytest.mark.parametrize(
        "url,expected",
        [
            ("https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"),
            ("https://youtube.com/watch?v=dQw4w9WgXcQ&t=42s", "dQw4w9WgXcQ"),
            ("https://www.youtube.com/watch?v=dQw4w9WgXcQ&feature=share", "dQw4w9WgXcQ"),
            ("https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"),
            ("https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ"),
            ("https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ"),
            ("https://www.youtube.com/v/dQw4w9WgXcQ", "dQw4w9WgXcQ"),
            ("dQw4w9WgXcQ", "dQw4w9WgXcQ"),  # raw ID
            ("https://m.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"),  # mobile
        ],
    )
    def test_accepts_canonical_forms(self, url: str, expected: str) -> None:
        assert _extract_video_id(url) == expected

    @pytest.mark.parametrize(
        "url",
        [
            "",
            "https://example.com/video/abc",
            "https://www.youtube.com/watch",  # no v= param
            "https://www.youtube.com/watch?v=tooshort",  # < 11 chars
            "https://www.youtube.com/watch?v=way_way_too_long_id",  # > 11 chars
            "not-a-url-at-all",
            "https://youtu.be/",  # no path
        ],
    )
    def test_rejects_garbage(self, url: str) -> None:
        assert _extract_video_id(url) is None


class TestNormalizeTranscript:
    """Bracket stripping, whitespace collapsing, paragraph splitting."""

    def test_strips_bracketed_annotations(self) -> None:
        text = "Hello [Music] world [Applause] welcome"
        out = _normalize_transcript(text)
        assert "[Music]" not in out
        assert "[Applause]" not in out
        assert "Hello" in out and "world" in out and "welcome" in out

    def test_collapses_whitespace(self) -> None:
        text = "a   b\n\n\nc\t\td"
        out = _normalize_transcript(text)
        # Single spaces between tokens, no runs of whitespace inside paragraphs
        assert "   " not in out
        assert "\t" not in out

    def test_splits_long_text_into_paragraphs(self) -> None:
        # Sentence boundaries should produce paragraph breaks once we
        # accumulate more than ~400 chars.
        sentence = "This is a sentence with several words in it. "
        long_text = sentence * 30  # well past the 400-char paragraph threshold
        out = _normalize_transcript(long_text)
        assert "\n\n" in out, "expected paragraph breaks for long transcripts"

    def test_returns_empty_for_empty_input(self) -> None:
        assert _normalize_transcript("") == ""

    def test_handles_only_brackets(self) -> None:
        # All-noise input → empty (or whitespace-only) result.
        assert _normalize_transcript("[Music] [Applause]").strip() == ""


class TestParseVtt:
    """VTT parsing — strips timing lines, NOTE/STYLE blocks, dedupes auto-captions."""

    def test_extracts_text_from_simple_vtt(self) -> None:
        vtt = (
            "WEBVTT\n"
            "Kind: captions\n"
            "Language: en\n"
            "\n"
            "00:00:00.000 --> 00:00:02.000\n"
            "Hello there\n"
            "\n"
            "00:00:02.000 --> 00:00:04.000\n"
            "General Kenobi\n"
        )
        result = _parse_vtt(vtt)
        assert "Hello there" in result
        assert "General Kenobi" in result
        assert "-->" not in result
        assert "WEBVTT" not in result
        assert "Kind:" not in result

    def test_strips_html_style_tags(self) -> None:
        vtt = (
            "WEBVTT\n"
            "\n"
            "00:00:00.000 --> 00:00:02.000\n"
            "<c.colorE5E5E5>tagged text</c>\n"
        )
        assert "<" not in _parse_vtt(vtt)
        assert "tagged text" in _parse_vtt(vtt)

    def test_dedupes_repeated_lines(self) -> None:
        # YouTube auto-captions often repeat the same cue across overlapping
        # timing windows — the parser should drop consecutive duplicates.
        vtt = (
            "WEBVTT\n"
            "\n"
            "00:00:00.000 --> 00:00:01.000\n"
            "hello world\n"
            "\n"
            "00:00:01.000 --> 00:00:02.000\n"
            "hello world\n"
        )
        # Only one occurrence after dedup.
        assert _parse_vtt(vtt).count("hello world") == 1

    def test_skips_numeric_cue_identifiers(self) -> None:
        vtt = (
            "WEBVTT\n"
            "\n"
            "1\n"
            "00:00:00.000 --> 00:00:02.000\n"
            "first cue\n"
        )
        assert "first cue" in _parse_vtt(vtt)
        assert "1" not in _parse_vtt(vtt).split()


# ---------------------------------------------------------------------------
# YouTubeHandler.fetch — mocked end-to-end paths
# ---------------------------------------------------------------------------


def _make_handler() -> YouTubeHandler:
    return YouTubeHandler(FetchConfig())


def _spec(url: str = "https://www.youtube.com/watch?v=dQw4w9WgXcQ") -> SourceSpec:
    return SourceSpec(type="youtube", url=url)


class _FakeYoutubeDL:
    """yt-dlp stand-in. Subclasses customise behavior per test."""

    info: dict[str, Any] = {
        "title": "Test Video Title",
        "uploader": "Channel Name",
        "channel": "Channel Name",
        "upload_date": "20240115",
        "duration": 600,
        "channel_url": "https://www.youtube.com/channel/UC123",
        "view_count": 1_000_000,
    }

    def __init__(self, _opts: dict[str, Any] | None = None) -> None:
        pass

    def __enter__(self) -> "_FakeYoutubeDL":
        return self

    def __exit__(self, *_: Any) -> None:
        return None

    def extract_info(self, _url: str, download: bool = False) -> dict[str, Any]:  # noqa: ARG002
        return self.info

    def download(self, _urls: list[str]) -> None:  # noqa: ARG002
        # Default: refuse to download. Tests that exercise the yt-dlp subs
        # fallback override this.
        from yt_dlp.utils import DownloadError

        raise DownloadError("subtitles unavailable")


class _FakeSnippet:
    def __init__(self, text: str) -> None:
        self.text = text


class _FakeTranscript:
    """One transcript entry inside a transcript_list."""

    def __init__(
        self,
        text: str,
        *,
        is_generated: bool = False,
        language_code: str = "en",
    ) -> None:
        self._text = text
        self.is_generated = is_generated
        self.language_code = language_code

    def fetch(self) -> Any:
        return [_FakeSnippet(self._text)]


class _FakeTranscriptList:
    """Stand-in for the iterable returned by api.list(video_id)."""

    def __init__(self, transcripts: list[_FakeTranscript]) -> None:
        self._items = transcripts

    def find_manually_created_transcript(self, _langs: list[str]) -> _FakeTranscript:
        from youtube_transcript_api import NoTranscriptFound

        for t in self._items:
            if not t.is_generated:
                return t
        raise NoTranscriptFound("video_id", _langs, self._items)

    def find_generated_transcript(self, _langs: list[str]) -> _FakeTranscript:
        from youtube_transcript_api import NoTranscriptFound

        for t in self._items:
            if t.is_generated:
                return t
        raise NoTranscriptFound("video_id", _langs, self._items)

    def __iter__(self) -> Any:
        return iter(self._items)


def _patch_yt_dlp(
    monkeypatch: pytest.MonkeyPatch,
    youtube_dl_cls: type = _FakeYoutubeDL,
) -> None:
    import yt_dlp

    monkeypatch.setattr(yt_dlp, "YoutubeDL", youtube_dl_cls)


def _patch_transcript_api(
    monkeypatch: pytest.MonkeyPatch,
    *,
    list_returns: _FakeTranscriptList | None = None,
    list_raises: type[Exception] | None = None,
    list_raise_args: tuple[Any, ...] = (),
) -> None:
    """Replace `youtube_transcript_api.YouTubeTranscriptApi` with a fake."""
    import youtube_transcript_api

    class FakeApi:
        def __init__(self, **_kwargs: Any) -> None:
            pass

        def list(self, _video_id: str) -> _FakeTranscriptList:
            if list_raises is not None:
                raise list_raises(*list_raise_args)
            assert list_returns is not None
            return list_returns

    monkeypatch.setattr(youtube_transcript_api, "YouTubeTranscriptApi", FakeApi)


class TestFetchHappyPath:
    def test_returns_metadata_and_transcript(self, monkeypatch: pytest.MonkeyPatch) -> None:
        _patch_yt_dlp(monkeypatch)
        _patch_transcript_api(
            monkeypatch,
            list_returns=_FakeTranscriptList(
                [_FakeTranscript("This is the transcript content. " * 5)]
            ),
        )

        content = _make_handler().fetch(_spec())

        assert content.title == "Test Video Title"
        assert content.author == "Channel Name"
        assert content.duration_seconds == 600
        assert content.published_at == "2024-01-15"
        assert "transcript content" in content.body
        assert content.metadata["video_id"] == "dQw4w9WgXcQ"
        assert content.metadata["transcript_source"] == "youtube-transcript-api"
        assert content.metadata["transcript_type"] == "manual"

    def test_picks_generated_when_no_manual(self, monkeypatch: pytest.MonkeyPatch) -> None:
        _patch_yt_dlp(monkeypatch)
        _patch_transcript_api(
            monkeypatch,
            list_returns=_FakeTranscriptList(
                [_FakeTranscript("auto-caption text", is_generated=True)]
            ),
        )

        content = _make_handler().fetch(_spec())
        assert content.metadata["transcript_type"] == "auto"
        assert "auto-caption text" in content.body


class TestFetchErrors:
    def test_invalid_url_raises_hard_failure(self) -> None:
        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec("https://example.com/not-youtube"))
        assert "video ID" in str(exc.value)

    def test_rate_limit_surfaces_useful_message(self, monkeypatch: pytest.MonkeyPatch) -> None:
        _patch_yt_dlp(monkeypatch)
        from youtube_transcript_api._errors import IpBlocked

        _patch_transcript_api(
            monkeypatch,
            list_raises=IpBlocked,
            list_raise_args=("blocked by youtube",),
        )

        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec())
        assert "rate-limit" in str(exc.value).lower()

    def test_age_restricted_falls_back_then_hard_fails(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        # AgeRestricted on the primary path → handler tries the yt-dlp subs
        # fallback. When both fail, the user sees HandlerHardFailure. The
        # exact message depends on which side failed last — we just check
        # the failure surfaces cleanly and references yt-dlp (the path that
        # ran most recently). The contract: never silently return empty.
        _patch_yt_dlp(monkeypatch)
        from youtube_transcript_api._errors import AgeRestricted

        _patch_transcript_api(
            monkeypatch,
            list_raises=AgeRestricted,
            list_raise_args=("video_id",),
        )

        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec())
        assert "yt-dlp" in str(exc.value).lower()

    def test_yt_dlp_age_gate_surfaces_clear_message(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        # When yt-dlp itself emits the "Sign in to confirm your age" error
        # during metadata fetch, we should translate to a curator-readable msg.
        from yt_dlp.utils import DownloadError

        class AgeGatedYoutubeDL(_FakeYoutubeDL):
            def extract_info(self, _url: str, download: bool = False) -> dict[str, Any]:  # noqa: ARG002
                raise DownloadError("ERROR: Sign in to confirm your age")

        _patch_yt_dlp(monkeypatch, youtube_dl_cls=AgeGatedYoutubeDL)

        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec())
        assert "age" in str(exc.value).lower()

    def test_private_video_surfaces_clear_message(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        from yt_dlp.utils import DownloadError

        class PrivateYoutubeDL(_FakeYoutubeDL):
            def extract_info(self, _url: str, download: bool = False) -> dict[str, Any]:  # noqa: ARG002
                raise DownloadError("ERROR: Private video. Sign in if you've been granted access.")

        _patch_yt_dlp(monkeypatch, youtube_dl_cls=PrivateYoutubeDL)

        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec())
        assert "private" in str(exc.value).lower()


class TestFetchFallback:
    def test_falls_back_to_yt_dlp_subs_when_api_disabled(
        self, monkeypatch: pytest.MonkeyPatch, tmp_path: Any
    ) -> None:
        # Primary path: TranscriptsDisabled — the legal "no captions" reason.
        # Fallback: yt-dlp downloads a subtitle VTT we stage on disk.
        from youtube_transcript_api import TranscriptsDisabled

        _patch_transcript_api(
            monkeypatch,
            list_raises=TranscriptsDisabled,
            list_raise_args=("video_id",),
        )

        # yt-dlp's download writes a .vtt file matching the video_id pattern
        # into the tempdir its caller supplied. We can't see that tempdir from
        # here, but we can intercept the write via a fake YoutubeDL that
        # creates the file inside the outtmpl directory.
        from pathlib import Path

        class SubsYoutubeDL(_FakeYoutubeDL):
            def __init__(self, opts: dict[str, Any] | None = None) -> None:
                super().__init__(opts)
                self._opts = opts or {}

            def download(self, _urls: list[str]) -> None:  # noqa: ARG002
                outtmpl = self._opts.get("outtmpl", "")
                tmpdir = Path(outtmpl).parent if outtmpl else tmp_path
                vtt = tmpdir / "dQw4w9WgXcQ.en.vtt"
                vtt.write_text(
                    "WEBVTT\n\n"
                    "00:00:00.000 --> 00:00:02.000\n"
                    "fallback transcript line\n",
                    encoding="utf-8",
                )

        _patch_yt_dlp(monkeypatch, youtube_dl_cls=SubsYoutubeDL)

        content = _make_handler().fetch(_spec())
        assert "fallback transcript line" in content.body
        assert content.metadata["transcript_source"] == "yt-dlp"

    def test_both_paths_fail_raises_hard_failure(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        # TranscriptsDisabled → primary marks _TranscriptUnavailable →
        # fallback also fails → user gets HandlerHardFailure. _raise_yt_dlp_error
        # currently converts DownloadError to HardFailure directly (so the
        # primary path's reason is lost); we just verify the failure is loud
        # and identifies the URL.
        from youtube_transcript_api import TranscriptsDisabled
        from yt_dlp.utils import DownloadError

        _patch_transcript_api(
            monkeypatch,
            list_raises=TranscriptsDisabled,
            list_raise_args=("video_id",),
        )

        class FailingSubsYoutubeDL(_FakeYoutubeDL):
            def download(self, _urls: list[str]) -> None:  # noqa: ARG002
                raise DownloadError("ERROR: no subtitle tracks available")

        _patch_yt_dlp(monkeypatch, youtube_dl_cls=FailingSubsYoutubeDL)

        with pytest.raises(HandlerHardFailure) as exc:
            _make_handler().fetch(_spec())
        msg = str(exc.value)
        assert "dQw4w9WgXcQ" in msg or "https://www.youtube.com/watch?v=dQw4w9WgXcQ" in msg
