# designing-for-ai-uncertainty

## Scope

This Operating Layer is for this job:

> When an AI feature in my product can be wrong, I want to show how far to trust it and let people check its work, so they rely on it the right amount.

Use this file to guide how AI sessions work with this local Mixtape. Use `mixtape/MIXTAPE.md` as the main reading artifact. Open deeper source files when the answer needs evidence.

## How To Use This Project

- Load `LINER.md` first, then `mixtape/MIXTAPE.md` for the source-grounded corpus.
- Use this file to choose working mode, source priority, boundaries, and maintenance behavior.
- Do not present the project as a persona, chatbot, or generic knowledge bot.
- Before giving a strong recommendation, name the corpus stance, rule, source section, or Project Skill that supports it.
- For changes to this Liner Project's canonical artifacts, use its maintenance workflow and require review. For changes in a consuming project, follow that project's own permissions and workflow.

## Working Loop

1. Orient: restate the user's job in the language of this mixtape.
2. Retrieve: load `mixtape/MIXTAPE.md`, then source files or the active Project Skill only when the answer needs deeper evidence.
3. Apply: turn source-grounded stances into a concrete answer, critique, plan, or question.
4. Check: test the answer against conflict, abstention, and source-use rules before finalizing.
5. Maintain: when the corpus is thin, name the missing source, note, or Project Skill boundary that should be updated.

## Corpus-Derived Operating Contract

- Core action: Convert an AI feature's actual error modes, evidence quality, user expertise, action stakes, and reversibility into a specific uncertainty treatment, inspectable evidence, verification and correction controls, and a deferral rule.
- Operating thesis: The design problem is not how to make people trust AI. It is how to help them rely on it in proportion to what it can actually do, in the situation they are in, with the consequences they face. Trust is an attitude; reliance is a behavior. A person may say they distrust a system and still follow its recommendation, or report high trust while checking every result. The interface therefore has to support a decision: accept this output, inspect it, correct it, seek another judgment, or stop.
- Capability pattern: calibrated-reliance interaction design.

### Required Output

- **Context and risk read:** the user, decision or action, plausible failure modes, reversibility, stakes, and assumptions. Separate supplied facts and observable interface facts from inferences and unknowns.
- **Uncertainty treatment:** the words, visual treatment, and placement to use; whether uncertainty should be qualitative, categorical, numeric, or omitted; and why that representation fits the evidence and user decision.
- **Evidence beside the output:** what provenance, citations, excerpts, source links, input-to-output trace, or other support to expose; what each item lets the person verify; and what must not be implied by its presence.
- **Verification and correction path:** the shortest meaningful way to inspect the basis, compare against authoritative material, edit or reject the result, report a problem, and recover from a mistake. Do not make “verify this” a warning without an actionable path.
- **Fallback, deferral, and escalation:** what the feature should do when evidence is absent, contradictory, stale, outside scope, or too weak for the stakes; when it should abstain or route to a person; and how that state should be explained.
- **Recommendation rationale:** cite the corpus sources supporting each major choice and state the tradeoff accepted—for example, speed versus scrutiny, simplicity versus precision, or helpfulness versus over-reliance.
- **Validation checks:** specify what to test with representative users and failure cases, including whether people interpret the uncertainty correctly, notice and use evidence, catch plausible errors, and know when not to proceed.

### Runtime Boundaries

- The future agent should operate autonomously when the audience, decision, stakes, reversibility, likely failure modes, and available evidence are clear enough to choose a bounded interface treatment. It should state assumptions; distinguish observations, source-supported conclusions, design inferences, unknowns, and verification needs; and avoid claiming that any single disclosure or explanation “solves” trust.
- Ask one targeted question only when a blocking gap—especially unknown audience or stakes—could reverse the recommendation or change whether the feature may answer at all. Missing visual polish, exact copy tone, or a nonessential implementation detail should limit confidence, not block progress.
- Abstain from recommending an answer-first experience when supporting evidence is unavailable or cannot be tied to the output, when the system is outside its validated scope, or when a plausible error could directly cause an irreversible or high-stakes action. In those cases, recommend deferral, an authoritative source check, explicit human confirmation, or a reversible draft state. Never encourage sending, paying, deleting, or making medical, legal, or financial decisions from unchecked AI output.
- Escalate to domain, safety, security, privacy, accessibility, or legal specialists when the feature handles sensitive data, vulnerable users, regulated decisions, consequential permissions, or organizational obligations. Require authoritative evidence for claims in those areas. Never infer compliance, safety, accessibility, or release readiness from screenshots, surface copy, or the presence of a human-review control alone.

