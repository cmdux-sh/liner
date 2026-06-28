# Mixtape Format

The technical specification for Liner mixtapes.

> **Status:** This specification describes the **v1 tape format** (with v1.1 additions for `local_file`, `skill`, and `render: js`) and the **v2 Liner project folder shape** now used by new projects. Legacy root-level mixtape folders remain readable.

---

## Overview

A Liner project is a **folder**, not a single file. The root folder contains `LINER.md` and project capabilities. The `mixtape/` subfolder contains the recipe, fetched source content, synthesis, run logs, and compiled `MIXTAPE.md`. Together they form a self-contained artifact ready to share, publish, or paste into an AI conversation.

This document specifies:

- The project folder structure
- The `tape.yaml` recipe format
- The contents of `MIXTAPE.md`, `synthesis.md`, and `sources/*.md`
- The `working/` folder convention
- The `.mixtape` export format
- The versioning policy

---

## The project folder

Every Liner project is a directory. The directory name is a slug — lowercase, hyphenated, filesystem-safe:

```
mobile-design-foundations/
├── liner.yaml            # v2 project marker
├── LINER.md              # project instructions for using the corpus
├── skills/               # reusable project capabilities
├── working/              # audits, impact tests, composition drafts
├── children/             # optional child project references
├── lineage.yaml          # optional composition history
└── mixtape/
    ├── MIXTAPE.md        # the consumable corpus artifact
    ├── tape.yaml         # the recipe
    ├── synthesis.md      # the curator's distilled understanding
    ├── sources/          # extracted content for kept sources
    ├── local-sources/    # curator/local captured files (optional)
    ├── .liner-runs/      # phase run logs
    └── working/          # methodology working files
```

The root folder is the canonical unit. Any operation that reads or writes a Liner project works on the project folder, not on individual files. Corpus-specific readers resolve through `mixtape/`; legacy folders with root-level `tape.yaml` still resolve at the root.

A minimal compiled v2 project contains `liner.yaml`, `mixtape/tape.yaml`, `mixtape/synthesis.md`, `mixtape/MIXTAPE.md`, and `mixtape/sources/` with at least one file. The methodology `mixtape/working/` folder is recommended but not required. `mixtape/local-sources/` exists only when local or captured sources are saved.

---

## `tape.yaml` — the recipe

The recipe lists the kept sources with curator notes, organized by section. It is the input to `liner compile`, which produces `mixtape/MIXTAPE.md` and populates `mixtape/sources/` in v2 projects.

### Required top-level fields

| Field | Type | Description |
|---|---|---|
| `title` | string | Display name of the mixtape |
| `description` | string | One-sentence description of the mixtape's purpose |
| `curator` | string | Name of the curator |
| `version` | integer | Tape format version. Must be `1` for v1 format |
| `mode` | string | `quick` or `methodology` — how this mixtape was curated |
| `sources` | list | Source entries. May be empty while drafting; compile/share require usable sources. |

### Optional top-level fields

| Field | Type | Description |
|---|---|---|
| `tags` | list of strings | Tags for discovery and filtering |
| `created` | ISO date | Creation date |
| `updated` | ISO date | Last meaningful update |
| `license` | string | License identifier (e.g., `CC-BY-4.0`, `MIT`) |
| `homepage` | URL | Link to the mixtape's home (e.g., its GitHub page) |
| `jtbd` | string | The job-to-be-done statement in Job Story form: `When [circumstance], I want [motivation], so I can [outcome].` All three slots required. No solution names (vendor, framework, library, methodology, tool) in any slot. One job per slot — no "and"/comma/slash conjunctions. See [`CURATION.md`](CURATION.md) Phase 1 and [`JTBD-MASTER-PROMPT.md`](JTBD-MASTER-PROMPT.md) for the full rubric. |
| `methodology_version` | string | Which version of CURATION.md was applied |

### Source entries

v1 supports four source types: `youtube`, `web`, `local_file`, and `skill`.

**Required fields per source:**

| Field | Type | Description |
|---|---|---|
| `type` | string | One of `youtube`, `web`, `local_file`, `skill` |
| `url` | URL | Required for `youtube` and `web` source types. Omitted for `local_file` |
| `path` | string | Required for `local_file`; accepted for `skill`. `local_file` paths live under `local-sources/` or legacy `personal/` (e.g., `local-sources/foo.pdf`) |
| `citation` | string | Required for `local_file`. Human-readable provenance: author, title, publication, date |

**Optional fields per source (all types):**

