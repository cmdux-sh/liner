# Liner website final language notes

The copy deck is ready for implementation. This edit used Arturo Writer, its bundled Humanizer reference, and the authorized current voice and style files. No private corpus was opened. No website source or Git state was changed by this editor.

## Scope

The pass covered the homepage, shared metadata and navigation, all ten complete docs pages, twelve release introductions, 47 curated release entries, social-card wording, and proposed clipboard, motion, search, and missing-page strings. The deck keeps its route markers, frontmatter fences, command components, file-tree components, and JSON dataset for implementation.

The original received draft is preserved at `/tmp/liner-copy-pre-humanizer.md`. The corrected substantive version immediately before Humanizer is preserved at `/tmp/liner-copy-substantive-pre-humanizer.md`. The final language diff is `/tmp/liner-final-language.diff`.

## Factual corrections

The factual reviewer accepted the substantive deck with six requested corrections. All six are resolved.

1. The CLI flags section now separates independent intentions into separate copy controls. Cache clearing is described as a separate command with confirmation.
2. Each export option has its own copy control. The text explains that another export to the same path replaces the archive.
3. Troubleshooting gives the launcher resolver order as `LINER_BIN`, repository virtual environment, bundled platform package, then `PATH`.
4. The temporary-workspace subsection and its troubleshooting link are removed. Changing `HOME` alone cannot promise isolated caches or AI-runner settings when explicit environment variables are inherited.
5. Adapter instructions distinguish inspection, preview, and authorized apply. Commands without `--yes` print the approval-required plan; `--yes` applies changes and prints a receipt afterward. Install, update, and removal examples follow that sequence.
6. Copyable Project inspection commands use the concrete `mobile-design-foundations` example. Their surrounding text names the working directory and tells readers to use their actual Project path.

These corrections were checked against the cited launcher, adapter action, cache confirmation, browser cache, and runner configuration code in `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/`.

## Final language stage

One complete Humanizer pass followed factual settlement. The edit made the homepage's evidence selection and reuse explanations more concrete, introduced synthesis and corpus for new readers, replaced decorative glossary labels with a reference table, and simplified maintenance explanations. Useful procedural repetition stays because readers can enter any docs page directly. Exact interface names and necessary limitations stay explicit.

The final fidelity read checked claim scope and command consequences against `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/docs/liner-website-truth-audit.md`. It preserves the distinction between local files and provider processing, separate Source and Synthesis approval, cached fetching, partial extraction, archive privacy limits, retained Source content, destructive purge, recoverable Project deletion, and the current instruction-regeneration limit.

## Checks

Comparison with the corrected substantive version confirms that all 21 fenced blocks, 53 command components, ten route addresses, and 31 absolute source bindings are unchanged by Humanizer. That includes exact YAML/frontmatter, file-tree syntax, and the entire release JSON dataset. The JSON parses as twelve versions with 47 entries. Each route has matching start and end markers.

The Arturo Writer prose linter reports one retained structural finding for 74 short procedural paragraphs across ten standalone pages. Each introduces a command, prerequisite, or separate action; joining them across copy controls or routes would obscure the instructions. It reports no other findings. Required code identifiers, metadata syntax, and stock generated strings remain exact.

## Implementation checks still needed

The deck defines proposed text. The website must implement successful and failed clipboard states based on the actual clipboard result, a working pause/resume control with reduced-motion behavior, production search and missing-page states, and the revised raster social card. Rendered layout, accessibility, copy controls, command execution, and published behavior remain the implementation owner's verification work. This prose edit makes no runtime or publication claim.
