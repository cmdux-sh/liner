# Liner Master Overview

This document explains what Liner is, how it works, how it is built, how it is used, and where it fits relative to skills, AI projects, notebooks, and ordinary source lists.

## Short Version

Liner is an open-source toolkit for creating and maintaining local,
source-grounded AI context Projects. Each Project contains a curated corpus
called a **Mixtape** plus an Operating Layer that makes the evidence reusable by
future AI sessions.

A Liner project is a folder. Its mixtape corpus lives under `mixtape/` in v2 projects and contains:

- a recipe, `mixtape/tape.yaml`
- a curator synthesis, `mixtape/synthesis.md`
- extracted source files in `mixtape/sources/`
- optional working notes in `mixtape/working/`
- a compiled master file, `mixtape/MIXTAPE.md`

The point is simple: AI systems are much better when they are given a thoughtful, domain-specific context bundle instead of a pile of random links or whatever happens to be in the model's training data.

Liner makes that context bundle portable, inspectable, repeatable, and shareable.

## The Premise

Most AI work is bottlenecked by context quality.

If you ask an AI for mobile design feedback with no extra context, it leans on general knowledge. If you give it a carefully curated bundle of Apple HIG references, touch-target research, practitioner essays, pattern critiques, and your own synthesis of how those sources fit together, the answer gets more specific and more useful.

That pattern applies across domains:

- designing a native iOS app
- evaluating CLI/TUI interaction patterns
- choosing an API architecture
- writing an onboarding guide for a team
- reviewing product strategy
- building a hiring rubric
- understanding a technical standard
- comparing implementation approaches

The hard part is not just collecting links. The hard part is curating sources against a specific job-to-be-done, extracting the useful content, explaining why each source matters, and packaging the result so any AI tool can use it.

That is what Liner does.

## Core Terms

### Tape

A **tape** is the recipe. It lives in `mixtape/tape.yaml` in v2 projects.

It names the mixtape, describes its purpose, identifies the curator, and lists the kept sources with notes, sections, and source metadata.

The tape is small and shareable. It can be stored in Git, reviewed in a pull
request, forked, or recompiled later. Initial curation may author the recipe,
but supported post-creation Source maintenance goes through Liner Core so
identity, lineage, lifecycle invalidation, and receipts remain intact.

### Mixtape

A **mixtape** is the corpus inside the project folder.

It includes the tape, the curator's synthesis, extracted source content, working notes, and compiled output.

### Consumable Mixtape

The **consumable mixtape** is what an AI actually reads: `mixtape/MIXTAPE.md` plus the referenced files in `mixtape/sources/`.

`MIXTAPE.md` contains the curator synthesis, a source index, and links to the extracted source files.

### Curator

The **curator** is the person shaping the context bundle.

The curator's job is not to collect everything. It is to decide what belongs, what does not, what each source contributes, and what the AI should understand before using the sources.

## What Liner Produces

A compiled project folder looks like this:

```text
mobile-design-foundations/
|-- liner.yaml
|-- LINER.md
|-- SKILL.md
`-- mixtape/
    |-- MIXTAPE.md
    |-- tape.yaml
    |-- synthesis.md
    |-- sources/
    |   |-- 01-touch-target-size.md
    |   |-- 02-apple-gestures.md
    |   `-- ...
    |-- local-sources/
    |   `-- touch-target-paper.pdf
    |-- .liner-runs/
    `-- working/
        |-- 01-jtbd-and-knowledge-map.md
        |-- 02-candidate-longlist.md
        |-- 03-evaluation.yaml
        |-- 04-quality-checks.md
        |-- 05-operating-fit-audit.md
        |-- 07-tape-draft.yaml
        `-- empirical-tests.yaml
