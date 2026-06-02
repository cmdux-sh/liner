"""Mixtape process manifest.

Walks a project's `.liner-runs/` directory and rolls every agent JSONL
transcript into a single `process.json` at the mixtape root. The file is
purely derived from the run logs — safe to delete and regenerate.

What it captures (per run + aggregated):
  - tokens (input / output / cache read / cache create)
  - cost in USD (when the agent reports it; Codex may not)
  - tool calls grouped by tool name
  - fetched URLs (WebFetch tool inputs)
  - derived domain frequency table
  - model + agent + duration + exit code

Not captured: low-level message bodies. Those live in `.liner-runs/` already
and would balloon the manifest. The manifest is the *index*; the raw logs
are the archive.
"""

from __future__ import annotations

import json
from collections import Counter
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from liner.tape import load_tape, TapeValidationError

MANIFEST_FILENAME = "process.json"
RUN_DIR = ".liner-runs"

# Tool names that represent a URL fetch. WebFetch is Claude's tool; the
# other variants cover Codex naming and possible future renames.
FETCH_TOOLS = frozenset({"WebFetch", "web_fetch", "fetch", "Fetch"})


@dataclass
class TokenUsage:
    input: int = 0
    output: int = 0
    cache_read: int = 0
    cache_create: int = 0

    def add(self, other: "TokenUsage") -> "TokenUsage":
        return TokenUsage(
            input=self.input + other.input,
            output=self.output + other.output,
            cache_read=self.cache_read + other.cache_read,
            cache_create=self.cache_create + other.cache_create,
        )


@dataclass
class RunSummary:
    """One row per agent execution."""

    task_label: str
    agent: str
    model: str | None
    started_at: str
    ended_at: str | None
    duration_s: float | None
    exit_code: int | None
    num_turns: int
    tokens: TokenUsage
    cost_usd: float | None
    tools: dict[str, int]
    fetches: list[str]
    log_path: str

    def to_dict(self) -> dict[str, Any]:
        d = asdict(self)
        return d


@dataclass
class Manifest:
    mixtape: dict[str, Any]
    totals: dict[str, Any]
    agents_used: list[str]
    models_used: list[str]
    domains: list[dict[str, Any]]
    fetches: list[dict[str, Any]]
    runs: list[dict[str, Any]] = field(default_factory=list)
    generated_at: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "generated_at": self.generated_at,
            "mixtape": self.mixtape,
            "totals": self.totals,
            "agents_used": self.agents_used,
            "models_used": self.models_used,
            "domains": self.domains,
            "fetches": self.fetches,
            "runs": self.runs,
        }


# --- Parsing ---------------------------------------------------------------


def _parse_token_usage(usage: dict[str, Any] | None) -> TokenUsage:
    if not usage:
        return TokenUsage()
    return TokenUsage(
        input=int(usage.get("input_tokens") or 0),
        output=int(usage.get("output_tokens") or 0),
        cache_read=int(usage.get("cache_read_input_tokens") or 0),
        cache_create=int(usage.get("cache_creation_input_tokens") or 0),
    )


def _iso_to_seconds(start_iso: str, end_iso: str | None) -> float | None:
    if not end_iso:
        return None
    try:
        start = datetime.fromisoformat(start_iso.replace("Z", "+00:00"))
        end = datetime.fromisoformat(end_iso.replace("Z", "+00:00"))
        return round((end - start).total_seconds(), 3)
    except ValueError:
        return None


