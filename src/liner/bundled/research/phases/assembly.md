## Phase 7 — Assembly (draft)

You are preparing a *draft* of the sources list for the curator to review. **DO NOT touch tape.yaml directly.** Write your proposal to `working/07-tape-draft.yaml` and stop.

Read these inputs:
- `working/03-evaluation.yaml` — the keep-list with curator notes and ratings.
- `synthesis.md` — the curator's framing, to inform section ordering.
- `tape.yaml` — preserve any existing source order the curator already chose; you're proposing additions/changes, not erasing.
- `local-sources/sources-manifest.yaml`, when present — active custom sources the curator added. These are curator-selected sources, not optional research candidates.

Build the draft as YAML with a top-level `sources:` list. Each entry needs:

```yaml
sources:
  - type: web         # web | youtube | local_file | skill | repository | liner
    url: https://...
    section: foundations
    priority: required   # required | optional
    kind: prescription   # reference | principle | prescription | example
    note: |
      One-paragraph curator note: role of this source, where the value lives,
      and any limitations. Use the three-thing template.
```

Rules:
- Include every candidate marked `kept` or `trim` in 03-evaluation.yaml only when it has `fetch_status: readable|partial`, `content_quality: high|medium`, and at least two `evidence` bullets. If an older evaluation marks a source kept/trim without this evidence, stop and repair `working/03-evaluation.yaml` instead of drafting.
- Drop evaluated URL candidates marked `dropped`, `metadata_only`, `unavailable`, or `content_quality: low`.
- Preserve every existing `local_file` and `skill` source from tape.yaml even though those entries do not appear in the URL-only evaluation file. Carry forward `path`, `url`, `citation`, `note`, `priority`, `section`, and `kind` where present. If any required review metadata is missing, add it without changing the source identity.
- Preserve every `active: true` source from `local-sources/sources-manifest.yaml`, even if it is not present in `tape.yaml` yet. Custom sources are curator-selected. Include active recovered `local_file` entries from `local-sources/recovered/`; do not include inactive original URL entries when a recovered local copy replaced them.
- Do not silently convert `local_file` or `skill` sources into `web` sources. Skill sources are reference material, not active instructions.
- Preserve current tape order first, then append newly kept/trimmed URL candidates that were not already on the tape. Dedupe by URL for web/youtube, by `path`+`citation` for local_file, and by `path` or `url` for skill.
- Within a section, order new evaluated URL candidates by rating (highest first).
- Order sections in reading order: foundations / patterns / applications / craft (or whatever the synthesis suggests).
- Notes come straight from 03-evaluation.yaml's `note` field. Do not paraphrase.
- Mark a source `priority: optional` only when the methodology genuinely treats it as skippable (rare).
- `local_file` and `skill` entries: never invent paths or skill identifiers. Include them only when they already exist in tape.yaml or as active entries in `local-sources/sources-manifest.yaml`.

**Source `kind` is required on every entry.** It tells the synthesis prompt and the consuming AI which sources to weight how. Pick the best fit:

- `reference` — specs / standards cited but not deep-read (POSIX, RFCs, official manual pages). The consuming AI cites these for authority but doesn't quote them at length.
- `principle` — read for stance / framing (essays, manifestos, philosophical pieces). Shapes how the AI thinks about the domain.
- `prescription` — read for concrete rules (style guides, conventions, explicit do/don't lists). The AI quotes these directly when applying rules.
- `example` — read for taste / illustration (case studies, tool surveys, worked examples). Shows what good looks like.

A source can plausibly fit two categories; pick the one closest to *how the consuming AI should USE this source*. When in doubt: rules → `prescription`; arguments → `principle`; examples → `example`; authority without narrative → `reference`.

When done, write `working/07-tape-draft.yaml` and stop. The TUI will show the curator a review screen to accept, edit, or discard. DO NOT modify tape.yaml.
