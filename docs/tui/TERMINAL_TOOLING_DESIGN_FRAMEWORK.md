# Terminal Tooling Design Framework

This document captures the design framework behind Liner's terminal UI and CLI
work. It turns the recent feedback, corrections, and design decisions into a
shared reference for future agents.

Use this together with:

- [GO_TUI_SCREEN_PATTERNS.md](GO_TUI_SCREEN_PATTERNS.md)
- [../project/PRODUCT.md](../project/PRODUCT.md)
- [../project/CONTEXT.md](../project/CONTEXT.md)

## Design Posture

The TUI is not a decorative wrapper around commands. It is a working surface for
creating, opening, inspecting, and improving Liner projects over time.

The design should feel:

- calm enough to read
- dense enough to manage real projects
- explicit enough to avoid accidental writes
- restrained enough that state, actions, and navigation do not compete
- durable enough that a future agent can extend it without re-litigating the IA

The design should not feel like:

- a marketing page
- a command dump
- a wizard that disappears after setup
- a dashboard full of duplicated status
- a themed terminal toy where color and decoration obscure meaning

## Product Model

Keep the product model artifact-first.

- A Liner project is the main working folder and management object.
- `MIXTAPE.md` is the compiled context artifact.
- `LINER.md` is the project instruction file for how AI sessions should
  use that context.
- The Project Skill is root `SKILL.md`, created by default so future AI sessions
  can invoke the project directly.
- Audit is later maintenance work, not part of default Operating Layer
  creation.
- Composition is an advanced project-combination model and should not leak into
  the default creation flow.
- External Use Evidence is a V2 concept produced by agents using the project,
  not an internal v1 Impact Tests surface.

Avoid persona-heavy framing such as `advisor` as the primary product noun. Avoid
vague layers such as `living layer` when the product can name the actual thing:
Project Skill, Operating Layer, Status Snapshot, composition, or project
instructions.

The v1 lifecycle is intentionally small:

1. Started: the Project Shell exists.
2. Corpus Ready: `MIXTAPE.md` exists and the Mixtape can be used as grounded
   context, even when the Compile Console still carries reviewed source
   warnings.
3. Project Complete: the Operating Layer has written `LINER.md`, root
   `SKILL.md`, and `liner.yaml`.

Do not turn parked or future-facing capabilities into visible v1 product
promises. If a concept belongs to V2, document it as V2 and remove it from
navigation, Capabilities, and `Next`.

The installed product model is also part of the design. Users install one npm
package, `linersh`, and run one command, `liner` or `npx linersh`. The Go TUI
and bundled Python CLI are separate implementation pieces, but they should feel
like one local tool. Do not make public docs, setup screens, or error recovery
sound like the user has to assemble two products.

Optional dependencies should stay opt-in and legible. Plain install opens the
TUI without Python setup or Chromium. `liner setup-js` is the explicit browser
rendering path for JavaScript-heavy sources. The UI can offer it during
onboarding or from compile warnings, but it should explain the download and why
the user needs it.

## Capability And Evidence Contract

Liner creates hyper-specific corpora that help an AI agent perform one task
better than it could from general training data alone. A Liner project is not a
reading list, a source dump, or a persona. It is a local artifact set that
teaches an agent what evidence to trust, how to use it, where the limits are,
and what kind of output the task requires.

The TUI exists to make that artifact-building process visible and recoverable:

- The user describes the capability they want a future AI agent to have.
- Liner frames that capability into a JTBD, a knowledge map, and required source
  roles.
- The agent gathers, evaluates, and checks evidence against those roles.
- The user reviews meaningful decisions at phase boundaries.
- Compile produces `MIXTAPE.md`, `sources/`, and the project files a downstream
  agent can read.

Phase 1 must do more than write a knowledge map. It must create an evidence plan
that Phase 2 can search against and Phase 5 can audit.

`working/01-jtbd-and-knowledge-map.md` must contain:

- `## Capability Brief`: the concrete capability this Liner should give a
  downstream AI agent.
- `## Required source roles`: the kinds of evidence required before the corpus
  can be trusted.
