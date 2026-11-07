"""Check what a research phase produced. Everything here is read-only."""

from __future__ import annotations

import math
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from liner.citations import parse_citations

CHECK_CONTRACT = "liner.research_check"
CONTRACT_VERSION = 1
CHECKED_PHASES = ("candidates", "evaluation", "quality", "synthesis", "assembly")

_DECISIONS = {"kept", "trim", "dropped"}
_SOURCE_TYPES = {"web", "youtube", "local_file", "skill", "repository", "liner"}
_PRIORITIES = {"required", "optional"}
_KINDS = {"reference", "principle", "prescription", "example"}
_JTBD_FITS = {"direct", "bridge", "background"}
_FETCH_STATUSES = {"readable", "partial", "metadata_only", "unavailable"}
_USABLE_FETCH_STATUSES = {"readable", "partial"}
_CONTENT_QUALITIES = {"high", "medium", "low"}
_USABLE_CONTENT_QUALITIES = {"high", "medium"}

_TODO = re.compile(r"(^|\n)\s*(?:[-*]\s*)?TODO\s*(?:[—:-]|\Z)", re.IGNORECASE)
# A reused source from another liner is a candidate too, under its liner:// address.
_URL = re.compile(r"(?:https?|liner)://[^\s\"'<>]+")
_MARKDOWN_LINK = re.compile(r"\[([^\]\n]+)\]\((https?://[^)\s]+)\)")
_SECTION_HEADING = re.compile(r"##\s+(.+?)\s*\Z")
_ANY_HEADING = re.compile(r"#{1,6}\s+(.+?)\s*\Z")
_CANDIDATE_HEADING = re.compile(r"###\s+(.+?)\s*\Z")
_CANDIDATE_REASON = re.compile(
    r"\s*[-*]\s*(?:Candidate\s+reason|Reason|Rationale):\s*(.+?)\s*\Z", re.IGNORECASE
)

_QUALITY_MARKERS = (
    "## Test 0",
    "## Test 1",
    "## Test 2",
    "## Test 3",
    "## Test 4",
    "### Perspectives audit",
    "## Test 5",
    "Distribution:",
    "## Test 6",
    "## Test 7",
    "Source-role fit",
    "## Test 8",
    "Capability-pattern fit",
)
_MIN_CITED_SOURCES = 3
_SYNTHESIS_MARKERS = ("## Generative rules", "## Stances this corpus takes")

_LONGLIST = "working/02-candidate-longlist.md"
_EVALUATION = "working/03-evaluation.yaml"
_QUALITY = "working/04-quality-checks.md"
_DRAFT = "working/07-tape-draft.yaml"


_DECIMAL = re.compile(r"[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?\Z")
_PREFIXED_INTEGER = re.compile(r"0[xXoObB][0-9a-fA-F]+\Z")


class _ArtifactLoader(yaml.SafeLoader):
    """Read YAML 1.2 scalars and refuse repeated keys, so `on` and dates stay text."""

    def construct_mapping(self, node: yaml.MappingNode, deep: bool = False) -> dict[Any, Any]:
        seen = set()
        for key_node, _ in node.value:
            key = self.construct_object(key_node, deep=True)
            if key in seen:
                raise yaml.constructor.ConstructorError(
                    None, None, f"repeated key: {key}", key_node.start_mark
                )
            seen.add(key)
        return super().construct_mapping(node, deep=deep)


_ArtifactLoader.yaml_implicit_resolvers = {
    first: [
        (tag, pattern)
        for tag, pattern in resolvers
        if tag.rsplit(":", 1)[-1] not in {"bool", "int", "float", "timestamp"}
    ]
    for first, resolvers in yaml.SafeLoader.yaml_implicit_resolvers.items()
}
_ArtifactLoader.add_implicit_resolver(
    "tag:yaml.org,2002:bool", re.compile(r"^(?:true|True|TRUE|false|False|FALSE)$"), list("tTfF")
)
_ArtifactLoader.add_implicit_resolver(
    "tag:yaml.org,2002:int",
    re.compile(r"^(?:[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+)$"),
    list("-+0123456789"),
)
_ArtifactLoader.add_implicit_resolver(
    "tag:yaml.org,2002:float",
    re.compile(
        r"^(?:[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?"
        r"|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$"
    ),
    list("-+0123456789."),
)
_ArtifactLoader.add_constructor(
    "tag:yaml.org,2002:int",
    lambda _loader, node: int(node.value, 0 if _PREFIXED_INTEGER.match(node.value) else 10),
)


