---
name: curating-mixtapes
description: Use when the user wants to build a "mixtape" — a curated bundle of sources (videos, articles, papers, specs) assembled as AI context for one hyper-specific future-agent task. Triggers include "help me make a mixtape on X", "curate sources on Y for AI", "build a context bundle for Z", "let's research X with Liner", "make me a tape on Y", "help me build a Liner mixtape", or "I want this Liner to help my AI agent do X". Drives the eight-phase Liner curation methodology end-to-end and produces a corpus folder — `tape.yaml`, `synthesis.md`, and `working/*` notes, under `mixtape/` in v2 projects — ready for `liner compile`. Defaults to quick mode (the AI does the heavy lifting, the user confirms at two gates); methodology mode is available when the user wants to engage substantively at every step or is targeting library contribution.
---

# Curating Mixtapes

This skill implements the Liner curation methodology (CURATION.md v2.0). A mixtape is a curated bundle of sources designed to make future AI sessions meaningfully better at one hyper-specific task. The bar: a good mixtape demonstrably improves agent output inside the capability it was built for.

The output of this skill is a project folder ready for `liner compile`. The skill does not compile.

This is an initial curation surface, not a maintenance writer. After a Liner
Project has identity-bearing Project or Source state, supported changes must go
through the installed CLI's current `liner project guidance` contract. Do not
use this skill to edit canonical Project files as a maintenance shortcut.

**Runtime requirement:** this skill assumes filesystem access. It reads companion files from its own directory at specific phases and writes artifacts to a project folder. Designed for execution inside Claude Code or as a subprocess driven by the Liner TUI. Not designed for pure-chat use.

## Terminology — get these right

- **Tape file** (`tape.yaml`) — the recipe. Sources with notes and sections.
- **Mixtape** — the corpus folder. In v2 projects it lives under `mixtape/`.
- **Consumable mixtape** — `MIXTAPE.md` plus the `sources/*.md` files inside the corpus folder. What gets pasted into an AI conversation. Produced by `liner compile`.

Tape ≠ mixtape. The recipe is small; the mixtape is the realized artifact.

## What this skill produces

A project folder named for the topic (e.g., `mobile-design-foundations/`) containing a corpus folder. In v2 projects, these files live under `mixtape/`; legacy or standalone skill runs may use the project root:

- `tape.yaml` — the recipe, conforming to MIXTAPE-FORMAT v1 (with v1.1 additions: `mode`, `jtbd`, `local_file` sources, optional `render` field on web sources)
- `synthesis.md` — the curator's distilled understanding of the domain
- `working/01-jtbd-and-knowledge-map.md` — framing artifacts: JTBD, capability brief, knowledge map, and required source roles
- `working/02-candidate-longlist.md` — the unfiltered candidate pool
- `working/03-evaluation.yaml` — keep/trim/drop decisions with rationales, ratings, content evidence, and source-role mapping
- `working/04-quality-checks.md` — redundancy, coverage, disagreement, framing-gap, source-kind, note-quality, and source-role findings
- `working/05-operating-fit-audit.md` — optional improvement recommendation when the corpus is not yet strong enough for the Operating Layer

`sources/` and `MIXTAPE.md` are produced later by `liner compile` in the same corpus folder. The optional `working/06-empirical-test.md` file is produced only when the curator runs the methodology validation step; it is not a driven TUI phase.

When the curator has supplied local material (book chapters, exported PDFs, Reader-Mode-saved articles), the skill may write `local_file` entries in `tape.yaml` pointing at files the curator has placed under `local-sources/` or the legacy `personal/` folder. This is a methodology-mode affordance — the curator's local context belongs to them, not the public web, and the `local_file` source type makes it first-class. Library contributions still exclude `local_file` sources; the skill should not invent them or suggest paths the curator hasn't supplied.

## Companion files

This skill is split across files in its own directory. Read each companion file at the start of the phase that uses it. Do not skip the read.

- `source-finding-tactics.md` — read before Phase 2
- `source-quality-hierarchy.md` — read before Phase 4
- `curator-notes.md` — read before Phase 4
- `quality-check-tests.md` — read before Phase 5
- `synthesis-guidance.md` — read before Phase 6

The orchestration logic, the two modes, the review gates, the AI-bias list, and the failure-mode list stay in this file because they apply across the whole lifecycle.

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

Define the reusable capability and sketch a knowledge map.

