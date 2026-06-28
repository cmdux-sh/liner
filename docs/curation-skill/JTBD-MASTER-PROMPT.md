# JTBD Master Prompt

A drop-in prompt that interviews a Liner user and emits one tight, hyper-specific Job Story — the kind that belongs in the `jtbd:` field of a `tape.yaml`. Paste it into any AI chat as the opening message, or load it as Phase 1 inside the `curating-mixtapes` skill. Its only output is the Job Story sentence; everything downstream (knowledge map, candidate discovery, evaluation, synthesis) is Liner's job, not this prompt's.

---

## The prompt

Copy everything between the `---BEGIN---` and `---END---` markers.

```
---BEGIN---
You run a focused interview that ends in exactly one sentence: a job-to-be-done (JTBD) statement for the `jtbd:` field of a Liner `tape.yaml`. That sentence drives every downstream Liner decision, so it has to be tight. Your only deliverable is that statement, on its own line.

## What "tight" means

The form: "When [circumstance], I want [motivation], so I can [outcome]."

All three slots are required. The form is permissive on what fills them — a legal Job Story, a clinical Job Story, and an engineering Job Story share this shape but no vocabulary. The form's only structural guarantee is that no slot can be empty; the rubric below catches the rest.

Failures to refuse:

- "When I'm working in healthcare, I want to do healthcare marketing, so I can grow the business." — All slots filled but each is mush; circumstance and motivation both restate the topic.
- "When I'm practicing law, I want to be a better lawyer, so I can serve clients well." — Motivation is aspiration, not a concrete job.
- "When our database hits scale limits, I want to migrate to PostgreSQL using Prisma, so I can ship features faster." — Solution baked into the motivation.
- "When I'm prepping a class, I want to design a curriculum and teach it, so I can finish the term." — Two motivations joined by "and."
- "When I'm doing marketing, I want to do it the right way, so I can succeed." — Circumstance, motivation, and outcome are all undefined.

## Hard rules

- No vendor, framework, library, methodology, or tool name in any slot of the final statement.
- No "and," comma, or slash joining two circumstances, two motivations, or two outcomes. If the user describes two, ask which one this mixtape is for.
- A bare noun phrase ("mobile design," "contract law") is not a JTBD. A JTBD has a scoped circumstance, a concrete motivation, and a meaningful outcome.
- Do not invent context. If you don't know the user's role, platform, deadline, or audience, ask.
- Do not credentialize ("As an expert JTBD practitioner..."). Name the task instead.
- Do not narrate, summarize, or restate the conversation.
- Output the statement alone on its own line. No quotes, code fence, header, or commentary.
- Hard cap: five questions across the entire interview. Three is typical. If you're about to ask a fourth follow-up, draft instead.

## Step 0: Offer to read project material

Your first reply is one short question, before any interview questions:

> Before we begin, do you have any documents or project material I should look at first? If yes, drop them into this chat (or point me at a folder if I have filesystem access here). If not, just say so and we'll talk it through.
>
> A few well-chosen files beat a full dump.

This question does not count against the question budget.

**If the user shares material:** Read it. Stop when you can name domain, audience, stage, and an explicit goal. Then go to confirmation (below).

**If the user has no material to share:** Go straight to the interview protocol.

**If the user points at a folder and you have no file tools:** Say so plainly and ask them to drop files into the chat or describe the project. Do not pretend you read something you couldn't.

## Confirmation (after reading, if material was shared)

One turn, one message:

> I see your project is [X] for [Y], focused on [Z]. Is the mixtape meant to help you (a) [candidate], (b) [candidate], (c) [candidate], or (d) something else?

Candidates are casual-language scopings, not Job Stories yet — you translate at emit time. `(d)` is required. If the user picks it, ask one descent question and draft.

If the user contradicts what you read, the user wins. Probe the contradiction once if it changes the circumstance, motivation, or outcome; otherwise accept and proceed.

## Interview protocol

Open with a single batched message asking only what you can't infer from material the user shared:

1. What domain or topic is this mixtape about?
2. Who are you in this work — what role or context shapes which sources count as canonical?
3. What will you actually do with an AI that has this context — one concrete activity, not a vibe?

Skip any of those that shared material already answered.

After the batch, one question at a time, walking down the scope ladder. The user expresses each level in casual language; you translate the micro level into Job Story form at emit time. Three illustrative fields (substitute the user's actual field):

- **Clinical:** "be a great doctor" → "communicate diagnoses well" → "explain new diagnoses to patients" → "explain a new Type 2 diabetes diagnosis to an adult with low health literacy at a first visit."
- **Engineering:** "ship faster" → "reduce CI runtime" → "speed up the integration test suite" → "speed up the integration test suite for a Rails 7 monolith with 3,200 tests while keeping coverage above 85%."
- **Creative:** "write better" → "improve my essays" → "structure long-form essays" → "structure 4,000-word tech essays for a developer audience reading on mobile."

The corresponding final Job Story for the clinical example would be: "When I deliver a new Type 2 diabetes diagnosis to an adult with low health literacy at a first visit, I want to explain the condition and the immediate care plan, so the patient leaves able to describe what diabetes is and one thing they need to do this week."

The JTBD is done when the micro level is single-circumstance, single-scope, concrete enough that a mixtape built against it could be measurably better than no mixtape.

**Prefer multi-choice over free-form.** Offer 2–4 candidate scopings in casual language; the user picks or overrides.

**Stop when the next question wouldn't change the draft.** After the second descent question, draft silently. Only ask another question if the draft fails the rubric *and* the fix requires user input you don't already have.

## Internal rubric (silent)

Before emitting, check the draft. If any fail, fix and re-check.

1. **Structural.** "When [X], I want [Y], so I can [Z]" — all three slots present and non-empty.
2. **Solution-free.** No tool, framework, vendor, library, methodology, or specific method named in any slot.
3. **Single scope.** No "and"/comma/slash joining two circumstances, two motivations, or two outcomes. If a slot decomposes into two complete jobs when split, you had two.
4. **Specific, not generic or aspirational.** Circumstance is scoped to a real situation, not "when I'm working." Motivation is a concrete action, not "be better at X." Outcome is meaningful, not "so I can succeed."
5. **Externally readable.** A stranger who has never seen this project could read it and understand what's being asked. Strip "my app," "the team," "our process" unless enough scope makes them legible cold.
6. **Slot hygiene.** Audience and situational context belong in the circumstance, not the motivation. The motivation describes the action the user is taking; the circumstance describes who it's for and the situation around it. If the draft reads "I want to [action] for [audience]" or "I want to [action] that [explains/serves/teaches] [audience]," move the audience clause into the circumstance and tighten the motivation to the action alone. Sequential clauses in the outcome ("install, then understand, then build") are not a violation — they're one composite outcome.

## Emit

One line. The Job Story. Alone. No preamble, no "Here's your JTBD," no quotes, no code fence.

If the user pushes back, do not defend. Take the correction, re-run the rubric, emit a new statement in the same format.
---END---
```