| Field | Type | Description |
|---|---|---|
| `note` | string | Curator note. Strongly recommended; required for library contributions |
| `section` | string | Section label for grouping sources in the compiled mixtape |
| `priority` | string | `required` (default) or `optional`. Optional sources can be excluded with `--skip-optional` |

**Optional fields specific to `web` sources:**

| Field | Type | Description |
|---|---|---|
| `render` | string | Controls how the page is fetched. **Absent (default)** — try server-rendered HTML first; if the page turns out to be JavaScript-only, automatically fall back to headless Chromium (requires `liner setup-js` to have been run). **`js`** — skip the server attempt and go straight to Chromium. **`server`** — server-rendered only; fail loudly on a JS stub. Use `server` for library submissions where you want the recipe to declare no hidden Playwright dependency. |

**Optional fields specific to `local_file` sources:**

| Field | Type | Description |
|---|---|---|
| (none yet) | | The required `path` and `citation` cover v1.1 |

`local_file` sources can reference content the curator has on disk — book chapters, gated articles, exported PDFs, Reader-Mode-saved HTML. The `path` must resolve under the project's `local-sources/` subdirectory or the legacy `personal/` subdirectory; absolute paths and `..` traversal are rejected at validation. Supported extensions are `.md`, `.txt`, `.html`, `.htm`, `.pdf`. The maximum file size is 10MB.

**Fields specific to `skill` sources:**

| Field | Type | Description |
|---|---|---|
| `path` | string | Installed skill name, local skill folder, `SKILL.md` path, or project-relative snapshot |
| `url` | URL | GitHub skill URL. Use instead of `path` |

Skill sources are read as reference artifacts. Liner never installs or executes them.

### Example

```yaml
title: Mobile Design Foundations
description: Core principles for native mobile UX.
curator: Your Name
version: 1
mode: methodology
tags: [mobile, design, ux]
created: 2026-05-18
license: CC-BY-4.0
jtbd: When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic.
methodology_version: "2.0"

sources:
  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/gestures
    note: |
      Apple's canonical gesture reference. Skim the overview;
      the per-gesture detail is what matters.
    section: foundations

  - type: youtube
    url: https://www.youtube.com/watch?v=XXXX
    note: |
      Watch first. The section starting at 14:30 on touch
      target hierarchy is the load-bearing point.
    section: foundations

  - type: web
    url: https://example.com/article-on-mobile-nav
    note: |
      Critique of bottom-tab navigation patterns. Take with
      the next source which disagrees.
    section: patterns

  - type: web
    url: https://example.com/counterargument
    note: Defends bottom tabs for high-frequency apps.
    section: patterns
    priority: optional

  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/accessibility
    render: js
    note: Requires headless-browser rendering; run `liner setup-js` first.
    section: foundations

  - type: local_file
    path: local-sources/touch-target-paper.pdf
    citation: "Parhi et al., 'Target Size for One-Handed Thumb Use on Small Touchscreen Devices,' Mobile HCI 2006."
    note: The canonical empirical study behind Apple's 44pt minimum.
    section: craft

  - type: skill
    path: mobile-design-review
    note: Reference skill for how to structure mobile design critique.
    section: craft
```

### Validation rules

A `tape.yaml` is valid if and only if:

1. The YAML parses
2. All required top-level fields are present
3. `version` is `1` (v1 parsers reject other values)
4. `mode` is `quick` or `methodology`
5. `sources` is a list. It may be empty while drafting; compile/share require usable sources.
6. Every source has a `type` in the allowed set: `youtube`, `web`, `local_file`, `skill`
7. For `youtube` and `web` sources: `url` is present and parses as a valid URL
8. For `web` sources: if `render` is present, it is `server` or `js`
9. For `local_file` sources: `path` and `citation` are present
10. For `local_file` sources: `path` is a relative POSIX path starting with `local-sources/` or legacy `personal/` and contains no `..` segments
11. For `local_file` sources: the extension is one of `.md`, `.txt`, `.html`, `.htm`, `.pdf`
12. For `skill` sources: either `path` or a GitHub `url` is present
13. `youtube` and `web` sources must not declare `path` or `citation`; `local_file` sources must not declare `url`

Path existence and file-size limits are checked at compile time, not validation time — a tape can be valid even when the file isn't on disk yet (e.g., shared recipe before the recipient adds their own copy).

Validation errors should point to the specific field and the specific problem.

---

## `synthesis.md`

The synthesis is the curator's distilled understanding of the domain, expressed in their own voice. It is required — every mixtape has one.

