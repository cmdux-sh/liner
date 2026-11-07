# Changelog

## 1.2.1

- **A mistyped command says so.** An unknown first argument, such as
  `liner compil`, used to open the navigator. It now exits 2 with the closest
  command, and `--help`, `-h` or `help` anywhere shows the launcher's help.
  `liner --help` describes the navigator.
- **The navigator's `?` page lists every key**, including movement, paging,
  `shift+tab`, `ctrl+c`, and when `q` goes back or quits.
- **`scripts/uninstall.sh --purge-library` deletes your liners safely.** It
  finds the library the way Liner does, deletes only liner folders, and asks you
  to type how many first. `--purge-workspace` still works as an older name.
- **The version line uses one space around the dot**, matching the docs.

## 1.2.0

- **Pages with non-ASCII response headers load.** A site that sent an emoji in a
  header, such as WordPress's developer blog, failed every fetch with an encoding
  error.
- **Pages with a Markdown copy are read from it.** A page that declares
  `<link rel="alternate" type="text/markdown">` on the same site, as Apple's Human
  Interface Guidelines do, is read from that copy instead of needing a browser.
  Markdown and plain-text responses are kept as text.
- **Near-empty pages are checked before a browser is suggested.** When a saved copy
  holds only a page's intro, the evaluation step checks the page's HTML for the
  missing text and suggests JavaScript rendering only when it isn't there.
- **Blind tests stay blind.** `liner test start` refuses answers that hold links,
  footnotes, a source list, or a mention of the liner, since only the answer written
  with the liner would have them. The `liner` skill tells the agent to leave them out.
- **Evaluation reads a reused source's saved copy.** `liner fetch <slug>
  liner://<other-liner>/<source-id>` saves the copy compile will use, and
  `liner find-sources` lists reusable copies first.
- **Extraction fixes reach cached pages.** Pages in the shared cache record which
  extractor saved them, and ones saved by an older extractor are fetched again.
- **Icon-only table cells keep their label.** A cell holding only a check-mark icon,
  such as in Apple's App Store Connect tables, now reads as its label.
- **Pages that stop at their intro are read again.** When the looser setting also
  keeps only a page's intro, Liner tries trafilatura's fallback extractors on their
  own. Apple's product page guide used to save 385
  characters; it now saves 10,464, including the promotional text section.
- **A new liner's agent drafts first by default.** When a build asks what the agent
  should do with a request that's missing details, it now recommends drafting with
  the gaps marked and asking after, instead of asking first.
- **Builds keep the chat small.** `liner research next` and `submit` save each step's
  prompt to a file and return its path, and approval reviews point to files too, so
  long instructions and drafts stay out of the chat. Add `--with-prompt` if your
  client can't read files; the MCP server does this for you.
- **A build plans its subagents.** `liner research plan <slug>` lists the steps ahead
  and the kind of work each needs, such as `find`, `judge`, or `synthesize`. The
  `liner` skill shows it as a plan of subagents with suggested models, lets you change
  it, and runs the research in subagents while approvals stay in your chat.
- **You can see what research went through.** The sources step now keeps a search
  log, and a finished build and `liner status` show the funnel: searches and results
  as the agent reported them, then candidates, pages read, evaluated, kept, and final
  sources.
- **Reuse one source from another liner.** A `liner` source can now be
  `liner://<slug>/<source-id>`, which copies that source's saved text, even a local
  file, with its original address, so a citation of the original opens it.
  `liner find-sources` gives each source's address. It still goes through evaluation
  and your source approval, `liner share` warns before packing one that was a local
  file, and `liner refresh` says where its live page gets checked.
- **A new liner no longer says it's a starter tape.** `liner research start` used to keep
  the sample tape's description, "A starter tape with one YouTube video and one
  article", and its `example` and `starter` tags, and `MIXTAPE.md` opened with that line.
  New liners drop them, and compiling leaves a stand-in description out of `MIXTAPE.md`,
  so recompiling an older liner fixes it too.
- **Wrapped approvals still ask you.** `liner research guard install` also adds a
  Claude Code hook that reads the whole command, so an approval run inside `bash -c`,
  after `env`, through `npx`, or with a full path to `liner` asks you too. Run it again
  to add the hook to an existing install.
- **Builds ask about Playwright before they need it.** When a source renders with
  JavaScript and Playwright isn't installed, the sources approval lists it and tells the
  agent to ask, as its own question, whether to run `liner setup-js --yes`. The build
  step stops with that command instead of failing partway through the compile.
- **A finished build updates the `liner` skill's subject list.** Agents that have the
  skill installed list the new liner right away, without `liner agent-skill update`.
- **The research plan names the approvals in plain words:** the synthesis, the source
  list, and the instructions, in the order they come.
- **Reference pages save their full text.** When the page extractor keeps under a
  quarter of a page's visible words, Liner tries again with a looser setting. Apple's
  App Review Guidelines used to save 91 words of about 15,000; they now save 12,320.
