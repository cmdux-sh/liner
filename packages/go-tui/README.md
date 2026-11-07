# Liner terminal navigator

A live, read-only view of your library, built with Charm. Agents do the work in
chat, and this app shows what they leave behind. The reasons are in
[the decision log](../../docs/decisions.md), and the plan is in
[the navigator plan](../../docs/project/TUI-NAVIGATOR-PLAN.md).

## Run

```sh
go run ./cmd/liner-tui
```

`LINER_DIR` picks the library. `LINER_BIN` picks the Core binary, otherwise it uses
the bundled one, the repository's `.venv`, then `liner` on `PATH`.

## How it's built

- `internal/navigator/data.go` reads Core: `liner library`, `liner search`, and
  `liner handoff`. Core output is stripped of control characters, because titles
  and snippets come from web pages.
- `internal/navigator/model.go` holds a back stack of pages and polls every 3
  seconds, one load at a time.
- `internal/navigator/view.go` draws each page. Lists are rows of columns, and a
  row with a target can be selected.
- `internal/navigator/reader.go` lays out files and finds citations.

The app changes nothing in a liner. The only write it starts is
`liner agent-skill install`, from first run and Settings.

## Test

```sh
go test ./...
```
