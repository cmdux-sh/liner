---
title: Tape Format
description: Reference for tape.yaml metadata, source types, local-file requirements, and source fields.
---


`mixtape/tape.yaml` is the recipe for a Project's research packet. It lists the sources and the metadata used during compile.

This page is a format reference. Use the terminal UI or `liner sources` and
`liner project` commands for supported post-creation maintenance so immutable
IDs, lineage, lifecycle invalidation, approvals, and Change Receipts remain
intact. An optional Maintenance Adapter delegates to those commands; it does not
edit the recipe directly.

## Required fields

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

The following top-level fields are required.

- `title`
- `description`
- `version: 1`
- `curator`
- `sources`

`sources` can be empty while you are drafting a tape. The terminal UI requires at least one source before compiling.

## Optional top-level fields

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

The `jtbd` field records the task using Job Story form.

```text
When [circumstance], I want [motivation], so I can [outcome].
```

## Source types

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

The optional `render` field controls web fetching.

- With no `render` field, try server HTML first and use headless Chromium if needed.
- Set `render: js` to use Chromium immediately.
- Set `render: server` to use server-rendered HTML only.

### YouTube

```yaml
- type: youtube
  url: https://www.youtube.com/watch?v=XXXX
  note: Watch for the section on source evaluation.
  section: examples
```

The fetcher tries transcript fetching first and falls back through `yt-dlp` where possible.

### Local file

```yaml
- type: local_file
  path: local-sources/touch-target-paper.pdf
  citation: "Parhi et al., Target Size for One-Handed Thumb Use, Mobile HCI 2006."
  note: Canonical empirical study behind small-screen touch targets.
  section: research
```

In current Projects, put files under `mixtape/local-sources/`. The recipe uses a path relative to `mixtape/`, such as `path: local-sources/touch-target-paper.pdf`. Legacy root-level Projects resolve that recipe path from their root and also support `personal/`. Supported formats are Markdown, text, HTML, and PDF, up to 10 MiB per file.

### Skill

```yaml
- type: skill
  path: terminal-ui
  note: Extract terminal interaction guidance as reference material.
  section: craft
```

Skill sources can use an installed skill name, a local skill folder or `SKILL.md` path, a project snapshot under `local-sources/skills/`, or a GitHub skill URL. Skill files are read as reference material and are not installed or executed.

## Source fields

These optional fields apply across source types.

- `note` explains why the source belongs.
- `section` groups sources for compile filters.
- `priority` accepts `required` or `optional` and defaults to `required`.
- `kind` accepts `reference`, `principle`, `prescription`, or `example`.

`local_file` sources require `path` and `citation`. Web and YouTube sources require `url`. Skill sources require either `path` or a GitHub `url`.