- **A finished liner's synthesis can be revised on its own too.** A `synthesis.review`
  patch on a finished liner needs your approval, then starts a refresh: `liner compile`
  rebuilds `MIXTAPE.md`, and you review the instructions against the new synthesis.
- **Approvals point out rules that ask before drafting.** The synthesis and instructions
  approvals list any line that holds a draft back for questions, such as "ask before
  drafting", so you can fix it or keep it on purpose. New builds stop writing those
  rules unless a runtime boundary needs one.
- **A finished liner's instructions can be revised on their own.** An
  `operating_layer.review` patch no longer needs a source change first. It still
  needs your approval, and it's refused while a refresh is waiting for review.
- **Drafts come with their follow-up questions.** New liners tell agents to draft
  from what you gave, mark each gap, and then ask a short set of questions that say
  which gap each one fills.
- **Research checks your library first.** `liner find-sources` lists sources across
  your liners with earlier verdicts, including rejected ones, so agents don't repeat
  work. The output is capped, so it stays small as the library grows.
- **A build can't finish with sources missing.** `liner research submit` now stops
  and names any source that didn't compile, instead of reporting success.
- **Saved pages are reused.** `liner fetch` keeps a full page or transcript in the
  shared cache, so another fetch or compile, in any liner, doesn't download it again.
- **`liner fetch` reads YouTube transcripts.** It used to save only the video page's
  footer, which made good videos look empty during research. It now also warns
  when any page saves almost nothing, instead of reporting a plain success.
- **Building a liner starts faster.** `liner agent-skill install claude --yes` now
  also adds the page-reading subagent and the approval rules, so a build never stops
  for setup. Add `--no-subagent` or `--no-guard` to leave either out. Asking your
  agent for a liner counts as the go-ahead, and it asks for the job in plain words.
- **`liner test` can be blind.** `liner test start` saves both answers as A and B in
  random order, and `liner test judge` records your choice and then reveals which
  used the liner. `liner catalog` shows how many tests were blind.
- **Unused options are gone.** The `[output] include_preamble` setting never did
  anything and has been removed. `mode` remains on tapes as a label that changes
  nothing, and `--mode` is hidden.
- **`SECURITY.md` says what Liner does and doesn't protect against**, and the
  README shows how to connect an agent to `liner mcp`.
- **Claude Code can ask you before an agent approves anything.**
  `liner research guard install --yes` makes Claude Code prompt you directly before
  it runs `liner research approve` and the other commands that need your say-so.
  Approving stays in chat. It's a guardrail, not a lock: bypass mode skips it, and
  so does the same command written another way. Codex has no equivalent yet.
- **Finishing a liner can't get stuck on an existing `LINER.md`.** A build that
  stopped just after writing the Operating Layer now finishes on the next try. A
  different `LINER.md` stops with a clear message instead of asking for approval
  again and again, and the approval now always covers `LINER.md` and `SKILL.md`.
- **Imported liners are handled as someone else's work.** `liner import` no longer
  overwrites an existing folder unless you pass `--overwrite`, refuses unsafe paths
  and oversized archives, and drops agent transcripts, which `liner share` also
  stops packing. Local `skill` sources must now sit inside the liner, in an
  installed skills folder, or in a folder listed in `LINER_SKILL_PATHS`. A skill
  path anywhere else used to work, and now stops with a message saying how to allow it.
  A source only an agent could read, such as a bot-walled page, needs its text saved
  as a `local_file` before someone else can recompile the liner.
- **Fetching stays on the public internet.** A tape can come from a stranger, so
  Liner now refuses URLs that lead to this machine, your local network, or a cloud
  metadata address, including through redirects. To fetch from an intranet, set
  `allow_private_network = true` under `[fetch]` in `~/.liner/config.toml`. Web
  pages and PDFs now stop at 50 MB, and remote skill files at 2 MB.
- **Building moved to your agent.** The app's build, review, source, and
  maintenance screens are gone, along with its AI runner. `liner research improve`
  now shows an agent how to strengthen a finished liner and its open gaps.
- **The terminal app is now a navigator for your library.** It shows every liner,
  what needs you, your whole source collection, and how liners connect, and it
  updates by itself as your agent works. You can read any file and follow a
  citation to its source, and `y` copies a prompt for your agent. Building and
  approving now happen through your agent.
- **New commands work through npm.** The launcher opened the terminal app for
  `liner catalog`, `research`, `search`, and the other library commands, so an
  agent would hang. It now passes them to Core, and a test keeps the list complete.
- **`liner refresh` keeps its last report, and `liner library` shows it all.**
  `liner library` reports every liner, what needs attention, your whole source
  collection, and how liners connect, for the terminal app to read.
- **Syntheses cite their sources.** New syntheses mark claims with Markdown
  footnotes, and the synthesis check confirms each one points to a kept source.
  `liner citations` lists them and their source files. Older liners keep working.
- **`liner mcp` reaches agents with no shell.** It offers your catalog, search,
  file reading, gaps, findings, `liner fetch`, `liner refresh`, and the research
  steps over standard input and output. It runs the same commands as the shell.
  Approving anything stays with you in a terminal: it has no tool for approving
  research, accepting findings, recording a test verdict, or applying a change.
