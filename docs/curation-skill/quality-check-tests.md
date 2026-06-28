# Quality-check tests

Loaded by the `curating-mixtapes` skill at the start of Phase 5 (Quality checks).

This file describes the core-action test and the eight standard tests every
mixtape should pass before moving to synthesis. Apply them deliberately, in
this order. Write findings to `working/04-quality-checks.md`.

Phase 5 is the methodology's correction for single-pass curation. The first four phases work forward — frame, find, fetch, evaluate. Phase 5 works backward — looking at the whole keep-pile and asking what's wrong with it. The two passes do different work. If you skip this phase, the mixtape's blind spots ship with the mixtape.

The framing-gap and source-role tests are the highest-leverage checks and the ones most curators skip.

## Automated-run budget

When Phase 5 is running inside the TUI or another automated Liner flow, it must stay bounded. Treat it as an audit over `working/03-evaluation.yaml`, not as a second open-ended discovery phase.

- Classify existing kept/trim sources before searching. If a source is missing `kind`, infer it from title, URL/source type, section, rating, and curator note, then update `working/03-evaluation.yaml`.
- Missing `kind` metadata is not itself a corpus gap. It becomes a source-kind gap only after every existing kept/trim source has a kind and the distribution still has a zero column.
- Use at most one search pass for any one missing perspective.
- Use at most one source-kind backfill search total.
- Do not exceed four external search/fetch attempts across the whole Quality phase. If the budget is spent, stop searching, document the gap or defense, and write `working/04-quality-checks.md`.
- On resume after a paused/cancelled Quality run, do not continue a search loop. Use the evidence already gathered, repair weak curator notes from existing evidence, and close the artifact.
- If a material source-role gap remains after the bounded audit, write `working/05-operating-fit-audit.md` with `status: improvement_recommended` instead of declaring the corpus ready with a limitation. Liner will offer a focused improvement pass or let the user skip for now.

---

## Setting up the phase

Before running the tests, gather the inputs in one place:

- The JTBD (from `working/01-jtbd-and-knowledge-map.md`)
- The knowledge map (same file)
- The required source roles (same file)
- The kept and trim sources from `working/03-evaluation.yaml` — titles, sections, ratings, rationales, and the drafted curator notes

Read these together. The tests are about how the corpus *as a whole* serves the JTBD. You can't run them well by looking at sources one at a time.

---

## Test 0 — The core-action fit test

Name the exact action in the JTBD. Then ask whether the kept sources directly
teach or demonstrate that action, or merely orbit it.

This test exists because coverage can pass while the job still fails. A corpus
for "turn moodboards into HTML interface concepts" can technically cover
"visual hierarchy," "design systems," "CSS layout," and "process," yet still
miss the action of reading visual references, extracting transferable qualities,
and creating a new interface direction.

### How to run it

1. Quote or paraphrase the JTBD's core action in `working/04-quality-checks.md`.
2. Count kept sources by `jtbd_fit`: `direct / bridge / background`.
3. If older entries lack `jtbd_fit`, infer it from title, rationale, note, and
   section, then update `working/03-evaluation.yaml`.
4. List which kept sources are direct. A healthy action-oriented mixtape needs
   at least two direct sources, and each central knowledge-map section needs
   either one direct source or a written reason why bridge coverage is enough.
5. For curator-named anchors, say whether you retrieved rationale/process
   evidence or only a homepage/gallery/metadata pointer.

If direct coverage is weak, take one action:

- Backfill direct sources within the Phase 5 search budget, or
- Document the gap and recommend re-running Candidate discovery with
  action-specific search terms.

### Required output

```
## Test 0 — Core-action fit

Core action: [verb phrase from JTBD]
Distribution: X direct / Y bridge / Z background
Direct sources: [Source titles]
Finding: [pass / weak coverage / gap]
Action: [kept, backfilled, or documented rerun recommendation]
```

---

## Test 1 — The redundancy test

Read each kept source's role in the mixtape. Are any two sources making essentially the same point?

If yes, keep the better one and cut the other. Volume is not a virtue. Five excellent sources beat fifteen good ones.

