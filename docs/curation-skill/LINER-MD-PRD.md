# LINER.md Product Requirements

Status: draft
Date: 2026-06-14
Branch: `go-refactor`

## Summary

Liner's first product promise is a focused mixtape: the right context for a
specific job. The next promise is a Liner project: a project folder that owns
that compiled corpus plus the instructions, skills, audits, impact tests, and
composition records that make the artifact useful over time.

`mixtape/MIXTAPE.md` remains the source bundle. It contains the curated corpus,
source notes, synthesis, and citations. Root-level `LINER.md` tells an external
AI how to think with that corpus, when to use which source, what skills are
available, what boundaries to respect, and how to keep the project honest as new
material is added.

This is not an "advisor" product in name or framing. A design-engineering
mixtape should not merely pretend to be a design engineer. It should behave like
a carefully grounded design-engineering artifact with a conscience: it has
stance, memory, source discipline, skills, and maintenance routines, all traced
back to the mixtape that created it.

## Product Thesis

A normal research bundle answers:

- What did we find?
- Which sources matter?
- What did the curator conclude?

A Liner project additionally answers:

- How should an AI use this corpus?
- What should it do first, second, and never?
- Which source should win when there is tension?
- What skills can it perform repeatedly?
- What does it know it does not know?
- How can it improve when the user adds a book, recording, article, or note?

The result should feel less like a static report and more like a reusable
working instrument.

## Goals

- Turn a compiled mixtape into an operational artifact that external agents can
  load and use.
- Preserve source grounding. `LINER.md` must point back to
  `mixtape/MIXTAPE.md`, `mixtape/tape.yaml`, source notes, and skill files.
- Let users keep improving a mixtape by adding books, recordings, local notes,
  web sources, and skills.
- Support contradictions, cleanup, audits, Impact Tests, and composition as
  first-class management workflows in the Go TUI.
- Keep the default creation path simple: compile the mixtape first, then build
  or manage `LINER.md` as a completed-mixtape action.

## Non Goals

- Do not rename the product around "advisor."
- Do not silently rewrite sources, skills, or `LINER.md` without review.
- Do not make `LINER.md` required before the Go TUI becomes the default.
- Do not treat generated persona text as truth. Everything must stay grounded
  in the mixtape's sources and curator notes.
- Do not build a complex database layer before Markdown/YAML artifacts prove the
  workflow.

## Naming Model

Use these names consistently:

- `LINER.md`: root project instructions for using the corpus.
- `liner.yaml`: root Liner project metadata and layout marker.
- `mixtape/`: the compiled corpus artifact folder.
- `mixtape/MIXTAPE.md`: the compiled corpus and source-grounded reading packet.
- `mixtape/tape.yaml`: the structured source list and corpus metadata.
- `mixtape/synthesis.md`: the curator's distilled understanding of the corpus.
- `mixtape/sources/`: fetched source extracts.
- `mixtape/working/`: methodology working files through assembly/compile.
- `skills/`: reusable behaviors or methods grounded in the corpus.
- `working/audits/`: contradiction, alignment, and quality reports.
- `working/evals/`: storage path for Impact Test tasks, runs, comparisons, and
  notes.
- `children/`: nested or referenced child mixtapes.
- `lineage.yaml`: merge/nesting history and parent-child relationships.

Avoid these as product labels:

- Advisor
- Agent persona
- Knowledge bot
- Chatbot

Acceptable user-facing language:

- LINER.md
- Project instructions
- Mixtape behavior
- Skills
- Source discipline
- Artifact with a conscience
- Liner project

## Core Mental Model

The Liner project is the durable folder.

`mixtape/` is the compiled corpus artifact.

`mixtape/MIXTAPE.md` is what the corpus knows.

`LINER.md` is how the project tells another AI to use that corpus.

`skills/*.md` are repeatable moves the project can perform.

Audits and Impact Tests are how the project proves it is still coherent.

Children and composition are how smaller mixtapes become a larger working
system without flattening every domain into one blob.

## File Structure

Target project shape:

