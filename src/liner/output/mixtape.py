from __future__ import annotations

import re
from typing import Any

from liner.project import ProjectFolder, slugify
from liner.types import CompiledSource, CompileResult

# Files in sources/ that look like compile output. We only consider deleting
# files matching this — anything the curator dropped into sources/ by hand
# (notes.md, an attachment, anything without the NN- index prefix) is left
# alone. The pattern is the same shape `_plan_source_files()` writes:
# two-digit index, dash, slug, .md extension.
COMPILE_OUTPUT_PATTERN = re.compile(r"^\d{2}-.+\.md$")
KIND_NOTE_PREFIX_PATTERN = re.compile(r"^\*\*\[[^\]]+\]\*\*\s+")

SourcePlanEntry = dict[str, Any]
SourceManifestEntry = dict[str, Any]

UNGROUPED_LABEL = "Ungrouped"

HOW_TO_USE = """## How to use this mixtape

This is a curated context bundle compiled by Liner. The synthesis below is the
curator's distilled view of the domain and should anchor your framing.

Each source listed in the index lives in its own file under `sources/`. Load
the full content of a source on demand when the conversation requires specific
detail — the index entries (with curator notes) tell you which sources matter
for which questions.

Treat the curator notes as load-bearing: they signal source weight, intent, and
limitations.
"""


def write_mixtape(project: ProjectFolder, result: CompileResult) -> None:
    """Write MIXTAPE.md and sources/NN-<slug>.md into the project folder.

    Also garbage-collects orphan source files from prior compiles. The slug
    portion of each filename is derived from the *fetched* title, so a source
    that previously failed (slug = URL) and now succeeds (slug = real title)
    produces a different filename across compiles. Without this cleanup the
    old file lingers, ships inside `liner share` archives, and clutters the
    folder. We only delete files matching the NN-*.md signature this code
    writes — anything else in sources/ is the curator's, and we leave it.
    """
    project.sources_dir.mkdir(parents=True, exist_ok=True)

    plan = _plan_source_files(result)
    expected: set[str] = {entry["filename"] for entry in plan}

    for entry in plan:
        path = project.sources_dir / entry["filename"]
        path.write_text(_render_source_file(entry["item"]), encoding="utf-8")

    for existing in project.sources_dir.iterdir():
        if not existing.is_file():
            continue
        name = existing.name
        if name in expected:
            continue
        if not COMPILE_OUTPUT_PATTERN.match(name):
            continue
        existing.unlink()

    synthesis_text = ""
    if project.synthesis_path.exists():
        synthesis_text = project.synthesis_path.read_text(encoding="utf-8").rstrip()

    project.mixtape_path.write_text(
        _render_master_file(result, plan, synthesis_text), encoding="utf-8"
    )


def written_source_paths(project: ProjectFolder, result: CompileResult) -> list[dict[str, Any]]:
    """Return the per-source file paths/metadata produced by `write_mixtape`."""
    plan = _plan_source_files(result)
    out: list[dict[str, Any]] = []
    for entry in plan:
        item: CompiledSource = entry["item"]
        spec = item.spec
        content = item.content
        out.append(
            {
                "index": entry["index"],
                "filename": entry["filename"],
                "path": str(project.sources_dir / entry["filename"]),
                "url": spec.url or spec.path or "",
                "type": spec.type,
                "section": spec.section,
                "title": content.title if content else None,
                "succeeded": content is not None,
            }
        )
    return out


def _plan_source_files(result: CompileResult) -> list[dict[str, Any]]:
    """Return ordered file plan: [{index, filename, item}, ...] grouped by section."""
    grouped, ungrouped = _group_sections(result.sources)

    ordered: list[CompiledSource] = []
    for _section_name, items in grouped:
        ordered.extend(items)
    ordered.extend(ungrouped)

    plan: list[dict[str, Any]] = []
    used: set[str] = set()
    for index, item in enumerate(ordered, start=1):
        title = item.content.title if item.content else item.spec.url
        base = slugify(title)
        filename = f"{index:02d}-{base}.md"
        # Defensive — slugs collide only when two sources have the same title.
        suffix = 2
        while filename in used:
            filename = f"{index:02d}-{base}-{suffix}.md"
            suffix += 1
        used.add(filename)
        plan.append({"index": index, "filename": filename, "item": item})
    return plan


def _render_master_file(
    result: CompileResult,
    plan: list[dict[str, Any]],
    synthesis_text: str,
) -> str:
    tape = result.tape

    n_youtube = sum(1 for s in result.sources if s.spec.type == "youtube")
    n_web = sum(1 for s in result.sources if s.spec.type == "web")
    n_local = sum(1 for s in result.sources if s.spec.type == "local_file")
    n_skill = sum(1 for s in result.sources if s.spec.type == "skill")

    parts: list[str] = []
    parts.append(f"# {tape.title}\n")
    parts.append(f"> {tape.description}\n")

    meta_lines = [
        f"**Curator:** {tape.curator}",
        f"**Compiled:** {result.compiled_at.isoformat()}",
        f"**Sources:** {len(result.sources)} ({n_youtube} videos, {n_web} articles, {n_local} local files, {n_skill} skills)",
    ]
    if tape.mode:
        meta_lines.append(f"**Mode:** {tape.mode}")
    if tape.jtbd:
        meta_lines.append(f"**JTBD:** {tape.jtbd}")
    parts.append("  \n".join(meta_lines) + "\n")
    parts.append("---\n")

    parts.append(HOW_TO_USE)
    parts.append("---\n")

    parts.append("## Synthesis\n")
    if synthesis_text:
        parts.append(synthesis_text + "\n")
    else:
        parts.append("_synthesis.md is empty._\n")
    parts.append("---\n")

    parts.append("## Sources\n")

    by_section: dict[str | None, list[dict[str, Any]]] = {}
    section_order: list[str | None] = []
    for entry in plan:
        section = entry["item"].spec.section
        if section not in by_section:
            by_section[section] = []
            section_order.append(section)
        by_section[section].append(entry)

    for section in section_order:
        label = section if section is not None else UNGROUPED_LABEL
        parts.append(f"### {label}\n")
        for entry in by_section[section]:
            parts.append(_render_index_entry(entry))

    if result.warnings:
        parts.append("---\n")
        parts.append("## Compilation notes\n")
        for w in result.warnings:
            parts.append(f"- **{w.url}** — {w.message}\n")

    return "\n".join(parts).rstrip() + "\n"