**The test in action:** for each kept source, write down in one sentence what unique contribution it makes to the corpus. If two sources produce the same sentence (or near-same), one of them is redundant. Drop the weaker.

Redundancy is different from coverage. Two sources covering the same topic from different angles or with different evidence is coverage, not redundancy. The test is whether the AI would generate meaningfully different output with both vs. with only one. If not, redundant.

> TODO: Insert a worked example from the Cowork synthesis where two sources were collapsed because of redundancy. The Van Slyck talk appearing in transcripts 02 and 03 is a known case — same talk uploaded twice. Show how the redundancy was named and resolved.

---

## Test 2 — The coverage test

Walk through the knowledge map. For each section, ask: does this section have at least one strong source?

If a section has zero sources, two responses are valid:

- **Fill it.** Return to Phase 2 for that section. Find candidates. Fetch them. Evaluate them. Then come back to Phase 5.
- **Explicitly note the gap.** Some sections of a domain genuinely don't have credible sources available, or fall outside what this mixtape is trying to do. Document this in `working/04-quality-checks.md`: "This mixtape doesn't cover X — see [other mixtape] for that, or note that the field doesn't have a canonical reference here."

A mixtape that silently omits a major area is worse than one that says explicitly "this mixtape doesn't cover X." The latter sets expectations correctly; the former misleads the consuming AI by implying the corpus is complete.

If a section has sources but they're all rated 2 or 3, that's a weak-coverage signal — the section is technically populated but the sources aren't load-bearing. Either find stronger sources or note the weakness.

> TODO: Insert a worked example. If the Cowork synthesis had a section that needed reinforcement after first-pass evaluation, show the coverage check that surfaced it and how the gap was addressed.

---

## Test 3 — The disagreement test

Find the strongest claim made by sources in the mixtape. Is there someone credible who disagrees?

If yes, either include the disagreement or explicitly note that the mixtape takes a position.

A corpus where every source agrees teaches the AI no nuance. The AI then can't see the contested-ness of contested questions, which means it can't represent them honestly downstream. The fix is to include at least one strong dissenting source on the most-contested point in the corpus.

If the topic is genuinely uncontested — the W3C HTML spec, for example, isn't really contested in the way a design philosophy might be — that's fine, and the working notes should say so. The test isn't "manufacture disagreement"; it's "if disagreement exists, surface it; if it doesn't, document why."

**The test in action:** identify the strongest claim in the corpus. Search for "[claim] critique" or "[claim] is wrong" or "alternative to [claim]." If you find a credible critique you haven't included, you've found your dissenting source. Add it.

> TODO: Insert a worked example from CLI/TUI research. The corpus probably had a contested-question moment — possibly around whether CLIs should follow Unix conventions strictly vs. break them deliberately, or around accessibility being assumed-by-default vs. explicitly designed-for. Surface the actual disagreement and how it was handled.

---

## Test 4 — The framing-gap test

Step back from individual sources. Is there a major perspective on this topic missing entirely — not just an individual source, but a whole way of thinking about the JTBD?

If yes, the framing was too narrow. Expand the knowledge map and return to Phase 2.

**This test is the highest-leverage of the six and the one most curators skip.** Single-pass curation tends to produce mixtapes that are well-covered within one narrow framing. The framing-gap test catches what the first four phases couldn't see.

### How to run it (required steps, in order)

This test cannot be answered yes/no. Earlier versions of this methodology allowed a casual judgment ("does the framing feel wide enough?") and the answer was almost always "yes" — because the test, run that way, has no force. Run it as follows instead:

1. **Name 2–3 candidate perspectives that could exist *outside* the corpus.** Write them down in `working/04-quality-checks.md` before looking at the keep-list. Common candidates: designer voice, accessibility-first viewpoint, security/abuse perspective, non-English-speaking practitioner, contrarian who disagrees with the dominant framing, an adjacent discipline that owns part of the problem space (e.g. for a CLI mixtape: visual-design tradition, technical-writing tradition, OS-vendor convention tradition).

   The named perspectives should be specific enough to act on. "User perspective" is too vague. "iOS Human Interface Guidelines designer perspective on form-field labels" is actionable.

