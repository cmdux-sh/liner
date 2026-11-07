## Phase 5 — Quality checks

Read working/03-evaluation.yaml. Then read `quality-check-tests.md` in the skill bundle — note especially the strengthened framing-gap rules below.

**Bounded quality discipline:** Phase 5 is an audit of the evaluated corpus, not a fresh research phase. Start from the existing `kept` and `trim` entries in `working/03-evaluation.yaml`. If those entries lack `kind`, assign `reference / principle / prescription / example` from the title, note, section, rating, and source type before you consider any search. Missing `kind` metadata is not itself a reason to backfill.

When enriching `working/03-evaluation.yaml`, update only fields that are missing or intentionally changing; never insert a second copy of an existing YAML key. After every edit, parse the file with a strict duplicate-key-rejecting YAML parser. Ruby `YAML.load_file` and permissive PyYAML loads are not sufficient validation because they silently accept duplicate mapping keys.

Run the core-action test, then the eight standard tests against the keep-list:
0. Core-action fit — what is the exact action in the JTBD, and which kept sources directly teach or demonstrate it?
1. Redundancy — any two kept sources making essentially the same point?
2. Coverage — any knowledge-map section with zero sources?
3. Disagreement — strongest claim with a credible counter included?
4. Framing-gap — is any whole way of thinking about the JTBD missing?
5. Source-kind balance — distribution across the four kinds. Any kind at zero must be defended or backfilled.
6. Note-quality — does each kept/trim curator note say how to use the source, what value/bar it provides, and one limitation or boundary?
7. Source-role fit — do the kept/trim sources satisfy the Required source roles named in the Capability Brief?
8. Capability-pattern fit — if Phase 1 named a specialized pattern such as `reference-translation`, does the corpus satisfy the pattern-specific evidence contract?

**Core-action fit rules (strict)**:

- Quote or paraphrase the JTBD's core action before judging the source list.
- Count kept sources by `jtbd_fit`: `direct / bridge / background`. If older entries lack `jtbd_fit`, infer it from the title, rationale, note, and section, then update `working/03-evaluation.yaml`.
- A healthy action-oriented mixtape needs at least two `direct` kept sources, and every central knowledge-map section needs either one `direct` source or an explicit reason why a `bridge` source is sufficient.
- Curator-named anchors do not count as direct fit if the fetched source is only a homepage, gallery index, metadata page, or unreadable pointer. They may be taste anchors, but the quality report must say whether their rationale/process was actually retrieved.
- If the direct count is weak, take exactly one action: backfill direct sources within the Phase 5 search budget, or document the gap and recommend re-running Candidate discovery with action-specific search terms.

**Framing-gap test rules (strict)**:

- Before looking at the keep-list, NAME 2–3 candidate perspectives that could exist outside the corpus (e.g. designer voice, accessibility-first viewpoint, security/abuse perspective, contrarian, adjacent discipline). Write them into working/04-quality-checks.md.
- For each named perspective, classify coverage in TWO dimensions:
  - **Stance** — is there a verified source whose author *argues for* this perspective as a design center of gravity? `stance-represented` or `stance-absent`.
  - **Concerns** — do any kept sources address the perspective's operational concerns, even from a different stance? `concerns-addressed` or `concerns-absent`.
  A perspective is fully covered only when its stance is represented. `concerns-addressed` alone is a SOFT GAP — the corpus internalizes the concern without giving the consuming AI a worked rival argument to cite.
- **Unverified sources do NOT count.** A source whose Phase 4 note acknowledges unfetched content, broken URLs, or unobtainable transcripts cannot satisfy a perspective slot. Tokenism is the failure mode this rule prevents.
- For each perspective that is not `stance-represented`, take exactly one of three actions:
  - **(a) Search for a stance source.** Vocabulary: who would *argue for* this position, what would they write, where would they publish? (e.g., "Brandur Leach on terminal density," not "CLI scriptability best practices.") One search pass; if a verified source fits, add it.
  - **(b) Argue concerns are sufficient.** Write the argument explicitly: "Concerns covered by [Sources N, M] inside the [dominant] stance. The dissenting stance is real but its operational implications are absorbed; no separate source needed." Not a free pass — if you can't articulate the trade-off, fall back to (a) or (c).
  - **(c) Flag the bias explicitly.** Document: "Perspective: [name] — stance unrepresented after one search pass. Corpus argues from [X] but no source argues for [Y]. Recommendation: ship + document, OR weaken the JTBD claim."

**Search budget for Phase 5**:

- Use at most one search pass for any single missing perspective.
- Use at most one source-kind backfill search total, and only after assigning kinds to every existing kept/trim source.
- Do not exceed four external search/fetch attempts for the whole Quality phase. If the budget is spent, stop searching, document the gap or defense, and write `working/04-quality-checks.md`.
- If you backfill a source during Quality, add its URL to `working/02-candidate-longlist.md` before adding or changing its entry in `working/03-evaluation.yaml`. Parse both artifacts and confirm their candidate counts still match before reporting the phase complete.
- On resume after a paused/cancelled Quality run, do not continue a search loop. Use the evidence already gathered, finish the kind assignments, and write the quality report.

