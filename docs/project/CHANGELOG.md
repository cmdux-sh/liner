# Changelog

## Unreleased

- **Refined Compile Console source repair.** Compile now opens with a compact
  result summary and sends users to `View sources` for the navigable source
  table. Sources with errors, retryable custom sources, and recovered content
  needing Build Corpus are bubbled to the top with clearer `research source` and
  `custom source` labels.
- **Made custom source recovery explicit.** Repair retries unavailable custom
  YouTube/web sources, installs JS rendering first when needed, stops on a
  recovery review, and asks for a corpus rebuild only when content was actually
  recovered.
- **Kept replacement-source repair in context.** Adding sources from Compile now
  returns to Compile instead of restarting Clarify Goal, and recovered/custom
  local sources are preserved as active source material for the next corpus
  build.
- **Allowed usable partial compiles to continue after review.** Reviewed source
  warnings stay visible, but a usable `MIXTAPE.md` can proceed to Operating
  Layer creation instead of being blocked by stale compile-error state.
- **Aligned Project Complete with its primary action.** `Next` now opens
  `LINER.md` when the project is complete, so Enter no longer lands on a
  no-op management state.

## 1.0.0

- **Ships the Go TUI as the default Liner interface.** Running `liner` or
  `npx linersh` opens the Charm-based Go TUI for creating, opening, and
  managing local Liner Projects.
- **Packages the TUI and CLI through one npm install command.** The main
  `linersh` package provides the `liner` shim, TypeScript methodology runner,
  and bundled curation docs. The matching optional platform package provides
  both native binaries: `liner` for CLI subcommands and `liner-tui` for the Go
  interface.
- **Creates the Operating Layer for completed projects.** After the corpus is
  ready, Liner writes `LINER.md`, root `SKILL.md`, and `liner.yaml` so future
  AI sessions know how to use the project, what evidence to trust, and where
  the boundaries are.
- **Tightens corpus-to-method generation.** Operating Layer output now promotes
  corpus synthesis, quality checks, source roles, and capability patterns into
  executable project behavior instead of leaving generic source-use rules.
- **Keeps release packaging auditable.** `npm run acceptance:go -- release-smoke`
  now packs a local platform package plus the main package, installs both into
  a clean consumer project, and verifies that the installed shim launches the
  platform-provided Go TUI binary.

## 0.5.7

- **Capture PDF downloads during JS fallback.** When a `.pdf` web source first
  returns a JavaScript or bot-detection stub, Liner falls back to Playwright. If
  Chromium treats that PDF as a download, Liner now captures the downloaded file
  and runs the same shared PDF text extraction path instead of marking the
  source failed with `Page.goto: Download is starting`.

## 0.5.6

- **Extract remote PDFs as PDFs.** Web sources that return `application/pdf`,
  start with `%PDF-`, or use a `.pdf` URL path now run through the shared
  `pdfplumber` extraction path instead of being treated as HTML. This prevents
  raw PDF bytes from being written into `sources/` and `MIXTAPE.md`.
- **Reject stale bad cache entries.** Cached web content is now purged and
  refetched when it looks like raw PDF bytes, a PDF URL saved by the old HTML
  fallback, a tiny soft/playwright fallback, or a cookie-consent-only body.
- **Detect empty JavaScript app shells.** Server-rendered HTML that looks like a
  client-side shell with no useful readable body now triggers the existing
  Playwright fallback instead of becoming a misleading successful source.
- **Reject cookie-consent-only sources.** Liner now treats short cookie-banner
  output as unusable, including after Playwright rendering. The source is marked
  unavailable with a clear message instead of feeding consent text into the
  mixtape.
- **Clarify partial compile output.** The CLI now reports usable source count
  versus unavailable placeholder files when a compile is partial, e.g.
  `22/24 usable sources, 2 unavailable placeholders`.
- **Shared PDF extraction helper.** Local and remote PDF handling now use one
  extraction helper so future PDF fixes land in both paths.

## 0.5.5

- **Ships matching 0.5.5 platform binaries.** The Python core is now versioned
  0.5.5 and the npm TUI pins all five optional platform packages to
  `linersh-*@0.5.5`, so the web extraction fixes below reach clean `npx
  linersh` installs on macOS, Linux, and Windows x64.
- **Fixed done-screen scrolling.** The final response now has an internal
  scroll viewport in the terminal alternate screen. In the default done view,
  use ↑/↓ or pgup/pgdn to scroll the green final-response box; inspect mode
  still uses those keys for tool-call navigation and keeps the final response
  compact so the detail panel remains visible.