2. **For each named perspective, classify coverage in two dimensions, not one.** The earlier single-axis judgment ("covered / uncovered / partial") let a corpus mark a perspective `covered` just because some kept source addressed its concerns — even when no source actually argued *for* the perspective as a stance. That collapse is the failure mode this split exists to prevent.

   - **Stance** — is there a verified, fetchable source whose author explicitly *argues for* this perspective as a design center of gravity? Mark `stance-represented` or `stance-absent`.
   - **Concerns** — do any kept sources address the perspective's operational concerns mechanically, even while arguing from a different stance? Mark `concerns-addressed` or `concerns-absent`.

   A perspective is fully `covered` only when its stance is represented. `concerns-addressed` alone is a **soft gap** — the corpus internalizes the concern without giving the consuming AI a worked rival argument to cite. Soft gaps are real gaps; they show up later as synthesis bias.

3. **Unverified sources DO NOT count as coverage.** A source whose Phase 4 curator note acknowledges unfetched content, broken URLs, paywalled access, or transcripts that couldn't be obtained cannot satisfy a perspective slot. "We have one designer voice, but her transcript wasn't fetchable" is the same as "we have no designer voice." Tokenism passes the test cheaply and silently — don't allow it.

4. **For each perspective that is not `stance-represented`, take exactly one of three actions:**

   a. **Search for a stance source.** Return to Phase 2 with vocabulary derived from the perspective itself: *who would argue for this position, what would they write, where would they publish?* This is different from a topic search — you're looking for an opinionated author, not a neutral reference. Examples: "Brandur Leach on terminal density," "Rob Pike on Unix philosophy," "Don Norman on visceral design," not "CLI scriptability best practices." Spend one search pass. If you find a verified source that fits, add it to the keep-list.

   b. **Argue that the concerns are sufficient.** If concerns are addressed and you believe the perspective doesn't need a separate stance source — because the operational disagreement is genuinely thin — *write the argument explicitly*: "Concerns covered by [Sources N, M] inside the [dominant] stance. The dissenting stance is real but its operational implications are absorbed; no separate source needed."

      This is not a free pass. The methodology requires you to *state* the trade-off. If you can't articulate why concerns are sufficient without a stance source, you don't actually have the argument and should fall back to (a) or (c).

   c. **Flag the bias explicitly.** If no source is found and the concerns aren't sufficient, write in `working/04-quality-checks.md`:

      > *Perspective: [name] — stance unrepresented after one search pass. Corpus argues from [dominant stance] but no source argues for [rival stance]. Recommendation: ship as-is and document the bias in the synthesis, OR weaken the JTBD claim that requires the rival stance.*

   Do NOT silently leave a perspective `stance-absent` with no defense. The synthesis is what the consuming AI reads first; an undocumented bias becomes the AI's silent default.

### The output format

`working/04-quality-checks.md` must include a top-level section like this:

```
## Perspectives audit

- Designer voice — stance-represented (Source 14: Gillet, "Designing for CLI", verified) ✓
- Accessibility-first viewpoint — stance-absent, concerns-absent. Search added GitHub accessibility post → now stance-represented (Source 3) ✓
- Power-user / Unix-purist stance — stance-absent, concerns-addressed by Heroku/Vercel/Van Slyck.
  - Backfill attempt: searched "Brandur Leach terminals density," found brandur.org/interfaces. Verified, added (Source N). ✓
  OR
  - Argued sufficient: concerns covered by Heroku's grep-first rule + Vercel's stdout contract + Van Slyck's "friendly to humans AND scripts" pillar. The Unix-purist stake is that terseness should be the *center of gravity*, not a preserved capability — we're not adopting that. The corpus's framing covers the operational concerns without representing the stance; synthesis will name this trade-off explicitly.
```

This format is checked by the TUI: a quality-checks file without a "Perspectives audit" section signals that the test wasn't run honestly.

### Why this strictness

The earlier loose version of this test failed quietly in practice. A real session keeping one designer source whose content was never fetched produced a corpus the curator described as "engineering-heavy, one designer voice unverified" — and the quality-check still passed because "1 source ≥ 1 = covered." Naming perspectives in advance and refusing to count unverified sources is what makes the test do work instead of legitimize tokenism.

