"""An inbox for what real work learned, waiting for the person to review it."""

from __future__ import annotations

from contextlib import suppress
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from liner.gaps import GapError, close_gap, list_gaps
from liner.maintenance import (
    ProjectApplyError,
    ProjectInspectionError,
    apply_change_set,
    plan_source_add,
)
from liner.project import ProjectFolder
from liner.working_log import WorkingLogError, locked, read_yaml, write_yaml

FINDINGS_CONTRACT = "liner.findings"
CONTRACT_VERSION = 1
FINDINGS_PATH = "working/findings.yaml"

_MAX_FINDING_CHARS = 2000
_MAX_EVIDENCE_CHARS = 500
_SOURCE_FOLDER = "local-sources/findings"
_NEXT_STEPS = (
    "This changes the Sources, so review the synthesis, compile, and review the "
    "Operating Layer before relying on the update."
)


class FindingError(WorkingLogError):
    """A finding could not be recorded or reviewed."""


def add_finding(
    project: ProjectFolder, text: str, *, evidence: str = "", closes: str | None = None
) -> dict[str, Any]:
    """Put a finding in the inbox. Nothing about the liner changes until it is accepted.

    A finding already pending or rejected comes back as it stands, so a declined
    one isn't raised again.
    """
    body = text.strip()
    where = " ".join(evidence.split())
    if not body:
        raise FindingError("Describe what the work found.")
    if len(body) > _MAX_FINDING_CHARS:
        raise FindingError(f"Keep a finding under {_MAX_FINDING_CHARS} characters.")
    if len(where) > _MAX_EVIDENCE_CHARS:
        raise FindingError(f"Keep the evidence under {_MAX_EVIDENCE_CHARS} characters.")
    if closes is not None and closes not in {gap["id"] for gap in list_gaps(project)["gaps"]}:
        raise FindingError(f"{closes} is not an open gap in {project.path.name}.")
    with locked(_path(project)):
        findings = _read(project)
        for finding in findings:
            if finding["status"] in {"pending", "rejected"} and finding["finding"] == body:
                return finding
        finding = {
            "id": uuid4().hex[:8],
            "finding": body,
            "evidence": where,
            "closes_gap": closes,
            "status": "pending",
            "noted_at": _now(),
        }
        _write(project, [*findings, finding])
    return finding


def list_findings(project: ProjectFolder, *, include_reviewed: bool = False) -> dict[str, Any]:
    findings = _read(project)
    return {
        "contract": FINDINGS_CONTRACT,
        "version": CONTRACT_VERSION,
        "slug": project.path.name,
        "findings": [
            finding for finding in findings if include_reviewed or finding["status"] == "pending"
        ],
    }


def pending_finding_count(corpus: Path) -> int:
    return sum(finding["status"] == "pending" for finding in _read_path(corpus / FINDINGS_PATH))


def reject_finding(project: ProjectFolder, finding_id: str) -> dict[str, Any]:
    """Keep a declined finding on record, so it isn't suggested as new again."""
    with locked(_path(project)):
        findings = _read(project)
        finding = _pending(findings, finding_id)
        finding.update({"status": "rejected", "reviewed_at": _now()})
        _write(project, findings)
    return finding


def accept_finding(project: ProjectFolder, finding_id: str) -> dict[str, Any]:
    """Add a finding to the liner as a Source, through Core's reviewed path."""
    with locked(_path(project)):
        findings = _read(project)
        finding = _pending(findings, finding_id)
        relative = f"{_SOURCE_FOLDER}/{finding['id']}.md"
        receipt_id = None
        # A crash after Core applied leaves the Source in place, so a retry only records it.
        if not _source_is_on_tape(project, relative):
            receipt_id = _add_source(project, finding, relative)
        finding.update({"status": "accepted", "reviewed_at": _now(), "receipt_id": receipt_id})
        _write(project, findings)
    return {**finding, "next": _NEXT_STEPS, "closed_gap": _close_linked_gap(project, finding)}


def _add_source(project: ProjectFolder, finding: dict[str, Any], relative: str) -> str:
    note = project.corpus_path / relative
    try:
        note.parent.mkdir(parents=True, exist_ok=True)
        note.write_text(_note_body(finding), encoding="utf-8")
        change_set = plan_source_add(project.path, _source(finding, relative))
        return str(apply_change_set(project.path, change_set, approved=True).receipt_id)
    except (OSError, ProjectApplyError, ProjectInspectionError) as error:
        if not _source_is_on_tape(project, relative):
            with suppress(OSError):
                note.unlink(missing_ok=True)
        message = error.report.message if isinstance(error, ProjectApplyError) else str(error)
        raise FindingError(f"Core could not add this finding: {message}") from error


def _source_is_on_tape(project: ProjectFolder, relative: str) -> bool:
    tape = read_yaml(project.tape_path)
    sources = tape.get("sources") if isinstance(tape, dict) else None
    return isinstance(sources, list) and any(
        isinstance(source, dict) and source.get("path") == relative for source in sources
    )


def _close_linked_gap(project: ProjectFolder, finding: dict[str, Any]) -> str | None:
    gap_id = finding.get("closes_gap")
    if not gap_id:
        return None
    try:
        close_gap(project, gap_id)
    except GapError:
        return None
    return str(gap_id)


def _source(finding: dict[str, Any], relative: str) -> dict[str, str]:
    return {
        "type": "local_file",
        "path": relative,
        "citation": f"Finding {finding['id']} from real work",
        "section": "Findings from real work",
        "priority": "optional",
        "kind": "example",
        "note": "Reported by an agent during real work and accepted by the curator.",
    }


def _note_body(finding: dict[str, Any]) -> str:
    lines = [f"# Finding {finding['id']}", "", finding["finding"], ""]
    if finding["evidence"]:
        lines.extend([f"Evidence: {finding['evidence']}", ""])
    lines.append(f"Recorded {finding['noted_at']}. Reported by an agent, accepted by the curator.")
    return "\n".join(lines) + "\n"


def _pending(findings: list[dict[str, Any]], finding_id: str) -> dict[str, Any]:
    for finding in findings:
        if finding["id"] == finding_id:
            if finding["status"] != "pending":
                raise FindingError(f"Finding {finding_id} was already {finding['status']}.")
            return finding
    raise FindingError(f"No finding {finding_id}.")


def _path(project: ProjectFolder) -> Path:
    return project.corpus_path / FINDINGS_PATH


def _read(project: ProjectFolder) -> list[dict[str, Any]]:
    return _read_path(_path(project))


def _read_path(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    raw = read_yaml(path)
    findings = raw.get("findings") if isinstance(raw, dict) else None
    if not isinstance(findings, list) or not all(_valid(item) for item in findings):
        raise FindingError(f"{path} is not a findings log. Fix it by hand; nothing was changed.")
    return findings


def _valid(finding: Any) -> bool:
    return (
        isinstance(finding, dict)
        and isinstance(finding.get("id"), str)
        and isinstance(finding.get("finding"), str)
        and isinstance(finding.get("evidence"), str)
        and isinstance(finding.get("noted_at"), str)
        and finding.get("status") in {"pending", "accepted", "rejected"}
    )


def _write(project: ProjectFolder, findings: list[dict[str, Any]]) -> None:
    write_yaml(_path(project), {"version": 1, "findings": findings})


def _now() -> str:
    return datetime.now(UTC).isoformat()
