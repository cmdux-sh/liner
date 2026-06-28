# Product

## Register

product

## Users

Liner is for people who already use AI tools seriously enough to care about the
quality of the context they feed them: designers, engineers, researchers,
founders, technical operators, and agent builders who need reusable source
bundles for a specific job.

They are usually working locally, inside a terminal or editor, with a real
project folder on disk. They may be creating a new corpus, importing an
existing project, adding private notes, generating project instructions,
checking a skill against the corpus, or composing several related projects.
Their context is practical and interruptible: they need to see what exists, what
will be written, what is only a recommendation, and what evidence supports each
state.

The best Liner users are source-sensitive. They know a generic AI answer is
often less useful than an answer grounded in the right source set, curator
notes, synthesis, and project instructions. Liner should help them do that work
without turning the product into a hosted knowledge base, chat persona, or
opaque automation layer.

## Product Purpose

Liner turns a focused body of sources into a durable local project that AI
sessions can actually use. The first promise is a useful `MIXTAPE.md`: a
compiled, source-grounded context packet for one job to be done. The next
promise is a Liner project: a folder that owns that context plus `LINER.md`, a
default Project Skill, status, composition records, and maintenance history.

Success means the user can create, inspect, improve, import, and reuse a
Liner project without losing provenance or judgment. The product should make the
following distinctions obvious:

- The Liner project is the durable working folder.
- `mixtape/MIXTAPE.md` is the compiled knowledge/context packet.
- `LINER.md` is the instruction layer that tells AI sessions how to use
  the corpus.
- The Project Skill is the root `SKILL.md` created by default so future AI
  sessions can load the project directly.
- Status and composition records help the project stay inspectable over time.

The product is local-first. The core does not require accounts, telemetry,
hosted compilation, or hidden model calls. AI can assist with discovery,
evaluation, drafting, and maintenance, but source grounding and user review are
the trust model.

## Brand Personality

Precise, grounded, calm.

Liner should feel like a serious local instrument for building source-backed AI
context. The voice is direct and concrete: folders, sources, notes, synthesis,
`MIXTAPE.md`, `LINER.md`, Project Skills, status, children, lineage. It should
not lean on persona language or vague AI productivity claims.

The emotional goal is quiet confidence. The user should feel that Liner is
careful enough to trust, dense enough to manage real artifacts, and explicit
enough to avoid accidental writes.

## Anti-references

- Do not frame Liner as an `advisor`, chatbot, knowledge bot, or agent persona.
- Do not collapse a completed `MIXTAPE.md` into full Liner project completion.
- Do not treat generated text as truth without source grounding and review.
- Do not silently rewrite sources, skills, audits, or `LINER.md`.
- Do not use dashboard language when the product object is a local project
  folder and plain-text artifacts.
- Do not make the terminal UI feel like a marketing page, decorative terminal
  toy, wizard that disappears after setup, command dump, or card-heavy admin
  dashboard.
- Do not reintroduce duplicate navigation surfaces, `:` Commands, in-body key
  strips, or implementation nouns where product nouns are clearer.
- Do not bury unresolved product decisions in UI copy. Track unknowns directly.

## Design Principles

1. **Artifact First.** Start from the files the user can inspect, diff, move,
   and paste. Product language should name the artifact and its role before it
   names any agent behavior.
2. **Provenance Over Persona.** Liner earns trust through source notes,
   synthesis, citations, audits, and reviewed writes. Avoid pretending the
   product is a person.
3. **One Surface, One Job.** Every TUI screen needs a crisp job: command hub,
   project browser, section workspace, preference chooser, file picker, review
   surface, or audit table.
4. **Visible Writes.** Any action that creates, updates, deletes, imports,
   applies, or discards files must make that behavior clear before it happens.
   Preview and read-only routes must stay side-effect-free.
5. **Evidence-Based Readiness.** Status should come from files, command output,
   audits, and explicit decisions, not from optimistic labels.

## Accessibility & Inclusion

The terminal UI should target readable contrast across common dark terminals.
Primary content uses light ink, labels and descriptions use muted but readable
gray, and orange is reserved for selection, action, and primary state. Purple is
not a Liner action color.

The product is keyboard-first by nature. Every non-text screen needs clear
footer-owned escape routes, especially `h home`, `esc back`, `q quit`, and `?`
help where appropriate. Text-entry screens can reserve letters for input, but
non-text screens should not hide navigation.

Long paths, job stories, descriptions, and source titles must wrap inside their
own column or viewport instead of bleeding into adjacent UI. The TUI should
remain usable in narrow terminals, with stable widths, truncation, and wrapped
label/value rows.

Generated or agent-assisted changes require review. This is an inclusion rule as
much as a safety rule: users need to understand what changed, why it changed,
and what source evidence supported it.

## Core Product Model

The target v2 project shape is:

```text
project/
|-- liner.yaml
|-- LINER.md
|-- SKILL.md
|-- working/
|   |-- audits/
|-- children/
|-- lineage.yaml
`-- mixtape/
    |-- tape.yaml
    |-- synthesis.md
    |-- MIXTAPE.md
    |-- sources/
    |-- local-sources/
    |-- .liner-runs/
    `-- working/
```