- **`liner test` records whether a liner earned its place.** An agent answers a
  real question with and without the liner, you judge the two, and
  `liner test add` keeps the result. `liner catalog` shows how many tests came
  out better and whether the liner has changed since. As with
  `liner research approve`, Liner trusts the agent to report your verdict.
- **Liners can cite each other.** A `liner` Source, `liner://<slug>`, compiles
  another finished liner's reviewed synthesis into this one. `liner catalog`
  shows `cites` and `cited_by`, and `liner refresh` reports when a cited liner's
  synthesis has changed. Citations go by folder name, so moving a liner breaks
  them, and a shared liner can't recompile them outside your library.
- **Books and repositories can be Sources.** A `local_file` can now be an EPUB,
  read in chapter order. A `repository` Source takes a public GitHub URL and
  compiles its documentation files at one commit, or just one folder with a
  `/tree/<ref>/<folder>` URL. Code files and private repositories aren't read.
- **`liner refresh` shows what changed online.** It refetches a liner's web pages,
  videos, and remote skills, compares them with the compiled copies, and saves a
  diff for each change. It changes nothing in the liner, and it reports pages it
  couldn't read instead of treating them as changed.
- **Findings from real work wait for your review.** `liner findings add` puts
  what an agent learned in a liner's inbox. Accepting one adds it as a Source,
  which starts the usual synthesis, compile, and Operating Layer review. It never
  edits `LINER.md` directly. `liner catalog` counts what's waiting. As with
  `liner research approve`, Liner trusts the agent to ask you before `--yes`.
- **Liners remember what they couldn't answer.** `liner gaps add` records a
  question a liner failed on, `liner catalog` counts the open ones, and the next
  Improve Corpus pass is told to look for sources that answer them.
- **`liner use` points a repository at a liner.** It adds a short block to
  `CLAUDE.md` or `AGENTS.md` naming the liner and how to search it, previews it
  first, and can remove it. The block names the liner by slug, so nothing from
  your machine lands in a committed file.
- **Read web pages in a restricted session.** `liner research subagent install`
  adds an optional Claude Code subagent with no shell, network, or write tools,
  which reads pages saved by `liner fetch` and reports what they say. This
  reduces exposure but is not isolation: it can still read local files, and its
  report goes to the main agent.
- **Creating the Operating Layer now leaves a receipt.** The terminal app used to
  write `LINER.md`, `SKILL.md`, and `liner.yaml` one at a time. Core now writes
  them together after you confirm, and an older project gets its missing IDs in
  the same change.
- **`liner fetch` keeps web pages out of the agent's session.** It saves a page
  to a file in the liner and returns the path, so an agent opens only what it
  needs.
- **Any agent can build a liner.** `liner research start`, `next`, `submit`, and
  `approve` walk an agent through every step with its own tools, without the
  terminal app. Liner checks each result, and the person approves the synthesis,
  the sources, and the Operating Layer through their agent. Liner trusts the
  agent to ask: it cannot tell who ran `liner research approve`. An edit during
  the build reopens that approval. After a liner is finished, changes go through
  maintenance.
- **Liner no longer reads a logged-in Claude as logged out on macOS.** Liner set
  `CLAUDE_CONFIG_DIR` even to the default `~/.claude`, and Claude Code then
  couldn't find a claude.ai login, so every Claude phase stopped before it
  started. Liner now leaves the variable unset for the default home.
- **Claude can search the web while finding sources.** The candidates and
  improvement phases told the agent to search, but the Claude runner blocked
  `WebSearch`, so discovery was limited to URLs the model already knew. Both
  phases can now search. Every other phase still cannot.
- **A folder with no `SKILL.md` is refused when you add it.** The source inbox
  used to accept any local folder as a skill, and the compile failed later. It
  now skips the folder and says why.
- **`liner catalog` lists your whole library.** It reads every liner in your
  Project library, wherever your agent is working, and reports each job,
  readiness, staleness, and source count. A liner it cannot read is reported,
  not dropped.
- **A `liner` skill lets your agent find your research.** `liner agent-skill
  install claude --yes` adds an optional skill that names your usable liners, so
  your agent can suggest one that fits the task and ask before loading it.
  `liner agent-skill update` refreshes the list after your library changes.
- **`liner search` finds passages across your liners.** It searches syntheses
  and compiled sources with SQLite full-text search and returns the file, line,
  heading, and source URL. It writes nothing and needs no embeddings.

## 1.1.3

- **`liner` no longer starts itself in a loop when its platform package is
  missing.** If npm skipped the bundled `linersh-<platform>-<arch>` package (an
  unsupported platform, optional packages turned off, or an install in the
  minutes after a release), the launcher fell back to the first `liner` on
  `PATH`, which was its own npm symlink, and kept starting new copies. It now
  follows symlinks, never hands off to another Liner launcher, lists every
  `liner` on `PATH` instead of only the first, and stops with a clear message
  when no Liner CLI is available.

