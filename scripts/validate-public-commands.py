#!/usr/bin/env python3
"""Verify that public maintenance docs match the installed Liner CLI."""

from __future__ import annotations

import argparse
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LAUNCHER = ROOT / "packages" / "tui" / "bin" / "liner.js"
MAINTENANCE_GROUPS = ("project", "sources", "adapters")
PUBLIC_DOCS = (
    Path("README.md"),
    Path("marketing/site/src/content/docs/docs/cli.mdx"),
)
COMMAND_ROW = re.compile(r"^\s*│\s*([a-z][a-z0-9-]*)\s{2,}", re.MULTILINE)
OPTION_ROW = re.compile(
    r"^\s*│\s*(?P<required>\*)?\s*(?P<option>--[a-z][a-z0-9-]*)\b",
    re.MULTILINE,
)
DOCUMENTED_COMMAND = re.compile(
    r"`(?P<signature>liner\s+(?P<group>project|sources|adapters)\s+"
    r"(?P<command>[a-z][a-z0-9-]*)\b[^`]*)`"
)
DOCUMENTED_OPTION = re.compile(r"--[a-z][a-z0-9-]*\b")


@dataclass(frozen=True)
class InstalledCommand:
    options: frozenset[str]
    required_options: frozenset[str]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Compare installed maintenance commands with public documentation."
    )
    parser.add_argument("--root", type=Path, default=ROOT, help=argparse.SUPPRESS)
    args = parser.parse_args()

    root = args.root.resolve()
    installed = discover_installed_commands()
    failures: list[str] = []
    for relative in PUBLIC_DOCS:
        path = root / relative
        try:
            documented = discover_documented_commands(path)
        except OSError as exc:
            failures.append(f"{relative}: could not read documentation: {exc}")
            continue
        for group in MAINTENANCE_GROUPS:
            missing = sorted(installed[group].keys() - documented[group].keys())
            unknown = sorted(documented[group].keys() - installed[group].keys())
            failures.extend(
                f"{relative}: missing installed command: liner {group} {command}"
                for command in missing
            )
            failures.extend(
                f"{relative}: documents unknown command: liner {group} {command}"
                for command in unknown
            )
            for command in installed[group].keys() & documented[group].keys():
                command_contract = installed[group][command]
                documented_options = documented[group][command]
                missing_required = sorted(
                    command_contract.required_options - documented_options
                )
                unknown_options = sorted(documented_options - command_contract.options)
                failures.extend(
                    f"{relative}: liner {group} {command} omits required option: {option}"
                    for option in missing_required
                )
                failures.extend(
                    f"{relative}: liner {group} {command} documents unknown option: {option}"
                    for option in unknown_options
                )

    if failures:
        print("\n".join(failures), file=sys.stderr)
        return 1
    print("Public maintenance commands match the installed CLI.")
    return 0


def discover_installed_commands() -> dict[str, dict[str, InstalledCommand]]:
    node = shutil.which("node")
    if node is None:
        raise SystemExit("Could not inspect the installed CLI boundary because node is missing.")
    commands: dict[str, dict[str, InstalledCommand]] = {}
    for group in MAINTENANCE_GROUPS:
        result = subprocess.run(
            [node, str(LAUNCHER), group, "--help"],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip()
            raise SystemExit(f"Could not inspect launcher `liner {group} --help`: {detail}")
        discovered = set(COMMAND_ROW.findall(result.stdout))
        if not discovered:
            raise SystemExit(f"Could not find commands in `liner {group} --help`.")
        commands[group] = {
            command: discover_installed_command(node, group, command)
            for command in discovered
        }
    return commands


def discover_installed_command(node: str, group: str, command: str) -> InstalledCommand:
    result = subprocess.run(
        [node, str(LAUNCHER), group, command, "--help"],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip()
        raise SystemExit(
            f"Could not inspect launcher `liner {group} {command} --help`: {detail}"
        )
    rows = list(OPTION_ROW.finditer(result.stdout))
    options = frozenset(match.group("option") for match in rows)
    required = frozenset(
        match.group("option") for match in rows if match.group("required")
    )
    return InstalledCommand(options=options, required_options=required)


def discover_documented_commands(path: Path) -> dict[str, dict[str, set[str]]]:
    documented = {group: {} for group in MAINTENANCE_GROUPS}
    for match in DOCUMENTED_COMMAND.finditer(path.read_text(encoding="utf-8")):
        documented[match.group("group")][match.group("command")] = set(
            DOCUMENTED_OPTION.findall(match.group("signature"))
        )
    return documented


if __name__ == "__main__":
    raise SystemExit(main())
