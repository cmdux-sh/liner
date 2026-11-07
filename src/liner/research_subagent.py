"""Install the restricted subagent that reads fetched pages. Claude Code only for now."""

from __future__ import annotations

import os
import re
from importlib import resources
from pathlib import Path
from typing import Any

SUBAGENT_CONTRACT = "liner.research_subagent"
CONTRACT_VERSION = 1
SUBAGENT_NAME = "liner-researcher"

_MARKER = re.compile(r"^<!-- liner-researcher v\d+ -->$", re.MULTILINE)


class SubagentError(RuntimeError):
    """The subagent could not be changed safely."""


def inspect_subagent(home: Path) -> dict[str, Any]:
    target = _target(home)
    if target.parent.is_symlink():
        status = "foreign"
    elif not target.exists() and not target.is_symlink():
        status = "missing"
    elif (
        target.is_symlink()
        or target.parent.is_symlink()
        or not target.is_file()
        or not _MARKER.search(_read(target))
    ):
        status = "foreign"
    elif _read(target) == _bundled():
        status = "current"
    else:
        status = "update_required"
    return {
        "contract": SUBAGENT_CONTRACT,
        "version": CONTRACT_VERSION,
        "path": str(target),
        "status": status,
        "tools": ["Read", "Grep"],
    }


def install_subagent(home: Path) -> dict[str, Any]:
    if inspect_subagent(home)["status"] == "foreign":
        raise SubagentError(f"{_target(home)} is not Liner's. Move it, then install again.")
    target = _target(home)
    target.parent.mkdir(parents=True, exist_ok=True)
    temporary = target.with_name(f"{target.name}.{os.getpid()}.tmp")
    temporary.write_text(_bundled(), encoding="utf-8")
    os.replace(temporary, target)
    return inspect_subagent(home)


def remove_subagent(home: Path) -> dict[str, Any]:
    if inspect_subagent(home)["status"] == "foreign":
        raise SubagentError(f"{_target(home)} is not Liner's, so it was left alone.")
    _target(home).unlink(missing_ok=True)
    return inspect_subagent(home)


def _target(home: Path) -> Path:
    return home / ".claude" / "agents" / f"{SUBAGENT_NAME}.md"


def _bundled() -> str:
    path = resources.files("liner") / "bundled" / "research" / f"{SUBAGENT_NAME}.md"
    return path.read_text(encoding="utf-8")


def _read(path: Path) -> str:
    return path.read_text(encoding="utf-8", errors="replace")
