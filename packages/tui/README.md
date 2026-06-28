# Liner NPM Shim And Runner Package

This package is still named `linersh` because it owns the published `liner`
binary. It is no longer the React/Ink TUI implementation.

Active responsibilities:

- `bin/liner.js`: npm binary shim
- Go TUI launcher for `packages/go-tui/` in development
- TypeScript headless methodology runner under `src/agents/`
- packaged curation skill bundle copied from `../../docs/curation-skill/` into
  `cli-update-docs/`
- optional per-platform packages that contain the native CLI and Go TUI
  binaries for installed users

The old React/Ink TUI has been decommissioned and moved to the repo-root
`ink/` archive. `LINER_TUI=ink` intentionally fails with a decommissioning
message.

## Install

```sh
npx linersh
npm install -g linersh
liner
```

With no arguments, `liner` launches the Go TUI. CLI subcommands such as
`liner compile`, `liner share`, `liner import`, `liner status`, and
`liner setup-js` pass through to the Python core.

## Run Locally

From the repo root:

```sh
npm --prefix packages/tui run dev
```

From this package:

```sh
npm run dev
```

Override the projects folder:

```sh
LINER_DIR=/path/to/projects node bin/liner.js
```

Override the Python/core binary:

```sh
LINER_BIN=/absolute/path/to/liner node bin/liner.js
```

## Build And Test

```sh
npm run build
npm run build:go
npm run build:package
npm test
```

Notes:

- `build` compiles the TypeScript headless runner and shared runner utilities.
- `build:go` compiles `../go-tui/cmd/liner-tui` into `bin/liner-tui` for local
  development.
- `build:package` runs both.
- Release packaging is verified with `npm run acceptance:go -- release-smoke`,
  which packs a local platform package plus the main `linersh` package and
  installs both into a clean consumer project.

## Useful Environment Variables

- `LINER_DIR=/path/to/projects`: change the projects folder.
- `LINER_BIN=/path/to/liner`: force the Python/core binary.
- `LINER_GO_TUI_BIN=/path/to/liner-tui`: force the Go TUI binary.
- `LINER_HEADLESS_RUNNER=/path/to/headless-runner.js`: force the methodology
  runner script.
- `LINER_AGENT=codex|claude`: override the agent selected by Liner config.
- `LINER_TUI=go`: accepted for compatibility, but no longer required.

## Developer Gotchas

- Rebuild `bin/liner-tui` after Go changes or the npm shim may launch an older
  binary.
- Rebuild `dist/` after TypeScript runner changes or methodology will fail.
- The published main package intentionally excludes `bin/liner-tui`; installed
  users get the native TUI from `linersh-<platform>-<arch>`.
- If Project Health says `status failed`, check the resolved core. Older bundled
  cores may reject `--no-write`; the Go wrapper has a fallback, but direct
  `node bin/liner.js status ... --no-write` can still fail until platform cores
  are refreshed.
