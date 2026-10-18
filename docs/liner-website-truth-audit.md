# Liner website truth and coverage audit

Audit date: 2026-10-17. Scope: visitor-facing source in `marketing/site/src/`, Starlight configuration, source-generated changelog, npm launcher, Python Core, and active Go TUI. This is a read-only product audit; only this report was written. Source line references describe the pre-rewrite checkout and may move during implementation.

## Findings requiring changes

### P1: The onboarding copy button also copies an immediate uninstall

- Surface: `marketing/site/src/content/docs/docs/install.mdx:16-20` groups version checking, browser setup, and `uninstall --yes` in one copyable snippet under One-Shot. `CommandSnippet.astro:20-22,45-46` copies the entire string. A visitor following installation can delete existing settings and shared caches.
- Evidence: `packages/tui/bin/liner.js:101-103,220-279,297-335` implements uninstall as recursive deletion of `~/.liner`, the platform Playwright directory, and npm's `_npx` directory. `--yes` bypasses confirmation.
- Replacement: installation snippets contain only installation/launch or a version check. Put uninstall in its own explicitly labeled section, start with `npx --yes linersh@latest uninstall --dry-run`, then offer the interactive `npx --yes linersh@latest uninstall` separately. Explain the exact deletion scope before the command.

### P1: Release hard-reset commands should not be public getting-started instructions

- Surface: `marketing/site/src/content/docs/docs/install.mdx:95-122` includes global package removal, `rm -rf ~/.liner`, shared npm execution cache removal, full `npm cache clean --force`, and shared Playwright cache removal. The test-machine caveat does not make this a useful visitor install path.
- Evidence: the snippet itself names release testing and removes wider shared state than a Liner package uninstall. This is release engineering, not a prerequisite for first use.
- Replacement: remove this entire section from the public install page. Existing internal release/development documentation is the right home for maintainer clean-machine verification. Do not reproduce the reset commands elsewhere in visitor troubleshooting.

### P1: Uninstall language understates configuration and shared-cache loss

- Surfaces: `install.mdx:84-89`, `troubleshooting.mdx:43-48`, `cli.mdx:99-102`. The install page describes a local cache and uses non-interactive deletion; troubleshooting presents two alternative uninstall forms as one block that runs both.
- Evidence: `packages/tui/bin/liner.js:297-335` removes all of `~/.liner` (including configuration), the entire shared `ms-playwright` directory, and all npm `_npx` entries. The implementation does not recognize a Project specially if a user has placed one inside a deleted path.
- Replacement: “This removes Liner configuration and source cache, plus shared Playwright browser downloads and npm's npx execution cache. Other tools may need to download those files again. Inspect the paths with `uninstall --dry-run` first. Saved Projects outside those paths remain.” Use one command for each install method, as separate alternatives. Avoid an unconditional “leaves your folders alone.”

### P1: Sharing filters are not a privacy scrub

- Surface: `sharing.mdx:39-49` lists flags without their exact limits, then suggests `--no-personal` for library submissions. Homepage `index.astro:503` correctly asks users to check private Sources, but this needs a more precise docs destination.
- Evidence: `src/liner/share.py:98-128` walks all files, including hidden files and `.liner-runs`, unless one of three directory predicates excludes them. `--no-personal` omits only `local-sources/` and legacy `personal/`, not copies already extracted into `sources/`, synthesis, `MIXTAPE.md`, or run logs. `--no-source-content` omits only the `sources/` directory; it is not a no-content guarantee. `packages/tui/src/agents/run-log.ts:12-16,52-81` stores raw runner JSONL, and `src/liner/output/mixtape.py:53-55,67-73` writes extracted sources and synthesis-derived output.
- Replacement: describe each flag as a directory filter; explicitly say none anonymizes or removes private material from derived files or logs. Tell visitors to inspect the archive's full contents before sharing. `--minimal` contains only the tape recipe (`share.py:94-96`), which can still contain private URLs, notes, and metadata. Drop library-submission wording because no public submission path is offered.

### P1: Price, local storage, AI processing, and telemetry need separate boundaries

