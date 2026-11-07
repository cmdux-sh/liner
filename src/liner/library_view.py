"""Everything the terminal navigator shows, in one read-only report."""

from __future__ import annotations

import hashlib
import json
from itertools import islice
from pathlib import Path
from typing import Any

from liner.citations import citation_files, compiled_source_files, parse_citations
from liner.empirical_tests import list_tests
from liner.findings import list_findings
from liner.gaps import list_gaps
from liner.library import build_catalog
from liner.project import ProjectFolder
from liner.refresh import read_refresh_report
from liner.research import ResearchError, audit_path, current_step
from liner.tape import load_tape
from liner.types import SourceSpec
from liner.working_log import WorkingLogError

LIBRARY_CONTRACT = "liner.library_view"
CONTRACT_VERSION = 1

_BUILD_PHASES = ("framing", "candidates", "evaluation", "quality", "synthesis", "assembly")
_APPROVALS = {
    "approve:synthesis": "Synthesis needs your approval",
    "approve:sources": "Sources need your approval",
    "approve:operating_layer": "Instructions need your approval",
}
_HEADER_LINES = 40
_WATCHED_FILES = (
    "tape.yaml",
    "synthesis.md",
    "MIXTAPE.md",
    ".liner-progress.json",
    ".liner-research.json",
    "working/gaps.yaml",
    "working/findings.yaml",
    "working/empirical-tests.yaml",
)


def build_library_view(root: Path) -> dict[str, Any]:
    catalog = build_catalog(root)
    liners = [_liner_view(entry) for entry in catalog["liners"]]
    return {
        "contract": LIBRARY_CONTRACT,
        "version": CONTRACT_VERSION,
        "root": catalog["root"],
        "root_exists": catalog["root_exists"],
        "liners": liners,
        "unreadable": catalog["unreadable"],
        "attention": [item for liner in liners for item in _attention(liner)],
        "sources": _source_index(liners),
        "connections": _connections(liners),
    }


def _liner_view(entry: dict[str, Any]) -> dict[str, Any]:
    folder = ProjectFolder(Path(entry["path"]))
    step = _research_step(folder)
    sources = _sources(folder)
    return {
        **entry,
        "state": _state(entry, step, sources),
        "waiting_on": _APPROVALS.get(step or ""),
        "build_step": _BUILD_PHASES.index(step) + 1 if step in _BUILD_PHASES else None,
        "build_steps": len(_BUILD_PHASES),
        "sources": sources,
        "sections": _sections(sources),
        "changed_at": _changed_at(folder),
        "refreshed_at": _refreshed_at(folder),
        "synthesis": _synthesis(folder),
        "gaps": _safely(lambda: list_gaps(folder)["gaps"], []),
        "findings": _safely(lambda: list_findings(folder)["findings"], []),
        "tests": _safely(lambda: list_tests(folder)["tests"], []),
        "files": _files(folder),
    }


def _research_step(folder: ProjectFolder) -> str | None:
    try:
        return current_step(folder)
    except (ResearchError, OSError):
        return None


def _state(entry: dict[str, Any], step: str | None, sources: list[dict[str, Any]]) -> str:
    if step in _APPROVALS:
        return "waiting"
    if step is not None:
        return "building"
    if any(source["changed"] for source in sources):
        return "sources moved"
    return "ready" if entry["usable"] else "draft"


def _sources(folder: ProjectFolder) -> list[dict[str, Any]]:
    try:
        specs = load_tape(folder.tape_path).sources
    except (OSError, ValueError):
        return []
    files = compiled_source_files(folder)
    changed = _changed_locators(folder, files)
    views = []
    for spec in specs:
        locator = spec.url or spec.path or ""
        compiled = files.get(locator)
        views.append(
            {
                "locator": locator,
                "title": _title(spec, compiled),
                "type": _display_type(spec),
                "kind": spec.kind,
                "section": spec.section or "Other",
                "note": spec.note,
                "file": str(compiled) if compiled else None,
                "fetched_at": _fetched_at(compiled),
                "changed": locator in changed,
            }
        )
    return views


def _changed_locators(folder: ProjectFolder, files: dict[str, Path]) -> set[str]:
    """A saved change counts until a compile rewrites the page it was measured against."""
    report = read_refresh_report(folder)
    return {
        source["url"]
        for source in (report or {}).get("sources", [])
        if source.get("status") == "changed"
        and source["url"] in files
        and _file_hash(files[source["url"]]) == source.get("baseline")
    }