- A knowledge map: the topical areas the corpus must cover.

Each source role must include:

- role name
- why it matters
- what good evidence looks like
- minimum coverage

This is deliberately not hardcoded to Art Director. A medical SEO project, a CI
debugging project, a grant-writing project, and an Art Director project should
all produce different source roles from their own task.

If Phase 1 exits successfully but this contract is missing, the TUI should show
`needs repair`, keep the project on Phase 1, and let Enter resume the agent with
the validation message.

Liner should evaluate sources by content, not by URL shape. A URL can look
legitimate and still fetch garbage, shallow content, a 404 page, or a tiny
JavaScript stub. A source is useful only when fetched content supports the role
it was kept for.

Phase 2 should cast wide and verify candidates exist. Phase 4 should fetch and
read content before deciding keep, trim, or drop. Phase 5 should ask whether the
kept pile satisfies the required source roles, not only whether every
knowledge-map section has something in it.

Useful quality questions:

- Does this source make the downstream AI better at the target task?
- Which required role does it serve?
- Is the fetched content strong enough, or merely adjacent?
- Is the source role missing entirely?
- Should Liner run an improvement pass, ask the user for custom sources, or
  continue with a documented limitation?

The Operating Layer must inherit the corpus method, not only the corpus
inventory. A strong `mixtape/synthesis.md` is not enough if `LINER.md` and root
`SKILL.md` stay generic. When Phase 5 has extracted a core action, capability
pattern, required source roles, generative rules, constraints, and caller
handoff model, the Operating Layer should turn those into executable behavior
for the next AI session.

For example, an Art Director Liner should not merely say "load the mixtape and
cite sources." It should say what kind of art direction the corpus supports:
read visible evidence before naming mood, translate relationships rather than
motifs, make web constraints explicit, turn taste into observable criteria, and
hand off tokens, layout primitives, component rules, states, examples,
exclusions, and evaluation checks. This same principle should adapt to other
domains instead of becoming a hardcoded Art Director template.

## Information Architecture

Each screen should have one job.

Home is for starting or navigating. It should stay calm and action-oriented.
Project state does not belong there.

Projects is for browsing the project list. It should show enough metadata to
choose the right project, not every capability count.

Project is for managing one opened project. It can be denser because the user is
already inside the object.

Settings is for choosing the local projects folder and AI runner. It should not
become a config dump or command reference.

Add Sources is for source ingress, not source judgment. It may accept URLs,
YouTube links, GitHub skill URLs, installed skill names, local files, existing
`local-sources/...` paths, and pasted article text. It should make the source
save behavior clear without pretending every input has been evaluated yet.
Authenticated or paid material uses the current "capture yourself" path: the
user copies useful rendered text from their own browser and Liner saves it under
`local-sources/captured/`. Do not imply browser capture exists until it does.

Compile is the recovery surface for source problems. A usable `MIXTAPE.md` can
exist with warnings, but blocking source problems need visible actions: retry,
install JS rendering support, open a source, drop a source, add more sources, or
continue with a documented limitation when that is genuinely safe.

Compile has two stable surfaces: a compact result summary and a full Sources
review. The summary should explain what happened in a few lines, then point to
`View sources` when the user needs detail. It should not render the full source
table inline. Sources review owns the table, bubbles errors and retryable rows
to the top, and keeps usable rows scrollable behind the same table controls.
Accepted source notes may appear in the summary as informational `Source notes`,
but they must not turn a successful compile into `compiled with warnings` unless
there is an actionable source issue to review or repair.

Custom sources are user intent, not optional research candidates. If a custom
YouTube or web source is missing, say what happened: `transcript was not
fetched`, `no readable body`, `recovered content needs source evaluation
refresh`, or the equivalent concrete reason. Do not describe these rows as
merely `left outside tape.yaml`; that names an implementation effect, not the
failure the user can understand or repair.

Repair from Compile is a guided batch, not a mystery rerun. It retries
unavailable custom sources, installs JS rendering first when needed, and then
returns to a review state. If no sources recover, stop and send the user back to
Sources with repair, open, and add-replacement choices. If sources recover, say
where the local copies were saved, then ask the user to refresh source
evaluation from Evaluation onward so Liner can reconsider them without rerunning
discovery. Recovered custom sources should be preserved as active source
material, not silently dropped by the next agent pass.

