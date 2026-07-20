#!/usr/bin/env python3
"""Verify the public agent-maintenance story and its safety boundary."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROOT_README = Path("README.md")
HOMEPAGE = Path("marketing/site/src/pages/index.astro")
DOCS_INDEX = Path("marketing/site/src/content/docs/docs/index.mdx")
INSTALL_GUIDE = Path("marketing/site/src/content/docs/docs/install.mdx")
MAINTENANCE_GUIDE = Path("marketing/site/src/content/docs/docs/maintenance.mdx")
BUILD_GUIDE = Path("marketing/site/src/content/docs/docs/build-a-mixtape.mdx")
CHANGELOG_PAGE = Path("marketing/site/src/pages/changelog.astro")

HOME_SECTION_KEYS = (
    "hero",
    "project-proof",
    "examples",
    "workflow",
    "maintenance",
    "fit-trust",
    "install",
)

REQUIRED: dict[Path, tuple[str, ...]] = {
    ROOT_README: ("liner sources add mobile-design-foundations",),
    HOMEPAGE: (
        "Build reusable",
        "One request becomes a reusable Liner Project.",
        "MIXTAPE.md",
        "LINER.md",
        "SKILL.md",
        'id="use-cases"',
        "Three jobs where source choice changes the output.",
        "Worth a Project",
        "A chat is enough",
        "Let Codex or Claude maintain the Project safely.",
        "liner adapters install codex --yes",
        "Project Change Set",
        "Change Receipt",
        "Start building",
        "npx linersh",
    ),
    DOCS_INDEX: ("/docs/maintenance/", "Optional Maintenance Adapter"),
    INSTALL_GUIDE: ("liner adapters install codex --yes", "/docs/maintenance/"),
    MAINTENANCE_GUIDE: (
        "Project Skill",
        "Maintenance Adapter",
        "`type: skill` Source",
        "liner project guidance",
        "approval_required",
        "Change Receipt",
    ),
    CHANGELOG_PAGE: ("Candidate", "Maintenance Adapter", "Change Receipt"),
}

BANNED_DIRECT_EDIT: dict[Path, tuple[str, ...]] = {
    ROOT_README: (
        "edit mobile-design-foundations/mixtape/tape.yaml with your sources",
        "write mobile-design-foundations/mixtape/synthesis.md",
    ),
    HOMEPAGE: ("edit the saved Markdown and YAML files by hand",),
    BUILD_GUIDE: ("Edit `mobile-design-foundations/mixtape/tape.yaml` with sources",),
}

RETIRED_HOMEPAGE_COPY = (
        "Your setup work",
        "Keep the work behind the answer.",
        "The saved research keeps working after the first answer.",
        "The setup stops being temporary.",
        "A later answer is easier to trust when the sources are still there.",
        "A few practical edges before you run it.",
)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Check public Maintenance Adapter copy and direct-edit boundaries."
    )
    parser.add_argument("--root", type=Path, default=ROOT, help=argparse.SUPPRESS)
    args = parser.parse_args()

    root = args.root.resolve()
    failures: list[str] = []
    for relative, fragments in REQUIRED.items():
        body = read_surface(root, relative, failures)
        if body is None:
            continue
        failures.extend(
            f"{relative}: missing required public truth: {fragment}"
            for fragment in fragments
            if fragment not in body
        )
    for relative, fragments in BANNED_DIRECT_EDIT.items():
        body = read_surface(root, relative, failures)
        if body is None:
            continue
        failures.extend(
            f"{relative}: contains banned direct-edit guidance: {fragment}"
            for fragment in fragments
            if fragment in body
        )

    homepage_copy = read_surface(root, HOMEPAGE, failures)
    if homepage_copy is not None:
        failures.extend(
            f"{HOMEPAGE}: contains retired repetitive homepage copy: {fragment}"
            for fragment in RETIRED_HOMEPAGE_COPY
            if fragment in homepage_copy
        )

    homepage = read_surface(root, HOMEPAGE, failures)
    if homepage is not None:
        section_keys = tuple(re.findall(r'data-home-section="([^"]+)"', homepage))
        if section_keys != HOME_SECTION_KEYS:
            failures.append(
                f"{HOMEPAGE}: expected seven homepage narrative sections in order "
                f"{HOME_SECTION_KEYS}, found {section_keys}"
            )

    if failures:
        print("\n".join(failures), file=sys.stderr)
        return 1
    print("Public agent-maintenance surfaces match the canonical safety story.")
    return 0


def read_surface(root: Path, relative: Path, failures: list[str]) -> str | None:
    try:
        return (root / relative).read_text(encoding="utf-8")
    except OSError as exc:
        failures.append(f"{relative}: could not read public surface: {exc}")
        return None


if __name__ == "__main__":
    raise SystemExit(main())
