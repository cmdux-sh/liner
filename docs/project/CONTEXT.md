# Liner

Liner is the product context for local, source-grounded AI context projects. The
language in this glossary is the canonical product language for CLI, TUI,
documentation, and agent-facing planning.

## Language

**Liner Project**:
The durable working folder and primary management object in Liner. It contains
the corpus artifact plus project instructions, a default Project Skill, status,
and composition records.
_Avoid_: Mixtape project, archive, advisor, knowledge bot

**Mixtape**:
The curated corpus artifact inside a Liner Project. It is the source-backed
context bundle produced from a recipe, synthesis, source notes, and fetched or
captured source material.
_Avoid_: Liner Project, archive, bot

**`MIXTAPE.md`**:
The compiled context packet an AI can read. It represents corpus readiness, not
full Liner Project readiness.
_Avoid_: Project, complete project, final product

**`LINER.md`**:
The project instruction artifact that tells an AI how to use the Mixtape. It
turns corpus knowledge into operating guidance.
_Avoid_: Advisor prompt, persona, chatbot instructions

**`liner.yaml`**:
The root project metadata and layout marker for a Liner Project. It records
durable project-level state such as milestone, Operating Layer, and Project
Skill status.
_Avoid_: Tape recipe, runtime cache, hidden status

**Status Snapshot**:
The fast project state recorded in `liner.yaml` for list views and project
headers. It stores the current milestone and evidence pointers, and can be
refreshed by checking the files and audits it cites.
_Avoid_: Inferred-only status, stale badge, unchecked truth

**Project Milestone**:
The lifecycle value stored in a Status Snapshot: `started`, `corpus_ready`, or
`project_complete`. Stale is a confidence modifier on the snapshot, not a
milestone.
_Avoid_: Completion percent, stale milestone, done flag

**Stale Status Snapshot**:
A Status Snapshot whose cited evidence has changed after `status.updated`. The
saved milestone can still be shown, but it needs refresh before being treated as
current.
_Avoid_: Broken project, failed status, hidden mismatch

**Refresh Status**:
The scan that validates cited evidence and updates only the Status Snapshot in
`liner.yaml`. It never edits corpus files, `LINER.md`, root `SKILL.md`, or
Audit reports.
_Avoid_: Audit, repair, cleanup, project rewrite

**Next**:
The recommended next action for the current Liner Project state. It is not the
same thing as a selected-row action or a generic primary button. When the footer
pairs `Next` with Enter, Enter must perform that action.
_Avoid_: Milestone phrasing, shortcut hint, selected action, generic primary action

**Source**:
Any material Liner can use as evidence for a Mixtape. Sources can be web,
private, local, captured, skill-based, or agent-discovered, but each one needs a
clear role in the corpus.
_Avoid_: Resource, link dump, content item

**Source Inbox**:
The user-provided source entry surface in a Liner Project. It accepts known
material the user already has, then stages it for review before it becomes part
of the corpus.
_Avoid_: Custom sources, manual sources, human sources, personal sources, local sources

**User-Provided Source**:
A source supplied by the user through the Source Inbox before or during the
curation flow. This describes origin, not source type.
_Avoid_: Custom source, manual source, human source

**Curator**:
The accountable human or named owner whose judgment shapes a Mixtape. The
curator is responsible for source choices, curator notes, synthesis stance, and
final acceptance even when AI drafts or assists the work.
_Avoid_: AI curator, bot, operator, reviewer

**Project Skill**:
The root `SKILL.md` attached to a Liner Project. It gives future AI sessions a
direct way to load the project's `LINER.md`, `MIXTAPE.md`, and source set.
_Avoid_: Skill pack, optional decision, ungrounded skill

**Operating Layer**:
The guided pass that turns a Corpus Ready project into a Project Complete one.
It writes `LINER.md`, root `SKILL.md`, and `liner.yaml`. It does not
ask for a separate Project Skill choice or show a review screen.
_Avoid_: Skill layer, advisor layer, persona layer

**Audit**:
The maintenance inspection pass that can check a Liner Project for
contradictions, stale or weak source grounding, source-note issues, and Project
Skill fit. It is outside the default v1 Operating Layer creation step.
_Avoid_: Audit phase, checklist picker, automatic cleanup

**External Use Evidence**:
V2 evidence produced when an external agent uses a Liner Project and records
what happened. This is outside the v1 project-completion flow.
_Avoid_: Internal impact test, required completion step, benchmark theater

**Audit Output**:
The unified user-facing output of a future Audit surface. It groups Blocking
Findings, Advisory Findings, and Proposed Fixes instead of exposing separate
audit types.
_Avoid_: Operating Layer creation review, audit type picker, separate reports

**Blocking Finding**:
An Audit finding that must be resolved before a Liner Project can be considered
Project Complete because it would make `LINER.md` or the Project Skill
misleading, unsupported, stale, or contradictory.
_Avoid_: Advisory note, preference, polish issue

**Proposed Fix**:
A reviewable change suggested by an Audit Output. It is a dry-run proposal
until the user chooses to apply it.
_Avoid_: Automatic cleanup, silent repair, direct mutation

**Blocking Fix**:
A Proposed Fix that resolves a Blocking Finding. Blocking Fixes are the only
fixes required before a Liner Project can become Project Complete.
_Avoid_: Advisory fix, optional cleanup, polish pass

**Cleared Finding**:
A Blocking Finding the user has explicitly accepted or dismissed with a recorded
reason when no safe automatic fix exists.
_Avoid_: Ignored blocker, hidden override, silent pass

**Advisory Finding**:
An Audit finding that should remain recorded but does not block Project
Complete. It captures maintenance, quality, or improvement work that can be
handled later.
_Avoid_: Blocker, failure, required fix

**Job to Be Done**:
The scoped use case a Mixtape is built to help with. It defines the
circumstance, motivation, and outcome that source discovery and evaluation must
serve.
_Avoid_: Topic, goal, vibe, prompt, user-facing JTBD

**Job Story**:
The required sentence shape for a Job to Be Done: "When [circumstance], I want
[motivation], so I can [outcome]."
_Avoid_: JTBD format, task prompt, objective statement

**Clarify Job**:
The question flow that sharpens a Job to Be Done before agent-assisted research
continues. It records clarifications; it is not a separate artifact.
_Avoid_: Clarification wizard, research interview, extra setup