## 1.1.2

- **Repair no longer installs the browser.** On the Compile Console, `r` and
  Enter run only the recovery steps that cost nothing: retrying saved custom
  Sources, or retrying the compile for failures that do not need a browser.
  Sources that need a browser are counted, and `i` remains the only key that
  installs JS rendering, with its measured size. Keys are labeled with what
  they do, such as `retry compile`.
- **`liner setup-js` installs only the browser Liner launches.** Liner runs
  Chromium headless, so setup now downloads Playwright's headless shell
  (`playwright install --only-shell chromium`): about 100 MB to download and
  about 220 MB on disk, down from about 290 MB and 575 MB. Every size claim now
  states the measured figure instead of "about 150 MB". Liner now requires
  Playwright 1.49 or later, and setup upgrades an older one first, so the full
  Chromium build is never downloaded.
- **Review Draft Sources shows the agents' notes.** Attached Sources show the
  note the draft gives them instead of "no note". The selected Source's full
  note wraps on screen beside the accept panel at 100 × 30, and a Read column
  shows how much of each Source the agents could read (the evaluation's
  `fetch_status`). Attached Sources also show the draft's section and kind
  instead of "unsectioned"; the review says when those come from the draft,
  and accepting still keeps the saved Source as it is. A note too long for the
  screen scrolls with PgUp/PgDn instead of being cut off.
- **The synthesis and Project Skill editors accept new lines at any length.**
  Enter no longer stops once the text is as long as the visible editor. While
  you edit, the editor fills the screen and keeps the cursor line visible.
  After Ctrl+D, the review keeps the decision and the approve action on screen
  at 100 × 30, and Core's approval preview uses the full height for the
  replacement text.
- **Continue without failed Sources from where you are.** Create Operating
  Layer (`l`) on the Project screen shows the failed Sources instead of
  refusing. Once you have reviewed them, `l` creates the Operating Layer from
  the usable Sources, and every key on that path is listed. That shortcut
  still stops when no Source is usable, and a new compile or Source change
  asks for the review again. The compile docs no longer say repair installs
  the browser.
- **The improvement offer follows the latest status.** An applied improvement
  pass is no longer offered again because an older quality-check file still
  recommended it. When the offer does appear, it states the gaps in plain words
  instead of a raw heading.
- **Sol 6.1 and Sonnet 5.5 are the defaults, both at Medium effort.** OpenAI
  Auto runs Sol 6.1 at Medium for every task, and Claude runs Sonnet 5.5 at
  Medium. Settings offers Sol 6.1 for OpenAI and Sonnet 5.5 and Opus 5.5 for
  Claude, and Claude now has its own Thinking effort setting (Low through
  Maximum), passed to Claude Code as `--effort`. A saved GPT-5.6 model keeps
  working as a custom model ID, and a saved Sonnet or Opus choice moves to the
  5.5 model.

## 1.1.1

- **Resumed interrupted corpus work from the last durable checkpoint.** If an
  agent run stops or a generated artifact fails validation, reopening the
  Project restores the correct review or build action. Liner no longer treats
  partial work as complete or sends the curator back to the beginning of the
  corpus workflow.
- **Preserved approved Sources through Assembly.** User-provided Sources keep
  their accepted identity when the assembled corpus adds richer evaluation
  metadata, and pending Assembly is now the primary action when it is the real
  next step.
- **Made focused corpus improvements durable and visible.** Improve Corpus
  decisions survive restart, an applied improvement does not reopen the same
  recommendation, and the human-readable operating-fit audit now records
  `improvement_applied` while retaining the original gap as history.
- **Kept transition and deletion flows focused.** Source and Synthesis
  transitions show ordinary confirmation messages instead of internal result
  payloads. Recoverable Project deletion now keeps focus and Enter behavior on
  the exact confirmation step.

## 1.1.0

- **Prepared the maintenance stack as the Liner 1.1.0 release candidate.**
  Python, npm, platform packages, Go binaries, installed version output, and
  public version fields now agree on `1.1.0`. Release preparation fails closed
  if any candidate package version already exists on npm, and CI now compares
  the public maintenance command reference with the installed CLI so Project,
  Source, and adapter commands cannot silently drift.
- **Verified the installed maintenance authority end to end.** Release smoke now
  builds and installs the local npm and platform tarballs into an isolated HOME,
  then exercises CLI guidance, versioned Snapshot/Change Set/Receipt flows,
  Source lifecycle operations, Project rename, managed pointers, optional agent
  adapters, guarded legacy migration, and incompatible-version refusal. A Go
  source audit prevents supported maintenance entry points from reintroducing
  direct canonical-file writes; older composition, contradiction-cleanup, and
  production-merge apply paths now fail closed with exact Core remediation.
  First-run assembly now uses a Core-created one-shot boundary and Core Source
  additions, so later corpus rebuilds cannot replace identity-bearing Sources.
