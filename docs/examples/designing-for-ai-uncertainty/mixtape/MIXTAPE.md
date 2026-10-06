# designing-for-ai-uncertainty

> Guidance for designing-for-ai-uncertainty.

**Curator:** Arturo  
**Compiled:** 2026-10-03T04:44:57.074533+00:00  
**Sources:** 30 (0 videos, 30 articles, 0 local files, 0 skills)  
**Mode:** quick  
**JTBD:** When an AI feature in my product can be wrong, I want to show how far to trust it and let people check its work, so they rely on it the right amount.

---

## How to use this mixtape

This is a curated context bundle compiled by Liner. The synthesis below is the
curator's distilled view of the domain and should anchor your framing.

Each source listed in the index lives in its own file under `sources/`. Load
the full content of a source on demand when the conversation requires specific
detail — the index entries (with curator notes) tell you which sources matter
for which questions.

Treat the curator notes as load-bearing: they signal source weight, intent, and
limitations.

---

## Synthesis

# Designing for AI Uncertainty — Synthesis

The design problem is not how to make people trust AI. It is how to help them rely on it in proportion to what it can actually do, in the situation they are in, with the consequences they face. Trust is an attitude; reliance is a behavior. A person may say they distrust a system and still follow its recommendation, or report high trust while checking every result. The interface therefore has to support a decision: accept this output, inspect it, correct it, seek another judgment, or stop.

Appropriate reliance is not a stable property of either the person or the model. It is a relationship among the system’s performance, the person’s own knowledge and confidence, the task, and the cost of being wrong. Time pressure can make a modestly useful aid worth following; a consequential or irreversible action can make the same performance unacceptable. Prior experience matters too: one conspicuous mistake can cause people to abandon automation that remains better than the alternative, while a polished explanation or a confident first impression can encourage them to follow later errors. The design target is therefore not maximal acceptance, maximal skepticism, or even maximal subjective trust. It is better decisions and recoverable outcomes across both correct and incorrect AI behavior.

I see four linked layers in this work: operating boundary, uncertainty signal, inspectable support, and recourse. Each layer answers a different question. The operating boundary asks whether the system should answer at all. The uncertainty signal asks what the person needs to know about the limits of this particular output. Inspectable support asks what they can examine to judge important claims. Recourse asks what they can do when the output is weak, wrong, or too risky to use. A product that handles only one layer is incomplete. A confidence badge without evidence offers a mood, not a check. Citations without correction paths expose problems without helping users recover. An edit button without a clear boundary can quietly transfer responsibility to someone who lacks the information needed to detect the error.

The operating boundary comes first because interface treatment cannot rescue an unsafe operating mode. Risk combines likelihood and consequence: low-probability error can still demand review when it could trigger an irreversible financial, legal, health, or data action. The authoritative risk-management substrate supports explicit refusal criteria, human oversight, independent evaluation proportional to risk, monitoring of overrides, and stopping or staging a release when residual risk exceeds tolerance ([NIST AI RMF Generative AI Profile](https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence)). In product terms, this means uncertainty may change the system from answer to defer, from execute to draft, from a single recommendation to alternatives, or from automation to human handoff. “I’m not sure” is useful only when it changes what happens next.

The uncertainty signal should describe a decision-relevant limit, not display technical precision for its own sake. A number is justified only when it is calibrated, users can interpret it in context, and the product has tested how it changes behavior. Controlled studies show the danger of treating confidence as harmless metadata: calibrated confidence can improve decisions, while miscalibrated confidence can produce both automation bias and excessive rejection. Even correctly calibrated model confidence does not automatically improve joint performance. The person’s confidence and uncertainty also matter; a low-confidence person may defer too readily, while an overconfident person may ignore good advice. The useful comparison is often not “How certain is the model?” but “Where do the model’s and the person’s judgments diverge, and what inspection would resolve the difference?”

