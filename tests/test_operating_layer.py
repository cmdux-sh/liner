from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from liner.cli import app
from liner.maintenance import (
    ProjectApplyError,
    ProjectChangeSet,
    ProjectInspectionError,
    _payload_hash,
    _stable_uuid,
    _text_hash,
    apply_change_set,
    inspect_project,
    plan_operating_layer_create,
    plan_source_add,
    plan_synthesis_review,
)
from liner.operating_layer import project_skill_name, render_operating_layer
from liner.project import ProjectFolder, init_project, mark_corpus_ready, read_liner_metadata

SYNTHESIS = """# Hydration

Hydration follows the flour.

## Generative rules

- Start from absorption.
  - Not a top-level rule.
- Adjust in small steps.

## Stances this corpus takes

- Weigh everything.
"""
QUALITY = """Core action: Pick a hydration.

## Test 7
- Pattern: not this one

## Test 8

- Pattern: Reference translation.
- Finding: Passes.
"""


@pytest.fixture
def project(tmp_path: Path) -> ProjectFolder:
    folder = init_project(tmp_path / "hydration")
    tape = yaml.safe_load(folder.tape_path.read_text(encoding="utf-8"))
    tape.update({"title": "Sourdough Hydration", "jtbd": "Pick a dough hydration.", "sources": []})
    folder.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    source = {"type": "web", "url": "https://example.org/guide", "kind": "principle"}
    apply_change_set(folder.path, plan_source_add(folder.path, source))
    folder.synthesis_path.write_text(SYNTHESIS, encoding="utf-8")
    (folder.corpus_path / "working").mkdir(exist_ok=True)
    (folder.corpus_path / "working/04-quality-checks.md").write_text(QUALITY, encoding="utf-8")
    folder.mixtape_path.write_text("# Published MIXTAPE\n", encoding="utf-8")
    (folder.corpus_path / "sources").mkdir(exist_ok=True)
    (folder.corpus_path / "sources/guide.md").write_text("Guide body\n", encoding="utf-8")
    review = plan_synthesis_review(folder.path, "still_current")
    apply_change_set(folder.path, review, approved=True)
    mark_corpus_ready(folder)
    return folder


def test_render_uses_the_corpus(project: ProjectFolder) -> None:
    draft = render_operating_layer(project)

    assert draft.skill_name == "liner-sourdough-hydration"
    assert "> Pick a dough hydration." in draft.liner_content
    assert "- Operating thesis: Hydration follows the flour.\n" in draft.liner_content
    assert "- Capability pattern: Reference translation.\n" in draft.liner_content
    assert "- Start from absorption.\n" in draft.liner_content
    assert "Not a top-level rule" not in draft.liner_content
    assert "1 usable compiled source file(s)" in draft.liner_content
    assert draft.skill_content.startswith("---\nname: liner-sourdough-hydration\n")
    assert "<!-- liner-maintenance-routing:start v1 -->" in draft.skill_content


def test_answers_lead_with_the_advice_and_keep_sources_at_the_end(project: ProjectFolder) -> None:
    draft = render_operating_layer(project)

    assert "## Answer Style\n" in draft.liner_content
    assert "at most three" in draft.liner_content
    assert "short note at the end" in draft.liner_content
    assert "label observations" not in draft.liner_content
    assert "Answer Style in `LINER.md`" in draft.skill_content


def test_drafts_come_with_a_separate_list_of_gap_questions(project: ProjectFolder) -> None:
    liner = render_operating_layer(project).liner_content
    style = liner[liner.index("## Answer Style") :]
    style = style[: style.index("\n## ", 1)]

    assert "When the task is a draft, deliver two parts" in style
    assert "name the gap it fills and how the answer will change the next version" in style
    assert "put the draft and the questions in two files" in style


