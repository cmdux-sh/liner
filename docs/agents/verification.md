# Liner verification

Use checks appropriate to the changed surface. Checkpoints preserve useful milestones; the completed PR records validation of its final diff. Repair task-caused failures and rerun affected checks; report unrelated failures separately.

| Change | Relevant checks |
| --- | --- |
| Instructions/documentation | `git diff --check`; check changed links, product terminology, canonical ownership, and privacy boundaries. No provider calls, corpus compile, or application build is needed. |
| Python Core | From the root, use `uv run pytest <test-path>`, `uv run ruff check <changed-paths>`, and `uv run mypy` where types/contracts change. Broaden the offline pytest suite for shared maintenance or source contracts. Network tests remain opt-in. |
| Go TUI | From `packages/go-tui/`, run the affected `go test` packages and `go test ./...` for shared behavior. Follow `../tui/AUTOMATED_VISUAL_ACCEPTANCE.md` for real-binary interface acceptance with isolated fixtures. |
| TypeScript runner or npm shim | `npm --prefix packages/tui run typecheck`, `npm --prefix packages/tui test -- <test-path>`, and `npm --prefix packages/tui run build`. Use `build:go` when the Go binary changes. |
| Development isolation or provider preferences | Use `npm --prefix packages/tui run smoke:runner-preferences` as documented in `../development.md`; it uses fake providers and isolated writable state. Do not substitute Arturo's real library or credentials. |
| Marketing site | `npm --prefix marketing/site run build` with that package's Node version; inspect rendered changes. Run the existing public-surface/command validators when changing those claims. |
| Release/package contents | Follow `../../packages/tui/PUBLISH-CHECKLIST.md` and the relevant packaging validators under their release authority. Ordinary instructions do not require a release run. |

Keep Core's exact Change Set/receipt and canonical-writer checks for changes to those boundaries. Test source evidence and configured model preferences without silently replacing the user's selected model. Do not run live paid provider calls for a documentation check.

Review the final diff for scope, privacy, broken references, and conflicting instructions. Use the shared merge-authority and acceptance rules. Missing or skipped checks are not passing checks. Follow the shared development-server instructions for HTTP previews; a preview must belong to the intended checkout.
