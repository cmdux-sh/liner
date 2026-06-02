from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass
from pathlib import Path

from liner.tape import STARTER_TAPE

SYNTHESIS_PLACEHOLDER = """# Synthesis

> Replace this placeholder with the curator's distilled understanding of the domain
> (typically 800–2000 words). The synthesis is copied verbatim into `MIXTAPE.md`
> when you run `liner compile` and is the first thing the consuming AI reads.

## The framework I see in this domain

TODO — the principles, distinctions, and lenses you use to think about this topic.

## Contested questions and where I stand

TODO — places experts disagree, and the position this mixtape takes.

## When to use this mixtape (and when to look elsewhere)

TODO — what the corpus is good for, what it doesn't cover.
"""

WORKING_JTBD_TEMPLATE = """# JTBD and knowledge map

## Job-to-be-done

TODO — a single specific Job Story. Not the topic — the use case. Required form:
`When [circumstance], I want [motivation], so I can [outcome].` All three slots required.
Examples:
- "When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic."
- "When I review mobile design portfolios as a senior IC hiring manager, I want to compare candidates against a consistent rubric of taste and decision-making, so I can decide who to advance with confidence."

## Knowledge map

TODO — Phase 1 replaces this with 4–8 sections, each with sub-areas. The
example bullets below are placeholders; the agent (or you) will revise them
to reflect the actual structure of the domain.

- Foundations
  - …
- Patterns
  - …
- Craft
  - …
"""

WORKING_LONGLIST_TEMPLATE = """# Candidate long-list

The unfiltered pool of candidate sources from Phase 2. URLs and titles only — no fetching yet.

Group by knowledge-map section. Quantity over precision; you'll cut in Phase 4.

## Section: foundations

- [ ] https://example.com/...

## Section: patterns

- [ ] https://example.com/...
"""

WORKING_EVALUATION_TEMPLATE = """# Evaluation — keep/trim/drop decisions
#
# One entry per candidate from Phase 2. Decisions and rationales from Phase 4
# (after the AI has actually read the fetched content).

candidates: []
#  - url: https://example.com/great-article
#    decision: kept            # kept | trimmed | dropped
#    section: foundations
#    rating: 5                  # 1-5
#    rationale: Canonical reference for the foundations section.
"""

WORKING_QUALITY_TEMPLATE = """# Quality checks (Phase 5)

Run each test deliberately. Document findings even when "nothing to do."

## Redundancy test

TODO — any two sources making essentially the same point? Cut the weaker one.

## Coverage test

TODO — any bucket in the knowledge map with zero sources? Fill it or explicitly note the omission.

## Disagreement test

TODO — strongest claim in the corpus. Is there a credible counter? Include it or note the position taken.

## Framing-gap test

TODO — step back. Is there a whole way of thinking about this JTBD that's missing? If yes, revise the knowledge map and revisit Phase 2.
"""


@dataclass(frozen=True, slots=True)
class ProjectFolder:
    path: Path

    @property
    def tape_path(self) -> Path:
        return self.path / "tape.yaml"

    @property
    def synthesis_path(self) -> Path:
        return self.path / "synthesis.md"

    @property
    def mixtape_path(self) -> Path:
        return self.path / "MIXTAPE.md"

    @property
    def sources_dir(self) -> Path:
        return self.path / "sources"

    @property
    def working_dir(self) -> Path:
        return self.path / "working"

    @property
    def personal_dir(self) -> Path:
        return self.path / "personal"

    def is_valid(self) -> bool:
        return self.tape_path.exists()

    def has_synthesis(self) -> bool:
        return self.synthesis_path.exists()


def slugify(text: str, max_length: int = 60) -> str:
    """Lowercase, ASCII, hyphen-separated. Safe for filesystem use."""
    if not text:
        return "untitled"
    normalized = unicodedata.normalize("NFKD", text)
    ascii_text = normalized.encode("ascii", "ignore").decode("ascii")
    ascii_text = ascii_text.lower()
    ascii_text = re.sub(r"[^a-z0-9]+", "-", ascii_text)
    ascii_text = ascii_text.strip("-")
    if not ascii_text:
        return "untitled"
    if len(ascii_text) > max_length:
        ascii_text = ascii_text[:max_length].rstrip("-")
    return ascii_text


def init_project(path: Path, *, force: bool = False) -> ProjectFolder:
    """Create a project folder with starter tape, synthesis placeholder, and working/ stubs.

    `path` may be an existing or new directory. If it does not exist it is created.
    """
    if path.exists() and not path.is_dir():
        raise FileExistsError(f"{path} exists and is not a directory")
    path.mkdir(parents=True, exist_ok=True)

    project = ProjectFolder(path)

    if project.tape_path.exists() and not force:
        raise FileExistsError(
            f"{project.tape_path} already exists. Use --force to overwrite."
        )

    project.tape_path.write_text(STARTER_TAPE, encoding="utf-8")

    if not project.synthesis_path.exists() or force:
        project.synthesis_path.write_text(SYNTHESIS_PLACEHOLDER, encoding="utf-8")

    project.working_dir.mkdir(parents=True, exist_ok=True)
    _write_if_missing(
        project.working_dir / "01-jtbd-and-knowledge-map.md",
        WORKING_JTBD_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "02-candidate-longlist.md",
        WORKING_LONGLIST_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "03-evaluation.yaml",
        WORKING_EVALUATION_TEMPLATE,
        force=force,
    )
    _write_if_missing(
        project.working_dir / "04-quality-checks.md",
        WORKING_QUALITY_TEMPLATE,
        force=force,
    )

    return project


def _write_if_missing(path: Path, content: str, *, force: bool) -> None:
    if path.exists() and not force:
        return
    path.write_text(content, encoding="utf-8")