Regression coverage for this contract lives in
`TestRetrySourceEvaluationStartsEvaluationFresh` and
`TestCompileResultSurfacesSourceEvaluationIssuesAndRetriesDroppedSources`.
Before changing Compile repair, run the Go TUI app tests and confirm those cases
still prove the runner starts `evaluation`, not `candidates`, and that the UI
does not say `rebuild corpus` or `from Candidate discovery` for recovered source
repair.

Add Sources opened from Compile must return to Compile. Saving a replacement
source should append it to the tape and ask the user to retry the relevant
compile/build step; it should not restart Clarify Goal or send the user back to
the beginning of the project flow.

The current pattern set is:

- Command Hub: Home.
- Preference Chooser: Settings.
- Split Browser: Projects.
- Section Workspace: Project.
- File Picker: Import Project.
- Progress Console: Compile Console, Build Corpus, Create Operating Layer,
  Setup JS rendering, Clarify Goal, and Import Project.

When a new screen does not fit one of these patterns, define the job first, then
decide whether the pattern should be extended or a new pattern should be named.

## Navigation

There should be one navigation/menu surface: the footer.

Do not duplicate key hints in the body when the footer already owns them. Do not
create a second command strip or shortcut row above `Next`.

The top banner owns location. The body should not add another competing location
row such as `Commands ... System` or `Project ... Current project`.

Expected escape routes:

- `h` returns to Home from non-text screens.
- `esc` goes back when there is a previous meaningful screen.
- `?` expands help in the footer, not in the body.
- `:` should not resurrect a separate Commands surface.

Text-entry screens can reserve letters for typing; non-text screens should show
clear home/back routes.

## Visual Hierarchy

Use bold only for titles.

Descriptions, selected navigation items, table cells, table headers, footer
guidance, and `Next` copy are not bold. The UI should not need typographic
shouting to communicate state.

Color is semantic:

- Orange is the primary selected/action accent.
- Softer orange is for `Next`.
- Gray is for labels, descriptions, and secondary context.
- White is for values and readable content.
- Purple is not the Liner action color.

Selected command titles may turn orange, but selected descriptions stay gray.
The selected thing should be visible without repainting every nearby piece of
copy as an action.

Avoid decorative boxes and nested cards. Tables, spacing, and alignment should
carry structure before borders do.

## Copy And Naming

Use clear product nouns.

- Use `Status`, not `State`, in user-facing UI.
- Use `Name`, not `Project`, when labeling the selected project's name.
- Use `Settings` for the app-level projects folder and AI runner surface.
- Use `Operating Layer`, not `Capabilities`, for `LINER.md` and Project Skill
  readiness.
- Use `Open MIXTAPE.md` only when Enter actually opens the compiled artifact.
- Use `AI runner`, not `provider`, when the user is choosing Codex or Claude
  Code.
- Use `projects folder`, not `project library`, when the user chooses where
  Liner projects live.
- Use `saved sources`, not `accepted sources`, unless the screen is explicitly
  about accepting a review.
- Use `JS rendering support` or `JavaScript rendering support` for the
  `setup-js` path. Avoid implying the browser dependency is installed by
  default.
- Use `capture yourself` for authenticated or paid source capture until a real
  browser-capture feature ships.

Capitalize sentence starts and labels unless a proper noun intentionally starts
lowercase, such as `iOS`.

Do not show implementation concepts unless the user can act on them from that
screen. Config files, env overrides, active/saved runner internals, and command
maps should stay out of Settings unless they become real controls.

## Layout Rules

Every screen needs a stable layout contract.

For split layouts:

- Left side is selection.
- Right side is detail.
- Long text wraps inside the right detail area.
- Long paths should not wrap back into the left column.

For section workspaces:

- The opened artifact identity appears at the top as name plus description.
- The left pane shows section names only.
- Do not add mini-statuses such as `complete`, `ready`, `1 run`, or `basic` next
  to section names.
