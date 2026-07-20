---
name: Liner Public Website
description: Terminal-native public website for an open-source toolkit that builds local, source-grounded AI projects.
colors:
  shell: "#f4f3ed"
  shell-line: "#0a0a0a"
  bg-void: "#0a0a0a"
  bg-workspace: "#0f0f0f"
  bg-deep: "#050505"
  line-soft: "#ffffff0d"
  line-medium: "#ffffff1a"
  line-strong: "#ffffff33"
  text-primary: "#ffffff"
  text-bright: "#f4f3ed"
  text-secondary: "#d4d4d8"
  text-muted: "#a1a1aa"
  text-dim: "#8a8a93"
  text-dimmer: "#7a7a83"
  accent-orange: "#ff4500"
  accent-gold: "#ffb800"
  accent-magenta: "#9a00ff"
  tui-green: "#00ac11"
  tui-gold: "#b68800"
  tui-purple: "#ab65ff"
  tui-pink: "#ff0086"
  tui-blue: "#daf1fd"
  tui-grey: "#9a9a9a"
typography:
  display:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, sans-serif"
    fontSize: "clamp(2.5rem, 6vw, 4.75rem)"
    fontWeight: 300
    lineHeight: 1.05
    letterSpacing: "0"
  headline:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, sans-serif"
    fontSize: "clamp(1.75rem, 3vw, 2.5rem)"
    fontWeight: 400
    lineHeight: 1.15
    letterSpacing: "0"
  title:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: 1.25
    letterSpacing: "0"
  body:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.7
    letterSpacing: "0"
  label:
    fontFamily: "JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, Monaco, monospace"
    fontSize: "10px"
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: "0.1em"
rounded:
  none: "0"
  xs: "4px"
  sm: "6px"
  md: "8px"
  full: "999px"
spacing:
  hairline: "1px"
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "32px"
  section-sm: "72px"
  section-md: "96px"
  section-lg: "112px"
components:
  install-command:
    backgroundColor: "{colors.bg-deep}"
    textColor: "{colors.text-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "12px 16px"
  install-panel:
    backgroundColor: "{colors.accent-orange}"
    textColor: "{colors.shell}"
    rounded: "{rounded.none}"
    padding: "32px"
  workspace-panel:
    backgroundColor: "{colors.bg-workspace}"
    textColor: "{colors.text-secondary}"
    rounded: "{rounded.none}"
    padding: "32px"
  deep-panel:
    backgroundColor: "{colors.bg-deep}"
    textColor: "{colors.text-secondary}"
    rounded: "{rounded.none}"
    padding: "32px"
  site-nav:
    backgroundColor: "{colors.bg-workspace}"
    textColor: "{colors.text-secondary}"
    typography: "{typography.label}"
    height: "80px"
---

# Design System: Liner Public Website

## 1. Overview

**Creative North Star: "The Curator's Terminal"**

The Liner public website is a dark, gridded workbench for source-grounded project creation. It should feel like a local system already running: scan lines, file folders, command snippets, status dots, terminal output, and a hot orange install path. The landing page is brand work, while the docs and changelog are trust work. All surfaces earn credibility by showing product artifacts instead of decorating around them.

The visual language is restrained terminal maximalism. It uses a strict dark workspace, visible hairlines, sharp panels, compact mono labels, and one vivid orange accent. The page can be animated and sensory, especially in the hero WebGL panel and methodology rail, but every effect should feel tied to curation, compilation, or local execution.

It explicitly rejects generic AI SaaS, purple-blue gradient gloss, glass panels, beige productivity calm, and editorial magazine drama. The type is Inter and JetBrains Mono because the brand wants operational clarity, not literary affectation.

**Key Characteristics:**

- Dark workspace with visible structural grid lines.
- Orange as the command, progress, and install signal.
- Sharp panels, hairline borders, and small corner markers.
- Real artifacts: commands, file trees, terminal logs, source notes, `MIXTAPE.md`, `LINER.md`, `SKILL.md`, and `liner.yaml`.
- Motion that suggests a running system, with reduced-motion fallbacks.

## 2. Colors

The palette is a dark terminal workspace with one decisive hot orange action color, a small gold success glow, and rare TUI accents for process diagrams.

### Primary

- **Install Orange** (`#ff4500`): Primary action, active state, progress pulse, record dot, install panels, and key proof links. It should be visible on every major fold, but it should not flood ordinary content panels.

### Secondary

- **Compile Gold** (`#ffb800`): Confirmation, copied state, bloom highlights, and selected process accents.
- **Depth Magenta** (`#9a00ff`): Rare gradient depth inside orange bloom overlays. Do not use it as a general brand color.

