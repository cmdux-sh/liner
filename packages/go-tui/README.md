# Liner Go TUI

Charm-based terminal interface for Liner. This is the active default TUI. The
previous React/Ink UI is decommissioned; `LINER_TUI=ink` is intentionally
unsupported.

For the current v1 product and Go TUI model, start with
[../../docs/project/PRODUCT.md](../../docs/project/PRODUCT.md) and
[../../docs/project/DESIGN.md](../../docs/project/DESIGN.md).

## Run

From this package:

```sh
go run ./cmd/liner-tui
```

Through the npm shim:

```sh
npm --prefix ../tui run dev:go
```

From the repo root:

```sh
LINER_TUI=go npm --prefix packages/tui run dev:go
```

By default, Liner stores visible Liner Projects in `~/liner/projects`. Override
the projects folder for dev/testing:

```sh
LINER_DIR=/path/to/projects LINER_TUI=go npm --prefix packages/tui run dev:go
```

## Build And Test

```sh
go test ./...
npm --prefix ../tui run build:go
npm --prefix ../tui run build
npm --prefix ../tui run acceptance:go
```

`build:go` writes the Go binary to `packages/tui/bin/liner-tui`. `build` also
builds the TypeScript headless runner that powers the Build Corpus flow.

For real-binary, step-by-step visual testing with isolated fixtures, controlled
keyboard input, screenshots, cold restarts, and bounded automatic repair, use
the [Automated TUI Visual Acceptance And Repair runbook](../../docs/tui/AUTOMATED_VISUAL_ACCEPTANCE.md).

## Current Screen Model

- Home is the selectable/filterable command list.
- Projects is the project browser with a project-name list and compact
  label/value selected details.
- Project is a sectioned workspace with Health, Flow, Sources, Artifacts,
  Operating Layer, and Usage details. Its body starts with the project name and
  description, then uses a names-only section list plus selected-section detail.
- Project action hints live in the global footer/help row, not in a duplicate
  orange row inside the Project body.
- Project details use one title, one muted description, one blank line, and one
  table. Headers are left-aligned to row content. Health uses this table rhythm
  too.
- Project `Next` describes the recommended milestone action: continue Corpus
  Creation, Create Operating Layer, or show complete-project management
  actions.
- Maintain project is the Core-backed Project and Source maintenance surface;
  it renders versioned Change Sets and Change Receipts.
- Build Corpus uses the TypeScript headless runner bridge.
- Compile streams `liner compile --emit-events`.
- Preview remembers its origin screen.
- First Launch introduces the local projects folder and AI runner install check.
- Settings shows OpenAI/Claude provider choices, their independent model
  preferences, and OpenAI Thinking effort. Codex CLI and Claude Code remain the
  technical runtime names.
- Current reusable patterns include Command Hub, Preference Chooser, Split
  Browser, Section Workspace, File Picker, Progress Console, and Maintenance
  Preview.

## Important Packages

- `internal/app`: Bubble Tea model, screens, key handling, and UI rendering.
- `internal/core`: bridge to the Python core binary.
- `internal/agent`: bridge to the TypeScript headless runner used for corpus builds.
- `internal/progress`: progress/gate compatibility with existing projects.
- `internal/source`: source import/staging helpers.
- `internal/tape`: tape read/write helpers.

## Maintenance Authority

Post-creation Project maintenance is delegated to the running Liner Core. The
Go TUI requests a versioned Project Snapshot, sends a versioned maintenance
request, renders the returned Change Set risk/file effects/validation, and
applies only that exact Change Set. Validated additive and metadata operations
continue without a redundant prompt; approval-required risk waits for explicit
confirmation. The TUI then displays the Core receipt, durable receipt path,
stale corpus artifacts, and next actions.

Use `m maintain` from the open Project workspace to inspect immutable Source IDs
and submit one versioned operation for update, replace, remove, purge, rename,
or move. The ordinary Add Sources flow uses the same Core contract for additions.

Do not add new `tape.yaml`, `liner.yaml`, Source-retention, rename, or move
writers to the Go app. Extend `internal/core/maintenance.go` and the Core
contract instead. First-run corpus assembly remains a distinct curation flow.
Initial clarification and first Operating Layer publication are narrowly
allowlisted construction steps. Initial assembly is one-shot: Core creates an
empty construction boundary, the TUI adds accepted Sources through Core, and
later Build Corpus runs cannot replace canonical Sources from a draft.
Composition lineage and reusable `skills/*.md` authoring remain separate
artifact managers; they do not implement Project/Source maintenance
operations. Operating Layer regeneration is hidden until it can use a Core
semantic-review operation.
Legacy composition-draft apply, contradiction-cleanup apply, and production
merge actions are fail-closed until an equivalent Core operation exists; their
review artifacts remain readable and the refusal points to `Maintain project`.

Run the canonical writer boundary check directly with:

```sh
go test ./internal/app -run 'TestCanonicalWriterAudit|TestLegacyCanonicalWriterRefusal'
```

## Gotchas

- If Project Health reports `status failed`, verify which core binary the shim
  resolved. Older bundled cores may not support `liner status --no-write`; the
  Go wrapper fails closed instead of falling back to a writable status command.
- Do not shell out or mutate files from `View()`.
- Keep Home as commands and Projects as the browser.
- Keep footer/help as the single action-hint surface on Project.