- The right pane owns explanation, status, and evidence.
- Every section detail starts with one title, one muted description, one blank
  line, and one table.
- Each section owns one table. If a second table has a different purpose, make it
  a new section.
- Table headers and cells must start at the same left positions.
- Add enough spacing between columns so headings and values do not read as one
  long string.

## Tables And Detail Blocks

Use tables when comparing multiple rows or clarifying several facts.

Use label/value rows for compact single-choice detail blocks, such as Settings
runner details or selected-project metadata in the Projects browser.

Do not add `Field`/`Value` headers when the screen is using the compact
label/value pattern. Do use headers when the section is intentionally a table and
the columns help the user scan.

Tables should make dense information scannable:

- left-aligned headers
- left-aligned values
- visible column spacing
- no unnecessary bold
- no collisions between adjacent columns

## Action Semantics

Be precise about the difference between current action, recommended next step,
and product completeness.

`Next` is the recommended next action for the current Liner Project state. It
is not the same thing as the selected-row action or a generic primary button.
When Enter is shown next to it, the Enter behavior must actually perform that
action.

The default project path is:

- before Corpus Ready: continue Corpus Creation
- after Corpus Ready: Create Operating Layer
- after Project Complete: Open `LINER.md`

Do not imply a project is fully complete just because `MIXTAPE.md` exists. A
Mixtape can be ready while the larger Liner Project still needs the Operating
Layer.

`Create Operating Layer` is one guided write action. It explains the benefit,
then writes `LINER.md`, root `SKILL.md`, and `liner.yaml`. It should
not show review screens or optional-skill decision language in this step.

Creating or regenerating the Operating Layer should be blocked only by compile
problems that make the corpus unusable. If compile produced usable partial
output, the user can review source warnings and still continue with the current
`MIXTAPE.md`. The UI should keep the warnings visible, but the `Next` label and
guard must agree: do not say `Create Operating Layer` and then reject Enter with
a hidden compile-status error. Recovered browser-rendered sources can be shown
as success notes, not warnings that keep the project in a repair state.

The Create Operating Layer screen has no sub-feature panel. In the idle state,
it explains the benefit in one short message and leaves controls to the footer.
Keep the output inventory out of the idle body. After Enter starts the write, a
compact progress list can name `LINER.md`, `SKILL.md`, and `liner.yaml` because
the user is waiting on those files.

The Project Skill must be written as root `SKILL.md`, with skill frontmatter.
The legacy `skills/liner-*.md` shape breaks the skill-folder contract this
product relies on.

Generated Project Skills are entrypoints, not duplicate operating manuals. The
`description` should name when the skill should fire, `SKILL.md` should state
load order and checkable completion criteria, and `LINER.md` should remain the
single source of truth for detailed operating rules. If a method changes, update
`LINER.md`; do not let root `SKILL.md` accumulate copied rule prose.

On Project, `Esc` returns to Projects. Projects is the parent surface for an
opened project. The short footer should keep `h home` and `esc back` visible
before settings and quit so users can see the escape route on narrower
terminals.

Project Complete is a management state, and its default Enter action is to open
`LINER.md`. No primary action should be a no-op. Say the project is complete,
then expose regeneration and other management actions through footer help and
Home commands.

Regeneration is a management action and must be visible where the user needs
it. Do not hide it only in a command palette. On a Project Complete screen with
a ready corpus and no actionable compile warnings, expose a compact footer
control such as `r regen layer`, and route it through the same review/write
path as Operating Layer creation.

`Refresh status` is a narrow status-only write to `liner.yaml`. It may update
the Status Snapshot, but it must not edit corpus files, `LINER.md`, root
`SKILL.md`, Audit reports, sources, or working notes.

Internal Impact Tests should not appear in v1 navigation, Capabilities, `Next`,
or the default project-management flow. Treat existing Impact Test code as
parked V2 prototype work until External Use Evidence is designed from the
external-agent usage side.

## Safety And Trust

The TUI should make write behavior visible.

Keep action tables where the `Writes` column prevents unsafe ambiguity. Remove
action tables when they only repeat footer shortcuts.

