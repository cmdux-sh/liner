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

The old React/Ink TUI has been decommissioned and is no longer part of the
active package shape. `LINER_TUI=ink` intentionally fails with a decommissioning
message.

## Install

```sh
npx linersh
npm install -g linersh
liner
```

With no arguments, `liner` launches the Go TUI. CLI subcommands such as
`liner compile`, `liner share`, `liner import`, `liner project`,
`liner sources`, `liner adapters`, `liner status`, and `liner setup-js` pass
through to the Python core. If you are using one-shot
`npx` instead of a global install, pass subcommands through the package:
`npx --yes linersh@latest setup-js` or
`npx --yes linersh@latest uninstall --yes`.

## Safe Project Maintenance

The installed CLI is the authority for supported Project and Source changes.
Inspection and planning are write-free; apply revalidates one exact versioned
Change Set and writes a durable, redacted Change Receipt.

```sh
liner project guidance <project> --format markdown
liner project inspect <project> --json
liner sources add <project> --type web --url <url> --note <note>
liner project rename <project> --name <name>
```

The Go TUI's `Maintain project` command uses the same contract. Optional Codex
and Claude Maintenance Adapters are explicit CLI-delegating installations:

```sh
liner adapters inspect codex
liner adapters install codex --yes
```

The Maintenance Adapter is not the Project Skill, and a `type: skill` Source is
inert reference material rather than installed instructions.

## Try It Cleanly

For a regular-user smoke test without touching your real Liner workspace:

```sh
tmp=$(mktemp -d)
home="$tmp/home"
cache="$tmp/npm-cache"
projects="$tmp/projects"
mkdir -p "$home" "$cache" "$projects"

HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes linersh@latest --version
HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes linersh@latest

rm -rf "$tmp"
```

The second command opens the Go TUI. Create a test Liner Project, build the
corpus with your configured OpenAI (Codex CLI) or Claude (Claude Code) runner,
then run Create Operating Layer. Because `HOME`, the npm cache, and `LINER_DIR`
all point inside `$tmp`, cleanup is just `rm -rf "$tmp"`.

## Run Locally

For normal repository testing, use the isolated launch so Projects and Settings
cannot write to the production Liner library or profile:

```sh
npm --prefix packages/tui run dev:isolated
```

See [../../docs/development.md](../../docs/development.md) for explicit
authenticated provider-home options and the offline runner-preferences smoke.

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
- `npm run release:check` is offline and verifies repository consistency.
  `npm run release:check:publish` additionally refuses to publish the main
  package when its canonical version already exists on npm.

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