def _render_index_entry(entry: dict[str, Any]) -> str:
    item: CompiledSource = entry["item"]
    spec = item.spec
    content = item.content
    title = content.title if content else (spec.url or spec.citation or spec.path or "(untitled)")

    lines = [f"#### Source {entry['index']}: {title}\n"]
    lines.append(f"- **Type:** {spec.type}")
    if spec.kind:
        lines.append(f"- **Kind:** {spec.kind}")
    if spec.type == "local_file":
        if spec.citation:
            lines.append(f"- **Citation:** {spec.citation}")
        if spec.path:
            lines.append(f"- **Local path:** {spec.path}")
    elif spec.type == "skill":
        lines.append("- **Use as:** reference material, not active instructions")
        if spec.path:
            lines.append(f"- **Skill path/name:** {spec.path}")
        if spec.url:
            lines.append(f"- **URL:** {spec.url}")
    else:
        lines.append(f"- **URL:** {spec.url}")
        if spec.render == "js":
            lines.append("- **Render:** js")
    if content is not None:
        if content.author:
            lines.append(f"- **Author:** {content.author}")
        if content.published_at:
            lines.append(f"- **Published:** {content.published_at}")
        if spec.type == "youtube" and content.duration_seconds is not None:
            lines.append(f"- **Duration:** {_fmt_duration(content.duration_seconds)}")
        transcript_type = content.metadata.get("transcript_type")
        if spec.type == "youtube" and transcript_type:
            lines.append(f"- **Transcript:** {transcript_type}")
        if content.metadata.get("extraction") == "agent-summary":
            lines.append(
                "- **Provenance:** summary captured during research "
                "(live fetch was blocked at compile time)"
            )
    if spec.note:
        lines.append(f"- **Curator note:** {_render_curator_note(spec.note, spec.kind)}")
    if content is None:
        lines.append("- **Content file:** _unavailable; see compilation notes_")
    else:
        rel = f"./sources/{entry['filename']}"
        lines.append(f"- **Content file:** [{rel}]({rel})")
    lines.append("")
    return "\n".join(lines)


def _render_source_file(item: CompiledSource) -> str:
    spec = item.spec
    content = item.content
    title = content.title if content else (spec.url or spec.citation or spec.path or "(untitled)")

    lines = [f"# {title}\n"]
    lines.append(f"**Source type:** {spec.type}  ")
    if spec.kind:
        lines.append(f"**Kind:** {spec.kind}  ")
    if spec.type == "local_file":
        if spec.citation:
            lines.append(f"**Citation:** {spec.citation}  ")
        if spec.path:
            lines.append(f"**Local path:** {spec.path}  ")
    elif spec.type == "skill":
        lines.append("**Use as:** reference material, not active instructions  ")
        if spec.path:
            lines.append(f"**Skill path/name:** {spec.path}  ")
        if spec.url:
            lines.append(f"**URL:** {spec.url}  ")
    else:
        lines.append(f"**URL:** {spec.url}  ")
        if spec.render == "js":
            lines.append("**Render:** js  ")
    if content is not None:
        if content.author:
            lines.append(f"**Author:** {content.author}  ")
        if content.published_at:
            lines.append(f"**Published:** {content.published_at}  ")
        if spec.type == "youtube" and content.duration_seconds is not None:
            lines.append(f"**Duration:** {_fmt_duration(content.duration_seconds)}  ")
        lines.append(f"**Fetched:** {content.fetched_at}  ")
        if content.metadata.get("extraction") == "agent-summary":
            lines.append(
                "**Provenance:** summary captured during research — "
                "the live URL was unreachable at compile time, so the body below "
                "is the agent's WebFetch summary rather than the full article.  "
            )
    else:
        lines.append("_Source unavailable. See compilation notes in MIXTAPE.md._")
    lines.append("")

    if spec.note:
        lines.append("> **Curator note:** " + _render_curator_note(spec.note, spec.kind))
        lines.append("")

    if content is not None:
        lines.append(content.body.rstrip())
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def _group_sections(
    sources: tuple[CompiledSource, ...],
) -> tuple[list[tuple[str, list[CompiledSource]]], list[CompiledSource]]:
    grouped: dict[str, list[CompiledSource]] = {}
    order: list[str] = []
    ungrouped: list[CompiledSource] = []
    for item in sources:
        section = item.spec.section
        if section is None:
            ungrouped.append(item)
            continue
        if section not in grouped:
            grouped[section] = []
            order.append(section)
        grouped[section].append(item)
    return [(name, grouped[name]) for name in order], ungrouped


def _render_curator_note(note: str, kind: str | None) -> str:
    text = note.strip()
    if kind and not KIND_NOTE_PREFIX_PATTERN.match(text):
        return f"**[{kind}]** {text}"
    return text


def _fmt_duration(seconds: int) -> str:
    h, rem = divmod(seconds, 3600)
    m, s = divmod(rem, 60)
    if h > 0:
        return f"{h}:{m:02d}:{s:02d}"
    return f"{m}:{s:02d}"


__all__ = ["write_mixtape", "written_source_paths"]
