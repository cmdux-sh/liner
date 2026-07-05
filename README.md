# Liner

**Liner** is an open-source toolkit for building local, source-grounded context
projects that help an AI agent do one hyper-specific task better.

A *tape file* (`tape.yaml`) is the source recipe. A *Mixtape* is the compiled
corpus artifact Liner builds from that recipe: `synthesis.md`, `sources/`,
`working/`, and `MIXTAPE.md`. A *Liner Project* is the durable local folder
around that corpus. After the corpus is ready, Create Operating Layer writes
`LINER.md`, a root `SKILL.md`, and `liner.yaml` so future AI sessions know what
the resource is for, how to use the evidence, and where its boundaries are. Tape
!= Mixtape != Liner Project.

The creation prompt is intentionally plain-language: **What do you want this
Liner to help your AI agent do?** The answer should be narrow. "SEO" is too
broad; "SEO keyword research for a mental-health startup specialized in brain
surgery" gives Liner enough specificity to derive research lanes, find the right
sources, and produce a resource a future agent can actually use. Internally,
Liner still stores an inferred `jtbd` for compatibility, but the user does not
need to write a formal Job Story.

This repo contains the **CLI**, the **Go TUI**, the **npm package shell**, and
the project docs. The methodology lives in
[docs/curation-skill/CURATION.md](docs/curation-skill/CURATION.md), the
platform doc in
[docs/curation-skill/PLATFORM.md](docs/curation-skill/PLATFORM.md), the format
spec in
[docs/curation-skill/MIXTAPE-FORMAT.md](docs/curation-skill/MIXTAPE-FORMAT.md),
the curation skill in [docs/curation-skill/SKILL.md](docs/curation-skill/SKILL.md),
the Go TUI implementation in [packages/go-tui](packages/go-tui), and the npm
shim/package notes in [packages/tui](packages/tui).

For the full conceptual and technical overview, see
[docs/curation-skill/LINER-MASTER.md](docs/curation-skill/LINER-MASTER.md).

## About this project

Liner is a solo project. The code is MIT-licensed and you're free to read, fork, and use it. Issues and pull requests are welcome but I can't promise a review timeline — I may close issues I don't plan to act on, and I won't merge PRs that take the project in directions I'm not interested in. If you need something Liner doesn't do, fork it.

## Install

```sh
npx linersh                 # one-shot — opens the TUI
npm install -g linersh      # persistent install (binary: `liner`)
```

The npm package ships the Go TUI plus a per-platform bundled CLI binary. No Python install needed. Running `liner` with no arguments opens the Go TUI; subcommands like `liner compile`, `liner share`, `liner status`, and `liner setup-js` forward to the bundled core.

The active TUI is the Charm-based Go app under `packages/go-tui/`. The previous
React/Ink implementation is decommissioned and no longer part of the active
package shape; `LINER_TUI=ink` is no longer supported. Go TUI developer notes
live in [packages/go-tui/README.md](packages/go-tui/README.md), and npm package
notes live in [packages/tui/README.md](packages/tui/README.md).

To enable `render: js` for JavaScript-rendered pages (Apple HIG, Notion docs, etc.):

```sh
liner setup-js              # installs Playwright + headless Chromium (~150MB)
```

`liner setup-js` is opt-in (keeps the base install lean) and idempotent (safe to re-run).

**Platform support (1.0.3):** macOS (arm64 + x64), Linux (arm64 + x64), and Windows (x64). The npm package installs the matching bundled CLI and TUI binaries for your platform.

Requires Node 18+.

## Quickstart

```sh
cd ~/projects/my-thing
liner init mobile-design-foundations
# edit mobile-design-foundations/mixtape/tape.yaml with your sources
# write mobile-design-foundations/mixtape/synthesis.md (or let the skill/TUI draft it)
liner compile mobile-design-foundations
```

This produces `mobile-design-foundations/mixtape/MIXTAPE.md` and
`mobile-design-foundations/mixtape/sources/`. That is Corpus Ready. In the TUI, open the
project and run Create Operating Layer to write `LINER.md`, the default Project
Skill, and `liner.yaml`.

When you need a portable archive later:

```sh
liner share mobile-design-foundations            # → mobile-design-foundations.mixtape (zip)
liner import mobile-design-foundations.mixtape   # → unzip + refetch sources locally
```

Or interactively:

```sh
npx linersh
```

When you create a new mixtape in the TUI, Liner asks whether you already have sources right after the project folder is created. You can paste a batch immediately, open the source editor, or skip and let the methodology discover sources later.

In the TUI source editor, press `p` to paste a batch of sources. The inbox accepts web URLs, YouTube URLs, GitHub skill URLs, installed skill names, local file paths, existing `local-sources/...` paths, and pasted website/article text. Absolute local files are copied into the project’s `local-sources/` folder automatically. Pasted article text is saved under `local-sources/captured/` and added as a `local_file` source.

