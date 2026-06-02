# Curating Mixtapes

A methodology for assembling AI-consumable context bundles that actually work.

---

## What this document is

A methodology for building mixtapes — curated bundles of sources designed to give an AI substantive context on a specific topic, so it can produce better answers, better drafts, and better judgments than it would from its training data alone.

This document is platform-agnostic. It describes *how to curate well*, not how any specific tool implements curation. Liner is one implementation; the methodology applies equally to mixtapes built with other tools or by hand.

The methodology has two intended audiences:

- **Tools running the methodology automatically** — like the Liner skill, which encodes this document as instructions for an AI executing the lifecycle on a curator's behalf.
- **Curators practicing the methodology themselves** — building library-quality mixtapes by engaging substantively at each step.

Most users will be in the first group. Their tool runs the methodology for them; they confirm or lightly edit. The second group is the minority that produces the best mixtapes — including everything in the Liner library. This document serves both.

---

## The bar

A good mixtape demonstrably improves the AI's output on tasks within its domain. That's the only test that matters.

A bad mixtape is one of three things: a bookmark dump (no curation), a popular-content collection (no taste), or a topical Wikipedia-style overview (no specificity). All three are easy to produce. None of them help.

What makes a mixtape useful for an AI is different from what makes a reading list useful for a human. An AI doesn't need motivation, doesn't get bored, doesn't benefit from inspirational framing. What it does benefit from:

- **Primary sources over commentary.** The original specification beats someone explaining the specification.
- **Concrete over abstract.** "Touch targets should be 44pt minimum because..." beats "design with users in mind."
- **Specific over general.** A 20-minute deep dive on one pattern beats a 60-minute survey of ten.
- **Recent enough to be accurate, old enough to be durable.** The sweet spot for most topics is one to four years old for principles, very recent for platform specifics.
- **Diverse perspectives where the field is contested.** If experts disagree, include both sides. The AI synthesizes better from disagreement than from echo chambers.

A mixtape stuffed with motivational talks will produce worse AI output than a mixtape of three dense technical specifications. Counterintuitive but real.

---

## The lifecycle

Every mixtape moves through the same phases. The phases are ordered because each one's output is the next one's input. Skipping phases or running them out of order produces lower-quality mixtapes — even if the result looks complete.

### Phase 1 — Framing

Define the job-to-be-done. Not the topic. The *use case*.

The JTBD is written as a Job Story: **"When [circumstance], I want [motivation], so I can [outcome]."** All three slots are required. The form is permissive on what fills them — a legal, clinical, or engineering Job Story share this shape but no vocabulary. What it guarantees is that no slot can be empty, which raises the floor on what counts as a JTBD before any qualitative judgment kicks in.

"Mobile design" is a topic. It is too vague to curate against. Three Job Stories in that same topic produce three different mixtapes:

- *When I'm designing the core UI of a consumer iOS app, I want to follow Apple's interaction conventions while choosing where to deviate visually, so I can ship something that feels native without looking generic.*
- *When I review mobile design portfolios as a senior IC hiring manager, I want to compare candidates against a consistent rubric of taste and decision-making, so I can decide who to advance with confidence.*
- *When I'm choosing a platform strategy for a new mobile product, I want to weigh native vs. cross-platform against the specific constraints of my team and timeline, so I can commit to one path without re-litigating the question every quarter.*

If the job-to-be-done can't be stated as a tight Job Story — single circumstance, single motivation, single outcome, no solutions named, no aspirational mush — the mixtape will be unfocused. This is the single most common failure mode. The [JTBD Master Prompt](JTBD-MASTER-PROMPT.md) is a drop-in interview that elicits one in a single human turn; use it (or have the AI use it) at the start of any new mixtape.

Sketch a knowledge map alongside the JTBD: the major areas of the domain, the sub-areas, the edge cases. For mobile design, that might be foundations, patterns, craft, process, edge cases. For another domain, the buckets are different. The map serves three purposes: it structures research, it surfaces gaps, and it becomes the section structure of the finished mixtape.

Treat the map as a hypothesis. It will get revised during research.

### Phase 2 — Candidate discovery

Generate a long list of candidate sources covering the knowledge map. Lots of them. This phase is about recall, not precision — gather more than you need; you'll cut later.

Methods that produce better candidates than generic search:

**Follow the citations.** Pick one excellent piece on the topic and find what *it* cites or links to. Then follow those sources' citations. Three or four hops in, you'll find the canonical sources nobody links to anymore because everyone assumes you've read them.

**Reverse-engineer experts' reading lists.** Many domain experts publish their own reading lists or syllabi. Constrain searches to platforms like Notion, GitHub, Are.na, or personal blogs. Are.na in particular is underused.

**Mine course syllabi.** University courses, bootcamp curricula, and serious online courses have already done curation work. `"<topic> syllabus" filetype:pdf` is a productive query.

**Find who experts cite.** Pick three to five acknowledged experts. Look at *their* references — bibliographies, acknowledgments, who they cite in their own writing. The overlap is your shortlist.

**Conference proceedings.** Going through three years of a specific conference is more efficient than searching for talks topic-by-topic.