### Tertiary

- **TUI Green** (`#00ac11`), **TUI Purple** (`#ab65ff`), **TUI Pink** (`#ff0086`), **TUI Blue** (`#daf1fd`), and **TUI Gold** (`#b68800`): Process and terminal-state accents only. These colors are allowed when the page is showing a sequence, status, or terminal-inspired object.

### Neutral

- **Void Black** (`#0a0a0a`): Body background, marquees, deepest outer shell.
- **Workspace Black** (`#0f0f0f`): Main page sections and dark panels.
- **Terminal Black** (`#050505`): Embedded terminals, code blocks, WebGL surface, and high-contrast inner panels.
- **Shell Ivory** (`#f4f3ed`): Light text, logo treatment, and orange-panel contrast.
- **Primary White** (`#ffffff`): Main headline text and command text.
- **Secondary Text** (`#d4d4d8`): Body text on dark surfaces.
- **Muted Text** (`#a1a1aa`): Secondary explanations.
- **Dim Text** (`#8a8a93`) and **Dimmer Text** (`#7a7a83`): Metadata, labels, rails, and low-priority terminal text.
- **Hairlines** (`#ffffff0d`, `#ffffff1a`, `#ffffff33`): Grid lines, panel dividers, borders, and subtle state separators.

### Named Rules

**The Orange Is Command Rule.** Orange means action, progress, install, active state, or a proof link. Do not use it as decoration on generic cards.

**The Dark Workspace Rule.** Primary surfaces stay dark. Ivory is a text and contrast color, not a page background for this landing page.

## 3. Typography

**Display Font:** Inter, with system sans fallbacks<br>
**Body Font:** Inter, with system sans fallbacks<br>
**Label/Mono Font:** JetBrains Mono, with system monospace fallbacks

**Character:** Inter keeps the page sharp and plainspoken. JetBrains Mono gives metadata, commands, and terminal details the product-native texture. The combination should feel like a precise terminal manual, not a magazine spread.

### Hierarchy

- **Display** (300, `clamp(2.5rem, 6vw, 4.75rem)`, 1.05): Hero headlines and major landing-page claims only.
- **Headline** (400, `clamp(1.75rem, 3vw, 2.5rem)`, 1.15): Section-level arguments and major transitions.
- **Title** (400, `1.5rem`, 1.25): Panel headings and dense content surfaces.
- **Body** (400, `1rem` to `1.125rem`, 1.7): Explanatory copy, capped around 65 to 75 characters when possible.
- **Label** (500, `10px`, `0.1em`, uppercase): Eyebrows, metadata, status labels, version markers, nav, and command controls.
- **Terminal Text** (400 to 600, `11px` to `14px`, 1.5 to 1.7): Code, file trees, terminal logs, and process artifacts.

### Named Rules

**The Manual Scale Rule.** Use giant type only for the hero or deliberate campaign moments. Dense panels, docs, changelog, and command surfaces should stay compact and inspectable.

**The Mono Means Artifact Rule.** Mono type is for commands, file names, metadata, status, terminal logs, and process machinery. Do not use mono as a generic aesthetic wrapper for paragraphs.

## 4. Elevation

The landing page does not use conventional shadows. Depth comes from tonal layering, hard borders, hairline grids, corner markers, and contrast between workspace panels and deep terminal surfaces. This is a flat system with structural depth, not a floating-card system.

### Shadow Vocabulary

- **Record Pulse** (`box-shadow: 0 0 0 0 rgb(255 69 0 / 0.65)` animated outward): Use only for live status dots and active recording indicators.
- **No Resting Shadow** (`box-shadow: none`): Default for cards, panels, buttons, code, and docs components.

### Named Rules

**The Hairline Depth Rule.** Use 1px borders, alpha hairlines, panel seams, and background contrast before reaching for shadows.

## 5. Components

### Buttons

- **Shape:** Small radius only (`4px` to `6px`) for command pills. Most structural surfaces stay square.
- **Primary:** Orange panel CTAs use `#ff4500` as the surrounding field with a deep black command pill inside.
- **Hover / Focus:** Keep color changes subtle. Preserve contrast and stable dimensions.
- **Copy Button:** Mono uppercase label, fixed width, border-left divider, `Copy` and `Copied` states with Lucide copy/check icons. The button must not resize when its state changes.

### Chips

- **Style:** Tiny mono labels with 1px borders or muted text. Use them for version, status, and artifact metadata.
- **State:** Active chips may use orange text or a small orange dot. Avoid pill-heavy UI.

