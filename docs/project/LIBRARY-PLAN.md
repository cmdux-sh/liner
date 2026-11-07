# Library plan

Goal: any agent can find, read, and grow a personal library of liners without
opening the terminal app. A **liner** is a Liner Project: one folder, one narrow
job, reusable across sessions and projects. The reasons for this direction are
in [the decision log](../decisions.md).

## Where things stand

- Core is Python. It owns Projects, Sources, compile, and maintenance.
- Core owns the research steps: order, checks, prompts, and Evaluation chunks.
  The runner and the terminal app run the AI calls and show progress. The app's
  step labels are pinned to Core by `build_order_contract_test.go`.
- `liner research` runs every step for an agent, with approvals. The terminal app
  runs the same steps as one continuous build through `liner research build-*`.
  That build is being removed: see the [terminal navigator plan](TUI-NAVIGATOR-PLAN.md).

## Stage 1: Usable from any agent

| Work | Outcome |
| --- | --- |
| Fix Claude discovery | The Claude runner can search the web in the phases that find sources. |
| Fix local repo import | A folder with no `SKILL.md` is refused at import, not at compile. |
| `liner catalog` | Lists every liner in the library with its job, readiness, and freshness. |
| `liner search` | Returns cited passages from one liner or the whole library. |
| `liner agent-skill` | Installs an optional `liner` skill whose description names your subjects. |

Live check, 2026-10-30, Claude Code 2.1.304 on macOS: the candidates phase ran 18
searches and 30 fetches through the Liner runner, and a clean session found the
`liner` skill unprompted, asked first, then loaded and searched the liner.

Protocol run, 2026-10-30, Claude Sonnet 5.5 with only `liner` and web tools: a
clean session built a liner from a job story, about 4 minutes and $0.71, with 5
compiled sources and a usable catalog entry. Its approvals were given in advance
in the test prompt.

Final Stage 2 run, 2026-10-30, Claude Sonnet 5.5 with `liner`, web search, and the
subagent, without `WebFetch`: a clean session built a liner in about 2 minutes for
$0.86, with 9 pages saved by `liner fetch`, 3 subagent reads, 6 compiled sources,
and a usable catalog entry. Approvals were given in advance in the test prompt.

## Stage 2: Research protocol (done)

Core gains the steps, and the TUI becomes a client of them:

```sh
liner research start "<job story>"
liner research next <slug> --json
liner research submit <slug>
```

1. Move prompts and phase checks into files Core ships, keeping one copy. The
   prompts now live in `src/liner/bundled/research/`, and Core builds them. The checks are in
   `src/liner/research_checks.py` and `research_evaluation.py`, and the runner
   calls them.
2. Move the step order, Clarify Job, and Operating Layer writing into Core.
   The first Operating Layer is the `operating_layer.create` Change Set, and the
   TUI uses it. Go no longer renders or writes those files. `liner research start`, `next`,
   `submit`, and `approve` carry the step order and approvals, and Clarify Job is
   a `person` step the agent asks.
3. Point the Go TUI and Node runner at the Core commands.
4. Ship a restricted research subagent and a `liner fetch` that writes fetched
   pages to a file, so untrusted text does not land in a full-powered session.
   The Claude Code subagent has only `Read` and `Grep`: the main agent fetches
   and the subagent reads, so a page can't direct a shell or a network call. This
   reduces exposure but is not isolation: a hostile page could still get it to
   read a local file and quote it to the main agent, which has a shell. Only its
   prompt tells it not to. Other
   harnesses use `liner fetch` alone.
5. Enforce approval in the protocol: Core rejects unapproved changes, and the
   person approves sources and synthesis through their agent. The terminal app
   accepts its gates automatically and keeps doing so for now. Approval ties to
   the exact content, but Core cannot tell who ran `liner research approve`, so
   it depends on the agent asking.

## Stage 3: Compounding

- `liner use <slug>` adds a pointer to a repo's `CLAUDE.md` or `AGENTS.md`. Done.
- A gap log records what a liner could not answer, and the next research pass
  targets those gaps. Done.
- A findings inbox holds answers from real work for the person to review. An
  accepted finding becomes a local Source, so it follows the normal refresh. Done.
- `liner refresh` refetches sources and reports what changed online. Done.
- EPUB and repository Sources. Done.
- Liners citing other liners. Done.
- A with-and-without test whose result shows in the catalog. Done.

## Stage 4: Reach

- `liner mcp`, a thin stdio wrapper over the same commands, for agents with no
  shell. Done.
- Rewrite the README and site around the library, then release through the
  publish checklist. The README is done. The site rewrite and version bump wait
  on branch `release-1.2.0`, because merging them builds the site, and 1.2.0 now
  includes the terminal navigator.

## Checked before release

- A built macOS arm64 package, run with its bundled Core: research commands,
  prompts, FTS5 search, and subagent and skill install all worked.
- The real terminal binary with that Core and a scripted agent ran New Project,
  Clarify Job, all six phases, source acceptance, a real compile, and the
  Operating Layer to Project Complete.
- `liner research build-*` takes about 0.2 seconds in that package at a phase
  boundary, so those calls stay on the app's update loop.

## Open questions

- Does Codex discovery reach live search under `-s workspace-write`?
- Do the Linux and Windows release binaries include SQLite FTS5? `liner search`
  needs it, and a local macOS arm64 package build passed. Check the others at
  the next release build.
- Should the `candidates` phase lose write access beyond its own artifact? It can
  now read search results, and like `WebFetch` it runs with permission checks
  skipped.
- Which of Antigravity's extension points fits best: shell plus skill, or MCP?