**Ask the AI to find gaps, not sources.** Once you have a candidate list, ask: *what's missing from this collection for someone trying to [job-to-be-done]?* AI is bad at recall and good at critique. Use it for the right thing.

No fetching yet. This phase produces URLs and titles, not content.

### Phase 3 — Fetching

Pull content for every candidate into the cache. Transcripts, articles, metadata. This is the expensive phase — network calls, transcript scraping, content extraction.

Two things matter here. First, fetching is part of research, not a tail step. A source can't be honestly evaluated without reading it. Second, fetched content should be cached. The same source used in multiple mixtapes should be fetched once.

After fetching, every candidate has readable content available for evaluation.

### Phase 4 — Evaluation

Read the fetched content and rate each candidate against the job-to-be-done. This is where keep/trim/drop decisions actually happen.

Source quality, roughly in order of preference:

1. **Canonical references.** Apple HIG, Material Design docs, W3C specs, RFC documents, official platform documentation. Always include the primary source if one exists.
2. **Practitioner deep-dives from people who built things.** Engineers and designers at credible companies writing about specific decisions they made. Concrete and battle-tested.
3. **Conference talks from credible speakers.** Pre-filtered by the conference's curation. Better than random video search.
4. **Books, but chapters not whole books.** Link to a specific chapter or excerpt. A tape isn't a book club.
5. **Long-form interviews with practitioners.** Best when you cite a specific episode for a specific point.
6. **Academic papers when relevant.** Often underrepresented in design and engineering mixtapes.

What to avoid:

- Listicles ("10 tips for...")
- AI-generated content (now widespread, often subtly wrong)
- Twitter threads as primary sources
- Vendor blog posts with promotional angles
- Most TED-style talks — too much narrative, not enough density
- "Ultimate guides" — usually shallow despite the name

Evaluations should include a brief reason. "This source is canonical for the foundations section." "This source duplicates Source 4 — drop the weaker one." "This source has a strong perspective; useful but read alongside Source 7." These reasons go into the working notes.

### Phase 5 — Quality checks

Four tests every mixtape should pass. Run them deliberately.

**The redundancy test.** Read each source's role in the mixtape. Are any two sources making essentially the same point? If yes, keep the better one and cut the other. Mixtape volume is not a virtue. Five excellent sources beat fifteen good ones.

**The coverage test.** Walk through the knowledge map. Any bucket with zero sources? Either fill it or explicitly note that the mixtape doesn't cover that area. A mixtape that silently omits a major area is worse than one that says "this mixtape doesn't cover X — see [other mixtape] for that."

**The disagreement test.** Pick the strongest claim made by sources in the mixtape. Find someone credible who disagrees with it. Either include the disagreement or explicitly note that the mixtape takes a position. A mixtape that only includes one side of a contested question produces worse AI output, because the AI then can't see the contested-ness.

**The framing-gap test.** Step back and look at the whole mixtape. Is there a major perspective on this topic that's missing entirely — not just an individual source, but a whole way of thinking about the JTBD? If yes, the framing was too narrow. Expand the knowledge map and revisit Phase 2.

This last test is the one most curators skip. It's also the highest-leverage. Single-pass curation tends to produce mixtapes that are well-covered within one narrow framing. The framing-gap test catches this.

If a mixtape fails any of these tests, fix it before moving on.

### Phase 6 — Synthesis

Write the synthesis. This is the distilled understanding of the domain expressed in the curator's own voice — the principles, the contested questions, the framework distinctions, the recommended applications.

The synthesis lives at the top of the compiled mixtape. It's the first thing the consuming AI reads. Without a synthesis, the AI gets a list of sources and curator notes; with a synthesis, the AI gets the curator's framing of the entire domain. The difference in downstream AI output is substantial.

Length: 800–2000 words is typical. Shorter for narrow topics; longer for broader ones.

The synthesis is required, not optional. In automated curation, the AI drafts it. In methodology-mode curation, the curator writes or substantially edits it. Either way, every mixtape has one.

### Phase 7 — Final assembly

Compile the mixtape. The compile step:

- Reads the recipe and the synthesis
- Reads cached source content for every kept source
- Writes the master file (synthesis + how-to-use + source index)
- Writes individual source files referenced by the master file

The output is a self-contained mixtape folder ready to share, publish, or paste into an AI conversation.

### Phase 8 — Empirical test (optional but recommended)

Theory aside — here's how to know if the mixtape is actually working:

1. Compile the mixtape
2. Paste it into an AI conversation as context
3. Ask the AI a question that the mixtape's job-to-be-done should help with
4. Ask the same question without the mixtape in a separate conversation
5. Compare the answers honestly

If the answer with the mixtape is meaningfully better — more specific, more correct, more nuanced, less generic — the mixtape is working. If the answers are similar, the mixtape isn't earning its place. Either the sources are too generic, the synthesis isn't doing enough work, or the JTBD is too broad.

This empirical check is required for library contributions. Run it. Document the result.

---

## Curator notes

Every source in a mixtape has a curator note. The note tells the AI (and the human reading the mixtape) three things:

- *Why this source is in the mixtape.* Signals weight. "Watch first — this is the foundational piece" vs. "Skim if time permits — useful for context but not essential."
- *Where the high-value portion is.* "The first ten minutes are setup, the load-bearing content starts around 10:30."
- *What the source's limitations are.* "Dated on platform specifics, durable on principles."

Good notes are 15–40 words. Shorter is fine when the source is self-explanatory. Longer is usually a sign the source needs more framing — or that the note is doing work the synthesis should be doing.

In automated curation, the AI drafts notes during evaluation. In methodology-mode curation, the curator writes or substantially edits them. Notes are not optional.

---

## Personal sources

Not every source has a public URL. The strongest mixtapes often draw on material that isn't publicly available — book chapters, gated articles, course transcripts, personal notes. These are often the highest-density material in a mixtape and shouldn't be excluded just because they aren't on the public web.

Cite non-public sources clearly. Author, title, publication, date. The compiled mixtape uses these citations to signal what kind of source the AI is looking at, which matters for how it weights the content. A passage labeled as a 2019 book chapter gets read differently than one labeled as a recent newsletter post.

---

## Sharing mixtapes

Most mixtapes are personal — built for your own thinking, used by you, drawing on whatever you have legitimate access to. There's nothing to think hard about here.

Sharing is different. A mixtape passed to a colleague or posted publicly distributes its contents to people other than you. Sources that are fine for personal use may not be appropriate to redistribute. Use your own judgment.

When in doubt, share the recipe (the tape file) rather than the compiled mixtape. The recipe references sources; the compiled mixtape contains them. They have different properties when they leave your hands.

---

## The two modes

The methodology supports two intensities of engagement.

**Quick mode** runs the methodology automatically. The curator provides the JTBD and confirms the AI's proposals at each gate. The AI does the heavy lifting — generating candidates, evaluating, drafting curator notes, drafting the synthesis. The curator's job is to catch obvious errors and make final calls. A complete mixtape takes 15–30 minutes. Output is good enough for personal use.

**Methodology mode** treats the AI as a research assistant rather than the curator. The curator engages substantively at each gate — revises the knowledge map, reviews evaluation rationales, makes deliberate keep/drop decisions, writes their own synthesis. The AI accelerates the tedious work; the judgment stays human. A complete mixtape takes several hours. Output is library-eligible.

Both modes use the same lifecycle and produce the same artifact shape. They differ in how much human judgment shows up in the working notes and the synthesis. Quick mode mixtapes can be upgraded to methodology mode by going back and doing the substantive work; methodology mode mixtapes can be produced in one pass by curators who already know the domain.

The default is quick mode because that's what most users want. Methodology mode is the path for serious curation work, including all library contributions.

---

## Common failure modes

Patterns that produce bad mixtapes. Watch for them.

**The greatest-hits mixtape.** A list of the most famous resources on a topic. Looks impressive, adds little value because the AI already knows about famous resources from training. Curate for *non-obvious* sources, not popular ones.

**The unfocused mixtape.** No clear job-to-be-done; tries to cover the entire field. Result: shallow on everything, deep on nothing.

**The author-tribute mixtape.** Three of the five sources are by the same person, because the curator is a fan. This narrows perspective and reduces the AI's ability to synthesize across views.

**The transcript dump.** Many video transcripts, no articles, no specifications, no primary sources. Transcripts are useful but conversational and low-density. Mix sources.

**The noteless mixtape.** Sources without curator notes. Either the curator was lazy or didn't actually have a reason for each source. Both are problems. If a source's role can't be articulated, it doesn't belong.

**The aspirational mixtape.** Sources the curator thinks people *should* read, not sources that help with the actual job-to-be-done. Mixtapes are tools, not reading recommendations.

**The single-pass mixtape.** Curated without a framing-gap test. Well-covered within one narrow framing, blind to whole perspectives the curator didn't think to look for.

---

## A note on AI assistance

Most curators will use AI to help with research, source discovery, and drafting. This is fine and expected. The work goes better when the AI is treated as a collaborator with biases, not an oracle.

AI is especially prone to:

- Suggesting popular and recent sources at the expense of canonical or older ones
- Hallucinating sources that sound plausible but don't exist (always verify URLs)
- Producing generic curator notes that say "this is a useful resource" without specificity
- Missing primary documents (specs, papers) in favor of secondary commentary
- Failing the framing-gap test because it doesn't notice what it didn't search for

Compensate deliberately. Search for primary sources yourself even when AI doesn't surface them. Verify every URL. Treat AI-drafted notes as first drafts. Run the framing-gap test with intent.

The methodology assumes a human in the loop. Quick mode reduces what the human does; it doesn't eliminate it. The minimum is reviewing the AI's proposals at each gate. Anything less is automated source-listing, not curation.

---

## What this is not

Not a content recommendation system. Not a way to compile reading lists. Not optimized for human learning paths — those are different artifacts with different constraints.

This is a way of producing artifacts that make AI systems demonstrably more useful within a specific domain. Everything in this document serves that goal.

If decisions start prioritizing human readability or aesthetic completeness over AI utility, the methodology is drifting. Drift back.

---

*Current version: 2.0. Changes are tracked in the repository changelog.*
