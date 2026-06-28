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
- Build Corpus uses the TypeScript headless runner bridge.
- Compile streams `liner compile --emit-events`.
- Preview remembers its origin screen.
- First Launch introduces the local projects folder and AI runner install check.
- Settings shows Codex/Claude choices with simple Status/Runner detail rows and
  no config/env details.
- Current reusable patterns are Command Hub, Preference Chooser, and Split
  Browser.

## Important Packages

- `internal/app`: Bubble Tea model, screens, key handling, and UI rendering.
- `internal/core`: bridge to the Python core binary.
- `internal/agent`: bridge to the TypeScript headless runner used for corpus builds.
- `internal/progress`: progress/gate compatibility with existing projects.
- `internal/source`: source import/staging helpers.
- `internal/tape`: tape read/write helpers.

## Gotchas

- If Project Health reports `status failed`, verify which core binary the shim
  resolved. Older bundled cores may not support `liner status --no-write`; the
  Go wrapper now falls back to `liner status <project> --json`.
- Do not shell out or mutate files from `View()`.
- Keep Home as commands and Projects as the browser.
- Keep footer/help as the single action-hint surface on Project.
