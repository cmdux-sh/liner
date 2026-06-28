# Liner Product Unknowns - 2026-06-17

This is a historical decision log from the initial Go TUI cleanup. Current v1
behavior is defined in [PRODUCT.md](PRODUCT.md) and
[CONTEXT.md](CONTEXT.md). Any remaining unknowns here are subordinate to
those files.

## Completion Model

Current direction: Project Complete means the Operating Layer has written
`LINER.md`, root `SKILL.md`, and `liner.yaml`.

- `MIXTAPE.md` means Corpus Ready.
- `LINER.md` means the final Operating Layer exists.
- V1 milestones are `started`, `corpus_ready`, and `project_complete`.

## Post-Mixtape Flow

Resolved direction: after `MIXTAPE.md` exists, the intended path is Create
Operating Layer.

1. Explain that `MIXTAPE.md` is the context packet and `LINER.md` tells AI
   sessions how to use it.
2. Keep the screen concise and let Enter create `LINER.md`, root `SKILL.md`,
   and `liner.yaml`.
3. Write those files and mark `project_complete`.

## Next Semantics

Resolved direction: `Next` describes the recommended milestone action, not the
Enter key, selected row, or a generic missing capability.

- Started projects point toward sources/methodology/compile.
- Corpus Ready projects point toward Create Operating Layer.
- Project Complete projects show complete-project management actions.

- Decide whether a future surface should show a separate recommended capability
  action that is not bound to Enter.
- Decide where to show a recommended next capability when the selected section
  has a different local action.

## Evidence And Conflicts

Question: what does it mean to check evidence or conflicting evidence?

Known pieces for future maintenance surfaces:

- Contradiction audits can exist under `working/audits/`.
- Source-note quality audits can inspect saved source notes.
- Skill-corpus alignment audits can inspect skill grounding and boundaries.

Unknowns:

- Which evidence checks belong in maintenance rather than creation?
- Should conflicting evidence block future maintenance approval?
- Which specific checks should be blocking versus advisory in a later Audit
  surface?

## Project Skill

Resolved direction: Project Skill is singular and created by default during
Create Operating Layer.

- The default Project Skill is stored as root `SKILL.md` in the project folder.
- The Project Skill is represented in `liner.yaml` and referenced by
  `LINER.md`.
- Legacy declined or unresolved Project Skill statuses remain readable but are
  not created by the v1 flow.

## LINER.md

Resolved direction:

- `MIXTAPE.md` is the context artifact.
- `LINER.md` is the final Operating Layer for how AI sessions should use
  that context.
- `LINER.md` is required for Project Complete.

## Composition

Composition is tabled for a later phase. Do not implement merge, group,
container, child routing, or composition readiness in v1.

## UI Implications

Near-term UI rules:

- Avoid saying a compiled project is simply complete.
- Keep Project surfaces aligned to Started -> Corpus Ready -> Create Operating
  Layer -> Project Complete.
- Keep Health factual and evidence-based.
- Keep the left Project section list as navigation only; details belong on the
  right.
- Track unknowns here instead of hard-coding premature product promises into the
  TUI.
