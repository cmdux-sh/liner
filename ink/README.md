# Ink TUI Archive

This folder contains the decommissioned React/Ink terminal UI.

Liner no longer launches this implementation. The active terminal UI is the Go
Charm app in `packages/go-tui/`. The npm package shell remains in
`packages/tui/` because it still provides:

- the `liner` npm binary shim
- the Go TUI launcher
- the TypeScript headless methodology runner used by the Go TUI
- packaged curation skill files copied into `dist/`

Do not add new product work here. Keep this folder only as historical reference
while the Go TUI becomes the active implementation.

If old docs mention `LINER_TUI=ink`, treat that as historical. `liner` now
launches the Go TUI by default; `LINER_TUI=ink` intentionally fails with a
decommissioning message.
