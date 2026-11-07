"""Read the citations in a synthesis, so a reader can follow a claim to its source.

A citation is a Markdown footnote: `claim.[^apple]`, with `[^apple]: <url or path>`
on its own line. Older liners have none, and they read the same as before.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from liner.output.mixtape import COMPILE_OUTPUT_PATTERN
from liner.project import ProjectFolder

CITATIONS_CONTRACT = "liner.citations"
CONTRACT_VERSION = 1

_DEFINITION = re.compile(r"^\[\^([\w-]+)\]:[ \t]+<?([^\s>]+)>?.*$", re.MULTILINE)
_REFERENCE = re.compile(r"\[\^([\w-]+)\](?!:)")
_HEADER_LOCATOR = re.compile(r"^\*\*(?:URL|Local path|Skill path/name):\*\* (.+?)\s*$")
_HEADER_CITATION = re.compile(r"^\*\*Citation:\*\* (.+?)\s*$")
_CITATION_URL = re.compile(r"https?://[^\s<>\"')\]]+")


@dataclass(frozen=True, slots=True)
class ParsedCitations:
    targets: dict[str, str]
    references: dict[str, list[int]]


def parse_citations(text: str) -> ParsedCitations:
    targets = {match.group(1): match.group(2) for match in _DEFINITION.finditer(text)}
    references: dict[str, list[int]] = {}
    for number, line in enumerate(text.splitlines(), start=1):
        for label in _REFERENCE.findall(line):
            references.setdefault(label, []).append(number)
    return ParsedCitations(targets, references)


def list_citations(project: ProjectFolder) -> dict[str, Any]:
    """Match each citation to its compiled source file. A missing match is reported."""
    text = project.synthesis_path.read_text(encoding="utf-8", errors="replace")
    parsed = parse_citations(text)
    files = citation_files(project, parsed)
    return {
        "contract": CITATIONS_CONTRACT,
        "version": CONTRACT_VERSION,
        "slug": project.path.name,
        "synthesis_path": str(project.synthesis_path),
        "citations": [
            {
                "label": label,
                "source": target,
                "lines": parsed.references.get(label, []),
                "file": files[target],
            }
            for label, target in parsed.targets.items()
        ],
    }


def citation_files(
    project: ProjectFolder, parsed: ParsedCitations | None = None
) -> dict[str, str | None]:
    """Each citation target in the synthesis and the compiled file it opens, or None.

    The CLI, the build check, and the navigator all use this, so they agree.
    """
    if parsed is None:
        if not project.synthesis_path.is_file():
            return {}
        text = project.synthesis_path.read_text(encoding="utf-8", errors="replace")
        parsed = parse_citations(text)
    files = compiled_source_files(project)
    normalized = {_citation_key(locator): path for locator, path in files.items()}
    resolved: dict[str, str | None] = {}
    for target in parsed.targets.values():
        path = files.get(target) or normalized.get(_citation_key(target))
        resolved[target] = str(path) if path else None
    return resolved


def _citation_key(locator: str) -> str:
    """The same key the research check uses, so the build and the check agree."""
    from liner.research_checks import _citation_key as research_key

    return research_key(locator)


def unlinked_citations(project: ProjectFolder) -> list[str]:
    """Labels whose citation opens no compiled file."""
    if not project.synthesis_path.is_file():
        return []
    text = project.synthesis_path.read_text(encoding="utf-8", errors="replace")
    parsed = parse_citations(text)
    files = citation_files(project, parsed)
    return [label for label, target in parsed.targets.items() if files[target] is None]


def compiled_source_files(project: ProjectFolder) -> dict[str, Path]:
    """Map each compiled file's URL, local path, or skill path to the file.

    A local copy of a page also answers to any address in its citation, unless
    another file is that address's real source.
    """
    if not project.sources_dir.is_dir():
        return {}
    files: dict[str, Path] = {}
    cited: dict[str, Path] = {}
    for path in sorted(project.sources_dir.iterdir()):
        if not path.is_file() or not COMPILE_OUTPUT_PATTERN.match(path.name):
            continue
        with path.open(encoding="utf-8", errors="replace") as handle:
            for line in handle:
                if line.startswith("**Fetched:** "):
                    break
                match = _HEADER_LOCATOR.match(line)
                if match:
                    files[match.group(1)] = path
                citation = _HEADER_CITATION.match(line)
                if citation:
                    for url in _CITATION_URL.findall(citation.group(1)):
                        cited.setdefault(url.rstrip(".,;:"), path)
    return cited | files
