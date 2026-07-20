from __future__ import annotations

import hashlib
import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

import liner.maintenance as maintenance_module
from liner.cli import app
from liner.maintenance import (
    ProjectApplyError,
    ProjectChangeSet,
    ProjectInspectionError,
    apply_change_set,
    inspect_project,
    maintenance_guidance,
    plan_project_guidance_upgrade,
)
from liner.project import init_project, read_liner_metadata

runner = CliRunner()


def _write_operating_layer(project: object, skill: str | None) -> None:
    project.liner_path.write_text("# User Operating Layer\n\nKeep this exact.\n", encoding="utf-8")
    metadata = read_liner_metadata(project)
    metadata["status"] = {
        "milestone": "project_complete",
        "stale": False,
        "operating_layer": {"state": "ready", "evidence": "LINER.md"},
    }
    if skill is None:
        metadata["project_skill"] = {"status": "missing"}
    else:
        (project.path / "SKILL.md").write_text(skill, encoding="utf-8")
        metadata["project_skill"] = {
            "status": "active",
            "name": "liner-demo",
            "path": "SKILL.md",
        }
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )


def _legacy_skill() -> str:
    return """---
name: liner-demo
description: 'Use this project for its documented job.'
custom: keep-me
---

# liner-demo

## User Section

Keep this exact.
"""


