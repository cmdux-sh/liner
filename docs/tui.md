# TUI Guide

The TUI is the main Liner experience.

```sh
npx linersh
```

Run it from the folder where you want Liner to manage a `mixtapes/` workspace. Override the workspace with `LINER_DIR`:

```sh
LINER_DIR=~/mixtapes liner
```

## What It Does

The TUI lets you:

- browse mixtape project folders
- create a new mixtape with a guided wizard
- sharpen the job-to-be-done with optional Claude Code or Codex assistance
- edit `tape.yaml` through a structured source editor
- add `web`, `youtube`, and `local_file` sources
- run the authoring phases
- compile into `MIXTAPE.md` and `sources/`
- share and import `.mixtape` archives

Agent-assisted phases are optional. They run through Claude Code or Codex if available on your machine. Compile/share/import are handled by the local Liner core.

## Main Browser

| Key | Action |
| --- | --- |
| `up` / `down` or `j` / `k` | Move selection. |
| `enter` or `o` | Open the selected mixtape hub. |
| `e` | Open the selected source editor. |
| `n` | Create a new mixtape. |
| `c` | Compile selected mixtape. |
| `s` | Share selected mixtape as `.mixtape`. |
| `i` | Import a `.mixtape` archive. |
| `d` | Delete selected folder after confirmation. |
| `r` | Refresh. |
| `q` | Quit. |

## Project Hub

| Key | Action |
| --- | --- |
| `up` / `down` | Select phase. |
| `enter` | Run or open the selected phase's primary action. |
| `e` | Edit the selected artifact by hand. |
| `p` | Open the process manifest. |
| `s` | Share the mixtape. |
| `o` | Open the project folder. |
| `esc` | Back. |
| `q` | Quit. |

When `MIXTAPE.md` is current, the Compile row opens the existing result instead of fetching again. Re-run from the compile result screen with `r`.

## Source Editor

| Key | Action |
| --- | --- |
| `t` / `d` / `u` | Edit title, description, or curator. |
| `m` | Toggle `quick` / `methodology` mode. |
| `g` | Edit JTBD. |
| `E` | Open `synthesis.md` in `$EDITOR`. |
| `up` / `down` | Select source. |
| `a` / `e` / `x` | Add, edit, or remove source. |
| `K` / `J` | Move selected source up or down. |
| `s` | Save `tape.yaml`. |
| `c` | Save and compile. |
| `esc` | Back, with unsaved-change warning. |

The editor warns when sources are missing notes. Notes are used by the compiled source index to tell the consuming AI when each source matters.

## Source Modal

| Key | Action |
| --- | --- |
| `up` / `down` | Move between fields. |
| `enter` | Edit focused field, or open the file picker for a local file path. |
| `tab` | Next field while editing. |
| `T` | Toggle source type between web and local file. |
| `R` | Toggle web render mode. |
| `p` | Toggle required/optional priority. |
| `s` | Save source. |
| `esc` | Cancel. |

For web sources, paste a URL; YouTube URLs are detected automatically. For local files, place the file under `<project>/personal/`, then choose it in the file picker and add a citation.

## Compile Result

| Key | Action |
| --- | --- |
| `up` / `down` | Scroll sources. |
| `pgup` / `pgdn` | Jump through sources. |
| `j` | Install JS rendering support when needed. |
| `esc` | Cancel a running compile. |
| `y` | Copy `MIXTAPE.md` to clipboard. |
| `s` | Share as `.mixtape`. |
| `o` | Open the project folder. |
| `r` | Retry failed sources or re-run compile. |
| `b` | Back to hub. |
| `q` | Quit. |

A partial compile is still usable when `MIXTAPE.md` is written. The result screen lists warnings and any failed sources.

## Local Sources

`local_file` sources are for material you already have access to: PDFs, saved articles, Markdown notes, or exported HTML.

1. Put the file under `<project>/personal/`.
2. Add a source and press `T` until the modal shows `local_file`.
3. Pick the file and enter a citation.
4. Compile.

The visible TUI share action uses the default archive behavior, which includes `personal/`. To exclude local files, use the optional CLI path:

```sh
liner share <folder> --no-personal
```

## Development

```sh
cd packages/tui
npm install
npm run dev
npm run typecheck
npm test
npm run build
```

In development, `LINER_BIN=/path/to/liner npm run dev` points the TUI at a specific core binary.