def _file_hash(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _display_type(spec: SourceSpec) -> str:
    if spec.type == "local_file":
        return "book" if (spec.path or "").lower().endswith(".epub") else "local"
    return spec.type


def _title(spec: SourceSpec, compiled: Path | None) -> str:
    if compiled is not None:
        with compiled.open(encoding="utf-8", errors="replace") as handle:
            first = handle.readline().strip()
        if first.startswith("# "):
            return first[2:]
    return spec.citation or spec.url or spec.path or "(untitled)"


def _fetched_at(compiled: Path | None) -> str | None:
    if compiled is None:
        return None
    with compiled.open(encoding="utf-8", errors="replace") as handle:
        for line in islice(handle, _HEADER_LINES):
            if line.startswith("**Fetched:** "):
                return line.removeprefix("**Fetched:** ").strip()
    return None


def _sections(sources: list[dict[str, Any]]) -> list[dict[str, Any]]:
    counts: dict[str, int] = {}
    for source in sources:
        counts[source["section"]] = counts.get(source["section"], 0) + 1
    return [{"name": name, "sources": count} for name, count in counts.items()]


def _changed_at(folder: ProjectFolder) -> float | None:
    times = []
    for name in _WATCHED_FILES:
        path = folder.corpus_path / name
        if path.is_file():
            times.append(path.stat().st_mtime)
    return max(times) if times else None


def _refreshed_at(folder: ProjectFolder) -> str | None:
    report = read_refresh_report(folder)
    return report.get("checked_at") if report else None


def _synthesis(folder: ProjectFolder) -> dict[str, Any] | None:
    if not folder.synthesis_path.is_file():
        return None
    text = folder.synthesis_path.read_text(encoding="utf-8", errors="replace")
    lines = text.splitlines()
    parsed = parse_citations(text)
    return {
        "path": str(folder.synthesis_path),
        "lines": len(lines),
        "sections": sum(line.startswith("## ") for line in lines),
        "citations": len(parsed.targets),
        "files": citation_files(folder, parsed),
    }


def _files(folder: ProjectFolder) -> list[str]:
    candidates = (
        folder.liner_path,
        folder.path / "SKILL.md",
        folder.mixtape_path,
        folder.synthesis_path,
    )
    return [str(path) for path in candidates if path.is_file()]


def _safely(read: Any, fallback: Any) -> Any:
    try:
        return read()
    except (WorkingLogError, OSError):
        return fallback


def _attention(liner: dict[str, Any]) -> list[dict[str, Any]]:
    slug = liner["slug"]
    items: list[dict[str, Any]] = []

    def add(group: str, title: str, context: str = "liner") -> None:
        items.append({"group": group, "title": title, "slug": slug, "context": context})

    if liner["waiting_on"]:
        add("waiting", liner["waiting_on"], "approval")
    if liner["findings"]:
        add(
            "waiting", _count(len(liner["findings"]), "finding") + " to accept or reject", "finding"
        )
    changed = sum(source["changed"] for source in liner["sources"])
    if changed:
        add("changed", _count(changed, "source") + " changed online", "changed")
    if audit_path(ProjectFolder(Path(liner["path"]))) is not None:
        add("stronger", "The quality check recommends improving it", "gap")
    if liner["gaps"]:
        add("stronger", _count(len(liner["gaps"]), "question") + " it couldn't answer", "gap")
    summary = liner["test"] or {}
    if summary.get("runs") and summary["same"] + summary["worse"]:
        add(
            "stronger",
            f"Not better in {summary['same'] + summary['worse']} of {summary['runs']} tests",
        )
    if liner["usable"] and summary.get("runs") == 0:
        add("stronger", "Never tested")
    if liner["state"] == "building":
        step = liner["build_step"]
        add("working", f"Building, step {step} of {liner['build_steps']}" if step else "Working")
    return items


def _count(number: int, noun: str) -> str:
    return f"{number} {noun}" if number == 1 else f"{number} {noun}s"


def _source_index(liners: list[dict[str, Any]]) -> list[dict[str, Any]]:
    merged: dict[str, dict[str, Any]] = {}
    for liner in liners:
        for source in liner["sources"]:
            entry = merged.setdefault(_source_key(source, liner["slug"]), {**source, "liners": []})
            entry["liners"].append(liner["slug"])
            entry["changed"] = entry["changed"] or source["changed"]
    return sorted(merged.values(), key=lambda item: (-len(item["liners"]), item["title"].lower()))


def _source_key(source: dict[str, Any], slug: str) -> str:
    locator = source["locator"]
    return locator if "://" in locator else f"{slug}/{locator}"


def _connections(liners: list[dict[str, Any]]) -> dict[str, Any]:
    return {
        "citations": [
            {"from": liner["slug"], "to": cited, "stale": _citation_is_stale(liner, cited)}
            for liner in liners
            for cited in liner["cites"]
        ],
        "uncited": [
            liner["slug"] for liner in liners if not liner["cites"] and not liner["cited_by"]
        ],
    }


def _citation_is_stale(liner: dict[str, Any], cited: str) -> bool:
    return any(
        source["changed"]
        for source in liner["sources"]
        if source["locator"] == f"liner://{cited}"
        or str(source["locator"]).startswith(f"liner://{cited}/")
    )


def to_json(view: dict[str, Any]) -> str:
    return json.dumps(view, ensure_ascii=False, indent=2)
