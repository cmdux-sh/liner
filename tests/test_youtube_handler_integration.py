"""Live-network smoke test for the YouTube handler.

Skipped by default — pytest filters out `network`-marked tests via the
default `addopts`. Run on demand with:

    pytest -m network tests/test_youtube_handler_integration.py

When this test fails locally it usually means one of three things:

  1. YouTube changed something on their end (API surface, captions
     availability, rate-limit posture). Real, but not the test's fault.
  2. `youtube-transcript-api` or `yt-dlp` shipped a breaking change.
     Pin / upgrade as needed.
  3. The handler regressed. The mocked unit tests in test_youtube_handler.py
     should also flag this; if they pass but this one fails, the bug is in
     the integration layer between the two libraries and our handler — not
     in our extraction logic.

Target video: "Me at the zoo" (jNQXAC9IVRw) — the first YouTube video ever,
uploaded April 2005. As stable a real-world target as YouTube offers, and
its captions are auto-generated, exercising the generated-transcript path.
"""

from __future__ import annotations

import pytest

from liner.config import FetchConfig
from liner.handlers.youtube import YouTubeHandler
from liner.types import SourceSpec


# A canonical, very stable target. If this video ever disappears, swap to
# another short, long-lived video with English captions.
STABLE_VIDEO_URL = "https://www.youtube.com/watch?v=jNQXAC9IVRw"


@pytest.mark.network
def test_real_youtube_fetch_returns_metadata_and_transcript() -> None:
    handler = YouTubeHandler(FetchConfig())
    spec = SourceSpec(type="youtube", url=STABLE_VIDEO_URL)

    content = handler.fetch(spec)

    # Metadata: title + uploader + a known-stable upload date.
    assert content.title, "expected a non-empty title"
    assert content.url == STABLE_VIDEO_URL
    assert content.author, "expected an uploader/channel name"
    assert content.published_at and content.published_at.startswith("2005-"), (
        "Me at the zoo was uploaded in 2005 — date should reflect that"
    )

    # Transcript: at least a few words come back. Don't pin specific phrases
    # because YouTube's auto-captions can change over time.
    body = content.body or ""
    assert len(body) > 50, f"expected a meaningful transcript, got {len(body)} chars"

    # Metadata block should record which source supplied the transcript.
    source = content.metadata.get("transcript_source")
    assert source in {"youtube-transcript-api", "yt-dlp"}, (
        f"unexpected transcript_source: {source!r}"
    )

    # The video_id we extracted should be preserved in metadata.
    assert content.metadata.get("video_id") == "jNQXAC9IVRw"