If the framing gap is real, accept the cost of going back. Phase 2 is cheap relative to shipping a mixtape with a hole in it.

> TODO: Insert the canonical worked example from the Cowork synthesis. OpenClaw's initial 50-source list was anchored in "CLI/TUI implementation" framing and missed "the entire wizard/form-design tradition (Form Design Patterns, NN/g articles, Tidwell, Cooper)." Cowork ran the framing-gap test, named the missing tradition, added 43 net-new sources, and reframed the corpus. This is the textbook framing-gap-test rescue. Pull the specific moment the gap was identified and the specific sources added.

---

## Test 5 — Source-kind balance

Look at the distribution of `kind` values across the keep-list. Every source in `tape.yaml` should carry one of: `reference / principle / prescription / example`.

The four kinds represent four argumentative roles. A corpus missing one of them has a structural gap, even when every knowledge-map section is technically populated:

- **0 `reference` sources** → the consuming AI has no substrate to cite when defending a convention. The mixtape becomes opinion against opinion when challenged. The fix is at least one authoritative source — a spec, an official manual, a widely-adopted standard, an RFC — that the AI can point at as "this is canon, not just our preference."
- **0 `principle` sources** → the synthesis has no philosophical anchor. The mixtape can describe rules but can't justify the worldview behind them. The fix is a stance-setting essay or manifesto.
- **0 `prescription` sources** → the AI has no concrete rules to enforce. The synthesis becomes vibes without operational consequences. The fix is a style guide, rule-set, or explicit guideline doc.
- **0 `example` sources** → the AI has no taste calibration. It can recite rules but can't recognize good when it sees it. The fix is a worked instance, case study, or annotated specimen.

Imbalance is not a failure. A 7-principle / 2-prescription / 1-reference / 0-example corpus may be exactly right for a philosophy-heavy JTBD. But a zero in any column should be a **decision, not an accident.** If you accept a zero, write the defense in `working/04-quality-checks.md`:

> *Kinds audit: 7 principle / 9 prescription / 5 example / 0 reference. Accepted: this JTBD is about voice and microcopy, not architectural conventions. No reference source is needed because the synthesis doesn't make authority claims that would require backing.*

If the zero isn't defensible, first make sure every existing kept/trim source has been assigned a kind. Only then run one source-kind backfill search before shipping. The same rule as Test 4(a) applies: search for the role the kind plays (an authoritative spec, an opinionated essay, a tight rule-set, a worked example), not for the topic generically.

### Required output

`working/04-quality-checks.md` must include a Test 5 section like this:

```
## Test 5 — Source-kind balance

Distribution: X reference / Y principle / Z prescription / W example

[If any kind is zero: either a backfill action OR a defense paragraph in the form above.]
```

---

## Test 6 — Note-quality smell test

Read every kept/trim source note in `working/03-evaluation.yaml`. The note is the consuming AI's instruction for how to weight and use that source. A source with a vague note is a bookmark, not curation.

A useful curator note must do three jobs:

- **Use cue:** tell the AI how to read or apply the source.
- **Value/bar:** name the concrete value, standard, or quality bar the source contributes.
- **Boundary:** name one limitation, scope boundary, or reason not to over-weight it.

Thin notes often look like "useful background," "good overview," or "skim for examples." They may be true, but they do not tell the AI what to do with the source. Repair them before synthesis by updating `working/03-evaluation.yaml`; do not search or fetch just to improve note prose. Use the already gathered title, rationale, source type, section, rating, and partial content.

### Required output

`working/04-quality-checks.md` must include a Test 6 section like this:

```
## Test 6 — Note-quality

Checked: N kept/trim notes
Repaired: M notes

- Source/title/URL — repaired missing boundary; note now names [scope].
```

If no notes were repaired, write one sentence explaining why the notes already carry a use cue, value/bar, and limitation.

---

## Test 7 — Source-role fit

This test asks whether the corpus has the evidence roles the future AI actually
needs. It catches the failure mode where a source list technically covers every
section and every source `kind`, but still cannot do the job well.

