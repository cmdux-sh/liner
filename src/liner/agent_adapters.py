from __future__ import annotations

import hashlib
import json
import os
import re
import shlex
import shutil
from collections.abc import Iterator
from contextlib import contextmanager, suppress
from dataclasses import dataclass, replace
from importlib import resources
from pathlib import Path
from typing import Any
from uuid import NAMESPACE_URL, uuid5

from liner import __version__
from liner.library import build_catalog, default_library_root

ADAPTER_PLAN_CONTRACT = "liner.agent_adapter_plan"
ADAPTER_RECEIPT_CONTRACT = "liner.agent_adapter_receipt"
ADAPTER_STATE_CONTRACT = "liner.agent_adapter_state"
ADAPTER_CONTRACT_VERSION = 1
SKILL_NAME = "liner-maintenance"
ROUTER_SKILL_NAME = "liner"
SUPPORTED_ENVIRONMENTS = {
    "codex": Path(".codex/skills"),
    "claude": Path(".claude/skills"),
}
_DESCRIPTION_SLOT = "{{DESCRIPTION}}"
# Agent Skills caps a description at 1024 characters.
_MAX_DESCRIPTION_CHARS = 1024
_MAX_SUBJECT_CHARS = 60
_ROUTER_INTRO = (
    "Find the user's local research liners, suggest the right one, and build new ones. Use "
    "when the user mentions Liner or a liner, asks to build one, asks what research they "
    "already have, or works on "
)
_NO_SUBJECTS = "something that could use earlier research."
_UNSAFE_SUBJECT_CHARS = re.compile(r"[^\w\s&+/.,'()-]", re.UNICODE)


class AdapterError(RuntimeError):
    """An optional agent adapter operation was refused without unsafe fallback."""


@dataclass(frozen=True, slots=True)
class AdapterPlan:
    plan_id: str
    action: str
    environment: str
    home: Path
    target: Path
    liner_version: str
    status: str
    compatibility: str
    approval_required: bool
    file_effects: dict[str, list[str]]
    expected_state_hash: str
    skill: str = SKILL_NAME
    description: str | None = None

    def to_dict(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "contract": ADAPTER_PLAN_CONTRACT,
            "version": ADAPTER_CONTRACT_VERSION,
            "plan_id": self.plan_id,
            "action": self.action,
            "environment": self.environment,
            "home": str(self.home),
            "target": str(self.target),
            "liner_version": self.liner_version,
            "status": self.status,
            "compatibility": self.compatibility,
            "approval_required": self.approval_required,
            "file_effects": self.file_effects,
            "expected_state_hash": self.expected_state_hash,
            "remediation": _remediation(self.environment, self.home, self.skill),
        }
        return {**payload, **_plan_extras(self)}


@dataclass(frozen=True, slots=True)
class AdapterReceipt:
    receipt_id: str
    plan_id: str
    action: str
    environment: str
    liner_version: str
    target: str
    file_effects: dict[str, list[str]]
    receipt_path: str
    plan_hash: str
    replayed: bool = False

    def to_dict(self) -> dict[str, Any]:
        payload = {
            "contract": ADAPTER_RECEIPT_CONTRACT,
            "version": ADAPTER_CONTRACT_VERSION,
            "receipt_id": self.receipt_id,
            "plan_id": self.plan_id,
            "action": self.action,
            "environment": self.environment,
            "liner_version": self.liner_version,
            "target": self.target,
            "file_effects": self.file_effects,
            "receipt_path": self.receipt_path,
            "plan_hash": self.plan_hash,
            "replayed": self.replayed,
        }
        return {**payload, "receipt_hash": _hash_payload(payload)}


def inspect_agent_adapter(
    environment: str, home: Path, *, skill: str = SKILL_NAME
) -> dict[str, Any]:
    environment = _environment(environment)
    home = _home(home)
    target = _target(home, environment, skill)
    inspection = _inspect_target(environment, target, skill)
    return {
        "contract": "liner.agent_adapter_inspection",
        "version": ADAPTER_CONTRACT_VERSION,
        "environment": environment,
        "home": str(home),
        "target": str(target),
        "liner_version": __version__,
        "status": inspection["status"],
        "compatibility": inspection["compatibility"],
        "managed_files": inspection["managed_files"],
        "remediation": _remediation(environment, home, skill),
        **_skill_field(skill),
    }


