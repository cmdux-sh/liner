# Liner npm package

This package is named `linersh` because it owns the published `liner` binary.

- `bin/liner.js`: the launcher. It opens the navigator, or forwards a subcommand
  to Core.
- Optional per-platform packages hold the native CLI and navigator binaries.

Every Core command has to be listed in `bin/liner.js`, or the launcher opens the
navigator instead. A test in `tests/test_library_view.py` checks this.

## Install

```sh
npx linersh
npm install -g linersh
liner
```

With no arguments, `liner` opens the navigator. CLI subcommands such as
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

Optional Codex
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
corpus with your agent,
then run Create Operating Layer. Because `HOME`, the npm cache, and `LINER_DIR`
all point inside `$tmp`, cleanup is just `rm -rf "$tmp"`.

## Run Locally

```sh
npm --prefix packages/tui run dev:isolated
```

## Build And Test

```sh
npm run build
npm run build:go
npm run build:package
npm test
```

Notes:

- `build` compiles the version helper.
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
- `LINER_GO_TUI_BIN=/path/to/liner-tui`: force the navigator binary.
- `LINER_TUI=go`: accepted for compatibility, but no longer required.

## Developer Gotchas

- Rebuild `bin/liner-tui` after Go changes or the npm shim may launch an older
  binary.
- The published main package intentionally excludes `bin/liner-tui`; installed
  users get the native TUI from `linersh-<platform>-<arch>`.