def test_guidance_json_and_markdown_are_equivalent_and_skill_sources_are_inert(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["sources"] = [
        {
            "id": "a1dd99f9-a930-4da7-a81b-7248fc7d3516",
            "type": "skill",
            "path": "terminal-ui",
            "note": "Reference evidence only.",
        }
    ]
    project.tape_path.write_text(yaml.safe_dump(tape, sort_keys=False), encoding="utf-8")
    _write_operating_layer(project, _legacy_skill())

    payload = maintenance_guidance(project.path).to_dict()
    json_result = runner.invoke(app, ["project", "guidance", str(project.path), "--format", "json"])
    markdown_result = runner.invoke(
        app, ["project", "guidance", str(project.path), "--format", "markdown"]
    )

    assert json_result.exit_code == 0, json_result.output
    assert json.loads(json_result.stdout) == payload
    assert markdown_result.exit_code == 0, markdown_result.output
    for value in (
        payload["contract"],
        str(payload["version"]),
        payload["guidance_state"],
        "liner project inspect",
        "liner project plan",
        "liner project apply",
        "terminal-ui",
        "evidence_only",
        "Never edit liner.yaml or tape.yaml directly",
        "--approve",
        "--approved-destination",
    ):
        assert value in markdown_result.stdout
    assert payload["instruction_allowlist"]["skill_sources"][0]["role"] == "evidence_only"
    assert payload["instruction_allowlist"]["skill_sources"][0]["active"] is False
    inspected = inspect_project(project.path).to_dict()
    assert inspected["sources"][0]["role"] == "evidence_only"
    assert inspected["sources"][0]["active_instruction"] is False
    assert inspected["instruction_policy"]["skill_sources"][0]["active"] is False


@pytest.mark.parametrize(
    ("setup", "expected"),
    [
        ("none", "not_applicable"),
        ("missing", "missing"),
        ("legacy", "legacy"),
        ("current", "current"),
    ],
)
def test_guidance_reports_each_project_skill_state(
    tmp_path: Path,
    setup: str,
    expected: str,
) -> None:
    project = init_project(tmp_path / setup)
    if setup == "missing":
        _write_operating_layer(project, None)
    elif setup == "legacy":
        _write_operating_layer(project, _legacy_skill())
    elif setup == "current":
        _write_operating_layer(project, _legacy_skill())
        apply_change_set(
            project.path,
            plan_project_guidance_upgrade(project.path),
            approved=True,
        )

    assert maintenance_guidance(project.path).guidance_state == expected


def test_guidance_supports_legacy_project_without_liner_metadata(tmp_path: Path) -> None:
    project = init_project(tmp_path / "no-metadata")
    project.liner_metadata_path.unlink()

    guidance = maintenance_guidance(project.path).to_dict()

    assert guidance["project"]["compatibility_state"] == "identity_missing"
    assert guidance["compatibility"]["mutation_available"] is True
    assert guidance["compatibility"]["identity_migration_required"] is True


def test_guidance_supports_legacy_project_with_future_tape_and_no_metadata(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "no-metadata-future-tape")
    project.liner_metadata_path.unlink()
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["version"] = 99
    project.tape_path.write_text(yaml.safe_dump(tape, sort_keys=False), encoding="utf-8")

    guidance = maintenance_guidance(project.path).to_dict()

    assert guidance["project"]["compatibility_state"] == "incompatible_tape_format"
    assert guidance["project"]["tape_format_version"] == 99
    assert guidance["compatibility"]["mutation_available"] is False
    assert guidance["commands"][2]["available"] is False


def test_guidance_upgrade_is_reviewed_and_preserves_user_authored_content(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    _write_operating_layer(project, _legacy_skill())
    project.mixtape_path.write_text("# Curator-owned MIXTAPE\n", encoding="utf-8")
    liner_before = project.liner_path.read_bytes()
    mixtape_before = project.mixtape_path.read_bytes()

    change_set = plan_project_guidance_upgrade(project.path)
    planned = runner.invoke(
        app,
        [
            "project",
            "plan",
            str(project.path),
            "--request-json",
            json.dumps(
                {
                    "contract": "liner.maintenance_request",
                    "version": 1,
                    "operation": {"type": "project.guidance_upgrade"},
                }
            ),
            "--json",
        ],
    )

    assert change_set.risk == "semantic"
    assert change_set.approval_required is True
    assert change_set.file_effects["write"] == ["SKILL.md"]
    assert change_set.operations[0]["frontmatter_updates"] == ["description"]
    assert change_set.operations[0]["managed_section"] == "Maintenance Routing"
    assert planned.exit_code == 0, planned.output
    assert json.loads(planned.stdout)["change_set_id"] == change_set.change_set_id
    with pytest.raises(ProjectApplyError) as refused:
        apply_change_set(project.path, change_set)
    assert refused.value.report.code == "approval_required"

    receipt = apply_change_set(project.path, change_set, approved=True)
    upgraded = (project.path / "SKILL.md").read_text(encoding="utf-8")

    assert "Use or maintain this Liner Project" in upgraded
    assert "custom: keep-me" in upgraded
    assert "## User Section\n\nKeep this exact." in upgraded
    assert "<!-- liner-maintenance-routing:start v1 -->" in upgraded
    assert "liner project guidance --format markdown" in upgraded
    assert project.liner_path.read_bytes() == liner_before
    assert project.mixtape_path.read_bytes() == mixtape_before
    assert receipt.operations[0]["skill_path"] == "SKILL.md"
    assert maintenance_guidance(project.path).guidance_state == "current"


@pytest.mark.parametrize("invalid_version", [True, 1.0])
def test_maintenance_contract_versions_require_exact_integers(
    tmp_path: Path,
    invalid_version: object,
) -> None:
    project = init_project(tmp_path / f"contract-{invalid_version}")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)
    payload = change_set.to_dict()
    payload["version"] = invalid_version
    unsigned = dict(payload)
    unsigned.pop("change_set_hash")
    encoded = json.dumps(unsigned, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    payload["change_set_hash"] = hashlib.sha256(encoded.encode("utf-8")).hexdigest()

    with pytest.raises(ProjectInspectionError, match="Unsupported Change Set version"):
        ProjectChangeSet.from_dict(payload)

    request = runner.invoke(
        app,
        [
            "project",
            "plan",
            str(project.path),
            "--request-json",
            json.dumps(
                {
                    "contract": "liner.maintenance_request",
                    "version": invalid_version,
                    "operation": {"type": "project.guidance_upgrade"},
                }
            ),
            "--json",
        ],
    )
    assert request.exit_code == 1


def test_guidance_upgrade_lazily_assigns_missing_identity_without_staling_corpus(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "legacy")
    _write_operating_layer(project, _legacy_skill())
    metadata = read_liner_metadata(project)
    metadata.pop("id")
    metadata["custom"] = "keep-metadata"
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["custom"] = "keep-tape"
    tape["sources"] = [{"type": "web", "url": "https://example.com"}]
    project.tape_path.write_text(yaml.safe_dump(tape, sort_keys=False), encoding="utf-8")

    guidance = maintenance_guidance(project.path).to_dict()
    change_set = plan_project_guidance_upgrade(project.path)

    assert guidance["compatibility"]["mutation_available"] is True
    assert guidance["compatibility"]["identity_migration_required"] is True
    assert [operation["type"] for operation in change_set.operations] == [
        "identity.assign_project",
        "identity.assign_source",
        "project.guidance_upgrade",
    ]
    assert change_set.risk == "structural"
    assert change_set.file_effects["write"] == [
        "liner.yaml",
        "mixtape/tape.yaml",
        "SKILL.md",
    ]

    receipt = apply_change_set(project.path, change_set, approved=True)
    upgraded_metadata = yaml.safe_load(project.liner_metadata_path.read_text(encoding="utf-8"))
    upgraded_tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))

    assert upgraded_metadata["custom"] == "keep-metadata"
    assert upgraded_tape["custom"] == "keep-tape"
    assert upgraded_metadata["status"]["stale"] is False
    assert upgraded_metadata["id"] == change_set.project_id
    assert upgraded_tape["sources"][0]["id"]
    assert receipt.synthesis_disposition == "unchanged"


