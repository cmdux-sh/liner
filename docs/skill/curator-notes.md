# Curator Notes

Use this before Evaluation.

Curator notes are the routing layer for the consuming AI. A note should tell the AI when to use a source, what part is valuable, and what limitation to remember.

## Good Note Shape

Answer three questions:

1. **Role:** Why is this source in the mixtape?
2. **Value:** What specific part should the AI pay attention to?
3. **Limit:** What should the AI not overgeneralize from this source?

Good:

```text
Canonical platform reference for iOS gesture behavior. Use for system-level expectations around swipe, tap, and long-press interactions; do not treat it as visual-style guidance.
```

Weak:

```text
Useful article about gestures.
```

## Rules

- Do not write generic praise.
- Do not summarize the entire source.
- Do not make the note longer than the source entry needs.
- If a source only matters for one section, say that.
- If a source is included despite a weakness, name the weakness.