Explanations are similarly conditional. They can make a system feel understandable, speed up a task, or improve local accuracy without reducing reliance on wrong advice. Some explanations even intensify automation bias. Treat explanation as an instrument with a specific job: reveal a relevant limitation, show why a result changes, help compare alternatives, or direct attention to evidence. Then test that job behaviorally. Agreement, satisfaction, and self-reported trust are weak substitutes for whether people notice bad advice and retain useful advice. A persuasive rationale is not a safety mechanism simply because it is intelligible.

Inspectable support is stronger when it connects claims to evidence at the right granularity. The relevant distinction is not cited versus uncited. It is whether the important claims are supported completely, whether each cited passage actually entails the claim, whether the source is credible for that claim, and whether the user can reach the support without losing their place. Citation quality therefore has both recall and precision: unsupported claims must not hide among well-cited ones, and decorative or irrelevant citations must not borrow legitimacy from good sources. The best pattern is claim-level verification followed by selective repair—identify what lacks support, retrieve or expose relevant evidence, and correct the unsupported portion while preserving what remains useful. Visible references are necessary in many knowledge tasks, but they do not guarantee truth; weak retrieval can furnish convincing support from an unreliable source.

Recourse completes the loop. People need to dismiss a suggestion, edit it, choose an alternative, revert a change, correct the system’s assumptions, or escalate to a responsible human. Consequential submissions should be reversible, checked for correction, or reviewed before commitment. Manual takeover also needs situation awareness: what went wrong, what remains safe to use, what the next action is, and how to perform it. Feedback is not the same as recourse. A thumbs-down may help a future model while doing nothing for the person harmed now. Good recovery restores agency in the present and is candid about whether feedback will affect the current result, a later experience, or only internal evaluation.

## Generative rules

1. **Design the reliance decision, not a trust impression.** Specify when a person should accept, inspect, override, defer, or stop, and evaluate those behaviors rather than asking only whether the system feels trustworthy.
2. **Make uncertainty change the interaction.** Pair every uncertainty cue with an action such as inspecting evidence, comparing alternatives, narrowing the task, requesting missing information, saving as a draft, or escalating.
3. **Expose support at claim level.** Let people trace consequential claims to relevant evidence, distinguish unsupported content from supported content, and judge source quality without treating citation presence as proof.
4. **Scale friction and oversight with consequence.** Keep low-risk exploration fluid, but require review, confirmation, reversibility, or human responsibility before uncertain output causes a costly or irreversible action.
5. **Test wrong-AI moments deliberately.** Include confidently wrong outputs, low-confidence correct outputs, incomplete evidence, misleading explanations, and first-impression effects; measure over-reliance, under-reliance, correction, and recovery.

## Stances this corpus takes

This corpus leans toward calibrated reliance over trust maximization, behavioral evidence over reassuring sentiment, and contextual boundaries over universal confidence treatments. It resists anthropomorphic reassurance, unexplained scores, citations as decoration, and explanations presented as automatic safeguards. It favors draft-before-commit for consequential actions, claim-linked evidence for factual outputs, alternatives when the system is unsure, and explicit human handoff when responsibility cannot safely remain with the model.

It also resists the idea that every error calls for more warning or more friction. Visible mistakes can provoke algorithm aversion, and cognitive forcing can reduce over-reliance while making an experience less acceptable. Friction should therefore be selective, proportionate, and tested. A dissenting result should temper any universal rule: confidence displays can help when calibrated; explanations can improve performance in some tasks; fast acceptance can be rational in low-stakes contexts; and professional users may need system evidence without surrendering their own epistemic authority. The corpus chooses safeguards, but not ritualized caution.

The most contested question is whether uncertainty should be shown directly. My answer is: show what helps the user choose a safer next action, not whatever the model happens to emit. A calibrated probability may be useful in repeated prediction tasks, while a missing-data warning, range, alternatives, or a statement of scope may be more honest for generative work. If the underlying quantity is not validated against the intended use, translating it into a friendly label does not make it trustworthy.

A second contested question is whether to explain first or require independent thought first. On-demand explanations preserve flow and user choice; delayed advice, an initial human judgment, or other cognitive forcing can reduce anchoring on wrong AI output. The right answer depends on consequence, expertise, task frequency, and the user’s ability to evaluate the result. Default to low-friction access to evidence for ordinary use, but introduce deliberate independence before high-impact decisions where early AI exposure could distort judgment. Evaluate both effectiveness and acceptability, including who benefits and who is burdened.