```text
design-engineering/
|-- liner.yaml
|-- LINER.md
|-- skills/
|   |-- critique-interface-states.md
|   |-- design-terminal-flow.md
|   `-- evaluate-component-polish.md
|-- working/
|   |-- audits/
|   |   |-- 2026-06-14-contradictions.md
|   |   `-- 2026-06-14-skill-corpus-alignment.md
|   `-- evals/
|       |-- tasksets/
|       |-- runs/
|       `-- summaries/
|-- mixtape/
|   |-- tape.yaml
|   |-- MIXTAPE.md
|   |-- synthesis.md
|   |-- sources/
|   |   |-- 01-source.md
|   |   `-- ...
|   |-- local-sources/
|   `-- working/
|       |-- 01-jtbd-and-knowledge-map.md
|       |-- 02-candidate-longlist.md
|       |-- 03-evaluation.yaml
|       |-- 04-quality-checks.md
|       `-- 07-tape-draft.yaml
|-- children/
|   |-- ux-specialist.yaml
|   `-- ui-specialist.yaml
`-- lineage.yaml
```

Legacy folders with root-level `tape.yaml`, `synthesis.md`, `MIXTAPE.md`,
`sources/`, `local-sources/`, and methodology `working/` remain readable. New
projects should use the v2 shape above.

## LINER.md Structure

Initial generated `LINER.md` should be Markdown-first:

```md
# Design Engineering

## Scope

What this mixtape is for, what it is not for, and the job it serves.

## Operating Stance

The few strong positions this corpus takes.

## Source Use Rules

How to load, prioritize, and cite sources from MIXTAPE.md and sources/.

## Skills

The reusable moves available in skills/.

## Conflict Rules

How to handle contradictions between source types, time periods, or child
mixtapes.

## Abstention Rules

When to say the corpus is insufficient.

## Maintenance Rules

How new sources, recordings, books, and notes should be incorporated.

## Impact Test Rules

How to test whether this Liner project is actually improving outputs.
```

## Skill Structure

Skills should begin as plain Markdown files. Avoid front matter until the product
needs machine routing beyond filename plus headings.

Example:

```md
# Critique Interface States

## Use When

The user is reviewing an app screen, flow, or component and wants design
engineering critique.

## Inputs

- screenshot or UI description
- target user and workflow
- known constraints

## Method

1. Identify the user's actual job in the screen.
2. Check hierarchy, density, feedback, state coverage, and failure modes.
3. Compare against source rules from MIXTAPE.md.
4. Produce prioritized findings, not generic polish advice.

## Source Grounding

- MIXTAPE.md section: Interaction states and feedback
- Source: `sources/07-charmbracelet-bubbles.md`
- Source: `sources/12-apple-hig-feedback.md`

## Boundaries

