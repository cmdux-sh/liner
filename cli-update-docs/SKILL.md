---
name: curating-mixtapes
description: Use when the user wants to build a "mixtape" — a curated bundle of sources (videos, articles, papers, specs) assembled as AI context for a specific job-to-be-done. Triggers include "help me make a mixtape on X", "curate sources on Y for AI", "build a context bundle for Z", "let's research X with Liner", "make me a tape on Y", "help me build a Liner mixtape". Drives the eight-phase Liner curation methodology end-to-end and produces a mixtape project folder — `tape.yaml`, `synthesis.md`, and `working/*` notes — ready for `liner compile`. Defaults to quick mode (the AI does the heavy lifting, the user confirms at two gates); methodology mode is available when the user wants to engage substantively at every step or is targeting library contribution.
---

# Curating Mixtapes

This skill implements the Liner curation methodology (CURATION.md v2.0). A mixtape is a curated bundle of sources designed to make AI conversations meaningfully better within a specific domain. The bar: a good mixtape demonstrably improves AI output on tasks within its job-to-be-done.

The output of this skill is a project folder ready for `liner compile`. The skill does not compile.

**Runtime requirement:** this skill assumes filesystem access. It reads companion files from its own directory at specific phases and writes artifacts to a project folder. Designed for execution inside Claude Code or as a subprocess driven by the Liner TUI. Not designed for pure-chat use.

## Terminology — get these right

- **Tape file** (`tape.yaml`) — the recipe. Sources with notes and sections.
- **Mixtape** — the project folder. Recipe + synthesis + working notes + source content + master file.
- **Consumable mixtape** — `MIXTAPE.md` plus the `sources/*.md` files. What gets pasted into an AI conversation. Produced by `liner compile`.

Tape ≠ mixtape. The recipe is small; the mixtape is the realized artifact.

## What this skill produces

A project folder named for the topic (e.g., `mobile-design-foundations/`) containing:

- `tape.yaml` — the recipe, conforming to MIXTAPE-FORMAT v1 (with v1.1 additions: `mode`, `jtbd`, `local_file` sources, optional `render` field on web sources)
- `synthesis.md` — the curator's distilled understanding of the domain
- `working/01-jtbd-and-knowledge-map.md` — framing artifacts
- `working/02-candidate-longlist.md` — the unfiltered candidate pool
- `working/03-evaluation.yaml` — keep/trim/drop decisions with rationales and ratings
- `working/04-quality-checks.md` — redundancy, coverage, disagreement, and framing-gap findings

`sources/` and `MIXTAPE.md` are produced later by `liner compile`. The optional `working/05-empirical-test.md` file is produced only when the curator runs the methodology validation step; it is not a driven TUI phase.

When the curator has supplied local material (book chapters, exported PDFs, Reader-Mode-saved articles), the skill may write `local_file` entries in `tape.yaml` pointing at files the curator has placed under `personal/`. This is a methodology-mode affordance — the curator's local context belongs to them, not the public web, and the `local_file` source type makes it first-class. Library contributions still exclude `local_file` sources; the skill should not invent them or suggest paths the curator hasn't supplied.

## Companion files

This skill is split across files in its own directory. Read each companion file at the start of the phase that uses it. Do not skip the read.

- `source-finding-tactics.md` — read before Phase 2
- `source-quality-hierarchy.md` — read before Phase 4
- `curator-notes.md` — read before Phase 4
- `quality-check-tests.md` — read before Phase 5
- `synthesis-guidance.md` — read before Phase 6

The orchestration logic, the two modes, the two gates, the AI-bias list, and the failure-mode list stay in this file because they apply across the whole lifecycle.

## Two modes

Default to **quick mode** unless the user says they want methodology mode or library contribution.

**Quick mode.** The AI does maximum work. Defaults at every gate are "looks good, continue." Total wall-clock target: 15–30 minutes. Output quality: good enough for personal use.

**Methodology mode.** The curator engages substantively at every phase. Revises the knowledge map. Reviews evaluation rationales. Writes or substantially rewrites the synthesis in their own voice. Runs the empirical test. Several hours over one or more sessions. Output quality: library-eligible.

Same lifecycle. Same artifacts. The difference is how much human judgment shows up in the working notes and the synthesis.

## Two human-in-the-loop gates

Two gates exist regardless of mode. Quick mode defaults them to "continue." Methodology mode pauses for substantive review.

- **Gate 1 — Candidate confirmation** (after Phase 2, before Phase 3). Show the candidate long-list. The user confirms before any fetching happens. Saves wasted network calls; catches missing categories early.
- **Gate 2 — Evaluation review** (after Phase 4, before Phase 6). Show keep/trim/drop with rationales and quality-check findings. The user confirms before synthesis is drafted.

## The lifecycle

Run these phases in order. Each phase's output feeds the next. Do not reorder.

### Phase 1 — Framing

Define the job-to-be-done and sketch a knowledge map.

