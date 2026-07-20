from __future__ import annotations

import json
from dataclasses import replace
from pathlib import Path

import pytest
from typer.testing import CliRunner

from liner import __version__
from liner.agent_adapters import (
    AdapterError,
    apply_agent_adapter_plan,
    inspect_agent_adapter,
    plan_agent_adapter,
)
from liner.cli import app
from liner.maintenance import (
    ProjectApplyError,
    apply_change_set,
    plan_pointer_adapter,
)
from liner.project import init_project

runner = CliRunner()


def test_install_update_remove_preserve_unmanaged_skill_content(tmp_path: Path) -> None:
    home = tmp_path / "home"
    existing_skill = home / ".codex" / "skills" / "user-skill" / "SKILL.md"
    existing_skill.parent.mkdir(parents=True)
    existing_skill.write_text("User-owned skill.\n", encoding="utf-8")
    plan = plan_agent_adapter("install", "codex", home)

    assert plan.approval_required is True
    assert plan.compatibility == "compatible"
    assert plan.target == home / ".codex" / "skills" / "liner-maintenance"
    assert not plan.target.exists()

    with pytest.raises(AdapterError, match="approval"):
        apply_agent_adapter_plan(plan)
    receipt = apply_agent_adapter_plan(plan, approved=True)
    skill_path = plan.target / "SKILL.md"
    state_path = plan.target / ".liner-managed.json"
    assert "liner project guidance" in skill_path.read_text(encoding="utf-8")
    assert json.loads(state_path.read_text(encoding="utf-8"))["liner_version"] == __version__
    assert Path(receipt.receipt_path).is_file()

    skill_path.write_text(
        skill_path.read_text(encoding="utf-8") + "\n## User notes\nKeep this.\n",
        encoding="utf-8",
    )
    update = plan_agent_adapter("update", "codex", home)
    apply_agent_adapter_plan(update, approved=True)
    assert "## User notes\nKeep this." in skill_path.read_text(encoding="utf-8")

    remove = plan_agent_adapter("remove", "codex", home)
    apply_agent_adapter_plan(remove, approved=True)
    preserved = "\n## User notes\nKeep this.\n"
    assert skill_path.read_text(encoding="utf-8") == preserved
    assert not state_path.exists()
    assert existing_skill.read_text(encoding="utf-8") == "User-owned skill.\n"
    assert remove.file_effects["write"] == [str(skill_path)]

    reinstall = plan_agent_adapter("install", "codex", home)
    assert reinstall.status == "preservable"
    apply_agent_adapter_plan(reinstall, approved=True)
    second_remove = plan_agent_adapter("remove", "codex", home)
    apply_agent_adapter_plan(second_remove, approved=True)
    assert skill_path.read_text(encoding="utf-8") == preserved


def test_adapter_refuses_changed_managed_content_and_unsupported_environment(
    tmp_path: Path,
) -> None:
    home = tmp_path / "home"
    install = plan_agent_adapter("install", "claude", home)
    apply_agent_adapter_plan(install, approved=True)
    skill = install.target / "SKILL.md"
    skill.write_text(
        skill.read_text(encoding="utf-8").replace(
            "Use the installed Liner CLI", "Silently edit YAML"
        ),
        encoding="utf-8",
    )

    with pytest.raises(AdapterError, match="managed content changed"):
        plan_agent_adapter("update", "claude", home)
    with pytest.raises(AdapterError, match="Supported environments"):
        plan_agent_adapter("install", "gemini", home)


def test_cli_adapter_flow_is_explicit_json_and_idempotent(tmp_path: Path) -> None:
    home = tmp_path / "home"
    preview = runner.invoke(
        app,
        ["adapters", "install", "codex", "--home", str(home), "--json"],
    )
    assert preview.exit_code == 2
    preview_payload = json.loads(preview.stdout)
    assert preview_payload["approval_required"] is True
    assert preview_payload["file_effects"]["create"]

    installed = runner.invoke(
        app,
        ["adapters", "install", "codex", "--home", str(home), "--yes", "--json"],
    )
    assert installed.exit_code == 0
    installed_payload = json.loads(installed.stdout)
    assert installed_payload["action"] == "install"

    inspected = runner.invoke(
        app,
        ["adapters", "inspect", "codex", "--home", str(home), "--json"],
    )
    assert inspected.exit_code == 0
    assert json.loads(inspected.stdout)["status"] == "current"

    replay = runner.invoke(
        app,
        ["adapters", "install", "codex", "--home", str(home), "--yes", "--json"],
    )
    assert replay.exit_code == 0
    assert json.loads(replay.stdout)["replayed"] is True


