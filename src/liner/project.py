from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from liner.tape import starter_tape_with_ids

MIXTAPE_DIR = "mixtape"
LINER_METADATA_FILENAME = "liner.yaml"
PROJECT_MILESTONES = {"started", "corpus_ready", "project_complete"}
PROJECT_SKILL_STATUSES = {"active", "missing", "declined", "unresolved"}


class SynthesisReviewRequiredError(RuntimeError):
    """A stale corpus cannot publish before its synthesis disposition is approved."""


SYNTHESIS_PLACEHOLDER = """# Synthesis

> Replace this placeholder with the curator's distilled understanding of the domain
> (typically 800–2000 words). The synthesis is copied verbatim into `MIXTAPE.md`
> when you run `liner compile` and is the first thing the consuming AI reads.

## The framework I see in this domain

TODO — the principles, distinctions, and lenses you use to think about this topic.

## Contested questions and where I stand

TODO — places experts disagree, and the position this mixtape takes.

## When to use this mixtape (and when to look elsewhere)

TODO — what the corpus is good for, what it doesn't cover.
"""

WORKING_JTBD_TEMPLATE = """# JTBD and knowledge map

## Job-to-be-done

TODO — a single specific Job Story. Not the topic — the use case. Required form:
`When [circumstance], I want [motivation], so I can [outcome].` All three slots required.
Examples:
- "When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic."
- "When I review mobile design portfolios as a senior IC hiring manager, I want to compare candidates against a consistent rubric of taste and decision-making, so I can decide who to advance with confidence."

## Knowledge map

TODO — Phase 1 replaces this with 4–8 sections, each with sub-areas. The
example bullets below are placeholders; the agent (or you) will revise them
to reflect the actual structure of the domain.

- Foundations
  - …
- Patterns
  - …
- Craft
  - …
"""

WORKING_LONGLIST_TEMPLATE = """# Candidate long-list

The unfiltered pool of candidate sources from Phase 2. URLs and titles only — no fetching yet.

Group by knowledge-map section. Quantity over precision; you'll cut in Phase 4.

## Section: foundations

- [ ] https://example.com/...

## Section: patterns

- [ ] https://example.com/...
"""

WORKING_EVALUATION_TEMPLATE = """# Evaluation — keep/trim/drop decisions
#
# One entry per candidate from Phase 2. Decisions and rationales from Phase 4
# (after the AI has actually read the fetched content).

candidates: []
#  - url: https://example.com/great-article
#    decision: kept            # kept | trimmed | dropped
#    section: foundations
#    rating: 5                  # 1-5
#    rationale: Canonical reference for the foundations section.
"""

WORKING_QUALITY_TEMPLATE = """# Quality checks (Phase 5)

Run each test deliberately. Document findings even when "nothing to do."

## Test 1 — Redundancy

TODO — any two sources making essentially the same point? Cut the weaker one.

## Test 2 — Coverage

TODO — any bucket in the knowledge map with zero sources? Fill it or explicitly note the omission.

## Test 3 — Disagreement

TODO — strongest claim in the corpus. Is there a credible counter? Include it or note the position taken.

## Test 4 — Framing-gap

TODO — step back. Is there a whole way of thinking about this JTBD that's missing? If yes, revise the knowledge map and revisit Phase 2.

### Perspectives audit

TODO — name 2–3 candidate perspectives and classify stance/concerns coverage.

## Test 5 — Source-kind balance

Distribution: TODO reference / TODO principle / TODO prescription / TODO example

TODO — assign missing kinds first; defend or backfill any zero.

## Test 6 — Note-quality

Checked: TODO kept/trim notes
Repaired: TODO notes

TODO — repair weak notes so each carries a use cue, value/bar, and boundary.
"""


@dataclass(frozen=True, slots=True)
class ProjectFolder:
    path: Path

    @property
    def corpus_path(self) -> Path:
        canonical = self.path / MIXTAPE_DIR
        legacy_tape = self.path / "tape.yaml"
        canonical_tape = canonical / "tape.yaml"
        if canonical_tape.exists():
            return canonical
        if legacy_tape.exists():
            return self.path
        return canonical

    @property
    def liner_metadata_path(self) -> Path:
        return self.path / LINER_METADATA_FILENAME

    @property
    def liner_path(self) -> Path:
        return self.path / "LINER.md"

    @property
    def tape_path(self) -> Path:
        return self.corpus_path / "tape.yaml"

    @property
    def synthesis_path(self) -> Path:
        return self.corpus_path / "synthesis.md"

    @property
    def mixtape_path(self) -> Path:
        return self.corpus_path / "MIXTAPE.md"

    @property
    def sources_dir(self) -> Path:
        return self.corpus_path / "sources"

    @property
    def working_dir(self) -> Path:
        return self.corpus_path / "working"

    @property
    def personal_dir(self) -> Path:
        return self.corpus_path / "personal"

    @property
    def local_sources_dir(self) -> Path:
        return self.corpus_path / "local-sources"

    def is_valid(self) -> bool:
        return self.tape_path.exists()

    def has_synthesis(self) -> bool:
        return self.synthesis_path.exists()


