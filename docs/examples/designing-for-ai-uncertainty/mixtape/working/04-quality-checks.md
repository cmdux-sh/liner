# Quality checks (Phase 5)

## Test 0 — Core-action fit

Core action: Convert an AI feature's actual error modes, evidence quality, user expertise, action stakes, and reversibility into a specific uncertainty treatment, inspectable evidence, verification and correction controls, and a deferral rule.

Distribution: 22 direct / 0 bridge / 0 background among kept sources.

Direct sources include *Trust in Automation: Integrating Empirical Evidence on Factors That Influence Trust*, *Effect of Confidence and Explanation on Accuracy and Trust Calibration in AI-Assisted Decision Making*, *RARR*, *Errors + Graceful Failure*, the NIST Generative AI Profile, and *The Effects of Explanations on Automation Bias*. Every central knowledge-map section has at least two direct kept sources.

Finding: Pass. The keep-list does not merely discuss trust; it teaches or demonstrates how to diagnose reliance, represent uncertainty, expose support, enable repair, impose safeguards, and test behavior.

Curator anchors: Google PAIR's *Explainability + Trust* was retrieved with substantive rationale and examples. The Microsoft HAX landing page and Apple Machine Learning HIG did not yield substantive content and remain dropped; they do not count as direct coverage or taste anchors.

Action: Kept the direct corpus; no core-action backfill required.

## Test 1 — Redundancy

Two duplicate representations were removed from the selected set: the IIS page for *To Trust or to Think* was dropped in favor of the canonical arXiv entry in the failure-focused section, and the PDF copy of *Guidelines for Human-AI Interaction* was dropped in favor of the readable Microsoft Research entry in the safeguards section.

The remaining apparent clusters are complementary rather than interchangeable: ALCE evaluates citation support, GopherCite tests claim-plus-quote support, RARR demonstrates evidence-driven repair, WebGPT supplies a browser-and-reference system case, and the fine-grained-reward paper contributes a citation-failure taxonomy. Likewise, NIST AI RMF supplies the general risk substrate while the Generative AI Profile supplies confabulation, refusal, recourse, and staged-release guidance.

Action: Changed the two duplicate candidates to `dropped`; retained the distinct evidence roles.

## Test 2 — Coverage

All six knowledge-map sections have strong kept coverage:

- The reliance decision — 3 kept; direct empirical synthesis, algorithm-aversion evidence, and a learnable error-boundary study.
- Representing uncertainty without false precision — 6 kept; calibrated and miscalibrated confidence, explanation styles, model and human confidence, and practitioner translation.
- Showing evidence people can inspect — 4 kept; citation evaluation, claim-level repair, quote support, and product patterns.
- Verification, correction, and recovery — 3 kept; control, graceful failure, and a question-driven design method.
- Abstention, deferral, and escalation — 4 kept; NIST, W3C, and practical interaction guidance.
- Evaluation and failure-focused critique — 2 kept, both rated 5; explanation-induced automation bias and cognitive-forcing tradeoffs, supported by three trims.

Finding: Pass. No knowledge-map section is empty or supported only by low-rated material.

Action: No coverage backfill required.

## Test 3 — Disagreement

Strongest claim tested: exposing confidence, explanations, and supporting evidence can help people rely on imperfect AI appropriately.

Credible counterevidence is included. *The Effects of Explanations on Automation Bias* finds that explanations can fail to reduce erroneous agreement and sometimes increase automation bias. *Too Sure for Our Own Good* shows that miscalibrated confidence produces both over-reliance and under-reliance. *To Trust or to Think* shows that effective checking friction can be less acceptable and can benefit subgroups unequally. *Algorithm Aversion* adds the opposite failure mode: visible errors can cause people to abandon automation that still outperforms them.

Finding: Pass. The corpus takes the position that no confidence cue, explanation, citation, or friction pattern is protective by default; each must be validated against behavioral reliance and task consequences.

Action: Retained the counterevidence rather than manufacturing a single pro-disclosure consensus.

## Test 4 — Framing-gap

Before judging the keep-list, three outside perspectives were named: an accessibility-first interaction stance, a skeptical human-factors stance that explanation can worsen reliance, and the stance of frontline domain experts who remain accountable for consequential decisions.

### Perspectives audit

- Accessibility-first interaction — **stance-represented** by *WCAG 2.2* and *Understanding Success Criterion 3.3.4* (verified) ✓; **concerns-addressed** by their requirements for perceivable status, review, correction, confirmation, and reversibility, plus the cognitive-forcing paper's subgroup finding.
- Skeptical human factors — **stance-represented** by *The Effects of Explanations on Automation Bias*, *Too Sure for Our Own Good*, and *To Trust or to Think* (verified) ✓; **concerns-addressed** through behavioral error detection, automation bias, acceptability, task load, and unequal effects rather than trust ratings alone.
- Frontline domain experts accountable for high-stakes action — initially **stance-absent**, although concerns were addressed by NIST, ICO guidance, and the question-driven healthcare case. Backfill search vocabulary: “clinician perspective AI decision support uncertainty verification workflow” and “clinician perspective AI uncertainty verification interface case study.” The verified PubMed abstract for *AI-driven decision support systems and epistemic reliance* was added as a trim source; it argues that clinicians remain epistemic authorities and responsibility holders who must judge reliability on their own terms. It is now **stance-represented** (partial substantive content) ✓ and **concerns-addressed**. Its clinical scope remains an explicit boundary.