def test_pointer_adapter_uses_project_change_set_and_preserves_user_content(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    pointer = project.path / "AGENTS.md"
    pointer.write_text("# User instructions\nKeep these.\n", encoding="utf-8")

    install = plan_pointer_adapter(project.path, "codex", "install")
    assert install.risk == "semantic"
    assert install.approval_required is True
    with pytest.raises(ProjectApplyError, match="approval"):
        apply_change_set(project.path, install)
    receipt = apply_change_set(project.path, install, approved=True)
    installed = pointer.read_text(encoding="utf-8")
    assert installed.startswith("# User instructions\nKeep these.\n")
    assert "liner-maintenance-pointer:start" in installed
    assert receipt.operations[0]["environment"] == "codex"

    remove = plan_pointer_adapter(project.path, "codex", "remove")
    apply_change_set(project.path, remove, approved=True)
    assert pointer.read_text(encoding="utf-8") == "# User instructions\nKeep these.\n"


def test_pointer_adapter_refuses_changed_block_and_wrong_allowlisted_file(
    tmp_path: Path,
) -> None:
    project = init_project(tmp_path / "demo")
    install = plan_pointer_adapter(project.path, "claude", "install")
    apply_change_set(project.path, install, approved=True)
    pointer = project.path / "CLAUDE.md"
    pointer.write_text(
        pointer.read_text(encoding="utf-8").replace(
            "Run the installed Liner CLI", "Edit tape.yaml directly"
        ),
        encoding="utf-8",
    )

    with pytest.raises(Exception, match="managed pointer block changed"):
        plan_pointer_adapter(project.path, "claude", "update")


def test_cli_pointer_command_compiles_canonical_project_change_set(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")

    result = runner.invoke(
        app,
        [
            "project",
            "pointer",
            str(project.path),
            "--environment",
            "codex",
            "--action",
            "install",
            "--json",
        ],
    )

    assert result.exit_code == 0
    payload = json.loads(result.stdout)
    assert payload["contract"] == "liner.project_change_set"
    assert payload["operations"][0]["type"] == "pointer.adapter"
    assert payload["operations"][0]["file"] == "AGENTS.md"


def test_adapter_rejects_forged_plan_and_precreated_receipt(
    tmp_path: Path,
) -> None:
    import liner.agent_adapters as adapters

    home = tmp_path / "home"
    plan = plan_agent_adapter("install", "codex", home)
    forged = replace(plan, target=home / ".codex" / "skills" / "other")
    with pytest.raises(AdapterError, match="not canonical"):
        apply_agent_adapter_plan(forged, approved=True)
    assert not forged.target.exists()

    receipt_id = adapters._stable_id(f"receipt:{plan.plan_id}")
    receipt_path = home / ".liner" / "adapter-receipts" / f"{receipt_id}.json"
    receipt_path.parent.mkdir(parents=True)
    forged_receipt = adapters.AdapterReceipt(
        receipt_id=receipt_id,
        plan_id=plan.plan_id,
        action=plan.action,
        environment=plan.environment,
        liner_version=plan.liner_version,
        target=str(plan.target),
        file_effects=plan.file_effects,
        receipt_path=str(receipt_path),
        plan_hash=adapters._hash_payload(plan.to_dict()),
    )
    receipt_path.write_text(
        json.dumps(forged_receipt.to_dict()),
        encoding="utf-8",
    )
    with pytest.raises(AdapterError, match="effects are not current"):
        apply_agent_adapter_plan(plan, approved=True)
    assert not plan.target.exists()

    receipt_path.unlink()
    apply_agent_adapter_plan(plan, approved=True)
    remove = plan_agent_adapter("remove", "codex", home)
    forged_remove = replace(remove, target=home / "not-the-adapter")
    forged_remove_receipt_id = adapters._stable_id(f"receipt:{forged_remove.plan_id}")
    forged_remove_receipt_path = (
        home / ".liner" / "adapter-receipts" / f"{forged_remove_receipt_id}.json"
    )
    forged_remove_receipt = adapters.AdapterReceipt(
        receipt_id=forged_remove_receipt_id,
        plan_id=forged_remove.plan_id,
        action=forged_remove.action,
        environment=forged_remove.environment,
        liner_version=forged_remove.liner_version,
        target=str(forged_remove.target),
        file_effects=forged_remove.file_effects,
        receipt_path=str(forged_remove_receipt_path),
        plan_hash=adapters._hash_payload(forged_remove.to_dict()),
    )
    forged_remove_receipt_path.write_text(
        json.dumps(forged_remove_receipt.to_dict()),
        encoding="utf-8",
    )
    with pytest.raises(AdapterError, match="target is not canonical"):
        apply_agent_adapter_plan(forged_remove, approved=True)
    assert inspect_agent_adapter("codex", home)["status"] == "current"


def test_adapter_exact_plan_replay_and_current_update_are_idempotent(tmp_path: Path) -> None:
    home = tmp_path / "home"
    install = plan_agent_adapter("install", "claude", home)
    first = apply_agent_adapter_plan(install, approved=True)
    replay = apply_agent_adapter_plan(install, approved=True)
    assert replay.receipt_id == first.receipt_id
    assert replay.replayed is True

    state_path = install.target / ".liner-managed.json"
    state_before = state_path.read_bytes()
    update = plan_agent_adapter("update", "claude", home)
    assert update.approval_required is False
    update_receipt = apply_agent_adapter_plan(update)
    assert update_receipt.replayed is True
    assert state_path.read_bytes() == state_before
    assert apply_agent_adapter_plan(update).receipt_id == update_receipt.receipt_id

    forged_replay = replace(
        install,
        status="forged",
        compatibility="incompatible",
        approval_required=False,
        expected_state_hash="forged",
    )
    with pytest.raises(AdapterError, match="incompatible|identity"):
        apply_agent_adapter_plan(forged_replay, approved=True)


def test_adapter_refuses_receipt_symlink_and_preserves_original_on_publish_failure(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    import liner.agent_adapters as adapters

    home = tmp_path / "home"
    outside = tmp_path / "outside"
    outside.mkdir()
    home.mkdir()
    (home / ".liner").symlink_to(outside, target_is_directory=True)
    plan = plan_agent_adapter("install", "codex", home)
    with pytest.raises(AdapterError, match="receipt path"):
        apply_agent_adapter_plan(plan, approved=True)
    assert not list(outside.iterdir())

    (home / ".liner").unlink()
    apply_agent_adapter_plan(plan, approved=True)
    before = {
        path.relative_to(plan.target).as_posix(): path.read_bytes()
        for path in plan.target.rglob("*")
        if path.is_file()
    }
    remove = plan_agent_adapter("remove", "codex", home)

    def fail_publish(target: Path, staging: Path, backup: Path) -> None:
        raise OSError("injected publish failure")

    monkeypatch.setattr(adapters, "_publish_staged_tree", fail_publish)
    with pytest.raises(AdapterError, match="could not be published"):
        apply_agent_adapter_plan(remove, approved=True)
    after = {
        path.relative_to(plan.target).as_posix(): path.read_bytes()
        for path in plan.target.rglob("*")
        if path.is_file()
    }
    assert after == before
    assert inspect_agent_adapter("codex", home)["status"] == "current"


def test_adapter_recovers_receipt_after_post_publish_write_failure(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    import liner.agent_adapters as adapters

    home = tmp_path / "home"
    install = plan_agent_adapter("install", "codex", home)
    apply_agent_adapter_plan(install, approved=True)
    remove = plan_agent_adapter("remove", "codex", home)
    real_atomic_write = adapters._atomic_write
    failed = False

    def fail_receipt_once(path: Path, content: str) -> None:
        nonlocal failed
        if "adapter-receipts" in path.parts and not failed:
            failed = True
            raise OSError("injected receipt failure")
        real_atomic_write(path, content)

    monkeypatch.setattr(adapters, "_atomic_write", fail_receipt_once)
    with pytest.raises(AdapterError, match="receipt could not be written"):
        apply_agent_adapter_plan(remove, approved=True)
    assert inspect_agent_adapter("codex", home)["status"] == "absent"

    monkeypatch.setattr(adapters, "_atomic_write", real_atomic_write)
    recovered = apply_agent_adapter_plan(remove, approved=True)
    assert recovered.replayed is True
    assert Path(recovered.receipt_path).is_file()


def test_preservable_target_preview_distinguishes_create_from_write(tmp_path: Path) -> None:
    home = tmp_path / "home"
    target = home / ".claude" / "skills" / "liner-maintenance"
    target.mkdir(parents=True)
    (target / "notes.txt").write_text("keep\n", encoding="utf-8")

    plan = plan_agent_adapter("install", "claude", home)

    skill_path = str(target / "SKILL.md")
    assert skill_path in plan.file_effects["create"]
    assert skill_path not in plan.file_effects["write"]


def test_adapter_corrupt_state_and_duplicate_marker_fail_closed(tmp_path: Path) -> None:
    home = tmp_path / "custom home"
    install = plan_agent_adapter("install", "codex", home)
    apply_agent_adapter_plan(install, approved=True)
    state = install.target / ".liner-managed.json"
    state.write_bytes(b"\xff")

    inspection = inspect_agent_adapter("codex", home)
    assert inspection["status"] == "modified"
    assert inspection["compatibility"] == "incompatible"
    with pytest.raises(AdapterError, match="incompatible") as error:
        plan_agent_adapter("update", "codex", home)
    assert f"--home '{home}'" in str(error.value)

    other_home = tmp_path / "other-home"
    other = plan_agent_adapter("install", "claude", other_home)
    apply_agent_adapter_plan(other, approved=True)
    skill = other.target / "SKILL.md"
    skill.write_text(
        skill.read_text(encoding="utf-8")
        + "\n<!-- liner-maintenance-skill:end -->\n",
        encoding="utf-8",
    )
    with pytest.raises(AdapterError, match="managed content changed"):
        plan_agent_adapter("update", "claude", other_home)


def test_pointer_remove_preserves_content_appended_after_managed_block(tmp_path: Path) -> None:
    project = init_project(tmp_path / "demo")
    pointer = project.path / "AGENTS.md"
    pointer.write_text("BEFORE", encoding="utf-8")
    install = plan_pointer_adapter(project.path, "codex", "install")
    apply_change_set(project.path, install, approved=True)
    pointer.write_text(pointer.read_text(encoding="utf-8") + "AFTER\n", encoding="utf-8")

    remove = plan_pointer_adapter(project.path, "codex", "remove")
    apply_change_set(project.path, remove, approved=True)

    assert pointer.read_text(encoding="utf-8") == "BEFORE\nAFTER\n"
