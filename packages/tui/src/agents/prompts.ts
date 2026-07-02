import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { LonglistCandidateGroup } from "./candidate-longlist.js";
import type { PhaseId } from "../phases.js";
import type { Tape } from "../types.js";

/**
 * Companion files referenced by each phase per SKILL.md. The agent is told
 * which ones to read at the top of the relevant phase.
 */
const COMPANIONS: Partial<Record<PhaseId, string[]>> = {
  candidates: ["source-finding-tactics.md"],
  evaluation: ["source-quality-hierarchy.md", "curator-notes.md"],
  quality: ["quality-check-tests.md"],
  synthesis: ["synthesis-guidance.md"],
};

export const WORKSPACE_DISCIPLINE = `# Workspace discipline

- Treat the project folder as a mixtape artifact folder, not a git worktree, unless a \`.git\` directory is actually present there or you are explicitly inspecting a source repository that belongs to the corpus.
- Do not run \`git diff\`, \`git status\`, or repository introspection in the mixtape project folder just to verify generated artifacts. Use file reads and parsers against the specific artifact paths you wrote.
- For validation snippets, prefer Node.js or Python. If you use Ruby, stay compatible with Ruby 2.6 and do not use newer helpers such as \`Array#tally\`.
- Keep shell checks narrow: read or parse the required artifacts, then stop.`;

