# Curating Mixtapes — Skill Bundle

This is the Liner methodology made executable by Claude. The bundle implements CURATION.md v2.0 as instructions to a running AI, drives the mixtape lifecycle from a topic to a project folder ready for `liner compile`, and supports both quick mode (the AI does most of the work, the user confirms at two gates) and methodology mode (the curator engages substantively at every phase).

It is one of Liner's surfaces, alongside the CLI, the TUI, the planned MCP server, and the planned web builder. The skill is the surface for Claude-Code-native and TUI-subprocess authoring. Methodology-mode curators can also use it directly when working in a Claude Code session.

## Bundle contents

Six files in one directory:

```
curating-mixtapes/
├── SKILL.md                       # the orchestrator
├── source-finding-tactics.md       # Phase 2
├── source-quality-hierarchy.md     # Phase 4
├── curator-notes.md                # Phase 4
├── quality-check-tests.md          # Phase 5
└── synthesis-guidance.md           # Phase 6
```

`SKILL.md` is the orchestrator. It defines the trigger description, the two modes, the two gates, the eight-phase lifecycle skeleton, the AI biases to compensate for, the boundaries (what the skill does NOT do), and the failure modes to watch. It is always loaded when the skill fires.

The five companion files are loaded only at the phases that use them. Phases 1, 3, 7, and 8 have no companion file — their guidance fits inside SKILL.md.

The split is deliberate. Every invocation pulls SKILL.md's ~2,300 words into context regardless. Companion files add 1,000–1,500 words each but only at the phase that needs them, so total context cost scales with where the user is in the lifecycle rather than with the bundle's full size.

## Where the bundle lives

**Local testing.** Drop this directory into `~/.claude/skills/curating-mixtapes/`. Claude Code and Claude Desktop both look there for user-installed skills. The skill fires when the user says something the description matches — "help me make a mixtape on [topic]" is the canonical trigger.

**Shipping.** The canonical bundle lives at repo-root `cli-update-docs/`. The TUI copies this directory into `packages/tui/cli-update-docs/` during `prepack` and ships it inside the `linersh` npm package, so end users do not need a separate skill install for TUI-driven agent phases.

**Direct skill use.** A documented copy/install flow for using the skill outside the TUI is still planned. Until then, direct local use is a developer workflow.

**TUI subprocess.** The Liner TUI bundles the skill internally and invokes it via the local Claude or Codex CLI in quick mode. The user doesn't see the skill directly; they see the TUI's wizard UI. The methodology being applied is this skill's.

The same file contents work in every location. Don't fork between a local-test version and the package-shipped version.

## Runtime requirements

Filesystem access is required, not optional. The skill reads its own companion files at specific phases and writes mixtape artifacts to a project folder. It is designed for execution inside Claude Code or as a TUI subprocess, both of which have filesystem access by default. It is not designed for pure-chat (claude.ai) use.

No MCP server is required. v1 uses Claude's own web search for candidate discovery and URL verification. When the Liner MCP server ships, a future skill version will use it for fetching and cache operations; v1 doesn't depend on it.

## How invocation works

The user says something the description matches. Claude loads SKILL.md. Claude reads it, identifies which mode applies (quick by default unless the user says otherwise or asks for library contribution), and begins the lifecycle.

At Phase 2, Claude reads `source-finding-tactics.md`. At Phase 4, Claude reads `source-quality-hierarchy.md` and `curator-notes.md`. At Phase 5, Claude reads `quality-check-tests.md`. At Phase 6, Claude reads `synthesis-guidance.md`. Each companion file is read at the start of its phase, before any work in that phase happens.

The two gates pause for user input. In quick mode, the default is "looks good, continue" — Claude proceeds unless the user actively objects. In methodology mode, Claude pauses for substantive review.

The skill writes artifacts to a project folder named for the topic (e.g., `mobile-design-foundations/`) as it goes. The working notes are produced phase by phase, so the user can stop at any phase boundary and resume later — the working files are picked up where they were left.

When all phases complete, the user runs `liner compile <project-folder>` themselves. The skill never compiles.

## How to test the bundle

Pick a small topic the curator hasn't already researched. "Writing better LLM evals," "Notion API patterns," "designing for color blindness" — narrow, technical, with enough source material that the corpus has shape but not so much that the run takes hours. Run the skill end-to-end in quick mode. Note where Claude struggles, where the methodology produces friction, where the gates fire awkwardly.

