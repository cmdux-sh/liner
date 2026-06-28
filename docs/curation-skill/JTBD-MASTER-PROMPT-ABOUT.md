# About the JTBD Master Prompt

Companion document to [`JTBD-MASTER-PROMPT.md`](./JTBD-MASTER-PROMPT.md). Explains what the prompt is, why it exists, how it works, where it came from, and where it can still go wrong.

---

## Why this exists

Liner's curation methodology has eight phases. Phase 1 — Framing — is the single highest-leverage step in the lifecycle. The job-to-be-done statement defined here drives every downstream decision: which sources land in the candidate longlist, how the knowledge map is sectioned, what gets cut at evaluation, how the synthesis is framed. A bad JTBD propagates through everything else and produces a mixtape that looks complete but doesn't measurably improve AI output. A good JTBD makes every downstream phase easier.

The methodology already knows what a tight JTBD looks like. `CURATION.md` says it must be a "single specific sentence." `SKILL.md` says the AI must "push back" if the user offers a topic. `MIXTAPE-FORMAT.md` documents the `jtbd:` field. But that guidance is spread across multiple docs and lives in the curator's head, not in a deployable prompt. The result, in practice: users land at "topic" level ("mobile design," "SEO content," "contract law") instead of "job" level. Once they do, the rest of curation drifts.

The JTBD Master Prompt fixes that gap. It's a single, drop-in prompt that takes a user from wherever they are — a vague topic, a folder of content, a half-formed idea — to one tight Job Story that passes a rubric. That Job Story is the input to the rest of Liner's pipeline. Everything else (knowledge map, candidate discovery, evaluation, synthesis) is already handled.

## What it is

A self-contained master prompt stored at `docs/curation-skill/JTBD-MASTER-PROMPT.md`. The body sits between `---BEGIN---` and `---END---` markers and is meant to be copy-pasted into an AI conversation, embedded inside the `curating-mixtapes` skill, or spawned as a TUI subprocess.

The prompt's sole deliverable is a single sentence in the Intercom Job Story form: *"When [circumstance], I want [motivation], so I can [outcome]."* No rationale, no knowledge map, no commentary — those belong to later phases. One sentence on one line.

The prompt is interactive: it offers to read project material, conducts a small interview (capped at five questions, three typical), runs the draft through an internal rubric, and emits.

## How it works

Five moves, in order:

**Step 0 — Offer to read project material.** The agent's first message is a single short question: *"Do you have any documents or project material I should look at first?"* The user can drop files into the chat, point at a folder (if the agent has filesystem access), or say no and move straight into conversation. This question fires unconditionally and does not count against the question budget. The reason for asking rather than detecting: users frequently have load-bearing material they wouldn't think to share unprompted, and silent inference of context produces wrong JTBDs that propagate through every downstream phase.

**Step 1 — Read what the user shared, if any.** If the user shared material, the agent reads enough to confidently characterize the project — start with project-description files (`README`, `CLAUDE.md`, `synthesis.md`, `tape.yaml`), then any flagged content, then any additional material needed to name domain, audience, stage, and explicit goal. Token cost is not optimized; user-time is. If the agent has no file tools and the user pointed at a folder, it says so plainly and asks for files-in-chat or a description instead. It never invents file contents.

**Step 2 — Confirm what was inferred.** After reading, the agent's next message is a confirmation: *"I see your project is [X] for [Y], focused on [Z]. Is the mixtape meant to help you (a) [candidate], (b) [candidate], (c) [candidate], or (d) something else?"* One turn, one confirmation. Option (d) is required. The agent never bakes inferred context into the JTBD silently.

**Step 3 — Run a budgeted interview, if needed.** If the user shared no material, or if reading wasn't enough, the agent opens with a single batched message of two or three basic questions (domain, role, concrete activity), then descends the scope ladder one question at a time. Each follow-up walks the user from aspiration → big → little → micro. The agent stops asking when the next question wouldn't change the draft. Hard cap: five total questions, three typical.

