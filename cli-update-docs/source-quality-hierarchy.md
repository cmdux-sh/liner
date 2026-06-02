# Source-quality hierarchy

Loaded by the `curating-mixtapes` skill at the start of Phase 4 (Evaluation).

This file describes how to rank sources, what to avoid, and the patterns to watch for when deciding keep / trim / drop. Apply it after you've fetched every candidate in Phase 3 and before you write `working/03-evaluation.yaml`.

The goal of Phase 4 is honest assessment against the JTBD. Not breadth. Not completeness. Not even fairness to every source. *Does this source help the consuming AI do the JTBD better?* If yes, keep it. If partially, trim it. If no, drop it.

---

## Source-quality hierarchy

Roughly in order of preference. This is not a strict ranking — context matters — but it's the order to default to when comparing sources of similar topical relevance.

### 1. Canonical references

Specs, standards, official platform docs, RFC documents. The primary source for whatever the topic is.

Examples by domain: Apple Human Interface Guidelines, Material Design specs, W3C standards, IETF RFCs, language specifications, official API documentation, government regulatory documents.

**Always include the primary source if one exists.** No piece of commentary on a spec is a substitute for the spec itself. The AI synthesizes better from the primary document than from anyone's interpretation of it.

### 2. Practitioner deep-dives from people who built things

Engineers, designers, and operators at credible companies writing about specific decisions they made. Concrete, battle-tested, often with consequences described honestly. The opposite of motivational content.

Hallmarks: a specific decision is named, the trade-offs are named, the outcome is described (including when the outcome was bad). The post-mortem genre is the canonical form.

### 3. Conference talks from credible speakers

Pre-filtered by the conference's curation. A talk that made it onto a credible conference's main track has already passed at least one quality gate. Better baseline than random video search.

Two notes: (a) talk transcripts are lower-density than articles — assume more skim, less detail. (b) Same talk reposted at multiple conferences indicates substance; keep the longest cut and drop the duplicates.

### 4. Book chapters

Specific chapters or excerpts, not whole books. A mixtape isn't a book club — the consuming AI doesn't need 300 pages on a topic when 30 will do. Link to the chapter or include the excerpt.

Books are particularly valuable for foundational concepts and frameworks that take a chapter to develop. They're less valuable for recent specifics, which change faster than books update.

### 5. Long-form interviews with practitioners

Useful when you can cite a specific episode for a specific point — "Episode 47, the section on incident response, 18 minutes in." Less useful as a general "this practitioner is interesting" reference.

Interview transcripts are conversational and lower-density than written work. They earn their place when the practitioner says something in the interview they haven't written down anywhere else.

### 6. Academic papers when relevant

Often underrepresented in design and engineering mixtapes because the field skews toward practitioner blogs. When a question is genuinely empirical or has been studied formally, the paper beats any number of blog posts. Don't skip them by default just because the format is intimidating.

---

## What to avoid

Source types that consistently produce low-value mixtapes. Drop these unless a specific exception applies.

- **Listicles ("10 tips for...", "7 things every X should know").** Optimized for clicks, not depth. Almost always shallow despite specificity in the title.
- **AI-generated content.** Now widespread, often subtly wrong, frequently fluent enough to mislead. Detection is imperfect but the tells are: generic phrasing, no specific examples or named cases, no opinionated stance, suspiciously even coverage across sub-topics.
- **Twitter / X threads as primary sources.** Real expertise sometimes lives on Twitter, but threads are usually a teaser for longer-form work elsewhere. If a thread is the primary source, look for the article it was excerpted from. If there isn't one, ask whether a thread really earns a place in the corpus.
- **Vendor blog posts with promotional angles.** Even when the technical content is good, the promotional framing distorts the analysis. Use as background, not as primary references.
- **Most TED-style talks.** Optimized for narrative, emotional resonance, and applause moments. Low density of transferable content per minute. Exceptions exist but are rare.
- **"Ultimate guides," "complete guides," "everything you need to know."** The promise in the title is almost never met. Treat as a yellow flag.

---

## Decision rubric

For each fetched source, decide one of: kept, trim, dropped. Rate kept and trim sources 1–5.

