# Liner

**Liner** is an open-source toolkit for building curated *mixtapes* — context bundles that make AI systems more useful within a specific domain.

A *tape file* (`tape.yaml`) is the recipe. A *mixtape* is the project folder Liner produces when it compiles a tape: `tape.yaml`, `synthesis.md`, `sources/`, `working/`, and a master `MIXTAPE.md` ready to paste into an AI conversation. Tape ≠ mixtape — the recipe is small; the mixtape is the realized artifact.

This repo contains the **CLI**, the **TUI**, packaging scripts, tests, examples, and the curation methodology. The methodology lives in [cli-update-docs/CURATION.md](cli-update-docs/CURATION.md), the platform doc in [cli-update-docs/PLATFORM.md](cli-update-docs/PLATFORM.md), the format spec in [cli-update-docs/MIXTAPE-FORMAT.md](cli-update-docs/MIXTAPE-FORMAT.md), and the curation skill in [cli-update-docs/SKILL.md](cli-update-docs/SKILL.md).

For the full conceptual and technical overview, see [cli-update-docs/LINER-MASTER.md](cli-update-docs/LINER-MASTER.md).

## About this project

Liner is a solo project. The code is MIT-licensed and you're free to read, fork, and use it. Issues and pull requests are welcome but I can't promise a review timeline — I may close issues I don't plan to act on, and I won't merge PRs that take the project in directions I'm not interested in. If you need something Liner doesn't do, fork it.

## Install

```sh
npx linersh                 # one-shot — opens the TUI
npm install -g linersh      # persistent install (binary: `liner`)
```

The npm package ships the TUI plus a per-platform bundled CLI binary. No Python install needed. Running `liner` with no arguments opens the TUI; subcommands like `liner compile`, `liner share`, `liner status`, and `liner setup-js` forward to the bundled core.

To enable `render: js` for JavaScript-rendered pages (Apple HIG, Notion docs, etc.):

```sh
liner setup-js              # installs Playwright + headless Chromium (~150MB)
```

`liner setup-js` is opt-in (keeps the base install lean) and idempotent (safe to re-run).

**Platform support (0.5.7):** macOS (arm64 + x64), Linux (arm64 + x64), and Windows (x64). The npm package installs the matching bundled CLI binary for your platform.

Requires Node 18+.

## Quickstart

```sh
cd ~/projects/my-thing
liner init mobile-design-foundations
# edit mobile-design-foundations/tape.yaml with your sources
# write mobile-design-foundations/synthesis.md (or let the skill/TUI draft it)
liner compile mobile-design-foundations
```

This produces `mobile-design-foundations/MIXTAPE.md` and `mobile-design-foundations/sources/`. To share:

```sh
liner share mobile-design-foundations            # → mobile-design-foundations.mixtape (zip)
liner import mobile-design-foundations.mixtape   # → unzip + refetch sources locally
```

Or interactively:

```sh
npx linersh
```

## Commands

| Command | What it does |
|---|---|
| `liner init <folder>` | Scaffold a project folder: starter `tape.yaml`, `synthesis.md` placeholder, `working/01..04` stubs. |
| `liner compile <folder>` | Fetch every source and write `MIXTAPE.md` + `sources/NN-<slug>.md` into the folder. Requires `synthesis.md`. |
| `liner share <folder>` | Zip the folder into `<folder>.mixtape`. Flags: `--no-working-notes`, `--no-source-content`, `--minimal`, `--out`. |
| `liner import <archive> [dest]` | Unzip a `.mixtape` archive into `dest` and refetch any uncached sources. `--no-refetch` to skip the refetch step. |
| `liner clone <url-or-path> [dest]` | Fetch a remote tape file (raw URL) or copy a local one. Does not compile. |
| `liner list` | List mixtape project folders in the current directory. `--json` for programmatic use, `--recursive` to descend one level. |
| `liner cache {info,list,show,clear,purge}` | Inspect or wipe the URL-keyed source cache. |
| `liner setup-js` | One-time: install Playwright and download the headless Chromium binary used by `render: js` web sources. Idempotent. |

For TUI keybindings see [packages/tui/README.md](packages/tui/README.md).

## Tape format (v1)

Required: `title`, `description`, `version: 1`, `curator`, `sources` (≥1).
Optional: `tags`, `created`, `updated`, `license`, `homepage`, `mode` (`quick` | `methodology`), `jtbd`, `methodology_version`.
Source types: `youtube`, `web`, `local_file`.

Per-source optional fields: `note`, `section`, `priority`. `local_file` sources require `path` (under `personal/`) and `citation`. `web` sources accept an optional `render` field for explicit control over how the page is fetched — by default Liner tries server-rendered HTML and automatically falls back to headless Chromium if the page is JS-walled (provided you've run `liner setup-js`). Set `render: js` to skip the server attempt; set `render: server` to opt out of the fallback (useful for library submissions).

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
    path: personal/touch-target-paper.pdf
    citation: "Parhi et al., 'Target Size for One-Handed Thumb Use,' Mobile HCI 2006."
    note: "The canonical empirical study behind the 44pt minimum."
    section: foundations
```

`local_file` paths resolve under `<project>/personal/`. Supported formats: `.md`, `.txt`, `.html`, `.htm`, `.pdf`. Max 10MB per file.

A complete example project folder lives at [examples/mobile-design-foundations/](examples/mobile-design-foundations).

The full spec is in [cli-update-docs/MIXTAPE-FORMAT.md](cli-update-docs/MIXTAPE-FORMAT.md).

## Curating well

Liner is opinionated about what makes a useful mixtape: *primary sources over commentary, concrete over abstract, specific over general, recent enough to be accurate and old enough to be durable, diverse perspectives where the field is contested.* Curator notes are load-bearing — a source with no note is a bookmark, not curation. Every mixtape requires a synthesis: the curator's distilled view of the domain, copied verbatim into the top of `MIXTAPE.md`.

The full methodology (lifecycle, modes, quality checks, common failure modes) lives in [cli-update-docs/CURATION.md](cli-update-docs/CURATION.md).

## YouTube transcript reliability

YouTube blocks transcript scraping more aggressively over time. Liner uses `youtube-transcript-api` first and falls back to `yt-dlp`. For age-restricted, region-locked, or rate-limited content, pass a Netscape-format cookies file:

```sh
liner compile mobile-design-foundations --cookies ~/cookies.txt
```

or set `cookies_file = "/path/to/cookies.txt"` under `[fetch]` in `~/.liner/config.toml`.

## Cache

URL-keyed SQLite at `~/.liner/cache.db`. Default TTLs: 30 days for YouTube, 7 days for web. Override in `config.toml` or skip with `--no-cache`. Inspect with `liner cache list` and `liner cache show <url>`.

## What Liner does and doesn't do

- **No hosted compilation.** Everything runs on your machine.
- **No accounts, no telemetry.** The tools don't phone home.
- **No LLM calls in the core.** The CLI and TUI don't talk to AI models. The skill and MCP server (planned) use your own AI subscription; Liner doesn't pay for inference.
- **No paywall bypass.** Gated content requires you to provide it from your own access.

## Status

Active solo project. The current release is **v0.5.7** on npm (`latest`) — install via `npx linersh` (Mac, Linux, and Windows x64). The CLI and TUI are functional. The TUI now treats a partial compile as a usable finished state when `MIXTAPE.md` is written, opens the existing compile result instead of re-running by default, and prompts for optional JS rendering support when needed. A public mixtape library is planned. The curation skill and a future MCP server are also planned; their place in the stack is described in [cli-update-docs/LINER-MASTER.md](cli-update-docs/LINER-MASTER.md).

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
