"""Tripwire test: core modules must not import CLI-only deps (rich, typer)."""
from __future__ import annotations

import re
from pathlib import Path

import pytest

SRC = Path(__file__).resolve().parent.parent / "src" / "liner"

# Modules that ARE allowed to import rich/typer.
CLI_ONLY = {"cli.py", "cli_progress.py"}

FORBIDDEN = re.compile(r"^\s*(?:from|import)\s+(rich|typer)\b", re.MULTILINE)


def _core_modules() -> list[Path]:
    out: list[Path] = []
    for path in SRC.rglob("*.py"):
        if path.name in CLI_ONLY:
            continue
        out.append(path)
    return out


@pytest.mark.parametrize("module", _core_modules(), ids=lambda p: str(p.relative_to(SRC)))
def test_no_cli_deps_in_core(module: Path) -> None:
    text = module.read_text(encoding="utf-8")
    match = FORBIDDEN.search(text)
    assert match is None, (
        f"{module.relative_to(SRC)} imports {match.group(1)!r}; "
        "this module must stay CLI-free so a future web app can import it."
    )
