# Go TUI Screen Patterns

This is the shared pattern catalog for new Go TUI screens. Use it with the
[Terminal Tooling Design Framework](TERMINAL_TOOLING_DESIGN_FRAMEWORK.md) when
adding or reshaping a surface.

## Global Frame

Every screen sits inside the same app frame:

- Top banner: brand, visual divider, and one location label.
- Body: the screen's actual task.
- Footer: navigation and key hints.
- Activity row: only when there is a real next action.

Do not duplicate footer shortcuts in the body. Do not add a second location row
below the banner. Use orange for the selected item or primary action, softer
orange for `Next`, gray for labels and descriptions, and white for values. Use
bold only for titles. Descriptions, selected navigation items, table cells, and
footer guidance are not bold.

## Pattern 1: Command Hub

Use for a calm launch surface where the user chooses where to go next.

Current screen:

- Home

Code anchors:

- `packages/go-tui/internal/app/commands.go`
- `viewHome`
- `newCommandList`
- `commandItems`

Anatomy:

```text
liner v1 ////// home

New Liner
Create a project and add sources

Projects
Browse Liner projects

Settings
Choose the projects folder and AI runner

footer help
```

Rules:

- The body is a selectable, filterable command list.
- Keep project-state density out of Home.
- The selected command title uses orange.
- The selected command description stays gray.
- Filtering uses the list filter, not a separate filter panel.
- Footer owns `enter`, `/`, `h`, `esc`, and `?` guidance.

Use this when a screen is mostly navigation or launch actions.

## Pattern 2: Preference Chooser

Use for a small app-level choice with a few options and only the facts needed to
make the choice.

Current screen:

- Settings
- Setup: AI runner
- Setup: JS rendering
- Improve Corpus

Code anchors:

- `packages/go-tui/internal/app/settings.go`
- `viewSettings`
- `settingsProviderSelectorView`
- `settingsProviderDetailsView`
- `packages/go-tui/internal/app/screen_patterns.go`
- `renderChoiceSelector`
- `renderChoiceDetail`

Anatomy:

```text
liner v1 ////// settings

Settings
Choose the AI runner Liner uses to research sources and create project files.

Codex    Claude

Codex CLI. Active runner.

footer help
```

Rules:

- Put the choices near the top.
- Use orange only for the selected choice.
- Keep unselected choices gray.
- Show details below the selector, not in a side pane.
- The selected choice detail is gray body text.
- Do not repeat the selected option as a label in the detail line.
- Do not use `Label: Value` rows for compact chooser details.
- Add breathing room above and below the detail block.
- Do not show config files, env overrides, command maps, or install lists unless
  the user can act on them directly on this screen.
- Do not show `Next` when the only action is selecting the preference.
- For a flow decision, `Next` should describe the selected action. Example:
  `Improve now` → run the improvement pass; `Skip` → continue to the Operating
  Layer. Keep the preview or notes action secondary.

Use this when a screen is a compact chooser, toggle, or preference selector.

## Pattern 3: Split Browser

Use for browsing a collection while inspecting the selected item without opening
it.

Current screen:

- Projects

Code anchors:

- `packages/go-tui/internal/app/project.go`
- `viewProjects`
- `projectRows`
- `projectBrowserSelectedDetail`
- `packages/go-tui/internal/app/screen_patterns.go`
- `renderLabelValueBlock`

Anatomy:

```text
liner v1 ////// projects

Projects                         Name:        iOS appstore launch
Open and manage Liner projects   Status:      Compiled
                                 Description: Guidance for iOS appstore launch.
> iOS appstore launch            Job:         When I finish...
                                 Folder:      /path/to/project

footer help
```

Rules:

- Left pane is a focused list of item names.
- Right pane is the selected item detail.
- Do not repeat the selected item as a separate title above the detail block.
- Do not render Field/Value headers.
- Put the selected item's name as the first label/value row, labeled `Name`.
- Put descriptions inside the detail block.
- Keep the default Projects detail to identity and decision context: `Name`,
  `Status`, `Description`, `Job`, and `Folder`.
- Move source/skill/audit/impact/child counts to deeper workspace or capability
  surfaces.
- Hide empty default filter copy such as `Filter All projects`; show filter
  state only when the user is filtering or has a query.
- Detail labels are gray. Detail values are white.
- Long values, including paths, wrap under their value column.
- Footer owns open, filter, home, back, refresh, quit, and help hints.

Use this when the user is choosing from a library and needs enough metadata to
decide whether to open the selected item.

## Pattern 4: Section Workspace

Use for an opened artifact where the user reviews several related parts without
leaving the screen.

Current screen:

- Project

Code anchors:

- `packages/go-tui/internal/app/project.go`
- `viewProject`
- `projectPaneList`
- `projectPaneDetail`
- `projectHealthDetail`
- `packages/go-tui/internal/app/tables.go`
- `newDataTable`
- `newMetadataTable`

Anatomy:

```text
liner v1 ////// project

iOS appstore launch
Guidance for iOS appstore launch.

Sections                         Flow
  Health                         2 of 4 steps complete
> Flow
  Sources                        Step              Status    Evidence
  Artifacts
  Operating Layer                Project Shell     done      project folder
  Usage

footer help
```

Rules:

- Put the opened artifact identity at the top as name plus description.
- Do not render generic labels like `Project`, `Current Liner project`, or
  `Mixtape` above the workspace.
- The left pane is a section list, not a command list.
- The left pane shows section names only. Do not add mini-status text such as
  `complete`, `ready`, `1 run`, or `basic` beside section names.
- The right pane is the selected section detail.
- Every section detail starts with one title and one muted description.
- Use tables for section details when the screen is clarifying multiple facts or
  comparing rows. Health, Flow, Sources, Artifacts, Operating Layer, and Usage
  use tables.