- **Moved Go TUI maintenance onto Liner Core.** The TUI now consumes versioned
  Project Snapshots, Change Sets, Failure Reports, and Change Receipts for
  Source and Project maintenance. Core-provided risk and file effects drive
  explicit confirmation, malformed or crashed Core responses fail closed, and
  the retired Source append/drop paths no longer write `tape.yaml` directly.
  A `Maintain project` TUI command exposes immutable Source IDs and routes
  update, replace, remove, purge, rename, and move through that same contract.
  Confirmation now follows Core risk: additive and metadata plans continue
  after validation, while approval-required plans wait on the exact preview.
- **Kept curator approvals focused on the decision being made.** User-provided
  Source review now summarizes what will be added, what stays unchanged, and
  what happens next instead of exposing Core protocol payloads. After Sources
  are accepted, one stable transition leads to a separate Synthesis approval;
  confirming that approval records the receipt and starts Compile automatically.
- **Added recoverable whole-Project deletion.** Maintain Project now includes a
  guarded Delete Project action that requires the exact Project name, previews
  the identity-preserving move through Core, and writes a receipt. Deleted
  Projects move outside the active Project library into a recoverable Liner
  Trash location, so they disappear from the Project list without erasing files.
- **Accepted Markdown-linked evaluation candidates.** Evaluation validation now
  recognizes candidate titles whose URLs are written as Markdown links, avoiding
  a false `no candidates list` failure after Candidate discovery produced valid
  linked entries.
- **Bundled opt-in Maintenance Adapters.** Liner now ships a thin
  CLI-delegating maintenance skill for Codex and Claude, with explicit
  inspect/install/update/remove commands, durable receipts, fail-closed managed
  content checks, and canonical Change Sets for optional Project pointer blocks.
- **Published CLI-owned Project maintenance guidance.** `liner project guidance`
  now emits equivalent versioned JSON or Markdown, classifies Project Skill
  routing state, keeps skill Sources explicitly inert, and upgrades legacy root
  or declared Project Skill routing only through a reviewed Change Set, folding
  in lazy immutable-identity migration when required.
- **Sharpened generated Project Skills.** Root `SKILL.md` now acts as a concise
  project entrypoint with trigger-oriented description text, explicit load
  order, and checkable completion criteria while leaving detailed operating
  rules in `LINER.md`.
- **Recovered more browser-renderable source blocks.** Vercel Security
  Checkpoint responses now route through the existing JS-render fallback instead
  of surfacing as generic `HTTP 429` fetch failures when Playwright can recover
  the article.
- **Kept source repair refreshes scoped.** When recovered custom source content
  needs reconsideration, Compile now refreshes source evaluation from Evaluation
  onward instead of resetting the corpus build to Candidate discovery.
- **Separated source notes from warnings.** Accepted sources with notes remain
  visible in Compile summaries, but they no longer label an otherwise usable
  compile as `compiled with warnings` or prompt repair when there is nothing to
  fix.
- **Simplified JS rendering setup.** First-run JS rendering setup and Compile's
  Playwright Chromium install state now use one clear explanation, one `i`
  install action, and an indeterminate loader only while setup is actually
  running. The UI no longer shows a fake progress bar or duplicate repair/retry
  instructions before the browser has been installed, and it clears the retrying
  note once the automatic recompile finishes.

## 1.0.2

- **Refined Compile Console source repair.** Compile now opens with a compact
  result summary and sends users to `View sources` for the navigable source
  table. Sources with errors, retryable custom sources, and recovered content
  needing Build Corpus are bubbled to the top with clearer `research source` and
  `custom source` labels.
- **Made custom source recovery explicit.** Repair retries unavailable custom
  YouTube/web sources, installs JS rendering first when needed, stops on a
  recovery review, and asks for a corpus rebuild only when content was actually
  recovered.
- **Kept replacement-source repair in context.** Adding sources from Compile now
  returns to Compile instead of restarting Clarify Goal, and recovered/custom
  local sources are preserved as active source material for the next corpus
  build.
- **Allowed usable partial compiles to continue after review.** Reviewed source
  warnings stay visible, but a usable `MIXTAPE.md` can proceed to Operating
  Layer creation instead of being blocked by stale compile-error state.
- **Aligned Project Complete with its primary action.** `Next` now opens
  `LINER.md` when the project is complete, so Enter no longer lands on a
  no-op management state.

## 1.0.0

- **Ships the Go TUI as the default Liner interface.** Running `liner` or
  `npx linersh` opens the Charm-based Go TUI for creating, opening, and
  managing local Liner Projects.
- **Installs through one npm command.** `npx linersh` or
  `npm install -g linersh` installs the terminal app and the matching local
  binaries for the user's platform, so normal use does not require a Python
  setup.
- **Creates the Operating Layer for completed projects.** After the corpus is
  ready, Liner writes `LINER.md`, root `SKILL.md`, and `liner.yaml` so future
  AI sessions know how to use the project, what evidence to trust, and where
  the boundaries are.
- **Tightens corpus-to-method generation.** Operating Layer output now promotes
  corpus synthesis, quality checks, source roles, and capability patterns into
  executable project behavior instead of leaving generic source-use rules.
