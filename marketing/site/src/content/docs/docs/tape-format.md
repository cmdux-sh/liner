---
title: Tape Format
description: The required and optional fields in a Liner tape.yaml file.
---

`tape.yaml` is the source recipe for the Mixtape corpus inside a project.

This page is a format reference. Use the terminal UI or `liner sources` and
`liner project` commands for supported post-creation maintenance so immutable
IDs, lineage, lifecycle invalidation, approvals, and Change Receipts remain
intact. An optional Maintenance Adapter delegates to those commands; it does not
edit the recipe directly.

## Required Fields

```yaml
title: Mobile Design Foundations
description: Core touch and gesture references for native mobile UX.
version: 1
curator: cmdux-sh
mode: quick
jtbd: When I am designing native mobile interactions, I want grounded touch guidance, so I can avoid brittle gesture patterns.

sources:
  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/gestures
    note: Apple's canonical gesture reference.
    section: foundations
    kind: reference
```

Required top-level fields:

- `title`
- `description`
- `version: 1`
- `curator`
- `sources`

`sources` can be empty while you are drafting a tape. The terminal UI requires at least one source before compiling.

## Optional Top-Level Fields

- `mode`, either `quick` or `methodology`
- `tags`
- `created`
- `updated`
- `license`
- `homepage`
- `jtbd`
- `jtbd_clarifications`
- `methodology_version`
- `parent`

The `jtbd` field uses Job Story form:

```text
When [circumstance], I want [motivation], so I can [outcome].
```

## Source Types

The tape format supports four source types.

### Web

```yaml
- type: web
  url: https://example.com/article
  note: Why this source belongs.
  section: foundations
  priority: required
  kind: reference
```

Optional `render` values:

- absent: try server HTML first, then use headless Chromium if needed
- `js`: use Chromium immediately
- `server`: use server-rendered HTML only

### YouTube

```yaml
- type: youtube
  url: https://www.youtube.com/watch?v=XXXX
  note: Watch for the section on source evaluation.
  section: examples
```

The fetcher tries transcript fetching first and falls back through `yt-dlp` where possible.

### Local File

```yaml
- type: local_file
  path: local-sources/touch-target-paper.pdf
  citation: "Parhi et al., Target Size for One-Handed Thumb Use, Mobile HCI 2006."
  note: Canonical empirical study behind small-screen touch targets.
  section: research
```

Local files must live under the project's `local-sources/` folder or the legacy `personal/` folder. Supported formats are Markdown, text, HTML, and PDF, up to 10 MB per file.

### Skill

```yaml
- type: skill
  path: terminal-ui
  note: Extract terminal interaction guidance as reference material.
  section: craft
```

Skill sources can use an installed skill name, a local skill folder or `SKILL.md` path, a project snapshot under `local-sources/skills/`, or a GitHub skill URL. Skill files are read as reference material and are not installed or executed.

## Source Fields

Common optional fields:

- `note`: curator guidance about why the source belongs
- `section`: source grouping used by compile filters
- `priority`: `required` or `optional`; defaults to `required`
- `kind`: `reference`, `principle`, `prescription`, or `example`

`local_file` sources require `path` and `citation`. Web and YouTube sources require `url`. Skill sources require either `path` or a GitHub `url`.