def plan_agent_adapter(
    action: str, environment: str, home: Path, *, skill: str = SKILL_NAME
) -> AdapterPlan:
    if action not in {"install", "update", "remove"}:
        raise AdapterError("Adapter action must be install, update, or remove.")
    environment = _environment(environment)
    home = _home(home)
    target = _target(home, environment, skill)
    inspection = _inspect_target(environment, target, skill)
    status = str(inspection["status"])
    compatibility = str(inspection["compatibility"])
    if compatibility == "incompatible":
        raise AdapterError(
            "Installed adapter state is incompatible. " + _remediation(environment, home, skill)
        )
    if status in {"unmanaged", "modified"}:
        noun = "managed content changed" if status == "modified" else "target is unmanaged"
        raise AdapterError(
            f"Adapter {noun}; Liner will not overwrite {target}. "
            "Review it and move that target aside, then run "
            f"`{_command(skill)} install {environment} --yes --home {shlex.quote(str(home))}`."
        )
    if action == "update" and status in {"absent", "preservable"}:
        raise AdapterError(f"Adapter is not installed. {_remediation(environment, home, skill)}")
    if action == "remove" and status in {"absent", "preservable"}:
        raise AdapterError("Adapter is already absent; no removal is needed.")
    replay = action in {"install", "update"} and status == "current"
    target_files = _target_files(target)
    receipt_effect = str(_receipt_path(home, "<receipt-id>"))
    if replay:
        effects = {**_empty_effects(), "create": [receipt_effect]}
    elif action == "install" and status == "preservable":
        skill_path, openai_path, state_path = target_files
        effects = {
            **_empty_effects(),
            "create": [
                *([] if inspection.get("skill_exists") else [skill_path]),
                openai_path,
                state_path,
                receipt_effect,
            ],
            "write": [skill_path] if inspection.get("skill_exists") else [],
        }
    elif action == "install":
        effects = {**_empty_effects(), "create": [*target_files, receipt_effect]}
    elif action == "update":
        effects = {
            **_empty_effects(),
            "create": [receipt_effect],
            "write": target_files,
        }
    else:
        skill_path, openai_path, state_path = target_files
        effects = {
            **_empty_effects(),
            "create": [receipt_effect],
            "write": [skill_path] if inspection.get("has_unmanaged_suffix") else [],
            "delete": [openai_path, state_path]
            if inspection.get("has_unmanaged_suffix")
            else target_files,
        }
    plan = AdapterPlan(
        plan_id="",
        action=action,
        environment=environment,
        home=home,
        target=target,
        liner_version=__version__,
        status=status,
        compatibility=compatibility,
        approval_required=not replay,
        file_effects=effects,
        expected_state_hash=str(inspection["state_hash"]),
        skill=skill,
        description=(
            _router_description()
            if skill == ROUTER_SKILL_NAME and action in {"install", "update"}
            else None
        ),
    )
    return replace(
        plan,
        plan_id=_stable_id(f"plan:{_hash_payload(_plan_identity_payload(plan))}"),
    )


def _plan_identity_payload(plan: AdapterPlan) -> dict[str, Any]:
    return {
        "action": plan.action,
        "environment": plan.environment,
        "home": str(plan.home),
        "target": str(plan.target),
        "liner_version": plan.liner_version,
        "status": plan.status,
        "compatibility": plan.compatibility,
        "approval_required": plan.approval_required,
        "file_effects": plan.file_effects,
        "expected_state_hash": plan.expected_state_hash,
        **_plan_extras(plan),
    }