- **Keep (4–5).** Strong fit with JTBD. Canonical, distinctive, or load-bearing for a knowledge-map section that would otherwise be empty. The AI's downstream output is clearly better with this source than without it.
- **Trim (3).** Real content but selective. Worth including with a curator note flagging what to read and what to skip. Trim is not "kept with reservations" — it's "kept *because* the note tells the consuming AI how to use this source." The note is doing real work.
- **Drop (1–2).** Tangential, generic, low-density, or redundant with stronger sources already kept. Drop without guilt. The cost of dropping a marginal source is low; the cost of keeping it is higher than it looks because it dilutes the corpus.

**Rating calibration.** A 5 is "this source alone makes the mixtape meaningfully more useful." A 4 is "this source clearly belongs and pulls weight." A 3 is "this source belongs only with selective framing in the note." A 2 is "this source duplicates a stronger one or has shallow content." A 1 is "this source actively makes the corpus worse if included."

If you find yourself rating most sources 4–5, you're under-discriminating. The point of evaluation is to discriminate.

---

## Patterns to watch for

These are the recurring failure modes during evaluation. Each one is a moment where a source seems worth keeping but isn't.

### Redundancy

Two sources making the same point with similar examples. Keep the stronger one; drop the other. Two sources covering the same ground from different angles is not redundancy — it's coverage. The test is whether the AI would generate meaningfully different output from one vs. both. If not, redundant.

### The duplicate trap

Same talk reposted at different conferences. Same article republished on different sites. Same interview transcribed by multiple outlets. Keep one — usually the longest, most-cited, or earliest-published version. Drop the rest.

### Framework boilerplate masquerading as design content

Common with implementation tutorials. The video or article uses design vocabulary ("usability," "user experience," "accessibility") but teaches no transferable principles — it's a walkthrough of how to use a specific framework's API.

Drop unless the source encodes a genuinely transferable pattern. A Ratatui tutorial that happens to mention layout principles is a Ratatui tutorial. A Ratatui tutorial that articulates the immediate-mode rendering principle and shows why it matters for terminal UI architecture is a design source.

### The rescue move

A source flagged as "low-value" during candidate discovery (Phase 2) sometimes turns out to be the only one that addresses a specific gap once you've actually read every candidate. If a low-priority source uniquely covers a knowledge-map section that would otherwise be empty or thin, rescue it. Mark the decision and the rationale clearly.

> TODO: Insert worked example from the Cowork CLI/TUI synthesis. The canonical example is the Caligula segment inside transcript #34 "some cool Linux programs you've never seen" — OpenClaw flagged it as lower-value, Cowork rescued it as "the closest direct precedent in your whole corpus." This is the textbook rescue move. Pull the specific decision and the rationale used.

### Author overrepresentation

Three of five sources in a section are by the same author. This narrows perspective and reduces the AI's ability to synthesize across views. Keep the single strongest piece by that author; replace the others with sources by different practitioners. The exception: if the author has genuinely written the canonical work in multiple sub-areas, and no other practitioner has equivalent work, keeping multiple pieces is justified — but you should be able to defend it.

### Recency without substance

Recent does not mean better. A 2024 article that summarizes well-known points is weaker than a 2018 article that originated them. Don't auto-prefer recent. Don't auto-discount old. Apply the same JTBD-fit test regardless of publication date.

---

## Output: `working/03-evaluation.yaml`

Write every candidate's decision, regardless of whether it was kept or dropped. The dropped decisions matter — they're how the working notes show that evaluation actually happened.

```yaml
candidates:
  - url: https://example.com/source
    title: Source title
    decision: kept            # kept | trim | dropped
    rating: 5                 # 1-5; required for kept and trim
    section: foundations
    rationale: One-sentence reason this decision was made.
```

For dropped sources, the rationale should name *why* dropped — "duplicates Source 4," "shallow listicle," "AI-generated," "off-topic relative to JTBD." Generic "low quality" is not useful. Be specific.

> TODO: Insert two or three worked evaluation entries pulled from the Cowork synthesis — one keep with rating 5, one trim with rating 3, one drop with a specific rationale. These are the highest-leverage examples in the file.
