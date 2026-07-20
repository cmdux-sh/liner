from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "validate-public-surfaces.py"
SURFACES = (
    Path("README.md"),
    Path("marketing/site/src/pages/index.astro"),
    Path("marketing/site/src/content/docs/docs/index.mdx"),
    Path("marketing/site/src/content/docs/docs/install.mdx"),
    Path("marketing/site/src/content/docs/docs/maintenance.mdx"),
    Path("marketing/site/src/content/docs/docs/build-a-mixtape.mdx"),
    Path("marketing/site/src/pages/changelog.astro"),
)


def _fixture(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    for relative in SURFACES:
        source = ROOT / relative
        if not source.exists():
            continue
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)
    return root


def _run(root: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), "--root", str(root)],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=False,
    )


def test_public_surface_validator_accepts_canonical_agent_maintenance_story(
    tmp_path: Path,
) -> None:
    result = _run(_fixture(tmp_path))

    assert result.returncode == 0, result.stderr
    assert "Public agent-maintenance surfaces match" in result.stdout


def test_public_surface_validator_reports_missing_adapter_story(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    homepage = root / "marketing/site/src/pages/index.astro"
    homepage.write_text(
        homepage.read_text(encoding="utf-8").replace(
            "Let Codex or Claude maintain the Project safely.",
            "Use the Project again later.",
        ),
        encoding="utf-8",
    )

    result = _run(root)

    assert result.returncode != 0
    assert "missing required public truth" in result.stderr


def test_public_surface_validator_reports_direct_edit_guidance(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    homepage = root / "marketing/site/src/pages/index.astro"
    homepage.write_text(
        homepage.read_text(encoding="utf-8")
        + "\nedit the saved Markdown and YAML files by hand\n",
        encoding="utf-8",
    )

    result = _run(root)

    assert result.returncode != 0
    assert "contains banned direct-edit guidance" in result.stderr


def test_public_surface_validator_reports_homepage_section_budget_drift(
    tmp_path: Path,
) -> None:
    root = _fixture(tmp_path)
    homepage = root / "marketing/site/src/pages/index.astro"
    homepage.write_text(
        homepage.read_text(encoding="utf-8").replace(
            'data-home-section="workflow"',
            'data-home-section="workflow-extra"',
        ),
        encoding="utf-8",
    )

    result = _run(root)

    assert result.returncode != 0
    assert "expected seven homepage narrative sections" in result.stderr