def test_text_copied_from_the_synthesis_drops_footnote_marks(project: ProjectFolder) -> None:
    text = project.synthesis_path.read_text(encoding="utf-8")
    text = text.replace("Hydration follows the flour.", "Hydration follows the flour [^flour][^b].")
    text = text.replace("- Start from absorption.", "- Start from absorption.[^flour]")
    project.synthesis_path.write_text(text, encoding="utf-8")

    draft = render_operating_layer(project)

    for content in (draft.liner_content, draft.skill_content):
        assert "[^" not in content
    assert "Hydration follows the flour." in draft.liner_content
    assert "- Start from absorption.\n" in draft.liner_content


def test_answer_style_defers_to_abstention_and_a_fuller_intake(project: ProjectFolder) -> None:
    liner = render_operating_layer(project).liner_content
    style = liner[liner.index("## Answer Style") :]
    style = style[: style.index("\n## ", 1)]

    assert liner.count("## Answer Style") == 1
    assert "unless this liner's runtime boundaries or the person ask for a fuller intake" in style
    assert "lead with what's missing and who must decide" in style
    assert (
        "explicit unknowns and verification needs"
        not in render_operating_layer(project).skill_content
    )


def test_skill_name_keeps_a_single_prefix() -> None:
    assert project_skill_name({"title": "Liner Basics"}) == "liner-basics"
    assert project_skill_name({}) == "liner-project"


def test_plan_writes_nothing_and_needs_approval(project: ProjectFolder) -> None:
    change_set = plan_operating_layer_create(project.path)

    assert change_set.risk == "semantic"
    assert change_set.approval_required is True
    assert not project.liner_path.exists()
    assert not (project.path / "SKILL.md").exists()


def test_apply_creates_the_layer_and_completes_the_project(project: ProjectFolder) -> None:
    identity = read_liner_metadata(project)["id"]
    change_set = plan_operating_layer_create(project.path)

    apply_change_set(project.path, change_set, approved=True)

    draft = change_set.operations[0]
    assert project.liner_path.read_text(encoding="utf-8") == draft["liner_content"]
    assert (project.path / "SKILL.md").read_text(encoding="utf-8") == draft["skill_content"]
    metadata = read_liner_metadata(project)
    assert metadata["id"] == identity
    assert metadata["project_skill"] == {
        "status": "active",
        "name": "liner-sourdough-hydration",
        "path": "SKILL.md",
    }
    snapshot = inspect_project(project.path)
    assert snapshot.lifecycle["milestone"] == "project_complete"
    assert snapshot.lifecycle["operating_layer"]["state"] == "ready"


def test_apply_without_approval_writes_nothing(project: ProjectFolder) -> None:
    with pytest.raises(ProjectApplyError):
        apply_change_set(project.path, plan_operating_layer_create(project.path))

    assert not project.liner_path.exists()


def test_replay_returns_the_same_receipt(project: ProjectFolder) -> None:
    change_set = plan_operating_layer_create(project.path)

    first = apply_change_set(project.path, change_set, approved=True)
    second = apply_change_set(project.path, change_set, approved=True)

    assert second.receipt_id == first.receipt_id
    assert second.replayed is True


def test_legacy_drafts_are_removed_and_listed(project: ProjectFolder) -> None:
    draft = project.path / "working/LINER-draft.md"
    draft.parent.mkdir(exist_ok=True)
    draft.write_text("old\n", encoding="utf-8")
    legacy_skill = project.path / "skills/liner-sourdough-hydration.md"
    legacy_skill.parent.mkdir()
    legacy_skill.write_text("old\n", encoding="utf-8")

    change_set = plan_operating_layer_create(project.path)

    assert change_set.file_effects["delete"] == [
        "working/LINER-draft.md",
        "skills/liner-sourdough-hydration.md",
    ]
    apply_change_set(project.path, change_set, approved=True)
    assert not draft.exists()
    assert not legacy_skill.parent.exists()


def test_existing_liner_md_is_refused(project: ProjectFolder) -> None:
    project.liner_path.write_text("# Reviewed by hand\n", encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="operating_layer.review"):
        plan_operating_layer_create(project.path)


def test_unfinished_corpus_is_refused(tmp_path: Path) -> None:
    project = init_project(tmp_path / "unfinished")

    with pytest.raises(ProjectInspectionError, match="compiled corpus"):
        plan_operating_layer_create(project.path)


