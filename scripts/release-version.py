#!/usr/bin/env python3
"""Write or verify every owned Liner release-version representation."""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SEMVER_TOKEN = (
    r"(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)"
    r"(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?"
    r"(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?"
)
SEMVER = re.compile(rf"^{SEMVER_TOKEN}$")
PLATFORM_PACKAGES = (
    "linersh-darwin-arm64",
    "linersh-darwin-x64",
    "linersh-linux-arm64",
    "linersh-linux-x64",
    "linersh-win32-x64",
)
NPM_PACKAGES = ("linersh", *PLATFORM_PACKAGES)


@dataclass(frozen=True)
class PublicVersionField:
    label: str
    path: str
    write_pattern: str
    read_pattern: str
    expected_count: int = 1


PUBLIC_VERSION_FIELDS = (
    PublicVersionField(
        "README.md current source",
        "README.md",
        r"(current source targets \*\*v)[^*]+(\*\*)",
        r"current source targets \*\*v([^*]+)\*\*",
    ),
    PublicVersionField(
        "README.md platform support",
        "README.md",
        r"(\*\*Platform support \()[^)]+(\):\*\*)",
        r"\*\*Platform support \(([^)]+)\):\*\*",
    ),
    PublicVersionField(
        "docs/curation-skill/ABOUT.md current source",
        "docs/curation-skill/ABOUT.md",
        r"(current source targets v)[^,]+(, with macOS)",
        r"current source targets v([^,]+), with macOS",
    ),
    PublicVersionField(
        "marketing/site/README.md product version",
        "marketing/site/README.md",
        r"(Product version shown in the page: `)[^`]+(`)",
        r"Product version shown in the page: `([^`]+)`",
    ),
    PublicVersionField(
        "marketing/site/src/content/docs/docs/install.mdx support version",
        "marketing/site/src/content/docs/docs/install.mdx",
        r"(Version `)[^`]+(` supports)",
        r"Version `([^`]+)` supports",
    ),
    PublicVersionField(
        "marketing/site/src/content/docs/docs/install.mdx pinned command",
        "marketing/site/src/content/docs/docs/install.mdx",
        rf"(linersh@){SEMVER_TOKEN}((?![0-9A-Za-z.+-]))",
        rf"linersh@({SEMVER_TOKEN})(?![0-9A-Za-z.+-])",
        expected_count=2,
    ),
    PublicVersionField(
        "marketing/site/src/content/docs/docs/troubleshooting.mdx tui",
        "marketing/site/src/content/docs/docs/troubleshooting.mdx",
        r'(command="liner )[^ ]+( \(tui\))',
        r'command="liner ([^ ]+) \(tui\)',
    ),
    PublicVersionField(
        "marketing/site/src/content/docs/docs/troubleshooting.mdx core",
        "marketing/site/src/content/docs/docs/troubleshooting.mdx",
        r"( · )[^ ]+( \(core\))",
        r" · ([^ ]+) \(core\)",
    ),
)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Synchronize repository release metadata with the canonical VERSION file."
    )
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--write", action="store_true", help="Write the canonical version everywhere.")
    mode.add_argument("--check", action="store_true", help="Fail if any owned version has drifted.")
    parser.add_argument(
        "--check-registry",
        action="store_true",
        help="Fail unless the canonical version is unpublished for every Liner npm package.",
    )
    parser.add_argument(
        "--registry-package",
        action="append",
        choices=NPM_PACKAGES,
        help="Limit registry uniqueness checks to one package. Repeat for multiple packages.",
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=ROOT,
        help=argparse.SUPPRESS,
    )
    args = parser.parse_args()
    if args.check_registry and not args.check:
        parser.error("--check-registry requires --check")
    if args.registry_package and not args.check_registry:
        parser.error("--registry-package requires --check-registry")

    root = args.root.resolve()
    version = read_canonical_version(root)
    if args.write:
        write_versions(root, version)
        print(f"Synchronized Liner release {version} across owned metadata.")

    drift = find_drift(root, version)
    if drift:
        for path, observed in drift:
            print(
                f"{path}: expected {version}, found {observed}",
                file=sys.stderr,
            )
        return 1
    if args.check_registry:
        try:
            published = find_published_packages(
                root,
                version,
                packages=tuple(args.registry_package or NPM_PACKAGES),
            )
        except RegistryCheckError as exc:
            print(str(exc), file=sys.stderr)
            return 2
        if published:
            for package in published:
                print(f"{package}@{version} is already published on npm.", file=sys.stderr)
            print(
                "Choose a new canonical VERSION before building or publishing release artifacts.",
                file=sys.stderr,
            )
            return 1
        print(f"No Liner {version} package is published on npm.")
    print(f"Liner release metadata is synchronized at {version}.")
    return 0


