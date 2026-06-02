# Liner TUI

Interactive authoring UI for [Liner](../..) — create, edit, compile, and share mixtape **project folders** from the keyboard.

A *tape file* (`tape.yaml`) is the recipe. A *mixtape* is the project folder Liner builds from it. The TUI is for authoring: it scaffolds folders, edits tapes, surfaces the synthesis status, then runs `liner compile` to produce the consumable `MIXTAPE.md` + `sources/` artifacts.

## Install

```sh
# One-shot:
npx linersh

# Persistent:
npm install -g linersh    # adds `liner` to PATH
liner
```

The npm package includes the TUI and resolves the Python core from a per-platform optional package (`linersh-<platform>-<arch>`). With no arguments, `liner` opens the TUI. CLI subcommands such as `liner setup-js`, `liner compile`, `liner share`, and `liner status` pass through to the bundled core.

For JavaScript-rendered pages, run the setup command from the same npm install:

```sh
liner setup-js
```

That command is idempotent and downloads the Playwright browser used for `render: js` and automatic JS-wall fallback (about 150MB). The TUI also offers this during first-run onboarding, and again from compile results if a source needs JS rendering. For debugging or local development, `LINER_BIN=/path/to/liner liner` overrides the bundled binary.

To reset a machine for clean testing, run:

```sh
liner uninstall --yes
```

This removes `~/.liner`, Playwright's Chromium cache, and npm's `_npx` execution cache. It does not delete your mixtape project folders. If you installed Liner globally, also run `npm uninstall -g linersh`.

## Run

```sh
cd ~/projects/my-thing
liner
```

The TUI looks for (and creates) a `./mixtapes/` folder in your current directory. Each subdirectory inside it is a mixtape project folder (must contain a `tape.yaml`).

Override the workspace with `LINER_DIR`:

```sh
LINER_DIR=~/mixtapes liner
```

## Keys

### Browser
| Key | Action |
|---|---|
| `↑` `↓` or `j` `k` | move selection |
| `enter` or `o` | open the selected mixtape hub |
| `e` | open the selected mixtape source editor |
| `n` | new mixtape (prompts for slug) |
| `c` | compile selected |
| `s` | share selected (zip to `.mixtape`) |
| `i` | import a `.mixtape` archive |
| `d` | delete selected folder (with confirm) |
| `r` | refresh |
| `q` | quit |

### Project Hub
| Key | Action |
|---|---|
| `↑` `↓` | select phase |
| `enter` | run/open the selected phase's primary action |
| `e` | edit the selected artifact by hand |
| `p` | open the process manifest |
| `s` | share the mixtape |
| `o` | open the project folder |
| `esc` | back |
| `q` | quit |

Completed mixtapes land in a ready state. If `MIXTAPE.md` is current, pressing `enter` on Compile opens the existing compile result instead of fetching sources again. Press `r` from the compile result screen to explicitly re-run/retry.

### Source Editor
| Key | Action |
|---|---|
| `t` `d` `u` | edit title / description / curator |
| `m` | toggle mode (`quick` ⇄ `methodology`) |
| `g` | edit JTBD (job-to-be-done) |
| `E` | open `synthesis.md` in `$EDITOR` |
| `↑` `↓` | select source |
| `a` `e` `x` | add / edit / remove source |
| `K` `J` | reorder selected source up/down |
| `s` | save `tape.yaml` |
| `c` | save + compile (blocked if synthesis is empty or still a placeholder) |
| `esc` | back (warns on unsaved changes) |