**The JTBD is a Job Story, not a topic.** Required form: `When [circumstance], I want [motivation], so I can [outcome].` All three slots required. "Mobile design" is a topic. *"When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic."* is a JTBD. Three different Job Stories in the same topic area produce three different mixtapes.

Elicit the JTBD using the [JTBD Master Prompt](JTBD-MASTER-PROMPT.md) — drop its body into this phase as your interview script. It opens with a single short context-sharing question (offering to read project material if the user has any), reads what's shared if anything, then either runs a budgeted interview (five questions max) or descends the scope ladder from what was inferred. It drafts silently, checks the draft against a six-test rubric, and emits one sentence. If the user pushes back, take the correction and re-run the rubric — don't defend.

If you can't use the master prompt as-is (e.g., you're in a context where pasting it doesn't make sense), the minimum bar is: refuse any answer that's a bare topic, an aspiration ("be better at X"), a slot-missing fragment, or two jobs joined by "and." Stay on framing until the JTBD is a concrete single-sentence Job Story.

**Knowledge map.** A short outline of the conceptual territory the corpus needs to cover. Four to eight sections, each with a few sub-areas. Treat it as a hypothesis — it will get revised during research.

Write both to `working/01-jtbd-and-knowledge-map.md`. In quick mode: draft, show, get a one-line confirmation. In methodology mode: iterate with the curator until both feel right.

### Phase 2 — Candidate discovery

**Read `source-finding-tactics.md` before starting this phase.**

Generate a long list of candidate sources across the knowledge map. URLs and titles only — no fetching yet. Aim wide: two to four times the eventual kept count. This phase is recall, not precision.

Verify each candidate exists at the URL given. AI is prone to hallucinating plausible-sounding URLs. Treat unverified URLs as candidates only until they're confirmed.

Write to `working/02-candidate-longlist.md`, grouped by knowledge-map section. For each candidate: URL, title, one-line reason it's a candidate.

**At Gate 1, show the long-list and ask:** "Anything obviously missing? Anything obviously wrong? Confirm to start fetching." In quick mode the default is "looks good, continue." In methodology mode, expect real edits.

### Phase 3 — Fetching

Pull content for every confirmed candidate.

For each URL, retrieve the full text. YouTube: transcript. Articles, papers, docs: extracted body text. Skim each as it comes back to confirm the URL contains what the title promised.

If a URL fails to fetch (paywall, dead link, no transcript, geographic block): note the failure, decide whether the source is critical enough to chase via an alternate route (Wayback Machine, alternate venue), and either find a substitute or drop it. Don't silently lose candidates.

A successful transcript fetch is not the same as a high-value source. Many tutorial videos transcribe cleanly but contain no transferable content. The next phase handles that judgment.

Don't write fetched content into the project folder yet. That happens at compile time, only for kept sources.

### Phase 4 — Evaluation

**Read `source-quality-hierarchy.md` and `curator-notes.md` before starting this phase.**

Read every fetched source against the JTBD. Decide keep / trim / drop. Rate kept sources 1–5. Draft curator notes for kept and trim sources.

Write decisions to `working/03-evaluation.yaml`:

```yaml
candidates:
  - url: https://example.com/source
    title: Source title
    decision: kept            # kept | trim | dropped
    rating: 5                 # 1-5; required for kept and trim
    section: foundations
    rationale: One-sentence reason this decision was made.
```

Drafted curator notes land in `tape.yaml` at Phase 7. Write them now in their final form; don't postpone the work to assembly.

### Phase 5 — Quality checks

**Read `quality-check-tests.md` before starting this phase.**

Step back from the keep-pile. Run the four tests. Document findings in `working/04-quality-checks.md`.

If a test fails, fix it before moving on. Don't proceed to synthesis with known coverage or framing gaps.

*Tip: this phase benefits from running with fresh attention. If the conversation has been long, consider starting a separate chat with just the JTBD, knowledge map, and the kept-source list (titles + notes only) to run the checks. Optional, not required by the methodology.*

**At Gate 2, show the final keep-list, the curator notes, and the quality-check findings. Ask:** "Confirm to proceed to synthesis." Quick mode default: continue. Methodology mode: expect substantive edits.

### Phase 6 — Synthesis

**Read `synthesis-guidance.md` before starting this phase.**

Draft `synthesis.md`. The synthesis is the curator's distilled understanding of the domain expressed as continuous prose. 800–2000 words.

Quick mode: draft, show, default to "looks good, save it." The curator lightly edits if they want.

Methodology mode: draft a starting point. Expect the curator to rewrite it substantially in their own voice. Library mixtapes need a substantially curator-written synthesis.

### Phase 7 — Assembly

Write `tape.yaml`.

**Required top-level fields:** `title`, `description`, `curator`, `version: 1`, `mode` (`quick` or `methodology`), `sources` (non-empty list).

**Recommended top-level fields:** `tags`, `created` (ISO date), `jtbd`, `methodology_version: "2.0"`, `license`.

**Each kept source needs:**

