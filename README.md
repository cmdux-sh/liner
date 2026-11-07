# Liner

**Liner** is a personal library of reusable research that any AI agent can find,
use, and maintain. Each liner is a local folder built for one narrow job. It
holds the sources you reviewed, a synthesis of what they say, and instructions
for using them, so the research outlives the chat that produced it.

- **Your agent does the research.** Liner owns the steps and checks each result.
  Your agent does the work with its own tools, and Liner makes no AI calls.
- **You decide.** Agents suggest a liner and stop for your approval of the
  sources and the synthesis.
- **Liners improve with use.** Questions a liner couldn't answer become gaps,
  what real work teaches waits in an inbox for your review, `liner refresh`
  shows what changed online, and `liner test` records whether a liner helped.
- **It's all files.** Everything stays on your machine in folders you can read.

## Use it from your agent

Install the `liner` skill once, and your agent can find your liners:

```sh
npm install -g linersh
liner agent-skill install claude --yes    # or codex
```

On Claude Code, this also adds the subagent that reads fetched pages and the rules
that make Claude Code ask you before approvals. Add `--no-subagent` or `--no-guard`
to leave either out.

Then ask for lasting research on a subject. The agent follows
`liner research next` to build a liner, stopping for your approval, and later
sessions find it with `liner catalog` and `liner search`. Point a repository at
a liner with `liner use <slug>`. Agents with no shell can use [`liner mcp`](#connect-an-agent-over-mcp).

Run `liner` with no arguments to open the navigator, a live view of your library
to keep beside your chat. It shows every liner, what needs you, your whole source
collection, and how liners connect. You can read any file, follow a citation to
its source, and press `y` to copy a prompt for your agent. Agents do the work, and
the navigator never changes a liner.

## Terms

A *tape file* (`tape.yaml`) is a liner's source recipe. A *Mixtape* is the
compiled corpus Liner builds from that recipe: `synthesis.md`, `sources/`,
`working/`, and `MIXTAPE.md`. A *Liner Project*, a liner for short, is the local
folder around that corpus. After the corpus is ready and you approve the
instructions, the finish step of `liner research` writes `LINER.md`, a root `SKILL.md`, and `liner.yaml` so future AI sessions know
what the resource is for, how to use the evidence, and where its boundaries are.
Tape != Mixtape != Liner Project.

Each liner serves one **Job to Be Done**, captured in **Job Story** form.
**Clarify Job** then makes the situation, motivation, expected outcome, and
boundaries explicit before source work begins. "SEO" is too broad; "When I am
preparing a launch for a mental-health startup, I want evidence-backed keyword
direction, so I can focus the site on credible search intent" is narrow enough
to derive research lanes, find the right sources, and produce a resource a
future agent can use.

This repo contains the **CLI**, the **Go TUI**, the **npm package shell**, and
the project docs. The methodology lives in
[docs/curation-skill/CURATION.md](docs/curation-skill/CURATION.md), the
platform doc in
[docs/curation-skill/PLATFORM.md](docs/curation-skill/PLATFORM.md), the format
spec in
[docs/curation-skill/MIXTAPE-FORMAT.md](docs/curation-skill/MIXTAPE-FORMAT.md),
the curation skill in [src/liner/bundled/research/methodology/SKILL.md](src/liner/bundled/research/methodology/SKILL.md),
the Go TUI implementation in [packages/go-tui](packages/go-tui), and the npm
shim/package notes in [packages/tui](packages/tui).

For the full conceptual and technical overview, see
[docs/curation-skill/LINER-MASTER.md](docs/curation-skill/LINER-MASTER.md).

## About this project

Liner is a solo project. The code is MIT-licensed and you're free to read, fork, and use it. Issues and pull requests are welcome but I can't promise a review timeline — I may close issues I don't plan to act on, and I won't merge PRs that take the project in directions I'm not interested in. If you need something Liner doesn't do, fork it.

## Install

```sh
npx linersh                 # one-shot — opens the navigator
npm install -g linersh      # persistent install (binary: `liner`)
```

The npm package ships the navigator plus a per-platform bundled CLI binary. No Python install needed. Running `liner` with no arguments opens the navigator; subcommands like `liner research`, `liner compile`, and `liner setup-js` forward to the bundled core.

To enable `render: js` for JavaScript-rendered pages (Apple HIG, Notion docs, etc.):

```sh
liner setup-js              # installs Playwright's headless Chromium (about 100 MB download, 220 MB on disk)
```

`liner setup-js` is opt-in (keeps the base install lean) and idempotent (safe to re-run).

**Platform support (1.2.1):** macOS (arm64 + x64), Linux (arm64 + x64), and Windows (x64). The npm package installs the matching bundled CLI and TUI binaries for your platform.

Requires Node 18+.

## Quickstart

```sh
cd ~/projects/my-thing
liner init mobile-design-foundations
liner sources add mobile-design-foundations \
  --type web \
  --url https://developer.apple.com/app-store/review/guidelines/ \
  --note "Use the current review rules as primary evidence." \
  --priority required
# ask your agent to build and review the corpus, or compile what you added
liner compile mobile-design-foundations
```

This produces `mobile-design-foundations/mixtape/MIXTAPE.md` and
`mobile-design-foundations/mixtape/sources/`. That is Corpus Ready. Your agent then
creates `LINER.md`, the default Project Skill, and `liner.yaml` through
`liner research`.

When you need a portable archive later:

```sh
liner share mobile-design-foundations            # → mobile-design-foundations.mixtape (zip)
liner import mobile-design-foundations.mixtape   # → unzip + refetch sources locally
```

To add sources by hand, drop `.md`, `.txt`, `.html`, `.htm`, `.pdf`, or `.epub` files into the liner's `mixtape/local-sources/` folder and add them with `liner sources add`. Older liners that keep `tape.yaml` at the root use a root `local-sources/` folder instead.

For authenticated or paid sources, use "capture yourself" for now: open the page in your browser, copy the useful rendered article/body text, then save it under `mixtape/local-sources/` and add it as a `local_file` source. A browser extension that captures rendered page text/HTML from your already-authenticated browser session is planned for the next version; Liner should not read browser cookies directly.

## Commands

| Command | What it does |
|---|---|
| `liner init <folder>` | Scaffold a project folder: starter `mixtape/tape.yaml`, `mixtape/synthesis.md` placeholder, `mixtape/working/01..04` stubs. |
| `liner compile <folder>` | Fetch every source and write `mixtape/MIXTAPE.md` + `mixtape/sources/NN-<slug>.md` into the folder. Requires `mixtape/synthesis.md`. |
| `liner share <folder>` | Zip the folder into `<folder>.mixtape`. Flags: `--no-working-notes`, `--no-source-content`, `--no-personal`, `--minimal`, `--out`. |
| `liner import <archive> [dest]` | Unzip a `.mixtape` archive into `dest` and refetch any uncached sources. `--no-refetch` skips the refetch step, and `--overwrite` replaces an existing folder's matching files. |
| `liner clone <url-or-path> [dest]` | Fetch a remote tape file (raw URL) or copy a local one. Does not compile. |
| `liner list` | List mixtape project folders in the current directory. `--json` for programmatic use, `--recursive` to descend one level. |
| `liner catalog` | List every liner in your library with its job, readiness, and source count. Reads `LINER_DIR`, then `projects_dir` in `~/.liner/config.yaml`, then `~/liner/projects`. `--json` for agents, `--root` to read another folder. |
| `liner search <query>` | Find cited passages in your liners' syntheses and sources. `--liner <slug>` narrows the search; repeat it for more than one. `--limit`, `--root`, and `--json` are available. |
| `liner use <slug>` | Add a pointer to a liner in this repo's `CLAUDE.md` or `AGENTS.md`. Previews first; `--yes` writes, `--remove` takes it out, and `--repo` and `--agent` choose the target. |
| `liner gaps add <slug> <question>` | Record what a liner could not answer. `liner gaps list` and `liner gaps close` manage them, and `liner research improve` targets the open ones. |
| `liner test start <slug> <question> --without-file <file> --with-file <file>` | Save a with-and-without comparison as unlabeled answers A and B, in random order, for the person to judge. It refuses answers that show which one used the liner: links, footnotes, a source list, or the words liner or mixtape. |
| `liner test judge <slug> <id> --preferred <a\|b\|same>` | Record the person's choice as a verdict and reveal which answer used the liner. |
| `liner test add <slug> <question> --without-file <file> --with-file <file> --verdict <better\|same\|worse>` | Record a comparison the person judged with the labels showing. `liner test list <slug>` shows a liner's tests, and `liner catalog` shows all results and how many were blind. |
| `liner mcp` | Serve your liners to an agent with no shell over MCP. Add it to a client with the command `liner` and the argument `mcp`. It can't approve anything. |
| `liner refresh <slug>` | Refetch a liner's online sources and report what changed. It writes `mixtape/.liner-runs/refresh/report.json` and a diff for each changed source, which the navigator's Attention view shows, but it doesn't change the liner's sources or synthesis. It exits with 2 when some sources can't be reached and 3 when none can. |
| `liner findings add <slug> <finding>` | Put what real work learned in the inbox. `list` shows it; `accept --yes` adds it as a Source and `reject --yes` declines it. |
| `liner research start <job>` | Start a liner that your own agent builds. `liner research next <slug> --json` gives the current step with its prompt in a file, `liner research plan <slug>` lists the steps ahead and the kind of work each needs, `submit` checks a step, and `approve` records the person's approval. |
| `liner research check <project> <phase>` | Check a research phase's artifact without moving on. |
| `liner find-sources "<words>"` | List sources already in your library, with the liner that holds each one, its earlier verdict, and a `reuse` address that copies its saved text into another liner, in short capped rows. `--json` for agents. |
| `liner fetch <slug> <url>` | Save a page, or a YouTube video's transcript, to a file in the liner and return its path, so untrusted text stays out of the agent's session. It warns when almost nothing was saved. `--json` for agents. |
| `liner research subagent install --yes` | Add the optional Claude Code subagent that reads fetched pages without shell, network, or write tools. This reduces exposure but is not isolation: it can still read local files, and its report goes to the main agent. `liner agent-skill install claude` adds it for you. If your session can't use it yet, the next one will. `inspect` and `remove` are available. |
| `liner research guard install --yes` | Make Claude Code ask you directly before an agent approves research, accepts or rejects a finding, records a test verdict, or applies a change with `--approve`. `liner agent-skill install claude` adds it for you. It adds `ask` rules and a hook that reads the whole command, so a command wrapped in `bash -c` or `env` asks too, to `~/.claude/settings.json`. It keeps everything else there and saves a `settings.json.liner-backup` copy the first time. This is a guardrail, not a lock: an agent can still get past it, for example with a script that runs the command. `inspect` and `remove` are available. |
| `liner cache {info,list,show,clear,purge}` | Inspect or wipe the URL-keyed source cache. |
| `liner project inspect [path]` | Read a versioned Project Snapshot with immutable IDs, compatibility, lifecycle state, capabilities, and root-discovery evidence. |
| `liner project guidance <folder> --format json\|markdown` | Print the installed CLI's versioned maintenance contract, compatibility state, instruction allowlist, and exact remediation. |
| `liner project plan <folder> --request-json <json>` | Create a write-free Change Set from a versioned maintenance request. |
| `liner project pointer <folder> --environment <codex\|claude> --action <install\|update\|remove>` | Create a Change Set for an opt-in managed `AGENTS.md` or `CLAUDE.md` pointer block. |
| `liner project rename <folder> --name <name>` | Plan an identity-preserving Project display-name change. |
| `liner project move <folder> --destination <path>` | Plan an atomic Project root move with destination and collision checks. |
| `liner project apply <folder> --change-set-json <json>` | Apply one exact versioned Change Set atomically and write a durable receipt. |
| `liner sources add <folder> --type <type> ...` | Plan an atomic Source addition with duplicate detection and provenance. |
| `liner sources update <folder> --source-id <id> ...` | Update metadata or a locator while preserving the immutable Source ID. |
| `liner sources replace <folder> --source-id <id> --type <type> ...` | Replace a Source semantically with successor identity and lineage. |
| `liner sources remove <folder> --source-id <id>` | Detach a Source into the retention vault without destructive deletion. |
| `liner sources purge <folder> --source-id <id>` | Plan a separately approved destructive purge of retained Source content. |
| `liner adapters inspect <codex\|claude>` | Inspect an optional Maintenance Adapter without changing it. |
| `liner adapters install <codex\|claude>` | Explicitly install the thin CLI-delegating Maintenance Adapter. |
| `liner adapters update <codex\|claude>` | Update only content owned by the managed adapter. |
| `liner adapters remove <codex\|claude>` | Remove only Liner-managed adapter content while preserving user content. |
| `liner agent-skill inspect <codex\|claude>` | Check the optional `liner` skill, including whether its subject list is out of date. |
| `liner agent-skill install <codex\|claude>` | Preview, or with `--yes` install, a skill that lets your agent find, suggest, and build your liners. On Claude Code, it also adds the subagent and the approval rules; `--no-subagent` and `--no-guard` leave them out. `remove` takes out only the skill. |
| `liner agent-skill update <codex\|claude>` | Refresh the skill and the subjects it names. |
| `liner agent-skill remove <codex\|claude>` | Remove only Liner-managed content and keep your own text. |
| `liner setup-js` | One-time: install Playwright and download the headless Chromium binary used by `render: js` web sources. Idempotent. |
| `liner skills list` | Discover installed local skills that can be added as `skill` sources. |

The `liner` skill names your usable liners in its description, so an agent can notice one that fits the work in front of it. It suggests a liner and asks before loading it. It is never installed automatically, and it needs the `liner` command on your `PATH`.

The maintenance skill ships with Liner but is never installed automatically. It delegates to
the installed CLI's current guidance and does not replace the canonical Project Skill. Adapter
removal deletes only Liner-managed content and preserves user-authored files and text outside
managed markers.

For the package layout and developer commands see [packages/tui/README.md](packages/tui/README.md).

## Tape format (v1)

Required: `title`, `description`, `version: 1`, `curator`, `sources` (a list, which may be empty while a liner is being built).
Optional: `tags`, `created`, `updated`, `license`, `homepage`, `jtbd`, `methodology_version`. Tapes may also carry `mode`, a label that changes nothing.
Source types: `youtube`, `web`, `local_file` (including EPUB), `skill`, `repository` (the documentation files of a public GitHub repository, at the commit it was compiled from; folder URLs need a branch or tag name without a slash), `liner` (`liner://<slug>`, the reviewed synthesis of another finished liner in the same library, or `liner://<slug>/<source-id>`, one of its sources' saved text, including a local file, with its original address; `liner find-sources` gives each source's address).

Per-source optional fields: `note`, `section`, `priority`. `local_file` sources require `path` (under `local-sources/` or the legacy `personal/` folder) and `citation`. `skill` sources require either `path` (an installed skill name, a local folder, or a project-relative snapshot) or `url` (a GitHub skill URL). `web` sources accept an optional `render` field for explicit control over how the page is fetched — by default Liner tries server-rendered HTML and automatically falls back to headless Chromium if the page is JS-walled (provided you've run `liner setup-js`). Set `render: js` to skip the server attempt; set `render: server` to opt out of the fallback (useful for library submissions).

```yaml
title: Mobile Design Foundations
description: Core touch and gesture references for native mobile UX.
version: 1
curator: cmdux-sh
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

`local_file` paths resolve under `<project>/mixtape/local-sources/` or the legacy `personal/` folder beside `tape.yaml`. Older liners with `tape.yaml` at the root resolve them under `<project>/local-sources/`. Supported formats: `.md`, `.txt`, `.html`, `.htm`, `.pdf`, `.epub`. Max 10MB per file.

`skill` sources are read as reference artifacts. Liner never installs or executes them. A skill source can point to an installed skill name discovered by `liner skills list`, a project-relative snapshot under `local-sources/skills/`, or a GitHub `tree`/`blob` URL. Local skill folders must sit inside the liner, in an installed skills folder, or in a folder you list in `LINER_SKILL_PATHS`, because a tape can come from someone else. Compiled skill sources are labeled as reference material so downstream AI treats them as evidence and examples, not active system instructions.

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

When a YouTube transcript or web page is unavailable, save the text under
`mixtape/local-sources/` and add it as a `local_file` Source, then ask your agent to take it
into the liner.

## Private addresses and size limits

Liner refuses to fetch URLs that lead to this machine, your local network, or a cloud metadata address, including through redirects, because a tape can come from someone else. Web pages and PDFs stop at 50 MB. If your sources live on an intranet or a Tailscale network, add this to `~/.liner/config.toml`:

```toml
[fetch]
allow_private_network = true
```

## Cache

URL-keyed SQLite at `~/.liner/cache.db`. Default TTLs: 30 days for YouTube, 7 days for web. Override in `config.toml` or skip with `--no-cache`. Inspect with `liner cache list` and `liner cache show <url>`.

## Connect an agent over MCP

`liner mcp` serves your liners over standard input and output. It can't approve anything, so an agent that only has MCP can't approve its own work, but it can fetch pages into a liner, start a liner, submit research steps, and add gaps and findings.

```sh
claude mcp add --transport stdio liner -- liner mcp    # Claude Code
codex mcp add liner -- liner mcp                       # Codex
```

Other clients, including Cursor (`~/.cursor/mcp.json`) and Claude Desktop, take the same command in their JSON configuration:

```json
{ "mcpServers": { "liner": { "command": "liner", "args": ["mcp"] } } }
```

## What Liner does and doesn't do

- **No hosted compilation.** Everything runs on your machine.
- **No accounts, no telemetry.** The tools don't phone home.
- **No LLM calls in the core CLI.** The CLI does not call AI models. Your agent does the research on your account. Liner doesn't pay for inference.
- **No paywall bypass.** Gated content requires you to provide it from your own access.
- **No guarantee against a malicious page or a determined agent.** [SECURITY.md](SECURITY.md) lists what Liner protects against and what it doesn't.

## Status

Active solo project. The current source targets **v1.2.1**, the current
release on npm. Agents build and maintain liners through `liner research`,
`liner catalog`, `liner search`, and `liner mcp`, and the terminal navigator reads
the same folders. A public library of liners is planned. The methodology's
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
./scripts/uninstall.sh --purge-library  # also delete your liners
```

Without `--purge-library`, the script never touches your liners. With it, the script looks in your library folder: `LINER_DIR` if you set it, otherwise `projects_dir` in `~/.liner/config.yaml`, otherwise `~/liner/projects`, and also in `~/liner-workspace`, an older default. It deletes only the folders in them that are liners, lists them first, and asks you to type how many there are, even with `--yes`. Other files, symlinked liners and the folder around them stay; an emptied library folder is removed. If the library setting can't be read safely, it deletes nothing from the library and asks you to set `LINER_DIR` to the full path. `--purge-workspace` still works as an older name.

## Troubleshooting

**`ModuleNotFoundError: No module named 'liner'` after `pip install -e .` (developing from source)** — on macOS, the `.venv/` directory periodically gets the BSD `UF_HIDDEN` flag set (visible as `hidden` in `ls -lOd .venv`) by File Provenance / Spotlight heuristics. When that happens, every file inside inherits it, and Python's `site.addpackage()` silently skips hidden `.pth` files (the trace only fires under `PYTHONVERBOSE=1`), which makes editable installs invisibly broken.

`npm run dev` in `packages/tui/` auto-fixes this via a `predev` script that runs `chflags -R nohidden .venv`. If you're invoking `liner` directly (not via the TUI) and hit this, run:

```sh
chflags -R nohidden .venv
```

**The navigator can't find `liner`** — it tries `LINER_BIN`, the repo-local `.venv/bin/liner`, the bundled npm binary, then `liner` on PATH. If none of those work, activate the venv (`source .venv/bin/activate`) or set `LINER_BIN=$(pwd)/.venv/bin/liner`.

## License

MIT — see [LICENSE](LICENSE).