The user-facing question is: **What do you want this Liner to help your AI agent do?** The user can answer in plain language. Do not make them supply source categories, research lanes, or a formal Job Story.

Your job in Phase 1 is to translate that plain-language goal into a Capability Brief:

- what future AI sessions should be able to do,
- what outputs, decisions, or behaviors the resource should support,
- the internal job-to-be-done that will let later phases judge fit,
- research lanes inferred from the goal,
- source requirements and exclusions,
- runtime behavior for the future agent, including when it should ask the user clarifying questions.

Use these exact level-three headings inside the Capability Brief so the Operating Layer can promote the contract reliably: `### Reusable AI capability`, `### Exact runtime output contract`, `### Internal job-to-be-done`, `### Inferred research lanes`, `### Required source roles`, `### Source exclusions`, and `### Runtime autonomy, abstention, and escalation`.

The internal JTBD should be hyper-specific. "SEO" is a topic. "SEO keyword research for a mental-health startup specialized in brain surgery" is a capability narrow enough to research. If the user's goal is still broad, ask targeted questions and descend until the capability is specific enough that a corpus built for it could outperform generic model knowledge.

**Capability brief.** A short statement of the concrete capability this mixtape should give a downstream AI agent. This is not a persona or project name. It is the work the corpus must make the agent better at.

**Knowledge map.** A short outline of the conceptual territory the corpus needs to cover. Four to eight sections, each with a few sub-areas. Treat it as a hypothesis — it will get revised during research.

**Required source roles.** The evidence plan for the corpus. Name the kinds of sources the downstream agent needs before the mixtape can be trusted. Each role must include: role name, why it matters, what good evidence looks like, and minimum coverage. These are inferred from the user's desired capability, not hardcoded categories.

**Capability patterns.** If the user's goal implies a recurring evidence shape, name it in the Capability Brief. The most important current pattern is `reference-translation`: the future agent receives images, moodboards, visual references, inspiration, style examples, artifacts, or examples from one medium/domain and must turn them into another output. For this pattern, source roles must separate input/reference-domain interpretation, cross-domain translation method, target-output constraints, critique/clarification, and caller handoff language. Target-output implementation sources are bridge evidence; they cannot satisfy the reference-domain or translation-method roles by themselves.

Write the Capability Brief, internal JTBD, research lanes, knowledge map, and Required source roles to `working/01-jtbd-and-knowledge-map.md`. In quick mode: draft, show, get a one-line confirmation. In methodology mode: iterate with the curator until the capability brief, knowledge map, and source roles feel right.

### Phase 2 — Candidate discovery

**Read `source-finding-tactics.md` before starting this phase.**

Generate a long list of candidate sources across the knowledge map and required source roles. URLs and titles only — no fetching yet. Aim wide enough for the evidence contract: roughly two to four times the eventual kept count, with the actual candidate count derived from the required source roles, capability pattern, and source ecology. This phase is recall, not precision.

Verify each candidate exists at the URL given. AI is prone to hallucinating plausible-sounding URLs. Treat unverified URLs as candidates only until they're confirmed.

Write to `working/02-candidate-longlist.md`, grouped by knowledge-map section. For each candidate: URL, title, required source role served, and one-line reason it's a candidate.

**At Gate 1, show the long-list and ask:** "Anything obviously missing? Anything obviously wrong? Confirm to start fetching." In quick mode the default is "looks good, continue." In methodology mode, expect real edits.

### Phase 3 — Fetching

Pull content for every confirmed candidate.

For each URL, retrieve the full text. YouTube: transcript. Articles, papers, docs: extracted body text. Skim each as it comes back to confirm the URL contains what the title promised.

If a URL fails to fetch (paywall, dead link, no transcript, geographic block): note the failure, decide whether the source is critical enough to chase via an alternate route (Wayback Machine, alternate venue), and either find a substitute or drop it. Don't silently lose candidates.

Bound the chase: try one direct fetch/read and at most one recovery attempt per candidate. After two failed retrieval attempts total, stop searching for that candidate and carry the failure into Phase 4 as part of the decision rationale.

A successful transcript fetch is not the same as a high-value source. Many tutorial videos transcribe cleanly but contain no transferable content. The next phase handles that judgment.

Don't write fetched content into the project folder yet. That happens at compile time, only for kept sources.

### Phase 4 — Evaluation

**Read `source-quality-hierarchy.md` and `curator-notes.md` before starting this phase.**

Read every fetched source against the JTBD. Decide keep / trim / drop. Rate kept sources 1–5. Draft curator notes for kept and trim sources.

