from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
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
    plan_project_move,
    plan_project_rename,
    plan_source_add,
    plan_source_add_batch,
    plan_source_purge,
    plan_source_remove,
    plan_source_replace,
    plan_source_update,
)
from liner.project import init_project, read_liner_metadata

runner = CliRunner()


def test_project_rename_changes_only_managed_display_metadata(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    project.liner_path.write_text("# User-authored operating layer\n", encoding="utf-8")
    skill_path = project.path / "SKILL.md"
    skill_path.write_text("user-owned pointer to /old/path\n", encoding="utf-8")
    tape_before = project.tape_path.read_bytes()
    liner_before = project.liner_path.read_bytes()
    skill_before = skill_path.read_bytes()
    project_id = read_liner_metadata(project)["id"]

    change_set = plan_project_rename(project.path, "Renamed Project")

    assert change_set.risk == "metadata"
    assert change_set.approval_required is False
    assert change_set.operations == (
        {
            "type": "project.rename",
            "old_name": "demo",
            "new_name": "Renamed Project",
            "managed_reference_updates": ["liner.yaml:name"],
        },
    )
    assert change_set.file_effects["write"] == ["liner.yaml"]
    assert project.tape_path.read_bytes() == tape_before

    receipt = apply_change_set(project.path, change_set)

    assert read_liner_metadata(project)["id"] == project_id
    assert read_liner_metadata(project)["name"] == "Renamed Project"
    assert project.tape_path.read_bytes() == tape_before
    assert project.liner_path.read_bytes() == liner_before
    assert skill_path.read_bytes() == skill_before
    assert receipt.operations[0]["old_name"] == "demo"
    assert receipt.operations[0]["new_name"] == "Renamed Project"
    assert receipt.synthesis_disposition == "unchanged"


def test_project_rename_refuses_stale_plan(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_project_rename(project.path, "Renamed Project")
    project.synthesis_path.write_text("changed after planning\n", encoding="utf-8")
    before = _tree_bytes(project.path)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "stale_project"
    assert _tree_bytes(project.path) == before


def test_project_move_is_previewed_atomic_and_rediscoverable(tmp_path: Path) -> None:
    project = init_project(tmp_path / "old-root")
    project_id = read_liner_metadata(project)["id"]
    pointer = project.path / "AGENTS.md"
    pointer.write_text("user-owned pointer to /old-root\n", encoding="utf-8")
    destination = tmp_path / "new-root"
    before = _tree_bytes(project.path)

    change_set = plan_project_move(project.path, destination)

    assert change_set.risk == "structural"
    assert change_set.approval_required is True
    assert change_set.operations[0]["type"] == "project.move"
    assert change_set.operations[0]["old_root"] == str(project.path.resolve())
    assert change_set.operations[0]["new_root"] == str(destination.resolve())
    assert change_set.operations[0]["managed_reference_updates"] == ["liner.yaml:name"]
    assert change_set.operations[0]["display_name"] == "old-root"
    assert change_set.file_effects["write"] == ["liner.yaml"]
    assert change_set.file_effects["move"] == [
        f"{project.path.resolve()} -> {destination.resolve()}"
    ]
    assert _tree_bytes(project.path) == before

    with pytest.raises(ProjectApplyError) as refused:
        apply_change_set(project.path, change_set)
    assert refused.value.report.code == "approval_required"

    receipt = apply_change_set(
        project.path,
        change_set,
        approved=True,
        approved_destination=destination,
    )

    assert not project.path.exists()
    moved_snapshot = inspect_project(destination, expected_project_id=project_id)
    assert moved_snapshot.root == destination.resolve()
    assert moved_snapshot.name == "old-root"
    assert (destination / "AGENTS.md").read_text(encoding="utf-8") == (
        "user-owned pointer to /old-root\n"
    )
    assert receipt.project_id == project_id
    assert receipt.operations[0]["old_root"] == str(project.path.resolve())
    assert receipt.operations[0]["new_root"] == str(destination.resolve())
    child = destination / "mixtape" / "working"
    assert inspect_project(child, expected_project_id=project_id).root == destination.resolve()
    replay = apply_change_set(
        destination,
        change_set,
        approved=True,
        approved_destination=destination,
    )
    assert replay.receipt_id == receipt.receipt_id
    assert replay.replayed is True


def test_project_move_rejects_collision_nested_root_and_cross_device(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "old-root")
    collision = tmp_path / "collision"
    collision.mkdir()
    with pytest.raises(ProjectInspectionError, match="already exists"):
        plan_project_move(project.path, collision)
    with pytest.raises(ProjectInspectionError, match="nested"):
        plan_project_move(project.path, project.path / "nested")
    enclosing = init_project(tmp_path / "enclosing")
    with pytest.raises(ProjectInspectionError, match="nest inside"):
        plan_project_move(project.path, enclosing.path / "nested-project")

    original_device = maintenance_module._path_device

    def different_device(path: Path) -> int:
        return original_device(path) + (1 if path == tmp_path.resolve() else 0)

    monkeypatch.setattr(maintenance_module, "_path_device", different_device)
    with pytest.raises(ProjectInspectionError, match="cross-device"):
        plan_project_move(project.path, tmp_path / "other-root")


def test_project_move_rejects_dangling_destination_symlink(tmp_path: Path) -> None:
    project = init_project(tmp_path / "old-root")
    destination = tmp_path / "requested-root"
    destination.symlink_to(tmp_path / "redirected-target", target_is_directory=True)

    with pytest.raises(ProjectInspectionError, match="symbolic link"):
        plan_project_move(project.path, destination)

    assert destination.is_symlink()
    assert not (tmp_path / "redirected-target").exists()


def test_project_move_refuses_stale_destination_without_touching_original(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "old-root")
    destination = tmp_path / "new-root"
    change_set = plan_project_move(project.path, destination)
    destination.mkdir()
    (destination / "owned.txt").write_text("keep\n", encoding="utf-8")
    before = _tree_bytes(project.path)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "unsafe_project"
    assert _tree_bytes(project.path) == before
    assert (destination / "owned.txt").read_text(encoding="utf-8") == "keep\n"


def test_project_move_activation_failure_rolls_back_original(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "old-root")
    destination = tmp_path / "new-root"
    change_set = plan_project_move(project.path, destination)
    before = _tree_bytes(project.path)

    def fail_move(_source: Path, _destination: Path) -> None:
        raise OSError("injected move failure")

    monkeypatch.setattr(maintenance_module, "_rename_project_root_noreplace", fail_move)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "apply_failed"
    assert _tree_bytes(project.path) == before
    assert not destination.exists()


def test_project_move_preserves_post_activation_edits_and_reused_old_root(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "old-root")
    user_file = project.path / "user.txt"
    user_file.write_text("before\n", encoding="utf-8")
    destination = tmp_path / "new-root"
    change_set = plan_project_move(project.path, destination)
    original_move = maintenance_module._rename_project_root_noreplace

    def move_then_edit_and_reuse(source: Path, target: Path) -> None:
        original_move(source, target)
        (target / "user.txt").write_text("post-activation edit\n", encoding="utf-8")
        source.mkdir()
        (source / "unrelated.txt").write_text("do not delete\n", encoding="utf-8")
        raise OSError("injected post-activation failure")

    monkeypatch.setattr(
        maintenance_module,
        "_rename_project_root_noreplace",
        move_then_edit_and_reuse,
    )

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "activation_uncertain"
    assert raised.value.report.partial_success is True
    assert (destination / "user.txt").read_text(encoding="utf-8") == ("post-activation edit\n")
    assert (project.path / "unrelated.txt").read_text(encoding="utf-8") == ("do not delete\n")


def test_project_move_rolls_back_after_final_destination_collision(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "old-root")
    before = _tree_bytes(project.path)
    destination = tmp_path / "new-root"
    change_set = plan_project_move(project.path, destination)

    def collide_without_moving(_source: Path, target: Path) -> None:
        target.mkdir()
        (target / "unrelated.txt").write_text("keep\n", encoding="utf-8")
        raise FileExistsError("injected final collision")

    monkeypatch.setattr(
        maintenance_module,
        "_rename_project_root_noreplace",
        collide_without_moving,
    )

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "apply_failed"
    assert _tree_bytes(project.path) == before
    assert (destination / "unrelated.txt").read_text(encoding="utf-8") == "keep\n"


def test_project_move_rederives_destination_and_file_effects(tmp_path: Path) -> None:
    project = init_project(tmp_path / "old-root")
    destination = tmp_path / "new-root"
    payload = plan_project_move(project.path, destination).to_dict()
    operation = dict(payload["operations"][0])
    operation["new_root"] = str(tmp_path / "redirected")
    payload["operations"] = [operation]
    payload["file_effects"]["move"] = [
        f"{project.path.resolve()} -> {(tmp_path / 'redirected').resolve()}"
    ]
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            crafted,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "destination_approval_mismatch"
    assert project.path.exists()
    assert not (tmp_path / "redirected").exists()


def test_project_move_preserves_an_existing_explicit_name_without_rewriting_metadata(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "old-root")
    rename = plan_project_rename(project.path, "Stable Display Name")
    apply_change_set(project.path, rename)
    metadata_before = project.liner_metadata_path.read_bytes()
    destination = tmp_path / "new-root"

    move = plan_project_move(project.path, destination)

    assert move.operations[0]["managed_reference_updates"] == []
    assert move.file_effects["write"] == []
    apply_change_set(
        project.path,
        move,
        approved=True,
        approved_destination=destination,
    )
    assert (destination / "liner.yaml").read_bytes() == metadata_before
    assert inspect_project(destination).name == "Stable Display Name"


@pytest.mark.parametrize("stored_name", ["   ", "  Display Name  "])
def test_project_move_materializes_noncanonical_names_and_replays(
    tmp_path: Path,
    stored_name: str,
) -> None:
    project = init_project(tmp_path / "old-root")
    metadata = read_liner_metadata(project)
    metadata["name"] = stored_name
    project.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    destination = tmp_path / "new-root"

    move = plan_project_move(project.path, destination)
    expected_name = "old-root" if not stored_name.strip() else "Display Name"

    assert move.operations[0]["display_name"] == expected_name
    assert move.operations[0]["managed_reference_updates"] == ["liner.yaml:name"]
    first = apply_change_set(
        project.path,
        move,
        approved=True,
        approved_destination=destination,
    )
    assert read_liner_metadata(destination)["name"] == expected_name
    assert inspect_project(destination).name == expected_name
    replay = apply_change_set(
        destination,
        move,
        approved=True,
        approved_destination=destination,
    )
    assert replay.receipt_id == first.receipt_id
    assert replay.replayed is True


@pytest.mark.parametrize("operation", ["rename", "move"])
def test_project_root_mutations_preserve_user_owned_hardlinks(
    tmp_path: Path,
    operation: str,
) -> None:
    project = init_project(tmp_path / "old-root")
    first = project.path / "user-one.txt"
    second = project.path / "user-two.txt"
    first.write_text("shared inode\n", encoding="utf-8")
    second.hardlink_to(first)
    assert first.stat().st_ino == second.stat().st_ino

    if operation == "rename":
        change_set = plan_project_rename(project.path, "Display Name")
        apply_change_set(project.path, change_set)
        active_root = project.path
    else:
        active_root = tmp_path / "new-root"
        change_set = plan_project_move(project.path, active_root)
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=active_root,
        )

    assert (active_root / "user-one.txt").stat().st_ino == (
        active_root / "user-two.txt"
    ).stat().st_ino


def test_project_root_mutation_refuses_external_hardlinks(tmp_path: Path) -> None:
    project = init_project(tmp_path / "old-root")
    inside = project.path / "user.txt"
    outside = tmp_path / "outside.txt"
    inside.write_text("shared\n", encoding="utf-8")
    outside.hardlink_to(inside)

    with pytest.raises(ProjectInspectionError, match="hardlinks that leave"):
        plan_project_move(project.path, tmp_path / "new-root")

    assert inside.stat().st_ino == outside.stat().st_ino
    assert inside.read_text(encoding="utf-8") == "shared\n"


def test_project_apply_binds_external_hardlink_guard_to_activation_baseline(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "old-root")
    inside = project.path / "user.txt"
    outside = tmp_path / "outside.txt"
    inside.write_text("shared\n", encoding="utf-8")
    destination = tmp_path / "new-root"
    change_set = plan_project_move(project.path, destination)
    original_fingerprint = maintenance_module._activation_fingerprint
    calls = 0

    def fingerprint_then_link(root: Path) -> str:
        nonlocal calls
        calls += 1
        fingerprint = original_fingerprint(root)
        if calls == 1:
            outside.hardlink_to(inside)
        return fingerprint

    monkeypatch.setattr(
        maintenance_module,
        "_activation_fingerprint",
        fingerprint_then_link,
    )

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(
            project.path,
            change_set,
            approved=True,
            approved_destination=destination,
        )

    assert raised.value.report.code == "unsafe_project"
    assert inside.stat().st_ino == outside.stat().st_ino
    assert not destination.exists()
    assert list(tmp_path.glob(".old-root.liner-stage-*")) == []


def test_project_rename_refuses_hardlinked_managed_metadata(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    alias = project.path / "user-owned-alias.yaml"
    alias.hardlink_to(project.liner_metadata_path)
    before = alias.read_bytes()
    change_set = plan_project_rename(project.path, "Display Name")

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "unsafe_project"
    assert alias.read_bytes() == before
    assert project.liner_metadata_path.read_bytes() == before


def test_project_apply_detects_concurrent_permission_change_during_staging(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    user_file = project.path / "user.txt"
    user_file.write_text("keep\n", encoding="utf-8")
    user_file.chmod(0o644)
    change_set = plan_project_rename(project.path, "Display Name")
    original_copy = maintenance_module._copy_project_tree_preserving_hardlinks

    def copy_then_chmod(source: Path, destination: Path) -> None:
        original_copy(source, destination)
        user_file.chmod(0o600)

    monkeypatch.setattr(
        maintenance_module,
        "_copy_project_tree_preserving_hardlinks",
        copy_then_chmod,
    )

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "stale_project"
    assert user_file.stat().st_mode & 0o777 == 0o600


def test_project_apply_ignores_concurrent_file_mtime_churn_during_staging(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    user_file = project.path / "user.txt"
    user_file.write_text("keep\n", encoding="utf-8")
    change_set = plan_project_rename(project.path, "Display Name")
    original_copy = maintenance_module._copy_project_tree_preserving_hardlinks

    def copy_then_mutate(source: Path, destination: Path) -> None:
        original_copy(source, destination)
        prior = user_file.stat().st_mtime_ns
        user_file.touch()
        if user_file.stat().st_mtime_ns == prior:
            os.utime(user_file, ns=(prior + 1, prior + 1))

    monkeypatch.setattr(
        maintenance_module,
        "_copy_project_tree_preserving_hardlinks",
        copy_then_mutate,
    )

    receipt = apply_change_set(project.path, change_set)

    assert receipt.operations[0]["type"] == "project.rename"
    assert inspect_project(project.path).name == "Display Name"
    assert user_file.read_text(encoding="utf-8") == "keep\n"


def test_project_apply_ignores_concurrent_xattr_churn_during_staging(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    xattr_command = shutil.which("xattr")
    if xattr_command is None:
        pytest.skip("extended attributes are unavailable on this platform")
    project = init_project(tmp_path / "demo")
    user_file = project.path / "user.txt"
    user_file.write_text("keep\n", encoding="utf-8")
    change_set = plan_project_rename(project.path, "Display Name")
    original_copy = maintenance_module._copy_project_tree_preserving_hardlinks

    def copy_then_set_xattr(source: Path, destination: Path) -> None:
        original_copy(source, destination)
        result = subprocess.run(
            [xattr_command, "-w", "com.cmdux.liner-test", "volatile", str(user_file)],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.returncode != 0:
            pytest.skip(f"extended attributes cannot be set here: {result.stderr.strip()}")

    monkeypatch.setattr(
        maintenance_module,
        "_copy_project_tree_preserving_hardlinks",
        copy_then_set_xattr,
    )

    receipt = apply_change_set(project.path, change_set)

    assert receipt.operations[0]["type"] == "project.rename"
    assert inspect_project(project.path).name == "Display Name"
    assert user_file.read_text(encoding="utf-8") == "keep\n"


def test_project_apply_detects_concurrent_root_mode_change_during_staging(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_project_rename(project.path, "Display Name")
    original_copy = maintenance_module._copy_project_tree_preserving_hardlinks

    def copy_then_chmod(source: Path, destination: Path) -> None:
        original_copy(source, destination)
        source.chmod(0o700)

    monkeypatch.setattr(
        maintenance_module,
        "_copy_project_tree_preserving_hardlinks",
        copy_then_chmod,
    )

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "stale_project"


@pytest.mark.parametrize("name", [123, None, False])
def test_project_plan_rejects_non_string_rename_names(tmp_path: Path, name: object) -> None:
    project = init_project(tmp_path / "demo")
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "project.rename", "name": name},
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

    assert result.exit_code == 1
    assert "requires a string name" in result.output


def test_project_plan_reports_nul_move_destination_as_structured_error(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "project.move", "destination": "bad\u0000path"},
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

    assert result.exit_code == 1
    assert "Invalid Project move destination" in result.output


def test_project_plan_and_convenience_commands_compile_rename_and_move(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    rename_request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "project.rename", "name": "Display Name"},
    }
    planned = runner.invoke(
        app,
        [
            "project",
            "plan",
            str(project.path),
            "--request-json",
            json.dumps(rename_request),
            "--json",
        ],
    )
    renamed = runner.invoke(
        app,
        ["project", "rename", str(project.path), "--name", "Display Name", "--json"],
    )
    moved = runner.invoke(
        app,
        [
            "project",
            "move",
            str(project.path),
            "--destination",
            str(tmp_path / "moved"),
            "--json",
        ],
    )

    assert planned.exit_code == 0, planned.output
    assert json.loads(planned.stdout)["operations"][0]["type"] == "project.rename"
    assert renamed.exit_code == 0, renamed.output
    assert json.loads(renamed.stdout)["operations"][0]["new_name"] == "Display Name"
    assert moved.exit_code == 2
    assert json.loads(moved.stdout)["operations"][0]["type"] == "project.move"


def test_project_apply_cli_requires_and_forwards_reviewed_move_destination(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "old-root")
    destination = tmp_path / "new-root"
    change_set_json = json.dumps(
        plan_project_move(project.path, destination).to_dict(),
        ensure_ascii=False,
    )

    missing = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            change_set_json,
            "--approve",
            "--json",
        ],
    )
    applied = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            change_set_json,
            "--approve",
            "--approved-destination",
            str(destination),
            "--json",
        ],
    )

    assert missing.exit_code == 1
    assert json.loads(missing.stdout)["code"] == "approval_required"
    assert applied.exit_code == 0, applied.output
    assert json.loads(applied.stdout)["project_id"] == inspect_project(destination).project_id


def _web_source(url: str = "https://example.com/guide") -> dict[str, object]:
    return {
        "type": "web",
        "url": url,
        "note": "Primary guide for the new lane.",
        "section": "foundations",
        "kind": "reference",
    }


def _content_hash(seed: str) -> str:
    return f"sha256:{hashlib.sha256(seed.encode()).hexdigest()}"


def _source_by_type(project: object, source_type: str) -> dict[str, object]:
    tape_path = project.tape_path  # type: ignore[attr-defined]
    tape = yaml.safe_load(tape_path.read_text(encoding="utf-8"))
    return next(source for source in tape["sources"] if source["type"] == source_type)


def _tree_bytes(root: Path) -> dict[str, bytes]:
    return {
        path.relative_to(root).as_posix(): path.read_bytes()
        for path in root.rglob("*")
        if path.is_file()
    }


def _rehash_change_set(payload: dict[str, object]) -> dict[str, object]:
    unsigned = dict(payload)
    unsigned.pop("change_set_hash", None)
    encoded = json.dumps(unsigned, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return {**unsigned, "change_set_hash": hashlib.sha256(encoded.encode()).hexdigest()}


def test_plan_source_add_is_versioned_and_read_only(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    before = _tree_bytes(project.path)

    change_set = plan_source_add(project.path, _web_source())
    payload = change_set.to_dict()

    assert payload["contract"] == "liner.project_change_set"
    assert payload["version"] == 1
    assert payload["change_set_id"]
    assert payload["change_set_hash"]
    assert payload["project_id"] == read_liner_metadata(project)["id"]
    assert payload["expected_revision"].startswith("sha256:")
    assert payload["expected_content_hash"]
    assert payload["risk"] == "additive"
    assert payload["approval_required"] is False
    assert [operation["type"] for operation in payload["operations"]] == ["source.add"]
    assert "mixtape/tape.yaml" in payload["file_effects"]["write"]
    assert payload["validation"]
    assert _tree_bytes(project.path) == before


def test_source_add_batch_plans_applies_and_replays_one_atomic_change_set(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    tape_before = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    existing = {key: value for key, value in tape_before["sources"][0].items() if key != "id"}
    first = _web_source("https://example.com/batch-one")
    second = _web_source("https://example.com/batch-two")
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {
            "type": "source.add",
            "sources": [first, existing, second, first],
        },
    }

    planned = runner.invoke(
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

    assert planned.exit_code == 0, planned.output
    change_set = json.loads(planned.stdout)
    assert [operation["type"] for operation in change_set["operations"]] == [
        "source.add",
        "source.noop",
        "source.add",
        "source.noop",
    ]
    assert [
        operation["duplicate_classification"]
        for operation in change_set["operations"]
        if operation["type"] == "source.noop"
    ] == ["exact_duplicate", "exact_duplicate"]
    assert change_set["operations"][3]["source_id"] == change_set["operations"][0][
        "source_id"
    ]

    apply_args = [
        "project",
        "apply",
        str(project.path),
        "--change-set-json",
        json.dumps(change_set),
        "--json",
    ]
    first_apply = runner.invoke(app, apply_args)
    replay = runner.invoke(app, apply_args)

    assert first_apply.exit_code == 0, first_apply.output
    assert replay.exit_code == 0, replay.output
    receipt = json.loads(first_apply.stdout)
    replayed = json.loads(replay.stdout)
    assert receipt["change_set_id"] == change_set["change_set_id"]
    assert len(receipt["operations"]) == 4
    assert replayed["receipt_id"] == receipt["receipt_id"]
    assert replayed["applied_at"] == receipt["applied_at"]
    assert replayed["replayed"] is True
    tape_after = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert len(tape_after["sources"]) == len(tape_before["sources"]) + 2
    assert {source.get("url") for source in tape_after["sources"]} >= {
        "https://example.com/batch-one",
        "https://example.com/batch-two",
    }


def test_source_add_batch_failure_leaves_active_project_unchanged(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add_batch(
        project.path,
        [
            _web_source("https://example.com/batch-one"),
            _web_source("https://example.com/batch-two"),
        ],
    )
    before = _tree_bytes(project.path)

    def fail_swap(*_args: object, **_kwargs: object) -> None:
        raise OSError("injected batch activation failure")

    monkeypatch.setattr(maintenance_module, "_swap_staged_project", fail_swap)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "apply_failed"
    assert raised.value.report.partial_success is False
    assert _tree_bytes(project.path) == before


def test_project_plan_cli_supports_json_and_readable_output(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {"type": "source.add", "source": _web_source()},
    }

    json_result = runner.invoke(
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
    readable_result = runner.invoke(
        app,
        [
            "project",
            "plan",
            str(project.path),
            "--request-json",
            json.dumps(request),
        ],
    )

    assert json_result.exit_code == 0, json_result.output
    assert json.loads(json_result.stdout)["contract"] == "liner.project_change_set"
    assert readable_result.exit_code == 0, readable_result.output
    assert "Project Change Set" in readable_result.stdout
    assert "Risk: additive" in readable_result.stdout
    assert "Approval required: no" in readable_result.stdout


def test_apply_source_add_preserves_unknown_fields_and_writes_receipt(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    metadata = yaml.safe_load(project.liner_metadata_path.read_text(encoding="utf-8"))
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    metadata["extension"] = {"owner": "Arturo"}
    tape["extension"] = {"quality": "curated"}
    project.liner_metadata_path.write_text(yaml.safe_dump(metadata), encoding="utf-8")
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    change_set = plan_source_add(project.path, _web_source())

    result = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )
    assert result.exit_code == 0, result.output
    payload = json.loads(result.stdout)

    updated_metadata = yaml.safe_load(project.liner_metadata_path.read_text(encoding="utf-8"))
    updated_tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert updated_metadata["extension"] == {"owner": "Arturo"}
    assert updated_tape["extension"] == {"quality": "curated"}
    assert updated_metadata["status"]["stale"] is True
    assert updated_metadata["status"]["milestone"] == "started"
    assert updated_tape["sources"][-1]["url"] == "https://example.com/guide"
    assert updated_tape["sources"][-1]["id"]
    assert payload["contract"] == "liner.change_receipt"
    assert payload["before"]["revision"] == change_set.expected_revision
    assert payload["after"]["revision"] != payload["before"]["revision"]
    assert payload["synthesis_disposition"] == "review_required"
    assert "mixtape/MIXTAPE.md" in payload["stale_artifacts"]
    assert payload["next_actions"]
    receipt_path = (
        project.corpus_path / ".liner-runs" / "maintenance" / f"{payload['receipt_id']}.json"
    )
    assert receipt_path.is_file()
    receipt_text = receipt_path.read_text(encoding="utf-8")
    assert "Primary guide for the new lane" not in receipt_text
    assert "source_body" not in receipt_text


def test_apply_rejects_stale_plan_without_mutating_again(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source())
    project.synthesis_path.write_text("changed after planning\n", encoding="utf-8")
    before_apply = _tree_bytes(project.path)

    result = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )

    assert result.exit_code == 1
    report = json.loads(result.stdout)
    assert report["code"] == "stale_project"
    assert "fresh inspect and plan" in report["message"]
    assert _tree_bytes(project.path) == before_apply


def test_apply_failure_rolls_back_active_project(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source())
    before = _tree_bytes(project.path)

    def fail_swap(*_args: object, **_kwargs: object) -> None:
        raise OSError("injected swap failure")

    monkeypatch.setattr("liner.maintenance._swap_staged_project", fail_swap)

    result = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )

    assert result.exit_code == 1
    report = json.loads(result.stdout)
    assert report["code"] == "apply_failed"
    assert report["partial_success"] is False
    assert _tree_bytes(project.path) == before


def test_external_save_at_activation_is_atomically_rolled_back(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source())
    original_swap = maintenance_module._swap_staged_project
    calls = 0

    def save_then_swap(active: Path, staged: Path) -> None:
        nonlocal calls
        calls += 1
        if calls == 1:
            (project.working_dir / "external-save.md").write_text(
                "external save at activation\n", encoding="utf-8"
            )
        original_swap(active, staged)

    monkeypatch.setattr(maintenance_module, "_swap_staged_project", save_then_swap)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "stale_project"
    assert (project.working_dir / "external-save.md").read_text(
        encoding="utf-8"
    ) == "external save at activation\n"


def test_apply_rejects_symlinked_mutation_targets_without_external_writes(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    external_metadata = tmp_path / "external-liner.yaml"
    shutil.move(project.liner_metadata_path, external_metadata)
    project.liner_metadata_path.symlink_to(external_metadata)
    change_set = plan_source_add(project.path, _web_source())
    before = external_metadata.read_bytes()

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert raised.value.report.code == "unsafe_project"
    assert external_metadata.read_bytes() == before


def test_first_legacy_mutation_assigns_all_ids_atomically(tmp_path: Path) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    tape_path = root / "tape.yaml"
    tape_path.write_text(
        "title: Legacy\ndescription: Old layout\nversion: 1\ncurator: Arturo\n"
        "extension: keep\nsources:\n  - type: web\n    url: https://legacy.example.com\n",
        encoding="utf-8",
    )
    before_plan = _tree_bytes(root)

    change_set = plan_source_add(root, _web_source())

    assert [operation["type"] for operation in change_set.operations] == [
        "identity.assign_project",
        "identity.assign_source",
        "source.add",
    ]
    assert _tree_bytes(root) == before_plan

    refused = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(root),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )
    assert refused.exit_code == 1
    assert json.loads(refused.stdout)["code"] == "approval_required"

    applied = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(root),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--approve",
            "--json",
        ],
    )
    assert applied.exit_code == 0, applied.output

    metadata = yaml.safe_load((root / "liner.yaml").read_text(encoding="utf-8"))
    tape = yaml.safe_load(tape_path.read_text(encoding="utf-8"))
    assert metadata["id"]
    assert metadata["mixtape"] == "."
    assert tape["extension"] == "keep"
    assert all(source["id"] for source in tape["sources"])
    assert not (root / "mixtape" / "tape.yaml").exists()


def test_structural_risk_cannot_be_downgraded_in_a_rehashed_change_set(tmp_path: Path) -> None:
    root = tmp_path / "legacy"
    root.mkdir()
    (root / "tape.yaml").write_text(
        "title: Legacy\ndescription: Old\nversion: 1\ncurator: A\nsources: []\n",
        encoding="utf-8",
    )
    payload = plan_source_add(root, _web_source()).to_dict()
    payload["risk"] = "additive"
    payload["approval_required"] = False
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(root, crafted)

    assert raised.value.report.code == "invalid_change_set"


def test_existing_source_identity_cannot_be_overwritten(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    payload = plan_source_add(project.path, _web_source()).to_dict()
    operations = list(payload["operations"])
    operations.insert(
        0,
        {
            "type": "identity.assign_source",
            "index": 0,
            "source_id": "00000000-0000-4000-8000-000000000001",
        },
    )
    payload["operations"] = operations
    payload["risk"] = "structural"
    payload["approval_required"] = True
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted, approved=True)

    assert raised.value.report.code == "unsafe_project"


def test_exact_duplicate_is_idempotent_noop_and_replay_returns_receipt(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    existing = {key: value for key, value in tape["sources"][0].items() if key != "id"}

    duplicate = plan_source_add(project.path, existing)
    before_revision = duplicate.expected_revision

    assert duplicate.operations[0]["type"] == "source.noop"
    assert duplicate.operations[0]["source_id"] == tape["sources"][0]["id"]
    arguments = [
        "project",
        "apply",
        str(project.path),
        "--change-set-json",
        json.dumps(duplicate.to_dict()),
        "--json",
    ]
    first_result = runner.invoke(app, arguments)
    second_result = runner.invoke(app, arguments)
    assert first_result.exit_code == 0, first_result.output
    assert second_result.exit_code == 0, second_result.output
    first = json.loads(first_result.stdout)
    second = json.loads(second_result.stdout)

    assert second["receipt_id"] == first["receipt_id"]
    assert second["replayed"] is True
    assert first["after"]["revision"] == before_revision
    assert len(yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))["sources"]) == 2


def test_duplicate_normalization_treats_default_web_render_as_server(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    existing = next(source for source in tape["sources"] if source["type"] == "web")
    request = {key: value for key, value in existing.items() if key != "id"}
    request["render"] = "server"

    change_set = plan_source_add(project.path, request)

    assert [operation["type"] for operation in change_set.operations] == ["source.noop"]


def test_receipt_replay_refuses_unverifiable_receipt_after_later_project_work(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source())
    apply_change_set(project.path, change_set)
    project.synthesis_path.write_text("changed after receipt\n", encoding="utf-8")

    with pytest.raises(ProjectApplyError) as replay:
        apply_change_set(project.path, change_set)

    assert replay.value.report.code == "receipt_state_mismatch"


def test_receipt_replay_refuses_when_the_added_source_was_removed(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source())
    receipt = apply_change_set(project.path, change_set)
    added_id = next(
        operation["source_id"]
        for operation in change_set.operations
        if operation["type"] == "source.add"
    )
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    tape["sources"] = [source for source in tape["sources"] if source.get("id") != added_id]
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set)

    assert receipt.change_set_id == change_set.change_set_id
    assert raised.value.report.code == "receipt_effect_missing"


def test_receipt_replay_rejects_changed_content_under_the_same_id(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    original = plan_source_add(project.path, _web_source())
    apply_change_set(project.path, original)
    payload = original.to_dict()
    operations = list(payload["operations"])
    add_operation = dict(operations[-1])
    source = dict(add_operation["source"])
    source["url"] = "https://example.com/different"
    add_operation["source"] = source
    operations[-1] = add_operation
    payload["operations"] = operations
    altered = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, altered)

    assert raised.value.report.code == "receipt_state_mismatch"


def test_receipt_redacts_credentials_query_and_url_path(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    secret_url = "https://user:password@example.com/reset/secret-token?signature=private"

    change_set = plan_source_add(project.path, _web_source(secret_url))
    result = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )
    assert result.exit_code == 0, result.output
    receipt = json.loads(result.stdout)
    receipt_path = (
        project.corpus_path / ".liner-runs" / "maintenance" / f"{receipt['receipt_id']}.json"
    )
    receipt_text = receipt_path.read_text(encoding="utf-8")

    assert "password" not in receipt_text
    assert "secret-token" not in receipt_text
    assert "signature" not in receipt_text
    assert "https://example.com/[path-redacted]" in receipt_text


def test_project_apply_cli_and_sources_add_use_canonical_path(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    change_set = plan_source_add(project.path, _web_source("https://example.com/planned"))

    apply_result = runner.invoke(
        app,
        [
            "project",
            "apply",
            str(project.path),
            "--change-set-json",
            json.dumps(change_set.to_dict()),
            "--json",
        ],
    )
    add_result = runner.invoke(
        app,
        [
            "sources",
            "add",
            str(project.path),
            "--type",
            "web",
            "--url",
            "https://example.com/convenience",
            "--note",
            "Convenience path.",
            "--json",
        ],
    )

    assert apply_result.exit_code == 0, apply_result.output
    assert json.loads(apply_result.stdout)["contract"] == "liner.change_receipt"
    assert add_result.exit_code == 0, add_result.output
    assert json.loads(add_result.stdout)["contract"] == "liner.change_receipt"
    urls = [
        source.get("url")
        for source in yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))["sources"]
    ]
    assert "https://example.com/planned" in urls
    assert "https://example.com/convenience" in urls


