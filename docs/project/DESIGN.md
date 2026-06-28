---
name: Liner Core Product
description: Terminal-native product system for creating and managing source-grounded AI context projects.
colors:
  ink: "#F7F7F2"
  muted: "#8B8C89"
  soft-muted: "#C3C5CC"
  accent-orange: "#FF5A1F"
  next-orange: "#FF9A66"
  panel: "#30323D"
  success: "#7ACB7A"
  warning: "#F2C94C"
  error: "#FF5C5C"
  teal: "#2EE6BF"
typography:
  title:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "0"
  body:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0"
  label:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"
    fontWeight: 400
    lineHeight: 1.3
    letterSpacing: "0"
rounded:
  none: "0"
spacing:
  row-gap: "1 line"
  detail-gap: "1 blank line"
  table-cell-gap: "2 columns"
  chrome-min-width: "60 columns"
  chrome-max-width: "118 columns"
components:
  command-selected:
    textColor: "{colors.accent-orange}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
  detail-label:
    textColor: "{colors.muted}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
  detail-value:
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
  table-selected-focused:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
  footer-help:
    textColor: "{colors.muted}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
---

# Design System: Liner Core Product

## 1. Overview

**Creative North Star: "The Local Workbench"**

Liner's core product design is a restrained terminal workbench for creating and
maintaining source-grounded projects. It is not a decorative shell around CLI
commands. It is the place where a user starts a project, opens a project
list, manages one project, imports another project, reviews generated files, and
checks the evidence that makes the project trustworthy.

The system should feel calm, dense, explicit, and durable. Calm enough to read in
a terminal. Dense enough to manage real project state. Explicit enough that
writes are never ambiguous. Durable enough that future agents can add screens by
choosing a named pattern instead of re-litigating the information architecture.

It rejects marketing-page composition, dashboard card grids, duplicate command
surfaces, persona-heavy copy, and decorative terminal effects. The visual system
is built from text hierarchy, stable columns, footer-owned navigation, semantic
color, and plain tables.

**Key Characteristics:**

- One global frame: top banner, task body, footer help, optional activity row.
- Orange marks selection, action, and primary state.
- Gray labels and descriptions stay secondary; white values carry content.
- Bold is reserved for titles.
- Tables, label/value blocks, and viewports carry structure before boxes do.
- Every reusable screen shape has a named job.

## 2. Colors

The palette is terminal-restrained: light ink, muted grays, one strong orange
action color, a softer orange for `Next`, and semantic status colors.

### Primary

- **Command Orange**: primary selection, primary action, filter matches, cursor
  accents, progress fill, and important activity prompts. Use it sparingly so it
  keeps meaning.
- **Next Orange**: softer guidance color for `Next`. It should read as
  recommendation or current-action cue, not as the same state as selection.

### Secondary

- **Soft Muted Gray**: secondary section color, dim filesystem rows, and
  non-primary structural text.
- **Panel Slate**: focused table selection background, banner slash texture, and
  table border color.

### Tertiary

- **Signal Teal**: rare supporting signal for future progress or source-state
  moments. Do not use it as a second brand accent.
- **Success Green**, **Warning Yellow**, and **Error Red**: semantic feedback
  only.

### Neutral

- **Liner Ink**: primary readable content, values, focused table cells, and brand
  text.
- **Muted Gray**: labels, descriptions, footer guidance, subtitles, activity
  text, and placeholder text.

### Named Rules

**The Orange Means Action Rule.** Orange is for the selected thing, primary
action, progress, filter match, or current action cue. Do not use it as
decoration.

**The Gray Holds Context Rule.** Descriptions, labels, footer help, and selected
descriptions stay gray. Selection should not repaint context as action.

**The No Purple Rule.** Purple is not a Liner action color. Do not let Bubble's
default purple leak into Liner-owned lists or controls.

## 3. Typography

**Display Font:** The user's terminal monospace
**Body Font:** The user's terminal monospace
**Label/Mono Font:** The user's terminal monospace

**Character:** Liner relies on terminal-native typography. The hierarchy comes
from weight, spacing, color, and alignment rather than font pairing or large
display type.

### Hierarchy

- **Title** (bold, terminal size, tight line height): screen titles, project
  names, and document/viewer titles. This is the only routine bold style.
- **Body** (regular, terminal size): command titles, table cells, detail values,
  file names, and readable content.
- **Description** (regular, muted): subtitles, selected command descriptions,
  section descriptions, footer guidance, and supporting context.
- **Label** (regular, muted): label/value row labels such as `Status`, `Name`,
  `Folder`, `Mode`, and `Destination`.
- **Table Header** (regular, muted, bottom border): dense comparison headers.
  Headers align to the same left positions as row content.