class CheckFailed(Exception):
    """An artifact did not pass. The message says what to fix."""


@dataclass(slots=True)
class LonglistCandidate:
    url: str
    title: str | None
    section: str | None
    reason: str | None = None


def check_phase(corpus: Path, phase: str) -> dict[str, Any]:
    """Check one phase's artifact and report the result as a versioned payload."""
    if phase not in CHECKED_PHASES:
        raise ValueError(f"No check for phase {phase}. Use one of: {', '.join(CHECKED_PHASES)}.")
    try:
        _CHECKS[phase](corpus)
    except CheckFailed as failure:
        message: str | None = str(failure)
    else:
        message = None
    return {
        "contract": CHECK_CONTRACT,
        "version": CONTRACT_VERSION,
        "phase": phase,
        "ok": message is None,
        "message": message,
    }


def read_candidate_longlist(path: Path) -> list[LonglistCandidate]:
    if not path.exists():
        return []
    candidates: dict[str, LonglistCandidate] = {}
    section = ""
    candidate_title = ""
    current: LonglistCandidate | None = None
    for line in path.read_text(encoding="utf-8", errors="replace").split("\n"):
        heading = _SECTION_HEADING.match(line)
        if heading:
            section = re.sub(r"^\d+\.\s+", "", heading[1].strip())
            candidate_title = ""
            current = None
            continue
        if _is_search_log_heading(section):
            continue
        heading = _CANDIDATE_HEADING.match(line)
        if heading:
            candidate_title = heading[1].strip()
            current = None
            continue
        url = _first_url(line)
        if url:
            if url in candidates:
                current = candidates[url]
                continue
            title = candidate_title or _title_from_line(line, url)
            current = candidates[url] = LonglistCandidate(url, title, section or None)
            candidate_title = ""
            continue
        reason = _CANDIDATE_REASON.match(line)
        if reason and current is not None:
            current.reason = reason[1].strip()
    return list(candidates.values())


SEARCH_LOG_HEADING = "Search log"
FUNNEL_CONTRACT = "liner.research_funnel"


def read_search_log(path: Path) -> list[dict[str, Any]]:
    """Rows of the long list's search log: each query, its results, and how many were kept.

    The agent writes this table, so Liner can check that it exists but not that it's
    complete. Columns are found by their header, so extra columns are fine.
    """
    if not path.exists():
        return []
    rows: list[dict[str, Any]] = []
    inside = False
    columns: dict[str, int] | None = None
    for line in path.read_text(encoding="utf-8", errors="replace").split("\n"):
        heading = _ANY_HEADING.match(line)
        if heading:
            inside = _is_search_log_heading(heading[1])
            columns = None
            continue
        if not inside or not line.strip().startswith("|"):
            continue
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if set("".join(cells)) <= set("-: "):
            continue
        if columns is None:
            columns = _search_log_columns(cells)
            continue
        if not columns or len(cells) <= max(columns.values()):
            continue
        results = _count(cells[columns["results"]])
        shortlisted = _count(cells[columns["shortlisted"]])
        query = cells[columns["query"]]
        if query and results is not None and shortlisted is not None:
            rows.append({"query": query, "results": results, "shortlisted": shortlisted})
    return rows


def _is_search_log_heading(text: str) -> bool:
    return (
        re.sub(r"^\d+\.\s+", "", text.strip()).casefold().startswith(SEARCH_LOG_HEADING.casefold())
    )


def _search_log_columns(header: list[str]) -> dict[str, int]:
    names = [cell.casefold() for cell in header]
    found: dict[str, int] = {}
    for key, words in (
        ("query", ("query", "search")),
        ("results", ("result",)),
        ("shortlisted", ("shortlist", "kept", "candidate")),
    ):
        for index, name in enumerate(names):
            if index not in found.values() and any(word in name for word in words):
                found[key] = index
                break
    return found if len(found) == 3 else {}


def _count(cell: str) -> int | None:
    match = re.fullmatch(r"[~≈]?\s*(\d+)\+?", cell.replace(",", "").strip())
    return int(match[1]) if match else None