def slugify(text: str, max_length: int = 60) -> str:
    """Lowercase, ASCII, hyphen-separated. Safe for filesystem use."""
    if not text:
        return "untitled"
    normalized = unicodedata.normalize("NFKD", text)
    ascii_text = normalized.encode("ascii", "ignore").decode("ascii")
    ascii_text = ascii_text.lower()
    ascii_text = re.sub(r"[^a-z0-9]+", "-", ascii_text)
    ascii_text = ascii_text.strip("-")
    if not ascii_text:
        return "untitled"
    if len(ascii_text) > max_length:
        ascii_text = ascii_text[:max_length].rstrip("-")
    return ascii_text


def init_project(path: Path, *, force: bool = False) -> ProjectFolder:
    """Create a project folder with starter tape, synthesis placeholder, and working/ stubs.

    `path` may be an existing or new directory. If it does not exist it is created.
    """
    if path.exists() and not path.is_dir():
        raise FileExistsError(f"{path} exists and is not a directory")
    path.mkdir(parents=True, exist_ok=True)

    project = ProjectFolder(path)
    existing_metadata = read_liner_metadata(project)
    if not project.liner_metadata_path.exists() or force:
        existing_project_id = existing_metadata.get("id")
        project_id = (
            existing_project_id
            if isinstance(existing_project_id, str) and existing_project_id.strip()
            else str(uuid4())
        )
        write_liner_metadata(
            project,
            initial_liner_metadata(project_id=project_id),
        )

    project.corpus_path.mkdir(parents=True, exist_ok=True)

    if project.tape_path.exists() and not force:
        raise FileExistsError(f"{project.tape_path} already exists. Use --force to overwrite.")

    project.tape_path.write_text(starter_tape_with_ids(), encoding="utf-8")

    if not project.synthesis_path.exists() or force:
        project.synthesis_path.write_text(SYNTHESIS_PLACEHOLDER, encoding="utf-8")

    project.working_dir.mkdir(parents=True, exist_ok=True)
    _write_if_missing(
        project.working_dir / "01-jtbd-and-knowledge-map.md",
        WORKING_JTBD_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "02-candidate-longlist.md",
        WORKING_LONGLIST_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "03-evaluation.yaml",
        WORKING_EVALUATION_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "04-quality-checks.md",
        WORKING_QUALITY_TEMPLATE,
        force=force,
    )

    return project


def initial_liner_metadata(
    *,
    updated: str | None = None,
    project_id: str | None = None,
) -> dict[str, Any]:
    """Return the root metadata for a fresh Liner Project."""
    metadata: dict[str, Any] = {
        "version": 2,
        "artifact": "liner",
        "mixtape": MIXTAPE_DIR,
        "status": {
            "milestone": "started",
            "stale": False,
            "updated": updated or _now_iso(),
            "corpus": {
                "state": "missing",
                "evidence": f"{MIXTAPE_DIR}/MIXTAPE.md",
            },
            "operating_layer": {
                "state": "missing",
                "evidence": "LINER.md",
            },
        },
        "project_skill": {"status": "missing"},
    }
    if project_id is not None:
        metadata = {"id": project_id, **metadata}
    return metadata


def read_liner_metadata(project: ProjectFolder | Path) -> dict[str, Any]:
    folder = project if isinstance(project, ProjectFolder) else ProjectFolder(project)
    path = folder.liner_metadata_path
    if not path.is_file():
        return {}
    import yaml

    try:
        raw = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError):
        return {}
    if not isinstance(raw, dict):
        return {}
    return raw


def write_liner_metadata(project: ProjectFolder | Path, metadata: dict[str, Any]) -> None:
    folder = project if isinstance(project, ProjectFolder) else ProjectFolder(project)
    import yaml

    folder.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )


def status_snapshot(project: ProjectFolder, *, refresh: bool = False) -> dict[str, Any]:
    metadata = read_liner_metadata(project)
    existing_status = metadata.get("status") if isinstance(metadata.get("status"), dict) else {}
    if not refresh and existing_status:
        snapshot = _normalize_status_snapshot(project, existing_status)
        snapshot["stale"] = _status_snapshot_stale(project, snapshot)
        return _normalize_missing_operating_refresh(project, metadata, snapshot)

    return _infer_status_snapshot(project, metadata, updated=_now_iso() if refresh else None)


def _normalize_missing_operating_refresh(
    project: ProjectFolder,
    metadata: dict[str, Any],
    snapshot: dict[str, Any],
) -> dict[str, Any]:
    if _operating_artifacts_exist(project, metadata):
        return snapshot
    if snapshot.get("milestone") != "corpus_ready":
        return snapshot
    operating = dict(snapshot.get("operating_layer", {}))
    operating["state"] = "pending"
    operating.pop("last_verified_state", None)
    snapshot["operating_layer"] = operating
    raw_refresh = snapshot.get("refresh")
    if not isinstance(raw_refresh, dict):
        return snapshot
    refresh = dict(raw_refresh)
    refresh["operating_layer"] = {
        "state": "approved",
        "disposition": "not_applicable",
    }
    remaining = [
        str(item)
        for item in refresh.get("remaining_artifacts", [])
        if str(item) not in set(_operating_artifacts(metadata))
    ]
    refresh["remaining_artifacts"] = remaining
    synthesis = refresh.get("synthesis")
    corpus = refresh.get("corpus")
    complete = (
        isinstance(synthesis, dict)
        and synthesis.get("state") == "approved"
        and isinstance(corpus, dict)
        and corpus.get("state") == "current"
    )
    refresh["state"] = "current" if complete else "required"
    snapshot["refresh"] = refresh
    snapshot["stale"] = not complete
    return snapshot


def refresh_status_snapshot(project: ProjectFolder) -> dict[str, Any]:
    snapshot = status_snapshot(project, refresh=True)
    if not _can_write_liner_metadata(project):
        return snapshot
    metadata = read_liner_metadata(project)
    if not metadata:
        metadata = initial_liner_metadata()
    metadata["status"] = snapshot
    write_liner_metadata(project, metadata)
    return snapshot


def mark_corpus_ready(project: ProjectFolder) -> dict[str, Any]:
    metadata = read_liner_metadata(project)
    raw_existing_status = metadata.get("status")
    existing_status: dict[str, Any] = (
        dict(raw_existing_status) if isinstance(raw_existing_status, dict) else {}
    )
    refresh = existing_status.get("refresh")
    if isinstance(refresh, dict):
        synthesis = refresh.get("synthesis")
        if not isinstance(synthesis, dict) or synthesis.get("state") != "approved":
            raise SynthesisReviewRequiredError(
                "Synthesis review is required before publishing a refreshed MIXTAPE. "
                "Plan and approve `synthesis.review` first."
            )
        refresh = dict(refresh)
        refresh["corpus"] = {"state": "current", "refreshed_at": _now_iso()}
        operating = refresh.get("operating_layer")
        operating_not_applicable = (
            isinstance(operating, dict)
            and operating.get("state") == "approved"
            and operating.get("disposition") == "not_applicable"
        )
        operating_approved = operating_not_applicable or (
            isinstance(operating, dict)
            and operating.get("state") == "approved"
            and _operating_artifacts_exist(project, metadata)
        )
        refresh["state"] = "current" if operating_approved else "required"
        refresh["remaining_artifacts"] = (
            [] if operating_approved else _operating_artifacts(metadata)
        )
        snapshot = _status_snapshot_payload(
            project,
            milestone=_later_milestone(
                str(existing_status.get("milestone", "started")), "corpus_ready"
            ),
            updated=_now_iso(),
            corpus_ready=True,
            operating_layer_state=(
                "pending"
                if operating_not_applicable
                else "ready"
                if operating_approved and project.liner_path.is_file()
                else "stale"
            ),
            existing_status=existing_status,
        )
        snapshot["stale"] = not operating_approved
        snapshot["refresh"] = refresh
        if not _can_write_liner_metadata(project):
            return snapshot
        metadata["status"] = snapshot
        write_liner_metadata(project, metadata)
        return snapshot
    snapshot = _status_snapshot_payload(
        project,
        milestone="corpus_ready",
        updated=_now_iso(),
        corpus_ready=True,
        operating_layer_state="pending",
        existing_status=existing_status,
    )
    if not _can_write_liner_metadata(project):
        return snapshot
    if not metadata:
        metadata = initial_liner_metadata()
    metadata["status"] = snapshot
    write_liner_metadata(project, metadata)
    return snapshot