- `type` — `youtube` or `web` (v1 supports only these two; gated content, local files, and pasted content are v2)
- `url`
- `note` — the curator note drafted in Phase 4
- `section` — matches a knowledge-map section
- `priority: optional` — only for sources the user can skip with `--skip-optional`

Order sources within each section by importance (most important first). Order sections by reading order (foundations first, applications later).

**Validate before declaring done:**

- YAML parses
- All required fields present
- `version: 1`
- `mode` is `quick` or `methodology`
- `sources` is non-empty
- Every source has `type` and `url`
- Every `type` is `youtube` or `web`
- Every `url` parses

If validation fails, fix and re-check before showing the user.

### Empirical test — methodology validation (optional in quick mode, required for library contribution)

The only real test of whether the mixtape works.

1. Compile: `liner compile <project-folder>`
2. Open a fresh AI conversation. Don't paste the mixtape. Ask a substantive question that should benefit from the JTBD. Save the response.
3. Open another fresh AI conversation. Paste `MIXTAPE.md` plus relevant `sources/*.md` files. Ask the same question. Save the response.
4. Compare. Is the with-mixtape response meaningfully better — more specific, more accurate, more nuanced, less generic?

Write findings to `working/05-empirical-test.md`: the question, the two responses (or summaries), the honest verdict. If the mixtape didn't help: what's missing? Back to Phase 2 or 5.

If the answers are similar, the mixtape isn't earning its place. Either the sources are too generic, the synthesis isn't doing enough work, or the JTBD is too broad. Don't skip this for library contributions.

## AI biases this skill must compensate for

The skill is the AI running the methodology. The AI has systematic biases that produce worse mixtapes if left uncorrected. Watch for these in your own work throughout every phase — these are not phase-specific.

- **Popularity bias.** AI surfaces popular and recent sources at the expense of canonical or older ones. Search deliberately for primary sources. The Apple HIG, the W3C spec, the canonical paper — these often won't appear in generic search results and need to be found directly.
- **URL hallucination.** AI invents URLs that sound plausible but don't exist. Verify every URL with a web fetch before adding it to the candidate list. A 404 candidate is worse than a missing one.
- **Generic curator notes.** AI drafts notes that say "this is a useful resource" without specifying role, location of value, or limitations. Compensate by writing each note against the three-thing template in `curator-notes.md`.
- **Primary-document blindness.** AI gravitates to secondary commentary over primary sources. If a domain has a spec, a standard, or a foundational paper, search for it specifically. Don't accept commentary as a substitute.
- **Framing-gap blindness.** AI can't see what it didn't search for. The framing-gap test in Phase 5 is the methodological correction for this. Run it with intent — not as a checkbox, as a genuine "what am I missing" question.
- **Synthesis-as-recap.** AI-drafted synthesis tends to recap source content rather than distill the curator's framing of the domain. Push toward principles, contested questions, and framework distinctions — not "Source 1 says X; Source 2 says Y."

If you find yourself producing any of these patterns, stop and correct.

## Final delivery

When all phases complete, the project folder contains:

```
<topic-slug>/
├── tape.yaml
├── synthesis.md
└── working/
    ├── 01-jtbd-and-knowledge-map.md
    ├── 02-candidate-longlist.md
    ├── 03-evaluation.yaml
    ├── 04-quality-checks.md
    └── 05-empirical-test.md   # if empirical validation was run
```

Tell the user the folder is ready to compile with `liner compile <topic-slug>`. The compile step produces `MIXTAPE.md` and the `sources/` folder.

## What this skill does NOT do

- **Does not run `liner compile`.** That's the user's call, with the user's tools.
- **Does not fetch from gated or local sources.** v1 supports only public `youtube` and `web` URLs. Personal sources (book chapters, gated articles, local files) are real and often high-value, but v1 doesn't support them.
- **Does not maintain the shared content cache.** That's the CLI's job.
- **Does not publish to the library.** Library contributions happen via PR to the Liner repo, in methodology mode, after passing the empirical test.

## Common failure modes

Patterns that produce bad mixtapes. Watch for them in your own work:

- **The greatest-hits mixtape.** A list of the most famous resources. Looks impressive, adds little value because the AI already knows about famous resources from training. Curate for *non-obvious* sources.
- **The unfocused mixtape.** No clear JTBD. Tries to cover the field. Shallow on everything, deep on nothing.
- **The author-tribute mixtape.** Three of five sources by the same person because the curator is a fan. Narrows perspective, reduces synthesis quality.
- **The transcript dump.** All video transcripts, no articles, no specs, no primary sources. Mix source types.
- **The noteless mixtape.** Sources without curator notes. The curator either was lazy or didn't have a reason. Both are problems.
- **The aspirational mixtape.** Sources the curator thinks people *should* read, not sources that help with the JTBD. Mixtapes are tools, not reading recommendations.
- **The single-pass mixtape.** Curated without a framing-gap test. Well-covered within one narrow framing, blind to perspectives the curator didn't think to look for.

If the mixtape this skill produces fits any of these patterns, return to the relevant phase and fix it.