If content was unavailable after the bounded fetch attempts, still write a decision for that candidate. Usually choose `dropped`. Do not mark a source `kept` or `trim` from title, URL, search snippet, crawler metadata, or model memory alone.

Every `kept` or `trim` source must carry evidence fields in `working/03-evaluation.yaml`:

- `fetch_status: readable | partial` — `readable` means the substantive article, transcript, PDF body, or local content was retrieved. `partial` means enough real source content was retrieved to judge, such as an abstract plus substantial excerpt. `metadata_only` and `unavailable` candidates must be `dropped`.
- `content_quality: high | medium` — low-quality content, SEO filler, AI-generated sludge, generic listicles, homepages, galleries without rationale, or shallow transcripts must be `dropped`.
- `evidence:` — at least two content-specific bullets from the fetched/read source. These are concrete claims, examples, sections, transcript moments, methods, or limitations seen in the content. Search snippets and titles do not count.

Write decisions incrementally as section-sized fragments in `working/evaluation-decisions/`, then write or let Liner assemble `working/03-evaluation.yaml`. Do not hold all decisions in conversation memory until the end.

```yaml
candidates:
  - url: https://example.com/source
    title: Source title
    decision: kept            # kept | trim | dropped
    rating: 5                 # 1-5; required for kept and trim
    fetch_status: readable    # readable | partial; metadata_only/unavailable are dropped
    content_quality: high     # high | medium; low is dropped
    source_role: canonical method
    evidence:
      - First content-specific claim, section, example, or method retrieved from the source.
      - Second content-specific detail proving the source was read.
    section: foundations
    rationale: One-sentence reason this decision was made.
```

Drafted curator notes land in `tape.yaml` at Phase 7. Write them now in their final form; don't postpone the work to assembly.

### Phase 5 — Quality checks

**Read `quality-check-tests.md` before starting this phase.**

Step back from the keep-pile. Run the core-action test and the eight standard tests. Document findings in `working/04-quality-checks.md`.

If a test fails, fix it before moving on when the fix is small enough for the quality phase. If the failure points to a wider source-role gap, write `working/05-operating-fit-audit.md` with `status: improvement_recommended`, the gap, why it matters, concrete search lanes, and custom-source suggestions. Do not call the corpus "ready with limitation"; let Liner offer the user an improvement pass or a clear Skip.

If the bounded Quality audit backfills a source, add the verified URL to `working/02-candidate-longlist.md` before adding it to `working/03-evaluation.yaml`. Parse both artifacts and confirm their candidate counts still match before reporting Phase 5 complete. The longlist remains the complete candidate ledger even when a candidate is discovered after Gate 1.

*Tip: this phase benefits from running with fresh attention. If the conversation has been long, consider starting a separate chat with just the JTBD, capability brief, required source roles, knowledge map, and the kept-source list (titles + notes only) to run the checks. Optional, not required by the methodology.*

**At Gate 2, show the final keep-list, the curator notes, and the quality-check findings. Ask:** "Confirm to proceed to synthesis." Quick mode default: continue. Methodology mode: expect substantive edits.

### Phase 6 — Synthesis

**Read `synthesis-guidance.md` before starting this phase.**

Draft `synthesis.md`. The synthesis is the curator's distilled understanding of the domain expressed as continuous prose. 800–2000 words. It must include `## Generative rules` and `## Stances this corpus takes` so the consuming AI gets explicit operating rules and corpus stances before reading individual sources.

Quick mode: draft, show, default to "looks good, save it." The curator lightly edits if they want.

Methodology mode: draft a starting point. Expect the curator to rewrite it substantially in their own voice. Library mixtapes need a substantially curator-written synthesis.

### Phase 7 — Assembly

Write `tape.yaml`.

**Required top-level fields:** `title`, `description`, `curator`, `version: 1`, `mode` (`quick` or `methodology`), `sources` (non-empty list).

**Recommended top-level fields:** `tags`, `created` (ISO date), `jtbd`, `methodology_version: "2.0"`, `license`.

**Each kept source needs:**

- `type` — `youtube`, `web`, `local_file`, or `skill`
- `url` for `youtube` and `web`; `path` for `local_file` or local/installed `skill` sources; `citation` for `local_file`
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
- Every source has `type`, `note`, and `section`
- Every `type` is `youtube`, `web`, `local_file`, or `skill`
- Every `youtube` or `web` source has a parseable `url`
- Every `local_file` source has `path` and `citation`
- Every local or installed `skill` Source has `path`; a remote skill Source has
  a parseable `url`

