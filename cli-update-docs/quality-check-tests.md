# Quality-check tests

Loaded by the `curating-mixtapes` skill at the start of Phase 5 (Quality checks).

This file describes the four tests every mixtape should pass before moving to synthesis. Apply them deliberately, in this order. Write findings to `working/04-quality-checks.md`.

Phase 5 is the methodology's correction for single-pass curation. The first four phases work forward — frame, find, fetch, evaluate. Phase 5 works backward — looking at the whole keep-pile and asking what's wrong with it. The two passes do different work. If you skip this phase, the mixtape's blind spots ship with the mixtape.

The framing-gap test is the highest-leverage of the four and the one most curators skip.

---

## Setting up the phase

Before running the tests, gather the inputs in one place:

- The JTBD (from `working/01-jtbd-and-knowledge-map.md`)
- The knowledge map (same file)
- The kept and trim sources from `working/03-evaluation.yaml` — titles, sections, ratings, rationales, and the drafted curator notes

Read these together. The tests are about how the corpus *as a whole* serves the JTBD. You can't run them well by looking at sources one at a time.

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

**This test is the highest-leverage of the four and the one most curators skip.** Single-pass curation tends to produce mixtapes that are well-covered within one narrow framing. The framing-gap test catches what the first four phases couldn't see.

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

If the zero isn't defensible, run one search pass for a source of the missing kind before shipping. The same rule as Test 4(a) applies: search for the role the kind plays (an authoritative spec, an opinionated essay, a tight rule-set, a worked example), not for the topic generically.

### Required output

`working/04-quality-checks.md` must include a Test 5 section like this:

```
## Test 5 — Source-kind balance

Distribution: X reference / Y principle / Z prescription / W example

[If any kind is zero: either a backfill action OR a defense paragraph in the form above.]
```

---

## What to do when a test fails

Don't paper over failures. The point of running the tests is to find problems while they're still cheap to fix.

- **Redundancy fails:** drop the weaker source, update `working/03-evaluation.yaml` to reflect the change, document the decision in `working/04-quality-checks.md`.
- **Coverage fails:** return to Phase 2 for the missing section. Fetch new candidates. Evaluate them. Come back to Phase 5 and re-run all four tests on the new keep-list.
- **Disagreement fails:** find one strong dissenting source. Fetch it. Evaluate it. Add it to the keep-list with a note that names its role as counterpoint.
- **Framing-gap fails:** this is the expensive one. Expand the knowledge map. Return to Phase 2. Find candidates for the new sections. Fetch, evaluate. Come back to Phase 5 with a substantially revised keep-list and re-run all four tests.

Each backtrack is a sign the methodology is working, not failing. The first four phases produced a corpus that worked from one angle; Phase 5 found the angle that wasn't covered. Better to find it here than in the empirical test (Phase 8) or — worse — silently, when the mixtape is in use.

---

## Output: `working/04-quality-checks.md`

Document each test, the finding, and the action taken. Even tests that passed without changes should be noted — "redundancy test: no redundant sources found." The working notes are part of the methodology; they show what work was actually done.

> TODO: Insert a sample `working/04-quality-checks.md` from the Cowork synthesis. The full structure with each test, the finding, and the resulting action is the most useful worked example in this file.

---

## A note on fresh attention

Phase 5 benefits from running with fresh attention. The first four phases build up a mental model of the corpus that biases how you see it — you remember why you kept each source, which makes redundancy easier to miss; you remember the candidate searches you ran, which makes the framing-gap easier to miss.

If the conversation has been long, consider starting a separate chat with just the JTBD, the knowledge map, and the kept-source list (titles + notes only, no source content). Run the four tests there. The fresh chat doesn't carry the gathering pass's biases.

This is optional, not required by the methodology. Mention the option to the curator. Quick mode users may prefer to continue in the same conversation for speed; methodology mode users often benefit from the reset.
