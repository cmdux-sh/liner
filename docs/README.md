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
- [Terminal Tooling Design Framework](tui/TERMINAL_TOOLING_DESIGN_FRAMEWORK.md)
- [Go TUI Screen Patterns](tui/GO_TUI_SCREEN_PATTERNS.md)
- [Automated TUI Visual Acceptance And Repair](tui/AUTOMATED_VISUAL_ACCEPTANCE.md)
- [Curation Methodology](curation-skill/CURATION.md)
- [Mixtape Format](curation-skill/MIXTAPE-FORMAT.md)

These are the current source-of-truth docs for the 1.1 source tree. Version
`1.1.0` is still a release candidate until publish succeeds. The current
Operating Layer flow writes `LINER.md`, root `SKILL.md`, and `liner.yaml`; it
does not show extra review screens or a separate Project Skill choice.

Supported post-creation Project and Source maintenance goes through Liner Core's
versioned Snapshot, Change Set, apply, and Change Receipt contract. The curation
skill may author initial corpus artifacts, but neither it nor an optional
Maintenance Adapter is a second canonical-file writer. The public command
reference and task guide live under `marketing/site/src/content/docs/docs/`.

## Folders

- `curation-skill/`: tracked source for the methodology bundle copied into the
  npm package during `prepack`; edits here affect the shipped skill bundle.
- `examples/`: small committed example projects and fixtures.
- `project/`: product language, design system, and changelog.
- `tui/`: current Go TUI design references.

## Archived History

Older migration notes, raw CLI/TUI research, local mixtape work folders, and old
release tarballs are retained outside the public repository as historical
evidence only. Do not use them as the source of truth for current behavior.