**Step 4 — Draft, critique, refine, emit.** The agent drafts the Job Story silently. Before emitting, it runs the draft through a six-test rubric: structural (all three slots present and non-empty), solution-free (no vendor / framework / tool named), single-scope (one job per slot), specific (not generic or aspirational), externally readable (a stranger could understand it without context), and slot hygiene (audience clauses live in the circumstance, not the motivation). If a test fails, the agent fixes the draft and re-checks. The rubric is never shown to the user. Output is one line — the Job Story, no preamble, no commentary, no quote marks, no code fence. If the user pushes back, the agent doesn't defend; it takes the correction, re-runs the rubric, and emits a revised statement in the same format.

## Where it came from

The prompt is the engineering output of a mixtape compiled specifically for this question. That local source mixtape is archived at:

```
/Users/arturo/Documents/Projects/0-archive/liner-archive/2026-06-25-root-cleanup/mixtapes/jtbd-master-prompt/
├── MIXTAPE.md              # compiled master file
├── tape.yaml               # the recipe
├── synthesis.md            # the curator's framing
├── sources/                # 32 extracted sources
└── working/                # the curation trace
```

**Curator:** cmdux. **Compiled:** 2026-05-27. **Mode:** quick.

Thirty-two sources span two literatures that don't usually meet: JTBD theory (Ulwick, Christensen, Klement, Kalbach, Moesta, the Intercom Job Story canon and its critics) and prompt engineering (Anthropic's prompting best practices, the meta-prompting literature, elicitation patterns for agentic AI, the evaluator-optimizer pattern, the persona-injection ablation studies). The synthesis bridges them: how do you design a system prompt that talks to a human and produces a tight JTBD sentence on demand?

Six generative rules from that synthesis show up directly in the prompt:

1. **Job Story format with three required slots.** The synthesis recommends the Intercom "When [circumstance], I want [motivation], so I can [outcome]" form over alternatives because the required slots fail structurally when missing — a sentence-shaped non-job ("Help me with healthcare marketing") can't survive the Job Story format's structural check, where it could survive a single-slot form.
2. **Solution-free, single-scope, specific, externally readable** as the spine of the tightness rubric — those tests come from Ulwick's outcome-statement grammar via the SIVO clarity rubric, adapted to the Job Story slots.
3. **The aspiration → big → little → micro scope ladder** as the structure of the descent interview — from User Interviews' UX field guide on hyper-specificity.
4. **Evaluator-optimizer loop (draft → critique → refine)** as the internal generation pattern — from Anthropic's "Building Effective Agents" pattern taxonomy and the LLM self-critique prompting literature.
5. **One question at a time, hard budget, stop when next question wouldn't change the draft** — from the active-questioning and elicitation literature on agentic AI.
6. **No persona in the prompt** — from the Zheng et al. 2023 ablation showing persona injection doesn't improve task accuracy and optimal-persona prediction is no better than random.
7. **Failure-mode-first engineering register** — from the Claude 4 system-prompt teardown and the prompt-engineering anti-patterns catalog. Production system prompts are lists of mistakes the model used to make; the JTBD prompt is shaped the same way.

A few choices in the prompt deliberately diverge from the mixtape's center of gravity, and they're worth naming:

- **No passing examples in the final prompt.** The synthesis recommends concrete examples as a teaching tool. We dropped them because passing examples carry an anchoring risk — the model copies surface features from the example domain. Failing examples teach a rule that generalizes; passing examples teach a style that doesn't. The rubric is doing the work passing examples normally would. The single passing example that does appear (the clinical Job Story for the diabetes case in the descent section) illustrates *the casual-language-to-Job-Story translation*, not the target shape itself.
- **Quality over token economy in the read step.** Early drafts capped reads at "two or three sampled files." That cap is gone. If the user shared material, the agent reads enough to confidently characterize the project. Stopping condition is *confidence*, not file count.
- **Single context-sharing question, not a multi-path router.** An earlier draft offered the user three explicit routing options (folder / files / skip) in a ~120-word message. That turned out to be friction solving for the wrong problem. The deployment scenario is narrower than the router assumed: this prompt is only ever used to create a fresh JTBD from scratch. Users don't paste pre-drafted JTBDs (the whole point is they don't have one), and the choice between "files" and "folder" is mechanical, not strategic. The single short question — *"Do you have any documents or project material I should look at first?"* — surfaces the affordance without forcing the user through a decision tree.

