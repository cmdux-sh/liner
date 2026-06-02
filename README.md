# Liner

Liner is a keyboard-driven terminal app for building **mixtapes**: portable context bundles for AI work.

A mixtape gives an AI system a focused set of sources, source notes, and a synthesis for a specific job-to-be-done. Instead of pasting the same links and notes into every conversation, you build a project folder once, compile it, and reuse the result anywhere Markdown files can be used.

Learn more at [liner.sh](https://liner.sh).

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

The npm package includes the terminal app and a bundled local core for macOS arm64/x64, Linux arm64/x64, and Windows x64.

## How It Works

Liner creates project folders inside a `mixtapes/` workspace. Each project contains:

- `tape.yaml`: title, purpose, curator, and source list
- `synthesis.md`: the framing the AI should read first
- `working/`: notes from the research and review process
- `sources/`: extracted source files written during compile
- `MIXTAPE.md`: the compiled entry point for AI conversations

The usual flow is:

1. Create a mixtape.
2. Define the job-to-be-done.
3. Add web, YouTube, or local file sources.
4. Write or review the synthesis.
5. Compile the project.
6. Paste `MIXTAPE.md` into an AI conversation and load source files when useful.

Start from the directory where you want the workspace:

```sh
mkdir liner-workspace
cd liner-workspace
npx linersh
```

For keybindings and workflow details, see [docs/tui.md](docs/tui.md).

## Sources

Liner supports three source types:

- `web`: articles, docs, papers, and other web pages
- `youtube`: video transcripts
- `local_file`: files you place under a project's `personal/` folder

Source notes are important. They tell the AI when a source matters, what part to pay attention to, and what not to overgeneralize from it.

For JavaScript-heavy web pages, run:

```sh
liner setup-js
```

That installs Playwright's headless Chromium browser, about 150 MB. Most sources do not need it.

## Agent Assistance

If Claude Code or Codex is installed, Liner can use it to draft and revise research artifacts during the workflow. Agent runs use your local account and write ordinary files into the project folder.

Compiling, sharing, importing, and cache operations run locally. Liner does not require an account, does not call an LLM during compile, and does not send telemetry.

## Commands

Running `liner` with no arguments opens the terminal app.

| Command | Use |
| --- | --- |
| `liner setup-js` | Install browser support for JavaScript-rendered web sources. |
| `liner compile <folder>` | Fetch sources and write `MIXTAPE.md` plus `sources/`. |
| `liner share <folder>` | Pack a project folder into a `.mixtape` archive. |
| `liner import <archive> [dest]` | Unpack a `.mixtape` archive and refetch uncached sources. |
| `liner list` | List mixtape project folders. |

Advanced flags and developer commands are available through `liner --help`.

## Format

See [docs/mixtape-format.md](docs/mixtape-format.md) for the tape format and project-folder reference.

## Development

Python core:

```sh
uv run --extra dev pytest
```

Terminal app:

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
npm uninstall -g linersh   # only if installed globally
```

`liner uninstall` removes Liner's local cache/config, Playwright's Chromium cache, and npm's `_npx` execution cache. It does not delete mixtape project folders.

## About

Liner is an active solo project. The code is MIT-licensed and open to forks. Issues and pull requests are welcome, but I cannot promise a review timeline or merge work that pulls the project away from the direction I am taking it.

## License

MIT. See [LICENSE](LICENSE).
