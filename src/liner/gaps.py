"""Record what a liner could not answer, so the next research pass can target it."""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from liner.project import ProjectFolder
from liner.working_log import WorkingLogError, locked, read_yaml, write_yaml

GAPS_CONTRACT = "liner.gaps"
CONTRACT_VERSION = 1
GAPS_PATH = "working/gaps.yaml"

_MAX_QUESTION_CHARS = 500


class GapError(WorkingLogError):
    """The gap log could not be read or changed."""


def add_gap(project: ProjectFolder, question: str) -> dict[str, Any]:
    """Record an open question. The same open question is only recorded once."""
    text = " ".join(question.split())
    if not text:
        raise GapError("Describe what the liner could not answer.")
    if len(text) > _MAX_QUESTION_CHARS:
        raise GapError(f"Keep a gap under {_MAX_QUESTION_CHARS} characters.")
    with locked(project.corpus_path / GAPS_PATH):
        gaps = _read(project)
        for gap in gaps:
            if gap["status"] == "open" and gap["question"].casefold() == text.casefold():
                return gap
        gap = {
            "id": uuid4().hex[:8],
            "question": text,
            "status": "open",
            "noted_at": datetime.now(UTC).isoformat(),
        }
        _write(project, [*gaps, gap])
    return gap


def close_gap(project: ProjectFolder, gap_id: str) -> dict[str, Any]:
    with locked(project.corpus_path / GAPS_PATH):
        gaps = _read(project)
        for gap in gaps:
            if gap["id"] == gap_id:
                if gap["status"] != "closed":
                    gap["status"] = "closed"
                    gap["closed_at"] = datetime.now(UTC).isoformat()
                    _write(project, gaps)
                return gap
    raise GapError(f"No gap {gap_id} in {project.path.name}.")


def list_gaps(project: ProjectFolder, *, include_closed: bool = False) -> dict[str, Any]:
    gaps = [gap for gap in _read(project) if include_closed or gap["status"] == "open"]
    return {
        "contract": GAPS_CONTRACT,
        "version": CONTRACT_VERSION,
        "slug": project.path.name,
        "gaps": gaps,
    }


def open_gap_questions(corpus: Path) -> list[str]:
    """The open questions in a corpus folder, oldest first."""
    return [gap["question"] for gap in _read_path(corpus / GAPS_PATH) if gap["status"] == "open"]


def _read(project: ProjectFolder) -> list[dict[str, Any]]:
    return _read_path(project.corpus_path / GAPS_PATH)


def _read_path(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    try:
        raw = read_yaml(path)
    except WorkingLogError as error:
        raise GapError(str(error)) from error
    gaps = raw.get("gaps") if isinstance(raw, dict) else None
    if not isinstance(gaps, list) or not all(_valid(gap) for gap in gaps):
        raise GapError(f"{path} is not a gap log. Fix it by hand; nothing was changed.")
    return gaps


def _valid(gap: Any) -> bool:
    return (
        isinstance(gap, dict)
        and isinstance(gap.get("id"), str)
        and isinstance(gap.get("question"), str)
        and gap.get("status") in {"open", "closed"}
    )


def _write(project: ProjectFolder, gaps: list[dict[str, Any]]) -> None:
    write_yaml(project.corpus_path / GAPS_PATH, {"version": 1, "gaps": gaps})
