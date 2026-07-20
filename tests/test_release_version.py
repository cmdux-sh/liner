from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "release-version.py"
OWNED_PATHS = (
    Path("VERSION"),
    Path("pyproject.toml"),
    Path("src/liner/__init__.py"),
    Path("packages/tui/package.json"),
    Path("packages/tui/package-lock.json"),
    Path("README.md"),
    Path("docs/curation-skill/ABOUT.md"),
    Path("marketing/site/README.md"),
    Path("marketing/site/src/content/docs/docs/install.mdx"),
    Path("marketing/site/src/content/docs/docs/troubleshooting.mdx"),
)


def _fixture(tmp_path: Path, version: str = "2.3.4") -> Path:
    root = tmp_path / "repo"
    for relative in OWNED_PATHS:
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        if relative == Path("VERSION"):
            target.write_text(f"{version}\n", encoding="utf-8")
        else:
            shutil.copy2(ROOT / relative, target)
    return root


def _run(
    root: Path,
    mode: str,
    *extra: str,
    env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), mode, *extra, "--root", str(root)],
        text=True,
        capture_output=True,
        check=False,
        env=env,
    )


def _registry_env(tmp_path: Path, responses: dict[str, dict[str, object]]) -> dict[str, str]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    npm = bin_dir / "npm"
    npm.write_text(
        """#!/usr/bin/env python3
import json
import os
import sys

responses = json.loads(os.environ["LINER_TEST_NPM_RESPONSES"])
response = responses[sys.argv[2]]
print(json.dumps(response["stdout"]))
if response.get("stderr"):
    print(response["stderr"], file=sys.stderr)
raise SystemExit(response["returncode"])
""",
        encoding="utf-8",
    )
    npm.chmod(0o755)
    return {
        **os.environ,
        "PATH": f"{bin_dir}{os.pathsep}{os.environ.get('PATH', '')}",
        "LINER_TEST_NPM_RESPONSES": json.dumps(responses),
    }


def _missing_registry_responses(version: str) -> dict[str, dict[str, object]]:
    names = (
        "linersh",
        "linersh-darwin-arm64",
        "linersh-darwin-x64",
        "linersh-linux-arm64",
        "linersh-linux-x64",
        "linersh-win32-x64",
    )
    return {
        f"{name}@{version}": {
            "returncode": 1,
            "stdout": {
                "error": {
                    "code": "E404",
                    "summary": f"No match found for version {version}",
                }
            },
        }
        for name in names
    }


def test_release_version_write_updates_every_owned_representation(tmp_path: Path) -> None:
    root = _fixture(tmp_path)

    result = _run(root, "--write")

    assert result.returncode == 0, result.stderr
    assert "2.3.4" in result.stdout
    assert 'version = "2.3.4"' in (root / "pyproject.toml").read_text(encoding="utf-8")
    assert '__version__ = "2.3.4"' in (root / "src/liner/__init__.py").read_text(
        encoding="utf-8"
    )
    package = json.loads((root / "packages/tui/package.json").read_text(encoding="utf-8"))
    package_lock = json.loads(
        (root / "packages/tui/package-lock.json").read_text(encoding="utf-8")
    )
    assert package["version"] == "2.3.4"
    assert package_lock["version"] == "2.3.4"
    assert package_lock["packages"][""]["version"] == "2.3.4"
    for name in (
        "linersh-darwin-arm64",
        "linersh-darwin-x64",
        "linersh-linux-arm64",
        "linersh-linux-x64",
        "linersh-win32-x64",
    ):
        assert package["optionalDependencies"][name] == "2.3.4"
        assert package_lock["packages"][""]["optionalDependencies"][name] == "2.3.4"
        locked_platform = package_lock["packages"][f"node_modules/{name}"]
        assert locked_platform["version"] == "2.3.4"
        assert locked_platform["resolved"] == (
            f"https://registry.npmjs.org/{name}/-/{name}-2.3.4.tgz"
        )
        assert "integrity" not in locked_platform
    assert _run(root, "--check").returncode == 0
    assert "current source targets **v2.3.4**" in (root / "README.md").read_text(
        encoding="utf-8"
    )
    assert "**Platform support (2.3.4):**" in (root / "README.md").read_text(
        encoding="utf-8"
    )
    assert "current source targets v2.3.4" in (
        root / "docs/curation-skill/ABOUT.md"
    ).read_text(encoding="utf-8")
    assert "Product version shown in the page: `2.3.4`" in (
        root / "marketing/site/README.md"
    ).read_text(encoding="utf-8")
    install_text = (
        root / "marketing/site/src/content/docs/docs/install.mdx"
    ).read_text(encoding="utf-8")
    assert "Version `2.3.4` supports" in install_text
    assert install_text.count("linersh@2.3.4") == 2
    assert "liner 2.3.4 (tui) · 2.3.4 (core)" in (
        root / "marketing/site/src/content/docs/docs/troubleshooting.mdx"
    ).read_text(encoding="utf-8")


