## Phase 6 — Synthesis

Read synthesis.md (replacing the placeholder), working/03-evaluation.yaml (the keep-list), and `synthesis-guidance.md` in the skill bundle.

Draft synthesis.md as continuous prose, 800–2000 words. The synthesis is the curator's *distilled understanding of the domain* — principles, contested questions, framework distinctions. NOT a recap of source content.

⚠ Bias correction: do NOT write "Source 1 says X; Source 2 says Y." Push toward the curator's framing. Synthesis is the first thing the consuming AI reads — it sets the lens.

**Required operating sections**:
- Include a `## Generative rules` section with about five imperative rules the consuming AI can apply directly. These are the "if you remember nothing else" rules, phrased as actions or constraints, not summaries. Example shape: "Channel discipline: stdout is for data, stderr is for humans, and exit codes mean what they say." A rule about how the agent works with the person, such as when it asks questions, must match the output contract in `working/01-jtbd-and-knowledge-map.md`: draft first with gaps marked, then a short list of follow-up questions. Don't write "ask before drafting" or "interview before drafting" unless that contract has a runtime boundary that requires it.
- Include a `## Stances this corpus takes` section naming the corpus's opinionated positions and trade-offs. This section should say what the mixtape leans toward, what it resists, and where a dissenting source should still temper the rule.

**Weight sources by their `kind`** (set during Phase 7, but visible in 03-evaluation.yaml too):
- `reference` sources (specs, RFCs) — substrate. Cite once for authority; don't lean on them for framing.
- `principle` sources (essays, manifestos) — load-bearing for the synthesis's stance section. These earn the most quoting.
- `prescription` sources (style guides, rules) — useful for concrete distinctions but rarely for the synthesis's voice.
- `example` sources (case studies, surveys) — taste calibration. Cite when arguing what "good" looks like.

If a source's `kind` field is absent, treat it as principle-equivalent (the synthesis-writing default).

**Cite as you write.** After a claim that rests on a source, add a Markdown footnote such as `[^apple]`. At the end of synthesis.md, define each one on its own line as `[^apple]: https://example.com/page`, using the exact bare `url` from the keep-list, or the `path` for a local file. Cite at least 3 kept sources, or all of them if there are fewer. Cite only kept sources. Citations mark the evidence, so keep writing prose, not a recap of sources.

For methodology mode, mark spots where the curator should personalize ("[curator's view: …]") so they have explicit handoff points. For quick mode, write a coherent draft they can lightly edit.

Stop after synthesis.md is written.