def test_corpus_change_after_planning_is_refused(project: ProjectFolder) -> None:
    change_set = plan_operating_layer_create(project.path)
    project.synthesis_path.write_text(SYNTHESIS + "\n- Late edit.\n", encoding="utf-8")

    with pytest.raises(ProjectApplyError):
        apply_change_set(project.path, change_set, approved=True)

    assert not project.liner_path.exists()


def _forged(project: ProjectFolder, changes: dict[str, object]) -> ProjectChangeSet:
    payload = plan_operating_layer_create(project.path).to_dict()
    payload.pop("change_set_hash")
    operation = payload["operations"][0]
    operation.update(changes)
    operation["proposed_liner_hash"] = _text_hash(operation["liner_content"])
    payload["change_set_id"] = _stable_uuid(
        f"change-set:{payload['project_id']}:{payload['expected_revision']}:"
        f"{_payload_hash(operation)}"
    )
    payload["file_effects"]["write"] = ["liner.yaml", "LINER.md", operation["skill_path"]]
    payload["file_effects"]["delete"] = operation["legacy_paths"]
    return ProjectChangeSet.from_dict({**payload, "change_set_hash": _payload_hash(payload)})


@pytest.mark.parametrize(
    "changes",
    [
        {"liner_content": "# Forged\n"},
        {"legacy_paths": ["mixtape/MIXTAPE.md"]},
        {"skill_path": "mixtape/MIXTAPE.md"},
        {"skill_name": "liner-other"},
    ],
)
def test_a_consistent_forged_plan_is_refused(
    project: ProjectFolder, changes: dict[str, object]
) -> None:
    mixtape = project.mixtape_path.read_text(encoding="utf-8")

    with pytest.raises(ProjectApplyError, match="corpus changed"):
        apply_change_set(project.path, _forged(project, changes), approved=True)

    assert not project.liner_path.exists()
    assert project.mixtape_path.read_text(encoding="utf-8") == mixtape


@pytest.mark.parametrize(
    "changes", [{"legacy_paths": "SKILL.md"}, {"legacy_paths": None}, {"skill_name": None}]
)
def test_a_malformed_plan_gets_a_clean_error(
    project: ProjectFolder, changes: dict[str, object]
) -> None:
    payload = plan_operating_layer_create(project.path).to_dict()
    payload.pop("change_set_hash")
    payload["operations"][0].update(changes)
    change_set = ProjectChangeSet.from_dict({**payload, "change_set_hash": _payload_hash(payload)})

    with pytest.raises((ProjectApplyError, ProjectInspectionError), match="operating_layer.create"):
        apply_change_set(project.path, change_set, approved=True)


def test_an_empty_skills_folder_is_left_alone(project: ProjectFolder) -> None:
    (project.path / "skills").mkdir()

    apply_change_set(project.path, plan_operating_layer_create(project.path), approved=True)

    assert (project.path / "skills").is_dir()


def test_older_project_gets_ids_in_the_same_approved_change(project: ProjectFolder) -> None:
    metadata = read_liner_metadata(project)
    metadata.pop("id")
    project.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["sources"][0].pop("id")
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    change_set = plan_operating_layer_create(project.path)

    assert change_set.risk == "structural"
    assert [item["type"] for item in change_set.operations] == [
        "identity.assign_project",
        "identity.assign_source",
        "operating_layer.create",
    ]
    apply_change_set(project.path, change_set, approved=True)
    assert project.liner_path.is_file()
    assert read_liner_metadata(project)["id"] == change_set.project_id
    assert inspect_project(project.path).sources[0].source_id is not None


def test_cli_plans_the_operation(project: ProjectFolder) -> None:
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "operating_layer.create"},
    }
    result = CliRunner().invoke(
        app,
        ["project", "plan", str(project.path), "--request-json", json.dumps(request), "--json"],
    )

    assert result.exit_code == 0, result.stdout
    assert json.loads(result.stdout)["operations"][0]["type"] == "operating_layer.create"