### Quality Finding

- Pass. The corpus can help a future agent make a bounded interface recommendation and specify how to test it, rather than merely discussing trustworthy AI.

## Operating Stance

- Be opinionated only where the corpus is opinionated.
- Translate the corpus into decisions, not summaries.
- Start from the curator's synthesis before answering.
- Prefer source-grounded tradeoffs over generic advice.
- Keep the project's scope narrow; do not claim coverage beyond the corpus.
- When a request falls outside the corpus, say what is missing and ask for a source.

## Resource Map

- `LINER.md`: operating layer and behavior rules.
- `mixtape/MIXTAPE.md`: compiled corpus and source-grounded reading packet.
- `mixtape/sources/`: deeper evidence for claims that need source-level detail.
- Project Skill: `liner-designing-for-ai-uncertainty` at `SKILL.md`.
- `working/`: drafts and maintenance notes for this Liner Project; never use it as the workspace for a consuming product.

## Source Use Rules

- Corpus size: 30 saved source(s).
- Compiled availability: 30 usable compiled source file(s).
- Source kinds: example 6, prescription 8, principle 13, reference 3.
- Source sections: abstention-and-escalation 6, evaluation-and-critique 3, foundations 2, inspectable-evidence 7, uncertainty-signals 5, unsectioned 3, verification-and-recovery 4.
- Synthesis status: mixtape/synthesis.md present.
- Quality status: mixtape/working/04-quality-checks.md present.
- Load `mixtape/MIXTAPE.md` first, then open individual source files when the answer depends on detail.
- Respect source notes and source kinds; a canonical/reference source should outweigh a loose example.
- Cite source titles or sections when making a strong recommendation.

## Project Skill

Active Project Skill: `liner-designing-for-ai-uncertainty` at `SKILL.md`.

- This project has reusable guidance, so the Project Skill includes specific rules for using this corpus.
- Use it only when the user's request matches this project's job and corpus stance.
- Treat the skill as grounded in `mixtape/MIXTAPE.md`; do not let it override source hierarchy, conflict rules, or abstention rules.
- If the Project Skill needs to change, route the request through this Liner Project's maintenance workflow.

## Conflict Rules

- Prefer newer or more canonical sources when time-sensitive guidance conflicts.
- Prefer direct product or platform documentation over commentary when implementation details conflict.
- Preserve minority perspectives when the corpus intentionally includes them.
- Record unresolved contradictions in `working/audits/` instead of smoothing them away.

## Abstention Rules

- Do not answer as if the corpus covers laws, medical advice, finances, or safety-critical claims unless those sources are present.
- Do not invent source-backed confidence. Mark unsupported claims as outside scope.
- If a user asks for a decision the corpus cannot support, explain the missing evidence and propose what to add.

- Ask targeted questions only when missing evidence blocks a reliable answer; otherwise proceed with a bounded diagnosis and label observations, inferences, unknowns, and verification needs.
- For sensitive data, consent, security, privacy, irreversible actions, regulated contexts, or vulnerable users, do not make compliance, safety, or go/no-go claims without explicit criteria and adequate evidence.

## Readiness And Validation

- Project Complete means the corpus and Operating Layer artifacts are ready; it does not mean behavioral effectiveness has been validated.
- Validate important behavior with representative tasks and fixtures under `working/evals/`; keep evaluation results separate from corpus readiness.

## Maintenance Rules

- New books, recordings, local notes, and URLs should enter as sources before they change this file.
- After adding sources, refresh synthesis before changing operating rules.
- Keep generated changes review-first: draft, inspect, then update the project files deliberately.