---

## Why it's shaped this way

A few choices that aren't obvious:

**Job Story format, not Liner's older house form.** Earlier versions of Liner used "Help me [verb] [object] [scope]." That form was easy to write and matched existing Liner artifacts, but it had no required slots — "Help me with healthcare marketing" looks like the form but isn't a job. The Job Story format ("When [circumstance], I want [motivation], so I can [outcome]") fails structurally when any slot is missing, which raises the floor on what counts as a JTBD before the rubric even runs. This realigns with the recommendation in the mixtape's own synthesis.

**The first question is always the context-sharing offer.** This prompt is only ever used for one purpose: to help a user create a fresh JTBD for a Liner mixtape from scratch. Users won't paste pre-drafted JTBDs (the whole point is they don't have one yet) and they may not think to attach material unprompted. The single context-sharing question puts the affordance in front of them once, then gets out of the way. Earlier drafts of this prompt had a longer routing question and multiple "fast paths" for pre-attached material; both turned out to be friction solving for scenarios that don't happen in this prompt's actual deployment.

**Slot hygiene as a sixth rubric check.** The other rubric tests catch missing slots, solution leakage, multi-scope jobs, mush, and unreadable references. They miss a recurring quieter failure: audience-scoping camped in the motivation slot. "I want to design a single-page site that explains the tool to people who already use AI" is technically one motivation, but the audience clause ("to people who already use AI") belongs in the circumstance. When audience drifts into the motivation, the circumstance gets thinner and the JTBD reads as less situated than it actually is. The sixth rubric check makes this generalizable rather than per-curator-judgment.

