# Curating Mixtapes — Skill Bundle

This is the Liner methodology made executable by Claude or Codex. The bundle implements CURATION.md v2.0 as instructions to a running AI. It describes both quick mode (the AI does most of the work, with lightweight review) and methodology mode (the curator engages substantively at every phase).

The bundle is the methodology behind `liner research`. When a coding agent
builds a liner, each research step's prompt points it at this bundle's
`SKILL.md` and at the companion file for that step. It is not the `liner`
skill that finds your liners, the Project Skill, or the optional Maintenance
Adapter.

## Bundle contents

Six files in `src/liner/bundled/research/methodology/`:

```
methodology/
├── SKILL.md                       # the orchestrator
├── source-finding-tactics.md       # Phase 2
├── source-quality-hierarchy.md     # Phase 4
├── curator-notes.md                # Phase 4
├── quality-check-tests.md          # Phase 5
└── synthesis-guidance.md           # Phase 6
```

`SKILL.md` is the orchestrator. It defines the trigger description, the two modes, the review gates, the lifecycle skeleton, the AI biases to compensate for, the boundaries (what the skill does NOT do), and the failure modes to watch. It is always loaded when the skill fires.

The five companion files are loaded only at the phases that use them. Phases 1, 3, 7, and 8 have no companion file — their guidance fits inside SKILL.md.

Each `liner research` step has its own short prompt next to the bundle: `clarify.md` and `phases/*.md` in `src/liner/bundled/research/`. Those prompts say what to write for the step; the bundle says how to curate well.

The split is deliberate. Every invocation pulls SKILL.md's ~2,300 words into context regardless. Companion files add 1,000–1,500 words each but only at the phase that needs them, so total context cost scales with where the user is in the lifecycle rather than with the bundle's full size.

## Where the bundle lives

**Shipping.** The bundle lives at `src/liner/bundled/research/methodology/` and ships inside the bundled CLI in the `linersh` npm package. No separate install is needed.

**Through `liner research`.** This is the normal path. The `liner` skill, installed with `liner agent-skill install claude` or `codex`, tells the agent to follow `liner research next`. Each step returns a prompt file that names this bundle's files by their installed path. The agent runs the steps, often in subagents, and the person answers Clarify Job and gives the three approvals in chat.

**Local testing.** You can also drop this directory into `~/.claude/skills/curating-mixtapes/` and invoke it directly in Claude Code. That is a developer workflow; it skips the checks and approvals that `liner research` enforces.

The same file contents work in every location. Don't fork between a local-test version and the package-shipped version.

## Runtime requirements

Filesystem access is required, not optional. The skill reads its own companion files at specific phases and writes mixtape artifacts to a project folder. It is designed for a coding agent such as Claude Code or Codex, which has filesystem access by default. It is not designed for pure-chat (claude.ai) use.

The agent uses its own web search for candidate discovery. It reads pages with `liner fetch`, which saves each page to a file, and checks the library first with `liner find-sources`. `liner mcp` serves your liners to agents with no shell. It can start a build and run research steps, but it can't approve anything.

## How invocation works

The agent runs `liner research next <slug> --json`. Each step returns a prompt file. For an agent step, the prompt names the methodology `SKILL.md` and the step's companion file: `source-finding-tactics.md` for candidates, `source-quality-hierarchy.md` and `curator-notes.md` for evaluation, `quality-check-tests.md` for quality, and `synthesis-guidance.md` for synthesis. The agent reads them before doing the step's work.

The twelve steps are clarify, framing, candidates, evaluation, quality, synthesis, approve synthesis, assembly, approve sources, build, approve instructions, and finish. When the agent runs `liner research submit`, Liner checks the step's file before moving on. `liner research check` runs the same check without moving on.

Nothing auto-continues past the person. Clarify Job waits for their answers, and each of the three approvals waits for them in chat. The agent never approves for them.

The working notes are written step by step under the liner's `mixtape/` folder, so a build can stop and resume later. `liner research next` picks up where it left off.

At the build step, `liner research submit` compiles the corpus. The skill itself never runs `liner compile`. After the instructions approval, the finish step writes `LINER.md`, root `SKILL.md`, and `liner.yaml`.

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

The bundle is functional and ships inside the bundled CLI, so `liner research` can use it from `npx linersh` or a global install without a separate skill install.

The bundle still benefits from more worked examples in the companion files. Current product risk lives more in curation quality and fetcher edge cases than in basic skill availability.

In the `1.2` source tree (1.2.0 is unreleased; 1.1.3 is the latest published
version), compilation is the Corpus Ready milestone and the finish step, which
writes the Operating Layer, is the Project Complete milestone. A project can be
usable when `MIXTAPE.md` is written but some Sources are missing or
soft-failed after review. Framing must produce a Capability Brief and Required
source roles before discovery can proceed; `liner research submit` refuses the
step until it does.

## Three Skill Surfaces

- The **curating-mixtapes skill** guides the agent through the initial corpus
  recipe, synthesis, and working notes during `liner research`. It never
  performs post-creation maintenance.
- The root **Project Skill** is created by the Operating Layer and teaches future
  sessions how to use one completed Project.
- The optional **Maintenance Adapter** is installed explicitly for Codex or
  Claude and delegates supported changes to the installed CLI's current
  guidance. A `type: skill` Source remains inert evidence and is not installed.

## Versioning

The skill bundle has its own version, independent of the CLI, TUI, and methodology document. Three version numbers track separately:

- **Methodology version** is tracked in CURATION.md (`Current version: 2.0`).
- **Skill bundle version** is the bundle's own — currently v0.3 internally, not yet stamped into SKILL.md frontmatter. Adding a `version` field to the frontmatter is a small open decision; the convention isn't established yet across Liner's surfaces.
- **Methodology compatibility** is asserted by the `methodology_version` field the skill writes into generated `tape.yaml` files. The skill claims compatibility with the methodology version it was built against.

When CURATION.md updates, the skill bundle may need a corresponding update. The question to ask: does the methodology change affect what the running AI should do, or just how a human reader explains the work? If the former, the skill needs an update; if the latter, the skill is fine as-is. Bias toward updating both together when in doubt.

## Relationship to other Liner pieces

The skill is the methodology made executable. It sits alongside Liner's other surfaces and shares the format and methodology with them.

**CURATION.md** is the methodology described to humans. The skill bundle implements it for an AI executor. The two are paired artifacts that update together when the methodology changes substantively.

**The Liner CLI** runs the `liner research` steps, checks each step's files, compiles the corpus at the build step, and owns supported post-creation maintenance. The curation skill never runs `liner compile` itself and never bypasses the Core maintenance contract.

**The terminal navigator** is read-only. It shows a liner that's being built and what's waiting for the person, but it never runs the skill or changes a liner.

**The Liner MCP server** (`liner mcp`) serves your liners to agents with no shell. It can start a build and run research steps with this bundle's prompts inline, but it can't approve anything.

**The Liner library** publishes methodology-mode mixtapes. The skill can produce library-eligible output if run in methodology mode and the empirical test passes — but the bar for library contribution is real curator engagement, not the skill's quick-mode default.

Treat changes to the bundle the way you'd treat changes to the methodology: deliberately, with examples, with a clear reason. The skill is the methodology executed at scale; drift in the skill is drift in the methodology.