- **Keeps clean installs testable.** The installed package can be checked from
  a temporary home directory and project folder, which makes it easier to prove
  a fresh `npx linersh` run opens the same Go TUI a regular user receives.

## 0.5.7

- **Capture PDF downloads during JS fallback.** When a `.pdf` web source first
  returns a JavaScript or bot-detection stub, Liner falls back to Playwright. If
  Chromium treats that PDF as a download, Liner now captures the downloaded file
  and runs the same shared PDF text extraction path instead of marking the
  source failed with `Page.goto: Download is starting`.

## 0.5.6

- **Extract remote PDFs as PDFs.** Web sources that return `application/pdf`,
  start with `%PDF-`, or use a `.pdf` URL path now run through the shared
  `pdfplumber` extraction path instead of being treated as HTML. This prevents
  raw PDF bytes from being written into `sources/` and `MIXTAPE.md`.
- **Reject stale bad cache entries.** Cached web content is now purged and
  refetched when it looks like raw PDF bytes, a PDF URL saved by the old HTML
  fallback, a tiny soft/playwright fallback, or a cookie-consent-only body.
- **Detect empty JavaScript app shells.** Server-rendered HTML that looks like a
  client-side shell with no useful readable body now triggers the existing
  Playwright fallback instead of becoming a misleading successful source.
- **Reject cookie-consent-only sources.** Liner now treats short cookie-banner
  output as unusable, including after Playwright rendering. The source is marked
  unavailable with a clear message instead of feeding consent text into the
  mixtape.
- **Clarify partial compile output.** The CLI now reports usable source count
  versus unavailable placeholder files when a compile is partial, e.g.
  `22/24 usable sources, 2 unavailable placeholders`.
- **Shared PDF extraction helper.** Local and remote PDF handling now use one
  extraction helper so future PDF fixes land in both paths.

## 0.5.5

- **Ships the web extraction fixes to clean npm installs.** Fresh
  `npx linersh` runs on macOS, Linux, and Windows x64 receive the source
  fetching improvements below.
- **Fixed done-screen scrolling.** The final response now has an internal
  scroll viewport in the terminal alternate screen. In the default done view,
  use ↑/↓ or pgup/pgdn to scroll the green final-response box; inspect mode
  still uses those keys for tool-call navigation and keeps the final response
  compact so the detail panel remains visible.
- **Surfaced missing JS-rendering setup during compile.** If a compile hits
  JavaScript-only pages and Playwright Chromium is missing, the TUI now shows a
  dedicated JS-rendering callout with a `j` action that runs
  `liner setup-js --yes`, notes the ~150MB Chromium download, and retries the
  compile.
- **Fixed compile-result source scrolling.** Completed compile screens now keep
  the source list in an internal viewport with ↑/↓ and pgup/pgdn controls, so
  long source lists remain readable in the alternate screen.
- **Clarified the ready-to-compile milestone.** After Assembly, the hub now says
  the methodology is complete and the final mix is assembled, instead of only
  showing Compile as the next step.
- **Treat partial compiles as compiled.** If compile writes `MIXTAPE.md` with
  some missing sources, the hub now lands in a finished "compiled with
  warnings" state instead of asking the user to compile again.
- **Open existing compile results without rerunning.** When `MIXTAPE.md` is
  current, pressing Enter on the completed Compile row opens the final compile
  result screen reconstructed from disk. Recompile is now explicit with `r`
  from that screen.
- **Hardened web extraction fallback.** If the primary HTML extractor crashes
  on a page, Liner now falls back to tag-stripped HTML text and keeps the
  source with a soft warning instead of failing it with a Python exception.
- **Added first-run JS-rendering onboarding.** Fresh installs now get an
  optional setup screen explaining Playwright/Chromium support right after the
  agent/model setup flow, with install and skip paths.
- **Added `liner uninstall`.** The installed command can now remove local
  Liner state, Playwright's Chromium cache, and npm's `_npx` execution cache
  for clean reinstall testing.

## 0.5.4

- **Fixed npm skill-bundle resolution.** The TUI now probes the installed
  package root more directly, tolerates `LINER_SKILL_PATH` pointing either at
  the skill directory or at `SKILL.md`, and shows the checked paths when the
  bundle still cannot be found. This fixes the Phase 2 "Couldn't find the
  curating-mixtapes skill bundle" failure seen after `npx linersh`.
- **Prepared Windows x64 installs.** Windows support was added alongside the
  existing macOS and Linux install paths so Windows users can receive the
  bundled CLI binary through npm.
- **Marketing landing page polish.** The Astro marketing site has a refreshed
  hero, copyable `npx linersh` install CTA, version-consistent `0.5.3`
  labeling, Liner logo treatment, WebGL resize fix, continuous marquees,
  and tightened mobile behavior.