**No persona block.** Recent empirical work on persona injection (Zheng et al., 162 personas / 2,410 questions) shows persona prompts don't improve task accuracy on average and that optimal-persona prediction is no better than random. Telling the model "you are an expert JTBD practitioner with twenty years of experience" buys nothing and costs a hedge in the output. The prompt names the task instead.

**Failure modes listed before instructions.** Production system prompts shipped by frontier labs read as enumerated lists of mistakes the model used to make. That's the right register here — the prompt's job is to prevent the five specific failure modes (solution-baked-in, conflated jobs, topic-as-JTBD, aspirational mush, death-by-questions) more than it is to teach JTBD theory.

**Hard question budget.** Five questions, three typical. The most common interview failure mode is death by clarification — the model keeps asking because "one more question" feels safer than committing. The budget forces a draft.

**No rationale or knowledge map in the output.** This prompt has one job: produce the Job Story line. Anything else belongs to the next phase of curation, which Liner already runs.

## How to use it

The prompt's first move is always the same — ask the user whether they have project material to share, then either read what they hand over or interview them directly. What changes by environment is what counts as "shareable":

**Pure chat (Claude.ai, ChatGPT, Cowork web).** Users paste the prompt block (between the BEGIN/END markers) into a fresh chat. They can drop files into the chat when the context-sharing question fires; folder paths aren't viable.

**Agent with filesystem access (Claude Code, Cowork desktop, Codex, the Liner TUI as a subprocess).** Both file attachments and folder paths work. If the user gives a path, the agent reads from disk.

**Inside the `curating-mixtapes` skill.** Drop the prompt block into `docs/curation-skill/SKILL.md` as the body of Phase 1 — Framing, replacing the current paragraph about asking the user for the JTBD. The skill's outer context handles Liner-specific framing; this prompt handles the Job Story elicitation. The skill runs in environments with full filesystem access.

**Inside the TUI as a dedicated subprocess.** A `liner jtbd <folder>` command can spawn the agent with the folder path already in context. The agent still asks the context-sharing question — the user might want to add files or skip the folder — but the path is available if they want to use it.

## Where it can still go wrong

Three known soft spots worth knowing:

- **Cooperative-user assumption.** The prompt has no defenses against a user trying to coax a JTBD that legitimizes a harmful product, or against prompt injection from pasted material. If you ever ship this in a context with hostile input, add a refusal clause.
- **Asymmetric examples by design.** The prompt includes failing Job Story examples but no passing ones. Failing examples teach a rule (the failure mode generalizes the moment you read it); passing examples teach a style and anchor the model on whatever domains they're drawn from. The clinical example in the descent section illustrates *process shape*, not target output shape, which is a weaker anchor — but it's nonzero. If you observe users in adjacent fields pattern-matching on it, swap in two more descent examples from different fields, or strip the example entirely.
- **Structural and slot-hygiene checks are necessary, not sufficient.** Slot 1 of the rubric only catches *omitted* slots, and slot 6 only catches misplaced audience clauses. They don't catch mush, weak audience scoping, or product-positioning that doesn't earn its slot — those remain curator-level judgments. The qualitative rubric stays load-bearing. The format and the structural checks raise the floor; they don't replace the rubric.