def _infer_status_snapshot(
    project: ProjectFolder,
    metadata: dict[str, Any],
    *,
    updated: str | None,
) -> dict[str, Any]:
    raw_existing_status = metadata.get("status")
    existing_status: dict[str, Any] = (
        dict(raw_existing_status) if isinstance(raw_existing_status, dict) else {}
    )
    project_skill = project_skill_status(metadata)
    corpus_ready = project.mixtape_path.is_file()
    operating_ready = project.liner_path.is_file()
    skill_ready = project_skill.get("status") == "active"

    milestone = "started"
    if corpus_ready:
        milestone = "corpus_ready"
    if corpus_ready and operating_ready and skill_ready:
        milestone = "project_complete"

    operating_layer_state = "missing"
    if milestone == "project_complete":
        operating_layer_state = "ready"
    elif corpus_ready:
        operating_layer_state = "pending"

    return _status_snapshot_payload(
        project,
        milestone=milestone,
        updated=updated or _existing_updated(existing_status),
        corpus_ready=corpus_ready,
        operating_layer_state=operating_layer_state,
        existing_status=existing_status,
    )


def _status_snapshot_payload(
    project: ProjectFolder,
    *,
    milestone: str,
    updated: str,
    corpus_ready: bool,
    operating_layer_state: str,
    existing_status: dict[str, Any],
) -> dict[str, Any]:
    operating_layer: dict[str, Any] = {
        "state": operating_layer_state,
        "evidence": "LINER.md",
    }
    existing_operating = existing_status.get("operating_layer")
    if isinstance(existing_operating, dict) and isinstance(existing_operating.get("audit"), str):
        operating_layer["audit"] = existing_operating["audit"]
    else:
        latest_audit = _latest_operating_audit(project)
        if latest_audit is not None:
            operating_layer["audit"] = latest_audit

    return {
        "milestone": milestone,
        "stale": False,
        "updated": updated,
        "corpus": {
            "state": "ready" if corpus_ready else "missing",
            "evidence": _relative_project_path(project, project.mixtape_path),
        },
        "operating_layer": operating_layer,
    }


def record_project_skill_status(
    project: ProjectFolder,
    *,
    status: str,
    name: str | None = None,
    path: str | None = None,
) -> None:
    if status not in PROJECT_SKILL_STATUSES:
        raise ValueError("project skill status must be active or missing")
    metadata = read_liner_metadata(project)
    if not metadata:
        metadata = initial_liner_metadata()
    project_skill: dict[str, Any] = {"status": status}
    if status == "active":
        if name:
            project_skill["name"] = name
        if path:
            project_skill["path"] = path
    metadata["project_skill"] = project_skill
    metadata["status"] = status_snapshot(project, refresh=True)
    write_liner_metadata(project, metadata)


def project_skill_status(metadata: dict[str, Any]) -> dict[str, Any]:
    raw = metadata.get("project_skill")
    if not isinstance(raw, dict):
        return {"status": "missing"}
    status = raw.get("status")
    if status not in PROJECT_SKILL_STATUSES:
        return {"status": "missing"}
    project_skill: dict[str, Any] = {"status": status}
    if status == "active":
        if isinstance(raw.get("name"), str):
            project_skill["name"] = raw["name"]
        if isinstance(raw.get("path"), str):
            project_skill["path"] = raw["path"]
    return project_skill


def _existing_updated(existing_status: dict[str, Any]) -> str:
    updated = existing_status.get("updated")
    if isinstance(updated, datetime):
        return updated.astimezone(UTC).isoformat(timespec="microseconds").replace("+00:00", "Z")
    if isinstance(updated, str) and updated.strip():
        return updated
    return _now_iso()


def _status_snapshot_stale(project: ProjectFolder, existing_status: dict[str, Any]) -> bool:
    if bool(existing_status.get("stale")):
        return True
    updated = existing_status.get("updated")
    updated_at = _parse_status_time(updated)
    if updated_at is None:
        return True
    evidence = _status_evidence_paths(project, existing_status)
    for path in evidence:
        try:
            if path.is_file() and path.stat().st_mtime > updated_at.timestamp():
                return True
        except OSError:
            continue
    return False


def _status_evidence_paths(project: ProjectFolder, existing_status: dict[str, Any]) -> list[Path]:
    paths: list[Path] = []
    for section in ("corpus", "operating_layer"):
        raw = existing_status.get(section)
        if not isinstance(raw, dict):
            continue
        for key in ("evidence", "audit"):
            rel = raw.get(key)
            if isinstance(rel, str) and rel.strip():
                paths.append(project.path / rel)
    if not paths:
        paths.extend([project.mixtape_path, project.liner_path])
    return paths


