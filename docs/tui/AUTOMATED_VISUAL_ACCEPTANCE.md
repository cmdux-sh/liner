# Automated TUI Visual Acceptance And Repair

This runbook defines Liner's agent-driven acceptance loop for terminal UI work.
It tests the real compiled TUI one interaction at a time, pauses at meaningful
states, captures screenshots and terminal text, diagnoses failures, applies a
bounded code fix, rebuilds, and repeats the same path from a clean fixture.

This complements unit and snapshot tests. It does not replace them, and it is
not a license to update screenshots until a failure disappears.

## What This Test Proves

The loop can provide evidence that:

- the compiled binary starts in an isolated environment;
- navigation reaches the intended screen after each key sequence;
- input belongs to the focused component rather than leaking to global
  shortcuts;
- visible lifecycle language agrees with the Core Project Snapshot;
- long labels, paths, status, and error text remain inside the terminal frame;
- preview-only actions do not mutate canonical Project artifacts;
- stopping and resuming an editor preserves the expected local state;
- quitting and cold-starting the binary reconstructs the correct durable state;
- the acceptance process leaves no test session running.

It does not prove accessibility conformance, behavior in every terminal
emulator, or product correctness outside the exercised path.

## Layers Of Evidence

Use three evidence layers together:

1. **Behavioral control** — a named terminal session receives exact key bytes.
2. **Semantic receipt** — a terminal hardcopy records the text rendered at the
   checkpoint.
3. **Visual receipt** — a screenshot of the terminal window records layout,
   hierarchy, wrapping, selection, and clipping.

A screenshot alone can hide a wrong state. Terminal text alone can miss
overflow, collision, or focus styling. Keep both.

## Required Safety Boundary

Run against a disposable fixture, never a user's live Project.

Set all runtime locations explicitly:

```sh
export ACCEPTANCE_ROOT=/absolute/path/to/disposable-acceptance-run
export ACCEPTANCE_HOME="$ACCEPTANCE_ROOT/home"
export ACCEPTANCE_PROJECTS="$ACCEPTANCE_ROOT/fixtures/projects"
export ACCEPTANCE_CORE=/absolute/path/to/repo/.venv/bin/liner
```

The fixture should contain only the minimum state needed for the scenario. If a
flow previews a semantic or destructive Change Set, capture the preview and
discard it. Do not approve it merely to continue the test.

Before starting, record:

- repository revision and dirty-worktree status;
- binary path and build command;
- Core binary path;
- fixture path and expected initial lifecycle state;
- terminal dimensions;
- output directory for screenshots and text receipts.

## Build And Baseline Checks

Run the automated suites before opening the TUI:

```sh
go test ./...
npm --prefix packages/tui run build:go
npm --prefix packages/tui run build
npm --prefix packages/tui run acceptance:go
git diff --check
```

Use the freshly built `packages/tui/bin/liner-tui`. Do not accept evidence from
an older installed binary or a shell that resolved a different `liner` Core.

## Controlled Terminal Session

On macOS, a detached `screen` session provides deterministic input without
depending on which desktop or application currently has keyboard focus:

```sh
screen -dmS liner-acceptance env \
  HOME="$ACCEPTANCE_HOME" \
  LINER_DIR="$ACCEPTANCE_PROJECTS" \
  LINER_BIN="$ACCEPTANCE_CORE" \
  /absolute/path/to/repo/packages/tui/bin/liner-tui
```

Attach that session to a dedicated Terminal window and record the returned
window ID:

```applescript
tell application "Terminal"
  set acceptanceTab to do script "screen -x liner-acceptance"
  delay 1
  set acceptanceWindow to first window whose selected tab is acceptanceTab
  set bounds of acceptanceWindow to {80, 36, 926, 673}
  return id of acceptanceWindow
end tell
```

Capturing the window by ID is important. Region capture can accidentally record
another app when Terminal lives on a different macOS Space.

## Step, Stop, Capture

Send one logical interaction, wait for rendering to settle, then capture both
the terminal text and the window.

```sh
# Down, then Enter
screen -S liner-acceptance -p 0 -X stuff $'\033[B\r'
sleep 1

# Semantic receipt
screen -S liner-acceptance -p 0 -X hardcopy -h "$ACCEPTANCE_ROOT/state.txt"

# Visual receipt; WINDOW_ID comes from Terminal
screencapture -x -l "$WINDOW_ID" \
  "$ACCEPTANCE_ROOT/02-stop-projects.png"
```

Useful control bytes:

| Interaction | Bytes |
| --- | --- |
| Enter | `$'\r'` |
| Escape | `$'\033'` |
| Tab | `$'\t'` |
| Down | `$'\033[B'` |
| Up | `$'\033[A'` |
| Ctrl+D | `$'\004'` |
| Ctrl+C | `$'\003'` |

Do not send a long burst of unrelated keys. Event timing can conceal which
transition failed. Pause after every state whose behavior or layout matters.

## Checkpoint Contract

For every checkpoint, record:

- sequence number and short name;
- keys sent from the previous checkpoint;
- expected screen and selected element;
- required visible text;
- forbidden visible text or navigation;
- expected write behavior: none, preview only, or approved mutation;
- screenshot path and semantic receipt path;
- pass/fail and the reason.

Use stable names such as:

```text
01-start-home.png
02-stop-projects.png
03-resume-project.png
04-stop-review.png
05-start-skill-edit.png
06-stop-printables-owned.png
07-resume-edit-finished.png
08-stop-change-set-preview.png
09-cold-restart-home.png
10-cold-restart-projects.png
11-cold-restart-project.png
12-final-stop.png
```

The exact screens vary by scenario, but the naming should make the event
sequence readable without opening each image.

## High-Value Scenarios

Always include these when the changed code can affect them:

- **Input ownership:** start an editor, type printable global shortcuts such as
  `h` and `?`, and prove they remain text rather than opening Home or Help.
- **Editor lifecycle:** prove the key that starts editing is not inserted into
  the document, then prove Ctrl+D finishes editing without exiting the TUI.
- **Preview safety:** render the exact Core Change Set, inspect risk, writes,
  hashes, and validation, then discard it without applying the test revision.
- **Lifecycle language:** compare Project browser status, Project Health,
  primary action, completed-step count, and missing-next text with the Core
  Snapshot.
- **Wrapping:** exercise long unbroken paths and errors at the minimum supported
  width.
- **Cold restart:** terminate the binary, start a new process with the same
  fixture, and prove durable state is reconstructed correctly.
- **Final cleanup:** terminate the controlled session and confirm the named
  session no longer exists.

## Automated Diagnosis And Repair Loop

When a checkpoint fails:

1. Preserve the failed screenshot and semantic receipt. Never overwrite the
   first failure evidence.
2. Classify the failure as input routing, state transition, copy/semantics,
   wrapping/layout, lifecycle reconstruction, Core contract, or capture
   infrastructure.
3. Reproduce it with the smallest focused Go test when practical.
4. Patch the smallest responsible boundary. Do not alter unrelated screens or
   weaken the expected checkpoint.
5. Run the focused test, then the full relevant suite.
6. Rebuild the binary.
7. Recreate or reset the disposable fixture if the scenario may have written
   state.
8. Replay the complete path from checkpoint 01, not only the formerly failing
   step.
9. Preserve the passing screenshots separately and generate a contact sheet for
   human review.

An agent may perform this repair loop automatically only while the expected
behavior is already defined by product docs, Core contracts, tests, or explicit
user approval. Stop for human judgment when the failure exposes a new product
decision.

## Rules For Screenshot Baselines

- Never update a baseline as the first response to a mismatch.
- Treat copy, selected state, footer actions, and lifecycle language as semantic
  assertions, not pixel noise.
- Prefer structural and semantic checks over exact whole-image equality.
- Allow explicitly documented variation for font rasterization, window chrome,
  and terminal-emulator decoration.
- Require review before accepting a new layout, color hierarchy, or product
  noun as the baseline.
- Keep failed, before-fix, and after-fix evidence distinguishable.

## Stop Conditions

Stop automatic repair and request direction when:

- the expected product behavior is ambiguous or contradicted by active docs;
- proceeding requires applying a destructive or semantic test Change Set;
- the target is not a disposable fixture;
- the failure belongs to Core or another authority outside the approved change;
- a screenshot difference represents a new design choice rather than a defect;
- the same fix attempt fails repeatedly without narrowing the cause;
- the test would touch credentials, network accounts, production projects, or
  unrelated user files.

## Completion Receipt

The final acceptance record should include:

- code and documentation files changed;
- focused and full test results;
- binary and Core paths used;
- fixture identity;
- ordered checkpoint table;
- individual screenshot paths;
- one contact sheet;
- any intentionally unapplied Change Set;
- cold-restart result;
- confirmation that the controlled session stopped;
- remaining untested boundaries.

A passing contact sheet is evidence for the exercised path, not a declaration
that the entire TUI is correct.
