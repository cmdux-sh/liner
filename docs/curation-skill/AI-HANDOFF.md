# Liner AI Handoff

Last updated: 2026-06-25

This is the short orientation document for an AI agent continuing Liner work without the full chat history. It intentionally avoids release-candidate scratch paths and old local worktrees; verify the current checkout with `pwd` and `git status --short` before editing.

## Current Product State

- Repo version files currently agree on `1.0.0`:
  - `pyproject.toml`
  - `src/liner/__init__.py`
  - `packages/tui/package.json`
- The npm package is `linersh`; running `liner` opens the TUI and forwards CLI subcommands to the bundled core.
- The TUI can run agent-backed methodology phases with Claude or Codex, using
  the tracked skill bundle at `docs/curation-skill/`. During npm packaging, it
  is copied into `packages/tui/cli-update-docs/`.
- JS rendering support is optional. `liner setup-js` installs Playwright Chromium; the TUI offers this during onboarding and again when compile detects a source that needs browser rendering.
- Compilation is the product completion milestone. If `MIXTAPE.md` is written with warnings, the project can still be ready to use with warnings shown.

## What Liner Is Building

Liner creates a hyper-specific corpus that helps a downstream AI agent perform one task better. It is not a generic research collector and not a persona builder.

The durable artifact set is:

- `tape.yaml` - source recipe
- `synthesis.md` - curator framing
- `working/` - methodology trace
- `MIXTAPE.md` and `sources/` - compiled consumable context
- `LINER.md` and root `SKILL.md` - operating layer files that tell future agents how to discover and use the project

For the current product and Go TUI model, read `docs/project/PRODUCT.md`,
`docs/project/DESIGN.md`, and `packages/go-tui/README.md`.

## Current Methodology Shape

Phase 1 framing must now produce more than a knowledge map. `working/01-jtbd-and-knowledge-map.md` needs:

- `Capability Brief`
- `Required source roles`
- knowledge map

Each source role needs role name, why it matters, good evidence, and minimum coverage. This is generic: an Art Director Liner, a medical SEO Liner, and a CI debugging Liner should infer different roles from their task.

Phase 2 searches against both the knowledge map and required source roles. Phase 5 includes required source-role fit as a quality test. A source is useful only if fetched content actually supports the role it was kept for.

## TUI UX Rules To Preserve

- Enter should advance the primary next action. Preview/log/retry/open actions need their own keys.
- Long-running work should use the established loading/progress pattern and keep default logs short.
- Warnings need a recovery route: open, drop, replace, retry, add sources, or continue with a documented limitation.
- Letter commands must not interfere with typing.
- Two-option setup/improvement screens should use the shared option style: active orange, inactive gray, selected description white, inactive description gray.
- Completed compile should open the existing result by default; recompile is explicit.

## Active Docs

Read these first for product and methodology work:

1. `README.md`
2. `docs/project/PRODUCT.md`
3. `docs/project/DESIGN.md`
4. `packages/go-tui/README.md`
5. `docs/curation-skill/CURATION.md`
6. `docs/curation-skill/SKILL.md`
7. `docs/curation-skill/source-finding-tactics.md`
8. `docs/curation-skill/quality-check-tests.md`
9. `packages/tui/README.md` if changing npm packaging or installed-user flows

## Verification Commands

TUI:

```sh
npm --prefix packages/tui test -- --run
npm --prefix packages/tui run typecheck
npm --prefix packages/tui run build
```

Python/core:

```sh
uv run --extra dev pytest
python3 -m compileall -q src/liner
```

Docs hygiene:

```sh
git diff --check
rg -n "current release v0\\.5|docs/tui/|docs/README|marketing/site|ink/|go-refactor|personal/ only" README.md docs packages/tui packages/go-tui --glob "*.md" --glob "!docs/curation-skill/AI-HANDOFF.md" --glob "!docs/project/CHANGELOG.md"
```

## Gotchas

- `docs/curation-skill/` is packaged into the npm TUI, so stale methodology docs become user-facing.
- Do not update archived research transcripts just to quiet searches.
- Do not reintroduce "advisor" language for the operating layer. Use artifact names and concrete roles.
- Do not treat URL existence as source quality. Fetched content quality is the real gate.
- Before reporting state, separate your edits from pre-existing dirty worktree changes.