def _normalize_status_snapshot(project: ProjectFolder, raw: dict[str, Any]) -> dict[str, Any]:
    milestone = raw.get("milestone")
    if not isinstance(milestone, str) or milestone not in PROJECT_MILESTONES:
        milestone = "started"
    raw_corpus = raw.get("corpus")
    corpus: dict[str, Any] = dict(raw_corpus) if isinstance(raw_corpus, dict) else {}
    raw_operating = raw.get("operating_layer")
    operating: dict[str, Any] = dict(raw_operating) if isinstance(raw_operating, dict) else {}
    snapshot: dict[str, Any] = {
        "milestone": milestone,
        "stale": bool(raw.get("stale")),
        "updated": _existing_updated(raw),
        "corpus": {
            "state": _clean_status_string(corpus.get("state"), "missing"),
            "evidence": _clean_status_string(
                corpus.get("evidence"),
                _relative_project_path(project, project.mixtape_path),
            ),
        },
        "operating_layer": _normalize_operating_layer(project, operating),
    }
    if isinstance(raw.get("refresh"), dict):
        snapshot["refresh"] = dict(raw["refresh"])
    return snapshot


def _normalize_operating_layer(project: ProjectFolder, raw: dict[str, Any]) -> dict[str, Any]:
    operating_layer: dict[str, Any] = {
        "state": _clean_status_string(raw.get("state"), "missing"),
        "evidence": _clean_status_string(raw.get("evidence"), "LINER.md"),
    }
    audit = raw.get("audit")
    if isinstance(audit, str) and audit.strip():
        operating_layer["audit"] = audit.strip()
    return operating_layer


def ensure_compile_review_approved(project: ProjectFolder) -> None:
    metadata = read_liner_metadata(project)
    raw_status = metadata.get("status")
    status: dict[str, Any] = dict(raw_status) if isinstance(raw_status, dict) else {}
    refresh = status.get("refresh")
    if not isinstance(refresh, dict):
        return
    synthesis = refresh.get("synthesis")
    if not isinstance(synthesis, dict) or synthesis.get("state") != "approved":
        raise SynthesisReviewRequiredError(
            "Synthesis review is required before publishing a refreshed MIXTAPE. "
            "Plan and approve `synthesis.review` first."
        )


def _later_milestone(left: str, right: str) -> str:
    order = {"started": 0, "corpus_ready": 1, "project_complete": 2}
    return left if order.get(left, 0) >= order.get(right, 0) else right


def _operating_artifacts(metadata: dict[str, Any]) -> list[str]:
    artifacts = ["LINER.md"]
    skill = project_skill_status(metadata)
    if skill.get("status") == "active" and isinstance(skill.get("path"), str):
        artifacts.append(str(skill["path"]))
    return artifacts


def _operating_artifacts_exist(project: ProjectFolder, metadata: dict[str, Any]) -> bool:
    for relative in _operating_artifacts(metadata):
        path = project.path / relative
        if not path.is_file() or path.is_symlink():
            return False
    return True


def _latest_operating_audit(project: ProjectFolder) -> str | None:
    audit_dir = project.path / "working" / "audits"
    if not audit_dir.is_dir():
        return None
    candidates = [path for path in audit_dir.glob("*.md") if path.is_file()]
    if not candidates:
        return None
    latest = max(candidates, key=lambda path: (path.stat().st_mtime, path.name))
    return _relative_project_path(project, latest)


def _relative_project_path(project: ProjectFolder, path: Path) -> str:
    try:
        return path.relative_to(project.path).as_posix()
    except ValueError:
        return path.as_posix()


def _clean_status_string(value: Any, default: str) -> str:
    if isinstance(value, str) and value.strip():
        return value.strip()
    return default


def _parse_status_time(value: Any) -> datetime | None:
    if isinstance(value, datetime):
        parsed = value
    elif isinstance(value, str) and value.strip():
        try:
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
        except ValueError:
            return None
    else:
        return None
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)


def _can_write_liner_metadata(project: ProjectFolder) -> bool:
    if project.liner_metadata_path.exists():
        return True
    return project.corpus_path == project.path / MIXTAPE_DIR


def _now_iso() -> str:
    return datetime.now(UTC).isoformat(timespec="microseconds").replace("+00:00", "Z")


def _write_if_missing(path: Path, content: str, *, force: bool) -> None:
    if path.exists() and not force:
        return
    path.write_text(content, encoding="utf-8")