Legacy root-level mixtape folders remain readable. New product design should
prefer the v2 shape and use compatibility paths intentionally.

`liner.yaml` is the root metadata home for durable project-level state. The
default Project Skill should be recorded there instead of inferred from
`skills/*.md` file counts:

```yaml
status:
  milestone: project_complete
  stale: false
  updated: 2026-06-18T00:00:00Z
  corpus:
    state: ready
    evidence: mixtape/MIXTAPE.md
  operating_layer:
    state: ready
    liner: LINER.md
    status: liner.yaml
project_skill:
  status: active
  name: UI Design
  path: SKILL.md
```

Older projects may still contain legacy `declined` or `unresolved` Project
Skill status. The TUI should read those values without breaking, but new v1
creation should record an active default Project Skill.

The TUI should read this Status Snapshot for fast list views and project
headers. Refresh, Audit, import, and repair flows should validate the snapshot
against the evidence files it cites and update `liner.yaml` when the evidence
changes.

Project Milestone values are limited to `started`, `corpus_ready`, and
`project_complete`. `stale` is stored as a separate modifier on the Status
Snapshot, not as a milestone.

A Status Snapshot is stale when any cited evidence file is newer than
`status.updated`. The UI can still show the saved milestone, but it should mark
the snapshot stale and offer `Refresh status` before treating the milestone as
current.

`Refresh status` may update `liner.yaml`, but only the Status Snapshot fields.
It must not edit corpus files, `LINER.md`, Project Skill files, Audit reports,
sources, or working notes.

When Corpus Creation successfully compiles `MIXTAPE.md`, Liner should
automatically update the Status Snapshot to `corpus_ready`. This is a narrow
status write and does not require a separate confirmation.

`Create Operating Layer` should be visible before `corpus_ready`, but disabled
with a specific reason. It can only run after the Status Snapshot says
`corpus_ready`, or after Refresh Status verifies that the corpus evidence is
ready.

`Next` is the user-facing label for the recommended next milestone action. It
should point toward project completion, not merely mirror the Enter-key action
or selected-row behavior.

The stable nouns are:

- **Liner project:** the root folder and management object.
- **Mixtape:** the corpus artifact inside `mixtape/`.
- **`MIXTAPE.md`:** the compiled context packet an AI can read.
- **`LINER.md`:** project instructions for using the corpus.
- **Project Skill:** the default local skill grounded in the corpus and
  referenced by `LINER.md`.
- **Audit:** a maintenance inspection pass that can later check contradictions,
  source-note quality, stale or weak source grounding, `LINER.md`, and Project
  Skill fit before proposing fixes. It is not part of the default v1 Operating
  Layer creation step.
- **External Use Evidence:** V2 evidence produced when an external agent uses a
  Liner Project and records what happened.
- **Composition:** child projects, lineage, routing, copy packets, and merge
  decisions.

## Process Model

The default creation flow is intentionally small:

1. **Project Shell**: name the Liner Project, define the job to be done, name
   the curator, open the Source Inbox when the user already has material, and
   clarify the job.
2. **Corpus Creation**: discover, fetch, evaluate, and trim sources; write
   curator notes; run corpus quality checks; write or review synthesis; compile
   `MIXTAPE.md`.
3. **Operating Layer**: run one guided command that explains the benefit, then
   writes `LINER.md`, root `SKILL.md`, and `liner.yaml`. This step
   has no separate Project Skill choice and no review screen.

The primary milestones are:

- **Started:** the Project Shell exists.
- **Corpus Ready:** `MIXTAPE.md` exists and the Mixtape can be used as
  source-grounded AI context.
- **Project Complete:** the Corpus is ready and the Operating Layer has written
  `LINER.md`, root `SKILL.md`, and `liner.yaml`. The complete state
  should say the project is complete and expose management actions; it should
  not treat Enter as "open `LINER.md`" by default.

Audits are maintenance work outside the default v1 completion flow. A future
Audit surface can inspect contradictions, source-note quality, stale or weak
source grounding, `LINER.md`, and Project Skill fit, then propose fixes for
review. It should not be reintroduced into the Operating Layer creation step.

Internal Impact Tests are out of scope for v1. V2 should approach proof from the
external-agent side: when another agent uses a Liner Project, it can record what
it loaded, what task it attempted, what the project changed, what source or
instruction gaps it found, and whether the project needs maintenance. That
External Use Evidence can later inform audits, maintenance, and quality
assessment, but it is not part of ordinary Project Complete.

V1 should not expose Impact Tests in navigation, Capabilities, `Next`, or the
default project-management flow. Any existing internal Impact Test
implementation should be treated as parked V2 prototype work until External Use
Evidence is designed from the agent-usage side.

Composition remains a separate advanced conversation. The current hypothesis is
that combining projects should create a parent or container project with its own
`LINER.md` and Project Skill, instead of embedding one project directly inside
another.

The TUI does not need to force every step into the default path. It does need to
make the current artifact state, available capabilities, recommended next work,
and write behavior legible.

## Product Decisions Still Open

These should be tested in the grilling session before the flow is declared
settled:

- Which specific Audit checks should belong to a later maintenance surface?
- What is the composition model: merge, group, parent container, or another
  noun?