def test_guidance_upgrade_hashes_declared_nested_project_skill(tmp_path: Path) -> None:
    project = init_project(tmp_path / "nested")
    _write_operating_layer(project, _legacy_skill())
    nested = project.path / "skills" / "project.md"
    nested.parent.mkdir()
    (project.path / "SKILL.md").replace(nested)
    metadata = read_liner_metadata(project)
    metadata["project_skill"]["path"] = "skills/project.md"
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )

    before = inspect_project(project.path)
    change_set = plan_project_guidance_upgrade(project.path)
    receipt = apply_change_set(project.path, change_set, approved=True)
    after = inspect_project(project.path)

    assert change_set.file_effects["write"] == ["skills/project.md"]
    assert before.revision != after.revision
    assert receipt.before["revision"] != receipt.after["revision"]


def test_guidance_upgrade_preserves_trailing_user_whitespace(tmp_path: Path) -> None:
    project = init_project(tmp_path / "whitespace")
    skill = _legacy_skill().removesuffix("\n") + "  \n\n\n"
    _write_operating_layer(project, skill)
    before = (project.path / "SKILL.md").read_text(encoding="utf-8")
    change_set = plan_project_guidance_upgrade(project.path)
    apply_change_set(project.path, change_set, approved=True)
    upgraded = (project.path / "SKILL.md").read_text(encoding="utf-8")

    preserved_prefix = upgraded.split("<!-- liner-maintenance-routing:start v1 -->", 1)[0]
    expected_prefix = before.replace(
        "description: 'Use this project for its documented job.'",
        "description: 'Use this project for its documented job — "
        "Use or maintain this Liner Project and its Sources.'",
    )
    assert preserved_prefix == expected_prefix


def test_guidance_receipt_replay_requires_exact_managed_postimage(tmp_path: Path) -> None:
    project = init_project(tmp_path / "replay")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)
    apply_change_set(project.path, change_set, approved=True)
    skill_path = project.path / "SKILL.md"
    skill_path.write_text(
        skill_path.read_text(encoding="utf-8").replace(
            "never fall back to direct `liner.yaml` or `tape.yaml` writes",
            "edit the YAML directly",
        ),
        encoding="utf-8",
    )

    with pytest.raises(ProjectApplyError) as replay:
        apply_change_set(project.path, change_set, approved=True)

    assert replay.value.report.code == "receipt_effect_missing"