### Historical note: the house form

Earlier versions of Liner — and the first published version of this prompt — used a different JTBD form: *"Help me [verb] [object] [scope]."* That form matched the surface shape of every Liner artifact at the time (the methodology docs, the example mixtapes, the schema), so the prompt was designed to produce what Liner already consumed.

The trade-off became clear once the prompt was used in practice. The house form has no required slots: a sentence like "Help me with healthcare marketing" satisfies the form template but is not a JTBD. That meant the qualitative rubric had to do all the work of catching topic-shaped non-jobs — a soft check rather than a hard one. The Job Story format has the same forgivingness against mush in its individual slots but fails structurally when a slot is empty, which raises the floor before the rubric runs.

The format switch realigned the prompt with the original synthesis recommendation. The cost is a one-time migration of existing artifacts; the benefit is a structurally tighter floor on every future JTBD.

## How to use it

Three deployment paths, all using the same prompt:

**Pure-chat (Claude.ai, ChatGPT, Cowork web).** User pastes the prompt block into a fresh chat. When the context-sharing question fires, the user can drop files into the chat or say they have no material. Folder paths aren't viable in this environment; the prompt handles that case gracefully.

**Agent with filesystem access (Claude Code, Cowork desktop, Codex, the Liner TUI as a subprocess).** Both file attachments and folder paths are live. The user picks; the agent reads accordingly.

**Embedded inside the `curating-mixtapes` skill.** Drop the prompt block into `docs/curation-skill/SKILL.md` as the body of Phase 1 — Framing, replacing the current "Ask the user for the JTBD" paragraph. The skill's outer context provides Liner framing; this prompt handles JTBD elicitation. Filesystem access is the default in the skill's runtime.

**As a TUI subprocess.** A future `liner jtbd <folder>` command could spawn this prompt with the folder path in context. The context-sharing question still fires — the user might want to add files or skip the folder — but the path is available if they want it.

## Design choices worth understanding

A few decisions in the prompt aren't obvious and have been pushed back on during design:

**Why fire the context-sharing question unconditionally?** Capability detection is brittle: file tools may be advertised but broken, sandboxed, or scoped to a different path than the user expects. Silent inference of context is worse than no context. Asking once is one extra turn; getting it wrong is a wrong JTBD that propagates through every downstream phase. Asking respects user agency and removes a class of silent-failure bugs. The question is short — three sentences — to keep the friction low.

**Why a hard question budget?** The most reliable failure mode for interview prompts is death by clarification — the model keeps asking because "one more question" feels safer than committing. A budget forces a draft. Five questions max, three typical. If the model finds itself drafting a fourth follow-up, the prompt instructs it to draft the JTBD instead.

**Why no persona block?** Empirically (Zheng et al., 2023, 162 personas × 2,410 questions) persona injection doesn't improve task accuracy and optimal-persona prediction is no better than random. "You are an expert JTBD practitioner with twenty years of experience" buys hedging language, not skill. The prompt names the task instead.

**Why failure modes before instructions?** Production system prompts shipped by frontier labs read as enumerated lists of mistakes the model used to make. That's the right register: the prompt prevents the five specific failures (solution baked in, conflated jobs, topic-as-JTBD, aspirational mush, death-by-questions) more than it teaches JTBD theory.

