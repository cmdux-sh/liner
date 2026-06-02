"""Recover agent WebFetch results from `.liner-runs/` JSONL transcripts.

Used as a last-resort fallback at compile time. When the Python fetcher
(httpx → web_js Playwright) can't get a URL — for example, the site sits
behind an aggressive bot wall — we look up the most recent WebFetch the
Claude agent did against the same URL during research and use its summary
as the source body. Marked clearly so the curator knows the mixtape entry
is a summary, not a fresh extraction.

What we recover is a *Claude-generated summary*, not the raw article. The
agent's WebFetch tool returns a structured digest (title, author, key
points, ~500 chars). That's a graceful degradation, not a substitute for
trafilatura output — but it's far better than emitting "Failed to fetch."

Pure derivation over `.liner-runs/`. No new state to persist or invalidate;
the JSONL files are already the source of truth.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

RUN_DIR = ".liner-runs"

# Tool names that count as a fetch we can recover. Claude's tool is named
# "WebFetch"; the others cover Codex / future renames.
FETCH_TOOL_NAMES = frozenset({"WebFetch", "web_fetch", "fetch", "Fetch"})

# Lower bound — anything shorter is almost certainly an error envelope rather
# than usable summary text. Picked empirically: real WebFetch summaries we've
# observed in the wild bottom out around 200 chars.
MIN_USABLE_SUMMARY_CHARS = 80


@dataclass(frozen=True, slots=True)
class AgentFetch:
    """One captured WebFetch response from the agent's transcript."""

    url: str
    body: str
    captured_at: str
    run_path: str


def build_agent_fetch_cache(folder: Path) -> dict[str, AgentFetch]:
    """Walk `<folder>/.liner-runs/` and return URL → latest AgentFetch.

    "Latest" is the freshest WebFetch result for that URL across all runs,
    measured by the run's start timestamp. Earlier fetches are shadowed
    silently — we want the most current digest, not a merge.
    """
    base = folder / RUN_DIR
    if not base.is_dir():
        return {}

    out: dict[str, AgentFetch] = {}
    for path in sorted(base.rglob("*.jsonl")):
        try:
            fetches = _extract_fetches_from_log(path)
        except OSError:
            continue
        for fetch in fetches:
            existing = out.get(fetch.url)
            if existing is None or fetch.captured_at > existing.captured_at:
                out[fetch.url] = fetch
    return out


def _extract_fetches_from_log(path: Path) -> list[AgentFetch]:
    """Pull every (URL, summary) pair out of one JSONL run log.

    The Claude agent stream emits a `tool_use` event with the URL, then a
    matching `tool_result` event (in a `user` message) carrying the body
    keyed by `tool_use_id`. We pair them in one pass.
    """
    started_at = ""
    # Map id → URL while we walk; resolve when the matching tool_result lands.
    pending_url: dict[str, str] = {}
    fetches: list[AgentFetch] = []

    with path.open("r", encoding="utf-8") as fp:
        for line in fp:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            rec_type = rec.get("type")
            if rec_type == "_liner_meta":
                started_at = str(rec.get("startedAt") or "")
            elif rec_type == "assistant":
                msg = rec.get("message") or {}
                for c in (msg.get("content") or []):
                    if not isinstance(c, dict):
                        continue
                    if c.get("type") != "tool_use":
                        continue
                    if c.get("name") not in FETCH_TOOL_NAMES:
                        continue
                    url = (c.get("input") or {}).get("url")
                    call_id = c.get("id")
                    if isinstance(url, str) and isinstance(call_id, str):
                        pending_url[call_id] = url
            elif rec_type == "user":
                msg = rec.get("message") or {}
                for c in (msg.get("content") or []):
                    if not isinstance(c, dict):
                        continue
                    if c.get("type") != "tool_result":
                        continue
                    call_id = c.get("tool_use_id")
                    if not isinstance(call_id, str):
                        continue
                    url = pending_url.pop(call_id, None)
                    if url is None:
                        continue
                    body = _coerce_tool_result_text(c.get("content"))
                    if body and len(body) >= MIN_USABLE_SUMMARY_CHARS:
                        fetches.append(
                            AgentFetch(
                                url=url,
                                body=body,
                                captured_at=started_at,
                                run_path=str(path),
                            )
                        )
    return fetches


def _coerce_tool_result_text(content: Any) -> str:
    """Tool-result `content` may be a raw string or a list of content items.
    Concatenate every text part into a single string."""
    if content is None:
        return ""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts: list[str] = []
        for ci in content:
            if isinstance(ci, dict) and ci.get("type") == "text":
                t = ci.get("text")
                if isinstance(t, str):
                    parts.append(t)
        return "\n".join(parts)
    return ""


def title_from_summary(summary: str, fallback: str) -> str:
    """Try to lift a title out of the agent-summary text.

    The Claude WebFetch summary usually starts with `# Article Summary`
    followed by `**Title:** ...`. Grab that if present; otherwise the
    first non-blank line, capped; otherwise the URL.
    """
    for line in summary.splitlines():
        line = line.strip()
        if not line:
            continue
        if line.lower().startswith("**title:**"):
            return line[len("**title:**") :].strip().strip("*") or fallback
        if line.startswith("# "):
            heading = line[2:].strip()
            if heading.lower() != "article summary":
                return heading
    # Fallback to the first meaningful line under 120 chars.
    for line in summary.splitlines():
        line = line.strip(" #*-")
        if 5 <= len(line) <= 120:
            return line
    return fallback