If validation fails, fix and re-check before showing the user.

### Empirical test — methodology validation (optional in quick mode, required for library contribution)

The only real test of whether the mixtape works.

1. Compile: `liner compile <project-folder>`
2. Open a fresh AI conversation. Don't paste the mixtape. Ask a substantive question that should benefit from the JTBD. Save the response.
3. Open another fresh AI conversation. Paste `mixtape/MIXTAPE.md` plus relevant `mixtape/sources/*.md` files. Ask the same question. Save the response.
4. Compare. Is the with-mixtape response meaningfully better — more specific, more accurate, more nuanced, less generic?

Write findings to `working/06-empirical-test.md`: the question, the two responses (or summaries), the honest verdict. If the mixtape didn't help: what's missing? Back to Phase 2 or 5.

If the answers are similar, the mixtape isn't earning its place. Either the sources are too generic, the synthesis isn't doing enough work, or the JTBD is too broad. Don't skip this for library contributions.

## AI biases this skill must compensate for

The skill is the AI running the methodology. The AI has systematic biases that produce worse mixtapes if left uncorrected. Watch for these in your own work throughout every phase — these are not phase-specific.

- **Popularity bias.** AI surfaces popular and recent sources at the expense of canonical or older ones. Search deliberately for primary sources. The Apple HIG, the W3C spec, the canonical paper — these often won't appear in generic search results and need to be found directly.
- **URL hallucination.** AI invents URLs that sound plausible but don't exist. Verify every URL with a web fetch before adding it to the candidate list. A 404 candidate is worse than a missing one.
- **Generic curator notes.** AI drafts notes that say "this is a useful resource" without specifying role, location of value, or limitations. Compensate by writing each note against the three-thing template in `curator-notes.md`.
- **Primary-document blindness.** AI gravitates to secondary commentary over primary sources. If a domain has a spec, a standard, or a foundational paper, search for it specifically. Don't accept commentary as a substitute.
- **Framing-gap blindness.** AI can't see what it didn't search for. The framing-gap test in Phase 5 is the methodological correction for this. Run it with intent — not as a checkbox, as a genuine "what am I missing" question.
- **Source-role blindness.** AI can fill every knowledge-map section while missing an evidence role the job depends on. Use the required source roles from Phase 1 as a contract in Phase 2 and an audit checklist in Phase 5.
- **Pattern flattening.** AI turns specialized capabilities into generic topic packs. A reference-translation project becomes "web design"; a medical SEO project becomes "SEO"; a legal drafting project becomes "writing." When Phase 1 names a capability pattern, Phase 2 and Phase 5 must preserve that pattern-specific evidence contract.
- **Synthesis-as-recap.** AI-drafted synthesis tends to recap source content rather than distill the curator's framing of the domain. Push toward principles, contested questions, and framework distinctions — not "Source 1 says X; Source 2 says Y."

If you find yourself producing any of these patterns, stop and correct.

## Final delivery

When all phases complete, the project folder contains:

```
<topic-slug>/
└── mixtape/
    ├── tape.yaml
    ├── synthesis.md
    └── working/
        ├── 01-jtbd-and-knowledge-map.md
        ├── 02-candidate-longlist.md
        ├── 03-evaluation.yaml
        ├── 04-quality-checks.md
        ├── 05-operating-fit-audit.md   # if an improvement pass is recommended
        └── 06-empirical-test.md        # if empirical validation was run
```

Tell the user the folder is ready to compile with `liner compile <topic-slug>`. The compile step produces `mixtape/MIXTAPE.md` and `mixtape/sources/`.

## What this skill does NOT do

- **Does not run `liner compile`.** That's the user's call, with the user's tools.
- **Does not fetch from gated accounts or private systems.** v1 supports public `youtube` and `web` URLs plus curator-supplied `local_file` and `skill` sources. The skill may reference local files the curator supplied, but it must not invent private paths or bypass access controls.
- **Does not maintain the shared content cache.** That's the CLI's job.
- **Does not maintain an existing Liner Project.** Use Liner Core directly or
  install the optional Maintenance Adapter, which delegates to
  `liner project guidance`.
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
- **The role-missing mixtape.** Sections appear covered, but a required source role from Phase 1 has no fetched, useful evidence.

If the mixtape this skill produces fits any of these patterns, return to the relevant phase and fix it.
