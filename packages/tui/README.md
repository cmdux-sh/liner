# Liner TUI

Interactive terminal UI for Liner.

For user-facing installation, keybindings, and workflow details, see [../../docs/tui.md](../../docs/tui.md).

## Develop

```sh
cd packages/tui
npm install
npm run dev
npm run dev:built
npm run typecheck
npm test
npm run build
```

`npm run dev` uses `tsx` without watch mode. Watch mode keeps the terminal in raw mode for its own restart UI, which collides with Ink's keyboard handling. Restart manually after code changes.

In development, the binary resolver tries `LINER_BIN`, the bundled optional package, the repo-local `.venv/bin/liner`, then `liner` on `PATH`.

```sh
LINER_BIN=/path/to/liner npm run dev
```

## Architecture

The TUI is a React/Ink app that shells out to the Python core for filesystem operations:

- `liner list` discovers mixtape projects
- `liner init` scaffolds project folders
- `liner compile --emit-events` streams NDJSON progress
- `liner share` creates `.mixtape` archives
- `liner import` unpacks `.mixtape` archives
- `liner setup-js` installs browser rendering support

Agent-assisted authoring phases run through Claude Code or Codex when available. The skill bundle source lives at `docs/skill/`; `prepack` copies it into `packages/tui/cli-update-docs/` so the published npm package can find it.
