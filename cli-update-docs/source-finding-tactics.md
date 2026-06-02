# Source-finding tactics

Loaded by the `curating-mixtapes` skill at the start of Phase 2 (Candidate discovery).

This file describes the six tactics that produce better candidates than generic search. When you reach Phase 2, work through these tactics — not all of them for every mixtape, but enough that the candidate pool covers the knowledge map from multiple angles.

The goal of Phase 2 is recall, not precision. You're producing two to four times the eventual kept count. You will cut in Phase 4. Casting wide here is the methodology working as designed.

---

## 1. Follow the citations

Pick one excellent source you already have. Find what *it* cites — the references, the linked sources, the "see also" pointers. Then open one of those and find what *it* cites. Three or four hops in, you'll find the canonical sources nobody links to anymore because everyone assumes you've read them.

This tactic is the single highest-leverage move for discovering primary documents. It works against your popularity bias: cited-by-the-canon sources don't surface in keyword search, but they're what serious work in the field rests on.

**How to apply:** Open the most credible source you have. Scan the references, footnotes, or "further reading" section. Note every source that's cited *and* recurs in another credible source's references. Recurrence across multiple credible sources is the signal — it's how you find the references nobody links anymore.

> TODO: Insert worked example from the Cowork CLI/TUI synthesis. Look for a moment in the research where citation-chasing surfaced a canonical source that didn't show up in any initial keyword search. Form Design Patterns by Adam Silver is a candidate — it's cited across multiple Phase 1 sources but didn't surface from "CLI wizard design" searches.

---

## 2. Reverse-engineer experts' reading lists

Many domain experts publish their own reading lists or syllabi. These already encode someone else's curation work. Use them.

**Where to look:** Notion (search "site:notion.so reading list <topic>"), GitHub (`awesome-<topic>` repositories, individual `README.md` files in personal repos), Are.na (channels — Are.na in particular is underused for this), personal blogs, course-instructor pages.

Are.na deserves a specific note: it's a network designed for visual and textual reference-collecting. Practitioners use it to curate the kind of sources that don't make it into mainstream publications. Constrain a search with "site:are.na <topic>" and look for channels related to your JTBD.

**How to apply:** Find three to five experts in the domain. Search each one's name plus "reading list" or "syllabus" or "references." Catalog what they recommend. Cross-reference with the candidates already on your long-list — overlaps are strong signal; uniques are leads.

> TODO: Insert worked example from the Cowork CLI/TUI synthesis. Look for a tactical mention of an expert reading list that surfaced relevant work. If none exists, this section can use an example from a different domain — the methodology generalizes.

---

## 3. Mine course syllabi

University courses, bootcamp curricula, and serious online courses have already done curation work for their domain. A well-designed syllabus is a curated reading list optimized for learning. Use them.

**Productive queries:**

- `"<topic> syllabus" filetype:pdf`
- `"course readings" <topic>`
- `<topic> "required reading"`
- University course catalogs for the relevant department

**How to apply:** Find two to three syllabi on your topic. Note the readings that appear across multiple syllabi — those are the canonical pieces. Note the readings that appear once but are written by people you recognize — those are the practitioner-credible additions.

Course syllabi skew toward foundations. They're best for the early-section knowledge-map buckets. Recent practitioner work is rarely on them.

> TODO: Insert worked example. If no syllabus surfaced in the CLI/TUI research, note this and use a hypothetical that's specific enough to be useful.

---

## 4. Find who experts cite

Pick three to five acknowledged practitioners in your domain. Look at *their* references — bibliographies, acknowledgments, who they cite in their own writing. The overlap across these experts is your shortlist.

This tactic is structurally similar to citation-chasing (#1), but starts from people rather than from sources. Use it when you have a clear sense of who the credible practitioners are but no clear sense of which of their works is canonical.

**How to apply:** Identify three to five practitioners credible in the domain. Open their most recent significant work. Note the sources they cite. Open the sources cited by two or more practitioners. Those are your high-priority candidates.

> TODO: Insert worked example from CLI/TUI research. Carolyn Van Slyck, Aanand Prasad (clig.dev), Will McGugan (Textualize), and Amanda Pinsker (GitHub CLI) are candidates whose citation patterns could be traced.

---

## 5. Conference proceedings

Going through three years of a specific conference is more efficient than topic-by-topic search. The conference's curation has already filtered for credibility and substance.

**How to apply:** Identify one or two conferences serious in the domain. Open their proceedings, talk lists, or YouTube channels for the last three years. Skim titles and abstracts. Flag any talk whose abstract addresses a knowledge-map section.

Conference talks vary widely in density. The same talk reposted across multiple conferences is one signal of substance. Talks from speakers who are also cited by other practitioners (tactic #4) are another.

> TODO: Insert worked example. The CLI/TUI corpus has multiple Van Slyck talks across GopherCon and other venues — a real example of how conference-proceeding mining surfaced both her work and what to keep when the same talk appears at multiple conferences.

---

## 6. Ask the AI to find gaps, not sources

Once you have a candidate list, stop searching and ask: *what's missing from this collection for someone trying to [JTBD]?*

AI is bad at recall — at remembering and surfacing sources you don't know about. AI is good at critique — at looking at a list and noticing structural gaps. Use it for the right thing.

**How to apply:** Show the candidate list (titles and one-line rationales) along with the JTBD and knowledge map. Ask explicitly: "What perspective is missing? What discipline is underrepresented? What kind of source is this list short on?" Then go find sources to fill the gaps named.

This tactic is structurally related to the framing-gap test in Phase 5, but earlier and lower-stakes. Doing it in Phase 2 is cheaper than discovering the gap in Phase 5 and having to backtrack.

> TODO: Insert worked example from the Cowork synthesis. The Cowork run discovered that OpenClaw's initial 50-source list missed "the entire wizard/form-design tradition (Form Design Patterns, NN/g articles, Tidwell, Cooper)" — this is the canonical example of gap-finding working. Pull the specific moment where the gap was named and how it changed the candidate pool.

---

## What not to do

- **Don't rely on generic search alone.** The first page of search results is what your audience could also find. The point of curation is to surface what they couldn't.
- **Don't accept candidates without verifying URLs.** AI hallucinates plausible-looking URLs. Web-fetch every URL before it goes on the long-list. A 404 candidate is worse than a missing one.
- **Don't filter at Phase 2.** That's Phase 4's job. If a source might be useful, it goes on the long-list. Quality decisions come after fetching and reading.
- **Don't over-research one section while leaving others empty.** Walk the knowledge map. Each section should have a reasonable candidate pool before you go deep on any one section.
