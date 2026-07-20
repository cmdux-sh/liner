# Product

This context is scoped to the Liner public website in `marketing/site`: the landing page, docs, and changelog. It should guide brand-site and docs-site design work for the public website, not CLI core behavior, TUI product screens, release engineering, or package internals.

## Register

brand

## Users

The public website is for AI power users, developer-tool makers, designers, founders, and technical operators who already use coding agents, chat tools, and local CLIs. They are source-sensitive: they know AI output gets better when the input context is curated, current, and specific to the job.

They arrive with a focused work problem, often after feeling that generic search, broad research agents, or one-off prompts produce context that is too shallow to reuse. Their job is to understand whether Liner can help them build a durable local project, install it quickly, and trust that the output is just files they can inspect.

Docs users are a little further along. They need exact install commands, truthful CLI/TUI behavior, readable examples, and copyable snippets. Changelog readers need to understand what changed without parsing release engineering noise.

## Product Purpose

Liner is an open-source toolkit for building local, source-grounded AI projects. A Liner Project is a folder with a Mixtape corpus inside it plus an Operating Layer: `LINER.md`, root `SKILL.md`, and `liner.yaml`.

The landing page should make one thing feel obvious: source choice is part of the work, and future AI sessions need more than source files. The tool gives that work a shape, a local workflow, and a reusable project artifact. Success means the visitor understands the project model, trusts that the output is local and inspectable, and feels confident enough to run `npx linersh`.

The docs and changelog should preserve that same trust after the first click. They should feel like part of the product, use the same command and file artifacts as the landing page, and stay accurate to the current TUI, npm wrapper, and Python core.

## Brand Personality

Terminal-native, exacting, tactile.

Liner should feel like a serious local tool for people who care about source quality, not like another generic AI SaaS wrapper. The brand voice is direct, compressed, and confident. It uses concrete nouns: folders, sources, notes, synthesis, `MIXTAPE.md`, `LINER.md`, `SKILL.md`, `liner.yaml`, local files, commands.

The emotional target is focused momentum. The page should feel alive and operational, but never noisy. It should make careful curation feel like a craft with visible machinery.

## Anti-references

- Generic AI tool marketing with pastel gradients, floating chat bubbles, soft cards, and vague productivity claims.
- Purple-blue gradient SaaS pages, glassmorphism, decorative orbs, and abstract AI sparkle visuals.
- Editorial magazine pages with giant type, excessive whitespace, serif display drama, drop caps, or broadsheet affectation.
- Beige/cream productivity aesthetics that make the product feel detached from the terminal.
- Stale "framework" language. Liner is an open-source toolkit.
- "Not this, but this" contrast copy, em dashes, and copy that argues with straw men.
- Instruction-heavy interface text that explains UI mechanics instead of showing the working artifact.

## Design Principles

1. **Show The Artifact.** The page should repeatedly reveal the kept folder, source notes, terminal output, and install command. Liner is credible when the visitor can see the thing they get.
2. **Make Context Operational.** The page should turn research from an abstract promise into a sequence: frame, gather, evaluate, synthesize, compile, reuse.
3. **Keep The Brand Local.** Favor terminal-native surfaces, file-system metaphors, command snippets, and inspectable outputs over cloud-dashboard language.
4. **Use Accent As Signal.** Orange marks action, status, progress, and the install path. It should feel rare enough to matter.
5. **Practice The Project Thesis.** The landing page should show how a corpus becomes an Operating Layer. Preserve proof from real runs, but keep Art Director as an example rather than the product headline. Do not present imagined project examples as shipped inventory.

## Accessibility & Inclusion

Target WCAG AA contrast for body text, labels, links, code, and controls on dark surfaces. The current dim text tokens were chosen to meet AA on the dark workspace and should not be darkened casually.

Support reduced motion. WebGL, marquee, rail scans, cursor blinks, and reveal animations must either become static or settle cleanly under `prefers-reduced-motion: reduce`.

Interactive controls need stable hit areas and accessible names. Copy buttons must keep a fixed width before and after the copied state so the layout does not jump.

The page should avoid horizontal overflow on mobile and desktop. Long code, file names, marquees, and animated panels need explicit overflow handling.

## Public Website Surfaces

Landing page:

- Purpose: explain the Liner Project model and drive a confident install.
- Style: expressive product proof with WebGL, terminal artifacts, method rail, and orange install CTA.
- Risk: becoming generic AI marketing or too decorative.

Docs:

- Purpose: help users install, build, compile, complete, share, and troubleshoot accurately.
- Style: compact tool manual using the same colors, type, nav, command snippets, and file-tree artifacts as the landing page.
- Risk: drifting into default Starlight/editorial styling, oversized headings, over-wide measures, or stale product claims.

Changelog:

- Purpose: make release state readable and trustworthy.
- Style: narrower reading measure, large version numbers, small mono status labels, compact command snippets.
- Risk: letting command/output examples become too large or making version metadata compete with the version number.

## Implementation Notes

- The docs use Starlight, with the header overridden through `src/components/starlight/Header.astro`.
- Shared marketing components live in `src/components/`.
- Use `CommandSnippet.astro` for copyable commands and `FileTree.astro` for folder examples.
- Search is Pagefind generated during production build. Dev can show a search warning before build output exists.
- Before changing docs claims, check the source in `packages/tui/src/`, `packages/tui/bin/liner.js`, and `src/liner/`.
