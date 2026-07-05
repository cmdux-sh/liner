# Source-finding tactics

Loaded by the `curating-mixtapes` skill at the start of Phase 2 (Candidate discovery).

This file describes tactics that produce better candidates than generic search. When you reach Phase 2, work through the relevant tactics — not all of them for every mixtape, but enough that the candidate pool covers the knowledge map and required source roles from multiple angles.

The goal of Phase 2 is recall, not precision. You're producing roughly two to four times the eventual kept count, but the real stop condition is coverage of the required source roles and capability-pattern lanes. You will cut in Phase 4. Casting wide here is the methodology working as designed.

---

## 1. Follow the citations

Pick one excellent source you already have. Find what *it* cites — the references, the linked sources, the "see also" pointers. Then open one of those and find what *it* cites. Three or four hops in, you'll find the canonical sources nobody links to anymore because everyone assumes you've read them.

This tactic is the single highest-leverage move for discovering primary documents. It works against your popularity bias: cited-by-the-canon sources don't surface in keyword search, but they're what serious work in the field rests on.

**How to apply:** Open the most credible source you have. Scan the references, footnotes, or "further reading" section. Note every source that's cited *and* recurs in another credible source's references. Recurrence across multiple credible sources is the signal — it's how you find the references nobody links anymore.

---

## 2. Reverse-engineer experts' reading lists

Many domain experts publish their own reading lists or syllabi. These already encode someone else's curation work. Use them.

**Where to look:** Notion (search "site:notion.so reading list <topic>"), GitHub (`awesome-<topic>` repositories, individual `README.md` files in personal repos), Are.na (channels — Are.na in particular is underused for this), personal blogs, course-instructor pages.

Are.na deserves a specific note: it's a network designed for visual and textual reference-collecting. Practitioners use it to curate the kind of sources that don't make it into mainstream publications. Constrain a search with "site:are.na <topic>" and look for channels related to your JTBD.

**How to apply:** Find three to five experts in the domain. Search each one's name plus "reading list" or "syllabus" or "references." Catalog what they recommend. Cross-reference with the candidates already on your long-list — overlaps are strong signal; uniques are leads.

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

---

## 4. Find who experts cite

Pick three to five acknowledged practitioners in your domain. Look at *their* references — bibliographies, acknowledgments, who they cite in their own writing. The overlap across these experts is your shortlist.

