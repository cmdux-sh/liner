# Liner Platform

How the Liner project operates: the surfaces, the lifecycle, the library, and the boundaries.

This document assumes the reader has read [ABOUT.md](./ABOUT.md) (what Liner is) and [CURATION.md](./CURATION.md) (the methodology). It describes Liner's specific implementation choices, not the methodology itself.

---

## The surfaces

Liner is several tools sharing a methodology and a file format. Each surface exists for a specific moment in the workflow.

**CLI** (bundled Python core behind the npm `liner` binary). Atomic operations on Liner projects — `liner init`, `liner compile`, `liner share`, `liner import`, `liner list`, `liner clone`, `liner cache {info,list,show,clear,purge}`, `liner setup-js` (one-shot JavaScript-rendering setup). The right tool for scripting, automation, and one-shot operations.

**Go TUI** (Charm Bubble Tea, launched with `npx linersh`). Interactive multi-step lifecycle runner. Drives the methodology with sensible defaults, calls the bundled Python core for setup and compilation, and invokes the configured AI runner for research phases. The right tool for authoring new Liner projects. Features a projects browser, guided agent phases, structured review gates, optional JS-rendering setup, source-warning recovery paths, import file picker, and progress views for long-running work.

**Skill** (`curating-mixtapes`). A portable methodology document Claude reads when invoked. Drops into `~/.claude/skills/`. Works without the CLI for users who prefer chat-native authoring.

**MCP server** (planned). Will expose Liner's functions as tools any MCP-compatible host can call (Claude Code, Codex, Cursor, others). Local install, stdio transport. The right tool for agent-driven workflows.

**Web builder** (planned). Stateless form on `liner.sh` that produces a `tape.yaml` for download. The right tool for casual users who don't want to install anything.

The implemented surfaces are independent. Planned surfaces should preserve that contract.

---

## The two modes

Most curators won't practice the methodology in detail. They want a tool that produces a usable mixtape with minimal effort. A small minority will engage substantively at each step.

Liner supports both through two modes that share the same architecture and produce the same artifact shape.

**Quick mode** (default). The AI does maximum work. The curator confirms or lightly edits at each gate. Defaults are accept-all. A complete mixtape takes 15–30 minutes. Quality is "good enough for personal use."

**Methodology mode**. The curator engages substantively. Writes their own synthesis if they want. Reviews evaluation rationales. Adds and rejects sources deliberately. A complete mixtape takes several hours. Quality is library-eligible.

The TUI runs quick mode by default with methodology mode available as a slower path. The skill defaults to quick mode and includes instructions for methodology mode. Library contributions require methodology mode.

---

## The lifecycle

Every mixtape moves through the same phases regardless of mode. Quick mode runs through them with AI doing the work and the curator confirming. Methodology mode runs through them with the curator doing substantive work alongside the AI.

1. **Framing** — JTBD statement, capability brief, knowledge map, required source roles
2. **Candidate discovery** — long-list of URLs based on the knowledge map and source roles, no fetching yet
3. **Fetching** — pull content for candidates into the shared cache
4. **Evaluation** — AI reads fetched content, proposes keep/trim/drop with ratings against the JTBD
5. **Quality checks** — redundancy, coverage, disagreement, framing-gap, source-kind, and source-role tests
6. **Synthesis** — AI drafts the synthesis; curator may edit (synthesis is required, not optional)
7. **Final assembly** — `liner compile` writes `mixtape/MIXTAPE.md` and populates `mixtape/sources/` in v2 projects
8. **Empirical test** (optional methodology validation) — compile, paste into AI conversation, compare with/without. This is not a driven TUI phase.