def research_funnel(corpus: Path) -> dict[str, Any]:
    """How many searches, candidates, pages, and sources a build went through.

    Searches come from the agent's own log. The other counts come from Liner's files.
    """
    log = read_search_log(corpus / _LONGLIST)
    evaluated: list[Any] = []
    try:
        value = parse_yaml(corpus / _EVALUATION, "evaluation")
    except CheckFailed:
        value = None
    if isinstance(value, dict) and isinstance(value.get("candidates"), list):
        evaluated = value["candidates"]
    kept = sum(
        1
        for item in evaluated
        if isinstance(item, dict) and text_field(item, "decision") in {"kept", "trim"}
    )
    fetched = corpus / ".liner-runs" / "fetched"
    pages = len(list(fetched.glob("*.md"))) if fetched.is_dir() else 0
    final = 0
    try:
        tape = yaml.safe_load((corpus / "tape.yaml").read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError):
        tape = None
    if isinstance(tape, dict) and isinstance(tape.get("sources"), list):
        final = len(tape["sources"])
    return {
        "contract": FUNNEL_CONTRACT,
        "version": CONTRACT_VERSION,
        "searches": len(log) if log else None,
        "search_results": sum(row["results"] or 0 for row in log) if log else None,
        "searches_reported_by": "agent",
        "candidates": len(read_candidate_longlist(corpus / _LONGLIST)),
        "pages_read": pages,
        "evaluated": len(evaluated),
        "kept": kept,
        "final": final,
    }


def normalize_candidate_url(url: str) -> str:
    return re.sub(r"[)\],.;:]+$", "", url.strip())


def _first_url(line: str) -> str:
    link = _MARKDOWN_LINK.search(line)
    if link:
        return normalize_candidate_url(link[2])
    match = _URL.search(line)
    return normalize_candidate_url(match[0]) if match else ""


def _title_from_line(line: str, url: str) -> str | None:
    link = _MARKDOWN_LINK.search(line)
    if link:
        return link[1].strip()
    before_url = line[: line.find(url)]
    cleaned = re.sub(r"^\s*[-*]\s*", "", before_url).replace("**", "")
    return re.sub(r"[-—:|\s]+$", "", cleaned).strip() or None


def _check_candidates(corpus: Path) -> None:
    path = corpus / _LONGLIST
    body = _read_text(path)
    if _TODO.search(body):
        raise CheckFailed(f"{_LONGLIST} still contains TODO placeholders")
    if "https://example.com/..." in body:
        raise CheckFailed(f"{_LONGLIST} still contains the example URL placeholder")
    if not read_candidate_longlist(path):
        raise CheckFailed(f"{_LONGLIST} has no URL candidates")


def _check_evaluation(corpus: Path) -> None:
    value = parse_yaml(corpus / _EVALUATION, "evaluation")
    if not isinstance(value, dict):
        raise CheckFailed(f"{_EVALUATION} is not a YAML mapping")
    candidates = value.get("candidates")
    if not isinstance(candidates, list) or not candidates:
        raise CheckFailed(f"{_EVALUATION} has no candidates list")
    expected = len(read_candidate_longlist(corpus / _LONGLIST))
    if expected and len(candidates) != expected:
        raise CheckFailed(
            f"{_EVALUATION} has {len(candidates)} candidates, "
            f"but {_LONGLIST} has {expected} URL candidates"
        )
    for i, candidate in enumerate(candidates):
        label = f"{_EVALUATION} candidates[{i}]"
        if not isinstance(candidate, dict):
            raise CheckFailed(f"{label} is not a mapping")
        if not text_field(candidate, "url"):
            raise CheckFailed(f"{label}.url is required")
        decision = text_field(candidate, "decision")
        if decision not in _DECISIONS:
            raise CheckFailed(f"{label}.decision is invalid: {decision or '(empty)'}")
        if decision == "dropped":
            continue
        rating = number_field(candidate, "rating")
        if rating is None or rating < 1 or rating > 5:
            raise CheckFailed(f"{label}.rating must be 1-5 for {decision}")
        jtbd_fit = text_field(candidate, "jtbd_fit")
        if jtbd_fit not in _JTBD_FITS:
            raise CheckFailed(f"{label}.jtbd_fit is invalid: {jtbd_fit or '(empty)'}")
        if not text_field(candidate, "section"):
            raise CheckFailed(f"{label}.section is required for {decision}")
        if not text_field(candidate, "note"):
            raise CheckFailed(f"{label}.note is required for {decision}")
        _check_evidence(candidate, label, decision)


