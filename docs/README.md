# Liner Docs

This folder holds the active project documentation. Historical research dumps,
old runbooks, local mixtape work folders, and release artifacts are intentionally
excluded from the public repository.

## Start Here

- [Product](project/PRODUCT.md)
- [Context Glossary](project/CONTEXT.md)
- [Design System](project/DESIGN.md)
- [Liner Master Overview](curation-skill/LINER-MASTER.md)
- [AI Maintainer Orientation](curation-skill/AI-HANDOFF.md)
- [Terminal Tooling Design Framework](tui/TERMINAL_TOOLING_DESIGN_FRAMEWORK.md): historical design notes for the old terminal app
- [Terminal Navigator Plan](project/TUI-NAVIGATOR-PLAN.md)
- [Curation Methodology](curation-skill/CURATION.md)
- [Mixtape Format](curation-skill/MIXTAPE-FORMAT.md)

These are the current source-of-truth docs for the 1.2 source tree. Version
`1.2.1` is the current release. Your coding
agent builds a liner through `liner research` steps, and you approve the
synthesis, the sources, and the instructions in chat. After the last approval,
`liner research` writes `LINER.md`, root `SKILL.md`, and `liner.yaml`. The
terminal app is a read-only navigator for your library.

Supported post-creation Project and Source maintenance goes through Liner Core's
versioned Snapshot, Change Set, apply, and Change Receipt contract. The curation
skill may author initial corpus artifacts, but neither it nor an optional
Maintenance Adapter is a second canonical-file writer. The public command
reference and task guide live under `marketing/site/src/content/docs/docs/`.

## Folders

- `curation-skill/`: background on the methodology. The files agents read while
  researching live in `src/liner/bundled/research/methodology/`.
- `examples/`: small committed example projects and fixtures.
- `project/`: product language, design system, and changelog.
- `tui/`: historical design notes for the old terminal app that built liners. The
  current navigator is read-only.

## Archived History

Older migration notes, raw CLI/TUI research, local mixtape work folders, and old
release tarballs are retained outside the public repository as historical
evidence only. Do not use them as the source of truth for current behavior.