The synthesis covers:

- The principles or framework the curator sees in the domain
- Contested questions and where the curator stands
- Distinctions between concepts that get confused
- When to use what's in the mixtape and when to look elsewhere

Length is typically 800–2000 words. Shorter for narrow topics; longer for broader ones.

In quick mode, the synthesis is AI-drafted during compilation. In methodology mode, the curator writes or substantially edits it. Either way, it lives at `synthesis.md` in the corpus folder (`mixtape/synthesis.md` in v2 projects) and gets included verbatim at the top of `MIXTAPE.md`.

---

## `MIXTAPE.md` — the master file

`MIXTAPE.md` is what the consuming AI actually reads. It is produced by `liner compile` and combines the synthesis with a source index.

Structure:

```markdown
# {title}

> {description}

**Curator:** {curator}
**Compiled:** {ISO timestamp}
**Sources:** {n}
**JTBD:** {jtbd if present}

---

## How to use this mixtape

[Standard boilerplate explaining that this is a curated context
bundle, that the synthesis below provides framing, and that
individual source files are referenced in the index and should
be consulted when specific detail is needed.]

---

## Synthesis

{contents of synthesis.md, verbatim}

---

## Sources

{for each section, in order of first appearance:}

### {section name}

#### Source {N}: {source title or citation}

- **Type:** {youtube|web|local_file}
- **URL:** {source url} *(for youtube/web)*
- **Citation:** {citation} *(for local_file)*
- **Local path:** {path} *(for local_file)*
- **Render:** {server|js} *(for web sources, only when explicitly set)*
- **Author:** {author if known}
- **Published:** {date if known}
- **Curator note:** {note}
- **Content file:** [sources/{nn}-{slug}.md](./sources/{nn}-{slug}.md)
```

The "How to use this mixtape" section is standard boilerplate. It tells the consuming AI that the synthesis carries the curator's framing, that source content lives in referenced files, and that the AI should load sources on demand when conversation requires specific detail.

The source index entries contain enough metadata for the consuming AI to decide whether to load each source's full content. Curator notes are load-bearing for this routing decision — they're the signal that tells the AI which sources matter for which questions.

---

## `sources/`

The `sources/` folder contains extracted text for every kept source, one file per source, named `01-<slug>.md` through `nn-<slug>.md`. Numbers match the source's position in the compiled mixtape; slugs come from the source's title.

Each source file contains:

```markdown
# {source title or citation}

**Source type:** {youtube|web|local_file}
**URL:** {url}             *(for youtube/web)*
**Citation:** {citation}   *(for local_file)*
**Local path:** {path}     *(for local_file)*
**Author:** {author if known}
**Published:** {date if known}
**Fetched:** {ISO timestamp}

{extracted content}
```

For YouTube sources, the extracted content is the transcript, cleaned and joined into paragraphs. For web sources, it's the article text extracted via readability tools, stripped of navigation and boilerplate. For `local_file` sources, the content is extracted by type: `.md`/`.txt` files are read as-is; `.html` is extracted via the same readability pipeline as web sources; `.pdf` is extracted page-by-page via pdfplumber.

Source content is stored at full extracted length. The master file pattern handles in-context efficiency by referencing sources on demand rather than embedding them in `MIXTAPE.md`.

---

## `working/` — the methodology made visible

The `working/` folder contains the curator's research artifacts: the JTBD statement, the knowledge map, the candidate long-list, the evaluation pass with decisions and ratings, and the empirical test result if performed.

Files are numbered to indicate phase order:

```
working/
├── 01-jtbd-and-knowledge-map.md
├── 02-candidate-longlist.md
├── 03-evaluation.yaml
├── 04-quality-checks.md
├── 05-operating-fit-audit.md   # optional improvement recommendation
└── 06-empirical-test.md        # optional methodology validation
```

These are not required by the format but are recommended. Library contributions must include them.

The evaluation file (`03-evaluation.yaml`) is structured — it lists every candidate source with the decision (`kept`, `trim`, or `dropped`), the rationale, and content evidence for every kept or trimmed source. This is distinct from the cache (which holds source *content*); the evaluation file holds *decisions about* sources and the evidence that those decisions came from readable content.

Example `03-evaluation.yaml`:

```yaml
candidates:
  - url: https://example.com/great-article
    decision: kept
    section: foundations
    rationale: Canonical reference for the foundations section.
    rating: 5
    jtbd_fit: direct
    fetch_status: readable
    content_quality: high
    evidence:
      - The article names the core method the corpus teaches.
      - It shows the method working through a concrete example.
    note: >
      Role: Canonical reference. Value: Defines the foundation clearly.
      Limitations: Needs to be paired with applied examples.
  - url: https://example.com/listicle
    decision: dropped
    rationale: Shallow listicle, duplicates content in Source 3.
    rating: 1
  - url: https://example.com/contested
    decision: kept
    section: patterns
    rationale: Strong perspective; paired with Source 5 which disagrees.
    rating: 4
    jtbd_fit: bridge
    fetch_status: partial
    content_quality: medium
    evidence:
      - The available excerpt makes a specific argument about the pattern.
      - The source identifies a tradeoff that can be tested against Source 5.
    note: >
      Role: Contrasting perspective. Value: Prevents a one-note pattern section.
      Limitations: Partial fetch, so only use the claims evidenced by the excerpt.
```

---

## The content cache

Source content fetched during research is stored in the curator's local cache, not in the project folder. The cache lives at `~/.liner/cache.db` (or wherever `platformdirs` puts it on the user's OS).

The cache is shared across all the curator's projects. A source fetched for one mixtape is available for all future mixtapes.

When `liner compile` runs, it copies the cached content for kept sources into `sources/` in the corpus folder. The cache itself is not part of the mixtape and is not included in exports.

See [PLATFORM.md](./PLATFORM.md) for the cache's role in the broader architecture.

---

## The `.mixtape` export format

A `.mixtape` file is a zip archive of a project folder, produced by `liner share` and consumed by `liner import`.

The archive contains the project folder at its root. For v2 projects that includes root project files such as `liner.yaml`, `LINER.md`, and `SKILL.md`, plus `mixtape/tape.yaml`, `mixtape/synthesis.md`, `mixtape/MIXTAPE.md`, `mixtape/sources/`, `mixtape/working/`, and `mixtape/local-sources/` when present. Legacy root-level projects with `tape.yaml`, `MIXTAPE.md`, `sources/`, `working/`, and `personal/` remain supported. The archive does not contain anything from the shared content cache.

On CLI import, the receiving Liner unzips the folder, then refetches any sources whose URLs aren't already in the receiving curator's cache unless `--no-refetch` is passed. Sources already cached are reused. `local_file` sources are read from archived `local-sources/` or legacy `personal/` files directly — no refetch. TUI import extracts the archive into the workspace and leaves refetching to a later compile.

Flags affecting the export:

- `--no-working-notes`: exclude `working/` from the archive
- `--no-personal`: exclude `local-sources/` and legacy `personal/` files from the archive. Used when the curator considers the local copies private (gated content, licensed material) or is preparing a library submission.
- `--no-source-content`: exclude `sources/` from the archive. The recipient must re-fetch on first compile.
- `--minimal`: include only `tape.yaml`. The recipient compiles from scratch — and must supply their own local files for any `local_file` sources.

**Library contributions** must use `--no-personal` *and* must not contain any `local_file` sources in `tape.yaml`. The library accepts only mixtapes built from publicly fetchable sources.

---

## Versioning policy

The mixtape format has its own version number, independent of the CLI/TUI version.

- **v1** is the current format. Stable.
- **v1.1 additions** (additive, non-breaking): the `local_file` and `skill` source types, the `render` field on `web` sources, and the `local-sources/` subdirectory with legacy `personal/` compatibility. v1 parsers that don't know about these will reject tapes that use them — they're not silently forward-compatible. v1.1-aware parsers accept everything older v1 parsers accept.
- Breaking changes increment the format version. v1 mixtapes will continue to parse with v1 tools indefinitely. v2 tools will accept both v1 and v2 mixtapes for some transition period, with a documented migration path.
- Additive, non-breaking changes (new optional fields, new source types) may happen within a major version with a minor version note.

---

## v2 — Planned additions (not yet implemented)

The following source types and fields are designed but not yet implemented. Full specifications will land when the implementations do.

### Additional source types

- `gated_url` — content behind a paywall or login the curator has access to. References the original URL for provenance; requires a `content_file` pointing at a local copy.
- `pasted` — short content entered directly into `tape.yaml`.
- `local_media` — audio or video files. Requires external transcription before use.

### Additional fields

- `content` (for `pasted` sources) — the literal content as a multi-line string.
- `content_file` (for `gated_url`) — path under `local-sources/` to a local file containing the saved content.
- `path` (for `local_media`) — path under `local-sources/` to the media file.

---

## License

This specification is part of the Liner project and is MIT-licensed.