This tactic is structurally similar to citation-chasing (#1), but starts from people rather than from sources. Use it when you have a clear sense of who the credible practitioners are but no clear sense of which of their works is canonical.

**How to apply:** Identify three to five practitioners credible in the domain. Open their most recent significant work. Note the sources they cite. Open the sources cited by two or more practitioners. Those are your high-priority candidates.

---

## 5. Conference proceedings

Going through three years of a specific conference is more efficient than topic-by-topic search. The conference's curation has already filtered for credibility and substance.

**How to apply:** Identify one or two conferences serious in the domain. Open their proceedings, talk lists, or YouTube channels for the last three years. Skim titles and abstracts. Flag any talk whose abstract addresses a knowledge-map section.

Conference talks vary widely in density. The same talk reposted across multiple conferences is one signal of substance. Talks from speakers who are also cited by other practitioners (tactic #4) are another.

---

## 6. Ask the AI to find gaps, not sources

Once you have a candidate list, stop searching and ask: *what's missing from this collection for someone trying to [JTBD]?*

AI is bad at recall — at remembering and surfacing sources you don't know about. AI is good at critique — at looking at a list and noticing structural gaps. Use it for the right thing.

**How to apply:** Show the candidate list (titles and one-line rationales) along with the JTBD and knowledge map. Ask explicitly: "What perspective is missing? What discipline is underrepresented? What kind of source is this list short on?" Then go find sources to fill the gaps named.

This tactic is structurally related to the framing-gap test in Phase 5, but earlier and lower-stakes. Doing it in Phase 2 is cheaper than discovering the gap in Phase 5 and having to backtrack.

---

## 7. Search for the core action and the curator anchors

A good candidate list is not just "sources about the topic." It must contain
sources that help the consuming AI do the job.

Start by extracting the JTBD's core action. Examples:

- "translate moodboards into UI"
- "turn bug reports into reproduction steps"
- "write grant narratives from research evidence"

Then search for candidate URLs that might directly teach, demonstrate, or argue
for that action. Bridge candidates are allowed, but they cannot carry the corpus
by themselves. For a moodboard-to-interface mixtape, CSS layout, design tokens,
and design systems are bridge leads: useful after a direction exists, but not
enough to teach how to read images, extract mood, or create a new interface
artifact. The keep/drop decision still happens only after the content is fetched
and read.

When the curator names a person, studio, product, software tool, YouTube
channel, or site they like, deepen the anchor. Search for their case studies,
interviews, talks, process notes, docs, essays, videos, and worked examples.
Do not stop at the homepage, gallery index, or crawler metadata unless that is
the only readable evidence available. If it is only a pointer, say so in the
candidate reason.

For visual-output jobs, deliberately search for visual and worked-example
evidence: image-heavy case studies, moodboard breakdowns, design rationale,
portfolio projects with process, talks, and videos. A source list made mostly
of UX primers, CSS references, and design-system docs is probably under-serving
the job even if every knowledge-map section has a source.

---

## 8. Fill required source roles, not just sections

The Capability Brief should name **Required source roles**: the jobs the corpus
needs sources to perform for the future AI. These are not the same as source
`kind` (`reference / principle / prescription / example`) and not the same as
knowledge-map sections.

Examples:

- "core-action method" — sources that teach the exact action the agent must do
- "worked examples / case studies" — sources that show good output, process, or taste
- "domain authority" — primary/official sources the agent can cite for accuracy
- "implementation constraints" — specs, APIs, platform docs, safety limits, or production rules
- "critique / validation" — sources that help judge whether output is good
- "dissent / alternate stance" — sources that prevent one-frame bias
- "domain data" — keyword data, epidemiology, market signals, legal text, or other evidence specific to the niche

When you build the long-list, make sure every required role has candidates. A
populated section is not enough if a required role is thin. For a visual-output
mixtape, two dozen UX/design-system articles do not substitute for worked
visual/craft examples with rationale. For a medical SEO mixtape, SEO practice
articles do not substitute for authoritative medical sources and domain-specific
keyword or patient-language evidence.

**How to apply:** For each required source role, search until you have 3–6
plausible candidates or until it is clear public sources are thin. In the
candidate reason, name the role the source might satisfy. If you cannot find
credible candidates for a required role, write a "Role gaps" note in
`working/02-candidate-longlist.md` so Quality can recommend a focused
improvement pass instead of quietly accepting weak coverage.

Candidate count should fall out of the role plan:

- Narrow, single-domain projects may need fewer than 20 candidates.
- Ordinary projects usually land around 20–35 candidates.
- Broad, multi-domain, or `reference-translation` projects may need 40–60
  candidates because each source domain and translation lane needs its own
  candidate pool.

Do not pad the list to hit a number. Do not stop at a number when a material
role is still empty. Stop when every required role has credible candidates, the
knowledge map is covered, and any capability-pattern lanes named in Phase 1 have
enough evidence to evaluate.

Good candidate notes should say which role the source might serve:

- `source role: canonical method` — teaches the process the AI should follow
- `source role: domain authority` — grounds claims in a credible source
- `source role: applied example` — shows what good looks like in context
- `source role: boundary / risk` — prevents harmful or misleading output

---

## 9. Build a source ecology for reference-translation jobs

Some mixtapes are not just about a topic. They ask a future agent to read inputs
from one world and turn them into outputs in another. Common examples:

- images, moodboards, art direction, visual references, or style examples into a
  product/interface brief
- screenshots, bug reports, logs, or user complaints into engineering actions
- research evidence, interview notes, or case studies into a narrative or plan

These are **reference-translation** capabilities. Their candidate pool needs a
source ecology, not a stack of sources from the final output medium.

For visual/reference work, search at least these lanes:

- **Input/reference-domain interpretation:** how practitioners read the source
  worlds themselves, such as art, photography, architecture, interiors, posters,
  editorial design, fashion, packaging, sports/media identity, film, or product
  imagery.
- **Translation methods:** sources that show how observed qualities become new
  decisions: design-language extraction, style tiles, creative direction,
  case-study rationale, annotated redesigns, or critique.
- **Target-output constraints:** the platform/output realities that the final
  work must survive, such as web responsiveness, accessibility, motion safety,
  design systems, typography, or implementation mechanics.
- **Caller handoff language:** brief formats, vocabulary, do/don't lists,
  validation checks, and clarification patterns that make the result executable
  by another designer, engineer, or AI agent.

For an art-direction-to-web mixtape, CSS references, design-token specs, and
accessibility docs are valuable bridge sources. They are not substitutes for
visual-culture sources that teach the agent how to look, compare, interpret, and
translate. A candidate list made mostly of web-craft sources should include a
"Role gaps" note and keep searching for the missing input/reference domains.

**How to apply:** Before writing `working/02-candidate-longlist.md`, add a short
"Reference-translation source ecology" note to your own scratchpad or the
long-list: target output medium, input/reference domains represented, translation
method candidates, handoff candidates, and bridge/constraint candidates. If the
input/reference domains are thin, search those domains directly instead of doing
another generic target-output search. Do not stop at the ordinary 20–35 candidate
range if the ecology is still missing a source world the future agent must learn
to interpret.

---

## What not to do

- **Don't rely on generic search alone.** The first page of search results is what your audience could also find. The point of curation is to surface what they couldn't.
- **Don't accept candidates without verifying URLs.** AI hallucinates plausible-looking URLs. Web-fetch every URL before it goes on the long-list. A 404 candidate is worse than a missing one.
- **Don't filter at Phase 2.** That's Phase 4's job. If a source might be useful, it goes on the long-list. Quality decisions come after fetching and reading. A URL is a lead, not evidence that the content is any good.
- **Don't over-research one section while leaving others empty.** Walk the knowledge map. Each section should have a reasonable candidate pool before you go deep on any one section.
- **Don't let adjacency satisfy the JTBD.** A CSS reference is not an art-direction source just because the final artifact is HTML. A process article is not a writing source just because the writer needs a process. Label these as bridge candidates unless they directly teach the core action.
- **Don't collapse reference-translation work into the output medium.** If the future agent must translate paintings, photos, architecture, sports identity, or moodboards into a product direction, the candidate pool needs sources from those source worlds and from the translation move. Output-medium craft docs are necessary constraints, not the whole corpus.
- **Don't let "not absent" pass a required role.** If a required role is material to the future AI's job, thin coverage should trigger backfill or an improvement-pass recommendation.
- **Don't treat source roles as decorative.** A section can look populated while a required evidence role is absent. That is still a corpus gap.