def _validate_plan_structure(plan: AdapterPlan) -> None:
    if plan.action not in {"install", "update", "remove"}:
        raise AdapterError("Adapter plan has an unsupported action.")
    environment = _environment(plan.environment)
    home = _home(plan.home)
    target = _target(home, environment, plan.skill)
    if environment != plan.environment or home != plan.home or target != plan.target:
        raise AdapterError("Adapter plan target is not canonical for its environment and home.")
    if plan.liner_version != __version__ or plan.compatibility != "compatible":
        raise AdapterError("Adapter plan is incompatible with this Liner build.")
    if plan.status not in {"absent", "preservable", "current", "update_required"}:
        raise AdapterError("Adapter plan has an invalid inspection status.")
    replay = plan.action in {"install", "update"} and plan.status == "current"
    if plan.approval_required != (not replay):
        raise AdapterError("Adapter plan approval state is not canonical.")
    expected_plan_id = _stable_id(f"plan:{_hash_payload(_plan_identity_payload(plan))}")
    if plan.plan_id != expected_plan_id:
        raise AdapterError("Adapter plan identity does not match its approved fields.")

    skill_path, openai_path, state_path = _target_files(target)
    receipt_path = str(_receipt_path(home, "<receipt-id>"))
    effects = plan.file_effects
    if set(effects) != {"create", "write", "delete"} or not all(
        isinstance(paths, list) and all(isinstance(path, str) for path in paths)
        for paths in effects.values()
    ):
        raise AdapterError("Adapter plan file effects are invalid.")
    if len({path for paths in effects.values() for path in paths}) != sum(
        len(paths) for paths in effects.values()
    ):
        raise AdapterError("Adapter plan file effects overlap or contain duplicates.")
    if replay:
        valid_effects = effects == {"create": [receipt_path], "write": [], "delete": []}
    elif plan.action == "install" and plan.status == "absent":
        valid_effects = effects == {
            "create": [skill_path, openai_path, state_path, receipt_path],
            "write": [],
            "delete": [],
        }
    elif plan.action == "install" and plan.status == "preservable":
        valid_effects = effects in (
            {
                "create": [openai_path, state_path, receipt_path],
                "write": [skill_path],
                "delete": [],
            },
            {
                "create": [skill_path, openai_path, state_path, receipt_path],
                "write": [],
                "delete": [],
            },
        )
    elif plan.action == "update" and plan.status == "update_required":
        valid_effects = effects == {
            "create": [receipt_path],
            "write": [skill_path, openai_path, state_path],
            "delete": [],
        }
    elif plan.action == "remove" and plan.status in {"current", "update_required"}:
        valid_effects = effects in (
            {
                "create": [receipt_path],
                "write": [skill_path],
                "delete": [openai_path, state_path],
            },
            {
                "create": [receipt_path],
                "write": [],
                "delete": [skill_path, openai_path, state_path],
            },
        )
    else:
        valid_effects = False
    if not valid_effects:
        raise AdapterError("Adapter plan file effects are not canonical for its action and state.")


def apply_agent_adapter_plan(
    plan: AdapterPlan, *, approved: bool = False
) -> AdapterReceipt:
    _validate_plan_structure(plan)
    receipt_id = _stable_id(f"receipt:{plan.plan_id}")
    receipt_path = _receipt_path(plan.home, receipt_id)
    pending_path = _pending_receipt_path(plan.target, receipt_id)
    _ensure_safe_receipt_path(plan.home, receipt_path)
    if receipt_path.is_file():
        return _validated_receipt_replay(plan, receipt_path)
    recovered = _recover_pending_receipt(plan, pending_path, receipt_path)
    if recovered is not None:
        return recovered

    canonical = plan_agent_adapter(plan.action, plan.environment, plan.home, skill=plan.skill)
    if plan != canonical:
        raise AdapterError("Adapter plan is not canonical or state changed; inspect and plan again.")
    if canonical.approval_required and not approved:
        raise AdapterError("Adapter approval is required before writes.")
    if not canonical.approval_required:
        return _write_replay_receipt(canonical, receipt_id, receipt_path)
    _ensure_safe_target(canonical.home, canonical.target)
    receipt = AdapterReceipt(
        receipt_id=receipt_id,
        plan_id=canonical.plan_id,
        action=canonical.action,
        environment=canonical.environment,
        liner_version=__version__,
        target=str(canonical.target),
        file_effects=canonical.file_effects,
        receipt_path=str(receipt_path),
        plan_hash=_hash_payload(canonical.to_dict()),
    )

    try:
        with _adapter_lock(canonical.home, canonical.target):
            fresh = plan_agent_adapter(
                canonical.action, canonical.environment, canonical.home, skill=canonical.skill
            )
            if fresh != canonical:
                raise AdapterError("Adapter state changed after planning; inspect and plan again.")
            _write_pending_receipt(pending_path, canonical, receipt)
            _publish_adapter_mutation(canonical, receipt_id)
    except AdapterError:
        raise
    except OSError as error:
        raise AdapterError(
            "Adapter transaction could not be published. Inspect the target; the previous "
            "managed tree was preserved or retained as a sibling backup."
        ) from error

    _ensure_safe_receipt_path(canonical.home, receipt_path, create=True)
    try:
        _atomic_write(receipt_path, json.dumps(receipt.to_dict(), indent=2) + "\n")
    except OSError as error:
        raise AdapterError(
            "Adapter state was published but its receipt could not be written. Inspect the "
            "adapter, repair the receipt directory, and reapply the exact same plan to recover."
        ) from error
    with suppress(FileNotFoundError):
        pending_path.unlink()
    return receipt