const PHASE_INSTRUCTIONS: Partial<Record<PhaseId, string>> = {
  framing: `## Phase 1 — Framing

Define the reusable AI capability, derive the internal job-to-be-done, and sketch the knowledge map.

- The user provided a plain-language capability goal in \`tape.yaml.jtbd\` and \`working/01-jtbd-and-knowledge-map.md\`: what this Liner should help a future AI agent do. Read both.
- Treat that field as user intent, not as a finished formal JTBD. The user should not have to know research lanes, source categories, or Job Story syntax.
- **Read \`tape.yaml.jtbd_clarifications\` if present.** Those are the user's answers to sharpening questions about capability boundaries, output behavior, quality anchors, constraints, and future-agent autonomy. Use them to derive the research plan.
- Write a Capability Brief before the knowledge map: what future AI sessions should be able to do, what outputs/decisions/behaviors it supports, the internal JTBD, inferred research lanes, required source roles/exclusions, and runtime behavior for the future agent.
- The Capability Brief must include a **Required source roles** subsection. Source roles are capability-specific evidence jobs, not generic kinds. Derive them from what the future AI must do. Typical roles include: core-action methods, domain/primary authority, worked examples or case studies, implementation constraints, critique/validation evidence, dissenting practitioner voice, and domain-specific data. Delete roles that do not fit; add roles the capability clearly needs.
- For each required source role, name why it matters, what good evidence looks like, and the minimum kept/trim sources the corpus needs. Defaults: at least 2 strong sources for required roles; at least 3 substantive worked examples/case studies when the capability depends on taste, visual output, craft judgment, or recognizing quality; primary/official sources for legal, medical, financial, or safety-sensitive authority roles.
- Detect specialized **capability patterns** and write them into the Capability Brief. If the goal involves images, moodboards, visual references, inspiration, style, art direction, examples, or translating one medium/domain into another output, mark \`Capability pattern: reference-translation\`. For that pattern, source roles must separate: input/reference-domain interpretation, cross-domain translation method, target-output constraints, critique/clarification, and caller handoff language. Output-medium implementation sources are bridge evidence; they cannot satisfy the input/reference-domain or translation-method roles by themselves.
- For \`reference-translation\`, the brief must name the runtime output contract. It should say what the future agent returns to the caller, such as observations, interpretation, carry-forward principles, product/interface vocabulary, implementation constraints, avoid-list, validation checks, and clarification questions.
- The internal JTBD must be hyper-specific, not a broad topic. "SEO" is too broad; "SEO keyword research for a mental-health startup specialized in brain surgery" is specific enough to research.
- If the capability is still too broad after clarifications, push back in your output and ask the user to revise.
- The knowledge map is 4–8 sections, each with a few sub-areas. Treat it as a hypothesis to be revised during research. If multiple domains are needed for the capability, sections must visibly cover each one — uneven coverage is a framing failure.

**You MUST remove the placeholder line** \`TODO — Phase 1 replaces this with 4–8 sections…\` from working/01-jtbd-and-knowledge-map.md as part of writing the real knowledge map. That sentinel is how the TUI knows Phase 1 has been completed — leaving it in place will keep the hub's progress cursor stuck on Framing even after you write good content.

Write the result to working/01-jtbd-and-knowledge-map.md (overwriting placeholders including the sentinel above, keeping the user's original capability line visible and adding your derived Capability Brief). Stop after this file is written. Do not start Phase 2.`,

  candidates: `## Phase 2 — Candidate discovery

Read working/01-jtbd-and-knowledge-map.md for the Capability Brief, internal JTBD, research lanes, and knowledge map. Then read the companion file \`source-finding-tactics.md\` in the skill bundle.

- Generate a long-list of candidate sources covering every section of the knowledge map. URLs and titles only — NO fetching yet.
- Cover the **Required source roles** from the Capability Brief, not only the knowledge-map sections. Each candidate reason should name the source role it may satisfy. If a role is required, aim for 3–6 plausible candidates before stopping; if no good candidates exist, write a "Role gaps" note in the long-list.
- Aim wide enough for the evidence contract: roughly 2–4× the eventual kept count, but derive the candidate count from the Required source roles, capability pattern, and source ecology rather than a fixed target.
- **Verify each URL exists** via WebFetch (or your fetch tool) — AI hallucinates URLs that sound plausible. A 404 candidate is worse than a missing one.
- For each candidate, capture: URL, title, one-line reason it's a candidate.
- If \`tape.yaml\` already has curator-provided web or YouTube sources, seed the long-list with those URLs before searching. Treat local files and skill sources as curator-provided context that should inform the reasons and sections, even when they are not URL candidates.
- **Find sources for the capability, not just the topic.** Extract the core future-agent action from the Capability Brief ("translate moodboards into UI direction", "debug failing CI", "write grant narratives", "generate medical SEO keyword clusters") and make sure the long-list includes sources that directly teach, demonstrate, or constrain that action. Implementation references, design-system docs, tooling docs, or process articles may be useful bridge sources, but they cannot satisfy the core action by themselves.
- **Deepen curator-named anchors.** When the curator names a person, studio, product, software, channel, or example they like, research why that anchor matters: their case studies, interviews, talks, docs, essays, YouTube appearances, or worked examples. Do not stop at a homepage/gallery/crawler result unless no deeper readable source exists, and say so in the reason.
- **For visual-output JTBDs, include visual and worked-example evidence.** If the job involves images, moodboards, art, posters, editorial design, interface concepts, or craft taste, search for image-heavy examples, case studies, design breakdowns, portfolios with rationale, talks, or videos. A corpus made mostly of UX primers, CSS references, and design-system docs is probably under-serving the job even if every knowledge-map section has a source.
- **For reference-translation capabilities, build a source ecology.** If Phase 1 marked \`Capability pattern: reference-translation\`, the candidate list must include candidates for at least two input/reference domains outside the target output medium, plus candidates that teach the translation move and candidates that demonstrate handoff language for the caller. For an art-direction-to-web project, web craft and accessibility sources are bridge/constraint candidates; they do not replace sources from visual culture, art direction, graphic design, architecture/interiors, photography/film, fashion, packaging, sports/media identity, or other source worlds named or implied by the goal.
- **If this is an improvement pass, use the audit.** If \`working/05-operating-fit-audit.md\` exists with \`status: improvement_recommended\`, treat this run as a focused second pass. Preserve the strongest existing source roles, search for the missing roles named in the audit, and rebuild the candidate list around the gap instead of restarting from the generic topic.

**Bounded search discipline:** Phase 2 should produce a useful long-list, not exhaust the internet. Count coverage by source roles and capability-pattern lanes, not by a universal source target. For ordinary narrow projects, stop when every required role and central knowledge-map section has enough candidates; that may be under 20. For broad, multi-domain, or \`reference-translation\` projects, continue past 30 when needed until each required role and each declared input/reference domain has 3–6 plausible verified candidates, usually topping out around 40–60 unless the Capability Brief explicitly needs more. If added candidates are redundant or low-signal, stop and write a Role gaps note instead of padding. If a URL cannot be verified after one direct attempt and one alternate search, skip it or include a nearby verified substitute instead of continuing to chase it.

**Search YouTube too, not just the web.** Liner fetches video transcripts and titles automatically (\`youtube.com/watch?v=…\` and \`youtu.be/…\` URLs are first-class sources). For most JTBDs there will be conference talks, lectures, or long-form interviews on YouTube that cover the topic from a different angle than written articles do. Look there explicitly. The quality hierarchy (in \`source-quality-hierarchy.md\`) treats conference talks from credible speakers as tier-3 sources — equal in standing to expert articles. Skip TED-style talks unless density is unusually high.

Write to working/02-candidate-longlist.md, grouped by knowledge-map section. Replace the placeholder entirely. Stop after this file is written; do not keep searching after you have enough candidates.

⚠ Bias correction: do NOT default to popular/recent sources. Search deliberately for primary sources (specs, canonical papers, official docs, foundational books). Mix media types — articles, YouTube talks/lectures, papers, primary docs.`,

  evaluation: `## Phase 4 — Evaluation

Read working/02-candidate-longlist.md. Then read the companion files \`source-quality-hierarchy.md\` and \`curator-notes.md\` in the skill bundle. Then fetch each candidate's content (transcripts for YouTube; readable body for web articles).

**Bounded fetch discipline:** do not get stuck chasing unavailable content. Try one direct fetch/read per candidate. If that fails, make at most one recovery attempt (alternate transcript source, search result, mirror, or metadata page). After two failed retrieval attempts total, stop searching for that candidate and evaluate from verified metadata, the Phase 2 reason, and whatever partial content is available. Usually that means \`dropped\`. Do not mark a source \`kept\` or \`trim\` from title, URL, search snippet, crawler metadata, or model memory alone.

Every candidate still needs a decision in the YAML: keep / trim / drop against the JTBD. For kept and trimmed sources, write a rating (1–5), a \`jtbd_fit\` label, a primary \`source_role\` from the Capability Brief when available, fetch evidence, a content-quality judgment, and a curator note in the three-thing template (role / where the value is / limitations).

**Evidence contract for kept/trimmed sources:**
- \`fetch_status: readable | partial\`. Use \`readable\` when you retrieved the substantive article/transcript/PDF body. Use \`partial\` when you retrieved enough real source content to judge, such as an abstract plus substantial excerpt. If the page is \`metadata_only\` or \`unavailable\`, the decision must be \`dropped\`.
- \`content_quality: high | medium\`. If the readable content is shallow, generic, AI-slop, SEO filler, a homepage/gallery with no rationale, or otherwise low-value, mark it \`dropped\` instead of keeping it.
- \`evidence:\` at least two content-specific evidence bullets from the fetched/read source. These are not quotes; they are concrete claims, sections, examples, transcript moments, or methods you actually saw in the content. Search snippets, titles, and model memory do not count.

**JTBD-fit labels are required for kept and trimmed sources**:
- \`direct\` — teaches, demonstrates, or argues for the core action in the JTBD.
- \`bridge\` — helps execute the result after the core decision is made, such as implementation mechanics, tokens, layout, or process.
- \`background\` — broad context, vocabulary, or primer material. Background sources should rarely be \`required\`.

If a section contains only \`bridge\` or \`background\` sources, the evaluation must say it is weak coverage and either backfill a direct source or mark the section as a known gap. Do not let topic adjacency pass as JTBD fit.

**Source-role discipline**:
- Assign each kept/trim source to the strongest matching required source role. If it does not satisfy any required role, mark \`source_role: supporting/background\` and explain why it is still worth keeping.
- Do not count a homepage, gallery index, metadata page, or shallow listicle as satisfying a worked-example/case-study role unless the fetched content includes real rationale, process, analysis, or artifacts the future AI can learn from.
- For visual-output, taste, craft, or example-dependent capabilities, one or two examples is a warning sign, not a pass. Prefer several distinct worked examples with rationale over more primers.
- For \`reference-translation\` capabilities, direct fit means the source either teaches how to read/interpret the input reference domain or demonstrates a real translation from reference qualities into target-output decisions. Target-medium implementation mechanics, accessibility specs, token formats, and style-guide infrastructure are usually \`bridge\` fit unless they explicitly teach that translation move.

**Chunked decision discipline:** do not hold all decisions in conversation memory until the end. Create \`working/evaluation-decisions/\` and, after each knowledge-map section or every ~10 candidates, write/update a small YAML fragment there. Use filenames like \`01-foundations.yaml\`, \`02-patterns.yaml\`, etc. Each fragment uses the same \`candidates:\` list shape shown below and must include every candidate from that section/chunk. Liner can assemble \`working/03-evaluation.yaml\` from these fragments in the original longlist order.

**Artifact closure discipline:** write \`working/03-evaluation.yaml\` as soon as you have enough evidence to decide. Do not announce that you are "ready to write" before writing either the fragments or the final file. For dropped candidates, keep the entry compact: URL, title, decision, section if known, and one rationale sentence. Rating and note are required only for \`kept\` and \`trim\`. If the final YAML is long, prioritize complete fragments in \`working/evaluation-decisions/\`; Liner will assemble and validate the final YAML.

Write decisions to working/03-evaluation.yaml in the format:

\`\`\`yaml
candidates:
  - url: https://...
    title: ...
    decision: kept   # kept | trim | dropped
    rating: 5
    jtbd_fit: direct # direct | bridge | background
    source_role: core-action method
    fetch_status: readable # readable | partial; metadata_only/unavailable must be dropped
    content_quality: high # high | medium; low must be dropped
    evidence:
      - The fetched source includes a specific method/example/argument, not just a title.
      - Another source-specific detail that proves the content was read.
    section: foundations
    rationale: One specific sentence.
    note: |
      Multi-line curator note for kept/trim sources. Use the three-thing template.
\`\`\`

Stop after the fragments and/or final file are written. Do not draft synthesis yet.`,

  quality: `## Phase 5 — Quality checks

Read working/03-evaluation.yaml. Then read \`quality-check-tests.md\` in the skill bundle — note especially the strengthened framing-gap rules below.

**Bounded quality discipline:** Phase 5 is an audit of the evaluated corpus, not a fresh research phase. Start from the existing \`kept\` and \`trim\` entries in \`working/03-evaluation.yaml\`. If those entries lack \`kind\`, assign \`reference / principle / prescription / example\` from the title, note, section, rating, and source type before you consider any search. Missing \`kind\` metadata is not itself a reason to backfill.

Run the core-action test, then the eight standard tests against the keep-list:
0. Core-action fit — what is the exact action in the JTBD, and which kept sources directly teach or demonstrate it?
1. Redundancy — any two kept sources making essentially the same point?
2. Coverage — any knowledge-map section with zero sources?
3. Disagreement — strongest claim with a credible counter included?
4. Framing-gap — is any whole way of thinking about the JTBD missing?
5. Source-kind balance — distribution across the four kinds. Any kind at zero must be defended or backfilled.
6. Note-quality — does each kept/trim curator note say how to use the source, what value/bar it provides, and one limitation or boundary?
7. Source-role fit — do the kept/trim sources satisfy the Required source roles named in the Capability Brief?
8. Capability-pattern fit — if Phase 1 named a specialized pattern such as \`reference-translation\`, does the corpus satisfy the pattern-specific evidence contract?

**Core-action fit rules (strict)**:

- Quote or paraphrase the JTBD's core action before judging the source list.
- Count kept sources by \`jtbd_fit\`: \`direct / bridge / background\`. If older entries lack \`jtbd_fit\`, infer it from the title, rationale, note, and section, then update \`working/03-evaluation.yaml\`.
- A healthy action-oriented mixtape needs at least two \`direct\` kept sources, and every central knowledge-map section needs either one \`direct\` source or an explicit reason why a \`bridge\` source is sufficient.
- Curator-named anchors do not count as direct fit if the fetched source is only a homepage, gallery index, metadata page, or unreadable pointer. They may be taste anchors, but the quality report must say whether their rationale/process was actually retrieved.
- If the direct count is weak, take exactly one action: backfill direct sources within the Phase 5 search budget, or document the gap and recommend re-running Candidate discovery with action-specific search terms.

**Framing-gap test rules (strict)**:

- Before looking at the keep-list, NAME 2–3 candidate perspectives that could exist outside the corpus (e.g. designer voice, accessibility-first viewpoint, security/abuse perspective, contrarian, adjacent discipline). Write them into working/04-quality-checks.md.
- For each named perspective, classify coverage in TWO dimensions:
  - **Stance** — is there a verified source whose author *argues for* this perspective as a design center of gravity? \`stance-represented\` or \`stance-absent\`.
  - **Concerns** — do any kept sources address the perspective's operational concerns, even from a different stance? \`concerns-addressed\` or \`concerns-absent\`.
  A perspective is fully covered only when its stance is represented. \`concerns-addressed\` alone is a SOFT GAP — the corpus internalizes the concern without giving the consuming AI a worked rival argument to cite.
- **Unverified sources do NOT count.** A source whose Phase 4 note acknowledges unfetched content, broken URLs, or unobtainable transcripts cannot satisfy a perspective slot. Tokenism is the failure mode this rule prevents.
- For each perspective that is not \`stance-represented\`, take exactly one of three actions:
  - **(a) Search for a stance source.** Vocabulary: who would *argue for* this position, what would they write, where would they publish? (e.g., "Brandur Leach on terminal density," not "CLI scriptability best practices.") One search pass; if a verified source fits, add it.
  - **(b) Argue concerns are sufficient.** Write the argument explicitly: "Concerns covered by [Sources N, M] inside the [dominant] stance. The dissenting stance is real but its operational implications are absorbed; no separate source needed." Not a free pass — if you can't articulate the trade-off, fall back to (a) or (c).
  - **(c) Flag the bias explicitly.** Document: "Perspective: [name] — stance unrepresented after one search pass. Corpus argues from [X] but no source argues for [Y]. Recommendation: ship + document, OR weaken the JTBD claim."

**Search budget for Phase 5**:

- Use at most one search pass for any single missing perspective.
- Use at most one source-kind backfill search total, and only after assigning kinds to every existing kept/trim source.
- Do not exceed four external search/fetch attempts for the whole Quality phase. If the budget is spent, stop searching, document the gap or defense, and write \`working/04-quality-checks.md\`.
- On resume after a paused/cancelled Quality run, do not continue a search loop. Use the evidence already gathered, finish the kind assignments, and write the quality report.

**Source-kind balance rules**:

- The four kinds are \`reference / principle / prescription / example\`. They represent four argumentative roles in the corpus.
- If existing kept/trim entries are missing \`kind\`, classify them first. Use \`reference\` for specs/manuals/API docs, \`principle\` for essays/arguments, \`prescription\` for style guides/rules/how-to guidance, and \`example\` for case studies/demos/worked products. Update \`working/03-evaluation.yaml\` with those assignments.
- Count the distribution. A zero in any column is a decision, not an accident.
- 0 \`reference\` → no substrate authority. 0 \`principle\` → no philosophical anchor. 0 \`prescription\` → no concrete rules. 0 \`example\` → no taste calibration.
- For each zero, either backfill (one search pass for a source of that role) or write a defense: "Kinds audit: X/Y/Z/W. Accepted: [reason this JTBD legitimately doesn't need that kind]."

**Note-quality rules**:

- Review every kept/trim entry's \`note\`. A useful curator note must do three jobs: name how to use/read the source, name the concrete value or quality bar it contributes, and name one limitation, scope boundary, or reason not to over-weight it.
- If a note is filler, generic, or only says "useful background", repair it in \`working/03-evaluation.yaml\` before moving to synthesis. Do not fetch/search just to repair prose; use the already gathered rationale, title, source type, section, and partial content.
- In \`working/04-quality-checks.md\`, report how many notes were checked, how many were repaired, and cite the source titles or URLs for any repairs.

**Source-role fit rules (strict)**:

- Read the Required source roles from the Capability Brief in \`working/01-jtbd-and-knowledge-map.md\`. If the brief is older and lacks them, infer 4–7 required roles from the core action before judging the corpus, then write those inferred roles into \`working/04-quality-checks.md\`.
- For each required role, count kept/trim sources that actually satisfy that role. Count by content evidence, not by title, URL, kind, or topic adjacency.
- A source kind is not a source role. \`example: 2\` does not automatically satisfy "high-craft worked case studies" unless those examples contain enough fetched rationale/process/artifact evidence for the future AI to learn from.
- Required roles must meet the minimum named in the Capability Brief. If no minimum was named, use at least 2 strong sources; use at least 3 substantive worked examples/case studies when the capability depends on taste, visual output, craft judgment, or recognizing quality; require primary/official sources for legal, medical, financial, or safety-sensitive authority roles.
- Do not write "light but not absent" as a pass for a material role. Either improve the corpus inside the Phase 5 budget or write \`working/05-operating-fit-audit.md\` with \`status: improvement_recommended\`.

**Capability-pattern fit rules (strict)**:

- Read the Capability Brief for a \`Capability pattern:\` line. If none is present, write Test 8 as "No specialized capability pattern detected" and do not invent one.
- For \`reference-translation\`, audit four things:
  1. **Input/reference-domain coverage:** at least two distinct source domains outside the target output medium have substantive kept/trim evidence. Count by content, not by a passing mention. For web/UI targets, web implementation, accessibility, token, or design-system sources do not count as input/reference-domain coverage.
  2. **Translation-method coverage:** at least two direct kept/trim sources teach or demonstrate the move from source qualities to target decisions. Generic moodboard primers, galleries, and CSS mechanics are not enough.
  3. **Caller-handoff coverage:** the corpus supports a concrete runtime output shape for the caller, such as observations, interpretation, carry-forward rules, implementation vocabulary, constraints, avoid-list, validation checks, and clarification questions.
  4. **Constraint balance:** implementation/accessibility/system sources are present when needed, but they do not outnumber and replace the direct reference-reading and translation evidence in the core action.
- If any of those fail, update \`working/05-operating-fit-audit.md\` with \`status: improvement_recommended\`; do not call the corpus ready because it has enough general design/web sources.

**Operating-fit audit**:

- After Tests 0–7, ask whether the keep-list can actually help a future AI agent do the capability in the Capability Brief, not merely talk around the topic.
- Look for missing source roles, not just missing source kinds. Examples: high-craft case studies, worked image-to-interface translations, legal primary sources, medical authority, implementation constraints, dissenting practitioner voice, user-research evidence, or domain-specific keyword data.
- If a material role is weak and you cannot fix it inside the Phase 5 search budget, do not write "ready with limitation." Write \`working/05-operating-fit-audit.md\` with \`status: improvement_recommended\` so Liner can offer a focused improvement pass. Also summarize the recommendation in \`working/04-quality-checks.md\`.
- If the corpus is fit for the operating layer, do not write an improvement recommendation marker.

**Required output structure** in working/04-quality-checks.md:

\`\`\`
## Test 0 — Core-action fit
Core action: [verb phrase from JTBD]
Distribution: X direct / Y bridge / Z background
[finding + action]

## Test 1 — Redundancy
[finding + action]

## Test 2 — Coverage
[finding + action]

## Test 3 — Disagreement
[finding + action]

## Test 4 — Framing-gap

### Perspectives audit
- [Perspective name] — stance-represented (Source N, verified) ✓
- [Perspective name] — stance-absent, concerns-absent. Backfill: searched [vocabulary], found [source] → now stance-represented (Source M) ✓
- [Perspective name] — stance-absent, concerns-addressed by [Sources L, K]. Argued sufficient: [explicit trade-off paragraph].
   OR
- [Perspective name] — stance-absent after one search pass. Corpus argues from [X], no source argues for [Y]. Recommendation: [...].

## Test 5 — Source-kind balance

Distribution: X reference / Y principle / Z prescription / W example
[If any zero: backfill action OR defense paragraph]

## Test 6 — Note-quality

Checked: N kept/trim notes
Repaired: M notes
[If repaired: list Source/title/URL and the note issue fixed. If none repaired: one sentence explaining why the notes already carry use cue, value/bar, and limitation.]

## Test 7 — Source-role fit

Required roles:
- [role] — minimum: N; current: M; status: pass|weak|gap; evidence: [source titles]

[If any required role is weak/gap: backfill action OR operating-fit audit path]

## Test 8 — Capability-pattern fit

Pattern: [reference-translation | none | other named pattern]
[For reference-translation: input/reference-domain coverage, translation-method coverage, caller-handoff coverage, constraint balance]
[finding + action]
\`\`\`

If an improvement pass is recommended, also write:

\`\`\`
# Operating-fit audit

status: improvement_recommended
gap: [the missing source role or weak evidence role]
missing_roles:
  - role: [required source role]
    current_coverage: [what the keep-list has now]
    needed: [what is missing]
why_it_matters: [why this gap would hurt a future AI agent using the corpus]
recommended_pass:
  - [search lane or source type Liner should look for]
  - [search lane or source type Liner should look for]
user_can_add:
  - [concrete custom source the user could add if web research stays thin]
skip_note: [what remains weaker if the user skips for now]
\`\`\`

Every named perspective must appear in the audit with explicit coverage status. A missing "Perspectives audit" section, a Test 5 section without the distribution, a missing Test 6 note-quality section, a missing Test 7 source-role section, or a missing Test 8 capability-pattern section means the test wasn't run honestly.

If a test fails, fix the keep-list (update working/03-evaluation.yaml) before reporting done when the fix fits inside the Phase 5 budget. If it requires a wider second pass, write the operating-fit audit instead of presenting the corpus as fully ready. Even tests that pass without changes should be noted — silent passes train the model to treat the tests as ceremonial.

Stop after working/04-quality-checks.md is written and any necessary changes are reflected in 03-evaluation.yaml.`,

  synthesis: `## Phase 6 — Synthesis

Read synthesis.md (replacing the placeholder), working/03-evaluation.yaml (the keep-list), and \`synthesis-guidance.md\` in the skill bundle.

Draft synthesis.md as continuous prose, 800–2000 words. The synthesis is the curator's *distilled understanding of the domain* — principles, contested questions, framework distinctions. NOT a recap of source content.

⚠ Bias correction: do NOT write "Source 1 says X; Source 2 says Y." Push toward the curator's framing. Synthesis is the first thing the consuming AI reads — it sets the lens.

**Required operating sections**:
- Include a \`## Generative rules\` section with about five imperative rules the consuming AI can apply directly. These are the "if you remember nothing else" rules, phrased as actions or constraints, not summaries. Example shape: "Channel discipline: stdout is for data, stderr is for humans, and exit codes mean what they say."
- Include a \`## Stances this corpus takes\` section naming the corpus's opinionated positions and trade-offs. This section should say what the mixtape leans toward, what it resists, and where a dissenting source should still temper the rule.

**Weight sources by their \`kind\`** (set during Phase 7, but visible in 03-evaluation.yaml too):
- \`reference\` sources (specs, RFCs) — substrate. Cite once for authority; don't lean on them for framing.
- \`principle\` sources (essays, manifestos) — load-bearing for the synthesis's stance section. These earn the most quoting.
- \`prescription\` sources (style guides, rules) — useful for concrete distinctions but rarely for the synthesis's voice.
- \`example\` sources (case studies, surveys) — taste calibration. Cite when arguing what "good" looks like.

If a source's \`kind\` field is absent, treat it as principle-equivalent (the synthesis-writing default).

For methodology mode, mark spots where the curator should personalize ("[curator's view: …]") so they have explicit handoff points. For quick mode, write a coherent draft they can lightly edit.

Stop after synthesis.md is written.`,

  assembly: `## Phase 7 — Assembly (draft)

You are preparing a *draft* of the sources list for the curator to review. **DO NOT touch tape.yaml directly.** Write your proposal to \`working/07-tape-draft.yaml\` and stop.

Read these inputs:
- \`working/03-evaluation.yaml\` — the keep-list with curator notes and ratings.
- \`synthesis.md\` — the curator's framing, to inform section ordering.
- \`tape.yaml\` — preserve any existing source order the curator already chose; you're proposing additions/changes, not erasing.
- \`local-sources/sources-manifest.yaml\`, when present — active custom sources the curator added. These are curator-selected sources, not optional research candidates.

Build the draft as YAML with a top-level \`sources:\` list. Each entry needs:

\`\`\`yaml
sources:
  - type: web         # web | youtube | local_file | skill
    url: https://...
    section: foundations
    priority: required   # required | optional
    kind: prescription   # reference | principle | prescription | example
    note: |
      One-paragraph curator note: role of this source, where the value lives,
      and any limitations. Use the three-thing template.
\`\`\`

Rules:
- Include every candidate marked \`kept\` or \`trim\` in 03-evaluation.yaml only when it has \`fetch_status: readable|partial\`, \`content_quality: high|medium\`, and at least two \`evidence\` bullets. If an older evaluation marks a source kept/trim without this evidence, stop and repair \`working/03-evaluation.yaml\` instead of drafting.
- Drop evaluated URL candidates marked \`dropped\`, \`metadata_only\`, \`unavailable\`, or \`content_quality: low\`.
- Preserve every existing \`local_file\` and \`skill\` source from tape.yaml even though those entries do not appear in the URL-only evaluation file. Carry forward \`path\`, \`url\`, \`citation\`, \`note\`, \`priority\`, \`section\`, and \`kind\` where present. If any required review metadata is missing, add it without changing the source identity.
- Preserve every \`active: true\` source from \`local-sources/sources-manifest.yaml\`, even if it is not present in \`tape.yaml\` yet. Custom sources are curator-selected. Include active recovered \`local_file\` entries from \`local-sources/recovered/\`; do not include inactive original URL entries when a recovered local copy replaced them.
- Do not silently convert \`local_file\` or \`skill\` sources into \`web\` sources. Skill sources are reference material, not active instructions.
- Preserve current tape order first, then append newly kept/trimmed URL candidates that were not already on the tape. Dedupe by URL for web/youtube, by \`path\`+\`citation\` for local_file, and by \`path\` or \`url\` for skill.
- Within a section, order new evaluated URL candidates by rating (highest first).
- Order sections in reading order: foundations / patterns / applications / craft (or whatever the synthesis suggests).
- Notes come straight from 03-evaluation.yaml's \`note\` field. Do not paraphrase.
- Mark a source \`priority: optional\` only when the methodology genuinely treats it as skippable (rare).
- \`local_file\` and \`skill\` entries: never invent paths or skill identifiers. Include them only when they already exist in tape.yaml or as active entries in \`local-sources/sources-manifest.yaml\`.

**Source \`kind\` is required on every entry.** It tells the synthesis prompt and the consuming AI which sources to weight how. Pick the best fit:

- \`reference\` — specs / standards cited but not deep-read (POSIX, RFCs, official manual pages). The consuming AI cites these for authority but doesn't quote them at length.
- \`principle\` — read for stance / framing (essays, manifestos, philosophical pieces). Shapes how the AI thinks about the domain.
- \`prescription\` — read for concrete rules (style guides, conventions, explicit do/don't lists). The AI quotes these directly when applying rules.
- \`example\` — read for taste / illustration (case studies, tool surveys, worked examples). Shows what good looks like.

A source can plausibly fit two categories; pick the one closest to *how the consuming AI should USE this source*. When in doubt: rules → \`prescription\`; arguments → \`principle\`; examples → \`example\`; authority without narrative → \`reference\`.

When done, write \`working/07-tape-draft.yaml\` and stop. The TUI will show the curator a review screen to accept, edit, or discard. DO NOT modify tape.yaml.`,
};

