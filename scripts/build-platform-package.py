#!/usr/bin/env python3
"""Build the current machine's npm platform package for the Liner CLI."""

from __future__ import annotations

import argparse
import json
import platform
import shutil
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TUI_PACKAGE_JSON = ROOT / "packages" / "tui" / "package.json"
ENTRYPOINT = ROOT / "src" / "liner" / "__main__.py"
DEFAULT_DIST = ROOT / "build" / "platform-dist"
DEFAULT_WORK = ROOT / "build" / "platform-work"
DEFAULT_SPEC = ROOT / "build" / "platform-spec"
DEFAULT_OUT = ROOT / "packages" / "platform"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Build a PyInstaller CLI binary and wrap it as an npm optionalDependency package.",
    )
    parser.add_argument(
        "--out-dir",
        type=Path,
        default=DEFAULT_OUT,
        help="Directory that will receive linersh-<platform>-<arch>/.",
    )
    parser.add_argument(
        "--version",
        default=read_tui_version(),
        help="npm package version to write. Defaults to packages/tui/package.json.",
    )
    parser.add_argument(
        "--clean",
        action=argparse.BooleanOptionalAction,
        default=True,
        help="Remove previous build/package output before building.",
    )
    parser.add_argument(
        "--skip-build",
        action="store_true",
        help="Reuse build/platform-dist/liner instead of running PyInstaller.",
    )
    args = parser.parse_args()

    target = current_target()
    package_name = f"linersh-{target['node_platform']}-{target['node_arch']}"
    package_dir = args.out_dir.resolve() / package_name

    if args.clean:
        safe_rmtree(DEFAULT_DIST)
        safe_rmtree(DEFAULT_WORK)
        safe_rmtree(DEFAULT_SPEC)
        safe_rmtree(package_dir)

    if not args.skip_build:
        run_pyinstaller()

    built_dir = DEFAULT_DIST / "liner"
    if not built_dir.is_dir():
        raise SystemExit(f"Expected PyInstaller output directory at {built_dir}")

    package_dir.mkdir(parents=True, exist_ok=True)
    copy_tree_contents(built_dir, package_dir)
    write_package_json(package_dir, package_name, args.version, target)
    write_readme(package_dir, package_name, target)

    binary = package_dir / exe_name(target["node_platform"])
    if target["node_platform"] != "win32":
        binary.chmod(binary.stat().st_mode | 0o755)

    smoke(binary)
    print(f"Built {package_name}@{args.version} in {package_dir}")
    print(f"Next: npm pack --dry-run {package_dir}")
    return 0


def run_pyinstaller() -> None:
    uv = shutil.which("uv")
    if uv is None:
        raise SystemExit("uv is required to build platform packages. Install uv and retry.")

    cmd = [
        uv,
        "run",
        "--python",
        "3.11",
        "--extra",
        "binary",
        "python",
        "-m",
        "PyInstaller",
        "--clean",
        "--onedir",
        "--name",
        "liner",
        "--distpath",
        str(DEFAULT_DIST),
        "--workpath",
        str(DEFAULT_WORK),
        "--specpath",
        str(DEFAULT_SPEC),
        "--collect-all",
        "playwright",
        "--collect-all",
        "trafilatura",
        "--collect-all",
        "certifi",
        "--hidden-import",
        "liner.handlers.web_js",
        str(ENTRYPOINT),
    ]
    subprocess.run(cmd, cwd=ROOT, check=True)


def current_target() -> dict[str, str]:
    sys_platform = sys.platform
    machine = platform.machine().lower()

    if sys_platform == "darwin":
        node_platform = "darwin"
    elif sys_platform.startswith("linux"):
        node_platform = "linux"
    elif sys_platform.startswith(("win32", "cygwin", "msys")):
        node_platform = "win32"
    else:
        raise SystemExit(f"Unsupported platform for npm binary package: {sys_platform}")

    arch_map = {
        "arm64": "arm64",
        "aarch64": "arm64",
        "x86_64": "x64",
        "amd64": "x64",
    }
    try:
        node_arch = arch_map[machine]
    except KeyError as exc:
        raise SystemExit(f"Unsupported architecture for npm binary package: {machine}") from exc

    return {"node_platform": node_platform, "node_arch": node_arch}


def read_tui_version() -> str:
    data = json.loads(TUI_PACKAGE_JSON.read_text(encoding="utf-8"))
    return str(data["version"])


def write_package_json(
    package_dir: Path,
    package_name: str,
    version: str,
    target: dict[str, str],
) -> None:
    package_json = {
        "name": package_name,
        "version": version,
        "description": "Platform-specific Liner CLI binary used by the linersh TUI package.",
        "license": "MIT",
        "os": [target["node_platform"]],
        "cpu": [target["node_arch"]],
        "files": [exe_name(target["node_platform"]), "_internal/", "README.md"],
    }
    (package_dir / "package.json").write_text(
        json.dumps(package_json, indent=2, sort_keys=False) + "\n",
        encoding="utf-8",
    )


def write_readme(package_dir: Path, package_name: str, target: dict[str, str]) -> None:
    text = (
        f"# {package_name}\n\n"
        "Platform-specific Liner CLI binary for the `linersh` npm package.\n\n"
        "This package is installed as an optional dependency. It is not meant to be used directly.\n\n"
        f"- platform: `{target['node_platform']}`\n"
        f"- arch: `{target['node_arch']}`\n"
    )
    (package_dir / "README.md").write_text(text, encoding="utf-8")


def copy_tree_contents(src: Path, dest: Path) -> None:
    for child in src.iterdir():
        target = dest / child.name
        if child.is_dir():
            shutil.copytree(child, target, dirs_exist_ok=True)
        else:
            shutil.copy2(child, target)


def smoke(binary: Path) -> None:
    subprocess.run([str(binary), "--version"], cwd=ROOT, check=True)
    subprocess.run([str(binary), "setup-js", "--help"], cwd=ROOT, check=True)


def safe_rmtree(path: Path) -> None:
    path = path.resolve()
    allowed_roots = [
        (ROOT / "build").resolve(),
        (ROOT / "packages" / "platform").resolve(),
    ]
    if not any(path == root or root in path.parents for root in allowed_roots):
        raise SystemExit(f"Refusing to remove unexpected path: {path}")
    if path.exists():
        shutil.rmtree(path)


def exe_name(node_platform: str) -> str:
    return "liner.exe" if node_platform == "win32" else "liner"


if __name__ == "__main__":
    raise SystemExit(main())