Do not invent platform rules that are not in the corpus. If the issue needs
accessibility law, ask for a relevant source or mark it as outside scope.
```

## Primary Use Cases

### 1. Build A Design Engineering Liner Project

The user creates a design-engineering Liner project from public sources, local notes,
and curated examples. After compile, Liner generates:

- `mixtape/MIXTAPE.md`
- `LINER.md`
- starter skills such as terminal-flow critique, component polish review, and
  source-backed UI recommendation

When an external agent loads it, the agent gets not only sources but operating
rules for using them.

### 2. Add A Library Book

The user borrows a book, scans notes or adds chapter summaries, and imports them
as local sources.

Expected behavior:

1. TUI ingests the new material.
2. Liner classifies where the source fits.
3. Liner proposes changes to source notes, synthesis, `LINER.md`, and skills.
4. User reviews diffs.
5. Audit file records what changed and why.

### 3. Add A Recording Or Idea Dump

The user records thoughts after a project review.

Expected behavior:

1. Recording transcript is added as a local source.
2. Liner extracts claims, examples, and open questions.
3. Liner marks which claims are curator notes versus sourced facts.
4. `LINER.md` maintenance rules decide whether the recording should become a
   skill update, a source note, or a new impact-test task.

### 4. Clean Up Contradictions

The user asks the TUI to find inconsistencies between skills and corpus.

Expected behavior:

1. Agent writes `working/audits/<date>-contradictions.md`.
2. TUI shows findings in a table:
   - conflict
   - affected skill/source
   - recommendation
   - confidence
3. User opens details.
4. User applies reviewed changes only.
5. Audit records why one source or rule won over another.

### 5. Merge Or Nest Mixtapes

The user has:

- `design-engineering`
- `ux-specialist`
- `ui-specialist`

They create `product-design` as a parent.

Default behavior should prefer nesting first:

- `product-design` references children.
- Child scope remains intact.
- Parent `LINER.md` routes tasks to children.
- Merge is available only when the source sets are small and non-overlapping
  enough to stay coherent.

Example composition table:

```text
Child                Route                         Status
ux-specialist        research, flows, IA           ready
ui-specialist        hierarchy, layout, patterns   ready
design-engineering   states, systems, polish       warning
```

## Go TUI Surfaces

The Go TUI should manage Liner projects after compile.

Target Project actions:

```text
Key       Action              When
enter     Preview MIXTAPE.md  compiled artifact ready
l         Manage LINER.md     project instructions
k         Skills              reusable methods
a         Audits              contradictions and alignment
e         Impact Tests        compare output quality
m         Compose             merge or nest child mixtapes
c         Recompile           explicit rebuild
```

Use tables for management:

- source inventory
- skills list
- audit findings
- impact-test variants
- child mixtapes

Use viewports for:

- `LINER.md`
- skill bodies
- audit reports
- impact-test outputs

Avoid card-heavy dashboard layouts. This should feel like a workshop with a
clear next action.

## Generation Workflow

`LINER.md` generation should be a completed-mixtape action, not a required
creation phase.

Flow:

```text
Project
-> Manage LINER.md
-> Generate LINER.md
-> Review proposed LINER.md
-> Accept / edit / discard
```

Inputs:

- `mixtape/MIXTAPE.md`
- `mixtape/synthesis.md`
- `mixtape/tape.yaml`
- source notes and kinds
- `mixtape/working/04-quality-checks.md`

Outputs:

- `LINER.md`
- optional starter `skills/*.md`
- `working/audits/<date>-liner-md-generation.md`

Review rules:

- Generated files are proposed first.
- User can open files in editor.
- Accept writes files.
- Discard removes draft files.
- Audit explains the decisions and limitations.

## Audit Requirements

Initial audits:

1. Contradiction audit.
2. Skill-corpus alignment audit.
3. Source-note quality audit.

Contradiction audit should find:

- two skills giving incompatible instructions
- a skill that contradicts source hierarchy
- source notes that overstate what a source supports
- parent and child mixtapes with conflicting scope

Skill-corpus alignment should check:

- every skill points to at least one source or section
- skill boundaries are explicit
- skill method does not exceed the corpus
- old skills are marked stale when sources change materially

Source-note quality should check:

- notes say what to read for
- notes name the bar the source sets
- notes flag limitations or scope boundaries
- notes distinguish curator stance from source claim

## Impact Test Requirements

Start local and simple.

Variants:

- no project files
- `MIXTAPE.md` only
- `MIXTAPE.md + LINER.md`
- `MIXTAPE.md + LINER.md + skills/*.md`

Artifacts:

```text
working/evals/tasksets/
working/evals/runs/<timestamp>/
working/evals/summaries/
```

Initial scoring:

- human rubric first
- optional LLM judge later
- record qualitative notes, not just scores

Example impact-test task:

```md
# Impact Test Task: Terminal Flow Critique

Input: a screenshot or text description of a setup wizard.

Expected output:

- identifies the user's job
- finds contradictory helper text
- proposes a simpler linear flow
- cites or names the relevant source rule
- avoids generic design advice
```

## Acceptance Criteria

Phase 1:

- Existing compiled project can show `LINER.md` status in Project.
- User can generate a draft `LINER.md`.
- Draft is reviewed before write.
- Accepted `LINER.md` is visible in a viewport.
- Generation writes an audit note.

Phase 2:

- User can see skills in a table.
- User can open, create, disable, and audit skills.
- Skill-corpus alignment audit detects ungrounded skills.

Phase 3:

- User can run contradiction and source-note quality audits.
- Findings appear in a table with inspectable report bodies.
- Applying changes is review-only, never silent.

Phase 4:

- User can define a tiny local impact-test taskset.
- User can run or prepare at least two variants from the run packet.
- Results are saved and summarized.

Phase 5:

- User can nest child mixtapes under a parent.
- Parent `LINER.md` routes tasks to children.
- `lineage.yaml` records composition history.

## Open Questions

1. Should `LINER.md` generation use the existing methodology skill bundle or a
   new smaller project skill?
2. Should starter skills live in `skills/*.md` only, or should `LINER.md`
   include short embedded skill summaries?
3. What is the minimum useful Impact Tests UI: task table only, or side-by-side
   output comparison on day one?
4. When a parent mixtape nests children, should sources be copied, referenced,
   or both?
5. Should external agents load `LINER.md` first, then `MIXTAPE.md`, or should
   `LINER.md` explicitly instruct when to load deeper source files?

## Recommended First Build

Build the smallest useful loop:

1. Detect `LINER.md` on Project.
2. Add a `LINER.md` preview route.
3. Add a generate action that writes a draft file under `working/`.
4. Add review accept/discard.
5. Write one audit file explaining generation decisions.

Do not start with merge/nesting. Composition becomes valuable after individual
Liner projects have a stable `LINER.md` and at least one audit or
Impact Test loop.