export type PromptBuildArgs = {
  phaseId: PhaseId;
  project: string;
  skillPath: string;
  tape: Tape;
};

export type EvaluationSectionPromptArgs = {
  project: string;
  skillPath: string;
  tape: Tape;
  group: LonglistCandidateGroup;
};

/** Construct the system+user prompt blob passed to the agent. */
export function buildPhasePrompt(args: PromptBuildArgs): string {
  const { phaseId, project, skillPath, tape } = args;
  const skillContents = safeRead(join(skillPath, "SKILL.md")) ??
    "(SKILL.md not found at " + skillPath + ")";
  const phaseInstruction =
    PHASE_INSTRUCTIONS[phaseId] ??
    `## Phase ${phaseId}\n\nNo dedicated instruction. Use SKILL.md to figure out what to do for the ${phaseId} phase, then stop.`;

  const companionNote = (COMPANIONS[phaseId] ?? [])
    .map((f) => `  - ${join(skillPath, f)}`)
    .join("\n");

  return `You are running the curating-mixtapes methodology on behalf of a curator.

# Mixtape context

- Project folder: ${project}
- Title: ${tape.title || "(not set)"}
- Description: ${tape.description || "(not set)"}
- Mode: ${tape.mode ?? "quick"}
- AI-agent goal: ${tape.jtbd || "(not set)"}
- Curator: ${tape.curator || "(not set)"}
- Sources currently on tape: ${tape.sources.length}

# Skill bundle

The methodology is documented in:
  - ${join(skillPath, "SKILL.md")}

Companion files (read the ones listed in the phase instructions below):
${companionNote || "  (none for this phase)"}

# What to do

${phaseInstruction}

# Output discipline

- Work on the filesystem. Do not paste large file contents back in your response — write them to disk.
- Be concise in your spoken output. The user is watching a TUI; long monologues clutter the screen.
- If you cannot proceed (missing AI-agent goal, missing prior-phase artifact), say so clearly and stop. Do NOT make up content.

${WORKSPACE_DISCIPLINE}

# Final message format (required)

When the phase finishes, end with a markdown message in **this exact shape**.
The TUI renders this — section headings (**Bold:**) become colored chips, bullets render with • glyphs, and bracketed paths render dim. Following the format gives the curator a scannable summary; freeform prose does not.

\`\`\`
Done with Phase N.

**What I did:**
- Verb-first sentence describing one concrete output [path/to/file]
- Another verb-first sentence [path/to/another-file]

**Mix:** (only if you discovered or surveyed sources — skip otherwise)
- Category one: short list of representative items
- Category two: short list

**Review checklist:** (only if there's something the curator should verify in *this TUI screen* before moving on — skip if not)
- A specific judgment call to make from what's visible here
- Another specific judgment call
\`\`\`

Rules:
- **Lead every bullet with a verb-phrase**, not a file path or URL. Put paths in \`[brackets]\` at the very end of the bullet.
- Don't introduce extra sections. The TUI styles only these three headings; anything else renders as plain text.
- Skip sections that don't apply. Better to omit "Mix:" than to fill it with filler.
- 2–4 bullets per section. Be specific, not exhaustive.
- For the opening line ("Done with Phase N.") use the actual phase number — not "this phase" or "the current phase".

**Review checklist — scope rules** (important):

The "Review checklist" is what the curator should do on the **current TUI done screen**, before pressing Enter to advance. The TUI's only affordances at that moment are: read your message, open the artifact file in \`$EDITOR\`, re-run the phase, or continue. So bullets must be answerable by ONE of:

- A yes/no judgment the curator can make by skimming what you wrote ("Does the knowledge map cover X?")
- A specific edit they can make to the artifact you just wrote ("Drop the InfoQ Q&A in the long-list — weakest entry")
- A specific source/section worth a closer look ("The Pinsker entry skews designer-voice — confirm or swap")

Do NOT include bullets that ask the curator to:
- Go find more sources (the next phase handles selection — flag it for *yourself* in a later phase note, not for them)
- Add categories or sections you didn't include (you should have included them)
- "Flag if you want a dedicated source for X" — that's outside the loop the TUI gives them
- Do open-ended research, validation, or thinking that requires leaving this screen
- Anything that doesn't fit "read → edit artifact in \$EDITOR → advance"

If you don't have a concrete TUI-scoped checklist item, **omit the section**. An empty Review checklist is better than one full of asks the curator can't act on from where they are.

# Full SKILL.md follows

${skillContents}
`;
}