Finding: Pass after one bounded stance-source backfill. The dominant product-design framing now includes authoritative accessibility, critical human factors, and an accountable practitioner voice.

Action: Added the verified clinician-perspective source to the candidate ledger and evaluation; stopped the search loop after the single perspective pass.

## Test 5 — Source-kind balance

Distribution: 3 reference / 14 principle / 9 prescription / 7 example.

Finding: Pass. References provide authoritative substrate through NIST and WCAG; principles explain reliance and contested behavioral effects; prescriptions turn those findings into interaction rules; examples demonstrate systems, studies, and product cases. The principle-heavy shape is appropriate because the JTBD depends on empirical distinctions between perceived trust and behavioral reliance, while no kind is absent.

Action: Assigned `kind` to every kept/trim entry before counting; no source-kind backfill required.

## Test 6 — Note-quality

Checked: 33 kept/trim notes.

Repaired: 0 notes.

Every selected note already carries an explicit use cue (`Role`), a concrete value or quality bar (`Value`), and a limitation or weighting boundary (`Limitations`). The newly added clinician-perspective note uses the same structure and limits transfer from its 13-person UK maternity-care study and abstract-only retrieval.

## Test 7 — Source-role fit

Required roles:

- Calibrated-reliance foundations — minimum: 2, including 1 primary or foundational source; current: 4 strong; status: **pass**; evidence: *Trust in Automation: Integrating Empirical Evidence*, *Algorithm Aversion*, *Beyond Accuracy*, and *To Trust or to Think*.
- Uncertainty representation and confidence limits — minimum: 2, including 1 primary empirical source and 1 confidence-limit source; current: 6 strong; status: **pass**; evidence: *Effect of Confidence and Explanation*, *Too Sure for Our Own Good*, *Using AI Uncertainty Quantification*, and *Are You Really Sure?*.
- Evidence and verification interaction methods — minimum: 2, including 1 evaluated method or primary study; current: 5 substantive; status: **pass**; evidence: ALCE, RARR, GopherCite, WebGPT, and PAIR *Patterns*.
- Worked interface cases across core output types — minimum: 3 substantive examples spanning at least 2 output types and more than one organization or author; current: 3; status: **pass at the minimum**; evidence: WebGPT's referenced answer workflow, the healthcare prototype in *Question-Driven Design Process*, and the fetched Google Photos control-and-recovery case within PAIR's case-study entry. Salesforce's promotional examples were not counted toward the minimum.
- Deferral, safeguards, and human-control authority — minimum: 2, including 1 primary/official authority and 1 product-behavior translation; current: 5 strong; status: **pass**; evidence: NIST AI RMF, NIST Generative AI Profile, W3C 3.3.4, PAIR *Errors + Graceful Failure*, and Microsoft *Guidelines for Human-AI Interaction*.
- Critique and validation evidence — minimum: 2, including 1 failure-focused source and 1 reusable evaluation method; current: 5 substantive; status: **pass**; evidence: *The Effects of Explanations on Automation Bias*, *To Trust or to Think*, *Too Sure for Our Own Good*, ALCE, and *Question-Driven Design Process*.

Finding: Pass. The worked-case role is the narrowest pass and should not be diluted during final tape assembly: WebGPT, the question-driven healthcare prototype, and the Google Photos case must remain available if this role is claimed.

Action: No operating-fit audit marker written; the evaluated corpus satisfies every named minimum.

## Test 8 — Capability-pattern fit

Pattern: calibrated-reliance interaction design.

This is a named specialized pattern, but not `reference-translation`. Its evidence contract is satisfied as follows:

- Input diagnosis — pass: trust/reliance research, NIST, WCAG, and the clinician-perspective source support reading error modes, evidence quality, user expertise, stakes, reversibility, and responsibility before selecting a treatment.
- Translation method — pass: *Question-Driven Design Process*, *Errors + Graceful Failure*, *Explainability + Trust*, and *Guidelines for Human-AI Interaction* teach how to turn those conditions into uncertainty cues, evidence, correction, fallback, and handoff behavior.
- Caller handoff — pass: the corpus supports the Capability Brief's concrete output shape: context/risk read, uncertainty treatment, evidence beside output, verification/correction, deferral/escalation, sourced rationale, validation checks, and one blocking clarification question when needed.
- Constraint balance — pass: empirical principles do not replace product action. The 14 principles are balanced by 9 prescriptions, 7 examples, and 3 authoritative references; all six sections retain direct kept evidence.

Finding: Pass. The corpus can help a future agent make a bounded interface recommendation and specify how to test it, rather than merely discussing trustworthy AI.

Action: No pattern-specific backfill or operating-fit audit required.