- **Surfaced missing JS-rendering setup during compile.** If a compile hits
  JavaScript-only pages and Playwright Chromium is missing, the TUI now shows a
  dedicated JS-rendering callout with a `j` action that runs
  `liner setup-js --yes`, notes the ~150MB Chromium download, and retries the
  compile.
- **Fixed compile-result source scrolling.** Completed compile screens now keep
  the source list in an internal viewport with ↑/↓ and pgup/pgdn controls, so
  long source lists remain readable in the alternate screen.
- **Clarified the ready-to-compile milestone.** After Assembly, the hub now says
  the methodology is complete and the final mix is assembled, instead of only
  showing Compile as the next step.
- **Treat partial compiles as compiled.** If compile writes `MIXTAPE.md` with
  some missing sources, the hub now lands in a finished "compiled with
  warnings" state instead of asking the user to compile again.
- **Open existing compile results without rerunning.** When `MIXTAPE.md` is
  current, pressing Enter on the completed Compile row opens the final compile
  result screen reconstructed from disk. Recompile is now explicit with `r`
  from that screen.
- **Hardened web extraction fallback.** If the primary HTML extractor crashes
  on a page, Liner now falls back to tag-stripped HTML text and keeps the
  source with a soft warning instead of failing it with a Python exception.
- **Added first-run JS-rendering onboarding.** Fresh installs now get an
  optional setup screen explaining Playwright/Chromium support right after the
  agent/model setup flow, with install and skip paths.
- **Added `liner uninstall`.** The npm shim can now remove local Liner state,
  Playwright's Chromium cache, and npm's `_npx` execution cache for clean
  reinstall testing.

## 0.5.4

- **Fixed npm skill-bundle resolution.** The TUI now probes the installed
  package root more directly, tolerates `LINER_SKILL_PATH` pointing either at
  the skill directory or at `SKILL.md`, and shows the checked paths when the
  bundle still cannot be found. This fixes the Phase 2 "Couldn't find the
  curating-mixtapes skill bundle" failure seen after `npx linersh`.
- **Prepared Windows optional dependency wiring.** `linersh@0.5.4` declares
  `linersh-win32-x64@0.5.0` alongside the existing macOS/Linux platform
  packages. Publish the Windows platform package before publishing the main
  package so Windows installs receive the bundled CLI binary.
- **Marketing landing page polish.** The Astro marketing site has a refreshed
  hero, copyable `npx linersh` install CTA, version-consistent `0.5.3`
  labeling, Liner logo treatment, WebGL resize fix, continuous marquees,
  tightened mobile behavior, and a marketing-site handoff document that was
  later archived outside Git.
- **Per-phase model selection.** The two heaviest phases — candidate discovery
  and evaluation, ~65% of a cycle's tokens — now run on a cheaper model by
  default (Sonnet for Claude, `gpt-5-mini` for Codex). Framing, quality,
  synthesis, and assembly stay on the agent's default model, where curator
  taste matters most. Cuts a cycle's quota roughly 5×. Override per agent +
  phase in `~/.liner/config.yaml` under `models:` — e.g. set `candidates: opus`
  to undo the downgrade. On a subscription the saving is rate-limit quota, not
  money. Claude phases use the `sonnet` alias, which auto-tracks the latest
  Sonnet, so model version bumps don't need a release.
- **Model-rejection fallback.** If the agent rejects the configured model for a
  phase (renamed, retired, or mistyped), the run warns the curator with the
  exact fix — set/remove `models.<agent>.<phase>` in `~/.liner/config.yaml` —
  and retries once on the agent's default model, so a stale model id degrades
  to "ran on the default" instead of a hard failure.
- **Agent-unavailability watchdog.** A launched-but-mute agent (signed out,
  expired subscription, deprecated model) used to hang the run spinner
  indefinitely. The runner now declares the agent unreachable if it emits no
  output within 15s, surfaces a clear "no response from <agent> — check
  auth/subscription" message, and lands the run in the failed state where
  retry / switch-agent already live.
- **Removed the invented `$` cost** from the process-manifest screen — the
  PhaseRunner's was already dropped in 0.5.2. Subscription-driven runs aren't
  metered API billing, so the dollar figure was fictional; token counts stay.

## 0.5.3

Version-reporting fix + version-pin cleanup.