Preview, open, and render paths must not create files or directories. For
example, `local-sources/` should not be created merely by opening or viewing a
project.

Views should not shell out or mutate files. Status loading should happen through
explicit commands or cached state, not during render.

Installation and cleanup are part of trust. A clean trial should be possible with
temporary `HOME`, `npm_config_cache`, and `LINER_DIR` values so the user can test
from the website flow without touching their real projects, npm cache, or
Playwright cache. When Liner installs something for a test or recovery path, the
docs and UI should also name the cleanup path.

Progress indicators are also part of trust. A determinate progress bar promises
measured, forward-only progress. Use it only when Liner knows the real unit and
can move monotonically, such as sources, phases, or artifact steps. For
Playwright Chromium setup and other installs/downloads where the app cannot
measure true progress, use the title loader with an indeterminate wait/status
block instead of an animated fake percentage.

When browser rendering is available but not installed, do not show progress UI
at all. Explain that some pages need JavaScript/Playwright to reveal readable
article text, say that Liner will retry automatically after setup succeeds, and
end with one highlighted install action. Avoid stacking separate "missing",
"install", "retry", and "press key" messages for the same state.

Published package contents are user-facing surfaces. The curation skill bundle
copied into `packages/tui/cli-update-docs/`, the npm README, and the website docs
shape the installed experience just as much as the TUI screens. Stale handoffs,
unfinished placeholders, old Ink instructions, or archived release notes inside
a package are product bugs, not harmless internal clutter.

## Design History So Far

These are the main decisions that shaped the current framework:

- Removed duplicate body navigation menus; the footer became the single key-hint
  surface.
- Removed the second body location row; the top banner became the location
  source of truth.
- Replaced Bubble's purple selection/action color with Liner orange.
- Kept selected descriptions gray so actions and context do not blur together.
- Changed `State` to `Status`.
- Added `h home` and `esc back` where users need obvious escape routes.
- Kept Settings focused on the local projects folder and AI runner.
- Removed command help, config details, env overrides, install lists, and model
  internals from Settings when they were not actionable.
- Kept Home as a command hub and moved project density to Projects/Project.
- Made Projects a left-list/right-detail browser.
- Made Project a section workspace for one opened project.
- Removed generic top labels from Project and Projects in favor of object name
  plus description.
- Split Perspectives out of Quality because each section should own one table.
- Established one title, one description, one table as the Section Workspace
  detail shape.
- Reserved bold for titles only.
- Started a historical decision log instead of hard-coding unresolved
  completion and capability sequencing into the UI.
- Reframed the activity cue as `Next`, a milestone recommendation instead of an
  Enter-key echo.
- Reduced project completion to three milestones: Started, Corpus Ready, and
  Project Complete.
- Made `liner.yaml` the fast Status Snapshot home for milestones, stale state,
  evidence pointers, and Project Skill status.
- Moved Audit out of default Operating Layer creation and kept it as later
  maintenance work.
- Made the Project Skill default behavior inside Operating Layer creation, not a
  user decision.
- Made `LINER.md`, root `SKILL.md`, and `liner.yaml` the Operating Layer
  outputs that mark the project complete.
- Removed the idle Operating Layer output inventory. The screen now explains the
  benefit first and shows file progress only while the write is running.
- Moved the Project Skill from legacy `skills/liner-*.md` to root `SKILL.md`,
  matching the skill folder contract.
- Made `Esc` from Project return to Projects and kept `esc back` visible in the
  short footer before settings and quit.
- Removed default share/export cues from compile and project-complete surfaces.
  A ready `MIXTAPE.md` is enough for v1 unless the user chooses an archive flow.
- Removed internal Impact Tests from the v1 product surface and reframed future
  evidence as External Use Evidence produced by agents using the project.
- Reworked clarification and curation so questions and source counts are
  generated from the job rather than hardcoded. If question generation fails,
  the UI should say so and offer retry instead of falling back to canned
  prompts.
- Added source-role and capability-pattern checks so a project can verify
  whether its corpus fits the intended downstream capability, including
  reference-translation work such as an Art Director Liner.