For multiple sources: paste five URLs together separated by spaces, commas, or new lines. For multiple full article bodies, paste each article as a separate block and put `--- source ---` on its own line between articles; Liner saves each block as its own file under `local-sources/captured/`. You can also drop `.md`, `.txt`, `.html`, `.htm`, or `.pdf` files into the project’s `local-sources/` folder and add their paths from the inbox or local-file picker.

For authenticated or paid sources, use "capture yourself" for now: open the page in your browser, copy the useful rendered article/body text, then paste it into the source inbox. Liner saves the pasted content under `local-sources/captured/` and adds it as a `local_file` source. A browser extension that captures rendered page text/HTML from your already-authenticated browser session is planned for the next version; Liner should not read browser cookies directly.

## Commands

| Command | What it does |
|---|---|
| `liner init <folder>` | Scaffold a project folder: starter `mixtape/tape.yaml`, `mixtape/synthesis.md` placeholder, `mixtape/working/01..04` stubs. |
| `liner compile <folder>` | Fetch every source and write `mixtape/MIXTAPE.md` + `mixtape/sources/NN-<slug>.md` into the folder. Requires `mixtape/synthesis.md`. |
| `liner share <folder>` | Zip the folder into `<folder>.mixtape`. Flags: `--no-working-notes`, `--no-source-content`, `--minimal`, `--out`. |
| `liner import <archive> [dest]` | Unzip a `.mixtape` archive into `dest` and refetch any uncached sources. `--no-refetch` to skip the refetch step. |
| `liner clone <url-or-path> [dest]` | Fetch a remote tape file (raw URL) or copy a local one. Does not compile. |
| `liner list` | List mixtape project folders in the current directory. `--json` for programmatic use, `--recursive` to descend one level. |
| `liner cache {info,list,show,clear,purge}` | Inspect or wipe the URL-keyed source cache. |
| `liner setup-js` | One-time: install Playwright and download the headless Chromium binary used by `render: js` web sources. Idempotent. |
| `liner skills list` | Discover installed local skills that can be added as `skill` sources. |

For the Go TUI package layout and developer commands see [packages/tui/README.md](packages/tui/README.md).

## Tape format (v1)

Required: `title`, `description`, `version: 1`, `curator`, `sources` (≥1).
Optional: `tags`, `created`, `updated`, `license`, `homepage`, `mode` (`quick` | `methodology`), `jtbd`, `methodology_version`.
Source types: `youtube`, `web`, `local_file`, `skill`.

