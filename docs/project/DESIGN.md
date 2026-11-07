---
name: Liner Core Product
description: Terminal-native navigator for reading a library of source-grounded liners.
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

Liner's core product design is a restrained terminal workbench for reading a
library of source-grounded liners. It is not a decorative shell around CLI
commands. It is the place where a user browses their liners, sees what needs
them, reads a synthesis or a source, follows a citation, and checks the evidence
that makes a liner trustworthy. The coding agent builds and changes liners; the
navigator is read-only and never changes a liner.

The system should feel calm, dense, explicit, and durable. Calm enough to read in
a terminal. Dense enough to show real library state. Explicit enough that
the person always knows what to ask their agent next. Durable enough that future agents can add screens by
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
`Settings`, `Operating Layer`, `Open MIXTAPE.md`, `LINER.md`, and
`Project Skill`.

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
- **Usage:** a liner's detail page and Settings details.
- **Wrapping:** long values wrap under the value column, never into the label
  column or adjacent pane.
- **Rules:** do not add `Field`/`Value` headers to this pattern.

### Data Table

- **Shape:** muted regular headers, ink cells, two-column padding between cells,
  optional focused-row fill.
- **Usage:** library lists, source inventories, and connections between
  liners.
- **Rules:** use one table per section detail. If a second table has a different
  purpose, promote it to a separate section.

### Library List

- **Shape:** numbered view tabs above one list; `space` opens a short peek of
  the selected liner below the list.
- **Usage:** the Library's All, Attention, Sources, and Connections views.
- **Rules:** the peek stays to identity and decision context. Deeper detail
  belongs on the liner's page.

### Liner Page

- **Shape:** the liner's job at top, an optional waiting or building line, then
  label/value rows for the synthesis, sources, and other files.
- **Usage:** one opened liner. `enter` reads the selected file in the Reader,
  where `tab` moves between citations and `enter` follows one to its source.
- **Rules:** reading only. Work on the liner goes to the agent: `y` offers
  prompts to copy.

### Retired Patterns

File Picker, Review Surface, and Maintenance Preview belonged to the older app
that imported projects, reviewed generated files, and ran maintenance from the
terminal. The navigator does none of that. Reviews and approvals now happen in
the agent's chat, and maintenance goes through the CLI. Do not bring these
patterns back into the navigator.

## 6. Do's and Don'ts

### Do:

- **Do** keep the Library as the home page, with its All, Attention, Sources,
  and Connections views.
- **Do** keep a liner's page as the place to open its synthesis and files.
- **Do** use footer/help as the only navigation and key-hint surface.
- **Do** use Command Orange only for selection, action, progress, and filter
  matches.
- **Do** hand work to the agent with a copyable prompt instead of a write action.
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
- **Don't** let the navigator or an adapter write canonical Project files.
  The navigator never changes a liner.
- **Don't** present Source removal as deletion; removal retains content, while
  purge is a separately approved destructive action.