Several distinctions should remain sharp. Model uncertainty is not decision uncertainty: even a technically confident prediction may be unsafe when inputs are incomplete or consequences are high. Explanation is not evidence: a rationale can describe model behavior without establishing that a claim is true. Evidence support is not source truth: a passage may literally support a false or low-quality claim. Transparency is not recourse: knowing why something happened does not provide a way to correct it. And abstention is not failure: refusing, narrowing, or handing off can be the competent output when the system lacks a defensible basis to proceed.

Use this mixtape to design and critique AI-assisted answers, recommendations, summaries, predictions, and generated content where people need to judge how far to rely on an imperfect result. It is especially useful for confidence and uncertainty treatments, citations and evidence inspection, verification friction, corrections, confirmation, and escalation. Look elsewhere for model-calibration implementation, domain-specific safety thresholds, legal compliance determinations, clinical validation, or accessibility conformance as a whole; those require technical, regulatory, and domain evidence beyond this interaction-focused corpus.

---

## Sources

### foundations

#### Source 1: Trust in automation: integrating empirical evidence on factors that influence trust.

- **Type:** web
- **Kind:** principle
- **URL:** https://scholars.duke.edu/publication/1163926
- **Curator note:** **[principle]** Role: Keep as the main empirical synthesis for calibrated reliance. Value: Use the three trust layers and situational moderators to frame feature-specific reliance decisions. Limitations: It reviews automation studies from 2002–2013 and predates generative-AI interaction patterns.
- **Content file:** [./sources/01-trust-in-automation-integrating-empirical-evidence-on-factor.md](./sources/01-trust-in-automation-integrating-empirical-evidence-on-factor.md)

#### Source 2: Beyond Accuracy: The Role of Mental Models in Human-AI Team Performance

- **Type:** web
- **Kind:** example
- **URL:** https://ojs.aaai.org/index.php/HCOMP/article/view/5285
- **Author:** Gagan Bansal; Besmira Nushi; Ece Kamar; Walter S. Lasecki; Daniel S. Weld; Eric Horvitz
- **Published:** 2019/10/28
- **Curator note:** **[example]** Role: Keep as the mental-model and error-boundary method. Value: Use it to specify what users must learn before asking them to accept or override an AI output and what feedback can teach them. Limitations: The controlled binary-classifier game and MTurk sample do not directly establish patterns for summaries, answers, or recommendations.
- **Content file:** [./sources/02-beyond-accuracy-the-role-of-mental-models-in-human-ai-team-p.md](./sources/02-beyond-accuracy-the-role-of-mental-models-in-human-ai-team-p.md)

### uncertainty-signals

#### Source 3: Effect of Confidence and Explanation on Accuracy and Trust Calibration in AI-Assisted Decision Making

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2001.02114
- **Author:** Zhang, Yunfeng; Liao, Q. Vera; Bellamy, Rachel K. E.
- **Published:** 2020/01/07
- **Curator note:** **[principle]** Role: Foundational empirical test of confidence and local explanations. Value: Use the experiment results to separate calibrated reliance from actual decision improvement. Limitations: One controlled case study; its explanation design and task do not generalize to every AI feature.
- **Content file:** [./sources/03-effect-of-confidence-and-explanation-on-accuracy-and-trust-c.md](./sources/03-effect-of-confidence-and-explanation-on-accuracy-and-trust-c.md)

#### Source 4: Too Sure for Our Own Good: A User Study on AI Confidence and Human Reliance

- **Type:** web
- **Kind:** principle
- **URL:** https://ojs.aaai.org/index.php/AAAI/article/view/38798
- **Author:** Caterina Fregosi; Lucia Vicente; Andrea Campagner; Federico Cabitza
- **Published:** 2026/03/14
- **Curator note:** **[principle]** Role: Recent failure-focused evidence on confidence cues. Value: Use the calibrated-versus-miscalibrated comparison to set a hard requirement for validating any displayed confidence. Limitations: Logic-puzzle task and controlled study; it does not establish the best visual or textual format for other domains.
- **Content file:** [./sources/04-too-sure-for-our-own-good-a-user-study-on-ai-confidence-and.md](./sources/04-too-sure-for-our-own-good-a-user-study-on-ai-confidence-and.md)