def _check_evidence(candidate: dict[str, Any], label: str, decision: str) -> None:
    fetch_status = text_field(candidate, "fetch_status")
    if fetch_status not in _FETCH_STATUSES:
        raise CheckFailed(
            f"{label}.fetch_status must be readable, partial, metadata_only, "
            f"or unavailable for {decision}"
        )
    if fetch_status not in _USABLE_FETCH_STATUSES:
        raise CheckFailed(
            f"{label}.fetch_status cannot be {fetch_status} for {decision}; "
            "unavailable or metadata-only candidates must be dropped"
        )
    quality = text_field(candidate, "content_quality")
    if quality not in _CONTENT_QUALITIES:
        raise CheckFailed(f"{label}.content_quality must be high, medium, or low for {decision}")
    if quality not in _USABLE_CONTENT_QUALITIES:
        raise CheckFailed(
            f"{label}.content_quality cannot be low for {decision}; "
            "low-quality content must be dropped"
        )
    if len(text_list_field(candidate, "evidence")) < 2:
        raise CheckFailed(
            f"{label}.evidence needs at least two content-specific bullets for {decision}"
        )


def _check_quality(corpus: Path) -> None:
    body = _read_text(corpus / _QUALITY)
    if _TODO.search(body):
        raise CheckFailed(f"{_QUALITY} still contains TODO placeholders")
    for marker in _QUALITY_MARKERS:
        if marker not in body:
            raise CheckFailed(f"{_QUALITY} is missing {marker}")
    try:
        _check_evaluation(corpus)
    except CheckFailed as failure:
        raise CheckFailed(f"{_EVALUATION} is invalid after quality checks: {failure}") from failure


def _check_synthesis(corpus: Path) -> None:
    body = _read_text(corpus / "synthesis.md")
    if "Replace this placeholder" in body or _TODO.search(body):
        raise CheckFailed("synthesis.md still contains placeholder text")
    for marker in _SYNTHESIS_MARKERS:
        if marker not in body:
            raise CheckFailed(f"synthesis.md is missing {marker}")
    _check_citations(body, corpus)


def _check_citations(body: str, corpus: Path) -> None:
    parsed = parse_citations(body)
    undefined = sorted(set(parsed.references) - set(parsed.targets))
    if undefined:
        raise CheckFailed(f"synthesis.md cites [^{undefined[0]}] without defining it")
    unused = sorted(set(parsed.targets) - set(parsed.references))
    if unused:
        raise CheckFailed(f"synthesis.md defines [^{unused[0]}] but never cites it")
    available = _citable_sources(corpus)
    for label, target in parsed.targets.items():
        if _citation_key(target) not in available:
            raise CheckFailed(f"synthesis.md [^{label}] is not a kept source: {target}")
    cited = {_citation_key(target) for target in parsed.targets.values()}
    if len(cited) < min(_MIN_CITED_SOURCES, len(available)):
        raise CheckFailed(
            f"synthesis.md must cite at least {min(_MIN_CITED_SOURCES, len(available))} "
            f"sources with [^label] footnotes, and cites {len(cited)}"
        )


def _citable_sources(corpus: Path) -> set[str]:
    sources: list[Any] = []
    evaluation = _optional_yaml(corpus / _EVALUATION, "evaluation")
    if isinstance(evaluation, dict) and isinstance(evaluation.get("candidates"), list):
        sources.extend(
            candidate
            for candidate in evaluation["candidates"]
            if isinstance(candidate, dict) and text_field(candidate, "decision") != "dropped"
        )
    tape = _optional_yaml(corpus / "tape.yaml", "tape")
    if isinstance(tape, dict) and isinstance(tape.get("sources"), list):
        sources.extend(tape["sources"])
    locators = (
        text_field(source, field)
        for source in sources
        if isinstance(source, dict)
        for field in ("url", "path")
    )
    return {_citation_key(locator) for locator in locators if locator}


def _optional_yaml(path: Path, label: str) -> Any:
    return parse_yaml(path, label) if path.is_file() else None


def _citation_key(locator: str) -> str:
    return normalize_candidate_url(locator) if "://" in locator else locator


def _check_assembly(corpus: Path) -> None:
    value = parse_yaml(corpus / _DRAFT, "assembly draft")
    if not isinstance(value, dict):
        raise CheckFailed(f"{_DRAFT} is not a YAML mapping")
    sources = value.get("sources")
    if not isinstance(sources, list) or not sources:
        raise CheckFailed(f"{_DRAFT} has no sources list")
    for i, source in enumerate(sources):
        _check_draft_source(source, i)
    _check_keeps_tape_sources(sources, corpus / "tape.yaml")
    manifest = _active_manifest_sources(corpus / "local-sources/sources-manifest.yaml")
    _check_keeps_manifest_sources(sources, manifest)
    protected = {key for key in map(_source_key, manifest) if key is not None}
    _check_draft_evidence(sources, corpus / _EVALUATION, protected)


