# Liner

Liner is a terminal app for building **mixtapes**: portable context bundles for AI work.

The main experience is the **TUI**. It helps you create a mixtape project, sharpen the job-to-be-done, add sources, review the authoring phases, compile the sources, and share the result. The CLI still exists, but it is an optional path for scripting or for people who prefer editing YAML and Markdown by hand.

A **tape** is the recipe: `tape.yaml`. A **mixtape** is the project folder: `tape.yaml`, `synthesis.md`, `working/`, compiled `sources/`, and `MIXTAPE.md`. The compiled `MIXTAPE.md` is the entry point you paste into an AI conversation, with source files loaded when the conversation needs detail.

## Install

Requires Node 18+.

```sh
npx linersh
```

For a persistent install:

```sh
npm install -g linersh
liner
```

The npm package ships the TUI plus the matching bundled core binary for your platform. Current platform support is macOS arm64/x64, Linux arm64/x64, and Windows x64.

## Start

Run Liner from the folder where you want your mixtape workspace to live:

```sh
mkdir liner-workspace
cd liner-workspace
npx linersh
```

The TUI creates and manages a `mixtapes/` folder in the current directory. Each subfolder is one mixtape project.

Inside the TUI you can:

- create a new mixtape with the guided wizard
- add web, YouTube, and local file sources
- edit the tape recipe without writing YAML by hand
- run optional agent-assisted authoring phases with Claude Code or Codex when installed
- compile the mixtape into `MIXTAPE.md` and `sources/`
- share or import `.mixtape` archives

For keybindings and the detailed TUI flow, see [docs/tui.md](docs/tui.md).

## JavaScript-Rendered Pages

Most web sources work without extra setup. For pages that require a browser, such as JavaScript-heavy docs, enable Playwright support:

```sh
liner setup-js
```

That downloads headless Chromium, about 150 MB. The setup is optional and safe to rerun. The TUI also offers this during onboarding and when compile warnings show that JS rendering is needed.

## Agent Assistance

Liner can drive some authoring phases with Claude Code or Codex if one is installed and available on your `PATH`. Those runs use your own local agent account and write ordinary project files into the mixtape folder.

The core compile/share/import path is local and deterministic. It does not call an LLM, does not require an account, and does not send telemetry.

## Optional CLI

The CLI is there when you want a direct command or automation hook:

| Command | Use |
| --- | --- |
| `liner` | Open the TUI. |
| `liner setup-js` | Install browser support for `render: js` web sources. |
| `liner compile <folder>` | Fetch sources and write `MIXTAPE.md` plus `sources/`. |
| `liner share <folder>` | Pack a project folder into a `.mixtape` archive. |
| `liner import <archive> [dest]` | Unpack a `.mixtape` archive and refetch uncached sources. |
| `liner list` | List mixtape project folders. |

Advanced CLI flags and developer commands are available through `liner --help`.

## Tape Format

The short version:

- `tape.yaml` needs `title`, `description`, `version: 1`, `curator`, and `sources`.
- Source types are `web`, `youtube`, and `local_file`.
- Source notes matter because they tell the consuming AI when and how to use each source.
- `synthesis.md` is required before a mixtape is useful; it is copied into the top of `MIXTAPE.md`.

See [docs/mixtape-format.md](docs/mixtape-format.md) for the full reference.

## Examples

The example folder is [mixtape-examples/](mixtape-examples/). The old early sample was removed because it no longer represented the quality bar. Better examples will live there when they are ready.

## Development

Python core:

```sh
uv run --extra dev pytest
```

TUI:

```sh
cd packages/tui
npm ci
npm run typecheck
npm test
npm run build
```

## Uninstall

```sh
liner uninstall --yes
npm uninstall -g linersh   # only if you installed it globally
```

`liner uninstall` removes Liner's local cache/config, Playwright's Chromium cache, and npm's `_npx` execution cache. It does not delete your mixtape project folders.

## About

Liner is an active solo project. The code is MIT-licensed and open to forks. Issues and pull requests are welcome, but I cannot promise a review timeline or merge work that pulls the project away from the direction I am taking it.

## License

MIT. See [LICENSE](LICENSE).