- Made partial compile states actionable. Failed sources should route back to
  compile repair; recovered browser-rendered sources should be informational
  success notes.
- Made Project Complete expose regeneration as a visible management action, not
  a hidden command-palette-only affordance.
- Updated Operating Layer generation so `LINER.md` and root `SKILL.md` carry
  corpus-derived method, not only source counts and generic source-use rules.
- Shipped the Go TUI through the single `linersh` install path while preserving
  CLI subcommands behind the same `liner` command.
- Added clean-user smoke testing as a release design requirement: temporary home,
  npm cache, and Liner workspace, followed by cleanup.
- Treated bundled docs as shipped product surface. The package skill docs,
  public docs, and internal working docs should not carry stale handoffs,
  unfinished placeholders, or old Ink-era instructions.
- Kept JavaScript rendering explicit and recoverable: `setup-js` is opt-in,
  surfaced by onboarding and compile warnings, and paired with uninstall/cleanup
  guidance.
- Simplified Compile Console into compact summary first, Sources detail on
  demand. Source tables now prioritize errors and retryable custom sources,
  keep usable rows scrollable, and use `research source` / `custom source`
  labels instead of mixing source origins into one undifferentiated list.
- Made custom source repair explicit. Missing custom YouTube/web sources say why
  they are missing, repair retries them as a batch, recovered content is saved
  under `local-sources/recovered/`, and the next step is a visible source
  evaluation refresh from Evaluation onward instead of a sudden full-process
  jump.
- Made source replacement local to Compile. Adding replacements from Compile
  returns to Compile and preserves project context instead of restarting
  Clarify Goal.
- Made usable partial compiles continue-able after review. Source warnings stay
  visible, but they do not block Operating Layer creation when `MIXTAPE.md` is
  usable and the user has reviewed the issues.
- Made Project Complete Enter open `LINER.md`, so the visible `Next` action and
  the keyboard behavior agree.

## How Future Agents Should Use This

Before adding or reshaping a screen:

1. Identify the screen's job.
2. Choose an existing pattern from `GO_TUI_SCREEN_PATTERNS.md`.
3. Check this framework for IA, copy, visual hierarchy, and action semantics.
4. Add or update tests if the work affects wrapping, spacing, color, keyboard
   behavior, or write safety.
5. Update the pattern catalog when a new stable shape emerges.

When the user gives design feedback, treat it as product input, not cosmetic
preference. The corrections usually encode a boundary: action versus state,
project versus artifact, navigation versus content, current action versus future
readiness, or compact choice versus management surface.

## Always Last: Arturo As A Designer

Keep this section at the bottom of the document. It is not only for future
agents. It is a record of Arturo's design process, values, and working method
for terminal tools and command-line interfaces.

Arturo designs from the actual surface, not from an abstract feature list. He
looks at the screen in use, notices where the interface produces hesitation, and
turns that hesitation into a product question. If a screen says `home`,
`commands`, and `system` at the same time, the issue is not visual clutter
alone. The issue is that the product has not decided where the user is. If a
panel called Settings does not let the user change settings, the issue is not
wording alone. The issue is that the information architecture is lying about
the job of the screen.

His process is usually:

1. Start with the live interface.
2. Name what feels confusing, duplicated, vague, or falsely important.
3. Ask which product concept should own that information.
4. Remove the extra layer, row, label, color, or command if it is not doing real
   work.
5. Rename the thing until the noun matches the product model.
6. Separate state, action, navigation, and evidence.
7. Turn the cleaned-up shape into a reusable pattern.
8. Write the decision down so the next screen can inherit the thinking.

This is why his feedback often starts as UI critique but becomes architecture.
Changing `State` to `Status`, `library folder` to `Projects folder`, or
`Provider` to `AI runner` is not copy polish. Those changes make the product
model more precise. The words tell the user what kind of object they are dealing
with, what can be changed, and what is only being reported.

He looks for interfaces that are calm, direct, and durable. A terminal interface
should feel like a tool someone can return to every day, not a one-time wizard
or a command dump. It should help the user know where they are, what object they
are inspecting, what action will happen next, and whether that action writes
anything. It should be dense when the work is dense, but only after the screen
has a clear job.

