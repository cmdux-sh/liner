"""Write the prompts the terminal navigator hands to your agent. Nothing runs from here."""

from __future__ import annotations

import re
from typing import Any

HANDOFF_CONTRACT = "liner.handoff"
CONTRACT_VERSION = 1
_SLUG = re.compile(r"[\w][\w.-]*", re.ASCII)
_MAX_SUBJECT_CHARS = 80
CONTEXTS = ("start", "liner", "approval", "finding", "gap", "changed", "synthesis", "source")


def handoff_prompts(
    context: str, slug: str = "", *, line: int = 0, subject: str = ""
) -> dict[str, Any]:
    if context not in CONTEXTS:
        raise ValueError(f"Unknown context {context}. Use one of: {', '.join(CONTEXTS)}.")
    if slug and not _SLUG.fullmatch(slug):
        raise ValueError(f"{slug} is not a liner name.")
    return {
        "contract": HANDOFF_CONTRACT,
        "version": CONTRACT_VERSION,
        "context": context,
        "options": [
            {"label": label, "prompt": prompt}
            for label, prompt in _options(context, slug, line, _quoted(subject))
        ],
    }


def _quoted(subject: str) -> str:
    """A source's title comes from the web, so it goes in as a short quotation."""
    words = " ".join(subject.split()).replace('"', "'")
    return f'"{words[:_MAX_SUBJECT_CHARS]}"' if words else "this source"


def _options(context: str, slug: str, line: int, subject: str) -> list[tuple[str, str]]:
    specific = {
        "start": [
            (
                "Start a new liner",
                "I want lasting research for this job: when I ___, I want ___, so I can ___. "
                "Use Liner to build it.",
            ),
        ],
        "approval": [
            (
                "Review what's waiting",
                f"Continue the {slug} liner with `liner research next`, and show me what "
                "needs my approval.",
            ),
        ],
        "finding": [
            ("Go through the findings", f"Show me the findings waiting in {slug}, one at a time."),
        ],
        "gap": [
            ("Find answers", f"Use `liner research improve {slug}` to answer its open gaps."),
        ],
        "changed": [
            (
                "See what changed",
                f"Run `liner refresh {slug}`, read the diffs, and tell me what changed "
                "and whether the synthesis still holds.",
            ),
        ],
        "synthesis": [
            (
                "Check this claim",
                f"In {slug}, is the claim on line {line} of the synthesis fairly supported?",
            ),
            ("Change something", f"In {slug}, revise the synthesis: "),
            (
                "Find more evidence",
                f"Find stronger sources for the claims in {slug} that rest on one source.",
            ),
        ],
        "source": [
            (
                "Ask about this source",
                f"In {slug}, what does {subject} contribute to the synthesis?",
            ),
            (
                "Check if it's current",
                f"Run `liner refresh {slug}` and tell me if {subject} changed.",
            ),
        ],
    }
    general = [
        ("Use this liner", f"Use the {slug} liner for this task: "),
        ("Improve this liner", f"Use `liner research improve {slug}` to strengthen it."),
        ("Test this liner", f"Run a with-and-without test on {slug}. Let me pick the verdict."),
    ]
    return [*specific.get(context, []), *([] if context == "start" else general)]