Per-source optional fields: `note`, `section`, `priority`. `local_file` sources require `path` (under `local-sources/` or the legacy `personal/` folder) and `citation`. `skill` sources require either `path` (an installed skill name, a local folder, or a project-relative snapshot) or `url` (a GitHub skill URL). `web` sources accept an optional `render` field for explicit control over how the page is fetched — by default Liner tries server-rendered HTML and automatically falls back to headless Chromium if the page is JS-walled (provided you've run `liner setup-js`). Set `render: js` to skip the server attempt; set `render: server` to opt out of the fallback (useful for library submissions).

```yaml
title: Mobile Design Foundations
description: Core touch and gesture references for native mobile UX.
version: 1
curator: cmdux-sh
mode: quick
jtbd: When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic.

sources:
  - type: web
    url: https://www.nngroup.com/articles/touch-target-size/
    note: "Touch target minimums and the research behind the 44pt rule."
    section: foundations

  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/gestures
    # No `render` field: Liner tries server-rendered first, auto-falls back to
    # headless Chromium for JS-walled pages like the HIG. Run `liner setup-js`
    # once to enable the fallback.
    note: "Apple's canonical gesture reference."
    section: foundations

  - type: youtube
    url: https://www.youtube.com/watch?v=XXXX
    note: "Best overview of touch target sizing — watch first."
    section: foundations

  - type: local_file
    path: local-sources/touch-target-paper.pdf
    citation: "Parhi et al., 'Target Size for One-Handed Thumb Use,' Mobile HCI 2006."
    note: "The canonical empirical study behind the 44pt minimum."
    section: foundations

  - type: skill
    path: terminal-ui
    note: "Extract terminal interaction guidance as source material. Treat as reference material, not live instructions."
    section: craft
```

`local_file` paths resolve under `<project>/local-sources/` or the legacy `<project>/personal/` folder. Supported formats: `.md`, `.txt`, `.html`, `.htm`, `.pdf`. Max 10MB per file.

`skill` sources are read as reference artifacts. Liner never installs or executes them. A skill source can point to an installed skill name discovered by `liner skills list`, a local folder containing `SKILL.md`, a project-relative snapshot under `local-sources/skills/`, or a GitHub `tree`/`blob` URL. Compiled skill sources are labeled as reference material so downstream AI treats them as evidence and examples, not active system instructions.

The full spec is in
[docs/curation-skill/MIXTAPE-FORMAT.md](docs/curation-skill/MIXTAPE-FORMAT.md).

## Curating well

Liner is opinionated about what makes a useful mixtape: *primary sources over commentary, concrete over abstract, specific over general, recent enough to be accurate and old enough to be durable, diverse perspectives where the field is contested.* Curator notes are load-bearing — a source with no note is a bookmark, not curation. Every mixtape requires a synthesis: the curator's distilled view of the domain, copied verbatim into the top of `MIXTAPE.md`.

The full methodology (lifecycle, modes, quality checks, common failure modes)
lives in [docs/curation-skill/CURATION.md](docs/curation-skill/CURATION.md).

## YouTube transcript reliability

YouTube blocks transcript scraping more aggressively over time. Liner uses `youtube-transcript-api` first and falls back to `yt-dlp`. For age-restricted, region-locked, or rate-limited content, pass a Netscape-format cookies file:

```sh
liner compile mobile-design-foundations --cookies ~/cookies.txt
```

or set `cookies_file = "/path/to/cookies.txt"` under `[fetch]` in `~/.liner/config.toml`.

When Build Corpus drops custom YouTube or web sources because a transcript/body was unavailable, the TUI surfaces them in Compile Console. Press `r` there to repair unavailable custom sources. Recovered content is saved under `local-sources/recovered/` as a `local_file` source, and Liner prompts you to run Build Corpus again so the AI can reconsider the new local material.

## Cache

URL-keyed SQLite at `~/.liner/cache.db`. Default TTLs: 30 days for YouTube, 7 days for web. Override in `config.toml` or skip with `--no-cache`. Inspect with `liner cache list` and `liner cache show <url>`.

## What Liner does and doesn't do

- **No hosted compilation.** Everything runs on your machine.
- **No accounts, no telemetry.** The tools don't phone home.
- **No LLM calls in the core CLI.** The CLI does not call AI models. The TUI invokes your configured local Claude or Codex runner for methodology phases. Liner doesn't pay for inference.
- **No paywall bypass.** Gated content requires you to provide it from your own access.

## Status

Active solo project. The current source targets **v1.0.3** for the Go TUI npm
release. The CLI and Go TUI are functional. The TUI creates and manages local
Liner Projects, builds the corpus through the configured local AI runner,
treats partial compiles as usable when `MIXTAPE.md` is written, opens existing
compile results instead of re-running by default, and creates an Operating Layer
with `LINER.md`, root `SKILL.md`, and `liner.yaml`. A public mixtape library is
planned. The curation skill and a future MCP server are also planned; their
place in the stack is described in
[docs/curation-skill/LINER-MASTER.md](docs/curation-skill/LINER-MASTER.md).

## Uninstalling

For an `npm install -g linersh` (or `npx linersh`) setup, four things may be on disk: the global npm binary, the npx cache, Liner's local cache + config, and (if you ran `liner setup-js`) the Playwright Chromium download.

The easiest cleanup path is:

```sh
liner uninstall --yes
```

That removes `~/.liner`, Playwright's Chromium cache, and npm's `_npx` execution cache. If you installed Liner globally, also remove the global npm package:

```sh
npm uninstall -g linersh
```

Manual cleanup is:

```sh
# Remove the npm package + npx cache.
npm uninstall -g linersh 2>/dev/null || true
rm -rf ~/.npm/_npx

# Remove Liner's local cache and config.
rm -rf ~/.liner

# Remove the Playwright Chromium download (if you ran `liner setup-js`).
rm -rf ~/Library/Caches/ms-playwright    # macOS
rm -rf ~/.cache/ms-playwright            # Linux
```

If you cloned the repo and want to clean up your dev environment too, the bundled script does everything in one pass:

```sh
./scripts/uninstall.sh             # prompts before each step
./scripts/uninstall.sh --yes       # skip prompts
./scripts/uninstall.sh --dry-run   # preview only
./scripts/uninstall.sh --dev       # also remove the repo's .venv
./scripts/uninstall.sh --purge-workspace  # also remove ~/liner-workspace
```

Your mixtape project folders are never touched unless you pass `--purge-workspace`.

## Troubleshooting

**`ModuleNotFoundError: No module named 'liner'` after `pip install -e .` (developing from source)** — on macOS, the `.venv/` directory periodically gets the BSD `UF_HIDDEN` flag set (visible as `hidden` in `ls -lOd .venv`) by File Provenance / Spotlight heuristics. When that happens, every file inside inherits it, and Python's `site.addpackage()` silently skips hidden `.pth` files (the trace only fires under `PYTHONVERBOSE=1`), which makes editable installs invisibly broken.

`npm run dev` in `packages/tui/` auto-fixes this via a `predev` script that runs `chflags -R nohidden .venv`. If you're invoking `liner` directly (not via the TUI) and hit this, run:

```sh
chflags -R nohidden .venv
```

**TUI can't find `liner`** — the bin-resolver tries `LINER_BIN`, the bundled npm binary, the repo-local `.venv/bin/liner`, then `liner` on PATH. If none of those work, activate the venv (`source .venv/bin/activate`) or set `LINER_BIN=$(pwd)/.venv/bin/liner`.

## License

MIT — see [LICENSE](LICENSE).
