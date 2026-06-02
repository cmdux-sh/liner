import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { PhaseId } from "../phases.js";
import type { Tape } from "../types.js";

/**
 * Companion files referenced by each phase per SKILL.md. The agent is told
 * which ones to read at the top of the relevant phase.
 */
const COMPANIONS: Partial<Record<PhaseId, string[]>> = {
  candidates: ["source-finding-tactics.md"],
  evaluation: ["source-quality-hierarchy.md", "curator-notes.md"],
  quality: ["quality-check-tests.md"],
  synthesis: ["synthesis-guidance.md"],
};

const PHASE_INSTRUCTIONS: Partial<Record<PhaseId, string>> = {
  framing: `## Phase 1 — Framing

Define the job-to-be-done and sketch the knowledge map.

- The JTBD is a *single specific sentence*, not a topic. The user already provided a draft in tape.yaml.jtbd and in working/01-jtbd-and-knowledge-map.md. Read both.
- **Read \`tape.yaml.jtbd_clarifications\` if present.** Those are the user's answers to sharpening questions about scope splits, expected output kinds, quality anchors, and audience model. Weight the knowledge map by them. For example: if the user said two domains are equally weighted, give them equal section counts; if the user named specific tools as quality anchors, build sections that match the conceptual coverage those tools cover.
- If the draft JTBD is genuinely a topic (e.g. "Mobile design") and the clarifications didn't sharpen it, push back in your output and ask the user to revise.
- The knowledge map is 4–8 sections, each with a few sub-areas. Treat it as a hypothesis to be revised during research. If two domains were named in the JTBD or clarifications, sections must visibly cover BOTH — uneven coverage is a framing failure.

**You MUST remove the placeholder line** \`TODO — Phase 1 replaces this with 4–8 sections…\` from working/01-jtbd-and-knowledge-map.md as part of writing the real knowledge map. That sentinel is how the TUI knows Phase 1 has been completed — leaving it in place will keep the hub's progress cursor stuck on Framing even after you write good content.

Write the result to working/01-jtbd-and-knowledge-map.md (overwriting placeholders including the sentinel above, keeping the JTBD line as-is). Stop after this file is written. Do not start Phase 2.`,

  candidates: `## Phase 2 — Candidate discovery

Read working/01-jtbd-and-knowledge-map.md for the JTBD and knowledge map. Then read the companion file \`source-finding-tactics.md\` in the skill bundle.

- Generate a long-list of candidate sources covering every section of the knowledge map. URLs and titles only — NO fetching yet.
- Aim wide: 2–4× the eventual kept count.
- **Verify each URL exists** via WebFetch (or your fetch tool) — AI hallucinates URLs that sound plausible. A 404 candidate is worse than a missing one.
- For each candidate, capture: URL, title, one-line reason it's a candidate.

**Search YouTube too, not just the web.** Liner fetches video transcripts and titles automatically (\`youtube.com/watch?v=…\` and \`youtu.be/…\` URLs are first-class sources). For most JTBDs there will be conference talks, lectures, or long-form interviews on YouTube that cover the topic from a different angle than written articles do. Look there explicitly. The quality hierarchy (in \`source-quality-hierarchy.md\`) treats conference talks from credible speakers as tier-3 sources — equal in standing to expert articles. Skip TED-style talks unless density is unusually high.

Write to working/02-candidate-longlist.md, grouped by knowledge-map section. Stop after this file is written.

⚠ Bias correction: do NOT default to popular/recent sources. Search deliberately for primary sources (specs, canonical papers, official docs, foundational books). Mix media types — articles, YouTube talks/lectures, papers, primary docs.`,

  evaluation: `## Phase 4 — Evaluation

Read working/02-candidate-longlist.md. Then read the companion files \`source-quality-hierarchy.md\` and \`curator-notes.md\` in the skill bundle. Then fetch each candidate's content (transcripts for YouTube; readable body for web articles).

For each candidate, decide keep / trim / drop against the JTBD. For kept and trimmed sources, write a rating (1–5) and a curator note in the three-thing template (role / where the value is / limitations).

Write decisions to working/03-evaluation.yaml in the format:

\`\`\`yaml
candidates:
  - url: https://...
    title: ...
    decision: kept   # kept | trim | dropped
    rating: 5
    section: foundations
    rationale: One specific sentence.
    note: |
      Multi-line curator note for kept/trim sources. Use the three-thing template.
\`\`\`

Stop after this file is written. Do not draft synthesis yet.`,

  quality: `## Phase 5 — Quality checks

Read working/03-evaluation.yaml. Then read \`quality-check-tests.md\` in the skill bundle — note especially the strengthened framing-gap rules below.

Run the five tests against the keep-list:
1. Redundancy — any two kept sources making essentially the same point?
2. Coverage — any knowledge-map section with zero sources?
3. Disagreement — strongest claim with a credible counter included?
4. Framing-gap — is any whole way of thinking about the JTBD missing?
5. Source-kind balance — distribution across the four kinds. Any kind at zero must be defended or backfilled.

**Framing-gap test rules (strict)**:

- Before looking at the keep-list, NAME 2–3 candidate perspectives that could exist outside the corpus (e.g. designer voice, accessibility-first viewpoint, security/abuse perspective, contrarian, adjacent discipline). Write them into working/04-quality-checks.md.
- For each named perspective, classify coverage in TWO dimensions:
  - **Stance** — is there a verified source whose author *argues for* this perspective as a design center of gravity? \`stance-represented\` or \`stance-absent\`.
  - **Concerns** — do any kept sources address the perspective's operational concerns, even from a different stance? \`concerns-addressed\` or \`concerns-absent\`.
  A perspective is fully covered only when its stance is represented. \`concerns-addressed\` alone is a SOFT GAP — the corpus internalizes the concern without giving the consuming AI a worked rival argument to cite.
- **Unverified sources do NOT count.** A source whose Phase 4 note acknowledges unfetched content, broken URLs, or unobtainable transcripts cannot satisfy a perspective slot. Tokenism is the failure mode this rule prevents.
- For each perspective that is not \`stance-represented\`, take exactly one of three actions:
  - **(a) Search for a stance source.** Vocabulary: who would *argue for* this position, what would they write, where would they publish? (e.g., "Brandur Leach on terminal density," not "CLI scriptability best practices.") One search pass; if a verified source fits, add it.
  - **(b) Argue concerns are sufficient.** Write the argument explicitly: "Concerns covered by [Sources N, M] inside the [dominant] stance. The dissenting stance is real but its operational implications are absorbed; no separate source needed." Not a free pass — if you can't articulate the trade-off, fall back to (a) or (c).
  - **(c) Flag the bias explicitly.** Document: "Perspective: [name] — stance unrepresented after one search pass. Corpus argues from [X] but no source argues for [Y]. Recommendation: ship + document, OR weaken the JTBD claim."

**Source-kind balance rules**:

- The four kinds are \`reference / principle / prescription / example\`. They represent four argumentative roles in the corpus.
- Count the distribution. A zero in any column is a decision, not an accident.
- 0 \`reference\` → no substrate authority. 0 \`principle\` → no philosophical anchor. 0 \`prescription\` → no concrete rules. 0 \`example\` → no taste calibration.
- For each zero, either backfill (one search pass for a source of that role) or write a defense: "Kinds audit: X/Y/Z/W. Accepted: [reason this JTBD legitimately doesn't need that kind]."

**Required output structure** in working/04-quality-checks.md:

\`\`\`
## Test 1 — Redundancy
[finding + action]

## Test 2 — Coverage
[finding + action]

## Test 3 — Disagreement
[finding + action]

## Test 4 — Framing-gap

### Perspectives audit
- [Perspective name] — stance-represented (Source N, verified) ✓
- [Perspective name] — stance-absent, concerns-absent. Backfill: searched [vocabulary], found [source] → now stance-represented (Source M) ✓
- [Perspective name] — stance-absent, concerns-addressed by [Sources L, K]. Argued sufficient: [explicit trade-off paragraph].
   OR
- [Perspective name] — stance-absent after one search pass. Corpus argues from [X], no source argues for [Y]. Recommendation: [...].

## Test 5 — Source-kind balance

Distribution: X reference / Y principle / Z prescription / W example
[If any zero: backfill action OR defense paragraph]
\`\`\`

Every named perspective must appear in the audit with explicit coverage status. A missing "Perspectives audit" section, or a Test 5 section without the distribution, means the test wasn't run honestly.

If a test fails, fix the keep-list (update working/03-evaluation.yaml) before reporting done. Even tests that pass without changes should be noted — silent passes train the model to treat the tests as ceremonial.

Stop after working/04-quality-checks.md is written and any necessary changes are reflected in 03-evaluation.yaml.`,

  synthesis: `## Phase 6 — Synthesis

Read synthesis.md (replacing the placeholder), working/03-evaluation.yaml (the keep-list), and \`synthesis-guidance.md\` in the skill bundle.

Draft synthesis.md as continuous prose, 800–2000 words. The synthesis is the curator's *distilled understanding of the domain* — principles, contested questions, framework distinctions. NOT a recap of source content.

⚠ Bias correction: do NOT write "Source 1 says X; Source 2 says Y." Push toward the curator's framing. Synthesis is the first thing the consuming AI reads — it sets the lens.

**Weight sources by their \`kind\`** (set during Phase 7, but visible in 03-evaluation.yaml too):
- \`reference\` sources (specs, RFCs) — substrate. Cite once for authority; don't lean on them for framing.
- \`principle\` sources (essays, manifestos) — load-bearing for the synthesis's stance section. These earn the most quoting.
- \`prescription\` sources (style guides, rules) — useful for concrete distinctions but rarely for the synthesis's voice.
- \`example\` sources (case studies, surveys) — taste calibration. Cite when arguing what "good" looks like.

If a source's \`kind\` field is absent, treat it as principle-equivalent (the synthesis-writing default).

For methodology mode, mark spots where the curator should personalize ("[curator's view: …]") so they have explicit handoff points. For quick mode, write a coherent draft they can lightly edit.

Stop after synthesis.md is written.`,

  assembly: `## Phase 7 — Assembly (draft)

You are preparing a *draft* of the sources list for the curator to review. **DO NOT touch tape.yaml directly.** Write your proposal to \`working/07-tape-draft.yaml\` and stop.

Read these inputs:
- \`working/03-evaluation.yaml\` — the keep-list with curator notes and ratings.
- \`synthesis.md\` — the curator's framing, to inform section ordering.
- \`tape.yaml\` — preserve any existing source order the curator already chose; you're proposing additions/changes, not erasing.

Build the draft as YAML with a top-level \`sources:\` list. Each entry needs:

\`\`\`yaml
sources:
  - type: web         # web | youtube | local_file
    url: https://...
    section: foundations
    priority: required   # required | optional
    kind: prescription   # reference | principle | prescription | example
    note: |
      One-paragraph curator note: role of this source, where the value lives,
      and any limitations. Use the three-thing template.
\`\`\`

Rules:
- Only include candidates marked \`kept\` or \`trim\` in 03-evaluation.yaml. Drop everything else.
- Within a section, order by rating (highest first).
- Order sections in reading order: foundations / patterns / applications / craft (or whatever the synthesis suggests).
- Notes come straight from 03-evaluation.yaml's \`note\` field. Do not paraphrase.
- Mark a source \`priority: optional\` only when the methodology genuinely treats it as skippable (rare).
- \`local_file\` entries: only include them if 03-evaluation.yaml already has them. Never invent paths.

**Source \`kind\` is required on every entry.** It tells the synthesis prompt and the consuming AI which sources to weight how. Pick the best fit:

- \`reference\` — specs / standards cited but not deep-read (POSIX, RFCs, official manual pages). The consuming AI cites these for authority but doesn't quote them at length.
- \`principle\` — read for stance / framing (essays, manifestos, philosophical pieces). Shapes how the AI thinks about the domain.
- \`prescription\` — read for concrete rules (style guides, conventions, explicit do/don't lists). The AI quotes these directly when applying rules.
- \`example\` — read for taste / illustration (case studies, tool surveys, worked examples). Shows what good looks like.

A source can plausibly fit two categories; pick the one closest to *how the consuming AI should USE this source*. When in doubt: rules → \`prescription\`; arguments → \`principle\`; examples → \`example\`; authority without narrative → \`reference\`.

When done, write \`working/07-tape-draft.yaml\` and stop. The TUI will show the curator a review screen to accept, edit, or discard. DO NOT modify tape.yaml.`,
};

