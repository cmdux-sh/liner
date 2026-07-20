#!/usr/bin/env python3
"""Build the current machine's npm platform package for Liner native binaries."""

from __future__ import annotations

import argparse
import json
import platform
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_FILE = ROOT / "VERSION"
ENTRYPOINT = ROOT / "src" / "liner" / "__main__.py"
GO_TUI_DIR = ROOT / "packages" / "go-tui"
DEFAULT_DIST = ROOT / "build" / "platform-dist"
DEFAULT_WORK = ROOT / "build" / "platform-work"
DEFAULT_SPEC = ROOT / "build" / "platform-spec"
DEFAULT_OUT = ROOT / "packages" / "platform"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Build native CLI/TUI binaries and wrap them as an npm optionalDependency package.",
    )
    parser.add_argument(
        "--out-dir",
        type=Path,
        default=DEFAULT_OUT,
        help="Directory that will receive linersh-<platform>-<arch>/.",
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

    version = read_release_version()
    target = current_target()
    package_name = f"linersh-{target['node_platform']}-{target['node_arch']}"
    package_dir = args.out_dir.resolve() / package_name

    if args.clean:
        safe_rmtree(DEFAULT_DIST)
        safe_rmtree(DEFAULT_WORK)
        safe_rmtree(DEFAULT_SPEC)
        safe_rmtree(package_dir, extra_allowed=[args.out_dir.resolve()])

    if not args.skip_build:
        run_pyinstaller()

    built_dir = DEFAULT_DIST / "liner"
    if not built_dir.is_dir():
        raise SystemExit(f"Expected PyInstaller output directory at {built_dir}")

    package_dir.mkdir(parents=True, exist_ok=True)
    copy_tree_contents(built_dir, package_dir)
    build_go_tui(package_dir / tui_name(target["node_platform"]), version)
    write_package_json(package_dir, package_name, version, target)
    write_readme(package_dir, package_name, target)

    binary = package_dir / exe_name(target["node_platform"])
    if target["node_platform"] != "win32":
        binary.chmod(binary.stat().st_mode | 0o755)
        (package_dir / tui_name(target["node_platform"])).chmod(
            (package_dir / tui_name(target["node_platform"])).stat().st_mode | 0o755
        )

    smoke(binary)
    print(f"Built {package_name}@{version} in {package_dir}")
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
        "--collect-data",
        "liner",
        "--hidden-import",
        "liner.handlers.web_js",
        str(ENTRYPOINT),
    ]
    subprocess.run(cmd, cwd=ROOT, check=True)


def build_go_tui(out: Path, version: str) -> None:
    go = shutil.which("go")
    if go is None:
        raise SystemExit("go is required to build the platform TUI binary. Install Go and retry.")
    subprocess.run(
        [
            go,
            "build",
            "-trimpath",
            "-ldflags",
            f"-X main.version={version}",
            "-o",
            str(out),
            "./cmd/liner-tui",
        ],
        cwd=GO_TUI_DIR,
        check=True,
    )


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


def read_release_version() -> str:
    version = VERSION_FILE.read_text(encoding="utf-8").strip()
    if not version:
        raise SystemExit(f"{VERSION_FILE}: canonical release version is empty")
    return version


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
        "homepage": "https://liner.sh",
        "repository": {
            "type": "git",
            "url": "git+https://github.com/cmdux-sh/liner.git",
        },
        "bugs": {
            "url": "https://github.com/cmdux-sh/liner/issues",
        },
        "os": [target["node_platform"]],
        "cpu": [target["node_arch"]],
        "files": [
            exe_name(target["node_platform"]),
            tui_name(target["node_platform"]),
            "_internal/",
            "README.md",
        ],
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
        f"- binaries: `{exe_name(target['node_platform'])}`, `{tui_name(target['node_platform'])}`\n"
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


def safe_rmtree(path: Path, *, extra_allowed: list[Path] | None = None) -> None:
    path = path.resolve()
    allowed_roots = [
        (ROOT / "build").resolve(),
        (ROOT / "packages" / "platform").resolve(),
    ]
    if extra_allowed:
        allowed_roots.extend(root.resolve() for root in extra_allowed)
    if not any(path == root or root in path.parents for root in allowed_roots):
        raise SystemExit(f"Refusing to remove unexpected path: {path}")
    if path.exists():
        shutil.rmtree(path)


def exe_name(node_platform: str) -> str:
    return "liner.exe" if node_platform == "win32" else "liner"


def tui_name(node_platform: str) -> str:
    return "liner-tui.exe" if node_platform == "win32" else "liner-tui"


if __name__ == "__main__":
    raise SystemExit(main())
