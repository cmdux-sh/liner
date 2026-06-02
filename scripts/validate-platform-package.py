#!/usr/bin/env python3
"""Validate generated npm platform packages before publishing."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_GLOB = ROOT / "packages" / "platform"


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate linersh platform package directories.")
    parser.add_argument("packages", nargs="*", type=Path, help="Package directories to validate.")
    parser.add_argument(
        "--pack-dry-run",
        action="store_true",
        help="Run npm pack --dry-run inside each package directory.",
    )
    args = parser.parse_args()

    package_dirs = args.packages or sorted(DEFAULT_GLOB.glob("linersh-*"))
    if not package_dirs:
        raise SystemExit("No platform packages found. Run scripts/build-platform-package.py first.")

    for package_dir in package_dirs:
        validate_package(package_dir.resolve(), pack_dry_run=args.pack_dry_run)
    print(f"Validated {len(package_dirs)} platform package(s).")
    return 0


def validate_package(package_dir: Path, *, pack_dry_run: bool) -> None:
    package_json_path = package_dir / "package.json"
    if not package_json_path.is_file():
        raise SystemExit(f"{package_dir}: missing package.json")

    data = json.loads(package_json_path.read_text(encoding="utf-8"))
    name = data.get("name")
    if not isinstance(name, str) or not name.startswith("linersh-"):
        raise SystemExit(f"{package_dir}: package name must start with linersh-")
    if package_dir.name != name:
        raise SystemExit(f"{package_dir}: directory name must match package name {name}")

    os_values = data.get("os")
    cpu_values = data.get("cpu")
    if not (isinstance(os_values, list) and len(os_values) == 1):
        raise SystemExit(f"{package_dir}: package.json must contain exactly one os value")
    if not (isinstance(cpu_values, list) and len(cpu_values) == 1):
        raise SystemExit(f"{package_dir}: package.json must contain exactly one cpu value")

    exe = package_dir / ("liner.exe" if os_values[0] == "win32" else "liner")
    if not exe.is_file():
        raise SystemExit(f"{package_dir}: missing root executable {exe.name}")
    if os_values[0] != "win32" and not os.access(exe, os.X_OK):
        raise SystemExit(f"{package_dir}: {exe.name} is not executable")

    internal = package_dir / "_internal"
    if not internal.is_dir():
        raise SystemExit(f"{package_dir}: missing PyInstaller _internal directory")

    subprocess.run([str(exe), "--version"], cwd=package_dir, check=True)
    subprocess.run([str(exe), "setup-js", "--help"], cwd=package_dir, check=True)

    if pack_dry_run:
        npm = shutil.which("npm")
        if npm is None:
            raise SystemExit("npm is required for --pack-dry-run")
        proc = subprocess.run(
            [npm, "pack", "--dry-run", "--json"],
            cwd=package_dir,
            check=True,
            text=True,
            capture_output=True,
        )
        pack = json.loads(proc.stdout)[0]
        print(
            f"{name}: npm pack ok, "
            f"{pack['filename']}, "
            f"{pack['size'] / 1024 / 1024:.1f} MiB packed, "
            f"{pack['unpackedSize'] / 1024 / 1024:.1f} MiB unpacked"
        )


if __name__ == "__main__":
    raise SystemExit(main())
