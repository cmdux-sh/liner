## Phase 4 — Evaluation

Read working/02-candidate-longlist.md. Then read the companion files `source-quality-hierarchy.md` and `curator-notes.md` in the skill bundle. Then fetch each candidate's content (transcripts for YouTube; readable body for web articles).

**Reused sources:** read a `liner://<slug>/<source-id>` candidate with `liner fetch <this-liner> <address>`, which saves the other liner's copy the way compile will use it. Don't fetch its original page instead: that copy is what this liner will get.

**Near-empty copies:** when `liner fetch` saves only a page's intro or a few lines, don't assume the page needs JavaScript. Check whether the missing text is in the page's HTML, for example with `curl -sL <url> | grep -c "<a phrase you expect>"`. If it is, the page is fine and Liner's extraction fell short: judge the source on the page itself, and note "near-empty copy, text is in the HTML" so the person can report it. Suggest `render: js` or `liner setup-js` only when the HTML really lacks the text.

**Bounded fetch discipline:** do not get stuck chasing unavailable content. Try one direct fetch/read per candidate. If that fails, make at most one recovery attempt (alternate transcript source, search result, mirror, or metadata page). After two failed retrieval attempts total, stop searching for that candidate and evaluate from verified metadata, the Phase 2 reason, and whatever partial content is available. Usually that means `dropped`. Do not mark a source `kept` or `trim` from title, URL, search snippet, crawler metadata, or model memory alone.

Every candidate still needs a decision in the YAML: keep / trim / drop against the JTBD. For kept and trimmed sources, write a rating (1–5), a `jtbd_fit` label, a primary `source_role` from the Capability Brief when available, fetch evidence, a content-quality judgment, and a curator note in the three-thing template (role / where the value is / limitations).

**Evidence contract for kept/trimmed sources:**
- `fetch_status: readable | partial`. Use `readable` when you retrieved the substantive article/transcript/PDF body. Use `partial` when you retrieved enough real source content to judge, such as an abstract plus substantial excerpt. If the page is `metadata_only` or `unavailable`, the decision must be `dropped`.
- `content_quality: high | medium`. If the readable content is shallow, generic, AI-slop, SEO filler, a homepage/gallery with no rationale, or otherwise low-value, mark it `dropped` instead of keeping it.
- `evidence:` at least two content-specific evidence bullets from the fetched/read source. These are not quotes; they are concrete claims, sections, examples, transcript moments, or methods you actually saw in the content. Search snippets, titles, and model memory do not count.

**JTBD-fit labels are required for kept and trimmed sources**:
- `direct` — teaches, demonstrates, or argues for the core action in the JTBD.
- `bridge` — helps execute the result after the core decision is made, such as implementation mechanics, tokens, layout, or process.
- `background` — broad context, vocabulary, or primer material. Background sources should rarely be `required`.

If a section contains only `bridge` or `background` sources, the evaluation must say it is weak coverage and either backfill a direct source or mark the section as a known gap. Do not let topic adjacency pass as JTBD fit.

**Source-role discipline**:
- Assign each kept/trim source to the strongest matching required source role. If it does not satisfy any required role, mark `source_role: supporting/background` and explain why it is still worth keeping.
- Do not count a homepage, gallery index, metadata page, or shallow listicle as satisfying a worked-example/case-study role unless the fetched content includes real rationale, process, analysis, or artifacts the future AI can learn from.
- For visual-output, taste, craft, or example-dependent capabilities, one or two examples is a warning sign, not a pass. Prefer several distinct worked examples with rationale over more primers.
- For `reference-translation` capabilities, direct fit means the source either teaches how to read/interpret the input reference domain or demonstrates a real translation from reference qualities into target-output decisions. Target-medium implementation mechanics, accessibility specs, token formats, and style-guide infrastructure are usually `bridge` fit unless they explicitly teach that translation move.

**Chunked decision discipline:** do not hold all decisions in conversation memory until the end. Create `working/evaluation-decisions/` and, after each knowledge-map section or every ~10 candidates, write/update a small YAML fragment there. Use filenames like `01-foundations.yaml`, `02-patterns.yaml`, etc. Each fragment uses the same `candidates:` list shape shown below and must include every candidate from that section/chunk. Liner can assemble `working/03-evaluation.yaml` from these fragments in the original longlist order.

**Artifact closure discipline:** write `working/03-evaluation.yaml` as soon as you have enough evidence to decide. Do not announce that you are "ready to write" before writing either the fragments or the final file. For dropped candidates, keep the entry compact: URL, title, decision, section if known, and one rationale sentence. Rating and note are required only for `kept` and `trim`. If the final YAML is long, prioritize complete fragments in `working/evaluation-decisions/`; Liner will assemble and validate the final YAML.

Write decisions to working/03-evaluation.yaml in the format:

```yaml
candidates:
  - url: https://...
    title: ...
    decision: kept   # kept | trim | dropped
    rating: 5
    jtbd_fit: direct # direct | bridge | background
    source_role: core-action method
    fetch_status: readable # readable | partial; metadata_only/unavailable must be dropped
    content_quality: high # high | medium; low must be dropped
    evidence:
      - The fetched source includes a specific method/example/argument, not just a title.
      - Another source-specific detail that proves the content was read.
    section: foundations
    rationale: One specific sentence.
    note: |
      Multi-line curator note for kept/trim sources. Use the three-thing template.
```

Stop after the fragments and/or final file are written. Do not draft synthesis yet.