def _check_draft_source(source: Any, i: int) -> None:
    label = f"{_DRAFT} sources[{i}]"
    if not isinstance(source, dict):
        raise CheckFailed(f"{label} is not a mapping")
    source_type = text_field(source, "type") or "web"
    if source_type not in _SOURCE_TYPES:
        raise CheckFailed(f"{label}.type is invalid: {source_type}")
    if source_type == "local_file":
        if not text_field(source, "path"):
            raise CheckFailed(f"{label}.path is required for local_file")
    elif source_type == "skill":
        if not text_field(source, "path") and not text_field(source, "url"):
            raise CheckFailed(f"{label}.path or url is required for skill")
    elif not text_field(source, "url"):
        raise CheckFailed(f"{label}.url is required for {source_type}")
    priority = text_field(source, "priority")
    if priority not in _PRIORITIES:
        raise CheckFailed(f"{label}.priority is invalid: {priority or '(empty)'}")
    kind = text_field(source, "kind")
    if kind not in _KINDS:
        raise CheckFailed(f"{label}.kind is invalid: {kind or '(empty)'}")
    if not text_field(source, "section"):
        raise CheckFailed(f"{label}.section is required")
    if not text_field(source, "note"):
        raise CheckFailed(f"{label}.note is required")


def _check_keeps_tape_sources(draft: list[dict[str, Any]], tape_path: Path) -> None:
    try:
        tape = parse_yaml(tape_path, "tape")
    except CheckFailed as failure:
        raise CheckFailed(
            f"tape.yaml could not be parsed for custom-source preservation: {failure}"
        ) from failure
    if not isinstance(tape, dict):
        raise CheckFailed("tape.yaml is not a YAML mapping for custom-source preservation")
    current = tape.get("sources")
    if not isinstance(current, list):
        return
    custom = [
        source
        for source in current
        if isinstance(source, dict) and text_field(source, "type") in {"local_file", "skill"}
    ]
    missing = _missing_sources(custom, draft)
    if missing:
        raise CheckFailed(f"{_DRAFT} dropped existing custom source: {_source_label(missing[0])}")


def _check_keeps_manifest_sources(
    draft: list[dict[str, Any]], manifest: list[dict[str, Any]]
) -> None:
    missing = _missing_sources(manifest, draft)
    if missing:
        raise CheckFailed(f"{_DRAFT} dropped active custom source: {_source_label(missing[0])}")


def _missing_sources(
    required: list[dict[str, Any]], draft: list[dict[str, Any]]
) -> list[dict[str, Any]]:
    draft_keys = {key for key in map(_source_key, draft) if key is not None}
    return [
        source
        for source in required
        if (key := _source_key(source)) is not None and key not in draft_keys
    ]