- Surfaces: `index.astro:158,191,209,502,530,584-589,599`; `changelog.astro:212-215`; install/overview/build docs omit a clear preflight for the AI runner. “No account ... required by the tool” beside the main guided workflow obscures that research uses a configured provider.
- Evidence: `packages/go-tui/internal/app/onboarding.go:88-105` asks users to install/select an AI runner; `methodology.go:108-124` starts the headless methodology runner; `packages/tui/src/agents/runner.ts:398-443,841-927` launches Codex or Claude and sends the prompt to its stdin. The configured provider handles research; provider terms, processing, account limits, and any charges are separate. `marketing/site/src/layouts/BaseLayout.astro:19-20,88-122,141-164` loads PostHog on non-local hosts when configured and records pageviews/pageleave and intent clicks. No telemetry client was found in the audited Core/TUI/runner source.
- Replacement: “Liner is free and MIT licensed. AI-assisted research uses your installed Codex CLI or Claude Code runner and that account's usage limits or charges. Project files are saved locally; material read by the runner is processed under your provider's terms. The installed Liner tool has no telemetry.” For site analytics, describe the site separately wherever a visitor-facing privacy statement is added. Do not claim offline research, that all information stays on-device, or that AI usage is included.
- Precision: Compile does not make a new model call (`src/liner/compile.py:60-79,119-159`). Current Create Operating Layer also builds files locally from saved artifacts (`packages/go-tui/internal/app/liner.go:85-151,492-540,846+`). Do not invent a model charge for those two mechanical steps.

### P2: The CLI reference wrongly labels Source addition as plan-only and removal as applied

- Surfaces: `cli.mdx:53-57`: `sources add` says “Plan”; `sources remove` says “Detach.”
- Evidence: `src/liner/cli.py:1340-1366` applies a low-risk addition immediately and prints a receipt; an approval-required identity migration instead emits a Change Set and exits 2. `cli.py:1401-1416` applies metadata/locator updates. `cli.py:1497-1522` only plans removal and tells the user to apply the exact JSON. Replace/purge also plan rather than immediately perform the destructive operation.
- Replacement: add = “Add a Source and record a receipt when approval is not required; otherwise return the exact plan for review.” Update = “Apply a metadata/locator update while preserving identity.” Remove = “Plan retention-first removal; apply the exact approved Change Set separately.” Keep explicit plan/apply distinctions for rename, move, replacement, and purge.

### P2: Documented Project shortcuts do not match the active key handler

- Surface: `tui.mdx:52-53` promises `k` for skills and `u` for audits; `tui.mdx:39,57` promises `q` exits from Projects and Project.
- Evidence: `packages/go-tui/internal/app/keys.go:393-488` has no Project `k` branch. Its `u` branch (`441-443`) retries excluded local sources only from the Sources pane. `keys.go:224-225` handles `q` as quit only on Home, while Ctrl+C is global. The declared old bindings at `keys.go:58-59` are not proof that a screen handles them.
- Replacement: remove unsupported Project shortcuts; describe `u` only if documenting its actual pane-specific recovery behavior. Use Ctrl+C for the always-available quit route and `q` only where implemented. Compare all remaining key rows to their screen handlers, not just binding definitions. Existing compile rows map to `keys.go:491-526`.

### P2: Recompile instructions promise an unavailable regeneration path

- Surface: `build-a-mixtape.mdx:75-81` says rerun Compile and recreate instruction files if the research changed.
- Evidence: `packages/go-tui/internal/app/liner.go:29-34` allows creation only when no `LINER.md` exists and explicitly returns false for regeneration. `keys.go:477-484` consequently cannot regenerate an existing Operating Layer. Core already supports reviewed `synthesis.review` and `operating_layer.review` dispositions (`src/liner/maintenance.py:792-850,1213-1231,2619-2628`).
- Replacement: after Source maintenance, inspect lifecycle/next actions and follow the separately reviewed synthesis, compile, and instruction refresh steps through `liner project guidance` / the Maintenance Adapter. Do not tell users that pressing Create Operating Layer again regenerates existing files. Preserve the simple first-creation instructions.

### P2: “Fresh extracted source files” ignores the cache

- Surfaces: `sharing.mdx:35` and `tui.mdx:145` say to compile after import for fresh files.
- Evidence: `src/liner/compile.py:127-140` reuses a valid cached source; `src/liner/config.py:12-15` defaults to seven-day web and thirty-day YouTube TTLs. `src/liner/cli.py:458-460` supplies `--no-cache` to force a refetch.
- Replacement: “Compile to produce or update the extracted source files. Compile normally reuses valid cached content; use `liner compile <folder> --no-cache` when you specifically need a new fetch.” Do not promise retrieval succeeds for every URL.