class RegistryCheckError(RuntimeError):
    """The npm registry could not provide a trustworthy uniqueness result."""


def find_published_packages(
    root: Path,
    version: str,
    *,
    packages: tuple[str, ...],
) -> list[str]:
    npm = shutil.which("npm")
    if npm is None:
        raise RegistryCheckError(
            "Could not verify npm release uniqueness because npm is not installed."
        )

    published: list[str] = []
    for package in packages:
        spec = f"{package}@{version}"
        result = subprocess.run(
            [npm, "view", spec, "version", "--json"],
            cwd=root,
            text=True,
            capture_output=True,
            check=False,
        )
        payload = parse_registry_payload(result.stdout)
        if result.returncode == 0:
            observed = payload if isinstance(payload, str) else None
            if observed != version:
                raise RegistryCheckError(
                    f"Could not verify {spec} against npm: expected version {version!r}, "
                    f"received {result.stdout.strip() or 'an empty response'}."
                )
            published.append(package)
            continue

        error_code = None
        if isinstance(payload, dict):
            error = payload.get("error")
            if isinstance(error, dict):
                error_code = error.get("code")
        if error_code == "E404":
            continue

        detail = result.stderr.strip() or result.stdout.strip() or f"npm exited {result.returncode}"
        raise RegistryCheckError(f"Could not verify {spec} against npm: {detail}")
    return published


def parse_registry_payload(stdout: str) -> object:
    try:
        return json.loads(stdout)
    except json.JSONDecodeError:
        return None


def read_canonical_version(root: Path) -> str:
    path = root / "VERSION"
    try:
        version = path.read_text(encoding="utf-8").strip()
    except OSError as exc:
        raise SystemExit(f"Could not read {path}: {exc}") from exc
    if not SEMVER.fullmatch(version):
        raise SystemExit(f"{path}: expected a semantic version, found {version!r}")
    return version


def write_versions(root: Path, version: str) -> None:
    update_text_version(
        root / "pyproject.toml",
        r'(?ms)(^\[project\]\n.*?^version = ")[^"]+("$)',
        version,
    )
    update_text_version(
        root / "src" / "liner" / "__init__.py",
        r'(?m)(^__version__ = ")[^"]+("$)',
        version,
    )
    package_path = root / "packages" / "tui" / "package.json"
    package = read_json(package_path)
    package["version"] = version
    package_optional = require_mapping(package, "optionalDependencies", package_path)
    for name in PLATFORM_PACKAGES:
        package_optional[name] = version
    write_json(package_path, package)

    lock_path = root / "packages" / "tui" / "package-lock.json"
    package_lock = read_json(lock_path)
    package_lock["version"] = version
    locked_packages = require_mapping(package_lock, "packages", lock_path)
    root_package = require_mapping(locked_packages, "", lock_path)
    root_package["version"] = version
    lock_optional = require_mapping(root_package, "optionalDependencies", lock_path)
    for name in PLATFORM_PACKAGES:
        lock_optional[name] = version
        node = require_mapping(locked_packages, f"node_modules/{name}", lock_path)
        if node.get("version") != version:
            node.pop("integrity", None)
        node["version"] = version
        node["resolved"] = platform_tarball_url(name, version)
    write_json(lock_path, package_lock)
    for field in PUBLIC_VERSION_FIELDS:
        update_text_versions(
            root / field.path,
            field.write_pattern,
            version,
            expected_count=field.expected_count,
        )