### Cards / Containers

- **Corner Style:** Square by default. If radius is needed, cap at `8px`.
- **Background:** Use `#0f0f0f` for workspace panels and `#050505` for embedded terminal or artifact surfaces.
- **Shadow Strategy:** No shadow at rest. Use hairline borders and grid seams.
- **Border:** 1px alpha white lines, with occasional orange corner markers for important artifacts.
- **Internal Padding:** `24px` to `32px` for dense panels, `48px` to `80px` for large landing-page sections.

### Inputs / Fields

The landing page has no traditional input system. Search or docs inputs should inherit the command-surface language: dark fill, 1px border, mono labels, and an orange focus border.

### Navigation

Header navigation is compact and mono. The brand mark is the orange record dot plus the Liner logo. Desktop header height is 80px to 96px depending on context. Active and high-intent links use orange. GitHub links use a Lucide arrow icon.

### Install Command

This is the signature component. It is a dark command pill with a selectable `$ npx linersh` code line and a fixed-width copy control. Use it anywhere the page asks the visitor to run Liner. It should feel identical across hero, docs, changelog, and bottom CTA surfaces.

### File Tree / Artifact Panel

File trees and output panels should look like Liner artifacts: mono type, dark surface, hairline border, compact rows, optional orange label, and no decorative terminal titlebar. Use them for `MIXTAPE.md`, `sources/`, `tape.yaml`, and related folder examples.

### Docs Shell

The docs shell uses Starlight, but it should not look like default Starlight. Preserve the landing-page header, dark workspace, mono side navigation, orange active states, compact headings, and command/file artifact components.

Guidelines:

- Keep content aligned to the same perceived max width as the landing page.
- Keep docs `h2` sizes closer to mobile scale than hero scale.
- Use compact vertical rhythm. Avoid wide editorial gaps between sections.
- Place search in the top navigation so it is always available.
- Keep sidebar, content, and table of contents borders as hairlines.
- Remove decorative dividers between ordinary paragraphs.
- Keep bottom prev/next navigation small, mono-labeled, and secondary.
- Remove edit-page links unless a real edit workflow is intentionally added.

### Changelog

The changelog is a reading surface, so cap line length more tightly than broad landing-page sections. Version numbers may be large, but status labels such as unreleased or current should be small mono metadata rather than part of the huge version line.

Use the same command snippet and file-tree components as docs. Command snippets in changelog should stay near landing-page scale and never become oversized hero objects.

### Motion

Motion should suggest running systems: WebGL block columns, rail scans, marquee tracks, masked headline reveals, progress bars, and terminal cursor blink. Every animation must respect `prefers-reduced-motion: reduce`.

## 6. Do's and Don'ts

### Do:

- **Do** keep the first viewport product-specific: logo, headline, WebGL/product-shot panel, and install command.
- **Do** use the existing colors from `src/styles/global.css`; do not invent a parallel palette.
- **Do** make docs and changelog feel like first-party Liner surfaces, not a separate documentation template.
- **Do** use orange for commands, active state, progress, proof links, and installation.
- **Do** show concrete artifacts: commands, files, source notes, terminal states, folder output, compiled corpus details, and Operating Layer files.
- **Do** keep body text comfortable and accessible on dark surfaces, using `#d4d4d8` or brighter.
- **Do** keep command snippets copyable and stable in width.
- **Do** use Lucide icons for interface controls when an icon is needed.
- **Do** keep reduced-motion behavior current when adding animation.

### Don't:

- **Don't** make Liner look like generic AI tool marketing with pastel gradients, floating chat bubbles, soft cards, and vague productivity claims.
- **Don't** add purple-blue gradients, glassmorphism, decorative orbs, abstract AI sparkle visuals, or bokeh backgrounds.
- **Don't** move the landing page into an editorial magazine lane with giant whitespace, serif display type, drop caps, or broadsheet affectation.
- **Don't** use beige or cream as a primary background for this landing page.
- **Don't** call Liner a framework. Use open-source toolkit.
- **Don't** use em dashes or "Not this, but this" contrast framing in marketing copy.
- **Don't** use side-stripe accent borders. Use full hairline borders, corner markers, dots, or labels.
- **Don't** use large rounded cards or nested cards.
- **Don't** let marquees, code, file trees, or animated panels create horizontal overflow.
- **Don't** allow dark syntax highlighting to make YAML, code, or command output unreadable.
- **Don't** document TUI keys, CLI commands, or tape schema from memory. Check the source first.
