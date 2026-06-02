---
name: curating-mixtapes
description: Use when the user wants to build a Liner mixtape: a curated project folder of sources, notes, synthesis, and compiled AI context for a specific job-to-be-done. Drives the current Liner authoring phases and writes ordinary project files ready for `liner compile`.
---

# Curating Mixtapes

This skill helps author a Liner mixtape project folder. It is used by agent-assisted TUI phases and can also be run directly by an agent with filesystem access.

The skill writes authoring artifacts. It does not run `liner compile`; the TUI or user runs compile separately.

## Terms

- **Tape:** `tape.yaml`, the source recipe.
- **Mixtape project:** the full folder: recipe, synthesis, working notes, compiled output.
- **Compiled mixtape:** `MIXTAPE.md` plus `sources/*.md`, produced by `liner compile`.

## Output

Produce or update:

- `working/01-jtbd-and-knowledge-map.md`
- `working/02-candidate-longlist.md`
- `working/03-evaluation.yaml`
- `working/04-quality-checks.md`
- `synthesis.md`
- `tape.yaml`

Compile later produces:

- `MIXTAPE.md`
- `sources/`

## Companion Files

Read these files from this skill directory when the phase needs them:

- `source-finding-tactics.md` before Candidate discovery
- `source-quality-hierarchy.md` and `curator-notes.md` before Evaluation
- `quality-check-tests.md` before Quality checks
- `synthesis-guidance.md` before Synthesis

## Modes

Default to **quick** mode unless the project is already marked `mode: methodology` or the user explicitly asks for deeper review.

Quick mode means the agent does more drafting and the user confirms gates lightly. Methodology mode means the user is expected to review, rewrite, and make judgment calls. Both modes produce the same file shape.

## Phases

Run only the requested phase. Do not jump ahead.

### Framing

Read `tape.yaml`, `working/01-jtbd-and-knowledge-map.md`, and any JTBD clarifications.

The JTBD should be a single Job Story:

```text
When [circumstance], I want [motivation], so I can [outcome].
```

If the JTBD is a bare topic, a multi-job sentence, or too vague to guide source selection, say so and stop. Otherwise write a knowledge map with 4-8 sections and a few sub-areas per section.

Write `working/01-jtbd-and-knowledge-map.md`. Stop after this phase.

### Candidate Discovery

Read `working/01-jtbd-and-knowledge-map.md` and `source-finding-tactics.md`.

Produce a broad candidate list grouped by knowledge-map section. Include titles, URLs, and one-line reasons. Verify URLs before keeping them in the list. Search YouTube as well as the web when the domain likely has talks, demos, lectures, or interviews.

Write `working/02-candidate-longlist.md`. Stop after this phase.

### Evaluation

Read the framing file, candidate list, `source-quality-hierarchy.md`, and `curator-notes.md`.

Fetch and read candidate sources as needed. Decide `kept`, `trimmed`, or `dropped`. For kept or trimmed sources, include rating, section, rationale, and a specific curator note.

Write `working/03-evaluation.yaml`. Stop after this phase.

### Quality Checks

Read `working/03-evaluation.yaml` and `quality-check-tests.md`.

Run the redundancy, coverage, disagreement, framing-gap, and source-kind checks. If the corpus has a known weakness, document the weakness and the recommended next action.

Write `working/04-quality-checks.md`. Stop after this phase.

### Synthesis

Read the framing, evaluation, quality checks, and `synthesis-guidance.md`.

Write `synthesis.md` as a real framing document, not a source-by-source recap. It should explain the structure of the domain, important distinctions, contested questions, and where this mixtape is useful.

Stop after this phase.

### Assembly

Write `tape.yaml`.

Required fields:

- `title`
- `description`
- `version: 1`
- `curator`
- `mode`
- `sources`

Supported source types:

- `web` with `url`
- `youtube` with `url`
- `local_file` with `path` under `personal/` and `citation`

Optional source fields include `note`, `section`, `priority`, `kind`, and `render` for web sources.

Validate that YAML parses and that every kept source has the fields required by its source type. Stop after this phase.

## Boundaries

- Do not invent sources or citations.
- Do not silently keep broken URLs.
- Do not include local files unless the user supplied them or they already exist under `personal/`.
- Do not claim the mixtape is compiled until `liner compile` has run.
- Do not publish or submit anything.