- **`liner --version` now reports the TUI/npm version**, not just the bundled
  Python core. It used to forward straight to the core and print e.g.
  `liner 0.5.0` regardless of which package you installed — confusing when
  verifying a publish. Now prints `liner <tui> (tui)  ·  <core> (core)` (and
  just the TUI version when the core isn't resolvable).
- **Single source of truth for the version.** The header chip and the boot
  splash now read the version from `package.json` at runtime instead of
  separately-edited string constants. (The splash had silently sat at `0.5.0`
  through the entire 0.5.2 cycle.) Only `package.json` needs bumping on release.

## 0.5.2

UX redesign release. The project hub, import flow, navigation, and color
language all get a pass; the optional empirical-test phase is removed.

- **Removed the empirical-test phase from the TUI.** Compiling is now the
  completion milestone — *compiled = complete*. There's no longer a trailing
  optional phase the hub parks you on. (The with-vs-without validation lives
  on as methodology guidance, not a driven phase.)
- **Hub redesigned into a single summary view:** a status box (Ready to use /
  In progress / Not started) plus an always-visible phase checklist. Replaces
  the old focus-card / list dual mode.
- **Import auto-detects `.mixtape` files** in the current folder, offers a
  re-scan, drops the opaque "archive" label, and lands you on the imported
  mixtape's hub rather than a phase prompt.
- **Consistent quit key:** `q` quits from any screen; `esc` goes back.
- **One call-to-action per screen:** a single "next · …" description line
  above a uniform key-hint navbar.
- **Removed the misleading dollar-cost figure** from run summaries —
  subscription-driven runs aren't metered API billing. Token counts stay.
- **Reduced scrollback laddering:** the run's final-response panel caps to the
  terminal height (on top of the alternate-screen buffer added in 0.5.1).
- **Fixed the mixtape browser laddering + garbled selected row.** The project
  list now renders into a terminal-sized scrolling viewport (windowed around
  the selection, with above/below indicators) and each row is a single
  truncating line. Previously, on a small terminal the list rendered taller and
  wider than the window — so arrow-key navigation reprinted the header on every
  keystroke and the selected row's highlight bled across wrapped lines. The
  header subtitle also no longer wraps.
- **Fixed a hard API-400 crash on long agent runs.** A multi-turn methodology
  phase (and any resume of a failed run) could die with `400 … thinking blocks
  in the latest assistant message cannot be modified` once Claude Code
  auto-compacted a conversation that had interleaved thinking on. The spawned
  agent now runs with extended thinking disabled (`MAX_THINKING_TOKENS=0`), so
  the failure class can't occur.
- **Isolated the methodology agent.** The spawned Claude now runs with
  `--strict-mcp-config`, so it no longer inherits the user's global MCP servers
  and connectors — less context bloat, narrower reach, no irrelevant tools.

## 0.5.1

First broad bug-fix + polish pass after the initial public release.

- **Fixed:** the `curating-mixtapes` skill bundle now ships inside the npm
  package, so AI-assisted JTBD clarification works for npm installs (it was
  silently falling back to canned questions in 0.5.0).
- New color palette: red-orange primary, green for success/progress, goldenrod
  for the running state, purple headings, hot-pink errors.
- Legibility: replaced Ink's terminal-relative dim with an explicit grey, and
  named-"black" text on colored fills with true black (`#000000`).
- Fixed an ESC navigation loop; ESC now always walks back to the splash.
- Terminal alternate-screen buffer + a clear on run completion to cut
  scrollback artifacts.
- Contextual notes below boxes (curator note, share result) are pink; the
  share result is a persistent note instead of a vanishing toast.
- New `o` command opens the mixtape folder in the OS file manager.
- Cassette splash logo cleaned up and rendered in the brand color.

## 0.5.0 — first public release

TUI npm bundle release. This is the package path for publishing the interactive TUI with a bundled per-platform Liner core.

- `linersh` npm package now declares per-platform optional CLI packages (`linersh-darwin-arm64`, `linersh-darwin-x64`, `linersh-linux-arm64`, `linersh-linux-x64`, `linersh-win32-x64`) so the TUI can find the bundled core on clean machines.
- The npm `liner` shim now opens the TUI when run with no arguments and forwards CLI subcommands (`setup-js`, `compile`, `share`, `status`, etc.) to the bundled core.
- Added publish guards: `prepack` builds `dist/`, and `prepublishOnly` runs typecheck, tests, and build before publish.
- Bumped the TUI package/version labels to `0.5.0`.
- Updated npm/TUI docs to describe the bundled CLI and `liner setup-js` path for JavaScript rendering.

## 0.3.0 — unreleased

`local_file` source type + JavaScript-rendering support. Additive to the v1 tape format (now versioned as v1.1 in MIXTAPE-FORMAT.md). No breaking changes to existing tapes.

- New source type `local_file` for content the curator has on disk (PDF, Markdown, text, HTML). Required fields: `path` (must resolve under the project's new `personal/` subdirectory) and `citation`. Max 10MB per file.
- New `render` field on `web` sources. Default `server` (existing behavior); `js` opts into Playwright-backed rendering for JavaScript-only sites.
- New optional dependency group `linersh[js]` — installs Playwright + Chromium for `render: js`. The default `pipx install linersh` stays lean (~30MB); `pipx install 'linersh[js]'` adds ~150MB.
- `pdfplumber` added to core dependencies for PDF extraction.
- New `liner share --no-personal` flag. `personal/` is included in exports by default; library submissions must pass `--no-personal` and use only public sources.
- `share` prints a soft warning when the tape contains `local_file` sources, noting the result isn't library-eligible.
- Web handler signature changed from `fetch(url)` to `fetch(spec)` so handlers can see metadata they may need (e.g. `render`). Internal change; doesn't affect tape format.
- TUI: new file-picker screen for `local_file` paths; source-modal type toggle (`T`) and `render` toggle (`R`); source-row icons updated.
- 8 new tests; existing test suites extended.
- New `scripts/uninstall.sh` — interactive bash uninstaller that removes the CLI, the Playwright browser, the local cache, and (optionally) globally installed TUI / dev `.venv` / mixtape workspace. Documented in README.
- New `liner setup-js` subcommand. One-shot: installs Playwright via `pipx inject` (or `pip` for non-pipx envs) and downloads the Chromium binary. Idempotent. Replaces the documented two-step "`pipx install 'linersh[js]'` + remember the playwright path" recipe.
- **Web rendering is now auto-fallback by default.** A `web` source with no `render` field tries server-rendered HTML first, and if the page turns out to be JS-walled, automatically falls back to headless Chromium (provided `liner setup-js` has been run). The compile output shows a warning when the fallback triggers so the curator sees the cost. Existing tapes with `render: js` continue to skip the server attempt; new `render: server` opt-out is available for library submissions or diagnostics.

## 0.2.0 — unreleased

Folder-model migration. The mixtape is now a project folder (`tape.yaml`, `synthesis.md`, `sources/`, `working/`, `MIXTAPE.md`), not a single Markdown file. Breaking changes vs. 0.1:

- `liner compile` now takes a project folder and writes outputs in place. The `--output`, `--format`, and `--no-preamble` flags are removed.
- `liner init` now scaffolds a project folder with the starter tape, a `synthesis.md` placeholder, and `working/01..04` stubs.
- `synthesis.md` is required by `liner compile` — missing it is a hard error.
- New commands: `liner share <folder>` (zip → `.mixtape`) and `liner import <archive> [dest]` (unzip + refetch). Share flags: `--no-working-notes`, `--no-source-content`, `--minimal`, `--out`.
- New cache subcommands: `liner cache list`, `liner cache show <url>`.
- Tape format adds optional fields: `mode` (`quick` | `methodology`), `jtbd`, `methodology_version`.
- `liner list` walks for project folders (directories containing `tape.yaml`) instead of flat `*.yaml` files.
- `--emit-events` final `result` payload now reports the produced folder, `MIXTAPE.md` path, and per-source file paths instead of a rendered Markdown string.
- Example reshaped from `docs/examples/mobile-design.yaml` to `docs/examples/mobile-design-foundations/` (runnable project folder).

## 0.1.0 — unreleased

Initial Liner MVP. The project was originally prototyped as "yoyotape" and renamed to Liner before first release; this is the public 0.1.0.

- CLI: `liner init`, `liner compile`, `liner clone`, `liner list`, `liner cache {info,clear,purge}`.
- TUI (`npx linersh` / `liner`): interactive tape editor with browser, editor, source modal, and live-streaming compile view.
- YouTube handler with `youtube-transcript-api` → `yt-dlp` fallback and `--cookies` support.
- Web handler via trafilatura (no JavaScript rendering in v1).
- SQLite cache at `~/.liner/cache.db` (30-day TTL for YouTube, 7-day for web).
- Markdown and JSON output. Compiled mixtapes save as `<tape>.mixtape.md`.
- `liner compile --emit-events` streams NDJSON progress events to stdout for programmatic clients (TUI, future MCP server).
- Soft curator-note nudge in TUI per CURATION.md §6.
