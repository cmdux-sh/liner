from __future__ import annotations

import hashlib
import json
from datetime import UTC, datetime
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.compile import compile_project
from liner.config import Config
from liner.maintenance import (
    ProjectApplyError,
    ProjectChangeSet,
    ProjectInspectionError,
    apply_change_set,
    inspect_project,
    plan_operating_layer_review,
    plan_source_add,
    plan_source_update,
    plan_synthesis_review,
)
from liner.project import (
    SynthesisReviewRequiredError,
    ensure_compile_review_approved,
    init_project,
    mark_corpus_ready,
    read_liner_metadata,
)
from liner.types import SourceContent, SourceSpec

runner = CliRunner()


class StubHandler:
    def fetch(self, spec: SourceSpec) -> SourceContent:
        return SourceContent(
            title="Refreshed source",
            url=spec.url or "",
            body="refreshed body",
            fetched_at=datetime.now(UTC).isoformat(),
        )


def _rehash_change_set_payload(payload: dict[str, object]) -> ProjectChangeSet:
    unsigned = dict(payload)
    unsigned.pop("change_set_hash")
    payload["change_set_hash"] = hashlib.sha256(
        json.dumps(unsigned, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()
    return ProjectChangeSet.from_dict(payload)


def _completed_project(tmp_path: Path) -> object:
    project = init_project(tmp_path / "demo")
    project.mixtape_path.write_text("# Published MIXTAPE\n", encoding="utf-8")
    project.liner_path.write_text("# Verified Operating Layer\n", encoding="utf-8")
    metadata = read_liner_metadata(project)
    metadata["status"] = {
        "milestone": "project_complete",
        "stale": False,
        "updated": "2026-01-01T00:00:00Z",
        "corpus": {"state": "ready", "evidence": "mixtape/MIXTAPE.md"},
        "operating_layer": {"state": "ready", "evidence": "LINER.md"},
    }
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )
    return project


def _add_source(project: object) -> object:
    return apply_change_set(
        project.path,
        plan_source_add(
            project.path,
            {"type": "web", "url": "https://example.com/refresh", "note": "Primary"},
        ),
    )


def test_source_mutation_preserves_milestone_and_names_refresh_gates(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)

    plan = plan_source_add(
        project.path,
        {"type": "web", "url": "https://example.com/refresh", "note": "Primary"},
    )
    assert plan.lifecycle["milestone"] == "preserved"
    assert plan.lifecycle["affected_artifacts"] == [
        "mixtape/synthesis.md",
        "mixtape/MIXTAPE.md",
        "LINER.md",
    ]
    tampered_payload = plan.to_dict()
    tampered_payload["lifecycle"] = {
        "milestone": "preserved",
        "stale": False,
        "affected_artifacts": [],
        "next_actions": ["No refresh needed."],
    }
    with pytest.raises(ProjectApplyError, match="lifecycle consequences"):
        apply_change_set(project.path, _rehash_change_set_payload(tampered_payload))
    receipt = apply_change_set(project.path, plan)
    snapshot = inspect_project(project.path)
    refresh = snapshot.lifecycle["refresh"]

    assert snapshot.lifecycle["milestone"] == "project_complete"
    assert snapshot.lifecycle["stale"] is True
    assert refresh["synthesis"]["state"] == "review_required"
    assert refresh["corpus"]["state"] == "compile_required"
    assert refresh["operating_layer"]["state"] == "review_required"
    assert refresh["remaining_artifacts"] == [
        "mixtape/synthesis.md",
        "mixtape/MIXTAPE.md",
        "LINER.md",
    ]
    assert receipt.synthesis_disposition == "review_required"
    assert "LINER.md" in receipt.stale_artifacts
    assert "synthesis.review" in receipt.next_actions[0]


def test_source_refresh_before_operating_layer_returns_to_create_action(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "corpus-only")
    project.mixtape_path.write_text("# Published MIXTAPE\n", encoding="utf-8")
    mark_corpus_ready(project)

    plan = plan_source_add(
        project.path,
        {"type": "web", "url": "https://example.com/refresh", "note": "Primary"},
    )
    assert plan.lifecycle["affected_artifacts"] == [
        "mixtape/synthesis.md",
        "mixtape/MIXTAPE.md",
    ]
    apply_change_set(project.path, plan)
    invalidated = inspect_project(project.path)
    refresh = invalidated.lifecycle["refresh"]
    assert invalidated.lifecycle["operating_layer"]["state"] == "pending"
    assert refresh["operating_layer"] == {
        "state": "approved",
        "disposition": "not_applicable",
    }

    apply_change_set(
        project.path,
        plan_synthesis_review(project.path, "still_current"),
        approved=True,
    )
    mark_corpus_ready(project)

    current = inspect_project(project.path)
    assert current.lifecycle["milestone"] == "corpus_ready"
    assert current.lifecycle["stale"] is False
    assert current.lifecycle["operating_layer"]["state"] == "pending"
    assert current.lifecycle["refresh"]["state"] == "current"
    assert current.lifecycle["refresh"]["remaining_artifacts"] == []


def test_note_update_restarts_refresh_without_rewriting_published_artifacts(
    tmp_path: Path,
) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    source_id = inspect_project(project.path).sources[0].source_id
    mixtape_before = project.mixtape_path.read_bytes()
    liner_before = project.liner_path.read_bytes()

    apply_change_set(
        project.path,
        plan_source_update(project.path, str(source_id), {"note": "Reframed by curator"}),
    )

    assert project.mixtape_path.read_bytes() == mixtape_before
    assert project.liner_path.read_bytes() == liner_before
    assert inspect_project(project.path).lifecycle["refresh"]["synthesis"]["state"] == (
        "review_required"
    )


def test_refresh_requires_review_then_clears_only_after_all_artifacts(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    published_before = project.mixtape_path.read_bytes()

    with pytest.raises(SynthesisReviewRequiredError):
        ensure_compile_review_approved(project)
    assert project.mixtape_path.read_bytes() == published_before

    synthesis_receipt = apply_change_set(
        project.path,
        plan_synthesis_review(project.path, "still_current"),
        approved=True,
    )
    ensure_compile_review_approved(project)
    after_review = inspect_project(project.path)
    assert after_review.lifecycle["stale"] is True
    assert synthesis_receipt.synthesis_disposition == "approved_still_current"

    mark_corpus_ready(project)
    after_compile = inspect_project(project.path)
    assert after_compile.lifecycle["milestone"] == "project_complete"
    assert after_compile.lifecycle["stale"] is True
    assert after_compile.lifecycle["refresh"]["corpus"]["state"] == "current"
    assert after_compile.lifecycle["refresh"]["remaining_artifacts"] == ["LINER.md"]

    final_receipt = apply_change_set(
        project.path,
        plan_operating_layer_review(project.path, "still_current"),
        approved=True,
    )
    final = inspect_project(project.path)
    assert final.lifecycle["stale"] is False
    assert final.lifecycle["refresh"]["state"] == "current"
    assert final_receipt.stale_artifacts == ()


def test_cli_compile_refuses_to_publish_before_synthesis_review(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    published_before = project.mixtape_path.read_bytes()

    result = runner.invoke(app, ["compile", str(project.path), "--no-cache"])

    assert result.exit_code == 1
    assert "Synthesis review is required" in result.output
    assert project.mixtape_path.read_bytes() == published_before


def test_operating_layer_review_requires_successful_compile(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    apply_change_set(
        project.path,
        plan_synthesis_review(project.path, "still_current"),
        approved=True,
    )

    with pytest.raises(ProjectInspectionError, match="successful corpus compile"):
        plan_operating_layer_review(project.path, "still_current")


def test_compile_publication_is_atomic_and_records_receipt(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    apply_change_set(
        project.path,
        plan_synthesis_review(project.path, "still_current"),
        approved=True,
    )
    before_mixtape = project.mixtape_path.read_bytes()
    before_status = read_liner_metadata(project)["status"]

    import liner.output.mixtape as output

    original_write = output.write_mixtape

    def fail_after_staged_write(staged_project: object, result: object) -> None:
        original_write(staged_project, result)
        raise OSError("simulated staged publication failure")

    monkeypatch.setattr(output, "write_mixtape", fail_after_staged_write)
    with pytest.raises(ProjectApplyError, match="active Project was left unchanged"):
        compile_project(
            project,
            cache=None,
            handlers={"web": StubHandler()},
            config=Config(),
        )
    assert project.mixtape_path.read_bytes() == before_mixtape
    assert read_liner_metadata(project)["status"] == before_status

    monkeypatch.setattr(output, "write_mixtape", original_write)
    compile_project(
        project,
        cache=None,
        handlers={"web": StubHandler()},
        config=Config(),
    )
    receipts = sorted((project.corpus_path / ".liner-runs" / "maintenance").glob("*.json"))
    compile_receipts = [
        json.loads(path.read_text(encoding="utf-8"))
        for path in receipts
        if json.loads(path.read_text(encoding="utf-8"))["operations"][0]["type"] == "corpus.compile"
    ]
    assert len(compile_receipts) == 1
    assert compile_receipts[0]["stale_artifacts"] == ["LINER.md"]
    assert "mixtape/MIXTAPE.md" in compile_receipts[0]["operations"][0]["refreshed_artifacts"]


def test_operating_review_binds_declared_skill_and_requires_artifacts(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    apply_change_set(
        project.path,
        plan_synthesis_review(project.path, "still_current"),
        approved=True,
    )
    mark_corpus_ready(project)
    project.liner_path.unlink()
    with pytest.raises(ProjectInspectionError, match="must exist"):
        plan_operating_layer_review(project.path, "still_current")

    project.liner_path.write_text("# Restored\n", encoding="utf-8")
    victim = project.path / "victim.md"
    victim.write_text("keep me\n", encoding="utf-8")
    plan = plan_operating_layer_review(project.path, "patch", liner_content="# Reviewed\n")
    payload = plan.to_dict()
    payload["operations"][0]["skill_path"] = "victim.md"
    payload["operations"][0]["skill_content"] = "overwritten"
    payload["operations"][0]["expected_skill_hash"] = "missing"
    payload["operations"][0]["proposed_skill_hash"] = (
        "sha256:" + hashlib.sha256(b"overwritten").hexdigest()
    )
    payload["file_effects"]["write"].append("victim.md")
    crafted = _rehash_change_set_payload(payload)
    victim_before = victim.read_bytes()
    with pytest.raises(ProjectApplyError):
        apply_change_set(project.path, crafted, approved=True)
    assert victim.read_bytes() == victim_before


def test_reviewed_patches_are_separate_semantic_changes_and_preserve_other_content(
    tmp_path: Path,
) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    mixtape_before = project.mixtape_path.read_bytes()
    liner_before = project.liner_path.read_bytes()

    synthesis = plan_synthesis_review(project.path, "patch", content="# Revised synthesis\n")
    assert synthesis.risk == "semantic"
    assert synthesis.approval_required is True
    apply_change_set(project.path, synthesis, approved=True)
    assert project.synthesis_path.read_text(encoding="utf-8") == "# Revised synthesis\n"
    assert project.mixtape_path.read_bytes() == mixtape_before
    assert project.liner_path.read_bytes() == liner_before

    mark_corpus_ready(project)
    operating = plan_operating_layer_review(
        project.path,
        "patch",
        liner_content="# Reviewed Operating Layer\n",
    )
    assert operating.risk == "semantic"
    apply_change_set(project.path, operating, approved=True)
    assert project.liner_path.read_text(encoding="utf-8") == "# Reviewed Operating Layer\n"
    assert project.mixtape_path.read_bytes() == mixtape_before


def test_real_cli_plans_explicit_still_current_review(tmp_path: Path) -> None:
    project = _completed_project(tmp_path)
    _add_source(project)
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "synthesis.review", "disposition": "still_current"},
    }

    result = runner.invoke(
        app,
        [
            "project",
            "plan",
            str(project.path),
            "--request-json",
            json.dumps(request),
            "--json",
        ],
    )

    assert result.exit_code == 0, result.output
    change_set = json.loads(result.stdout)
    assert change_set["risk"] == "semantic"
    assert change_set["operations"][0]["disposition"] == "still_current"
    assert "content" not in change_set["operations"][0]
