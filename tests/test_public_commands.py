from __future__ import annotations

import importlib.util
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "validate-public-commands.py"
DOC_PATHS = (
    Path("README.md"),
    Path("marketing/site/src/content/docs/docs/cli.mdx"),
)


def _validator_module():
    spec = importlib.util.spec_from_file_location("validate_public_commands", SCRIPT)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def _fixture(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    for relative in DOC_PATHS:
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(ROOT / relative, target)
    return root


def _run(root: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), "--root", str(root)],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=False,
    )


def test_public_command_validator_accepts_documented_installed_commands(tmp_path: Path) -> None:
    result = _run(_fixture(tmp_path))

    assert result.returncode == 0, result.stderr
    assert "Public maintenance commands match the installed CLI." in result.stdout


def test_public_command_validator_reports_missing_command(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    page = root / "marketing/site/src/content/docs/docs/cli.mdx"
    page.write_text(
        page.read_text(encoding="utf-8").replace("`liner sources purge", "`liner sources hidden"),
        encoding="utf-8",
    )

    result = _run(root)

    assert result.returncode != 0
    assert "missing installed command: liner sources purge" in result.stderr
    assert "documents unknown command: liner sources hidden" in result.stderr


def test_public_command_validator_reports_option_drift(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    page = root / "marketing/site/src/content/docs/docs/cli.mdx"
    page.write_text(
        page.read_text(encoding="utf-8")
        .replace(" --source-id <id> --type <type>", " --type <type>", 1)
        .replace("`liner sources purge <folder> --source-id <id>`", "`liner sources purge <folder> --bogus <id>`"),
        encoding="utf-8",
    )

    result = _run(root)

    assert result.returncode != 0
    assert "liner sources replace omits required option: --source-id" in result.stderr
    assert "liner sources purge omits required option: --source-id" in result.stderr
    assert "liner sources purge documents unknown option: --bogus" in result.stderr


def test_help_parser_accepts_ci_rendering_variants() -> None:
    validator = _validator_module()
    help_text = validator.normalize_help_output(
        "\x1b[36m| inspect   Inspect a Project.\x1b[0m\n",
        "┃ guidance  Publish guidance.\n  plan      Plan a change.\n",
    )

    assert validator.discover_command_rows(help_text) == ["inspect", "guidance", "plan"]