He prioritizes orientation first. There should be one location marker, one
navigation surface, and obvious ways home or back. He does not want competing
rows of shortcuts, repeated section labels, or multiple headers that imply
different locations. The top banner says where you are. The body does the work.
The footer owns navigation and key hints.

He prioritizes semantic visual hierarchy over decoration. Orange means selected
or primary action. Softer orange means guidance. Gray means label, explanation,
or secondary context. White means readable content. Bold is reserved for titles.
This keeps the interface from turning every piece of text into urgency.

He treats layout as trust. Wrapped text must stay in its column. Tables should
align with their headers. Long paths should not spill into a selection list.
Section names should not carry tiny summaries if the right pane already explains
the section. These details matter because a terminal UI has very little visual
material to work with; alignment, spacing, and naming carry most of the
experience.

He does not remove information because he wants the interface to be sparse. He
removes information when it belongs somewhere else, repeats something already
visible, or exposes implementation details without giving the user control. The
goal is not minimalism. The goal is a tool where every visible element earns its
place.

He is especially sensitive to the difference between artifact, project, and
process. A `MIXTAPE.md` can be compiled while the larger Liner project still
needs its Operating Layer. The UI should not collapse those stages into one word
like complete. It should show which artifact is ready, which project milestone
has been reached, and which local files still need to be written.

He designs by reducing the number of decisions a user has to make, not by
hiding complexity. When the product could have exposed separate audit types,
skill tools, impact tests, and operating instructions as peers, he pushed toward
one guided `Create Operating Layer` action. Inside that action, the user should
understand the benefit before seeing implementation detail. The rest of the flow
should write the Operating Layer consistently: `LINER.md`, root `SKILL.md`, and
`liner.yaml`.

He treats file shape as part of the user experience. If Liner says it creates a
Project Skill, the generated file needs to match how skills are picked up. A
root `SKILL.md` tells the truth about the artifact. A file named
`skills/liner-project.md` creates a mismatch between the UI promise and the
artifact the runner can pick up.

He designs for AI enthusiasts and AI pro users. The product can use exact file
names, but the screen should explain what those files do in ordinary terms.
`MIXTAPE.md` is the context packet. `LINER.md` tells AI sessions how to use it.
`SKILL.md` lets the project be loaded again by name. When a screen explains
those jobs well, the user can follow the process without already knowing the
internals.

He prefers saved state when saved state makes the product faster, but only when
it remains tied to evidence. The Status Snapshot belongs in `liner.yaml` because
project lists and headers should not rescan an entire folder every time. The
snapshot still needs cited evidence paths, a stale marker, and a narrow Refresh
Status action that updates status without touching project content. Speed is
good; ungrounded badges are not.

He is willing to remove a working feature from the visible product when the
concept is not mature enough. Impact Tests already had implementation shape, but
the model was wrong for v1 because impact is proven when an external agent uses
the project. The right move was not to keep a half-true internal tool in
navigation. The right move was to park it as V2 External Use Evidence and keep
the v1 flow honest.

He tables complexity deliberately. Composition, sharing, and external use
evidence are important, but they should not leak into the creation flow until
their nouns and boundaries are ready. Future agents should treat "we will talk
about that later" as an architectural boundary, not as permission to add a small
placeholder button.

He treats naming as a design instrument. `Source Inbox`, `Project Skill`,
`Status Snapshot`, `Audit Output`, `Blocking Finding`, `Cleared Finding`,
`Operating Layer`, and `Next` all encode product boundaries. Do not rename them
casually. If a term feels slightly off, use that discomfort as a signal that the
model may still be blurry.

As a mini case study, Arturo's Liner TUI work can be described this way:

Arturo inherited a terminal interface with useful capability but blurry
boundaries. The screens exposed commands, status, runner data, project data,
and corpus-build details in ways that competed with each other. Rather than
restyling the UI, he repeatedly asked what each screen was for. Home became a
command hub. Projects became a project browser. Settings became a compact
projects-folder and AI-runner chooser. Project became a section workspace. Along
the way, duplicate menus were removed, purple was replaced by Liner orange,
selected descriptions stayed gray, generic labels disappeared, and dense tables
were given alignment and breathing room.

