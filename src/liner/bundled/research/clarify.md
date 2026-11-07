## Clarify Job

Help the person sharpen their capability goal so Liner can build a hyper-specific, source-grounded resource for a future AI agent. They should not have to name research lanes, source categories, or formal JTBD syntax. Ask only what is needed to infer those.

Ask the person 3-4 targeted questions that are specific to this capability, and wait for their answers. Do not answer for them and do not use a generic checklist. Prefer questions that clarify:

- what future AI sessions should produce, decide, critique, translate, or prevent
- the narrow niche or situation where the resource should be excellent
- domain constraints, risk boundaries, regulated contexts, or things the agent must avoid
- quality anchors only when the person already hinted at examples, people, products, or styles they care about
- what the future agent should do when a request is missing details. Recommend Liner's default: draft from what it has with each gap marked, then ask a few short follow-up questions. Offer stopping to ask first only for a real runtime boundary, such as a missing fact the answer can't be right without

If the goal involves images, moodboards, references, examples, inspiration, style, art direction, visual language, or translating one medium or domain into another output, include at least one question that clarifies the runtime inputs and output contract: what kinds of references the future agent may receive and what the caller needs back.

If the goal names multiple domains, ask how they relate in the capability. Do not ask "what sources should we gather" or "what should Liner extract"; Liner infers that from the capability.

Save the answers as a JSON array of `{"question": "...", "answer": "..."}` objects, then pass that file to `liner research submit --answers`.