def _publish_adapter_mutation(plan: AdapterPlan, receipt_id: str) -> None:
    target = plan.target
    target.parent.mkdir(parents=True, exist_ok=True)
    _ensure_safe_target(plan.home, target)
    staging = target.with_name(f".{target.name}.{plan.plan_id}.stage")
    backup = target.with_name(f".{target.name}.{plan.plan_id}.backup")
    if staging.exists() or backup.exists():
        raise AdapterError("A previous adapter transaction needs manual cleanup before retrying.")
    if target.exists():
        shutil.copytree(target, staging)
    else:
        staging.mkdir()
    try:
        if plan.action == "install":
            _install_staged(
                staging,
                plan.environment,
                receipt_id,
                preserving=plan.status == "preservable",
                skill=plan.skill,
                description=plan.description,
            )
        elif plan.action == "update":
            _update_staged(staging, plan.environment, receipt_id, plan.skill, plan.description)
        else:
            _remove_staged(staging, plan.skill)
        _publish_staged_tree(target, staging, backup)
    finally:
        if staging.exists():
            shutil.rmtree(staging, ignore_errors=True)


def _install_staged(
    target: Path,
    environment: str,
    receipt_id: str,
    *,
    preserving: bool,
    skill: str,
    description: str | None,
) -> None:
    skill_path = target / "SKILL.md"
    preserved = skill_path.read_text(encoding="utf-8") if preserving and skill_path.is_file() else ""
    skill_text, openai_text = _bundle_text(skill, description)
    installed_skill = skill_text + preserved
    _atomic_write(skill_path, installed_skill)
    _atomic_write(target / "agents" / "openai.yaml", openai_text)
    _write_state(target, environment, installed_skill, openai_text, receipt_id, skill)


def _update_staged(
    target: Path, environment: str, receipt_id: str, skill: str, description: str | None
) -> None:
    inspection = _inspect_target(environment, target, skill)
    if inspection["status"] not in {"current", "update_required"}:
        raise AdapterError("Adapter managed content changed; inspect and reinstall explicitly.")
    installed = (target / "SKILL.md").read_text(encoding="utf-8")
    _, suffix = _split_managed_skill(installed, skill)
    source_skill, source_openai = _bundle_text(skill, description)
    source_prefix, _ = _split_managed_skill(source_skill, skill)
    updated_skill = source_prefix + suffix
    _atomic_write(target / "SKILL.md", updated_skill)
    _atomic_write(target / "agents" / "openai.yaml", source_openai)
    _write_state(target, environment, updated_skill, source_openai, receipt_id, skill)


def _remove_staged(target: Path, skill: str) -> None:
    state = _load_state(target)
    if state is None:
        raise AdapterError("Adapter managed state is missing; remove was refused.")
    skill_path = target / "SKILL.md"
    agents_path = target / "agents" / "openai.yaml"
    installed = skill_path.read_text(encoding="utf-8")
    prefix, suffix = _split_managed_skill(installed, skill)
    if _hash_text(prefix) != state.get("managed_prefix_hash"):
        raise AdapterError("Adapter managed content changed; remove was refused.")
    if _hash_file(agents_path) != state.get("openai_hash"):
        raise AdapterError("Adapter managed content changed; remove was refused.")
    if suffix:
        _atomic_write(skill_path, suffix)
    else:
        skill_path.unlink()
    agents_path.unlink()
    state_path = target / ".liner-managed.json"
    state_path.unlink()
    agents_dir = target / "agents"
    if agents_dir.is_dir() and not any(agents_dir.iterdir()):
        agents_dir.rmdir()
    if target.is_dir() and not any(target.iterdir()):
        target.rmdir()


def _publish_staged_tree(target: Path, staging: Path, backup: Path) -> None:
    had_target = target.exists()
    if had_target:
        os.replace(target, backup)
    try:
        if staging.exists():
            os.replace(staging, target)
    except Exception:
        if had_target and backup.exists() and not target.exists():
            os.replace(backup, target)
        raise
    if backup.exists():
        shutil.rmtree(backup, ignore_errors=True)


