# Final message format (required)

When the phase finishes, end with a markdown message in **this exact shape**.
The TUI renders this — section headings (**Bold:**) become colored chips, bullets render with • glyphs, and bracketed paths render dim. Following the format gives the curator a scannable summary; freeform prose does not.

```
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
```

Rules:
- **Lead every bullet with a verb-phrase**, not a file path or URL. Put paths in `[brackets]` at the very end of the bullet.
- Don't introduce extra sections. The TUI styles only these three headings; anything else renders as plain text.
- Skip sections that don't apply. Better to omit "Mix:" than to fill it with filler.
- 2–4 bullets per section. Be specific, not exhaustive.
- For the opening line ("Done with Phase N.") use the actual phase number — not "this phase" or "the current phase".

**Review checklist — scope rules** (important):

The "Review checklist" is what the curator should do on the **current TUI done screen**, before pressing Enter to advance. The TUI's only affordances at that moment are: read your message, open the artifact file in `$EDITOR`, re-run the phase, or continue. So bullets must be answerable by ONE of:

- A yes/no judgment the curator can make by skimming what you wrote ("Does the knowledge map cover X?")
- A specific edit they can make to the artifact you just wrote ("Drop the InfoQ Q&A in the long-list — weakest entry")
- A specific source/section worth a closer look ("The Pinsker entry skews designer-voice — confirm or swap")

Do NOT include bullets that ask the curator to:
- Go find more sources (the next phase handles selection — flag it for *yourself* in a later phase note, not for them)
- Add categories or sections you didn't include (you should have included them)
- "Flag if you want a dedicated source for X" — that's outside the loop the TUI gives them
- Do open-ended research, validation, or thinking that requires leaving this screen
- Anything that doesn't fit "read → edit artifact in $EDITOR → advance"

If you don't have a concrete TUI-scoped checklist item, **omit the section**. An empty Review checklist is better than one full of asks the curator can't act on from where they are.