- Each section owns one table. If a table needs a different purpose, promote it
  into its own section instead of nesting it under another section.
- Add one blank line between a section title/description and its table.
- Table headers and row values must be left-aligned to the same columns.
- Keep action shortcuts in the footer/help row.
- `Next` on Project should describe the recommended next action, not the
  selected section.
- Before Corpus Ready, `Next` continues Corpus Creation.
- After Corpus Ready, `Next` recommends `Create Operating Layer`.
- After Project Complete, `Next` opens `LINER.md`.
- When `Next` is paired with Enter, Enter must perform that action.

Use this when a screen has one current object and several inspectable sections.

## Pattern 5: File Picker

Use for choosing a local file or folder when typing a path would make the flow
unnecessarily brittle.

Current screen:

- Import Project

Code anchors:

- `packages/go-tui/internal/app/import.go`
- `viewImport`
- `newImportPicker`
- `charm.land/bubbles/v2/filepicker`

Anatomy:

```text
liner v1 ////// import

Import Project
Choose a .mixtape file to import as a Liner project.

Folder:      /Users/arturo/Downloads
Destination: ~/liner/projects
Sources:     Use archived source files

> project.mixtape
  folder/
  notes.txt

footer help
```

Rules:

- Use a real picker component when the user is choosing from the filesystem.
- Filter or disable invalid file types instead of asking the user to remember
  the expected extension.
- Keep implementation nouns such as archive out of the screen title when the
  user is acting on a Liner project.
- Use label/value rows for stable facts such as current folder, destination,
  and import mode.
- Keep path typing out of the primary flow unless there is a deliberate fallback
  mode.
- Footer owns movement, folder navigation, import, refresh, home, back, and
  help keys.
- Disabled files stay gray. The selected valid file uses orange.
- Preserve write behavior in copy: importing creates a project and skips source
  refetch.

Use this when a screen needs filesystem selection without turning the user into
a path parser.

## Pattern 6: Progress Console

Use for a running file or corpus operation where the user needs to see what is
happening, what has completed, and when control returns.

Current screens:

- Compile Console
- Build Corpus
- Create Operating Layer
- Setup: JS rendering
- Clarify Goal
- Import Project

Code anchors:

- `packages/go-tui/internal/app/screen_patterns.go`
- `renderLoadingTitle`
- `renderProgressStatusBlock`
- `newTaskProgressBar`
- `taskProgressWidth`
- `packages/go-tui/internal/app/compile.go`
- `packages/go-tui/internal/app/methodology_view.go`
- `packages/go-tui/internal/app/liner.go`

Anatomy:

```text
liner v1 ////// compile

Compile Console [loader]
Fetch sources, assemble MIXTAPE.md, and report anything that needs attention.

Working  Fetching sources, assembling MIXTAPE.md, and checking the result.
[progress bar]  11/27 sources

body details or log

footer help
```

Rules:

- Use `renderLoadingTitle` whenever the screen is actively waiting on a
  background task. The title loader is the top-level signal that this screen is
  still doing work.
- Use `newTaskProgressBar` and `taskProgressWidth` for the progress bar.
- Use `renderProgressStatusBlock` when the screen needs a status line, detail
  text, progress bar, and count as one block.
- Use a progress bar only when the app has real monotonic progress, such as
  source count, phase count, or artifact steps. Do not use a looping or
  back-and-forth percentage for installs, downloads, or other tasks where the
  app cannot measure progress. For those states, use the title loader plus
  `renderWaitStatusBlock` and plain wait text.
- Name the state in plain language: `Working`, `Project complete`, `Needs
  attention`, or the equivalent product state.
- The detail text says the current operation, such as `Generating LINER.md.` or
  `Fetching sources...`.
- The count describes the real unit: sources, phases, or steps.
- Show row-level progress only when it clarifies what is running or queued.
- Keep visible logs to a compact three-line tail. Preserve scrollback for
  review, but use the title loader and a short status line for quiet running
  states instead of growing the log area.
- Keep action shortcuts out of the body; footer owns controls.
- If the operation completes on this screen, show the completed state and the
  next key explicitly instead of instantly navigating away.
- Compile Console uses this pattern in two modes. The result summary is compact:
  one result, one short source summary, and `Next: View sources` when review is
  needed. The full source table lives in Sources review, not in the result
  summary. Accepted source notes may appear in the summary, but they are
  informational; do not label the compile `compiled with warnings` or prompt
  repair unless there is an actionable source issue.
- Sources review is a navigable table. Bubble source problems to the top, keep
  usable rows scrollable with the screen controls, and show row detail below the
  table. Use clear origin labels such as `research source` and `custom source`.
- Source repair is explicit. `r repair sources` retries unavailable custom
  sources, installs JS rendering first when needed, and stops on a recovery
  review. If content recovered, Enter refreshes source evaluation from
  Evaluation onward without rerunning Candidate discovery. If nothing recovered,
  Enter returns to Sources.
- Regression coverage for source repair must prove the runner starts
  `evaluation`, the progress file resets to `PhaseEvaluation`, and the review UI
  does not say `rebuild corpus` or `from Candidate discovery`.
- Add Sources opened from Compile returns to Compile on save or cancel. It must
  not restart Clarify Goal.

Use this when a screen writes files, runs a multi-step process, or waits for a
background task.

## Adding A Pattern

When a screen settles into a reusable shape:

- Name the pattern by its job, not by the first screen that used it.
- Add the current screen and code anchors here.
- Describe anatomy and rules in product language.
- Extract a helper only when another screen can reuse the mechanics.
- Add a regression test for the pattern if the helper affects wrapping,
  spacing, color, or keyboard behavior.