export function buildEvaluationSectionPrompt(args: EvaluationSectionPromptArgs): string {
  const base = buildPhasePrompt({
    phaseId: "evaluation",
    project: args.project,
    skillPath: args.skillPath,
    tape: args.tape,
  });
  const candidates = args.group.candidates
    .map((candidate, i) => {
      const lines = [
        `${i + 1}. ${candidate.title || "(untitled)"}`,
        `   URL: ${candidate.url}`,
        `   Section: ${candidate.section || args.group.section}`,
      ];
      if (candidate.reason) lines.push(`   Phase 2 reason: ${candidate.reason}`);
      return lines.join("\n");
    })
    .join("\n\n");

  return `${base}

# Section-scoped Evaluation run

This is Evaluation chunk ${args.group.index} of ${args.group.total}. Keep the context short and finish only this chunk.

Evaluate only these ${args.group.candidates.length} candidates from the "${args.group.section}" section:

${candidates}

# Required chunk output

- Create \`working/evaluation-decisions/\` if needed.
- Write exactly one YAML fragment at \`${args.group.fragmentPath}\`.
- The fragment must have a top-level \`candidates:\` list.
- Include every URL listed above exactly once.
- Do not write or edit \`working/03-evaluation.yaml\`; Liner assembles it after all chunks finish.
- Do not evaluate candidates from any other section.
- Stop after the fragment is written and parsed.
`;
}

function safeRead(path: string): string | null {
  try {
    return readFileSync(path, "utf8");
  } catch {
    return null;
  }
}