### P2: Maintenance snippets collapse inspect, change, and removal into one copy action

- Surfaces: `maintenance.mdx:84-86` copies inspect, update, and remove together, so a visitor trying an update finishes by uninstalling the adapter. Install/setup alternative forms are similarly grouped in `install.mdx:79-80,86-87` and `troubleshooting.mdx:22-23,45-46`.
- Evidence: `CommandSnippet.astro:21,45-46` copies every line. `src/liner/cli.py:161-188` treats adapter install/update/remove as separate actions with `--yes` authorizing file effects.
- Replacement: distinct snippets for distinct intentions and for global-vs-npx alternatives. Keep sequential commands together only when all are part of the same intended task. Prefer preview/inspection before the explicitly authorized mutation; do not imply `--yes` waits for a later confirmation.

### P2: The public changelog imports internal release and verification notes verbatim

- Surface: `changelog.astro:23,35-55,130-138,242-247` renders every bullet from `docs/project/CHANGELOG.md`. `docs/project/CHANGELOG.md:25-40` exposes release candidate preparation, isolated package smoke, CI protocol drift checking, and retired internal writer paths. Historical entries also include clean reinstall testing and internal helper changes (`133-135,164-165`); the public introductory copy calls all historical entries shipped behavior (`changelog.astro:224-227`).
- Evidence: the parser has no editorial filtering: it merges bullet continuation text and renders it, while the “shipped” classifier checks heading text rather than registry/releases (`changelog.astro:88-91,118-128`). Current registry latest is 1.1.1, so the displayed version is currently correct; this logic alone does not prove future releases shipped.
- Replacement: preserve release numbers and visitor-relevant effects, while curating public summaries in site-owned data or explicitly selecting public bullets. Keep actual internal release/test details in the source changelog. State candidate status only from explicit release state; never infer that all numbered headings are published.

### P2: First-use docs skip the installed AI runner prerequisite

- Surfaces: `install.mdx:8-14`; `index.mdx:30-38`; `build-a-mixtape.mdx:13-30`; `tui.mdx:12` starts immediately with Project creation. The only detailed Codex/Claude install section describes the optional Maintenance Adapter, which may make the research runner seem optional too.
- Evidence: first-run setup includes Projects folder, AI runner selection and optional JS setup (`packages/go-tui/internal/app/onboarding.go:80-125`). The app points to Codex CLI and Claude Code docs if neither runner is present (`100-105`).
- Replacement: include a compact preflight before launch: Node 18+, supported OS/CPU, installed/authenticated Codex CLI or Claude Code for AI-assisted research, and a local Project location. Explain that the optional Maintenance Adapter is a separate opt-in feature for subsequent Project maintenance, not the runner installation.

### P2: Local-source paths need the `mixtape/` root made explicit

- Surfaces: `tape-format.md:95-103`; `tui.mdx:75-79` describe the Project's `local-sources/`. The file tree in `mixtapes.mdx:17-25` correctly nests it under `mixtape/`.
- Evidence: `src/liner/handlers/local_file.py:88-105` resolves recipe paths against `ProjectFolder.corpus_path`; for new Projects that is `<project>/mixtape` (`src/liner/project.py:137-146`). The valid recipe path stays `local-sources/file.pdf`, but its actual location is `<project>/mixtape/local-sources/file.pdf`.
- Replacement: explicitly distinguish recipe-relative paths from disk locations: “In current Projects, put the file in `mixtape/local-sources/`; in `mixtape/tape.yaml`, use `path: local-sources/...`. Legacy root-level Projects resolve the same recipe path from their root.”

### P3: The landing explanation contradicts the plain-language on-ramp

- Surfaces: `index.astro:71` introduces Liner Project, Mixtape, Job to Be Done, Job Story, and Clarify Job in the first paragraph; `459` uses Project Complete. `tui.mdx:12` advertises the active Go versus archived Ink implementation. These are implementation/model details before a new reader has the useful objects clear.
- Evidence: `marketing/site/MARKETING_COPY_BEST_PRACTICES.md` explicitly orders local CLI → sources → folder → files → optional product nouns and calls for internal milestones to be translated. This is a coverage/clarity issue rather than a nonexistent capability.
- Replacement: lead with “A local CLI and terminal UI that turns focused research sources into files your AI assistant can use later.” Show sources, synthesis, and instructions before naming the product model. Use the exact TUI action names only in procedural docs. Remove the archived Ink reference from visitor onboarding.

