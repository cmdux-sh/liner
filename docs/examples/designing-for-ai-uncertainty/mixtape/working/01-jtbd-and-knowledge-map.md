# JTBD and knowledge map

## Original capability goal

When an AI feature in my product can be wrong, I want to show how far to trust it and let people check its work, so they rely on it the right amount.

## Capability Brief

### Reusable AI capability

Help a product team design one AI-assisted workflow—such as a generated summary, suggested answer, extracted field, or recommendation—so people can calibrate their reliance when an output may be fluent, plausible, and wrong. Given the feature context, intended audience, likely failure modes, available supporting evidence, and optionally a proposed interface, the future agent should recommend how to communicate uncertainty, expose support, enable verification and correction, and defer or escalate when answering would be unsafe.

Capability pattern: **calibrated-reliance interaction design**. This is not generic “responsible AI” advice and not a compliance review. It translates the uncertainty and consequence profile of a specific AI output into interface behavior that helps a person decide whether to accept, verify, correct, or decline it.

### Exact runtime output contract

For one AI feature, return a concrete, source-grounded design recommendation containing:

1. **Context and risk read:** the user, decision or action, plausible failure modes, reversibility, stakes, and assumptions. Separate supplied facts and observable interface facts from inferences and unknowns.
2. **Uncertainty treatment:** the words, visual treatment, and placement to use; whether uncertainty should be qualitative, categorical, numeric, or omitted; and why that representation fits the evidence and user decision.
3. **Evidence beside the output:** what provenance, citations, excerpts, source links, input-to-output trace, or other support to expose; what each item lets the person verify; and what must not be implied by its presence.
4. **Verification and correction path:** the shortest meaningful way to inspect the basis, compare against authoritative material, edit or reject the result, report a problem, and recover from a mistake. Do not make “verify this” a warning without an actionable path.
5. **Fallback, deferral, and escalation:** what the feature should do when evidence is absent, contradictory, stale, outside scope, or too weak for the stakes; when it should abstain or route to a person; and how that state should be explained.
6. **Recommendation rationale:** cite the corpus sources supporting each major choice and state the tradeoff accepted—for example, speed versus scrutiny, simplicity versus precision, or helpfulness versus over-reliance.
7. **Validation checks:** specify what to test with representative users and failure cases, including whether people interpret the uncertainty correctly, notice and use evidence, catch plausible errors, and know when not to proceed.

Make the best grounded recommendation when the audience, stakes, evidence, and action are sufficiently known. If stakes or audience are unknown and the answer could materially change the safe design, ask **one targeted clarifying question** before recommending. If a non-blocking detail is missing, proceed with a bounded recommendation, label the assumption and confidence limit, and state what evidence would verify it.

### Internal job-to-be-done

When a product team is designing a generated summary, suggested answer, extracted datum, or recommendation that a person may act on, help it convert the feature’s actual error modes, evidence quality, user expertise, action stakes, and reversibility into a specific interface treatment—uncertainty language or visuals, supporting evidence, verification and correction controls, and a deferral rule—so users neither accept plausible errors too readily nor abandon useful automation unnecessarily.

### Inferred research lanes

- **Calibrated reliance and human factors:** appropriate trust versus maximal trust; automation bias, over-reliance, under-reliance, cognitive load, and how expertise or time pressure changes checking behavior.
- **Uncertainty communication:** qualitative language, categorical states, confidence scores, ranges, distributions, calibration, comprehension, and cases where numeric confidence creates false precision.
- **Evidence, provenance, and explainability:** citations and source excerpts, traceability to inputs, evidence quality and freshness, limits of explanations, and interfaces that support checking rather than merely reassuring.
- **Verification, correction, and recovery:** progressive disclosure, inspect-and-compare flows, editable outputs, feedback and contestability, error recovery, and reversible action design.
- **Abstention, deferral, and action safeguards:** insufficient or conflicting evidence, out-of-scope requests, human handoff, confirmation before consequential actions, and boundaries for regulated or high-stakes contexts.
- **Evaluation in realistic use:** comprehension and calibration measures, behavioral reliance, error-detection tasks, representative failure cases, longitudinal trust, accessibility, and subgroup differences.

### Required source roles

Sources may satisfy more than one role when their substantive evidence genuinely covers both, but every role must meet its own minimum.

1. **Calibrated-reliance foundations**
   - **Why it matters:** The target is appropriate reliance, not making an AI seem trustworthy or explaining it for its own sake.
   - **Good evidence:** Peer-reviewed human-factors or HCI research that defines calibration, distinguishes trust from reliance, and reports behavioral effects such as accepting, checking, overriding, or rejecting outputs.
   - **Minimum coverage:** At least **2 strong kept/trim sources**, including at least **1 primary empirical or foundational research source**.

2. **Uncertainty representation and confidence limits**
   - **Why it matters:** Words, icons, categories, and numbers can be misunderstood; model confidence may not map cleanly to correctness for a particular user decision.
   - **Good evidence:** Primary experiments, rigorous syntheses, or authoritative technical guidance comparing uncertainty formats, documenting comprehension and calibration effects, and explaining when numeric confidence is or is not valid.
   - **Minimum coverage:** At least **2 strong kept/trim sources**, with **1 primary empirical source** and **1 source addressing the limits or misuse of confidence scores**.

3. **Evidence and verification interaction methods**
   - **Why it matters:** The interface must help a person check a claim’s basis, not merely add a credibility cue.
   - **Good evidence:** Methods or evaluations covering provenance, citations, source excerpts, traceability, retrieval support, inspectability, and concrete verify/correct flows; evidence should reveal both successful patterns and known failure modes such as unsupported or misleading citations.
   - **Minimum coverage:** At least **2 strong kept/trim sources**, including **1 evaluated method or primary study**.

