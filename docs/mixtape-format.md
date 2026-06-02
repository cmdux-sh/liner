# Mixtape Format

This is the current format reference for Liner v0.5.7.

## Terms

**Tape:** the recipe file, `tape.yaml`.

**Mixtape project:** the folder that contains the recipe, synthesis, working notes, and compiled output.

**Compiled mixtape:** `MIXTAPE.md` plus the files in `sources/`. This is the AI-facing artifact.

## Project Folder

```text
my-mixtape/
  tape.yaml
  synthesis.md
  working/
    01-jtbd-and-knowledge-map.md
    02-candidate-longlist.md
    03-evaluation.yaml
    04-quality-checks.md
  personal/          # optional local files
  sources/           # written by compile
  MIXTAPE.md         # written by compile
```

`liner compile <folder>` reads `tape.yaml` and `synthesis.md`, fetches sources, writes source files into `sources/`, and writes `MIXTAPE.md`.

## tape.yaml

Required top-level fields:

| Field | Type | Notes |
| --- | --- | --- |
| `title` | string | Human-readable mixtape title. |
| `description` | string | One-sentence purpose. |
| `version` | integer | Must be `1`. |
| `curator` | string | Person or handle responsible for the mixtape. |
| `sources` | list | Source entries. Fresh projects may be empty while in progress; useful mixtapes need sources. |

Optional top-level fields:

| Field | Type | Notes |
| --- | --- | --- |
| `tags` | list of strings | Search/grouping metadata. |
| `created` | string | ISO date recommended. |
| `updated` | string | ISO date recommended. |
| `license` | string | License for the mixtape project. |
| `homepage` | string | Project or source homepage. |
| `mode` | string | `quick` or `methodology`. Used by the TUI workflow. |
| `jtbd` | string | Job-to-be-done statement. Recommended for agent-assisted phases. |
| `jtbd_clarifications` | list | Question/answer pairs captured by the TUI wizard. |
| `methodology_version` | string | Optional marker for the authoring process used. |
| `parent` | string | Set by `liner replay` when cloning a prior JTBD into a new project. |

## Source Entries

Shared optional fields:

| Field | Type | Notes |
| --- | --- | --- |
| `note` | string | Curator note. Strongly recommended. |
| `section` | string | Groups sources in the compiled source index. |
| `priority` | string | `required` by default; `optional` can be skipped with `--skip-optional`. |
| `kind` | string | Optional role marker: `reference`, `principle`, `prescription`, or `example`. |

### web

```yaml
sources:
  - type: web
    url: https://example.com/article
    note: "Why this source matters and when to use it."
    section: foundations
```

Optional `render` values:

- absent: try server-rendered HTML first, then fall back to browser rendering when needed
- `js`: use browser rendering immediately
- `server`: server-rendered HTML only

Browser rendering requires `liner setup-js`.

### youtube

```yaml
sources:
  - type: youtube
    url: https://www.youtube.com/watch?v=...
    note: "What this transcript contributes."
    section: talks
```

Liner tries transcript extraction first and falls back through the available YouTube tooling. Some videos may still fail because of region locks, age gates, rate limits, removed transcripts, or platform changes.

### local_file

```yaml
sources:
  - type: local_file
    path: personal/report.pdf
    citation: "Author, Report Title, 2025."
    note: "Specific reason this local source belongs."
    section: research
```

Rules:

- `path` must be relative and must start with `personal/`.
- Supported extensions are `.md`, `.txt`, `.html`, `.htm`, and `.pdf`.
- Each local file is capped at 10 MB.
- `citation` is required so the compiled source has provenance.

## Synthesis

`synthesis.md` is the curator's framing of the domain. Liner copies it verbatim into the top of `MIXTAPE.md`.

The TUI surfaces placeholder warnings and blocks the source editor's save-and-compile shortcut when `synthesis.md` is missing or still a placeholder. The CLI requires `synthesis.md` to exist; it does not call an LLM to write it for you.

## Sharing

`liner share <folder>` creates `<folder>.mixtape`, a zip archive of the project folder.

By default the archive includes `tape.yaml`, `synthesis.md`, `working/`, `sources/`, `MIXTAPE.md`, and `personal/` when those folders exist. CLI flags can trim the archive:

| Flag | Effect |
| --- | --- |
| `--no-working-notes` | Exclude `working/`. |
| `--no-source-content` | Exclude `sources/`. |
| `--no-personal` | Exclude `personal/`. |
| `--minimal` | Include only `tape.yaml`. |
| `--out <path>` | Choose the archive path. |

`liner import <archive> [dest]` unpacks the archive and refetches uncached remote sources unless `--no-refetch` is passed. Local files are read from the unpacked `personal/` folder.