### P3: Overview Source definition omits a supported input

- Surface: `index.mdx:53` defines Sources as web pages, YouTube transcripts, or local files.
- Evidence: `src/liner/tape.py:12` and `src/liner/types.py:7` also support `skill`; tape docs already explain inert skill Sources.
- Replacement: add “skill files read as reference material” to the definition. Preserve the distinction from active Project Skill and optional Maintenance Adapter.

### P3: A plain text output example is presented as an executable command

- Surface: `troubleshooting.mdx:12-14` passes version output to `CommandSnippet` with a Copy control and “Command to run” accessible label (`CommandSnippet.astro:17,22`).
- Replacement: render expected version output as plain output/code, without an executable-command copy button or command label. Consider dynamic version injection for current docs; show the two version fields as an example if installed versions legitimately differ.

### P3: Social image copy still uses the older research-only headline

- Surface: `BaseLayout.astro:13,43,47` points social cards to `public/og-image.png`; visual inspection confirms “Deep research for one specific job” / “steer the angle · keep the sources · reuse the corpus.” `public/og-image.svg:7-14` contains the same copy.
- Replacement: update the social image text to the accepted new homepage promise when implementation settles it, retaining a concrete local-files subtitle. This is consistency work, not a false shipped-feature claim. Both the raster used by metadata and its SVG source must agree.

## Claims checked and supported

- `npx linersh` opens the active Go TUI; subcommands dispatch to Core. Evidence: `packages/tui/bin/liner.js:86-149`. Node 18+ and the five named optional native platform packages match `packages/tui/package.json:50-64`. Package bundling means normal npm users do not need a separate Python installation. Windows support is x64; do not broaden that to all Windows CPUs.
- Local `VERSION`, npm wrapper/package/platform dependencies, and site examples are 1.1.1. A read-only `npm view linersh version dist-tags --json` returned latest 1.1.1 on 2026-10-17; `next` is 0.5.7. Do not casually recommend `@next`.
- The public GitHub destination `https://github.com/cmdux-sh/liner` returned HTTP 200 without authentication; `https://api.github.com/repos/cmdux-sh/liner` reported `private: false`, MIT license, default branch `main`. The working remote is separately `https://github.com/cmdux-sh/liner-private.git`. No visitor source directly links to the private repository. Preserve the existing public GitHub destination, not the development remote. Header, hero, footer, Starlight social link and package metadata agree.
- The default TUI library is `~/liner/projects`, with `LINER_DIR` and settings overrides (`packages/go-tui/internal/core/core.go:378-392`). CLI relative folder commands operate relative to the shell's working directory; they do not automatically resolve a slug against the TUI library.
- There are four recipe Source types: web, YouTube, local file, skill. Local files support `.md`, `.txt`, `.html`, `.htm`, `.pdf` and a 10 MiB cap (`src/liner/handlers/local_file.py:23-26,60-74`). No claim of OCR, images, DOCX, audio transcription, authenticated browser capture, or a public library should be added.
- `MIXTAPE.md` contains synthesis plus an index linking separate saved source files, not necessarily all full source bodies in one portable file (`src/liner/output/mixtape.py:22-34,124-155`). Keep `mixtape/sources/` available for detail. Copy/paste reuse works, but a chat receiving only the index cannot follow inaccessible local files.
- Compile may use fetched content, cache, or a marked research-agent summary fallback when extraction fails (`src/liner/compile.py:42-57,94-117,127-140`). It may produce partial output with warnings. Avoid “every source saved in full” or “automatically verified truth.”
- Export is a zip archive; import extracts to the chosen/current directory and CLI refetch is enabled by default (`src/liner/share.py:28-85`; `src/liner/cli.py:633-714`). TUI import's no-refetch behavior is a legitimate difference and should remain described. Import does not itself synthesize or regenerate instructions.
- New Projects have `mixtape/`, recipe, synthesis, sources and instruction/status artifacts as shown by the site. Instructions are a later step; a bare new/compiled Project need not already contain all completed files.
- Immutable Project/Source IDs, plan/apply stale guards, retention-first removal, separately destructive purge, redacted maintenance receipts, and recoverable whole-Project move-to-trash are supported. Scope “redacted” to maintenance receipts; runner logs are separate raw records.
- Root `SKILL.md`, inert `type: skill` Sources, and opt-in Maintenance Adapters are separate concepts. An assistant's ability to load a folder/skill depends on that assistant and user choice. Avoid universal automatic activation or auto-install claims.
- Base metadata (`BaseLayout.astro:10-18,28-47`) makes no additional unsupported capability claim. Canonical site and Starlight configuration agree on `https://www.liner.sh`. Footer copy repeats the free/local/telemetry boundary problem identified above.