**Why no passing examples (mostly)?** Asymmetric example design — failing examples teach a generalizable rule, passing examples anchor on style and domain. The one passing example in the prompt (the clinical Job Story for the diabetes case) appears only to demonstrate the casual-language-to-Job-Story translation, not the shape of the target output. If users in adjacent fields start pattern-matching on it, swap or strip it.

**Why a slot hygiene check?** The first five rubric checks (structural, solution-free, single-scope, specific, externally readable) caught most failures in early use but consistently missed one: audience-scoping camped in the motivation slot. A draft like "I want to design a single-page site that explains the tool to people who already use AI" is technically one motivation, but the audience clause belongs in the circumstance. When audience drifts into the motivation, the circumstance reads thinner than the JTBD actually is, and the synthesis underneath misses signal it should have caught. Slot hygiene is the rule that promotes this from per-curator judgment to a generalizable check. It does not police mush, weak audience scoping, or product-positioning that doesn't earn its slot — those remain curator-level judgments and the qualitative rubric stays load-bearing.

## What this prompt does NOT do

- It does not produce a knowledge map. Phase 1 of curation wants both JTBD and knowledge map; this prompt produces only the JTBD. The knowledge map is the next move in the skill.
- It does not produce sources, citations, or curator notes. Those are Phase 2 and Phase 4.
- It does not produce a synthesis. Phase 6.
- It does not validate the JTBD against a real mixtape. The rubric is necessary but not sufficient. The empirical test (Phase 8) is the only real validation.
- It does not defend against adversarial users. Cooperative-user assumption is baked in. If the prompt ever ships in a context with hostile input (public web form, untrusted API), add a refusal clause.
- It does not accept a pre-drafted JTBD as input. The deployment scenario is fresh JTBD creation from scratch; users with an existing draft don't need this prompt.

## Known limits and watch items

- **Anchoring risk on the process examples.** The descent section contains examples from clinical, engineering, and creative domains. These illustrate *the scope-ladder shape*, not target output, so the anchoring risk is weaker than it would be for passing JTBD examples — but it's nonzero. If users in those fields consistently produce ladder-shaped artifacts that don't fit their actual work, swap or strip them.
- **The structural and slot-hygiene rubric checks are necessary, not sufficient.** Slot 1 catches *omitted* slots; slot 6 catches misplaced audience clauses. Neither catches mush, weak audience scoping, or product-positioning that doesn't earn its slot. A Job Story like "When I'm doing marketing, I want to do it well, so I can succeed" passes both structural checks and fails on the qualitative ones. The qualitative rubric stays load-bearing.
- **Soft dependency on user calibration.** The prompt assumes the user can identify their most relevant material. Almost always true for someone curating a mixtape; less true for a first-time user. If you observe weak JTBDs from less-experienced users, the fix is a checklist of *kinds* of content that tend to be load-bearing (project briefs, requirements docs, customer research, prior synthesis attempts), not more rules in the prompt.
- **Big-repo reads.** If a user points the agent at a very large folder, the "read enough to confidently characterize" rule may still under-sample the part of the repo that defines domain. The fix isn't a token cap; it's better signposting in the context-sharing question if you observe drift.

## Related files

- The prompt itself: [`JTBD-MASTER-PROMPT.md`](./JTBD-MASTER-PROMPT.md)
- The current format spec: [`MIXTAPE-FORMAT.md`](./MIXTAPE-FORMAT.md)
- The archived source mixtape: `/Users/arturo/Documents/Projects/0-archive/liner-archive/2026-06-25-root-cleanup/mixtapes/jtbd-master-prompt/`
- The Liner curation methodology that places this work in Phase 1: [`CURATION.md`](./CURATION.md)
- The skill that will likely embed this prompt: [`SKILL.md`](./SKILL.md)
- The mixtape format spec defining the `jtbd:` field the prompt's output lands in: [`MIXTAPE-FORMAT.md`](./MIXTAPE-FORMAT.md)
- Project-level overview of Liner: [`LINER-MASTER.md`](./LINER-MASTER.md)