def parse_run_log(path: Path) -> RunSummary | None:
    """Roll a single .jsonl run log into a RunSummary.

    Returns None if the file is empty or the meta header is missing. Malformed
    individual lines are skipped — the agent JSON stream is occasionally
    truncated mid-write and we'd rather report a partial run than crash.
    """
    meta: dict[str, Any] | None = None
    close: dict[str, Any] | None = None
    result: dict[str, Any] | None = None
    tools: Counter[str] = Counter()
    fetches: list[str] = []
    model: str | None = None

    try:
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
                    meta = rec
                elif rec_type == "_liner_close":
                    close = rec
                elif rec_type == "result":
                    result = rec
                elif rec_type == "assistant":
                    msg = rec.get("message") or {}
                    if isinstance(msg, dict):
                        if not model and isinstance(msg.get("model"), str):
                            model = msg["model"]
                        for c in msg.get("content") or []:
                            if not isinstance(c, dict):
                                continue
                            if c.get("type") == "tool_use":
                                name = c.get("name") or "?"
                                tools[name] += 1
                                if name in FETCH_TOOLS:
                                    url = (c.get("input") or {}).get("url")
                                    if isinstance(url, str):
                                        fetches.append(url)
    except OSError:
        return None

    if not meta:
        return None

    # taskLabel was previously called phaseId — accept both for backward compat.
    task_label = meta.get("taskLabel") or meta.get("phaseId") or "unknown"
    agent = meta.get("agent") or "unknown"
    started_at = meta.get("startedAt") or ""
    ended_at = (close or {}).get("endedAt")
    exit_code = (close or {}).get("exitCode")

    cost_usd = None
    num_turns = 0
    tokens = TokenUsage()
    if result:
        cost_usd = result.get("total_cost_usd")
        num_turns = int(result.get("num_turns") or 0)
        tokens = _parse_token_usage(result.get("usage"))
        # Result also carries modelUsage map — first key is the canonical model id.
        mu = result.get("modelUsage") or {}
        if mu and not model:
            model = next(iter(mu.keys()), None)

    duration_s = _iso_to_seconds(started_at, ended_at)

    return RunSummary(
        task_label=task_label,
        agent=agent,
        model=model,
        started_at=started_at,
        ended_at=ended_at,
        duration_s=duration_s,
        exit_code=exit_code if isinstance(exit_code, int) else None,
        num_turns=num_turns,
        tokens=tokens,
        cost_usd=cost_usd if isinstance(cost_usd, (int, float)) else None,
        tools=dict(tools),
        fetches=fetches,
        log_path=str(path),
    )


def _discover_run_logs(folder: Path) -> list[Path]:
    base = folder / RUN_DIR
    if not base.is_dir():
        return []
    return sorted(base.rglob("*.jsonl"))


def _domain_of(url: str) -> str:
    try:
        host = urlparse(url).hostname or ""
        # Drop a leading "www." for friendlier dedup.
        if host.startswith("www."):
            host = host[4:]
        return host
    except ValueError:
        return ""


# --- Building --------------------------------------------------------------


def build_manifest(folder: Path) -> Manifest:
    """Aggregate every run log in `folder` into a Manifest.

    Tolerant: missing tape.yaml, missing .liner-runs/, malformed lines all
    degrade gracefully. The output is always a complete (possibly empty)
    Manifest.
    """
    mixtape: dict[str, Any] = {"path": str(folder), "name": folder.name}
    try:
        tape = load_tape(folder / "tape.yaml")
        mixtape.update(
            {
                "title": tape.title,
                "description": tape.description,
                "curator": tape.curator,
                "jtbd": tape.jtbd,
                "mode": tape.mode,
                "source_count": len(tape.sources),
            }
        )
    except (TapeValidationError, FileNotFoundError, OSError):
        pass

    runs: list[RunSummary] = []
    for path in _discover_run_logs(folder):
        summary = parse_run_log(path)
        if summary:
            runs.append(summary)

    # Stable sort: oldest first by startedAt, then by file path as tiebreaker.
    runs.sort(key=lambda r: (r.started_at, r.log_path))

    tokens_total = TokenUsage()
    cost_total = 0.0
    cost_known = False
    tool_total = 0
    fetch_total = 0
    fetches_flat: list[dict[str, Any]] = []
    domain_counter: Counter[str] = Counter()
    agents: set[str] = set()
    models: set[str] = set()

    for r in runs:
        tokens_total = tokens_total.add(r.tokens)
        if r.cost_usd is not None:
            cost_total += r.cost_usd
            cost_known = True
        tool_total += sum(r.tools.values())
        fetch_total += len(r.fetches)
        agents.add(r.agent)
        if r.model:
            models.add(r.model)
        for url in r.fetches:
            domain_counter[_domain_of(url) or "(unknown)"] += 1
            fetches_flat.append(
                {
                    "url": url,
                    "task_label": r.task_label,
                    "at": r.started_at,
                }
            )

    totals: dict[str, Any] = {
        "runs": len(runs),
        "tokens": asdict(tokens_total),
        "tool_calls": tool_total,
        "fetches": fetch_total,
        "cost_usd": round(cost_total, 6) if cost_known else None,
    }

    domains = [
        {"domain": d, "count": c}
        for d, c in sorted(domain_counter.items(), key=lambda kv: (-kv[1], kv[0]))
    ]

    return Manifest(
        mixtape=mixtape,
        totals=totals,
        agents_used=sorted(agents),
        models_used=sorted(models),
        domains=domains,
        fetches=fetches_flat,
        runs=[r.to_dict() for r in runs],
        generated_at=datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    )


def write_manifest(folder: Path, manifest: Manifest) -> Path:
    target = folder / MANIFEST_FILENAME
    target.write_text(
        json.dumps(manifest.to_dict(), indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    return target


def read_manifest(folder: Path) -> dict[str, Any] | None:
    target = folder / MANIFEST_FILENAME
    if not target.is_file():
        return None
    try:
        return json.loads(target.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