**Source-kind balance rules**:

- The four kinds are `reference / principle / prescription / example`. They represent four argumentative roles in the corpus.
- If existing kept/trim entries are missing `kind`, classify them first. Use `reference` for specs/manuals/API docs, `principle` for essays/arguments, `prescription` for style guides/rules/how-to guidance, and `example` for case studies/demos/worked products. Update `working/03-evaluation.yaml` with those assignments.
- Count the distribution. A zero in any column is a decision, not an accident.
- 0 `reference` → no substrate authority. 0 `principle` → no philosophical anchor. 0 `prescription` → no concrete rules. 0 `example` → no taste calibration.
- For each zero, either backfill (one search pass for a source of that role) or write a defense: "Kinds audit: X/Y/Z/W. Accepted: [reason this JTBD legitimately doesn't need that kind]."

**Note-quality rules**:

- Review every kept/trim entry's `note`. A useful curator note must do three jobs: name how to use/read the source, name the concrete value or quality bar it contributes, and name one limitation, scope boundary, or reason not to over-weight it.
- If a note is filler, generic, or only says "useful background", repair it in `working/03-evaluation.yaml` before moving to synthesis. Do not fetch/search just to repair prose; use the already gathered rationale, title, source type, section, and partial content.
- In `working/04-quality-checks.md`, report how many notes were checked, how many were repaired, and cite the source titles or URLs for any repairs.

**Source-role fit rules (strict)**:

- Read the Required source roles from the Capability Brief in `working/01-jtbd-and-knowledge-map.md`. If the brief is older and lacks them, infer 4–7 required roles from the core action before judging the corpus, then write those inferred roles into `working/04-quality-checks.md`.
- For each required role, count kept/trim sources that actually satisfy that role. Count by content evidence, not by title, URL, kind, or topic adjacency.
- A source kind is not a source role. `example: 2` does not automatically satisfy "high-craft worked case studies" unless those examples contain enough fetched rationale/process/artifact evidence for the future AI to learn from.
- Required roles must meet the minimum named in the Capability Brief. If no minimum was named, use at least 2 strong sources; use at least 3 substantive worked examples/case studies when the capability depends on taste, visual output, craft judgment, or recognizing quality; require primary/official sources for legal, medical, financial, or safety-sensitive authority roles.
- Do not write "light but not absent" as a pass for a material role. Either improve the corpus inside the Phase 5 budget or write `working/05-operating-fit-audit.md` with `status: improvement_recommended`.

**Capability-pattern fit rules (strict)**:

- Read the Capability Brief for a `Capability pattern:` line. If none is present, write Test 8 as "No specialized capability pattern detected" and do not invent one.
- For `reference-translation`, audit four things:
  1. **Input/reference-domain coverage:** at least two distinct source domains outside the target output medium have substantive kept/trim evidence. Count by content, not by a passing mention. For web/UI targets, web implementation, accessibility, token, or design-system sources do not count as input/reference-domain coverage.
  2. **Translation-method coverage:** at least two direct kept/trim sources teach or demonstrate the move from source qualities to target decisions. Generic moodboard primers, galleries, and CSS mechanics are not enough.
  3. **Caller-handoff coverage:** the corpus supports a concrete runtime output shape for the caller, such as observations, interpretation, carry-forward rules, implementation vocabulary, constraints, avoid-list, validation checks, and clarification questions.
  4. **Constraint balance:** implementation/accessibility/system sources are present when needed, but they do not outnumber and replace the direct reference-reading and translation evidence in the core action.
- If any of those fail, update `working/05-operating-fit-audit.md` with `status: improvement_recommended`; do not call the corpus ready because it has enough general design/web sources.

**Operating-fit audit**:

- After Tests 0–7, ask whether the keep-list can actually help a future AI agent do the capability in the Capability Brief, not merely talk around the topic.
- Look for missing source roles, not just missing source kinds. Examples: high-craft case studies, worked image-to-interface translations, legal primary sources, medical authority, implementation constraints, dissenting practitioner voice, user-research evidence, or domain-specific keyword data.
- If a material role is weak and you cannot fix it inside the Phase 5 search budget, do not write "ready with limitation." Write `working/05-operating-fit-audit.md` with `status: improvement_recommended` so `liner research improve` can target it. Also summarize the recommendation in `working/04-quality-checks.md`.
- If the corpus is fit for the operating layer, do not write an improvement recommendation marker.

**Required output structure** in working/04-quality-checks.md:

```
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
```

If an improvement pass is recommended, also write:

```
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
```

Every named perspective must appear in the audit with explicit coverage status. A missing "Perspectives audit" section, a Test 5 section without the distribution, a missing Test 6 note-quality section, a missing Test 7 source-role section, or a missing Test 8 capability-pattern section means the test wasn't run honestly.

If a test fails, fix the keep-list (update working/03-evaluation.yaml) before reporting done when the fix fits inside the Phase 5 budget. If it requires a wider second pass, write the operating-fit audit instead of presenting the corpus as fully ready. Even tests that pass without changes should be noted — silent passes train the model to treat the tests as ceremonial.

Stop after working/04-quality-checks.md is written and any necessary changes are reflected in 03-evaluation.yaml.