Source roles are capability-specific evidence jobs named in the Capability
Brief. They are not source `kind`s and not section labels. A source can be an
`example` kind without satisfying the required "worked high-craft case study"
role if the fetched content is only a homepage, gallery index, or shallow
portfolio blurb. A source can be a `reference` kind without satisfying the
required "medical authority" role if it is a secondary explainer instead of a
primary clinical or institutional source.

### How to run it

1. Read `working/01-jtbd-and-knowledge-map.md` and copy the Required source
   roles from the Capability Brief into `working/04-quality-checks.md`.
2. If the project was framed before Required source roles existed, infer 4–7
   roles from the core action and write them into the quality report before
   scoring coverage.
3. For each role, count kept/trim sources whose fetched evidence actually
   satisfies that role. Count by content, not by title, URL, section, or generic
   kind.
4. Compare coverage to the minimum named in the Capability Brief. If no minimum
   was named, use these defaults:
   - 2 strong sources for ordinary required roles.
   - 3 substantive worked examples/case studies when the capability depends on
     taste, visual output, craft judgment, or recognizing quality.
   - primary/official sources for legal, medical, financial, or safety-sensitive
     authority roles.
5. A material role with weak coverage must either be improved within the Phase 5
   budget or written to `working/05-operating-fit-audit.md` with
   `status: improvement_recommended`.

"Light but not absent" is not a pass for a required source role. It is a weak
coverage finding.

### Required output

`working/04-quality-checks.md` must include:

```
## Test 7 — Source-role fit

Required roles:
- Core-action method — minimum: 2; current: 3; status: pass; evidence: [Source titles]
- Worked craft examples — minimum: 3; current: 1; status: weak; evidence: [Source title]

Action: [kept / backfilled / wrote working/05-operating-fit-audit.md]
```

If a required role is missing or weak, take one of three actions:

- **Backfill now.** Search for the role directly, add a verified candidate, evaluate it, and update the keep-list.
- **Document the limitation.** If credible sources are unavailable, write a concrete limitation and what custom source the curator should add.
- **Narrow the capability claim.** If the role is not actually needed, update the Capability Brief and Required source roles so Phase 1 tells the truth.

Do not leave a missing required role as "covered by the overall corpus." That is exactly how generic-looking mixtapes pass while still failing the user's job.

---

## Test 8 — Capability-pattern fit

Some capabilities have a specialized evidence contract beyond generic source
roles. Phase 1 should name these as `Capability pattern: ...` in the Capability
Brief. Test 8 asks whether the final keep-list still satisfies that pattern, or
whether the corpus drifted into a generic topic pack.

If no capability pattern was named, write:

```
## Test 8 — Capability-pattern fit

Pattern: none
Finding: pass. No specialized capability pattern was named in the Capability Brief.
Action: none.
```

Do not invent a pattern in Phase 5 just to make the test look busy. If Phase 1
clearly missed a pattern and the omission affects corpus quality, document that
as a framing gap and write an operating-fit audit.

### Reference-translation pattern

Use this section when the Capability Brief says `Capability pattern:
reference-translation`, or when Phase 1 should have named it because the job
involves images, moodboards, visual references, inspiration, examples, style,
art direction, or translating one medium/domain into another output.

This pattern fails when a corpus becomes mostly about the target output medium.
For example, an art-direction-to-web mixtape can pass ordinary "web design"
coverage while still failing the real job: reading paintings, architecture,
photos, posters, sports identity, or other references and translating their
underlying qualities into product-design language.

Run four checks:

1. **Input/reference-domain coverage.** Count substantive kept/trim sources from
   at least two input/reference domains outside the target output medium. Count
   by content evidence, not by passing mention. For web/UI targets, CSS,
   accessibility, token, component, and design-system docs do not count here.
2. **Translation-method coverage.** Count direct kept/trim sources that teach or
   demonstrate the move from source qualities to target decisions. Generic
   inspiration galleries, moodboard primers without translation, and target
   mechanics do not count.
3. **Caller-handoff coverage.** Confirm the corpus supports a concrete runtime
   output shape: observations, interpretation, carry-forward rules, product or
   implementation vocabulary, constraints, avoid-list, validation checks, and
   clarification questions.