```

`liner.yaml`, `LINER.md`, and `SKILL.md` are the Operating Layer. `mixtape/tape.yaml` is the source recipe. `mixtape/synthesis.md` is the curator's distilled framing. `mixtape/sources/` contains extracted content. `mixtape/working/` contains the curation trace. `mixtape/.liner-runs/` holds fetched pages, step prompts, and review copies. `mixtape/MIXTAPE.md` is the compiled entry point.

## What Liner Is Good For

Liner is good when the AI needs durable, domain-specific context and when source choice matters.

Examples:

- **Mobile design foundations**: Apple HIG pages, touch target research, gesture references, accessibility guidance, and a synthesis of how native mobile conventions fit together.
- **CLI/TUI design**: command-line UX talks, terminal UI framework docs, interaction design principles, and examples of developer tools that feel good to use.
- **API design migration**: internal docs, public API guidelines, REST/GraphQL trade-off essays, platform-specific constraints, and migration examples.
- **Hiring rubrics**: calibration documents, role expectations, interview design references, examples of good and bad signals, and a curator-written evaluation philosophy.
- **Product strategy context**: market notes, user research excerpts, competitor docs, positioning essays, and a synthesis of what decisions the team is trying to make.
- **Technical standards**: RFCs, official docs, canonical papers, implementation notes, and practitioner critiques.
- **Design system audits**: internal component docs, accessibility standards, platform guidelines, usage examples, and known exceptions.

Liner works especially well when:

- the topic is narrower than a whole field
- the job-to-be-done is explicit
- sources include primary or canonical material
- the curator can explain why each source belongs
- the output will be reused across many AI conversations or tools

## What Liner Is Not Good For

Liner is not the right tool for every context problem.

It is usually not good for:

- **One-off questions** where a quick search is enough.
- **Huge document corpora** where retrieval over thousands of files is the real task.
- **Real-time monitoring** where the source set changes constantly.
- **Private data you cannot safely store in a project folder.**
- **Bypassing paywalls or access controls.** If a source is gated, you need legitimate access and should use `local_file` only for material you are allowed to use.
- **Pure reading lists for humans.** A mixtape is optimized for AI context, not for teaching a human in a pleasant sequence.
- **Automatic truth.** Liner can package sources and expose provenance, but it does not guarantee the sources are correct.
- **Generic topic overviews.** A broad "everything about design" mixtape will usually be shallow and noisy.

The key test is: will a curated context bundle make an AI meaningfully better at a specific job? If not, Liner is probably more ceremony than value.

## How Liner Is Used

There are four common ways to use Liner.

### 1. Agent-Built

Most liners are built by your coding agent. Install the `liner` skill once:

```sh
npm install -g linersh
liner agent-skill install claude --yes    # or codex
```

Then ask your agent for lasting research on a subject. The agent writes your
answer as a job story, runs `liner research start "<job story>" --title "<name>"`,
and follows `liner research next <slug> --json` until it reports `done`.

Liner owns the steps and checks each result. The agent does the research with
its own tools, usually one subagent per step, and Liner makes no AI calls. The
agent stops in chat for you to approve the synthesis, the source list, and the
instructions. It never approves for you.

Agents with no shell can use `liner mcp`. It can start a liner and submit steps,
but it can't approve anything.

### 2. CLI-First

Use the installed CLI when you already have a Project folder or want a
scriptable workflow. Add known Sources through Core, then ask your agent to build
and review the corpus, or compile what you added.

```sh
liner init mobile-design-foundations
liner sources add mobile-design-foundations \
  --type web \
  --url https://developer.apple.com/design/human-interface-guidelines/gestures \
  --note "Apple's canonical gesture reference." \
  --priority required