def _inspect_target(environment: str, target: Path, skill: str = SKILL_NAME) -> dict[str, Any]:
    if not target.exists():
        return {
            "status": "absent",
            "compatibility": "compatible",
            "managed_files": [],
            "state_hash": "absent",
            "skill_exists": False,
        }
    if target.is_symlink() or not target.is_dir():
        return {
            "status": "unmanaged",
            "compatibility": "incompatible",
            "managed_files": [],
            "state_hash": "unsafe",
            "skill_exists": False,
        }
    state = _load_state(target)
    if state is None:
        state_path = target / ".liner-managed.json"
        if state_path.exists():
            return {
                "status": "modified",
                "compatibility": "incompatible",
                "managed_files": [],
                "state_hash": "invalid-state",
                "skill_exists": (target / "SKILL.md").is_file(),
            }
        skill_path = target / "SKILL.md"
        openai_path = target / "agents" / "openai.yaml"
        if openai_path.exists() or (skill_path.exists() and not skill_path.is_file()):
            return {
                "status": "unmanaged",
                "compatibility": "compatible",
                "managed_files": [],
                "state_hash": "unmanaged",
                "skill_exists": skill_path.is_file(),
            }
        if skill_path.is_file() and not skill_path.is_symlink():
            try:
                text = skill_path.read_text(encoding="utf-8")
            except (OSError, UnicodeError):
                text = ""
                return {
                    "status": "unmanaged",
                    "compatibility": "incompatible",
                    "managed_files": [],
                    "state_hash": "unreadable",
                    "skill_exists": True,
                }
            if f"{skill}-skill:" in text:
                return {
                    "status": "modified",
                    "compatibility": "incompatible",
                    "managed_files": [],
                    "state_hash": _hash_text(text),
                    "skill_exists": True,
                }
        else:
            text = ""
        return {
            "status": "preservable",
            "compatibility": "compatible",
            "managed_files": [str(skill_path)] if text else [],
            "state_hash": _hash_text(text) if text else "preservable-empty",
            "skill_exists": skill_path.is_file(),
        }
    state_hash = _hash_payload(state)
    if (
        state.get("contract") != ADAPTER_STATE_CONTRACT
        or state.get("version") != ADAPTER_CONTRACT_VERSION
        or state.get("environment") != environment
    ):
        return {
            "status": "modified",
            "compatibility": "incompatible",
            "managed_files": [],
            "state_hash": state_hash,
            "skill_exists": (target / "SKILL.md").is_file(),
        }
    skill_path = target / "SKILL.md"
    openai_path = target / "agents" / "openai.yaml"
    suffix = ""
    try:
        prefix, suffix = _split_managed_skill(skill_path.read_text(encoding="utf-8"), skill)
    except (OSError, UnicodeError, AdapterError):
        prefix = ""
    if (
        _hash_text(prefix) != state.get("managed_prefix_hash")
        or _hash_file(openai_path) != state.get("openai_hash")
    ):
        status = "modified"
    else:
        source_skill, source_openai = _bundle_text(skill)
        source_prefix, _ = _split_managed_skill(source_skill, skill)
        status = (
            "current"
            if state.get("liner_version") == __version__
            and _hash_text(prefix) == _hash_text(source_prefix)
            and _hash_text(source_openai) == state.get("openai_hash")
            else "update_required"
        )
    return {
        "status": status,
        "compatibility": "compatible",
        "managed_files": [str(skill_path), str(openai_path), str(target / ".liner-managed.json")],
        "state_hash": state_hash,
        "state": state,
        "has_unmanaged_suffix": bool(suffix),
        "skill_exists": skill_path.is_file(),
    }


def _write_state(
    target: Path,
    environment: str,
    skill_text: str,
    openai_text: str,
    receipt_id: str,
    skill: str,
) -> None:
    prefix, _ = _split_managed_skill(skill_text, skill)
    state = {
        "contract": ADAPTER_STATE_CONTRACT,
        "version": ADAPTER_CONTRACT_VERSION,
        "environment": environment,
        "liner_version": __version__,
        "managed_prefix_hash": _hash_text(prefix),
        "openai_hash": _hash_text(openai_text),
        "last_receipt_id": receipt_id,
    }
    _atomic_write(target / ".liner-managed.json", json.dumps(state, indent=2) + "\n")


def _write_replay_receipt(
    plan: AdapterPlan,
    receipt_id: str,
    receipt_path: Path,
) -> AdapterReceipt:
    receipt = AdapterReceipt(
        receipt_id=receipt_id,
        plan_id=plan.plan_id,
        action=plan.action,
        environment=plan.environment,
        liner_version=__version__,
        target=str(plan.target),
        file_effects=plan.file_effects,
        receipt_path=str(receipt_path),
        plan_hash=_hash_payload(plan.to_dict()),
        replayed=True,
    )
    _ensure_safe_receipt_path(plan.home, receipt_path, create=True)
    try:
        _atomic_write(receipt_path, json.dumps(receipt.to_dict(), indent=2) + "\n")
    except OSError as error:
        raise AdapterError("Adapter replay receipt could not be written; no adapter files changed.") from error
    return receipt


def _pending_receipt_path(target: Path, receipt_id: str) -> Path:
    return target.parent / f".{target.name}.{receipt_id}.pending.json"