Three natural human-in-the-loop gates: framing confirmation before source search, candidate confirmation before fetching (saves wasted network calls), and evaluation review before synthesis (catches the AI's misjudgments). Quick mode can auto-accept gates while preserving progress state; methodology mode pauses for real review.

The TUI also enforces a Phase 1 artifact contract. Framing is not complete until `working/01-jtbd-and-knowledge-map.md` contains a capability brief and required source roles with why/evidence/minimum coverage. If the agent exits cleanly but misses the contract, the TUI keeps the project on Phase 1 and resumes the agent with the validation issue.

---

## The artifacts

A Liner project is a folder, not a file. The project folder is the canonical unit; the mixtape corpus lives under `mixtape/` in v2 projects and everything else is derived from it.

```
mobile-design-foundations/
├── liner.yaml            # v2 project marker and metadata
├── LINER.md              # operating instructions for future agents
├── SKILL.md              # optional project skill entry point
├── skills/               # reusable project capabilities
├── working/              # audits, impact tests, composition drafts
└── mixtape/
    ├── MIXTAPE.md        # the consumable corpus artifact
    ├── tape.yaml         # the recipe — kept sources, sections, notes
    ├── synthesis.md      # the curator's distilled understanding
    ├── sources/          # extracted content for kept sources
    ├── local-sources/    # curator/local captured files (optional)
    ├── .liner-runs/      # phase run logs
    └── working/          # methodology working files
```

Three concepts to distinguish:

- A **tape file** (`mixtape/tape.yaml` in v2 projects) is the recipe. References sources by URL or path. Shareable on its own; anyone with the file can recompile from scratch.
- A **mixtape** is the corpus inside the project folder. In v2 projects it lives under `mixtape/`; legacy root-level mixtape folders remain readable.
- The **consumable mixtape** is `mixtape/MIXTAPE.md` plus the `mixtape/sources/*.md` files it references. This is what gets pasted into an AI conversation.

[MIXTAPE-FORMAT.md](./MIXTAPE-FORMAT.md) specifies the structure in detail.

---

## The content cache

The cache is a shared, curator-level resource. One cache per machine, used across all projects.

Default location: `~/.liner/cache.db` (or wherever `platformdirs` puts it on the user's OS).

The cache holds extracted source content keyed by canonical URL. The same source used in three different mixtapes is fetched once. Sources that change over time (transcripts, articles) have per-type TTLs; permanent references (specs, papers) can be pinned.

The cache is the curator's local research library. Treat it that way. `liner cache list` shows what's there. `liner cache show <url>` displays a cached source. `liner cache purge <url>` removes one.

The cache is never included in `.mixtape` exports. When a recipient imports a `.mixtape`, the receiving Liner repopulates *their* cache by fetching sources fresh — reusing whatever they already have.

---

## The library (planned)

The Liner library is planned as a curated collection of public mixtapes at [liner.sh/library](https://liner.sh/library), backed by a Git repository.

**What the library contains.** Methodology-mode mixtapes authored by Liner maintainers and invited guest curators. Every mixtape uses only sources anyone can fetch. The folders are published in full — recipe, synthesis, working notes, sources, the compiled `MIXTAPE.md`. Library readers can browse the compiled artifact directly without compiling anything themselves.

**What the library does not contain.** Quick-mode tapes. Tapes containing `local_file` sources (which by definition reference content the maintainers can't republish). Tapes with `render: js` that depend on opt-in dependencies. Tapes that haven't passed the empirical test.

**Structure.** Each mixtape will be a directory in the repo. The website will render these as individual pages. Users will be able to clone the tape file and edit it, browse the compiled mixtape, or read the working notes.

**Curation philosophy.** Small, slow-growing, opinionated. Quality over coverage. A library of fifty excellent mixtapes is more useful than five hundred mediocre ones.

---

## Contributing

The planned library will accept pull requests for methodology-mode mixtapes only.

1. Build the mixtape in methodology mode, applying [CURATION.md](./CURATION.md) substantively
2. Run the empirical test — compile the mixtape, compare AI output with and without it on a real question. Document the result in `mixtape/working/06-empirical-test.md`. This is methodology guidance, not a driven TUI phase.
3. Submit a PR with the full project folder — exported via `liner share --no-personal`, and with no `local_file` sources in `mixtape/tape.yaml`
4. A maintainer reviews. Submissions can be accepted, returned with feedback, or declined

The bar is "demonstrably improves AI output for the stated job-to-be-done." Effort alone isn't sufficient. A well-meaning mixtape that doesn't move the needle isn't a library mixtape.

Accepted mixtapes are credited to the curator by name. Curators retain authorship. Library mixtapes are MIT-licensed by default unless otherwise specified.

---

## Sharing

Mixtapes are folders. To share a mixtape as a single file, use `liner share <project-folder>`. This produces `<project-folder>.mixtape` — a zip containing the project folder.

The exported `.mixtape` includes everything in the project folder: root project files such as `liner.yaml`, `LINER.md`, and `SKILL.md`, plus `mixtape/tape.yaml`, `mixtape/synthesis.md`, `mixtape/MIXTAPE.md`, `mixtape/sources/`, `mixtape/working/`, and `mixtape/local-sources/` when present. Legacy root-level `sources/`, `working/`, and `personal/` folders are still supported. The archive does not include anything from the shared content cache.

On `liner import <file.mixtape>`, the receiving Liner unzips the folder, then refetches any sources that aren't already in the receiving curator's cache unless import is run with `--no-refetch`. Sources already cached are reused. `local_file` sources read from archived `local-sources/` or legacy `personal/` files directly — no refetch. The TUI import flow extracts the archive and leaves refetching to a later compile.

**Flags worth knowing about:**

- `--no-working-notes`: exclude `mixtape/working/` and legacy root-level `working/` from the export. For curators who consider their working notes private. Default is to include them.
- `--no-personal`: exclude `local-sources/` and legacy `personal/` files from the export. Used when the local copies are licensed/gated material the curator can't redistribute, or when preparing a library submission. The recipient won't be able to compile `local_file` sources without supplying their own copies.
- `--no-source-content`: skip `mixtape/sources/` and legacy root-level `sources/`. The recipient must re-fetch on first compile.
- `--minimal`: produce a smaller zip containing only `mixtape/tape.yaml` in v2 projects. For sharing the recipe only. The recipient compiles fresh.

---

## What Liner doesn't do

Boundaries that shape the project's scope:

- **No hosted compilation.** All compilation happens on the curator's machine. Liner does not run a server that fetches sources for users.
- **No accounts in v1.** The CLI, TUI, planned library, and website all work without authentication. A future community index will use GitHub OAuth only.
- **No paywall bypass.** Liner does not help users access content they don't already have. Source handlers for gated content require the user to provide content from their existing access.
- **No LLM calls in the core CLI.** The CLI does not call AI models. The TUI invokes the user's configured local runner for methodology phases. All AI work happens through the user's own subscription or local toolchain. Liner doesn't pay for inference.
- **No headless browser by default.** Plain `npx linersh` works without downloading Chromium. JavaScript-rendered sites (Apple HIG, Notion, many SPAs) require the opt-in `liner setup-js` step, which installs Playwright browser support locally. There is no hosted rendering service.
- **No telemetry.** The tools don't phone home. Usage is the user's own business.
- **No curator-required compression decisions.** Sources are stored at full extracted length. In-context efficiency comes from the master file pattern — `MIXTAPE.md` is small and always-loaded, sources are referenced and loaded on demand by the consuming AI.

These boundaries are stable. Anything outside them either lives in a separate project or doesn't get built.

---

## The community index (planned, deferred)

A future addition will be a community index — a directory of community-published mixtapes hosted by their curators on GitHub. Liner does not host the content; it indexes links to user repositories.

The design:

- Curators sign in with GitHub OAuth and submit a link to a public GitHub repository containing their mixtape folder
- The repository must be on GitHub (no other hosts in v1)
- Entries are reviewed before featuring; the front page is editorially curated
- No comments, ratings, or community voting in v1
- A passive popularity signal may be shown but does not drive default sorting

This feature is not part of the initial Liner release. It will be built only if there's evidence of demand from real users of the v1 tools and planned library.

---

## Maintenance contract

Liner is a side project. The maintenance contract:

- Issues and PRs are reviewed when time permits, not on any schedule
- No SLA, no guaranteed compatibility, no support commitments
- Breaking changes are possible between versions; versioning will be honest about them
- The transcript-fetching and content-extraction layers depend on third-party platforms that change over time; the tools will sometimes break and will be fixed when fixes are practical
- The planned library will be curated by the maintainer and invited guest curators; submissions are welcomed but not guaranteed acceptance or response

Open source does not mean free maintenance. Users who need guaranteed support should use a hosted product.

---

## Versioning

Liner versions three things independently:

- **Tape format** has its own version field. Breaking format changes increment the format version; older tapes either continue to work or get a clear migration path. See [MIXTAPE-FORMAT.md](./MIXTAPE-FORMAT.md) for the versioning policy.
- **Methodology** is versioned in [CURATION.md](./CURATION.md). Substantive changes increment the methodology version.
- **Tools** (CLI, Go TUI, skill, and planned MCP server) follow semver. Major versions can break compatibility; minor and patch versions don't.

The methodology, the format, and the tools are versioned independently. Updates to one don't force updates to the others unless there's a real dependency.

---

## License

MIT.