#### Source 5: Using AI Uncertainty Quantification to Improve Human Decision-Making

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2309.10852
- **Author:** Marusich, Laura R.; Bakdash, Jonathan Z.; Zhou, Yan; Kantarcioglu, Murat
- **Published:** 2023/09/19
- **Curator note:** **[principle]** Role: Empirical source on calibrated uncertainty quantification and display forms. Value: Use the two experiments to compare point, distribution, needle, and dotplot treatments against decision accuracy. Limitations: Controlled behavioral tasks and model-level UQ; it does not provide general copy guidance or prove transfer to high-stakes product decisions.
- **Content file:** [./sources/05-using-ai-uncertainty-quantification-to-improve-human-decisio.md](./sources/05-using-ai-uncertainty-quantification-to-improve-human-decisio.md)

#### Source 6: "Are You Really Sure?" Understanding the Effects of Human Self-Confidence Calibration in AI-Assisted Decision Making

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2403.09552
- **Author:** Ma, Shuai; Wang, Xinru; Lei, Ying; Shi, Chuhan; Yin, Ming; Ma, Xiaojuan
- **Published:** 2024/03/14
- **Curator note:** **[principle]** Role: Human-side calibrated-reliance evidence. Value: Use the study’s mechanisms and mixed results when deciding whether a confidence display should prompt reflection or comparison. Limitations: Controlled prediction task and no answer to which model-uncertainty visual is best.
- **Content file:** [./sources/06-are-you-really-sure-understanding-the-effects-of-human-self.md](./sources/06-are-you-really-sure-understanding-the-effects-of-human-self.md)

#### Source 7: Explaining the Uncertainty in AI-Assisted Decision Making

- **Type:** web
- **Kind:** principle
- **URL:** https://ojs.aaai.org/index.php/AAAI/article/view/26920
- **Author:** Thao Le
- **Published:** 2023
- **Curator note:** **[principle]** Role: Small conceptual counterpoint on explaining why confidence changes. Value: Route to the counterfactual distinction and the aleatoric-versus-epistemic vocabulary when choosing what uncertainty a UI should expose. Limitations: Work in progress, only two pages, and lacks validated user effects.
- **Content file:** [./sources/07-explaining-the-uncertainty-in-ai-assisted-decision-making.md](./sources/07-explaining-the-uncertainty-in-ai-assisted-decision-making.md)

### inspectable-evidence

#### Source 8: Enabling Large Language Models to Generate Text with Citations

- **Type:** web
- **Kind:** principle
- **URL:** https://aclanthology.org/2023.emnlp-main.398/
- **Author:** Tianyu Gao; Howard Yen; Jiatong Yu; Danqi Chen
- **Published:** 2023/12
- **Curator note:** **[principle]** Role: Foundational method for evaluating inspectable generated answers. Value: Separates correctness from citation support and gives reusable recall/precision tests. Limitations: Benchmark findings concern retrieved QA, not a finished product interaction.
- **Content file:** [./sources/08-enabling-large-language-models-to-generate-text-with-citatio.md](./sources/08-enabling-large-language-models-to-generate-text-with-citatio.md)

#### Source 9: RARR: Researching and Revising What Language Models Say, Using Language Models

- **Type:** web
- **Kind:** example
- **URL:** https://arxiv.org/abs/2210.08726
- **Author:** Gao, Luyu; Dai, Zhuyun; Pasupat, Panupong; Chen, Anthony; Chaganty, Arun Tejasvi; Fan, Yicheng; Zhao, Vincent Y.; Lao, Ni; Lee, Hongrae; Juan, Da-Cheng; Guu, Kelvin
- **Published:** 2022/10/17
- **Curator note:** **[example]** Role: Core method for claim-level verification and repair. Value: Shows how to expose evidence and correct only unsupported portions while preserving useful structure. Limitations: Its automated web-search and editing pipeline can inherit retrieval or source-quality errors and is not a UI prescription.
- **Content file:** [./sources/09-rarr-researching-and-revising-what-language-models-say-using.md](./sources/09-rarr-researching-and-revising-what-language-models-say-using.md)