### Named Rules

**The Titles Only Bold Rule.** Bold is for titles. Descriptions, selected
navigation items, table cells, table headers, footer guidance, and `Next` copy
are not bold.

**The Product Noun Rule.** UI copy uses exact product nouns: `Status`, `Name`,
`Settings`, `Operating Layer`, `Import Project`, `Open MIXTAPE.md`, `LINER.md`,
`Project Skill`, and `Audit Output`.

## 4. Elevation

The terminal product has no traditional elevation. Depth comes from frame
position, blank lines, column structure, table borders, focused row background,
and muted versus primary text. Boxes are rare. Nested cards are prohibited.

### Shadow Vocabulary

- **No Resting Shadow** (`none`): terminal surfaces do not use shadows.
- **Focused Row Fill** (`#30323D`): the only routine filled state, used for
  focused table selection.

### Named Rules

**The Structure Before Box Rule.** Use alignment, spacing, table columns,
label/value rows, and viewports before adding borders or containers.

## 5. Components

### Global Frame

- **Shape:** one terminal frame with top banner, body, footer, and optional
  activity row.
- **Top banner:** owns brand, slash texture, and one location label.
- **Body:** owns the current screen's task only.
- **Footer:** owns navigation and key hints.
- **Activity row:** appears only when there is a real next action.

### Command List

- **Shape:** selectable/filterable list.
- **Selected title:** Command Orange.
- **Selected description:** Muted Gray.
- **Filter match:** Command Orange with underline.
- **Rules:** no separate command page, no `:` Commands surface, no extra in-body
  command header.

### Label/Value Block

- **Shape:** compact rows with muted labels and ink values.
- **Usage:** selected project detail, Settings details, Import Project facts.
- **Wrapping:** long values wrap under the value column, never into the label
  column or adjacent pane.
- **Rules:** do not add `Field`/`Value` headers to this pattern.

### Data Table

- **Shape:** muted regular headers, ink cells, two-column padding between cells,
  optional focused-row fill.
- **Usage:** section details, audit lists, composition,
  source inventories, and write-aware action tables.
- **Rules:** use one table per section detail. If a second table has a different
  purpose, promote it to a separate section.

### Section Workspace

- **Shape:** opened artifact identity at top, names-only section list on the
  left, selected section detail on the right.
- **Usage:** Project and future opened-object workspaces.
- **Section list:** names only. No mini-statuses beside section names.
- **Detail rhythm:** one title, one muted description, one blank line, one
  table.

### Split Browser

- **Shape:** collection list on the left, compact selected-item detail on the
  right.
- **Usage:** Projects and other libraries where the user chooses an item before
  opening it.
- **Rules:** default detail stays to identity and decision context. Deeper
  capability counts belong in the opened workspace.

### File Picker

- **Shape:** title, muted instruction, label/value facts, picker viewport.
- **Usage:** Import Project and future filesystem selection flows.
- **Rules:** filter or disable invalid file types. Keep path typing out of the
  primary flow unless a deliberate fallback mode is designed.

### Review Surface

- **Shape:** read-only preview or table plus explicit accept/open/discard/back
  controls in footer or action table.
- **Usage:** source cleanup, import checks, and future maintenance reviews.
- **Rules:** generated fixes are proposed before they are written. Apply writes,
  discard removes drafts, and review output explains decisions.

## 6. Do's and Don'ts

### Do:

- **Do** keep Home as the command hub and Projects as the project browser.
- **Do** keep Project as a section workspace for one opened project.
- **Do** use footer/help as the only navigation and key-hint surface.
- **Do** use Command Orange only for selection, action, progress, and filter
  matches.
- **Do** show write behavior before writes happen.
- **Do** keep `MIXTAPE.md` readiness separate from full Liner project readiness.
- **Do** use current evidence from files, command output, and audits when showing
  status.
- **Do** document unresolved capability sequencing instead of hard-coding it into
  UI promises.

### Don't:

- **Don't** frame Liner as an `advisor`, chatbot, knowledge bot, or agent
  persona.
- **Don't** reintroduce duplicate body shortcut strips, `:` Commands, body help
  tables, or second location rows.
- **Don't** use `Import archive` or other implementation nouns when the user is
  acting on a Liner project.
- **Don't** use Bubble's purple defaults for Liner-owned action or selection
  states.
- **Don't** make selected descriptions, table cells, headers, footer guidance, or
  `Next` copy bold.
- **Don't** use card grids, nested cards, decorative boxes, or marketing hero
  composition for product work screens.
- **Don't** imply a project is complete only because `MIXTAPE.md` exists.
- **Don't** let previews, read-only views, or render paths create files or
  directories.