- **Per-phase model selection.** The two heaviest phases — candidate discovery
  and evaluation, ~65% of a cycle's tokens — now run on a cheaper model by
  default (Sonnet for Claude, `gpt-5-mini` for Codex). Framing, quality,
  synthesis, and assembly stay on the agent's default model, where curator
  taste matters most. Cuts a cycle's quota roughly 5×. Override per agent +
  phase in `~/.liner/config.yaml` under `models:` — e.g. set `candidates: opus`
  to undo the downgrade. On a subscription the saving is rate-limit quota, not
  money. Claude phases use the `sonnet` alias, which auto-tracks the latest
  Sonnet, so model version bumps don't need a release.
- **Model-rejection fallback.** If the agent rejects the configured model for a
  phase (renamed, retired, or mistyped), the run warns the curator with the
  exact fix — set/remove `models.<agent>.<phase>` in `~/.liner/config.yaml` —
  and retries once on the agent's default model, so a stale model id degrades
  to "ran on the default" instead of a hard failure.
- **Agent-unavailability watchdog.** A launched-but-mute agent (signed out,
  expired subscription, deprecated model) used to hang the run spinner
  indefinitely. The runner now declares the agent unreachable if it emits no
  output within 15s, surfaces a clear "no response from <agent> — check
  auth/subscription" message, and lands the run in the failed state where
  retry / switch-agent already live.
- **Removed the invented `$` cost** from the process-manifest screen — the
  PhaseRunner's was already dropped in 0.5.2. Subscription-driven runs aren't
  metered API billing, so the dollar figure was fictional; token counts stay.

## 0.5.3

Version-reporting fix + version-pin cleanup.

