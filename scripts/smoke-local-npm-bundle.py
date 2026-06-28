#!/usr/bin/env python3
"""Install local npm packages into a temp project and smoke resolved binaries."""

from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TUI_DIR = ROOT / "packages" / "tui"
PLATFORM_DIR = ROOT / "packages" / "platform"


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test local linersh npm package wiring.")
    parser.add_argument(
        "--keep-temp",
        action="store_true",
        help="Keep the temp npm project after the smoke test.",
    )
    parser.add_argument(
        "--platform-package",
        type=Path,
        default=find_single_platform_package(),
        help="Path to a generated linersh-<platform>-<arch> package.",
    )
    args = parser.parse_args()

    temp_dir = Path(tempfile.mkdtemp(prefix="liner-npm-smoke."))
    try:
        subprocess.run(["npm", "init", "-y"], cwd=temp_dir, check=True, stdout=subprocess.DEVNULL)
        subprocess.run(
            ["npm", "install", str(TUI_DIR), str(args.platform_package.resolve())],
            cwd=temp_dir,
            check=True,
            stdout=subprocess.DEVNULL,
        )

        liner = temp_dir / "node_modules" / ".bin" / ("liner.cmd" if sys.platform == "win32" else "liner")
        platform_tui = (
            temp_dir
            / "node_modules"
            / args.platform_package.resolve().name
            / ("liner-tui.exe" if sys.platform == "win32" else "liner-tui")
        )
        subprocess.run([str(liner), "--version"], cwd=temp_dir, check=True)
        subprocess.run([str(platform_tui), "--version"], cwd=temp_dir, check=True)

        env = {**os.environ, "LINER_BIN": "/bin/echo"}
        if sys.platform == "win32":
            env["LINER_BIN"] = "cmd"
            subprocess.run([str(liner), "/c", "echo"], cwd=temp_dir, env=env, check=True)
        else:
            subprocess.run([str(liner), "setup-js"], cwd=temp_dir, env=env, check=True)

        print(f"Local npm bundle smoke passed in {temp_dir}")
    finally:
        if args.keep_temp:
            print(f"Kept temp project: {temp_dir}")
        else:
            shutil.rmtree(temp_dir, ignore_errors=True)
    return 0


def find_single_platform_package() -> Path:
    packages = sorted(PLATFORM_DIR.glob("linersh-*"))
    if len(packages) == 1:
        return packages[0]
    if not packages:
        return PLATFORM_DIR / "linersh-<platform>-<arch>"
    raise SystemExit(
        "Multiple platform packages found; pass --platform-package explicitly:\n"
        + "\n".join(f"  {pkg}" for pkg in packages)
    )


if __name__ == "__main__":
    raise SystemExit(main())
