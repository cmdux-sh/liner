# Liner Platform

How the Liner project operates: the surfaces, the lifecycle, the library, and the boundaries.

This document assumes the reader has read [ABOUT.md](./ABOUT.md) (what Liner is) and [CURATION.md](./CURATION.md) (the methodology). It describes Liner's specific implementation choices, not the methodology itself.

---

## The surfaces

Liner is several tools sharing a methodology and a file format. Each surface exists for a specific moment in the workflow.

**CLI** (bundled Python core behind the npm `liner` binary). Building,
compilation, sharing, search, inspection, and maintenance operations. `liner
research` runs the steps your coding agent follows to build a liner. The full
list is the command table in the [README](../../README.md#commands). The right
tool for agents, scripting, and one-shot operations.

**Go TUI** (Charm Bubble Tea, opened by running `liner` with no arguments). A
read-only navigator for your library. It shows every liner, what needs you,
your sources, and how liners connect, and it copies prompts for your agent. It
never changes a liner.

**`liner` skill** (installed with `liner agent-skill install claude` or
`codex`). Lets the agent find your liners, suggest one, and build a new one
through `liner research`.

**Curation methodology** (`curating-mixtapes`, in
`src/liner/bundled/research/methodology/`). The methodology the agent reads at
each `liner research` step. It produces corpus inputs for compile; it does not
maintain an existing Project.

**Project Skill** (root `SKILL.md`). The canonical entrypoint created by the
Operating Layer so future AI sessions can load one completed Project.

**Maintenance Adapter** (optional Codex or Claude installation). A thin router
to the installed CLI's current `liner project guidance`. It is explicit,
managed, removable, and never a second writer. A `type: skill` Source is a
fourth, inert use of skill text as evidence and is never installed.

**MCP server** (`liner mcp`). Serves your liners as tools to any MCP host, for agents with no shell. Local, stdio transport. It can't approve anything.

**Web builder** (planned). Stateless form on `liner.sh` that produces a `tape.yaml` for download. The right tool for casual users who don't want to install anything.

The implemented surfaces are independent. Planned surfaces should preserve that contract.

---

## The two modes

Most curators won't practice the methodology in detail. They want a tool that produces a usable mixtape with minimal effort. A small minority will engage substantively at each step.

Liner supports both through two modes that share the same architecture and produce the same artifact shape.

**Quick mode** (the default outside `liner research`). The AI does maximum work. The curator confirms or lightly edits at each gate. Defaults are accept-all. A complete mixtape takes 15–30 minutes. Quality is "good enough for personal use."

**Methodology mode**. The curator engages substantively. Writes their own synthesis if they want. Reviews evaluation rationales. Adds and rejects sources deliberately. A complete mixtape takes several hours. Quality is library-eligible.

`liner research` has no mode setting. `liner research start` labels the tape `mode: methodology`, but the `mode` field is only a label and changes nothing. Inside `liner research`, each step's prompt and the three approvals replace both modes' gates. Outside it, the methodology defaults to quick mode and includes instructions for methodology mode. Library contributions require methodology mode.

---

## The lifecycle

Every liner moves through the same `liner research` steps regardless of mode. The agent does the work for each step, often in a subagent, and runs `liner research submit` when it's done.

1. **Clarify** — the person answers a few questions that sharpen the job
2. **Framing** — capability brief, knowledge map, required source roles
3. **Candidates** — long-list of sources based on the knowledge map and source roles, ending in a search log
4. **Evaluation** — the agent reads pages with `liner fetch` and proposes keep/trim/drop with ratings against the JTBD
5. **Quality** — redundancy, coverage, disagreement, framing-gap, source-kind, and source-role tests
6. **Synthesis** — the agent drafts the synthesis (required, not optional)
7. **Approve synthesis** — the person reviews and approves it
8. **Assembly** — the agent drafts the source list
9. **Approve sources** — the person reviews and approves it
10. **Build** — Liner adds the sources and compiles `mixtape/MIXTAPE.md` and `mixtape/sources/`
11. **Approve instructions** — the person reviews the drafts of `LINER.md` and the Project Skill
12. **Finish** — Liner writes `LINER.md`, root `SKILL.md`, and `liner.yaml`

The empirical test is separate. `liner test` records whether a liner improved an answer.

Three approvals keep the person in charge: the synthesis, the sources, and the instructions. The agent asks for each one in chat and never approves for the person. An approval covers the exact content reviewed; if it changes, it has to be reviewed again.

Liner Core checks each step's files when the agent runs `liner research submit`, and `liner research check` runs the same check without moving on. Framing is not complete until `working/01-jtbd-and-knowledge-map.md` contains a capability brief and required source roles with why/evidence/minimum coverage. Until it does, submit refuses the step and says what is missing.

After initial creation, supported maintenance follows a separate safety
contract:

1. Inspect a versioned, read-only Project Snapshot.
2. Produce a write-free Project Change Set with exact file effects, lifecycle
   consequences, risk, and approval requirements.
3. Apply only that exact plan after Project identity, revision, compatibility,
   content, paths, and required curator approval are revalidated.
4. Persist a durable, redacted Change Receipt or return structured remediation.

Project and Source IDs are immutable. Remove retains Source content in the
Retention Vault; purge is a separate destructive operation. Source changes
invalidate dependent corpus and Operating Layer state, which must move through
the implemented refresh review gates.

---

## The artifacts

A Liner project is a folder, not a file. The project folder is the canonical unit; the mixtape corpus lives under `mixtape/` in v2 projects and everything else is derived from it.

```
mobile-design-foundations/
├── liner.yaml            # v2 project marker and metadata
├── LINER.md              # operating instructions for future agents
├── SKILL.md              # default Project Skill entrypoint
└── mixtape/
    ├── MIXTAPE.md        # the consumable corpus artifact
    ├── tape.yaml         # the recipe — kept sources, sections, notes
    ├── synthesis.md      # the curator's distilled understanding
    ├── sources/          # extracted content for kept sources
    ├── local-sources/    # curator/local captured files (optional)
    ├── .liner-runs/      # phase run logs
    └── working/          # methodology working files
```

Older projects may also have a root `skills/` folder or a root `working/` folder. Liner still reads them, but new liners don't write them.

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
2. Run the empirical test — compile the mixtape, compare AI output with and without it on a real question, and record the result with `liner test start` and `liner test judge` for a blind comparison, or `liner test add`. This is methodology guidance, not a `liner research` step.
3. Submit a PR with the full project folder — exported via `liner share --no-personal`, and with no `local_file` sources in `mixtape/tape.yaml`
4. A maintainer reviews. Submissions can be accepted, returned with feedback, or declined

The bar is "demonstrably improves AI output for the stated job-to-be-done." Effort alone isn't sufficient. A well-meaning mixtape that doesn't move the needle isn't a library mixtape.

Accepted mixtapes are credited to the curator by name. Curators retain authorship. Library mixtapes are MIT-licensed by default unless otherwise specified.

---

## Sharing

Mixtapes are folders. To share a mixtape as a single file, use `liner share <project-folder>`. This produces `<project-folder>.mixtape` — a zip containing the project folder.

The exported `.mixtape` includes everything in the project folder: root project files such as `liner.yaml`, `LINER.md`, and `SKILL.md`, plus `mixtape/tape.yaml`, `mixtape/synthesis.md`, `mixtape/MIXTAPE.md`, `mixtape/sources/`, `mixtape/working/`, and `mixtape/local-sources/` when present. Legacy root-level `sources/`, `working/`, and `personal/` folders are still supported. The archive does not include anything from the shared content cache.

On `liner import <file.mixtape>`, the receiving Liner unzips the folder, then refetches any sources that aren't already in the receiving curator's cache unless import is run with `--no-refetch`. Sources already cached are reused. `local_file` sources read from archived `local-sources/` or legacy `personal/` files directly — no refetch.

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
- **No LLM calls in the core CLI.** The CLI does not call AI models. The user's own coding agent does the research with its own tools. All AI work happens through the user's own subscription or local toolchain. Liner doesn't pay for inference.
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
- **Tools** (CLI, Go TUI navigator, skills, and MCP server) follow semver. Major versions can break compatibility; minor and patch versions don't.

The methodology, the format, and the tools are versioned independently. Updates to one don't force updates to the others unless there's a real dependency.

---

## License

MIT.