def find_drift(root: Path, expected: str) -> list[tuple[str, str]]:
    observed = {
        "pyproject.toml": read_text_version(
            root / "pyproject.toml",
            r'(?ms)^\[project\]\n.*?^version = "([^"]+)"$',
        ),
        "src/liner/__init__.py": read_text_version(
            root / "src" / "liner" / "__init__.py",
            r'(?m)^__version__ = "([^"]+)"$',
        ),
    }
    package = read_json(root / "packages" / "tui" / "package.json")
    package_lock = read_json(root / "packages" / "tui" / "package-lock.json")
    observed["packages/tui/package.json"] = display_value(package.get("version"))
    observed["packages/tui/package-lock.json"] = display_value(package_lock.get("version"))
    locked_packages = require_mapping(
        package_lock, "packages", root / "packages" / "tui" / "package-lock.json"
    )
    root_package = require_mapping(
        locked_packages, "", root / "packages" / "tui" / "package-lock.json"
    )
    observed["packages/tui/package-lock.json packages['']"] = display_value(
        root_package.get("version")
    )
    package_optional = require_mapping(
        package, "optionalDependencies", root / "packages" / "tui" / "package.json"
    )
    lock_optional = require_mapping(
        root_package,
        "optionalDependencies",
        root / "packages" / "tui" / "package-lock.json",
    )
    for name in PLATFORM_PACKAGES:
        observed[f"packages/tui/package.json optionalDependencies.{name}"] = display_value(
            package_optional.get(name)
        )
        observed[
            f"packages/tui/package-lock.json packages[''].optionalDependencies.{name}"
        ] = display_value(lock_optional.get(name))
        node = require_mapping(
            locked_packages,
            f"node_modules/{name}",
            root / "packages" / "tui" / "package-lock.json",
        )
        observed[f"packages/tui/package-lock.json node_modules/{name}.version"] = display_value(
            node.get("version")
        )
        observed[f"packages/tui/package-lock.json node_modules/{name}.resolved"] = (
            expected
            if node.get("resolved") == platform_tarball_url(name, expected)
            else display_value(node.get("resolved"))
        )
    for field in PUBLIC_VERSION_FIELDS:
        values = re.findall(
            field.read_pattern,
            (root / field.path).read_text(encoding="utf-8"),
        )
        if len(values) != field.expected_count:
            observed[field.label] = f"{len(values)} owned fields"
            continue
        for index, value in enumerate(values, start=1):
            label = field.label
            if field.expected_count > 1:
                label = f"{label} {index}"
            observed[label] = value
    return [(path, value) for path, value in observed.items() if value != expected]


def update_text_version(path: Path, pattern: str, version: str) -> None:
    text = path.read_text(encoding="utf-8")
    updated, count = re.subn(pattern, rf"\g<1>{version}\g<2>", text, count=1)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one owned version field, found {count}")
    path.write_text(updated, encoding="utf-8")


def update_text_versions(
    path: Path,
    pattern: str,
    version: str,
    *,
    expected_count: int,
) -> None:
    text = path.read_text(encoding="utf-8")
    updated, count = re.subn(pattern, rf"\g<1>{version}\g<2>", text)
    if count != expected_count:
        raise SystemExit(
            f"{path}: expected {expected_count} owned version fields, found {count}"
        )
    path.write_text(updated, encoding="utf-8")


def read_text_version(path: Path, pattern: str) -> str:
    match = re.search(pattern, path.read_text(encoding="utf-8"))
    if match is None:
        return "missing"
    return match.group(1)


def read_json(path: Path) -> dict[str, object]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"Could not read {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise SystemExit(f"{path}: expected a JSON object")
    return value


def write_json(path: Path, value: dict[str, object]) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def require_mapping(
    value: dict[str, object], key: str, path: Path
) -> dict[str, object]:
    mapping = value.get(key)
    if not isinstance(mapping, dict):
        raise SystemExit(f"{path}: missing {key} metadata")
    return mapping


def platform_tarball_url(name: str, version: str) -> str:
    return f"https://registry.npmjs.org/{name}/-/{name}-{version}.tgz"


def display_value(value: object) -> str:
    return value if isinstance(value, str) else "missing"


if __name__ == "__main__":
    raise SystemExit(main())