#### Source 10: Teaching language models to support answers with verified quotes

- **Type:** web
- **Kind:** example
- **URL:** https://arxiv.org/abs/2203.11147
- **Author:** Menick, Jacob; Trebacz, Maja; Mikulik, Vladimir; Aslanides, John; Song, Francis; Chadwick, Martin; Glaese, Mia; Young, Susannah; Campbell-Gillingham, Lucy; Irving, Geoffrey; McAleese, Nat
- **Published:** 2022/03/21
- **Curator note:** **[example]** Role: Direct evidence-grounding and abstention reference. Value: Gives the agent a concrete claim-plus-quote structure and a test for whether evidence is sufficient, relevant, and truthful. Limitations: Model-centric evaluation and paid-rater judgments do not establish that end users will inspect quotes successfully.
- **Content file:** [./sources/10-teaching-language-models-to-support-answers-with-verified-qu.md](./sources/10-teaching-language-models-to-support-answers-with-verified-qu.md)

#### Source 11: People + AI Guidebook

- **Type:** web
- **Kind:** prescription
- **URL:** https://pair.withgoogle.com/guidebook-v2/patterns
- **Curator note:** **[prescription]** Role: Practitioner pattern reference for turning evidence and uncertainty into controls. Value: Supplies concrete “aim/avoid” guidance for sources, confidence displays, feedback, supervision, and handoff. Limitations: Google guidebook advice is not a controlled evaluation and its examples should not be treated as outcome proof.
- **Content file:** [./sources/11-people-ai-guidebook.md](./sources/11-people-ai-guidebook.md)

#### Source 12: WebGPT: Browser-assisted question-answering with human feedback

- **Type:** web
- **Kind:** example
- **URL:** https://arxiv.org/abs/2112.09332
- **Author:** Nakano, Reiichiro; Hilton, Jacob; Balaji, Suchir; Wu, Jeff; Ouyang, Long; Kim, Christina; Hesse, Christopher; Jain, Shantanu; Kosaraju, Vineet; Saunders, William; Jiang, Xu; Cobbe, Karl; Eloundou, Tyna; Krueger, Gretchen; Button, Kevin; Knight, Matthew; Chess, Benjamin; Schulman, John
- **Published:** 2021/12/17
- **Curator note:** **[example]** Role: Early system example linking browsing, references, and human evaluation. Value: Use the environment design and TruthfulQA failure analysis to explain why visible references do not guarantee reliable evidence. Limitations: Dated model-training focus and limited product-interface guidance; skip most optimization details.
- **Content file:** [./sources/12-webgpt-browser-assisted-question-answering-with-human-feedba.md](./sources/12-webgpt-browser-assisted-question-answering-with-human-feedba.md)

#### Source 13: Training Language Models to Generate Text with Citations via Fine-grained Rewards

- **Type:** web
- **Kind:** example
- **URL:** https://aclanthology.org/2024.acl-long.161/
- **Author:** Chengyu Huang; Zeqiu Wu; Yushi Hu; Wenya Wang
- **Published:** 2024/8
- **Curator note:** **[example]** Role: Failure-focused complement to the citation benchmarks. Value: Demonstrates that answer correctness, citation recall, and citation precision trade off, with concrete ID-mixup and redundancy errors. Limitations: Primarily a fine-tuning paper; it offers little guidance on how people should inspect citations in a product.
- **Content file:** [./sources/13-training-language-models-to-generate-text-with-citations-via.md](./sources/13-training-language-models-to-generate-text-with-citations-via.md)

#### Source 14: People + AI Guidebook