def _write_pending_receipt(
    path: Path,
    plan: AdapterPlan,
    receipt: AdapterReceipt,
) -> None:
    if path.is_symlink() or (path.exists() and (not path.is_file() or path.stat().st_nlink > 1)):
        raise AdapterError("Adapter pending receipt path is linked or not a regular file.")
    payload = {
        "contract": "liner.agent_adapter_pending_receipt",
        "version": ADAPTER_CONTRACT_VERSION,
        "plan": plan.to_dict(),
        "receipt": receipt.to_dict(),
    }
    _atomic_write(path, json.dumps(payload, indent=2) + "\n")


def _recover_pending_receipt(
    plan: AdapterPlan,
    pending_path: Path,
    receipt_path: Path,
) -> AdapterReceipt | None:
    if not pending_path.exists():
        return None
    _ensure_safe_target(plan.home, plan.target)
    if (
        pending_path.is_symlink()
        or not pending_path.is_file()
        or pending_path.stat().st_nlink > 1
    ):
        raise AdapterError("Adapter pending receipt is linked or invalid; recovery was refused.")
    try:
        raw = json.loads(pending_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        raise AdapterError("Adapter pending receipt is unreadable; recovery was refused.") from error
    if (
        not isinstance(raw, dict)
        or raw.get("contract") != "liner.agent_adapter_pending_receipt"
        or raw.get("version") != ADAPTER_CONTRACT_VERSION
        or raw.get("plan") != plan.to_dict()
        or not isinstance(raw.get("receipt"), dict)
    ):
        raise AdapterError("Adapter pending receipt does not match the supplied plan.")
    receipt = _receipt_from_dict(dict(raw["receipt"]), replayed=True)
    if (
        receipt.receipt_id != _stable_id(f"receipt:{plan.plan_id}")
        or receipt.plan_id != plan.plan_id
        or receipt.action != plan.action
        or receipt.environment != plan.environment
        or receipt.liner_version != plan.liner_version
        or receipt.target != str(plan.target)
        or receipt.file_effects != plan.file_effects
        or receipt.receipt_path != str(receipt_path)
        or receipt.plan_hash != _hash_payload(plan.to_dict())
        or bool(dict(raw["receipt"]).get("replayed"))
    ):
        raise AdapterError("Adapter pending receipt does not match the supplied plan.")
    if not _adapter_effects_are_published(plan):
        return None
    _ensure_safe_receipt_path(plan.home, receipt_path, create=True)
    try:
        _atomic_write(receipt_path, json.dumps(dict(raw["receipt"]), indent=2) + "\n")
    except OSError as error:
        raise AdapterError(
            "Adapter effects are published, but pending receipt recovery could not write the "
            "canonical receipt. Repair the receipt directory and reapply the same plan."
        ) from error
    pending_path.unlink()
    return receipt


def _adapter_effects_are_published(plan: AdapterPlan) -> bool:
    inspection = _inspect_target(plan.environment, plan.target, plan.skill)
    if plan.action in {"install", "update"}:
        return inspection.get("status") == "current"
    return inspection.get("status") in {"absent", "preservable"}


def _validated_receipt_replay(plan: AdapterPlan, receipt_path: Path) -> AdapterReceipt:
    try:
        raw = json.loads(receipt_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        raise AdapterError("Adapter receipt is unreadable or corrupt; mutation was refused.") from error
    if not isinstance(raw, dict):
        raise AdapterError("Adapter receipt is invalid; mutation was refused.")
    receipt = _receipt_from_dict(raw, replayed=True)
    expected = {
        "contract": ADAPTER_RECEIPT_CONTRACT,
        "version": ADAPTER_CONTRACT_VERSION,
        "receipt_id": _stable_id(f"receipt:{plan.plan_id}"),
        "plan_id": plan.plan_id,
        "action": plan.action,
        "environment": plan.environment,
        "liner_version": plan.liner_version,
        "target": str(plan.target),
        "file_effects": plan.file_effects,
        "receipt_path": str(receipt_path),
        "plan_hash": _hash_payload(plan.to_dict()),
        "replayed": not plan.approval_required,
    }
    if any(raw.get(key) != value for key, value in expected.items()):
        raise AdapterError("Adapter receipt does not match the approved plan; mutation was refused.")
    inspection = _inspect_target(plan.environment, plan.target, plan.skill)
    if plan.action in {"install", "update"}:
        if inspection.get("status") != "current":
            raise AdapterError("Adapter receipt exists but its installed effects are not current.")
        state = inspection.get("state")
        if plan.approval_required and (
            not isinstance(state, dict)
            or state.get("last_receipt_id") != receipt.receipt_id
        ):
            raise AdapterError("Adapter receipt is not bound to the installed managed state.")
    elif inspection.get("status") not in {"absent", "preservable"}:
        raise AdapterError("Adapter removal receipt exists but managed content is still present.")
    return receipt


def _receipt_from_dict(raw: dict[str, Any], *, replayed: bool) -> AdapterReceipt:
    try:
        if (
            raw.get("contract") != ADAPTER_RECEIPT_CONTRACT
            or raw.get("version") != ADAPTER_CONTRACT_VERSION
        ):
            raise ValueError
        supplied_hash = raw.get("receipt_hash")
        unsigned = dict(raw)
        unsigned.pop("receipt_hash", None)
        if supplied_hash != _hash_payload(unsigned):
            raise ValueError
        effects = raw["file_effects"]
        if not isinstance(effects, dict):
            raise TypeError
        normalized_effects = {
            str(key): [str(item) for item in value]
            for key, value in effects.items()
            if isinstance(value, list)
        }
        if len(normalized_effects) != len(effects):
            raise TypeError
        return AdapterReceipt(
            receipt_id=str(raw["receipt_id"]),
            plan_id=str(raw["plan_id"]),
            action=str(raw["action"]),
            environment=str(raw["environment"]),
            liner_version=str(raw["liner_version"]),
            target=str(raw["target"]),
            file_effects=normalized_effects,
            receipt_path=str(raw["receipt_path"]),
            plan_hash=str(raw["plan_hash"]),
            replayed=replayed,
        )
    except (KeyError, TypeError, ValueError) as error:
        raise AdapterError("Adapter receipt is invalid; mutation was refused.") from error


def _bundle_text(skill: str = SKILL_NAME, description: str | None = None) -> tuple[str, str]:
    root = resources.files("liner").joinpath("bundled", skill)
    skill_text = root.joinpath("SKILL.md").read_text(encoding="utf-8")
    if skill == ROUTER_SKILL_NAME:
        skill_text = skill_text.replace(
            _DESCRIPTION_SLOT,
            json.dumps(description or _router_description(), ensure_ascii=False),
        )
    return skill_text, root.joinpath("agents", "openai.yaml").read_text(encoding="utf-8")


def _router_description() -> str:
    """Name each usable liner, so the skill fires while working on one of those subjects."""
    catalog = build_catalog(default_library_root())
    subjects = [
        subject for entry in catalog["liners"] if entry["usable"] and (subject := _subject(entry))
    ]
    for count in range(len(subjects), 0, -1):
        left = len(subjects) - count
        ending = f"; and {left} more." if left else "."
        text = f"{_ROUTER_INTRO}these subjects: {'; '.join(subjects[:count])}{ending}"
        if len(text) <= _MAX_DESCRIPTION_CHARS:
            return text
    return _ROUTER_INTRO + _NO_SUBJECTS


def _subject(entry: dict[str, Any]) -> str:
    """Reduce a title to plain words, since it lands in text the agent always reads."""
    title = str(entry["title"])
    if title == entry["slug"]:
        title = title.replace("-", " ")
    words = " ".join(_UNSAFE_SUBJECT_CHARS.sub("", title).split())
    return re.sub(r"-{2,}", "-", words)[:_MAX_SUBJECT_CHARS].strip()


def _split_managed_skill(text: str, skill: str = SKILL_NAME) -> tuple[str, str]:
    managed_start = f"<!-- {skill}-skill:start"
    managed_end = f"<!-- {skill}-skill:end -->"
    if text.count(managed_start) != 1 or text.count(managed_end) != 1:
        raise AdapterError("Adapter managed content changed; markers are missing or duplicated.")
    marker = text.find(managed_end)
    if marker < 0:
        raise AdapterError("Adapter managed content changed; end marker is missing.")
    end = marker + len(managed_end)
    if text[end : end + 1] == "\n":
        end += 1
    prefix = text[:end]
    suffix = text[end:]
    return prefix, suffix


def _load_state(target: Path) -> dict[str, Any] | None:
    path = target / ".liner-managed.json"
    if not path.is_file() or path.is_symlink():
        return None
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return None
    return raw if isinstance(raw, dict) else None


def _target(home: Path, environment: str, skill: str) -> Path:
    return home / SUPPORTED_ENVIRONMENTS[environment] / skill


def _skill_field(skill: str) -> dict[str, str]:
    """Only the router names itself, so maintenance plans and receipts keep their shape."""
    return {} if skill == SKILL_NAME else {"skill": skill}


def _plan_extras(plan: AdapterPlan) -> dict[str, str]:
    """Bind the router's text to the plan: an agent reads it every session."""
    description = {"description": plan.description} if plan.description else {}
    return {**_skill_field(plan.skill), **description}


def _command(skill: str) -> str:
    return "liner adapters" if skill == SKILL_NAME else "liner agent-skill"


def _environment(value: str) -> str:
    normalized = value.strip().lower()
    if normalized not in SUPPORTED_ENVIRONMENTS:
        supported = ", ".join(sorted(SUPPORTED_ENVIRONMENTS))
        raise AdapterError(f"Supported environments: {supported}.")
    return normalized


def _home(path: Path) -> Path:
    resolved = path.expanduser().resolve()
    if resolved == Path(resolved.anchor):
        raise AdapterError("Adapter home must not be a filesystem root.")
    return resolved


def _ensure_safe_target(home: Path, target: Path) -> None:
    home.mkdir(parents=True, exist_ok=True)
    current = target
    while current != home:
        if current.is_symlink():
            raise AdapterError(f"Unsafe adapter target: {current} is a symbolic link.")
        current = current.parent
    if target.is_dir():
        for path in target.rglob("*"):
            if path.is_symlink() or (path.is_file() and path.stat().st_nlink > 1):
                raise AdapterError(f"Unsafe adapter target: {path} is linked or aliased.")


def _ensure_safe_receipt_path(home: Path, path: Path, *, create: bool = False) -> None:
    if path != _receipt_path(home, path.stem):
        raise AdapterError("Adapter receipt path is outside the canonical receipt directory.")
    current = path
    while current != home:
        if current.is_symlink():
            raise AdapterError(f"Unsafe adapter receipt path: {current} is a symbolic link.")
        if current == current.parent:
            raise AdapterError("Adapter receipt path is outside the approved home.")
        current = current.parent
    if create:
        path.parent.mkdir(parents=True, exist_ok=True)
        current = path.parent
        while current != home:
            if current.is_symlink():
                raise AdapterError(f"Unsafe adapter receipt path: {current} is a symbolic link.")
            current = current.parent
    if path.exists() and (not path.is_file() or path.stat().st_nlink > 1):
        raise AdapterError("Adapter receipt path is linked, aliased, or not a regular file.")


@contextmanager
def _adapter_lock(home: Path, target: Path) -> Iterator[None]:
    target.parent.mkdir(parents=True, exist_ok=True)
    _ensure_safe_target(home, target)
    lock = target.with_name(f".{target.name}.liner.lock")
    try:
        descriptor = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    except FileExistsError as error:
        raise AdapterError("Another adapter operation is already in progress; retry later.") from error
    try:
        os.write(descriptor, f"{os.getpid()}\n".encode())
        yield
    finally:
        os.close(descriptor)
        with suppress(FileNotFoundError):
            lock.unlink()


def _atomic_write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    temporary.write_text(content, encoding="utf-8")
    os.replace(temporary, path)


def _target_files(target: Path) -> list[str]:
    return [
        str(target / "SKILL.md"),
        str(target / "agents" / "openai.yaml"),
        str(target / ".liner-managed.json"),
    ]


def _empty_effects() -> dict[str, list[str]]:
    return {"create": [], "write": [], "delete": []}


def _receipt_path(home: Path, receipt_id: str) -> Path:
    return home / ".liner" / "adapter-receipts" / f"{receipt_id}.json"


def _remediation(
    environment: str, home: Path | None = None, skill: str = SKILL_NAME
) -> str:
    home_option = f" --home {shlex.quote(str(home))}" if home is not None else ""
    command = _command(skill)
    return (
        f"Run `{command} inspect {environment}{home_option}`. If managed content is current, "
        f"use `{command} remove {environment} --yes{home_option}`. If it is changed or "
        "incompatible, review and move the reported target aside before running "
        f"`{command} install {environment} --yes{home_option}`."
    )


def _hash_file(path: Path) -> str:
    if not path.is_file() or path.is_symlink():
        return "missing"
    try:
        return _hash_text(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError):
        return "unreadable"


def _hash_text(text: str) -> str:
    return f"sha256:{hashlib.sha256(text.encode('utf-8')).hexdigest()}"


def _hash_payload(payload: Any) -> str:
    encoded = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return hashlib.sha256(encoded.encode("utf-8")).hexdigest()


def _stable_id(seed: str) -> str:
    return str(uuid5(NAMESPACE_URL, f"https://liner.dev/adapter/{seed}"))
