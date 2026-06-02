# Synthesis guidance

Loaded by the `curating-mixtapes` skill at the start of Phase 6 (Synthesis).

This file describes how to draft `synthesis.md`. Apply it after Phase 5's quality checks have passed and the keep-list is locked. The synthesis is the most personal artifact in the mixtape and the one with the largest effect on downstream AI output. Don't shortcut it.

---

## What the synthesis is

The synthesis is the curator's distilled understanding of the domain expressed as continuous prose. It lives at the top of the compiled `MIXTAPE.md` and is the first thing the consuming AI reads.

Without a synthesis, the AI gets a list of sources and curator notes. It has to construct its own framing of the domain from those pieces, which dilutes the curator's view.

With a synthesis, the AI gets the curator's framing of the entire domain *before* it sees any source. The synthesis shapes how the AI reads everything that follows. This is the largest single lever for changing downstream AI output, and why the synthesis is required, not optional.

Length: **800–2000 words.** Shorter for narrow topics; longer for broader ones. If you find yourself below 600 words, the synthesis is probably too thin. If you're above 2500, the synthesis is doing work the sources should be doing.

---

## What goes in

Four kinds of content earn their place in the synthesis.

### Principles or framework

The conceptual structure the curator sees in the domain. Not a definition list — a framework. "Good CLI design rests on four pillars: predictable command shape, humane errors, useful defaults, and machine-readable output." That framework, once articulated, shapes how the AI reads every source in the corpus.

### Contested questions

Where credible practitioners disagree, name the disagreement and where the curator stands. "There's an ongoing argument about whether CLIs should mirror existing tool conventions or break them deliberately when the underlying behaviour differs. This mixtape leans toward mirroring; sources 7 and 12 represent the dissenting view."

Contested questions are gold for downstream AI output. They give the AI permission to represent the domain as contested rather than collapsing it into one answer.

### Distinctions between concepts that get confused

Where the domain has terms or concepts that get used interchangeably but shouldn't, draw the line clearly. "A *wizard* and a *form* are not the same thing. A form collects fields. A wizard collects fields *in a specific order, one at a time, with the option to navigate back*. The distinction matters because [...]."

These distinctions are what an AI without the synthesis would have to infer from context. With the synthesis, they're explicit and load-bearing.

### When to use the mixtape and when to look elsewhere

A short paragraph at the end: what kinds of questions this mixtape is meant to help with, and what kinds it isn't. This is honest about the corpus's scope. "This mixtape is calibrated for designing CLIs that humans will use interactively. It's less useful for non-interactive CI/CD scripts or for shell-script ergonomics — those are adjacent topics with different conventions."

---

## What doesn't go in

Content that looks like it belongs in the synthesis but doesn't.

### Source-by-source recap

"Source 1 argues X. Source 2 argues Y. Source 3 builds on Source 1 to argue Z." This is what a literature review does for an academic paper. The mixtape doesn't need it — the source index handles that role, and the consuming AI will read individual sources when it needs the specifics.

If the synthesis is recapping sources, the curator hasn't actually synthesized yet. Force the harder question: what does the curator think about the *domain*, given all these sources?

### Generic introductions

"Mobile design is the practice of designing applications for mobile devices, including smartphones and tablets, which have specific constraints around screen size, input methods, and connectivity." This is what the AI already knows. The synthesis isn't a Wikipedia stub. Start where the AI's training data ends.

### Tutorial content

"To design a good CLI, first identify your audience, then decide on a command structure, then..." This is how-to writing, not synthesis. The synthesis is *what the curator thinks about the domain*, not *how to do the work*.

### Cheerleading or motivation

"Good CLI design matters more than ever in an age of..." Cut. The AI doesn't need motivation. The curator's enthusiasm is irrelevant to the synthesis's job.

---

## Structure

A typical synthesis runs:

1. **Opening framing** (1–2 paragraphs). What this domain is fundamentally about. The frame the curator uses to organize their understanding. Not a definition — a stance.
2. **The framework or principles** (2–4 paragraphs). The structure the curator sees. Named, not just described.
3. **Contested questions** (1–3 paragraphs). Where credible practitioners disagree. Where the curator stands.
4. **Important distinctions** (1–2 paragraphs). Concepts the AI is likely to conflate without correction.
5. **Scope and limits** (1 short paragraph). What this mixtape covers and what it doesn't.

This is a default. Vary it when the topic warrants — some domains lead with contested questions, some with distinctions, some with a different opening move. The structure serves the content.

---

## Voice and stance

The synthesis is written in the curator's voice. First-person is fine. Strong stances are fine — preferred, actually. The AI synthesizes better from a confident framing than from a hedged one.

"I think the most important distinction in CLI design is..." is better than "Many practitioners consider one of the important distinctions to be..." The first is a stance the AI can engage with; the second is mush.

This doesn't mean overconfidence. Hedging is appropriate where the curator is genuinely uncertain. But default to stance. Mush in the synthesis produces mush in downstream output.

---

## Quick mode vs. methodology mode

**Quick mode.** The AI drafts the synthesis. Apply this guidance during drafting. Show the curator the draft. Default is "looks good, save it." The curator lightly edits if they want.

The risk in quick mode is that AI-drafted synthesis drifts toward source-by-source recap because that's the easier output shape. Resist this actively. The AI's draft should already be a real synthesis, not a placeholder that says "the curator will rewrite this."

**Methodology mode.** The AI drafts a starting point. Expect the curator to rewrite it substantially in their own voice. The methodology-mode synthesis is the curator's voice on the domain — AI-drafted starting points with light edits don't meet the bar.

Library contributions require methodology-mode synthesis. A synthesis that reads as AI-drafted is a signal the mixtape is quick-mode in methodology-mode clothing. The reviewers will catch it.

---

## A failure mode worth naming

The most common synthesis failure is one I'll call **synthesis-as-recap.**

An AI drafting a synthesis gravitates toward describing what each source says, because that's the immediately available material in context. It produces something that reads like a research summary: "Sources 1, 4, and 9 argue X. Sources 2 and 6 take a different view, arguing Y. Source 7 offers a synthesis of both positions."

This is not a synthesis. It's a recap. The consuming AI can produce its own recap from the source list; what it can't produce is the curator's framing of the domain.

The fix: stop describing sources, start describing the domain. The synthesis should be readable as a coherent essay on the domain by someone who happens to have read all these sources — not as a tour of the corpus.

If the synthesis names source numbers ("Source 1 argues..."), it's probably synthesis-as-recap. Real synthesis names ideas, not sources. The source index handles attribution; the synthesis handles framing.

---

## Worked examples

> TODO: Insert two worked synthesis excerpts from the Cowork CLI/TUI synthesis. The full Cowork document is the longest worked example available — pull the opening framing paragraphs as one example, and a contested-question or distinction paragraph as another.

> TODO: Insert one example of synthesis-as-recap (bad) for contrast. Either invent one against the CLI/TUI material or pull an earlier draft if one exists. The bad example should make the difference between "describing sources" and "framing a domain" obvious at a glance.

---

## Output: `synthesis.md`

Plain markdown. No front matter. Begins with a top-level heading that matches the mixtape title (e.g., `# Mobile Design Foundations — Synthesis`). The full file gets included verbatim at the top of `MIXTAPE.md` by `liner compile`.

Write the synthesis to `synthesis.md` in the project folder. Show the curator the result before declaring Phase 6 complete.