- **Type:** web
- **Kind:** prescription
- **URL:** https://pair.withgoogle.com/guidebook-v2/chapter/data-collection/
- **Curator note:** **[prescription]** Role: Bridge from evidence presentation to evidence quality and evaluation planning. Value: Helps the agent ask what data, labeling, representativeness, and blindspots sit behind a displayed result. Limitations: Much of the chapter concerns model development and data operations rather than user-facing verification.
- **Content file:** [./sources/14-people-ai-guidebook.md](./sources/14-people-ai-guidebook.md)

### verification-and-recovery

#### Source 15: People + AI Guidebook

- **Type:** web
- **Kind:** prescription
- **URL:** https://pair.withgoogle.com/guidebook-v2/chapter/errors-failing/
- **Curator note:** **[prescription]** Role: Load-bearing method for failure and recovery design. Value: Route here for error taxonomy, risk-sensitive responses, feedback opportunities, and handoff to manual control. Limitations: Google practitioner guidance; examples are illustrative rather than independently evaluated product results.
- **Content file:** [./sources/15-people-ai-guidebook.md](./sources/15-people-ai-guidebook.md)

#### Source 16: Question-Driven Design Process for Explainable AI User Experiences

- **Type:** web
- **Kind:** prescription
- **URL:** https://arxiv.org/abs/2104.03483
- **Author:** Liao, Q. Vera; Pribić, Milena; Han, Jaesik; Miller, Sarah; Sow, Daby
- **Published:** 2021/04/08
- **Curator note:** **[prescription]** Role: Core method for turning “how should I check this?” into design and engineering requirements. Value: Use the question elicitation, prioritization, mapping, and iterative evaluation steps. Limitations: The worked case is healthcare prediction, and the final system was partly proprietary.
- **Content file:** [./sources/16-question-driven-design-process-for-explainable-ai-user-exper.md](./sources/16-question-driven-design-process-for-explainable-ai-user-exper.md)

#### Source 17: People + AI Guidebook

- **Type:** web
- **Kind:** prescription
- **URL:** https://pair.withgoogle.com/guidebook-v2/chapter/feedback-controls/
- **Curator note:** **[prescription]** Role: Core interaction guidance for correction and control. Value: Use the sections on manual fallback, editability, reset, and feedback impact when designing recovery. Limitations: Mostly practitioner guidance, with limited empirical validation of the patterns.
- **Content file:** [./sources/17-people-ai-guidebook.md](./sources/17-people-ai-guidebook.md)

#### Source 18: People + AI Guidebook

- **Type:** web
- **Kind:** example
- **URL:** https://pair.withgoogle.com/guidebook-v2/case-studies
- **Curator note:** **[example]** Role: Selective worked-example directory. Value: Open the Google Photos case for concrete control, sensitivity, and model-versus-rule tradeoffs, then use the pattern labels to locate other cases. Limitations: The candidate page is a gallery index; do not treat every listed case as read evidence.
- **Content file:** [./sources/18-people-ai-guidebook.md](./sources/18-people-ai-guidebook.md)

### abstention-and-escalation

#### Source 19: Artificial Intelligence Risk Management Framework: Generative Artificial Intelligence Profile

- **Type:** web
- **Kind:** reference
- **URL:** https://www.nist.gov/publications/artificial-intelligence-risk-management-framework-generative-artificial-intelligence
- **Author:** Chloe Autio; Reva Schwartz; Jesse Dunietz; Shomik Jain; Martin Stanley; Elham Tabassi; Patrick Hall; Kamie Roberts; Chloe Autio, Reva Schwartz, Jesse Dunietz, Shomik Jain, Martin Stanley, Elham Tabassi, Patrick Hall, Kamie Roberts
- **Published:** 2024-07-26T08:00-04:00
- **Updated:** 2026-04-08T23:22-04:00
- **Curator note:** **[reference]** Role: Use as the generative-AI-specific authority for answer, defer, refuse, and escalate boundaries. Value: Route to the sections on human-AI configuration, refusal criteria, recourse, independent evaluation, override monitoring, risk controls, and structured field testing when designing a fallback state or human handoff. Limitations: The profile is risk-management guidance rather than a tested interaction pattern; it does not determine the right threshold for a particular product, user, or regulated domain.
- **Content file:** [./sources/19-artificial-intelligence-risk-management-framework-generative.md](./sources/19-artificial-intelligence-risk-management-framework-generative.md)