liner compile mobile-design-foundations
```

The compile step fetches and extracts sources, then writes:

- `mixtape/MIXTAPE.md`
- `mixtape/sources/NN-*.md`

To share:

```sh
liner share mobile-design-foundations
liner import mobile-design-foundations.mixtape
```

### 3. The Navigator

Run `liner` with no arguments to open the navigator, a live view of your library
to keep beside your chat.

```sh
npx linersh
```

It shows every liner, what needs you, your whole source collection, and how
liners connect. You can read any file, follow a citation to its source, and press
`y` to copy a prompt for your agent. The navigator is read-only. Agents do the
work, and the navigator never changes a liner.

CLI subcommands pass through to the bundled Python core:

```sh
liner research next mobile-design-foundations --json
liner compile mobile-design-foundations
liner share mobile-design-foundations
liner project inspect mobile-design-foundations
```

### 4. Agent-Assisted Maintenance

Install the optional Maintenance Adapter when Codex or Claude should inspect and
maintain an existing Project. The adapter is a thin router to the installed
CLI; it is not a second writer and it does not replace the Project Skill.

```sh
liner adapters inspect codex
liner adapters install codex --yes
liner project guidance mobile-design-foundations --format markdown
```

Every supported change starts from a read-only Project Snapshot and a write-free
Project Change Set. Approval-required operations stop on the exact preview.
Apply revalidates that plan and writes a durable, redacted Change Receipt.

## The Curation Lifecycle

`liner research` runs twelve steps in order:

1. `clarify`: Clarify Job.
2. `framing`
3. `candidates`
4. `evaluation`
5. `quality`
6. `synthesis`
7. `approve:synthesis`: you approve the synthesis.
8. `assembly`
9. `approve:sources`: you approve the source list.
10. `build`: compile.
11. `approve:operating_layer`: you approve the instructions.
12. `finish`: write the Operating Layer.

`liner research plan <slug>` lists the steps ahead and the kind of work each
needs. `liner research check <folder> <phase>` checks the artifact of
`candidates`, `evaluation`, `quality`, `synthesis`, or `assembly` without moving
on.

### Clarify Job

The agent asks you a few targeted questions about the job, and you answer in
chat. Liner records the answers. The agent never answers for you.

### Phase 1: Framing

Define the job-to-be-done, capability brief, knowledge map, and required source roles.

The job-to-be-done is not the topic. "Mobile design" is a topic. *"When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic"* is a job. The JTBD is written in Job Story form: `When [circumstance], I want [motivation], so I can [outcome].` See [CURATION.md](CURATION.md) Phase 1 and [JTBD-MASTER-PROMPT.md](JTBD-MASTER-PROMPT.md) for the full rubric.

The capability brief says what the corpus should make a downstream AI agent better at. Required source roles name the evidence types the corpus must satisfy before source discovery can proceed.

### Phase 2: Candidate Discovery

Gather more candidate sources than you expect to keep. This phase is about recall, not precision. The long list ends with a log of the searches that were run.

Good tactics include following citations, mining expert reading lists, searching course syllabi, checking official docs, searching by required source role, and asking AI to critique gaps rather than invent sources.

### Phase 3: Evaluation

Read each candidate and decide keep, trim, or drop. The agent saves pages with
`liner fetch`, so untrusted page text stays in files, not in the chat.

A kept source should have a specific role. The curator note should explain why the source belongs, what part matters, and any limitations.

### Phase 4: Quality Checks

Run tests for redundancy, coverage, disagreement, framing gaps, source-kind balance, and required source-role fit.

This is where the methodology catches "good but narrow" corpora. A wide gap goes
into `working/05-operating-fit-audit.md`, and `liner research improve` can target
it later.

### Phase 5: Synthesis

Write the curator's distilled understanding of the domain.

The synthesis is not a recap. It is the framing the AI should inherit before using the sources.

You review and approve it at `approve:synthesis`.

### Phase 6: Assembly

Draft the source list in `working/07-tape-draft.yaml` and order the sources. You
review and approve it at `approve:sources`. If some sources need JavaScript
rendering and Playwright isn't installed, the review lists them, and the agent
asks you about `liner setup-js --yes` as its own question.

### Build

Add the approved sources to `mixtape/tape.yaml` through Core, fetch every kept
source, extract readable content, write `mixtape/sources/`, and produce
`mixtape/MIXTAPE.md`. This is the Corpus Ready milestone.

### Instructions and Finish

The agent drafts `LINER.md` and `SKILL.md` for review. You approve them at
`approve:operating_layer`. `finish` writes `LINER.md`, `SKILL.md`, and
`liner.yaml`, and the liner is complete.

### Empirical Test

Ask the same AI question with and without the liner. If the answer with the liner is not meaningfully better, the liner is not earning its place.

This is not a research step. Record it with `liner test`. `liner test start`
saves the two answers as unlabeled A and B for you to judge blind, and
`liner test judge` records your choice. `liner test add` records a comparison you
judged with the labels showing. `liner catalog` shows the results.

## How Compilation Works

`liner compile <folder>` reads the project folder and performs a deterministic build.

At a high level:

1. Parse and validate `mixtape/tape.yaml`.
2. Read `mixtape/synthesis.md`.
3. Fetch or load every source.
4. Extract readable content.
5. Use cached content when valid.
6. Write one file per source into `mixtape/sources/`.
7. Write `mixtape/MIXTAPE.md`.

Supported source types:

- `web`
- `youtube`
- `local_file`
- `skill`
- `repository`: the documentation files of a public GitHub repository
- `liner`: another finished liner's synthesis, or one of its sources

Web sources use `httpx` and `trafilatura` for server-rendered extraction. If a page is JavaScript-walled and Playwright support has been set up, Liner can use headless Chromium. You enable that path once:

```sh
liner setup-js
```

During `liner research`, the sources approval lists any sources that need JS rendering, and the agent asks before running `liner setup-js --yes`. The build step won't compile those sources until it's installed. Liner never installs it on its own. Setup downloads only Playwright's headless Chromium: about 100 MB, about 220 MB on disk.

If `trafilatura` itself crashes on a page, the web handlers now keep the source through a tag-stripped HTML fallback where possible and surface a soft warning instead of losing the source to an internal Python exception.

YouTube sources use transcript fetching first and fall back through `yt-dlp` where possible.

Local files live under the project's `local-sources/` directory or the legacy `personal/` directory and can include Markdown, text, HTML, and PDFs.

## How Sharing Works

`liner share <folder>` creates a `.mixtape` archive.

The archive can include:

- the recipe
- synthesis
- source files
- working notes
- local source files, unless excluded

Because compiled source content may include copyrighted or private material, a curator can share just the recipe or exclude local/source content depending on the audience.

`liner import <archive>` unpacks a `.mixtape` and can refetch sources locally.

## How Liner Is Built

Liner has two main implemented surfaces plus supporting documentation and release tooling.

### Python Core

The Python package is the mechanical core.

Important areas:

- `src/liner/cli.py` exposes the `liner` command with Typer.
- `src/liner/maintenance.py` owns Project identity, versioned Snapshots, Change
  Sets, apply validation, Source retention, refresh gates, and Change Receipts.
- `src/liner/agent_adapters.py` manages the optional Codex and Claude adapter
  installations without overwriting user-authored content.
- `src/liner/tape.py` parses and validates `tape.yaml`.
- `src/liner/compile.py` orchestrates compilation.
- `src/liner/handlers/` contains source handlers:
  - `web.py`
  - `web_js.py`
  - `youtube.py`
  - `local_file.py`
  - `skill.py`
  - `repository.py`
  - `liner.py`
- `src/liner/research.py` runs the `liner research` steps, checks, and approvals.
- `src/liner/empirical_tests.py` records `liner test` results.
- `src/liner/output/mixtape.py` renders `MIXTAPE.md` and source files.
- `src/liner/cache.py` manages the local SQLite cache.
- `src/liner/share.py` handles `.mixtape` archives.
- `src/liner/manifest.py` aggregates process metadata from agent runs.

The Python core can still be installed directly for development or fallback testing:

```sh
pipx install linersh
```

### The Navigator

The terminal app is a read-only navigator, a Charm-based Go application in
`packages/go-tui`. `packages/tui` is the npm package shell.

Important areas:

- `packages/go-tui/cmd/liner-tui/` starts the navigator.
- `packages/go-tui/internal/navigator/` owns the library view, the file reader,
  and key handling.
- `packages/go-tui/internal/core/` calls the Python core.

The navigator reads what Core reports through `liner library` and gets agent
prompts from `liner handoff`. It never changes a liner.

### npm Distribution

The npm package is named `linersh`.

It contains a small `bin/liner.js` launcher. Platform-specific binaries, the Python core and the navigator, are distributed as optional npm dependencies:

- `linersh-darwin-arm64`
- `linersh-darwin-x64`
- `linersh-linux-arm64`
- `linersh-linux-x64`
- `linersh-win32-x64`

This follows the native-binary package pattern used by many developer tools: install the lightweight main package, let npm select the platform package that matches the host, then run the bundled binary.

**Windows status:** npm support includes `linersh-win32-x64` as an optional dependency alongside macOS and Linux packages.

The shim behavior:

- `liner` with no arguments opens the navigator.
- CLI subcommands such as `research`, `compile`, `share`, `project`, `sources`,
  `adapters`, `setup-js`, and `mcp` forward to the bundled Python binary.
- `LINER_BIN=/path/to/liner` overrides binary resolution for development/debugging.
- `liner uninstall` removes local Liner state, the npm `_npx` execution cache, and Playwright's Chromium cache for clean reinstall testing. It does not delete mixtape project folders.

### Platform Builds

Platform packages are built with PyInstaller.

Build script:

```sh
python3 scripts/build-platform-package.py
```

Validation:

```sh
python3 scripts/validate-platform-package.py --pack-dry-run
```

Version check:

```sh
uv run python scripts/release-version.py --check
uv run python scripts/release-version.py --check --check-registry
```

`release-version.py` keeps every version file in step with the canonical
`VERSION` file. `--check-registry` also confirms the version isn't already on
npm.

The manual GitHub workflow `.github/workflows/platform-bundles.yml` builds one
package artifact at a time from the public `cmdux-sh/liner` repo. Download each
artifact, then delete it before building the next. Publish the five platform
packages before the main `linersh` package. The full steps are in
`packages/tui/PUBLISH-CHECKLIST.md`.

### The Skill Bundle

The skill bundle ships inside the CLI, in `src/liner/bundled/`:

- `liner/`: the `liner` skill that `liner agent-skill install` adds to Codex or
  Claude Code.
- `research/`: the step prompts `liner research` hands to the agent, and the
  curation methodology in `research/methodology/`.
- `liner-maintenance/`: the optional Maintenance Adapter.

The bundle encodes the curation methodology as instructions for an AI agent. It
is not the same thing as the CLI or the navigator. The agent writes the research
artifacts. The CLI checks them, compiles them, and owns supported post-creation
Project and Source maintenance.

## How Liner Differs From Skills

This is an important distinction.

A **skill** is a set of instructions loaded by an AI assistant at runtime. It tells the assistant how to do a task.

Liner is a **toolchain and artifact format** for creating portable context bundles.

They overlap because Liner includes a skill, but they are not the same kind of thing.

| Question | Skill | Liner |
|---|---|---|
| What is it? | Runtime instructions for an AI assistant | A file format, CLI, navigator, methodology, and packaging workflow |
| Where does it live? | Inside a specific AI tool's skill system | On disk as YAML, Markdown, source files, and archives |
| What does it produce? | Usually task output in the current session | A durable mixtape project folder |
| Is it portable across AI tools? | Usually no, or only with adaptation | Yes, `MIXTAPE.md` and sources are plain files |
| Does it fetch/compile sources? | Not by itself | Yes, through the CLI |
| Does it preserve provenance? | Only if the skill is designed to | Yes, in `tape.yaml`, source files, and `MIXTAPE.md` |
| Can it be versioned in Git? | The skill can; outputs are often ad hoc | Yes, both recipes and mixtapes are plain files |

The relationship is:

- The `liner` skill and the bundled methodology guide your agent through
  `liner research`.
- The Liner CLI checks each step and compiles the mixtape.
- The Operating Layer creates the root Project Skill that teaches future
  sessions how to use that completed Project.
- The optional Maintenance Adapter delegates Codex or Claude maintenance to the
  installed CLI's current guidance.
- A `type: skill` Source stays inert evidence; it is never installed or executed.
- The Mixtape can then be used in any AI tool, with or without these skills.

So a skill is one way to make a mixtape. The mixtape is the portable artifact you keep.

## How Liner Differs From NotebookLM, Claude Projects, and Chat Uploads

Hosted AI tools let you upload documents into a workspace. That is useful, but the context is usually trapped inside that product.

Liner differs in four ways:

1. **Portable recipes.** `tape.yaml` can be shared, reviewed, and recompiled.
2. **Plain files.** Mixtapes are Markdown and YAML, not a database object inside a vendor product.
3. **Curator synthesis.** The curator's framing is a first-class artifact, not an afterthought.
4. **Repeatable builds.** Source fetching and compilation can be rerun locally.

Liner is less polished than a hosted product and asks more of the user. The trade-off is control, portability, and inspectability.

## Example Tape

```yaml
title: Mobile Design Foundations
description: Core touch and gesture references for native mobile UX.
version: 1
curator: cmdux-sh
mode: methodology
jtbd: When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic.