4. **Constraint balance.** Confirm target-medium constraints are present when
   needed but do not outnumber and replace the direct reference-reading and
   translation evidence in the core action.

Default minimums for reference-translation:

- At least 2 distinct input/reference domains outside the target output medium.
- At least 3 substantive direct sources from those input/reference domains or
  from cross-domain translation examples.
- At least 2 direct translation-method sources.
- At least 1 handoff/template/vocabulary source that shows how to make the
  translation executable for the caller.

If any minimum is weak or missing, write `working/05-operating-fit-audit.md` with
`status: improvement_recommended`. Do not pass the corpus because general design,
implementation, accessibility, or platform sources are strong.

### Required output

`working/04-quality-checks.md` must include:

```
## Test 8 — Capability-pattern fit

Pattern: reference-translation
Input/reference domains: [domain A: sources], [domain B: sources]
Translation-method sources: [source titles]
Caller-handoff sources: [source titles]
Constraint balance: [pass / weak / gap]
Finding: [pass / weak coverage / gap]
Action: [kept / backfilled / wrote working/05-operating-fit-audit.md]
```

---

## What to do when a test fails

Don't paper over failures. The point of running the tests is to find problems while they're still cheap to fix.

- **Redundancy fails:** drop the weaker source, update `working/03-evaluation.yaml` to reflect the change, document the decision in `working/04-quality-checks.md`.
- **Coverage fails:** return to Phase 2 for the missing section. Fetch new candidates. Evaluate them. Come back to Phase 5 and re-run all eight tests on the new keep-list.
- **Disagreement fails:** find one strong dissenting source. Fetch it. Evaluate it. Add it to the keep-list with a note that names its role as counterpoint.
- **Framing-gap fails:** this is the expensive one. Expand the knowledge map. Return to Phase 2. Find candidates for the new sections. Fetch, evaluate. Come back to Phase 5 with a substantially revised keep-list and re-run all eight tests.
- **Note-quality fails:** rewrite weak notes in `working/03-evaluation.yaml` so each one carries a use cue, value/bar, and boundary. Do this from existing evidence; do not open a new search loop.
- **Source-role fit fails / Operating-fit fails:** write `working/05-operating-fit-audit.md` with the missing source role, why it matters, concrete search lanes for an improvement pass, and a short list of custom sources the user could add if public research remains thin.

Each backtrack is a sign the methodology is working, not failing. The first four phases produced a corpus that worked from one angle; Phase 5 found the angle that wasn't covered. Better to find it here than in the empirical test (Phase 8) or — worse — silently, when the mixtape is in use.

---

## Output: `working/04-quality-checks.md`

Document each test, the finding, and the action taken. Even tests that passed without changes should be noted — "core-action fit: 4 direct sources; no backfill" or "redundancy test: no redundant sources found." The working notes are part of the methodology; they show what work was actually done.

If an improvement pass is recommended, also write `working/05-operating-fit-audit.md`:

```yaml
status: improvement_recommended
gap: The missing source role or weak evidence role.
why_it_matters: Why this gap would hurt a future AI agent using the corpus.
recommended_pass:
  - Search lane or source type Liner should look for.
  - Search lane or source type Liner should look for.
user_can_add:
  - Concrete custom source the user could add if web research stays thin.
skip_note: What remains weaker if the user skips for now.
```

> TODO: Insert a sample `working/04-quality-checks.md` from the Cowork synthesis. The full structure with each test, the finding, and the resulting action is the most useful worked example in this file.

---

## A note on fresh attention

Phase 5 benefits from running with fresh attention. The first four phases build up a mental model of the corpus that biases how you see it — you remember why you kept each source, which makes redundancy easier to miss; you remember the candidate searches you ran, which makes the framing-gap easier to miss.

If the conversation has been long, consider starting a separate chat with just the JTBD, the knowledge map, the required source roles, and the kept-source list (titles + notes only, no source content). Run the quality tests there. The fresh chat doesn't carry the gathering pass's biases.

This is optional, not required by the methodology. Mention the option to the curator. Quick mode users may prefer to continue in the same conversation for speed; methodology mode users often benefit from the reset.