#### Source 20: AI Risk Management Framework

- **Type:** web
- **Kind:** reference
- **URL:** https://www.nist.gov/itl/ai-risk-management-framework
- **Published:** 2021-07-12T14:09-04:00
- **Updated:** 2026-08-13T11:30-04:00
- **Curator note:** **[reference]** Role: Keep as the primary authority for deciding when an AI feature's risk and uncertainty should change its operating mode. Value: Use the risk-tolerance, residual-risk, lifecycle, and stop-until-managed passages to justify deferral, human confirmation, or escalation instead of treating trust as a visual treatment. Limitations: It is voluntary, high-level, and organizational; it does not specify user-facing copy, interaction layouts, or a complete runtime abstention algorithm.
- **Content file:** [./sources/20-ai-risk-management-framework.md](./sources/20-ai-risk-management-framework.md)

#### Source 21: Understanding SC 3.3.4 Error Prevention (Legal, Financial, Data) (Level AA)

- **Type:** web
- **Kind:** prescription
- **URL:** https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-legal-financial-data.html
- **Curator note:** **[prescription]** Role: Keep as the concrete safeguard authority for consequential actions that an AI feature may prepare or trigger. Value: Use its reversible, checked, or confirmed pattern to turn uncertainty into a review, correction, or draft-before-commit state. Limitations: It is an accessibility success criterion for web pages and specified transaction types, not an AI-specific abstention rule; apply the interaction pattern without claiming WCAG conformance or complete risk coverage from it alone.
- **Content file:** [./sources/21-understanding-sc-3-3-4-error-prevention-legal-financial-data.md](./sources/21-understanding-sc-3-3-4-error-prevention-legal-financial-data.md)

#### Source 22: Guidelines for human-AI interaction design - Microsoft Research

- **Type:** web
- **Kind:** prescription
- **URL:** https://www.microsoft.com/en-us/research/?p=564561
- **Author:** Emily Maryatt
- **Published:** 2019-02-01T16:59:48+00:00
- **Updated:** 2019-02-01T17:51:45+00:00
- **Curator note:** **[prescription]** Role: Keep as the practical interaction layer for implementing uncertainty, graceful degradation, correction, and user control. Value: Route to the "when wrong" guidelines and the AutoReplace example when turning an abstention or deferral rule into an actionable product state with alternatives, dismissal, correction, and explanation. Limitations: The page is a concise guideline overview with only a small number of examples, was developed for graphical interfaces, and does not establish high-stakes safety thresholds or empirical evidence for every recommendation.
- **Content file:** [./sources/22-guidelines-for-human-ai-interaction-design-microsoft-researc.md](./sources/22-guidelines-for-human-ai-interaction-design-microsoft-researc.md)

#### Source 23: Web Content Accessibility Guidelines (WCAG) 2.2

- **Type:** web
- **Kind:** reference
- **URL:** https://www.w3.org/TR/WCAG22/
- **Curator note:** **[reference]** Role: Use selectively as the normative bridge after deciding that a feature needs confirmation, recovery, or a visible status change. Value: Check the relevant success criteria for accessible error prevention and announcement of processing or deferral states, especially 3.3.4 and 4.1.3. Limitations: WCAG is not a trust-calibration framework and the full standard is too broad to serve as the corpus's main guidance; do not substitute conformance language for product-specific risk analysis.
- **Content file:** [./sources/23-web-content-accessibility-guidelines-wcag-2-2.md](./sources/23-web-content-accessibility-guidelines-wcag-2-2.md)

#### Source 24: Explaining decisions made with AI