def test_release_version_check_is_read_only_and_fails_on_drift(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0
    package_path = root / "packages/tui/package.json"
    package = json.loads(package_path.read_text(encoding="utf-8"))
    package["version"] = "9.9.9"
    package["optionalDependencies"]["linersh-linux-x64"] = "9.9.8"
    package_path.write_text(json.dumps(package, indent=2) + "\n", encoding="utf-8")
    lock_path = root / "packages/tui/package-lock.json"
    package_lock = json.loads(lock_path.read_text(encoding="utf-8"))
    locked_platform = package_lock["packages"]["node_modules/linersh-linux-x64"]
    locked_platform["version"] = "9.9.7"
    locked_platform["resolved"] = (
        "https://registry.npmjs.org/linersh-linux-x64/-/linersh-linux-x64-9.9.7.tgz"
    )
    lock_path.write_text(json.dumps(package_lock, indent=2) + "\n", encoding="utf-8")
    before = package_path.read_bytes()
    lock_before = lock_path.read_bytes()

    result = _run(root, "--check")

    assert result.returncode != 0
    assert "packages/tui/package.json" in result.stderr
    assert "optionalDependencies.linersh-linux-x64" in result.stderr
    assert "node_modules/linersh-linux-x64.version" in result.stderr
    assert "node_modules/linersh-linux-x64.resolved" in result.stderr
    assert "expected 2.3.4" in result.stderr
    assert package_path.read_bytes() == before
    assert lock_path.read_bytes() == lock_before


def test_release_version_prerelease_round_trip_is_idempotent(tmp_path: Path) -> None:
    version = "2.3.4-rc.1+build.7"
    root = _fixture(tmp_path, version=version)

    first = _run(root, "--write")

    assert first.returncode == 0, first.stderr
    install_path = root / "marketing/site/src/content/docs/docs/install.mdx"
    first_install = install_path.read_bytes()
    assert install_path.read_text(encoding="utf-8").count(f"linersh@{version}") == 2
    assert _run(root, "--check").returncode == 0

    second = _run(root, "--write")

    assert second.returncode == 0, second.stderr
    assert install_path.read_bytes() == first_install


def test_release_version_check_reports_public_surface_drift(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0
    page = root / "marketing/site/src/content/docs/docs/troubleshooting.mdx"
    page.write_text(
        page.read_text(encoding="utf-8").replace("liner 2.3.4 (tui)", "liner 9.9.9 (tui)"),
        encoding="utf-8",
    )

    result = _run(root, "--check")

    assert result.returncode != 0
    assert "marketing/site/src/content/docs/docs/troubleshooting.mdx tui" in result.stderr
    assert "expected 2.3.4, found 9.9.9" in result.stderr


def test_registry_check_rejects_an_existing_package_version(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0
    responses = _missing_registry_responses("2.3.4")
    responses["linersh@2.3.4"] = {"returncode": 0, "stdout": "2.3.4"}

    result = _run(
        root,
        "--check",
        "--check-registry",
        env=_registry_env(tmp_path, responses),
    )

    assert result.returncode != 0
    assert "linersh@2.3.4 is already published" in result.stderr


def test_registry_check_accepts_an_unpublished_release_set(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0

    result = _run(
        root,
        "--check",
        "--check-registry",
        env=_registry_env(tmp_path, _missing_registry_responses("2.3.4")),
    )

    assert result.returncode == 0, result.stderr
    assert "No Liner 2.3.4 package is published on npm." in result.stdout


def test_registry_check_can_limit_prepublication_to_the_main_package(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0
    responses = _missing_registry_responses("2.3.4")
    responses["linersh-darwin-arm64@2.3.4"] = {
        "returncode": 0,
        "stdout": "2.3.4",
    }

    result = _run(
        root,
        "--check",
        "--check-registry",
        "--registry-package",
        "linersh",
        env=_registry_env(tmp_path, responses),
    )

    assert result.returncode == 0, result.stderr


def test_registry_check_fails_closed_on_registry_errors(tmp_path: Path) -> None:
    root = _fixture(tmp_path)
    assert _run(root, "--write").returncode == 0
    responses = _missing_registry_responses("2.3.4")
    responses["linersh@2.3.4"] = {
        "returncode": 2,
        "stdout": {"error": {"code": "EAI_AGAIN", "summary": "registry unavailable"}},
        "stderr": "network timeout",
    }

    result = _run(
        root,
        "--check",
        "--check-registry",
        env=_registry_env(tmp_path, responses),
    )

    assert result.returncode != 0
    assert "Could not verify linersh@2.3.4 against npm" in result.stderr
    assert "network timeout" in result.stderr