A soft warning appears when any source is missing a curator note. Notes are load-bearing — see [CURATION.md](https://liner.sh) §6.

#### Local sources

`local_file` sources reference content you've placed under `<project>/personal/` — book chapter PDFs, Reader-Mode-saved articles, exported HIG pages. Use them for material that isn't on the public web or that lives behind a paywall you have legitimate access to.

Workflow:

1. Save the file into `<project>/personal/`. The TUI creates the folder lazily — the file picker offers a "new file (paste path)" option that you can use after saving the file manually with your OS file manager.
2. Add a source in the editor; press `T` in the source modal to switch the type to `local_file`.
3. Pick the file via the picker; supply a `citation` (author, title, publication, date — the AI uses this to weight the source).
4. Compile. The CLI extracts the content (PDFs via `pdfplumber`, HTML via the same readability pipeline as web sources, `.md`/`.txt` as-is).

`liner share` includes `personal/` by default. Pass `--no-personal` (or use the corresponding TUI flag) to exclude it — required for any tape intended as a library submission.

The synthesis indicator shows one of:
- `✗ missing` — `liner compile` will hard-fail
- `⚠ placeholder` — file exists but still has the boilerplate from `liner init`
- `⚠ empty` — file exists but is empty
- `✓ N chars` — ready to compile

### Source modal
| Key | Action |
|---|---|
| `↑` `↓` | move between fields |
| `enter` | edit the focused field (or open the file picker on the path field for `local_file` sources) |
| `tab` | next field while editing |
| `T` | toggle source type (`web` ⇄ `local_file`) |
| `R` | toggle `render: server` ⇄ `render: js` (web sources only) |
| `p` | toggle required/optional |
| `s` | save source |
| `esc` | cancel |

For `web` sources the URL is the load-bearing field; the type is auto-detected as `youtube` or `web`. For `local_file` sources the modal shows `path`, `citation`, `note`, `section`, `priority` — pick the path via the file picker (lists files under `personal/`, with a "paste path" affordance for new files).

### Compile
| Key | Action |
|---|---|
| `↑` `↓` | scroll sources |
| `pgup` `pgdn` | jump through sources |
| `j` | install JS rendering support when the result says it is needed |
| `esc` | cancel running compile |
| `y` | copy compiled `MIXTAPE.md` to clipboard (`pbcopy` / `xclip`) |
| `s` | share the compiled mixtape (zip to `.mixtape`) |
| `o` | open the project folder |
| `r` | retry failed sources / re-run compile |
| `b` | back to hub |
| `q` | quit |

The compile writes `MIXTAPE.md` and `sources/NN-<slug>.md` files directly into the project folder. A partial compile is still treated as a usable finished state when `MIXTAPE.md` is written; the hub shows "compiled with warnings" and the compile result screen lists the remaining warnings. There's nothing to "save" afterwards — the artifacts are already on disk.

## Uninstalling

The TUI ships in the same `linersh` package as the CLI. For npm/npx installs, use the package uninstaller:

```sh
liner uninstall
liner uninstall --yes
```

It removes `~/.liner`, Playwright's Chromium cache, and npm's `_npx` execution cache. Your mixtape project folders are left alone. If you installed Liner globally, also run `npm uninstall -g linersh`. See the root [README — Uninstalling](../../README.md#uninstalling) for the manual recipe.

## Develop

```sh
cd packages/tui
npm install
npm run dev          # tsx (no watch — Ctrl+C and re-run after code changes)
npm run dev:built    # builds, then runs the compiled output (closer to ship)
npm run typecheck
npm test             # vitest
npm run build        # emits dist/
```

**Why no `tsx watch`?** `tsx watch` keeps the TTY in raw mode for its own restart UI (Enter to restart, "rs" to force, Esc to clear). Ink TUIs also want raw stdin, and the two collide — typed characters get eaten, the app gets surprise-restarted, screens flicker. For interactive UIs, restart manually after code changes.

In dev, the bin-resolver tries `LINER_BIN`, the bundled optional package, the repo-local `.venv/bin/liner`, then `liner` on `$PATH`. To point at a specific binary:

```sh
LINER_BIN=/path/to/liner npm run dev
```

## Architecture

```
BootSplash ─────► TapeBrowser ──────► ProjectHub ──────┐
       │                            │
       │          ImportPrompt ─────┤
       │                            │
       ├────► NewProjectWizard ─────┤
       │                            │
       ├────► TapeEditor ⇄ SourceModal
       │                            │
       ├────► PhaseRunner / gates / draft review
       │                            │
       └────► CompileView (streams NDJSON from `liner compile --emit-events`)
```

State is global via `AppContext`. YAML is parsed and re-written in Node via the `yaml` package's `Document` API so user-written block comments survive edits. All folder operations (`init`, `compile`, `share`, `import`, `list`) go through the Python CLI as subprocesses — the TUI never reimplements them.

The CLI's `compile --emit-events` streams NDJSON progress events to stdout; the TUI parses each line and updates per-source state live. The final `result` event reports the produced folder, `MIXTAPE.md` path, and per-source file paths.

The same NDJSON protocol will power the planned MCP server, which calls into the same Python core directly without going through the CLI.