Run again in methodology mode on the same topic. Compare the artifacts. The two modes should produce same-shape folders with different depth of curator engagement showing through. If the methodology-mode output looks indistinguishable from the quick-mode output, that's a signal the skill isn't asking enough of methodology-mode curators.

For higher signal: run the skill in quick mode on a topic that already has a hand-curated mixtape (the CLI/TUI corpus is the obvious one). Compare the skill's tape file and synthesis against the hand-curated version. This is the empirical test for the skill itself, structurally identical to the methodology's Phase 8 test for mixtapes. If the skill's quick-mode output is meaningfully worse than the hand-curated version, that's expected — quick mode targets "good enough for personal use." If the gap is larger than expected, the skill needs work.

## How to update or extend the bundle

Three kinds of changes happen, each with a different shape.

**Tightening SKILL.md.** Phase descriptions, gate behavior, trigger description, the AI-biases list, the failure-modes list. Edit in place. SKILL.md is the source of truth for the bundle's structure; everything else conforms to what SKILL.md says.

**Editing a companion file.** Refinements to the tactics, hierarchy, notes guidance, tests, or synthesis advice. Edit in place. Keep the orchestrator references in SKILL.md unchanged unless the change affects when or how a companion gets read.

**Adding a new companion file.** Two questions to answer first. Is the content concretely better than what could fit in SKILL.md? (Worked examples and pattern catalogs are the usual reasons.) Does the content apply to one specific phase rather than the whole lifecycle? If yes to both, add the companion and reference it from SKILL.md's lifecycle section. If no to either, the content probably belongs in SKILL.md itself or in CURATION.md.

When adding a companion, update this README's "Bundle contents" section and the load-profile note in SKILL.md. Don't let the bundle's structure drift from how the README describes it.

## Current status

The bundle is functional and ships inside the npm TUI package so agent-backed phases can run from `npx linersh` without a separate skill install. The TUI copies this directory into the package during `prepack` and resolves it from the installed package root.

The bundle still benefits from more worked examples in the companion files, but it is no longer blocked by the old "second pass before any shipping" constraint. Current product risk lives more in curation quality and fetcher edge cases than in basic skill availability.

As of `linersh@0.5.7`, compilation is the product completion milestone. A project can be "ready to use — compiled with warnings" when `MIXTAPE.md` is written but some sources are missing or soft-failed. The TUI opens the existing compile result for a completed mixtape and requires an explicit retry/re-run from that screen.

## Versioning

The skill bundle has its own version, independent of the CLI, TUI, and methodology document. Three version numbers track separately:

- **Methodology version** is tracked in CURATION.md (`Current version: 2.0`).
- **Skill bundle version** is the bundle's own — currently v0.3 internally, not yet stamped into SKILL.md frontmatter. Adding a `version` field to the frontmatter is a small open decision; the convention isn't established yet across Liner's surfaces.
- **Methodology compatibility** is asserted by the `methodology_version` field the skill writes into generated `tape.yaml` files. The skill claims compatibility with the methodology version it was built against.

When CURATION.md updates, the skill bundle may need a corresponding update. The question to ask: does the methodology change affect what the running AI should do, or just how a human reader explains the work? If the former, the skill needs an update; if the latter, the skill is fine as-is. Bias toward updating both together when in doubt.

## Relationship to other Liner pieces

The skill is the methodology made executable. It sits alongside Liner's other surfaces and shares the format and methodology with them.

**CURATION.md** is the methodology described to humans. The skill bundle implements it for an AI executor. The two are paired artifacts that update together when the methodology changes substantively.

**The Liner CLI** runs `liner compile` against the project folder this skill produces. The skill never compiles; the CLI never authors. Clean split.

**The Liner TUI** invokes this skill internally for quick-mode authoring via the Claude Agent SDK. Same methodology, different UX.

**The Liner MCP server** (planned) will expose mechanical operations — fetch this URL, validate this tape file, query the cache — as tools. A future skill version will use those tools where v1 uses Claude's own web search.

**The Liner library** publishes methodology-mode mixtapes. The skill can produce library-eligible output if run in methodology mode and the empirical test passes — but the bar for library contribution is real curator engagement, not the skill's quick-mode default.

Treat changes to the bundle the way you'd treat changes to the methodology: deliberately, with examples, with a clear reason. The skill is the methodology executed at scale; drift in the skill is drift in the methodology.