## Coverage and limits

Reviewed homepage hero/proof/examples/workflow/maintenance/trust/install/footer, shared header/copy components and metadata, all ten docs pages, Starlight nav/social configuration, changelog generation plus its source history, robots/manifest text, and the actual social PNG plus its SVG source. Inspected the corresponding launcher, source types/handlers, compile/output, share/import, maintenance, configuration, runner, onboarding and key-handler code. No real Project library, private Source bodies, credentials, paid provider, CLI mutation, or Git mutation was used. Network reads were limited to public GitHub metadata and npm package metadata.

This is source-truth evidence, not proof of clean-machine installation, successful provider authentication, cross-platform runtime behavior, physical keyboard behavior, browser layout, or deploy status. The implementation lead owns rendered-page and performance checks. Final edited copy must be rechecked against this report, especially command grouping, privacy boundaries, and changelog output.

## Copy deck factual review

Reviewed the full `docs/liner-website-copy-deck.md` after the writer's READY message, including all ten complete route bodies, metadata/footer, social assets, 47 curated changes across twelve versions, and new clipboard/motion/search/404 strings. Independently checked `npm view linersh versions --json`: all twelve included versions exist; 1.0.1 and 1.0.3 lack source release notes and are correctly not invented. The new UI-state strings are accepted as proposed interface states, conditional on the implementation actually providing the announced behavior. All substantive product claims are accepted except the following six exact corrections, handed to the final language editor. Line numbers refer to the roughly 1,597-line reviewed deck and may move with editing.

1. **Useful flags, lines 822-829:** split independent examples into separate snippets. One Copy currently runs three compiles, an export, list operations, and non-interactive `cache clear --yes`. Remove or segregate the destructive cache action. Evidence: `CommandSnippet.astro:21,45-46` copies the entire string. This extends the command-grouping finding above across the deliverable.
2. **Export options, lines 1033-1037:** split each alternative flag into its own snippet. Copy currently executes all variants, repeatedly overwrites the archive, and ends with an unfiltered export to Desktop. Evidence: `src/liner/share.py:46-54` selects the same default path and opens the zip with write mode.
3. **Troubleshooting resolver, line 1240:** use the actual launcher order: `LINER_BIN` → repo-local `.venv` → bundled platform package → separate `liner` on `PATH`. Evidence: `packages/tui/bin/liner.js:191-204`.
4. **Isolated trial, lines 386-403; troubleshooting link around 1244:** remove the public trial subsection and link, or restrict the isolation claim. Overriding HOME does not clear inherited explicit Playwright/cache/provider paths. Evidence: `src/liner/playwright_env.py:15-21` honors `PLAYWRIGHT_BROWSERS_PATH` and `XDG_CACHE_HOME`; `packages/go-tui/internal/airunner/profile.go:198-215` honors native/Liner-specific provider homes. Do not promise normal caches or provider profiles are untouched merely because HOME is temporary.
5. **Adapter install prose/snippets around 379-380,633-642 and CLI table 783-785:** `--yes` does not display the exact plan before writing. `src/liner/cli.py:143-158` emits the plan only without approval and exits 2; with `--yes` it applies and prints the receipt afterward. `adapters inspect` (`111-130`) prints status/compatibility/target, not file effects. Provide `liner adapters install codex` as a separate no-write preview, then `liner adapters install codex --yes` as the deliberate apply; use the analogous Claude/update/remove sequence. Describe table commands without `--yes` as previews, with `--yes` required to apply.
6. **Copyable maintenance inspection around 652-653:** replace literal `<project>` with the concrete `mobile-design-foundations` example and explain its path. Angle-bracket placeholders are shell redirection when pasted; command-reference tables may retain documented placeholder notation.

No website source was changed by this auditor. The final editor owns those six corrections and the one Humanizer pass. The implementation lead owns their integration, actual rendered behavior, and verification. Factual review is complete; remaining work is six bounded editor corrections, not an open product question.
