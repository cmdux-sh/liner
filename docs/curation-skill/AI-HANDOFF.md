# Liner AI Handoff

Last updated: 2026-11-07

This is the short orientation for an AI agent working on Liner's curation docs
and methodology without the full chat history. For recent decisions, read
[docs/decisions.md](../decisions.md) first. Verify the checkout with `pwd` and `git status --short` before editing.

## How Liner works now

- The user's coding agent builds a liner through `liner research` steps:
  clarify, framing, candidates, evaluation, quality, synthesis, approve
  synthesis, assembly, approve sources, build, approve instructions, and
  finish. The steps are defined in `src/liner/research.py` (`_STEPS`).
- The agent often runs each step in a subagent. The person answers Clarify Job
  and gives three approvals in chat: the synthesis, the sources, and the
  instructions. The agent never approves for them.
- `liner research submit` checks each step's files. The build step compiles
  the corpus (Corpus Ready). The finish step writes `LINER.md`, root
  `SKILL.md`, and `liner.yaml` (Project Complete).
- The terminal app is a read-only navigator. It never changes a liner.
- Liner Core is the sole writer for supported post-creation Project and Source
  maintenance. Don't teach direct edits of `liner.yaml`, `tape.yaml`,
  `LINER.md`, or root `SKILL.md` as a maintenance workflow.
- 1.2.1 is the current release. Leave version
  files to `scripts/release-version.py`.

## Where the methodology lives

- `src/liner/bundled/research/methodology/`: the `curating-mixtapes` bundle
  (`SKILL.md` and five companion files) that agents read during research.
- `src/liner/bundled/research/clarify.md` and `phases/*.md`: the prompt for
  each step.
- `src/liner/bundled/liner/SKILL.md`: the `liner` skill that finds, suggests,
  and builds liners.
- `docs/curation-skill/`: background for people. CURATION.md is the methodology
  for human readers; keep it and the bundle in step when the methodology
  changes.
- `docs/decisions.md`: why things work the way they do. Newer entries win.

## Verification

```sh
uv run --extra dev pytest
uv run python scripts/validate-public-commands.py
uv run python scripts/validate-public-surfaces.py
python3 scripts/release-version.py --check
git diff --check
```

## Gotchas

- The bundle under `src/liner/bundled/research/` ships to users, so stale
  methodology text there becomes user-facing.
- `docs/curation-skill/ABOUT.md` must keep the exact phrase
  `current source targets v<version>, with macOS`, which
  `scripts/release-version.py` rewrites.
- Do not reintroduce "advisor" language for the operating layer. Use artifact
  names and concrete roles.
- Do not treat URL existence as source quality. Fetched content quality is the
  real gate.
- Before reporting state, separate your edits from other changes in the
  worktree.