def test_project_apply_cli_returns_failure_report_for_malformed_json(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    result = runner.invoke(
        app,
        ["project", "apply", str(project.path), "--change-set-json", "{", "--json"],
    )

    assert result.exit_code == 1
    payload = json.loads(result.stdout)
    assert payload["contract"] == "liner.failure_report"
    assert payload["code"] == "invalid_change_set"


def test_source_update_preserves_identity_and_uses_canonical_apply(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    source_id = str(source["id"])

    change_set = plan_source_update(
        project.path,
        source_id,
        {"note": "A sharper curator note.", "kind": "principle"},
    )

    assert change_set.risk == "metadata"
    assert change_set.approval_required is False
    assert change_set.operations[-1]["type"] == "source.update"
    receipt = apply_change_set(project.path, change_set)
    updated = _source_by_type(project, "web")
    assert updated["id"] == source_id
    assert updated["note"] == "A sharper curator note."
    assert updated["kind"] == "principle"
    assert receipt.operations[-1]["duplicate_classification"] == "metadata_update"


def test_source_replace_mints_successor_and_retains_predecessor_capture(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    original_path = project.local_sources_dir / "original.md"
    successor_path = project.local_sources_dir / "successor.md"
    original_path.parent.mkdir(parents=True, exist_ok=True)
    original_path.write_text("original evidence\n", encoding="utf-8")
    successor_path.write_text("replacement evidence\n", encoding="utf-8")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    predecessor_id = "00000000-0000-4000-8000-000000000111"
    tape["sources"].append(
        {
            "id": predecessor_id,
            "type": "local_file",
            "path": "local-sources/original.md",
            "citation": "Original capture",
            "content_hash": _content_hash("original evidence\n"),
        }
    )
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")

    change_set = plan_source_replace(
        project.path,
        predecessor_id,
        {
            "type": "local_file",
            "path": "local-sources/successor.md",
            "citation": "Successor capture",
            "content_hash": _content_hash("replacement evidence\n"),
        },
    )

    operation = change_set.operations[-1]
    assert change_set.risk == "semantic"
    assert change_set.approval_required is True
    assert operation["predecessor_source_id"] == predecessor_id
    assert operation["successor_source_id"] != predecessor_id
    with pytest.raises(ProjectApplyError) as refused:
        apply_change_set(project.path, change_set)
    assert refused.value.report.code == "approval_required"

    receipt = apply_change_set(project.path, change_set, approved=True)
    updated_tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    successor = next(
        source
        for source in updated_tape["sources"]
        if source.get("id") == operation["successor_source_id"]
    )
    assert successor["path"] == "local-sources/successor.md"
    assert all(source.get("id") != predecessor_id for source in updated_tape["sources"])
    assert original_path.read_text(encoding="utf-8") == "original evidence\n"
    payload = receipt.to_dict()
    assert payload["lineage"]["sources"] == [
        {"predecessor": predecessor_id, "successor": operation["successor_source_id"]}
    ]
    assert f"source:{predecessor_id}" in payload["file_effects"]["retain"]


def test_same_locator_changed_content_plans_replacement(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    target = next(source for source in tape["sources"] if source["type"] == "web")
    target["content_hash"] = _content_hash("old body")
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    replacement = {key: value for key, value in target.items() if key != "id"}
    replacement["content_hash"] = _content_hash("new body")

    change_set = plan_source_replace(project.path, str(target["id"]), replacement)

    assert change_set.operations[-1]["duplicate_classification"] == ("same_locator_changed_content")


def test_same_content_new_locator_requires_distinct_provenance_reason(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    web = next(source for source in tape["sources"] if source["type"] == "web")
    youtube = next(source for source in tape["sources"] if source["type"] == "youtube")
    shared_hash = _content_hash("shared body")
    web["content_hash"] = shared_hash
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    replacement = {
        "type": "web",
        "url": "https://mirror.example.com/guide",
        "note": "Mirror with distinct provenance.",
        "content_hash": shared_hash,
    }

    with pytest.raises(ProjectInspectionError, match="distinct provenance"):
        plan_source_replace(project.path, str(youtube["id"]), replacement)
    with pytest.raises(ProjectInspectionError, match="non-empty provenance reason"):
        plan_source_replace(
            project.path,
            str(youtube["id"]),
            replacement,
            provenance_intent="distinct",
        )

    change_set = plan_source_replace(
        project.path,
        str(youtube["id"]),
        replacement,
        provenance_intent="distinct",
        provenance_reason="Independent canonical mirror.",
    )
    assert change_set.operations[-1]["duplicate_classification"] == "duplicate_provenance"


def test_ambiguous_content_duplicates_fail_closed_with_source_choices(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    shared_hash = _content_hash("ambiguous body")
    for source in tape["sources"]:
        source["content_hash"] = shared_hash
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    target_id = str(tape["sources"][0]["id"])

    with pytest.raises(ProjectInspectionError) as raised:
        plan_source_replace(
            project.path,
            target_id,
            {
                "type": "web",
                "url": "https://third.example.com/guide",
                "content_hash": shared_hash,
            },
            provenance_intent="distinct",
            provenance_reason="A separate publication endpoint.",
        )

    assert "Ambiguous duplicate content" in str(raised.value)
    assert all(str(source["id"]) in str(raised.value) for source in tape["sources"])


def test_source_replace_replay_returns_original_receipt_without_rewriting(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    replacement = {key: value for key, value in source.items() if key != "id"}
    replacement["content_hash"] = _content_hash("replacement")
    change_set = plan_source_replace(project.path, str(source["id"]), replacement)
    first = apply_change_set(project.path, change_set, approved=True)
    after_first = _tree_bytes(project.path)

    replay = apply_change_set(project.path, change_set, approved=True)

    assert replay.receipt_id == first.receipt_id
    assert replay.replayed is True
    assert _tree_bytes(project.path) == after_first


def test_sources_update_and_replace_cli_compile_to_canonical_change_sets(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    source_id = str(source["id"])

    updated = runner.invoke(
        app,
        [
            "sources",
            "update",
            str(project.path),
            "--source-id",
            source_id,
            "--note",
            "Updated through the convenience compiler.",
            "--json",
        ],
    )
    replacement = runner.invoke(
        app,
        [
            "sources",
            "replace",
            str(project.path),
            "--source-id",
            source_id,
            "--type",
            "web",
            "--url",
            str(source["url"]),
            "--content-hash",
            _content_hash("new remote body"),
            "--json",
        ],
    )

    assert updated.exit_code == 0, updated.output
    assert json.loads(updated.stdout)["contract"] == "liner.change_receipt"
    assert replacement.exit_code == 2, replacement.output
    payload = json.loads(replacement.stdout)
    assert payload["contract"] == "liner.project_change_set"
    assert payload["operations"][-1]["type"] == "source.replace"


def test_source_update_can_move_a_local_locator_without_changing_identity(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    project.local_sources_dir.mkdir(parents=True, exist_ok=True)
    (project.local_sources_dir / "first.md").write_text("first\n", encoding="utf-8")
    (project.local_sources_dir / "moved.md").write_text("first\n", encoding="utf-8")
    added = plan_source_add(
        project.path,
        {
            "type": "local_file",
            "path": "local-sources/first.md",
            "citation": "Local evidence",
        },
    )
    receipt = apply_change_set(project.path, added)
    source_id = next(
        operation["source_id"]
        for operation in added.operations
        if operation["type"] == "source.add"
    )
    assert receipt.after["revision"] != receipt.before["revision"]

    change_set = plan_source_update(
        project.path,
        source_id,
        {"path": "local-sources/moved.md"},
    )
    apply_change_set(project.path, change_set)

    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    moved = next(source for source in tape["sources"] if source["id"] == source_id)
    assert moved["path"] == "local-sources/moved.md"


def test_source_replace_exact_duplicate_is_noop_with_existing_identity(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    web = _source_by_type(project, "web")
    youtube = _source_by_type(project, "youtube")

    change_set = plan_source_replace(
        project.path,
        str(youtube["id"]),
        {key: value for key, value in web.items() if key != "id"},
    )

    assert change_set.risk == "additive"
    assert change_set.approval_required is False
    assert change_set.operations == (
        {
            "type": "source.noop",
            "source_id": web["id"],
            "duplicate_classification": "exact_identity",
        },
    )
    receipt = apply_change_set(project.path, change_set)
    assert receipt.after == receipt.before


def test_source_replace_refuses_stale_project_without_partial_effects(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    replacement = {key: value for key, value in source.items() if key != "id"}
    replacement["content_hash"] = _content_hash("new body")
    change_set = plan_source_replace(project.path, str(source["id"]), replacement)
    before = project.tape_path.read_text(encoding="utf-8")
    project.synthesis_path.write_text("concurrent synthesis\n", encoding="utf-8")

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, change_set, approved=True)

    assert raised.value.report.code == "stale_project"
    assert project.tape_path.read_text(encoding="utf-8") == before


def test_replacement_receipt_records_lineage_without_private_provenance_reason(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    web = next(source for source in tape["sources"] if source["type"] == "web")
    youtube = next(source for source in tape["sources"] if source["type"] == "youtube")
    shared_hash = _content_hash("shared evidence")
    web["content_hash"] = shared_hash
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    private_reason = "Internal review secret: keep this out of receipts."
    change_set = plan_source_replace(
        project.path,
        str(youtube["id"]),
        {
            "type": "web",
            "url": "https://mirror.example.com/evidence?token=secret#private",
            "content_hash": shared_hash,
        },
        provenance_intent="distinct",
        provenance_reason=private_reason,
    )

    receipt = apply_change_set(project.path, change_set, approved=True)
    payload = receipt.to_dict()
    encoded = json.dumps(payload)

    assert payload["operations"][-1]["provenance_intent"] == "distinct"
    assert private_reason not in encoded
    assert "token=secret" not in encoded
    assert "#private" not in encoded
    assert payload["lineage"]["sources"]


def test_apply_rederives_update_duplicate_rules_from_rehashed_change_set(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    web = _source_by_type(project, "web")
    youtube = _source_by_type(project, "youtube")
    payload = plan_source_update(
        project.path,
        str(web["id"]),
        {"note": "Legitimate preview."},
    ).to_dict()
    payload["operations"][-1]["changes"] = {"url": youtube["url"]}
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted)

    assert raised.value.report.code == "unsafe_project"
    assert _source_by_type(project, "web")["url"] == web["url"]


def test_apply_rederives_replacement_provenance_from_rehashed_change_set(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    web = next(source for source in tape["sources"] if source["type"] == "web")
    youtube = next(source for source in tape["sources"] if source["type"] == "youtube")
    shared_hash = _content_hash("shared body")
    web["content_hash"] = shared_hash
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    payload = plan_source_replace(
        project.path,
        str(youtube["id"]),
        {
            "type": "web",
            "url": "https://unique.example.com/evidence",
            "content_hash": _content_hash("unique body"),
        },
    ).to_dict()
    payload["operations"][-1]["source"] = {
        "type": "web",
        "url": "https://mirror.example.com/evidence",
        "content_hash": shared_hash,
    }
    payload["operations"][-1]["duplicate_classification"] = "explicit_replacement"
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted, approved=True)

    assert raised.value.report.code == "unsafe_project"
    assert _source_by_type(project, "youtube")["id"] == youtube["id"]


@pytest.mark.parametrize("operation", ["update", "replace"])
def test_source_mutation_accepts_inspected_canonical_id_for_uppercase_yaml_uuid(
    tmp_path: Path,
    operation: str,
) -> None:
    project = init_project(tmp_path / operation)
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    web = next(source for source in tape["sources"] if source["type"] == "web")
    web["id"] = "ABCDEFAB-CDEF-4ABC-8DEF-ABCDEFABCDEF"
    project.tape_path.write_text(yaml.safe_dump(tape), encoding="utf-8")
    inspected_id = "abcdefab-cdef-4abc-8def-abcdefabcdef"

    if operation == "update":
        change_set = plan_source_update(
            project.path,
            inspected_id,
            {"note": "Canonical lookup works."},
        )
        apply_change_set(project.path, change_set)
        assert _source_by_type(project, "web")["note"] == "Canonical lookup works."
    else:
        replacement = {key: value for key, value in web.items() if key != "id"}
        replacement["content_hash"] = _content_hash("successor body")
        change_set = plan_source_replace(project.path, inspected_id, replacement)
        apply_change_set(project.path, change_set, approved=True)
        assert (
            _source_by_type(project, "web")["id"]
            == (change_set.operations[-1]["successor_source_id"])
        )


def test_project_plan_rejects_non_string_provenance_without_uncaught_exception(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    request = {
        "contract": "liner.maintenance_request",
        "version": 1,
        "operation": {
            "type": "source.replace",
            "source_id": source["id"],
            "source": {key: value for key, value in source.items() if key != "id"},
            "provenance_intent": "distinct",
            "provenance_reason": 42,
        },
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

    assert result.exit_code == 1
    assert not isinstance(result.exception, AttributeError)
    assert "provenance_reason must be a string" in result.output


def test_human_replacement_preview_includes_exact_apply_payload(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")

    result = runner.invoke(
        app,
        [
            "sources",
            "replace",
            str(project.path),
            "--source-id",
            str(source["id"]),
            "--type",
            "web",
            "--url",
            str(source["url"]),
            "--content-hash",
            _content_hash("changed body"),
        ],
    )

    assert result.exit_code == 2
    assert "Exact Change Set JSON:" in result.output
    assert "--change-set-json" in result.output
    assert '"change_set_hash"' in result.output


def test_update_receipt_names_changed_fields_and_redacts_locator_values(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    change_set = plan_source_update(
        project.path,
        str(source["id"]),
        {
            "url": "https://user:password@new.example.com/private?token=secret#fragment",
            "note": "Private curator context.",
        },
    )

    payload = apply_change_set(project.path, change_set).to_dict()
    operation = payload["operations"][-1]
    encoded = json.dumps(payload)

    assert operation["changed_fields"] == ["note", "url"]
    assert operation["locator"] == "https://new.example.com/[path-redacted]"
    assert "password" not in encoded
    assert "token=secret" not in encoded
    assert "Private curator context" not in encoded


def test_apply_rejects_rehashed_update_with_no_actual_changes(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    payload = plan_source_update(
        project.path,
        str(source["id"]),
        {"note": "Previewed note."},
    ).to_dict()
    payload["operations"][-1]["changes"] = {"note": source.get("note")}
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))
    before = _tree_bytes(project.path)

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted)

    assert raised.value.report.code == "unsafe_project"
    assert _tree_bytes(project.path) == before


@pytest.mark.parametrize("command", ["update", "replace"])
def test_source_mutation_cli_rejects_malformed_url_port_cleanly(
    tmp_path: Path,
    command: str,
) -> None:
    project = init_project(tmp_path / command)
    source = _source_by_type(project, "web")
    arguments = [
        "sources",
        command,
        str(project.path),
        "--source-id",
        str(source["id"]),
    ]
    if command == "replace":
        arguments.extend(["--type", "web"])
    arguments.extend(["--url", "https://example.com:bad/path", "--json"])

    result = runner.invoke(app, arguments)

    assert result.exit_code == 1
    assert not isinstance(result.exception, ValueError)
    assert "invalid port" in result.output


def _add_local_source_for_retention(
    project: object, *, note: str = "Retain me."
) -> tuple[str, Path]:
    local_sources_dir = project.local_sources_dir  # type: ignore[attr-defined]
    project_path = project.path  # type: ignore[attr-defined]
    local_sources_dir.mkdir(parents=True, exist_ok=True)
    artifact = local_sources_dir / "retained-evidence.md"
    artifact.write_text("captured evidence body\n", encoding="utf-8")
    change_set = plan_source_add(
        project_path,
        {
            "type": "local_file",
            "path": "local-sources/retained-evidence.md",
            "citation": "Retained evidence",
            "note": note,
        },
    )
    apply_change_set(project_path, change_set)
    source_id = next(
        operation["source_id"]
        for operation in change_set.operations
        if operation["type"] == "source.add"
    )
    return source_id, artifact


def test_source_remove_detaches_but_retains_capture_and_lineage(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(
        project,
        note="private curator detail",
    )
    before_plan = _tree_bytes(project.path)

    change_set = plan_source_remove(project.path, source_id)

    assert _tree_bytes(project.path) == before_plan
    assert change_set.risk == "semantic"
    assert change_set.approval_required is True
    assert change_set.operations[-1]["disposition"] == "detached_retained"
    artifact_effect = artifact.relative_to(project.path).as_posix()
    assert artifact_effect in change_set.file_effects["retain"]
    with pytest.raises(ProjectApplyError) as declined:
        apply_change_set(project.path, change_set)
    assert declined.value.report.code == "approval_required"

    receipt = apply_change_set(project.path, change_set, approved=True)
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    retention_path = project.corpus_path / ".liner-runs" / "retained-sources" / f"{source_id}.json"
    retained = json.loads(retention_path.read_text(encoding="utf-8"))

    assert all(source["id"] != source_id for source in tape["sources"])
    assert artifact.read_text(encoding="utf-8") == "captured evidence body\n"
    assert retained["source_id"] == source_id
    assert retained["source"]["note"] == "private curator detail"
    payload = receipt.to_dict()
    assert payload["lineage"]["retained_sources"] == [source_id]
    assert payload["operations"][-1]["disposition"] == "detached_retained"
    assert "private curator detail" not in json.dumps(payload)


def test_source_purge_is_separate_destructive_preview_and_replays_receipt(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(project)
    with pytest.raises(ProjectInspectionError, match="detached Source"):
        plan_source_purge(project.path, source_id)
    remove = plan_source_remove(project.path, source_id)
    apply_change_set(project.path, remove, approved=True)
    retention_path = project.corpus_path / ".liner-runs" / "retained-sources" / f"{source_id}.json"

    purge = plan_source_purge(project.path, source_id)

    assert purge.risk == "destructive"
    assert purge.approval_required is True
    assert artifact.relative_to(project.path).as_posix() in purge.file_effects["delete"]
    assert purge.file_effects["purge"] == purge.file_effects["delete"]
    with pytest.raises(ProjectApplyError) as declined:
        apply_change_set(project.path, purge)
    assert declined.value.report.code == "approval_required"
    assert artifact.is_file()
    assert retention_path.is_file()

    receipt = apply_change_set(project.path, purge, approved=True)
    after_first = _tree_bytes(project.path)
    replay = apply_change_set(project.path, purge, approved=True)

    assert not artifact.exists()
    assert not retention_path.exists()
    assert receipt.to_dict()["lineage"]["purged_sources"] == [source_id]
    assert receipt.synthesis_disposition == "unchanged"
    assert replay.receipt_id == receipt.receipt_id
    assert replay.replayed is True
    assert _tree_bytes(project.path) == after_first


def test_remove_and_purge_refuse_stale_or_missing_lineage(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, _ = _add_local_source_for_retention(project)
    stale_remove = plan_source_remove(project.path, source_id)
    project.synthesis_path.write_text("concurrent edit\n", encoding="utf-8")

    with pytest.raises(ProjectApplyError) as stale:
        apply_change_set(project.path, stale_remove, approved=True)
    assert stale.value.report.code == "stale_project"

    fresh_remove = plan_source_remove(project.path, source_id)
    apply_change_set(project.path, fresh_remove, approved=True)
    retention_path = project.corpus_path / ".liner-runs" / "retained-sources" / f"{source_id}.json"
    purge = plan_source_purge(project.path, source_id)
    retained = json.loads(retention_path.read_text(encoding="utf-8"))
    retained["retention_id"] = "00000000-0000-4000-8000-000000000999"
    retention_path.write_text(json.dumps(retained), encoding="utf-8")

    with pytest.raises(ProjectApplyError) as changed_lineage:
        apply_change_set(project.path, purge, approved=True)
    assert changed_lineage.value.report.code == "unsafe_project"
    retention_path.unlink()
    with pytest.raises(ProjectInspectionError, match="lineage is missing"):
        plan_source_purge(project.path, source_id)


def test_purge_staging_failure_leaves_retained_state_unchanged(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(project)
    apply_change_set(project.path, plan_source_remove(project.path, source_id), approved=True)
    purge = plan_source_purge(project.path, source_id)
    before = _tree_bytes(project.path)

    def fail_receipt(*args: object, **kwargs: object) -> object:
        raise RuntimeError("injected receipt failure after staged purge")

    monkeypatch.setattr(maintenance_module, "_build_receipt", fail_receipt)
    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, purge, approved=True)

    assert raised.value.report.code == "apply_failed"
    assert artifact.is_file()
    assert _tree_bytes(project.path) == before


def test_mixed_destructive_change_set_cannot_downgrade_approval(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, _ = _add_local_source_for_retention(project)
    apply_change_set(project.path, plan_source_remove(project.path, source_id), approved=True)
    payload = plan_source_purge(project.path, source_id).to_dict()
    existing = _source_by_type(project, "web")
    payload["operations"].append(
        {
            "type": "source.noop",
            "source_id": existing["id"],
            "duplicate_classification": "exact_identity",
        }
    )
    mixed = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    assert mixed.risk == "destructive"
    with pytest.raises(ProjectApplyError) as declined:
        apply_change_set(project.path, mixed)
    assert declined.value.report.code == "approval_required"

    payload["risk"] = "semantic"
    downgraded = ProjectChangeSet.from_dict(_rehash_change_set(payload))
    with pytest.raises(ProjectApplyError) as invalid:
        apply_change_set(project.path, downgraded, approved=True)
    assert invalid.value.report.code == "invalid_change_set"


def test_remove_and_purge_cli_emit_canonical_reviewable_change_sets(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, _ = _add_local_source_for_retention(project)
    removed = runner.invoke(
        app,
        ["sources", "remove", str(project.path), "--source-id", source_id, "--json"],
    )

    assert removed.exit_code == 2, removed.output
    remove_payload = json.loads(removed.stdout)
    assert remove_payload["operations"][-1]["type"] == "source.remove"
    apply_change_set(
        project.path,
        ProjectChangeSet.from_dict(remove_payload),
        approved=True,
    )
    purged = runner.invoke(
        app,
        ["sources", "purge", str(project.path), "--source-id", source_id, "--json"],
    )

    assert purged.exit_code == 2, purged.output
    purge_payload = json.loads(purged.stdout)
    assert purge_payload["operations"][-1]["type"] == "source.purge"
    assert purge_payload["risk"] == "destructive"
    assert purge_payload["file_effects"]["delete"]


def test_remove_moves_compiled_capture_into_retention_and_purge_deletes_it(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    compiled = project.sources_dir / "01-compiled-evidence.md"
    compiled.write_text(
        "# Compiled evidence\n\n"
        "**Source type:** web  \n"
        f"**URL:** {source['url']}  \n\n"
        "captured remote body\n",
        encoding="utf-8",
    )

    remove = plan_source_remove(project.path, str(source["id"]))
    move_effect = remove.file_effects["move"][-1]
    retained_relative = move_effect.split(" -> ", 1)[1]
    retained = project.path / retained_relative
    apply_change_set(project.path, remove, approved=True)

    assert not compiled.exists()
    assert retained.read_text(encoding="utf-8").endswith("captured remote body\n")
    purge = plan_source_purge(project.path, str(source["id"]))
    assert retained_relative in purge.file_effects["delete"]
    apply_change_set(project.path, purge, approved=True)
    assert not retained.exists()


def test_purge_refuses_capture_shared_by_an_active_source(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    first_id, artifact = _add_local_source_for_retention(project, note="First use.")
    second = plan_source_add(
        project.path,
        {
            "type": "local_file",
            "path": "local-sources/retained-evidence.md",
            "citation": "Retained evidence",
            "note": "A distinct active use of the same capture.",
        },
    )
    apply_change_set(project.path, second)
    second_id = next(
        operation["source_id"]
        for operation in second.operations
        if operation["type"] == "source.add"
    )
    apply_change_set(project.path, plan_source_remove(project.path, first_id), approved=True)

    with pytest.raises(ProjectInspectionError) as raised:
        plan_source_purge(project.path, first_id)

    assert second_id in str(raised.value)
    assert "still referenced" in str(raised.value)
    assert artifact.is_file()


def test_purge_refuses_replaced_content_at_a_retained_path(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(project)
    apply_change_set(project.path, plan_source_remove(project.path, source_id), approved=True)
    artifact.write_text("unrelated replacement data\n", encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="changed after removal"):
        plan_source_purge(project.path, source_id)

    assert artifact.read_text(encoding="utf-8") == "unrelated replacement data\n"


def test_apply_rederives_remove_lineage_and_file_effects(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(project)
    payload = plan_source_remove(project.path, source_id).to_dict()
    operation = payload["operations"][-1]
    operation["retention_record"]["artifacts"] = []
    operation["retention_record"]["artifact_fingerprints"] = []
    payload["file_effects"]["retain"] = [f"source:{source_id}"]
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted, approved=True)

    assert raised.value.report.code == "unsafe_project"
    assert artifact.is_file()
    assert any(
        source["id"] == source_id
        for source in yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))["sources"]
    )


def test_apply_rejects_redirected_purge_lineage_path_and_effects(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source_id, artifact = _add_local_source_for_retention(project)
    apply_change_set(project.path, plan_source_remove(project.path, source_id), approved=True)
    payload = plan_source_purge(project.path, source_id).to_dict()
    operation = payload["operations"][-1]
    canonical_relative = str(operation["retention_record_path"])
    canonical = project.path / canonical_relative
    redirected_relative = canonical.with_name("important.json").relative_to(project.path).as_posix()
    redirected = project.path / redirected_relative
    shutil.copy2(canonical, redirected)
    operation["retention_record_path"] = redirected_relative
    payload["file_effects"]["delete"] = [
        redirected_relative if item == canonical_relative else item
        for item in payload["file_effects"]["delete"]
    ]
    payload["file_effects"]["purge"] = list(payload["file_effects"]["delete"])
    crafted = ProjectChangeSet.from_dict(_rehash_change_set(payload))

    with pytest.raises(ProjectApplyError) as raised:
        apply_change_set(project.path, crafted, approved=True)

    assert raised.value.report.code == "unsafe_project"
    assert artifact.is_file()
    assert canonical.is_file()
    assert redirected.is_file()


def test_remove_refuses_existing_retention_vault_capture(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    source = _source_by_type(project, "web")
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    compiled = project.sources_dir / "01-existing-vault.md"
    compiled.write_text(
        f"# Compiled\n\n**Source type:** web  \n**URL:** {source['url']}  \n\nnew capture\n",
        encoding="utf-8",
    )
    destination = (
        project.corpus_path
        / ".liner-runs"
        / "retained-sources"
        / str(source["id"])
        / "captures"
        / compiled.name
    )
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text("older retained data\n", encoding="utf-8")

    with pytest.raises(ProjectInspectionError, match="already exists"):
        plan_source_remove(project.path, str(source["id"]))

    assert destination.read_text(encoding="utf-8") == "older retained data\n"
    assert compiled.read_text(encoding="utf-8").endswith("new capture\n")


def test_remove_refuses_compiled_capture_shared_by_an_active_source(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    project = init_project(tmp_path / "demo")
    first = _source_by_type(project, "web")
    second = plan_source_add(
        project.path,
        {
            "type": "web",
            "url": first["url"],
            "note": "Distinct curation over the same locator.",
        },
    )
    apply_change_set(project.path, second)
    second_id = next(
        operation["source_id"]
        for operation in second.operations
        if operation["type"] == "source.add"
    )
    project.sources_dir.mkdir(parents=True, exist_ok=True)
    compiled = project.sources_dir / "01-shared-compiled.md"
    compiled.write_text(
        "# Shared compiled capture\n\n"
        "**Source type:** web  \n"
        f"**URL:** {first['url']}  \n\n"
        "shared body\n",
        encoding="utf-8",
    )

    with pytest.raises(ProjectInspectionError) as planned:
        plan_source_remove(project.path, str(first["id"]))
    assert second_id in str(planned.value)
    assert "still referenced" in str(planned.value)

    original_guard = maintenance_module._ensure_capture_moves_unshared
    monkeypatch.setattr(
        maintenance_module,
        "_ensure_capture_moves_unshared",
        lambda *args, **kwargs: None,
    )
    crafted_at_plan_boundary = plan_source_remove(project.path, str(first["id"]))
    monkeypatch.setattr(
        maintenance_module,
        "_ensure_capture_moves_unshared",
        original_guard,
    )
    with pytest.raises(ProjectApplyError) as applied:
        apply_change_set(project.path, crafted_at_plan_boundary, approved=True)

    assert applied.value.report.code == "unsafe_project"
    assert compiled.is_file()
    tape = yaml.safe_load(project.tape_path.read_text(encoding="utf-8"))
    assert {str(source["id"]) for source in tape["sources"]} >= {
        str(first["id"]),
        second_id,
    }