4. **Worked interface cases across core output types**
   - **Why it matters:** Designing summaries, answers, extracted data, and recommendations involves craft and contextual judgment that abstract principles alone cannot teach.
   - **Good evidence:** Substantive case studies or documented product patterns that connect a known risk to interface choices, show the interaction in context, and report a rationale, tradeoff, evaluation, or observed outcome. Screenshot galleries without reasoning do not count.
   - **Minimum coverage:** At least **3 substantive kept/trim examples** spanning at least **2 core output types** and more than one organization or author.

5. **Deferral, safeguards, and human-control authority**
   - **Why it matters:** Some evidence gaps block a responsible answer, and unchecked AI output must not trigger irreversible or high-stakes action.
   - **Good evidence:** Primary standards, official guidance, or well-supported safety research on abstention, meaningful human oversight, confirmation, reversibility, escalation, and risk-sensitive deployment. For medical, legal, financial, privacy, security, or regulated boundaries, use primary or official authorities rather than generic design commentary.
   - **Minimum coverage:** At least **2 strong kept/trim sources**, including at least **1 primary or official authority** and **1 source translating safeguards into product behavior**.

6. **Critique and validation evidence**
   - **Why it matters:** Plausible design conventions can increase overconfidence, hide uncertainty, burden users with impossible checking, or work differently across abilities and levels of expertise.
   - **Good evidence:** Critical research, negative or null findings, dissenting practitioner analysis grounded in cases, and practical evaluation methods that test behavioral reliance, error detection, accessibility, and comprehension rather than preference alone.
   - **Minimum coverage:** At least **2 strong kept/trim sources**, including **1 dissenting or failure-focused source** and **1 source with a reusable evaluation method**.

### Source exclusions

- Generic “responsible AI,” trust, or UX principle lists that do not connect an error or evidence condition to a user decision and interface behavior.
- Vendor claims, product marketing, or pattern galleries without substantive rationale, methods, or outcome evidence.
- Sources that treat explanation as proof of correctness, citations as automatic verification, or user trust as the success metric.
- Confidence-score guidance that does not distinguish model probability, calibration, evidence strength, and the user-facing meaning of the number.
- Advice centered on model training, benchmark accuracy, governance programs, or legal compliance unless it directly informs the runtime interface decision.
- Domain-specific medical, legal, or financial advice presented as transferable compliance guidance for general products.
- Duplicate secondary summaries when the primary research, standard, or official guidance is available and usable.

### Runtime autonomy, abstention, and escalation

The future agent should operate autonomously when the audience, decision, stakes, reversibility, likely failure modes, and available evidence are clear enough to choose a bounded interface treatment. It should state assumptions; distinguish observations, source-supported conclusions, design inferences, unknowns, and verification needs; and avoid claiming that any single disclosure or explanation “solves” trust.

Ask one targeted question only when a blocking gap—especially unknown audience or stakes—could reverse the recommendation or change whether the feature may answer at all. Missing visual polish, exact copy tone, or a nonessential implementation detail should limit confidence, not block progress.

Abstain from recommending an answer-first experience when supporting evidence is unavailable or cannot be tied to the output, when the system is outside its validated scope, or when a plausible error could directly cause an irreversible or high-stakes action. In those cases, recommend deferral, an authoritative source check, explicit human confirmation, or a reversible draft state. Never encourage sending, paying, deleting, or making medical, legal, or financial decisions from unchecked AI output.

Escalate to domain, safety, security, privacy, accessibility, or legal specialists when the feature handles sensitive data, vulnerable users, regulated decisions, consequential permissions, or organizational obligations. Require authoritative evidence for claims in those areas. Never infer compliance, safety, accessibility, or release readiness from screenshots, surface copy, or the presence of a human-review control alone.

## Knowledge map

This map is a research hypothesis and should be revised if source discovery exposes a missing frame or a misleading boundary.

### 1. The reliance decision

- Appropriate reliance versus trust, confidence, satisfaction, and perceived transparency
- Automation bias, complacency, algorithm aversion, and the effects of user expertise or time pressure
- Error consequence, reversibility, frequency, detectability, and the cost of checking
- Failure profiles of summaries, suggested answers, extracted data, and recommendations

### 2. Representing uncertainty without false precision

- Plain-language caveats, categorical states, visual cues, ranges, and numeric confidence
- Calibration and the relationship between a score, evidence strength, and actual correctness
- Placement, timing, progressive disclosure, and cognitive load
- Ambiguity, accessibility, localization, and comprehension across audiences

### 3. Showing evidence people can inspect

- Provenance, citations, excerpts, input-to-output traceability, and evidence freshness
- Support at claim, field, answer, and aggregate levels
- Source quality, disagreement, missing support, and misleading or fabricated citations
- Explanations as decision support rather than persuasion or proof

### 4. Verification, correction, and recovery

- Inspect, compare, edit, accept, reject, and report flows
- Verification effort proportional to stakes and uncertainty
- Contestability, feedback loops, ownership, and recovery after errors
- Reversible drafts, confirmations, audit trails, and preserving the user’s agency

### 5. Abstention, deferral, and escalation

- Insufficient, stale, contradictory, or out-of-scope evidence
- Safe fallback states and language for “cannot determine”
- Human handoff, authoritative-source checks, and specialist review
- Safeguards for irreversible actions, sensitive data, vulnerable users, and regulated contexts

### 6. Evaluation and failure-focused critique

- Behavioral calibration: acceptance, checking, correction, override, and error detection
- Realistic plausible-error scenarios across the four core output types
- Comprehension, accessibility, subgroup differences, and burden of verification
- Short-term usability versus longitudinal reliance, drift, and learned complacency
