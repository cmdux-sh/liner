from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from liner.project import (
    SYNTHESIS_PLACEHOLDER,
    ProjectFolder,
    project_skill_status,
    read_liner_metadata,
    status_snapshot,
)
from liner.tape import TapeValidationError, load_tape
from liner.types import Tape

PROGRESS_FILENAME = ".liner-progress.json"
GATES_FILENAME = ".liner-gates.json"


@dataclass(frozen=True, slots=True)
class PhaseDefinition:
    id: str
    label: str
    artifact: str | None
    gate_key: str | None = None


PHASES: tuple[PhaseDefinition, ...] = (
    PhaseDefinition("framing", "Framing", "working/01-jtbd-and-knowledge-map.md"),
    PhaseDefinition("gate0", "Confirm framing", None, "gate0Accepted"),
    PhaseDefinition("candidates", "Candidate discovery", "working/02-candidate-longlist.md"),
    PhaseDefinition("gate1", "Confirm candidates", None, "gate1Accepted"),
    PhaseDefinition("evaluation", "Evaluation", "working/03-evaluation.yaml"),
    PhaseDefinition("quality", "Quality checks", "working/04-quality-checks.md"),
    PhaseDefinition("gate2", "Confirm evaluation", None, "gate2Accepted"),
    PhaseDefinition("synthesis", "Synthesis", "synthesis.md"),
    PhaseDefinition("assembly", "Assembly", "working/07-tape-draft.yaml"),
    PhaseDefinition("compile", "Compile", "MIXTAPE.md"),
)

PLACEHOLDER_MARKERS = (
    "TODO — a single specific sentence",
    "TODO — Phase 1 replaces this",
    "Quantity over precision",
    "candidates: []",
    "Run each test deliberately",
)


def build_status_payload(project: ProjectFolder, manifest_payload: dict[str, Any]) -> dict[str, Any]:
    """Return phase/progress status derived from the same cursor model as the TUI."""
    folder = project.corpus_path
    progress = _read_progress(folder)
    tape = _load_tape_or_none(folder)
    gates = _read_gates(folder)
    if progress["source"] == "missing":
        progress = _infer_progress(folder, tape, gates)

    step = int(progress["step"])
    runs_by_phase = _runs_by_phase(manifest_payload)
    phases: list[dict[str, Any]] = []
    for index, phase in enumerate(PHASES):
        artifact = _artifact_status(folder, phase.artifact)
        phase_runs = runs_by_phase.get(phase.id, [])
        phase_payload: dict[str, Any] = {
            "id": phase.id,
            "label": phase.label,
            "index": index,
            "status": _status_for_index(index, step),
            "artifact": artifact,
            "runs": _run_status(phase_runs),
        }
        if phase.gate_key is not None:
            phase_payload["gate"] = {
                "key": phase.gate_key,
                "accepted": bool(gates.get(phase.gate_key)),
            }
        phases.append(phase_payload)

    current_phase = PHASES[step].id if 0 <= step < len(PHASES) else None
    metadata = read_liner_metadata(project)
    return {
        "progress": {
            "step": step,
            "total": len(PHASES),
            "current_phase": current_phase,
            "source": progress["source"],
            "last_touched": progress.get("last_touched"),
            "error": progress.get("error"),
        },
        "gates": gates,
        "phases": phases,
        "status_snapshot": status_snapshot(project),
        "project_skill": project_skill_status(metadata),
    }


def _read_progress(folder: Path) -> dict[str, Any]:
    path = folder / PROGRESS_FILENAME
    if not path.is_file():
        return {"step": 0, "source": "missing", "last_touched": None}
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        return {
            "step": 0,
            "source": "default",
            "last_touched": None,
            "error": str(error),
        }
    raw_step = raw.get("step") if isinstance(raw, dict) else None
    step = int(raw_step) if isinstance(raw_step, (int, float)) else 0
    step = max(0, min(len(PHASES), step))
    last_touched = raw.get("lastTouched") if isinstance(raw, dict) else None
    return {
        "step": step,
        "source": "file",
        "last_touched": last_touched if isinstance(last_touched, str) else None,
    }


def _infer_progress(folder: Path, tape: Tape | None, gates: dict[str, bool]) -> dict[str, Any]:
    step = 0
    for phase in PHASES:
        if not _phase_complete(folder, tape, gates, phase):
            break
        step += 1
    return {"step": step, "source": "inferred", "last_touched": None}


def _phase_complete(folder: Path, tape: Tape | None, gates: dict[str, bool], phase: PhaseDefinition) -> bool:
    if phase.gate_key is not None:
        return bool(gates.get(phase.gate_key))
    if phase.id == "framing":
        return bool(tape and tape.jtbd and tape.jtbd.strip()) and _artifact_has_real_content(
            folder / "working/01-jtbd-and-knowledge-map.md"
        )
    if phase.id in {"candidates", "evaluation", "quality"} and phase.artifact is not None:
        return _artifact_has_real_content(folder / phase.artifact)
    if phase.id == "synthesis":
        return _synthesis_complete(folder / "synthesis.md")
    if phase.id == "assembly":
        return bool(tape and len(tape.sources) > 0)
    if phase.id == "compile":
        return (folder / "MIXTAPE.md").is_file()
    return False


def _load_tape_or_none(folder: Path) -> Tape | None:
    try:
        return load_tape(folder / "tape.yaml")
    except (TapeValidationError, FileNotFoundError, OSError):
        return None


def _read_gates(folder: Path) -> dict[str, bool]:
    state = {"gate0Accepted": False, "gate1Accepted": False, "gate2Accepted": False}
    path = folder / GATES_FILENAME
    if not path.is_file():
        return state
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return state
    if not isinstance(raw, dict):
        return state
    for key in state:
        state[key] = bool(raw.get(key))
    return state


def _artifact_status(folder: Path, rel_path: str | None) -> dict[str, Any] | None:
    if rel_path is None:
        return None
    path = folder / rel_path
    exists = path.is_file()
    size = path.stat().st_size if exists else 0
    return {
        "path": rel_path,
        "exists": exists,
        "bytes": size,
        "has_real_content": _artifact_has_real_content(path),
    }


def _artifact_has_real_content(path: Path) -> bool:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return False
    if not text.strip():
        return False
    return not any(marker in text for marker in PLACEHOLDER_MARKERS)


def _synthesis_complete(path: Path) -> bool:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return False
    stripped = text.strip()
    return (
        bool(stripped)
        and stripped != SYNTHESIS_PLACEHOLDER.strip()
        and "Replace this placeholder" not in text
    )


def _status_for_index(index: int, step: int) -> str:
    if index < step:
        return "complete"
    if index == step:
        return "in_progress"
    return "not_started"


def _runs_by_phase(manifest_payload: dict[str, Any]) -> dict[str, list[dict[str, Any]]]:
    runs = manifest_payload.get("runs")
    if not isinstance(runs, list):
        return {}
    grouped: dict[str, list[dict[str, Any]]] = {}
    for run in runs:
        if not isinstance(run, dict):
            continue
        phase = run.get("task_label")
        if not isinstance(phase, str) or not phase:
            continue
        grouped.setdefault(phase, []).append(run)
    return grouped


def _run_status(runs: list[dict[str, Any]]) -> dict[str, Any]:
    latest = runs[-1] if runs else {}
    return {
        "count": len(runs),
        "latest_exit_code": latest.get("exit_code") if isinstance(latest.get("exit_code"), int) else None,
        "latest_log_path": latest.get("log_path") if isinstance(latest.get("log_path"), str) else None,
    }