def _active_manifest_sources(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    try:
        manifest = parse_yaml(path, "source manifest")
    except CheckFailed:
        return []
    items = manifest.get("sources") if isinstance(manifest, dict) else None
    if not isinstance(items, list):
        return []
    return [
        item["source"]
        for item in items
        if isinstance(item, dict)
        and item.get("active") is not False
        and isinstance(item.get("source"), dict)
    ]


def _check_draft_evidence(
    draft: list[dict[str, Any]], evaluation_path: Path, protected: set[str]
) -> None:
    value = parse_yaml(evaluation_path, "evaluation")
    if not isinstance(value, dict):
        raise CheckFailed(f"{_EVALUATION} is not a YAML mapping for assembly evidence checks")
    candidates = value.get("candidates")
    if not isinstance(candidates, list):
        raise CheckFailed(f"{_EVALUATION} has no candidates list for assembly evidence checks")
    by_url = {
        normalize_candidate_url(text_field(candidate, "url")): candidate
        for candidate in candidates
        if isinstance(candidate, dict) and text_field(candidate, "url")
    }
    for i, source in enumerate(draft):
        source_type = text_field(source, "type") or "web"
        reused = source_type == "liner" and "/" in (text_field(source, "url") or "")[8:]
        if source_type not in {"web", "youtube"} and not reused:
            continue
        key = _source_key(source)
        if key is not None and key in protected:
            continue
        url = text_field(source, "url")
        candidate = by_url.get(normalize_candidate_url(url))
        if candidate is None:
            raise CheckFailed(f"{_DRAFT} sources[{i}] has no evaluation evidence for {url}")
        decision = text_field(candidate, "decision")
        if decision not in {"kept", "trim"}:
            raise CheckFailed(
                f"{_DRAFT} sources[{i}] includes {url}, "
                f"but evaluation decision is {decision or '(empty)'}"
            )
        _check_evidence(candidate, f"{_EVALUATION} candidate for {url}", decision)


def _source_key(source: dict[str, Any]) -> str | None:
    source_type = text_field(source, "type")
    url = text_field(source, "url")
    path = text_field(source, "path")
    if source_type in {"web", "youtube", "repository", "liner"}:
        return f"{source_type}|{normalize_candidate_url(url)}" if url else None
    if source_type == "local_file":
        return f"local_file|{path}|{text_field(source, 'citation')}" if path else None
    if source_type == "skill":
        return f"skill|{path}|{url}" if path or url else None
    return None


def _source_label(source: dict[str, Any]) -> str:
    source_type = text_field(source, "type") or "unknown"
    identity = (
        text_field(source, "path")
        or text_field(source, "url")
        or text_field(source, "citation")
        or "(missing id)"
    )
    return f"{source_type} {identity}"


def _read_text(path: Path) -> str:
    if not path.exists():
        raise CheckFailed(f"{path} is missing")
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError as error:
        raise CheckFailed(f"{path} could not be read: {error}") from error


def parse_yaml(path: Path, label: str) -> Any:
    body = _read_text(path)
    try:
        return yaml.load(body, Loader=_ArtifactLoader)  # noqa: S506
    except yaml.YAMLError as error:
        raise CheckFailed(f"{label} YAML could not be parsed: {error}") from error


def text_field(record: dict[str, Any], key: str) -> str:
    value = record.get(key)
    return value.strip() if isinstance(value, str) else ""


def number_field(record: dict[str, Any], key: str) -> float | None:
    value = record.get(key)
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    if not isinstance(value, str):
        return None
    text = value.strip()
    if not text:
        return 0.0
    if _PREFIXED_INTEGER.match(text):
        try:
            return float(int(text, 0))
        except ValueError:
            return None
    if not _DECIMAL.match(text):
        return None
    parsed = float(text)
    return parsed if math.isfinite(parsed) else None


def text_list_field(record: dict[str, Any], key: str) -> list[str]:
    value = record.get(key)
    if not isinstance(value, list):
        return []
    return [item for item in map(_list_item_text, value) if item]


def _list_item_text(item: Any) -> str:
    if isinstance(item, str):
        return item.strip()
    if not isinstance(item, dict) or len(item) != 1:
        return ""
    key, value = next(iter(item.items()))
    if isinstance(value, bool):
        return f"{key}: {str(value).lower()}".strip()
    if not isinstance(value, (str, int, float)):
        return ""
    return f"{key}: {value}".strip()


# Rules that hold a draft back for questions. Liner's Answer Style drafts first and
# asks after, so a rule like these disagrees with it unless a runtime boundary needs it.
# It's a heuristic, so approvals show the lines as warnings rather than blocking them.
_ASK_FIRST = re.compile(
    r"\bbefore (?:drafting|you draft)\b"
    r"|\bbefore (?:you )?(?:write|writing|produce|producing|give|giving) (?:a |the |any )?draft"
    r"|\b(?:ask|interview|clarify) (?:\w+ ){0,4}first\b"
    r"|\bask before\b"
    r"|\bask none\b",
    re.IGNORECASE,
)
# "Don't draft until they answer" holds the draft back with or without a negation.
_DRAFT_UNTIL = re.compile(r"\bdraft (?:\w+ ){0,3}until\b", re.IGNORECASE)
_NEGATION = re.compile(r"\b(?:don't|do not|never|not|no)\W+(?:\w+\W+){0,2}$", re.IGNORECASE)


def asking_conflicts(files: dict[str, str]) -> list[dict[str, Any]]:
    """Lines that tell the agent to ask before drafting, against the draft-first style."""
    conflicts: list[dict[str, Any]] = []
    for name, text in files.items():
        for number, line in enumerate(text.splitlines(), start=1):
            if _DRAFT_UNTIL.search(line) or any(
                not _NEGATION.search(line[: match.start()]) for match in _ASK_FIRST.finditer(line)
            ):
                conflicts.append({"file": name, "line": number, "text": line.strip()})
    return conflicts


_CHECKS = {
    "candidates": _check_candidates,
    "evaluation": _check_evaluation,
    "quality": _check_quality,
    "synthesis": _check_synthesis,
    "assembly": _check_assembly,
}