- **Type:** web
- **Kind:** prescription
- **URL:** https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/artificial-intelligence/explaining-decisions-made-with-artificial-intelligence/
- **Curator note:** **[prescription]** Role: Keep as a selective bridge when a deferral or handoff needs a meaningful explanation rather than a generic warning. Value: Use its impact- and use-case-sensitive selection of explanation types and its translation from system rationale to understandable reasons to support contestability and verification. Limitations: The supplied URL is a navigation page, so this evaluation relies on a linked practical section; the guidance is under review, compliance-oriented, and does not provide a tested UI pattern or a decision rule for when the system must abstain.
- **Content file:** [./sources/24-explaining-decisions-made-with-ai.md](./sources/24-explaining-decisions-made-with-ai.md)

### evaluation-and-critique

#### Source 25: To Trust or to Think: Cognitive Forcing Functions Can Reduce Overreliance on AI in AI-assisted Decision-making

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2102.09692
- **Author:** Buçinca, Zana; Malaya, Maja Barbara; Gajos, Krzysztof Z.
- **Published:** 2021/02/19
- **Curator note:** **[principle]** Role: Use as the failure-focused empirical anchor for deciding when explanations need deliberate verification friction. Value: It compares concrete interaction patterns against behavioral error detection and reports acceptability and subgroup tradeoffs. Limitations: The food-substitution task is non-critical and the authors caution against generalizing without further domain studies.
- **Content file:** [./sources/25-to-trust-or-to-think-cognitive-forcing-functions-can-reduce.md](./sources/25-to-trust-or-to-think-cognitive-forcing-functions-can-reduce.md)

#### Source 26: Toward General Design Principles for Generative AI Applications

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2301.05578
- **Author:** Weisz, Justin D.; Muller, Michael; He, Jessica; Houde, Stephanie
- **Published:** 2023/01/13
- **Curator note:** **[principle]** Role: Use selectively as a principles-and-evaluation scaffold for generative features. Value: The sections on imperfection, human review, mental models, and harms connect failure conditions to interface choices. Limitations: Workshop-era synthesis with cited examples, not evidence that these principles improve reliance in a measured product context.
- **Content file:** [./sources/26-toward-general-design-principles-for-generative-ai-applicati.md](./sources/26-toward-general-design-principles-for-generative-ai-applicati.md)

#### Source 27: Designerly Understanding: Information Needs for Model Transparency to Support Design Ideation for AI-Powered User Experience

- **Type:** web
- **Kind:** principle
- **URL:** https://arxiv.org/abs/2302.10395
- **Author:** Liao, Q. Vera; Subramonyam, Hariharan; Wang, Jennifer; Vaughan, Jennifer Wortman
- **Published:** 2023/02/21
- **Curator note:** **[principle]** Role: Use as a bridge from model transparency to product-team design decisions. Value: It names a qualitative method and a concrete gap between technical reporting and designer needs. Limitations: Only the abstract was recovered, and the study concerns design ideation rather than whether end users check or appropriately rely on AI output.
- **Content file:** [./sources/27-designerly-understanding-information-needs-for-model-transpa.md](./sources/27-designerly-understanding-information-needs-for-model-transpa.md)

### Ungrouped

#### Source 28: People + AI Guidebook

- **Type:** web
- **Kind:** principle
- **URL:** https://pair.withgoogle.com/chapter/explainability-trust/
- **Content file:** [./sources/28-people-ai-guidebook.md](./sources/28-people-ai-guidebook.md)

#### Source 29: Guidelines for Human-AI Interaction - Microsoft HAX Toolkit

- **Type:** web
- **Kind:** principle
- **URL:** https://www.microsoft.com/en-us/haxtoolkit/ai-guidelines/
- **Updated:** 2023-10-17T20:35:01+00:00
- **Content file:** [./sources/29-guidelines-for-human-ai-interaction-microsoft-hax-toolkit.md](./sources/29-guidelines-for-human-ai-interaction-microsoft-hax-toolkit.md)

#### Source 30: Machine learning | Apple Developer Documentation

- **Type:** web
- **Kind:** principle
- **URL:** https://developer.apple.com/design/human-interface-guidelines/machine-learning
- **Content file:** [./sources/30-machine-learning-apple-developer-documentation.md](./sources/30-machine-learning-apple-developer-documentation.md)
