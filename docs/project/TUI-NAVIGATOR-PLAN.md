# Terminal navigator plan

Goal: the terminal app becomes a live reader for your library. You talk to the
agent, and you read here. The reasons are in [the decision log](../decisions.md),
and the designs are in the Paper file "Liner — Terminal app redesign": pages
"1 · Navigator map" and "2 · Navigator screens".

## Screens

- **Library:** one table with views for All, Attention, Sources, and Connections.
- **Liner:** everything one liner holds.
- **Reader:** read any file and follow citations.
- **Settings:** library folder and connecting an agent, also used on first run.

`/` searches, `y` copies a prompt for your agent, and `?` shows keys, from anywhere.

## Core work first

Each of these is usable by agents too, so it comes before the app.

1. Structured citations. The synthesis step writes citations in a fixed format
   that names a Source, and a check enforces it. Older liners keep working and
   simply have no citations to follow.
   Done: Markdown footnotes, checked by the synthesis step, and read with
   `liner citations`.
2. Save `liner refresh` results, so "changed" labels survive the run, and clear
   them when a compile takes the new text in.
   Done.
3. A read-only library contract: what's waiting for approval and what's building,
   a Source index across liners with type, kind, and which liners use each one,
   and shared Sources. Done: `liner library`.

## App work

4. Library shell with the All view, peek, and live updates by polling that
   contract. Done.
5. Liner and Reader, with citation following and a back stack. Done.
6. Attention, Sources, and Connections views. Done.
7. Search and the `y` handoff, with prompts written by Core so they stay in one
   place. Done.
8. Settings and first run. A new person connects an agent here. Done, except
   changing the library folder, which still uses `LINER_DIR` or `projects_dir`.

## Removal, last

9. Remove the build, review, source, and maintenance screens, the 8 screens
   nothing opens, and the headless runner, once the navigator covers reading.
   Check what else uses `liner research build-*` and the provider settings. Done.
10. Update the docs and site. Done, with the site copy held in PR 43. `PRODUCT.md`,
    `DESIGN.md`, and the curation docs still describe the old app in places.

This ships in 1.2.0. About 10 PRs. The app can't lose building before step 8, so nobody is stranded
without a way to make a liner.

## Open questions

- How should the app tell someone who only ever used `npx linersh` that building
  moved to their agent? First run covers an empty library, but not an existing one.
- Should Settings change the library folder? The old first run could.