The result is a design approach where the interface becomes clearer because the
product model becomes clearer. The terminal is not treated as a limitation to
decorate around. It is treated as a precise working medium: text, color,
spacing, keyboard behavior, and file-backed truth all have to agree.

The latest Liner grilling session sharpened the operating philosophy further:
make v1 a personal local project workflow, make completion small and explicit,
make evidence visible, keep future systems out of the default path, and let
external agents eventually produce evidence from real use instead of asking the
TUI to simulate impact internally.

The latest Operating Layer polish sharpened the instruction philosophy further:
benefit first, mechanics only when they help. The idle screen should not read
like a release note. If the product knows the path, take the path. Let progress,
status, and generated files carry the mechanics.

The Art Director testing loop added another important Arturo pattern: do not
accept project completion as a label until the artifact does the job it claims
to do. A project can compile, write files, and still fail if the generated
Operating Layer is generic. Arturo compared the output against older attempts,
asked whether it would actually help a calling agent, and used the gap to find
the product bug: synthesis had been created, but not promoted into operating
behavior.

His correction was not "make the Art Director prettier." It was "make Liner as
a whole unable to repeat this failure." That distinction matters. The fix had to
change the generation contract, retry behavior, warning handling, and visible
management controls so the next project with the same description can produce a
usable project without relying on manual rescue.

This is also why he pushed on the terminal footer and warning layout. The issue
was not just that a message was long. The issue was that a terminal tool must
always leave the user with a route forward: what failed, whether it is
recoverable, how to retry, and how to reach the footer controls at any terminal
height. A helpful error is compact, actionable, and navigable.

The Compile Console source-repair pass made this even sharper. Arturo rejected
the long source dump because it answered an implementation question before the
user had a product question. The first screen should say: the mixtape is usable,
these source types need attention, and the next action is `View sources`. The
full table belongs one level deeper, where row navigation, source detail, open,
repair, and add-replacement actions can all work together.

He also caught the semantic bug behind `left outside tape.yaml`. Custom sources
are not ordinary research candidates that Liner may choose to omit. They are
the user's explicit curation choices. If one is missing, the product has to say
why: transcript not fetched, no readable body, blocked access, recovered content
needs source evaluation refresh, or another concrete cause. The row should
explain the problem and the next repair path, not name the internal file it
failed to enter.

The same principle applies to flow jumps. When a replacement source is added
from Compile, the user is still repairing Compile. Sending them back to Clarify
Goal makes the product feel like it forgot why they were acting. Arturo designs
these transitions as memory tests: after any action, the next screen should
prove that Liner still understands the user's current job.

He is also strict about partial success. If Liner has a usable `MIXTAPE.md`,
the UI should not strand the user behind stale warnings. The right behavior is
to show the warnings, invite repair, and still let the user create the
Operating Layer after review. A red line at the bottom that contradicts the
visible `Next` action is not an error message. It is a broken contract.

The v1 release pass added another Arturo pattern: treat the installed-user flow
as the real interface. It is not enough for the repo build to work. A person
arriving from the website should be able to run `npx linersh`, understand why
optional JS rendering exists, create a project in a clean workspace, and remove
the trial without leaving caches, Playwright downloads, or test folders behind.
That cleanup requirement is part of the product's respect for the user's
machine.

He also treats documentation as operational surface, not aftercare. The public
README, website docs, package README, bundled skill docs, release checklist, and
internal handoff notes all steer either users or future agents. If one of those
documents still describes the Ink TUI, a deleted handoff, an old tarball flow, or
an unfinished placeholder, the product model has split. Arturo's instinct is to
remove or archive stale documents rather than let future work inherit false
context.

That same instinct is why he wanted to test from the website "as a regular user"
after publishing. The release path is not validated by confidence in the code
alone. It is validated when the nouns, commands, package contents, cleanup path,
and first-run TUI all agree from a cold start.