export type PromptBuildArgs = {
  phaseId: PhaseId;
  project: string;
  skillPath: string;
  tape: Tape;
};

/** Construct the system+user prompt blob passed to the agent. */
export function buildPhasePrompt(args: PromptBuildArgs): string {
  const { phaseId, project, skillPath, tape } = args;
  const skillContents = safeRead(join(skillPath, "SKILL.md")) ??
    "(SKILL.md not found at " + skillPath + ")";
  const phaseInstruction =
    PHASE_INSTRUCTIONS[phaseId] ??
    `## Phase ${phaseId}\n\nNo dedicated instruction. Use SKILL.md to figure out what to do for the ${phaseId} phase, then stop.`;

  const companionNote = (COMPANIONS[phaseId] ?? [])
    .map((f) => `  - ${join(skillPath, f)}`)
    .join("\n");

  return `You are running the curating-mixtapes methodology on behalf of a curator.

# Mixtape context

- Project folder: ${project}
- Title: ${tape.title || "(not set)"}
- Description: ${tape.description || "(not set)"}
- Mode: ${tape.mode ?? "quick"}
- JTBD: ${tape.jtbd || "(not set)"}
- Curator: ${tape.curator || "(not set)"}
- Sources currently on tape: ${tape.sources.length}

# Skill bundle

The methodology is documented in:
  - ${join(skillPath, "SKILL.md")}

Companion files (read the ones listed in the phase instructions below):
${companionNote || "  (none for this phase)"}

# What to do

${phaseInstruction}

# Output discipline

- Work on the filesystem. Do not paste large file contents back in your response — write them to disk.
- Be concise in your spoken output. The user is watching a TUI; long monologues clutter the screen.
- If you cannot proceed (missing JTBD, missing prior-phase artifact), say so clearly and stop. Do NOT make up content.

# Final message format (required)

When the phase finishes, end with a markdown message in **this exact shape**.
The TUI renders this — section headings (**Bold:**) become colored chips, bullets render with • glyphs, and bracketed paths render dim. Following the format gives the curator a scannable summary; freeform prose does not.

\`\`\`
Done with Phase N.

**What I did:**
- Verb-first sentence describing one concrete output [path/to/file]
- Another verb-first sentence [path/to/another-file]

**Mix:** (only if you discovered or surveyed sources — skip otherwise)
- Category one: short list of representative items
- Category two: short list

**Review checklist:** (only if there's something the curator should verify in *this TUI screen* before moving on — skip if not)
- A specific judgment call to make from what's visible here
- Another specific judgment call
\`\`\`

Rules:
- **Lead every bullet with a verb-phrase**, not a file path or URL. Put paths in \`[brackets]\` at the very end of the bullet.
- Don't introduce extra sections. The TUI styles only these three headings; anything else renders as plain text.
- Skip sections that don't apply. Better to omit "Mix:" than to fill it with filler.
- 2–4 bullets per section. Be specific, not exhaustive.
- For the opening line ("Done with Phase N.") use the actual phase number — not "this phase" or "the current phase".

**Review checklist — scope rules** (important):

The "Review checklist" is what the curator should do on the **current TUI done screen**, before pressing Enter to advance. The TUI's only affordances at that moment are: read your message, open the artifact file in \`$EDITOR\`, re-run the phase, or continue. So bullets must be answerable by ONE of:

- A yes/no judgment the curator can make by skimming what you wrote ("Does the knowledge map cover X?")
- A specific edit they can make to the artifact you just wrote ("Drop the InfoQ Q&A in the long-list — weakest entry")
- A specific source/section worth a closer look ("The Pinsker entry skews designer-voice — confirm or swap")

Do NOT include bullets that ask the curator to:
- Go find more sources (the next phase handles selection — flag it for *yourself* in a later phase note, not for them)
- Add categories or sections you didn't include (you should have included them)
- "Flag if you want a dedicated source for X" — that's outside the loop the TUI gives them
- Do open-ended research, validation, or thinking that requires leaving this screen
- Anything that doesn't fit "read → edit artifact in \$EDITOR → advance"

If you don't have a concrete TUI-scoped checklist item, **omit the section**. An empty Review checklist is better than one full of asks the curator can't act on from where they are.

# Full SKILL.md follows

${skillContents}
`;
}

function safeRead(path: string): string | null {
  try {
    return readFileSync(path, "utf8");
  } catch {
    return null;
  }
}