- **`liner --version` now reports the TUI/npm version**, not just the bundled
  Python core. It used to forward straight to the core and print e.g.
  `liner 0.5.0` regardless of which package you installed. Now prints
  `liner <tui> (tui)  ·  <core> (core)` (and
  just the TUI version when the core isn't resolvable).
- **Single source of truth for the version.** The header chip and the boot
  splash now read the version from `package.json` at runtime instead of
  separately edited string constants.

## 0.5.2

UX redesign release. The project hub, import flow, navigation, and color
language all get a pass; the optional empirical-test phase is removed.

- **Removed the empirical-test phase from the TUI.** Compiling is now the
  completion milestone — *compiled = complete*. There's no longer a trailing
  optional phase the hub parks you on. (The with-vs-without validation lives
  on as methodology guidance, not a driven phase.)
- **Hub redesigned into a single summary view:** a status box (Ready to use /
  In progress / Not started) plus an always-visible phase checklist. Replaces
  the old focus-card / list dual mode.
- **Import auto-detects `.mixtape` files** in the current folder, offers a
  re-scan, drops the opaque "archive" label, and lands you on the imported
  mixtape's hub rather than a phase prompt.
- **Consistent quit key:** `q` quits from any screen; `esc` goes back.
- **One call-to-action per screen:** a single "next · …" description line
  above a uniform key-hint navbar.
- **Removed the misleading dollar-cost figure** from run summaries —
  subscription-driven runs aren't metered API billing. Token counts stay.
- **Reduced scrollback laddering:** the run's final-response panel caps to the
  terminal height (on top of the alternate-screen buffer added in 0.5.1).
- **Fixed the mixtape browser laddering + garbled selected row.** The project
  list now renders into a terminal-sized scrolling viewport (windowed around
  the selection, with above/below indicators) and each row is a single
  truncating line. Previously, on a small terminal the list rendered taller and
  wider than the window — so arrow-key navigation reprinted the header on every
  keystroke and the selected row's highlight bled across wrapped lines. The
  header subtitle also no longer wraps.
- **Fixed a hard API-400 crash on long agent runs.** A multi-turn methodology
  phase (and any resume of a failed run) could die with `400 … thinking blocks
  in the latest assistant message cannot be modified` once Claude Code
  auto-compacted a conversation that had interleaved thinking on. The spawned
  agent now runs with extended thinking disabled (`MAX_THINKING_TOKENS=0`), so
  the failure class can't occur.
- **Isolated the methodology agent.** The spawned Claude now runs with
  `--strict-mcp-config`, so it no longer inherits the user's global MCP servers
  and connectors — less context bloat, narrower reach, no irrelevant tools.

## 0.5.1

First broad bug-fix + polish pass after the initial public release.

- **Fixed:** the `curating-mixtapes` skill bundle now ships inside the npm
  package, so AI-assisted JTBD clarification works for npm installs (it was
  silently falling back to canned questions in 0.5.0).
- New color palette: red-orange primary, green for success/progress, goldenrod
  for the running state, purple headings, hot-pink errors.
- Legibility: replaced Ink's terminal-relative dim with an explicit grey, and
  named-"black" text on colored fills with true black (`#000000`).
- Fixed an ESC navigation loop; ESC now always walks back to the splash.
- Terminal alternate-screen buffer + a clear on run completion to cut
  scrollback artifacts.
- Contextual notes below boxes (curator note, share result) are pink; the
  share result is a persistent note instead of a vanishing toast.
- New `o` command opens the mixtape folder in the OS file manager.
- Cassette splash logo cleaned up and rendered in the brand color.

## 0.5.0 — first public release

First public npm release of the interactive terminal UI.

- `npx linersh` now opens the TUI on clean machines.
- Running `liner` with no arguments opens the TUI after a global install.
- CLI subcommands such as `setup-js`, `compile`, `share`, and `status` continue
  to work from the installed command.
- The install docs now describe the bundled CLI and `liner setup-js` path for
  JavaScript-rendered sources.

## 0.3.0 — unreleased

`local_file` source type + JavaScript-rendering support. Additive to the v1 tape format (now versioned as v1.1 in MIXTAPE-FORMAT.md). No breaking changes to existing tapes.

- New source type `local_file` for content the curator has on disk (PDF, Markdown, text, HTML). Required fields: `path` (must resolve under the project's new `personal/` subdirectory) and `citation`. Max 10MB per file.
- New `render` field on `web` sources. Default `server` (existing behavior); `js` opts into Playwright-backed rendering for JavaScript-only sites.
- JavaScript-rendered sources can use Playwright + Chromium for `render: js`.
  The default install stays lean; browser-backed extraction adds the Chromium
  download only when the user opts into JS rendering.
- `pdfplumber` added to core dependencies for PDF extraction.
- New `liner share --no-personal` flag. `personal/` is included in exports by default; library submissions must pass `--no-personal` and use only public sources.
- `share` prints a soft warning when the tape contains `local_file` sources, noting the result isn't library-eligible.
- Web handler signature changed from `fetch(url)` to `fetch(spec)` so handlers can see metadata they may need (e.g. `render`). Internal change; doesn't affect tape format.
- TUI: new file-picker screen for `local_file` paths; source-modal type toggle (`T`) and `render` toggle (`R`); source-row icons updated.
- 8 new tests; existing test suites extended.
- New `scripts/uninstall.sh` — interactive bash uninstaller that removes the CLI, the Playwright browser, the local cache, and (optionally) globally installed TUI / dev `.venv` / mixtape workspace. Documented in README.
- New `liner setup-js` subcommand. One-shot: installs Playwright via `pipx inject` (or `pip` for non-pipx envs) and downloads the Chromium binary. Idempotent. Replaces the documented two-step "`pipx install 'linersh[js]'` + remember the playwright path" recipe.
- **Web rendering is now auto-fallback by default.** A `web` source with no `render` field tries server-rendered HTML first, and if the page turns out to be JS-walled, automatically falls back to headless Chromium (provided `liner setup-js` has been run). The compile output shows a warning when the fallback triggers so the curator sees the cost. Existing tapes with `render: js` continue to skip the server attempt; new `render: server` opt-out is available for library submissions or diagnostics.

## 0.2.0 — unreleased

Folder-model migration. The mixtape is now a project folder (`tape.yaml`, `synthesis.md`, `sources/`, `working/`, `MIXTAPE.md`), not a single Markdown file. Breaking changes vs. 0.1:

- `liner compile` now takes a project folder and writes outputs in place. The `--output`, `--format`, and `--no-preamble` flags are removed.
- `liner init` now scaffolds a project folder with the starter tape, a `synthesis.md` placeholder, and `working/01..04` stubs.
- `synthesis.md` is required by `liner compile` — missing it is a hard error.
- New commands: `liner share <folder>` (zip → `.mixtape`) and `liner import <archive> [dest]` (unzip + refetch). Share flags: `--no-working-notes`, `--no-source-content`, `--minimal`, `--out`.
- New cache subcommands: `liner cache list`, `liner cache show <url>`.
- Tape format adds optional fields: `mode` (`quick` | `methodology`), `jtbd`, `methodology_version`.
- `liner list` walks for project folders (directories containing `tape.yaml`) instead of flat `*.yaml` files.
- `--emit-events` final `result` payload now reports the produced folder, `MIXTAPE.md` path, and per-source file paths instead of a rendered Markdown string.
- Example reshaped from `docs/examples/mobile-design.yaml` to `docs/examples/mobile-design-foundations/` (runnable project folder).

## 0.1.0 — unreleased

Initial Liner MVP. The project was originally prototyped as "yoyotape" and renamed to Liner before first release; this is the public 0.1.0.

- CLI: `liner init`, `liner compile`, `liner clone`, `liner list`, `liner cache {info,clear,purge}`.
- TUI (`npx linersh` / `liner`): interactive tape editor with browser, editor, source modal, and live-streaming compile view.
- YouTube handler with `youtube-transcript-api` → `yt-dlp` fallback and `--cookies` support.
- Web handler via trafilatura (no JavaScript rendering in v1).
- SQLite cache at `~/.liner/cache.db` (30-day TTL for YouTube, 7-day for web).
- Markdown and JSON output. Compiled mixtapes save as `<tape>.mixtape.md`.
- `liner compile --emit-events` streams NDJSON progress events to stdout for programmatic clients (TUI, future MCP server).
- Soft curator-note nudge in TUI per CURATION.md §6.
