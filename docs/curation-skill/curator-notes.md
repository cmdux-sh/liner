# Curator notes

Loaded by the `curating-mixtapes` skill at the start of Phase 4 (Evaluation).

This file describes how to draft curator notes. Apply it whenever you're writing the `note` field for a kept or trim source. Notes are drafted in Phase 4 during evaluation and land in `tape.yaml` at Phase 7 — write them in their final form now; don't postpone the work.

Curator notes are the highest-leverage feature in the mixtape format. The note is what tells the consuming AI which sources matter for which questions. The synthesis frames the domain; the notes route the AI's attention to specific sources when a specific question arises. Without notes, the AI sees a list of sources and has to guess which ones are relevant. With good notes, the AI can route precisely.

This is what "liner notes" means in the product name. The note is doing real work.

---

## The three-thing template

Every good note answers three questions about its source:

1. **Why is this source in the mixtape?** What role does it play? Is it foundational, supplementary, contested, illustrative? "Watch first — this is the foundational piece." "Skim if time permits — useful for context but not essential." "The contested counterpoint to Source 4."

2. **Where is the high-value portion?** Most sources are not uniformly dense. A talk's first ten minutes might be setup; a paper's middle section might be the methodology no one needs; an article's three closing paragraphs might be the only part that matters. Name it. "The first ten minutes are setup; the load-bearing content starts around 10:30." "Section 3 is the substantive one." "Skip the introduction; the framework is in the second half."

3. **What are the source's limitations?** Where is this source out of date, narrow, or biased? "Dated on platform specifics, durable on principles." "Strong perspective; read alongside Source 7 for the counterargument." "Pre-2022 — doesn't address [recent development]."

A note that addresses all three is doing its full job. A note that addresses one or two is adequate. A note that addresses none — "this is a useful resource" — is useless and shouldn't ship.

---

## Length

**15–40 words typically.** Shorter is fine when the source is self-explanatory. Longer is usually a sign of one of two things:

- The source needs more framing than a note should carry. Move some of the framing into the synthesis instead.
- The note is doing work the synthesis should be doing. If you're explaining a framework concept in a note, that's a synthesis problem — the note is patching for the synthesis's gap.

When a note exceeds 60 words, stop and ask which of these two patterns is happening. Fix the underlying issue rather than padding the note.

---

## What makes notes bad

Notes drift toward generic praise by default. These are the patterns to watch for in your own drafts.

- **"This is a useful resource."** Says nothing. Names no role, no location of value, no limitation. The AI can't route on this. Replace it with a specific answer to at least one of the three questions.
- **"Comprehensive overview of [topic]."** Implies the source is uniformly valuable, which is almost never true. Even if the source really is comprehensive, the AI needs to know which parts matter for which questions.
- **"Highly recommended."** Reviewer-style praise, not curator routing. The fact that a source was kept means it's recommended. The note's job is to say *how to use it*, not to vouch for it.
- **Restating the title or abstract.** If the title is "Designing CLIs People Love," the note shouldn't say "About designing CLIs people love." Say what *role* it plays in this corpus.
- **Pure summary.** "Argues that CLIs should follow Unix conventions." That's what the source says. The note should say what the source is *for* in this mixtape — foundational, contested, etc.

---

## How notes interact with the synthesis

The synthesis and the notes do different jobs.

- **The synthesis** frames the whole domain. It's what the consuming AI reads first, and it sets up the curator's view of the territory.
- **The notes** route the AI's attention to specific sources during a specific conversation. When a question comes up that one source addresses particularly well, the note is the signal that says "open this one."

This division is why notes don't need to explain the source's content — the synthesis already provided the conceptual frame. The note's job is positional, not explanatory.

A common failure mode is writing notes that try to teach the AI what the source says. That's the synthesis's job. The note's job is to tell the AI when and how to use the source.

---

## Quick mode vs. methodology mode

**Quick mode:** AI drafts the note during Phase 4. Use the three-thing template. Show the curator the drafted notes at Gate 2. Default is "looks good, save it." Light edits welcome.

**Methodology mode:** AI drafts a starting point. Expect the curator to rewrite each note substantively. The notes carry the curator's voice and the curator's specific framing of what each source is for. AI-drafted notes with light edits don't meet the methodology-mode bar.

In both modes, notes are required. A source without a note doesn't ship.

---

## Worked examples

> TODO: Insert three to five worked examples from the Cowork CLI/TUI synthesis. Each should show all three things landed. Look for notes in the Cowork material that are concrete about role, location of value, and limitations — not vague praise.
>
> Suggested examples to pull:
> - A "watch first" foundational note (likely on a Van Slyck talk or clig.dev)
> - A note with specific time-coding on a video ("the section starting at 14:30")
> - A trim note that explicitly says what to skip ("Skip the OpenStack-specific noise")
> - A "contested counterpoint" note that names the source it disagrees with
> - A note that names a temporal limitation ("dated on X, durable on Y")

> TODO: Insert one or two examples of bad notes for contrast. Pull the original AI-drafted notes from OpenClaw if they're available — those are the canonical "this is a useful resource" generic notes that Cowork rewrote.