sources:
  - type: web
    url: https://www.nngroup.com/articles/touch-target-size/
    note: Touch target minimums and the research behind the 44pt rule.
    section: foundations

  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/gestures
    note: Apple's canonical gesture reference.
    section: foundations

  - type: local_file
    path: local-sources/touch-target-paper.pdf
    citation: "Parhi et al., 'Target Size for One-Handed Thumb Use,' Mobile HCI 2006."
    note: The canonical empirical study behind the 44pt minimum.
    section: evidence
```

## Example End-to-End Flow

```sh
# Start a project.
liner init mobile-design-foundations

# Add known source material through Core, then ask your agent to review
# Sources, author synthesis, and assemble the corpus.
liner sources add mobile-design-foundations \
  --type web \
  --url https://developer.apple.com/design/human-interface-guidelines/gestures \
  --note "Apple's canonical gesture reference." \
  --priority required

# Enable JS rendering if needed.
liner setup-js

# Compile the mixtape.
liner compile mobile-design-foundations

# Use it in an AI conversation.
# Paste mixtape/MIXTAPE.md and selected files from mixtape/sources/.

# Share it.
liner share mobile-design-foundations
```

Or ask your agent to build the liner with `liner research`, and watch it in the
navigator:

```sh
npx linersh
```

## Design Principles

Liner is guided by a few product and architecture principles.

### Context Should Be Portable

The output should survive outside any one AI product.

### Curation Is Judgment

The value is not in collecting the most sources. The value is in choosing, framing, pruning, and explaining.

### Plain Text Wins

YAML and Markdown make the artifact easy to inspect, diff, copy, and version.
For supported maintenance, Core changes those files through reviewed operations
so plain-text transparency does not come at the expense of identity or receipts.

### Local First

The core runs locally. There is no hosted compilation service, no account system, and no telemetry.

### AI Assistance Is Useful But Not Sovereign

AI can help discover, evaluate, and draft. The methodology still assumes human review because AI is biased toward popular sources, generic notes, and shallow consensus.

## Common Failure Modes

Bad mixtapes usually fall into a few patterns:

- **Bookmark dump**: links without judgment.
- **Greatest hits list**: famous resources the AI likely already knows.
- **Transcript dump**: too many videos, not enough primary/reference material.
- **Unfocused topic**: no concrete job-to-be-done.
- **Noteless sources**: no explanation of why each source belongs.
- **Single-author tribute**: too much of one person's worldview.
- **No disagreement**: contested domains presented as consensus.
- **Synthesis as recap**: a summary of sources rather than a curator's framing.

Liner's methodology exists to catch these before compile.

## Project Status

1.2.0 is the current release, published to npm on 2026-11-07:

- The Python CLI is functional.
- Coding agents build liners through `liner research`, with subagents doing the
  steps. The person approves the synthesis, the source list, and the
  instructions in chat.
- The terminal app is a read-only navigator. The old authoring TUI is gone.
- Framing sets the evidence plan before source discovery. The framing step asks
  for a capability brief and required source roles with why, evidence, and
  minimum coverage, and `liner research submit` won't move on without the brief.
- The npm distribution uses a main `linersh` package plus five optional platform packages (`darwin-arm64`, `darwin-x64`, `linux-arm64`, `linux-x64`, `win32-x64`).
- Published releases install through `npx linersh` on Mac, Linux, and Windows
  x64.
- JavaScript-rendered web support is available through `liner setup-js`. The
  sources approval lists sources that need it, and the agent asks before
  installing.
- `liner uninstall` exists for clean local reinstall tests.
- The skill bundle ships in the CLI and supports Claude Code or Codex.
- Liner Core owns supported Project and Source maintenance through versioned
  Snapshots, Change Sets, exact-plan apply, retention-first removal, refresh
  gates, and durable Change Receipts.
- Maintenance runs through `liner project` and `liner sources`, and optional Codex
  and Claude Maintenance Adapters delegate to the installed CLI's current
  guidance.
- Project Skill, Maintenance Adapter, and inert `type: skill` Sources are
  separate surfaces with different authority.
- `liner mcp` serves liners to agents with no shell.
- `liner test` records whether a liner improved an answer.
- The hosted web builder and public library are planned.

## Where To Read Next

- Root README: `README.md`
- Curation methodology: `docs/curation-skill/CURATION.md`
- Mixtape format: `docs/curation-skill/MIXTAPE-FORMAT.md`
- Curation skill: `src/liner/bundled/research/methodology/SKILL.md`
- npm package and navigator docs: `packages/tui/README.md`
- Platform binary packaging: `packages/tui/PLATFORM-BUNDLES.md`
- Publish checklist: `packages/tui/PUBLISH-CHECKLIST.md`