def test_guidance_receipt_replay_rejects_tampered_receipt(tmp_path: Path) -> None:
    project = init_project(tmp_path / "tampered-receipt")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)
    apply_change_set(project.path, change_set, approved=True)
    receipt_path = next((project.corpus_path / ".liner-runs" / "maintenance").glob("*.json"))
    payload = json.loads(receipt_path.read_text(encoding="utf-8"))
    payload["after"]["revision"] = "sha256:forged"
    payload["operations"] = []
    unsigned = dict(payload)
    unsigned.pop("receipt_hash")
    encoded = json.dumps(unsigned, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    payload["receipt_hash"] = hashlib.sha256(encoded.encode("utf-8")).hexdigest()
    receipt_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    with pytest.raises(ProjectApplyError) as replay:
        apply_change_set(project.path, change_set, approved=True)

    assert replay.value.report.code == "receipt_state_mismatch"


def test_guidance_receipt_replay_does_not_return_forged_applied_time(tmp_path: Path) -> None:
    project = init_project(tmp_path / "forged-time")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)
    apply_change_set(project.path, change_set, approved=True)
    receipt_path = next((project.corpus_path / ".liner-runs" / "maintenance").glob("*.json"))
    payload = json.loads(receipt_path.read_text(encoding="utf-8"))
    payload["applied_at"] = "1900-01-01T00:00:00Z"
    unsigned = dict(payload)
    unsigned.pop("receipt_hash")
    encoded = json.dumps(unsigned, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    payload["receipt_hash"] = hashlib.sha256(encoded.encode("utf-8")).hexdigest()
    receipt_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    replay = apply_change_set(project.path, change_set, approved=True)

    assert replay.replayed is True
    assert replay.applied_at != "1900-01-01T00:00:00Z"


def test_guidance_receipt_replay_matches_coarse_filesystem_time(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    original_utime = maintenance_module.os.utime

    def coarse_utime(
        path: object,
        *,
        ns: tuple[int, int],
        follow_symlinks: bool = True,
    ) -> None:
        rounded = ns[1] // 2_000_000_000 * 2_000_000_000
        original_utime(path, ns=(ns[0], rounded), follow_symlinks=follow_symlinks)

    monkeypatch.setattr(maintenance_module.os, "utime", coarse_utime)
    monkeypatch.setattr(
        maintenance_module.os,
        "supports_follow_symlinks",
        {*maintenance_module.os.supports_follow_symlinks, coarse_utime},
    )
    project = init_project(tmp_path / "coarse-time")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)

    first = apply_change_set(project.path, change_set, approved=True)
    receipt_path = next((project.corpus_path / ".liner-runs" / "maintenance").glob("*.json"))
    stored = json.loads(receipt_path.read_text(encoding="utf-8"))
    replay = apply_change_set(project.path, change_set, approved=True)

    assert first.applied_at == stored["applied_at"] == replay.applied_at


def test_guidance_receipt_replay_without_no_follow_utime(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr(
        maintenance_module.os,
        "supports_follow_symlinks",
        maintenance_module.os.supports_follow_symlinks - {maintenance_module.os.utime},
    )
    project = init_project(tmp_path / "unsupported-no-follow-utime")
    _write_operating_layer(project, _legacy_skill())
    change_set = plan_project_guidance_upgrade(project.path)

    first = apply_change_set(project.path, change_set, approved=True)
    replay = apply_change_set(project.path, change_set, approved=True)

    assert first.applied_at == replay.applied_at
    assert replay.replayed is True


def test_receipt_reader_rejects_path_swap_after_open(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    receipt = tmp_path / "receipt.json"
    displaced = tmp_path / "displaced.json"
    replacement = tmp_path / "replacement.json"
    receipt.write_text("{}\n", encoding="utf-8")
    replacement.write_text("{}\n", encoding="utf-8")
    original_open = maintenance_module.os.open
    swapped = False

    def swapping_open(path: object, flags: int, mode: int = 0o777) -> int:
        nonlocal swapped
        descriptor = original_open(path, flags, mode)
        if Path(path) == receipt and not swapped:
            swapped = True
            receipt.replace(displaced)
            replacement.replace(receipt)
        return descriptor

    monkeypatch.setattr(maintenance_module.os, "open", swapping_open)

    with pytest.raises(OSError, match="changed"):
        maintenance_module._read_receipt_no_follow(receipt)


@pytest.mark.parametrize(
    "description_lines",
    [
        "description: >\n  Use this project for its documented job.",
        "description: |\n  Use this project for its documented job.",
        "description: &trigger 'Use this project for its documented job.'",
        "name: &trigger liner-demo\ndescription: *trigger",
    ],
)
def test_guidance_upgrade_rejects_non_preserving_yaml_description_forms(
    tmp_path: Path,
    description_lines: str,
) -> None:
    project = init_project(tmp_path / "yaml-form")
    skill = f"""---
name: liner-demo
{description_lines}
custom: keep-me
---

# liner-demo
"""
    _write_operating_layer(project, skill)
    before = (project.path / "SKILL.md").read_bytes()

    with pytest.raises(ProjectInspectionError, match="single-line YAML scalar"):
        plan_project_guidance_upgrade(project.path)

    assert (project.path / "SKILL.md").read_bytes() == before


def test_guidance_upgrade_rejects_inline_description_comment(tmp_path: Path) -> None:
    project = init_project(tmp_path / "inline-comment")
    skill = _legacy_skill().replace(
        "description: 'Use this project for its documented job.'",
        "description: 'Use this project for its documented job.' # keep this",
    )
    _write_operating_layer(project, skill)

    with pytest.raises(ProjectInspectionError, match="inline YAML comment"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_upgrade_rejects_reversed_managed_markers(tmp_path: Path) -> None:
    project = init_project(tmp_path / "reversed")
    skill = (
        _legacy_skill()
        + "\n<!-- liner-maintenance-routing:end -->\n"
        + "<!-- liner-maintenance-routing:start v1 -->\n"
    )
    _write_operating_layer(project, skill)

    with pytest.raises(ProjectInspectionError, match="reversed managed"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_upgrade_ignores_marker_text_inside_frontmatter(tmp_path: Path) -> None:
    project = init_project(tmp_path / "frontmatter-markers")
    skill = _legacy_skill().replace(
        "Use this project for its documented job.",
        "Use <!-- liner-maintenance-routing:start v1 --> and "
        "<!-- liner-maintenance-routing:end --> as literal examples.",
    )
    _write_operating_layer(project, skill)

    change_set = plan_project_guidance_upgrade(project.path)
    apply_change_set(project.path, change_set, approved=True)
    upgraded = (project.path / "SKILL.md").read_text(encoding="utf-8")
    closing = upgraded.find("\n---\n", 4)

    assert isinstance(yaml.safe_load(upgraded[4:closing]), dict)
    assert upgraded[closing:].count("<!-- liner-maintenance-routing:start v1 -->") == 1
    assert maintenance_guidance(project.path).guidance_state == "current"


def test_guidance_upgrade_rejects_declared_skill_symlink_at_plan(tmp_path: Path) -> None:
    project = init_project(tmp_path / "symlink")
    _write_operating_layer(project, _legacy_skill())
    skill_path = project.path / "SKILL.md"
    real_path = project.path / "REAL-SKILL.md"
    skill_path.replace(real_path)
    skill_path.symlink_to(real_path.name)

    with pytest.raises(ProjectInspectionError, match="symbolic link"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_upgrade_rejects_symlinked_skill_parent_at_plan(tmp_path: Path) -> None:
    project = init_project(tmp_path / "symlink-parent")
    _write_operating_layer(project, _legacy_skill())
    real_dir = project.path / "real-skills"
    real_dir.mkdir()
    (project.path / "SKILL.md").replace(real_dir / "project.md")
    (project.path / "skills").symlink_to(real_dir.name, target_is_directory=True)
    metadata = read_liner_metadata(project)
    metadata["project_skill"]["path"] = "skills/project.md"
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )

    with pytest.raises(ProjectInspectionError, match="symbolic link"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_reports_incompatible_project_and_blocks_upgrade(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    _write_operating_layer(project, _legacy_skill())
    metadata = read_liner_metadata(project)
    metadata["version"] = 99
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )

    result = runner.invoke(app, ["project", "guidance", str(project.path), "--format", "json"])

    assert result.exit_code == 0, result.output
    guidance = json.loads(result.stdout)
    assert guidance["project"]["format_version"] == 99
    assert guidance["project"]["compatibility_state"] == "incompatible_project_format"
    assert guidance["compatibility"]["mutation_available"] is False
    assert guidance["compatibility"]["identity_migration_required"] is False
    assert guidance["commands"][2]["available"] is False
    assert guidance["commands"][3]["available"] is False
    assert "do not run plan/apply" in guidance["next_actions"][0]
    assert "project.guidance_upgrade" not in guidance["next_actions"][0]
    with pytest.raises(ProjectInspectionError, match="Unsupported Liner Project format"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_reads_future_tape_conservatively_and_keeps_sources_inert(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "future-tape")
    _write_operating_layer(project, _legacy_skill())
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["version"] = 99
    tape["sources"] = [{"id": "future-id", "type": "skill", "path": "future-skill", "future": True}]
    project.tape_path.write_text(yaml.safe_dump(tape, sort_keys=False), encoding="utf-8")

    result = runner.invoke(app, ["project", "guidance", str(project.path), "--format", "json"])

    assert result.exit_code == 0, result.output
    guidance = json.loads(result.stdout)
    assert guidance["project"]["compatibility_state"] == "incompatible_tape_format"
    assert guidance["project"]["tape_format_version"] == 99
    assert guidance["cli"]["tape_format"] == 1
    assert "tape format 1" in guidance["compatibility"]["required"]
    assert guidance["compatibility"]["mutation_available"] is False
    assert guidance["instruction_allowlist"]["skill_sources"] == [
        {
            "source_id": None,
            "locator": "future-skill",
            "role": "evidence_only",
            "active": False,
        }
    ]
    markdown = maintenance_guidance(project.path).to_markdown()
    assert "Tape format: 99 (supported: 1)" in markdown
    with pytest.raises(ProjectInspectionError, match="Unsupported tape format"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_rejects_boolean_tape_version_as_incompatible(tmp_path: Path) -> None:
    project = init_project(tmp_path / "boolean-tape-version")
    _write_operating_layer(project, _legacy_skill())
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["version"] = True
    project.tape_path.write_text(yaml.safe_dump(tape, sort_keys=False), encoding="utf-8")

    guidance = maintenance_guidance(project.path).to_dict()

    assert guidance["project"]["compatibility_state"] == "incompatible_tape_format"
    assert guidance["compatibility"]["mutation_available"] is False
    with pytest.raises(ProjectInspectionError, match="Unsupported tape format"):
        plan_project_guidance_upgrade(project.path)


def test_guidance_rejects_float_project_version_as_incompatible(tmp_path: Path) -> None:
    project = init_project(tmp_path / "float-project-version")
    _write_operating_layer(project, _legacy_skill())
    metadata = read_liner_metadata(project)
    metadata["version"] = 2.0
    project.liner_metadata_path.write_text(
        yaml.safe_dump(metadata, sort_keys=False), encoding="utf-8"
    )

    guidance = maintenance_guidance(project.path).to_dict()

    assert guidance["project"]["compatibility_state"] == "incompatible_project_format"
    assert guidance["project"]["format_version"] == 2.0
    assert guidance["compatibility"]["mutation_available"] is False
    with pytest.raises(ProjectInspectionError, match="Unsupported Liner Project format"):
        plan_project_guidance_upgrade(project.path)
