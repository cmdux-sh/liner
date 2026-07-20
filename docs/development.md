# Isolated development and runner preferences

Use the isolated development launch when testing Liner from this repository:

```sh
npm --prefix packages/tui run dev:isolated
```

The command builds the current TypeScript runner and Go TUI, creates one
temporary root, prints every writable Liner path, and launches the normal npm
shim. The temporary root contains:

- `home/.liner/config.yaml`: the development Liner profile;
- `projects/`: the development Project library;
- `provider-homes/codex/`: an empty Codex CLI configuration home;
- `provider-homes/claude/`: an empty Claude Code configuration home.

The root is retained when Liner exits so failures can be inspected. The launch
prints the platform-appropriate cleanup command. It sets `HOME`, Windows
`USERPROFILE`, XDG and Windows application/cache directories, the temporary
directory, npm cache, Playwright browser path, and `LINER_DIR`, so Project
creation and writable runtime state cannot fall through to the normal user
profile. Ambient provider credential variables are removed.
That includes alternate cloud-backend selectors and AWS, Azure, and Google
credential variables. Ambient `LINER_*` overrides are also removed; the launch
pins the freshly built repository headless runner and lets the local shim
resolve the repository Core.

## Optional authenticated runtime profiles

An existing authenticated runtime profile is used only through an explicit
flag:

```sh
npm --prefix packages/tui run dev:isolated -- --codex-home /absolute/path/to/codex-home
npm --prefix packages/tui run dev:isolated -- --claude-home /absolute/path/to/claude-home
```

The selected directory is referenced in place through `CODEX_HOME` or
`CLAUDE_CONFIG_DIR`. Liner does not copy credentials or provider state. The
Liner profile and Project library remain inside the temporary root. Without an
explicit flag, ambient provider-home variables are replaced with empty
temporary directories.

Use `--dry-run` to print and prepare the boundary without launching. A stable
diagnostic root can be requested with `--root /absolute/path`, but the path
must not already exist. Liner creates and marks the root itself, refuses
pre-existing contents and managed-path symlinks, and only then advertises the
recursive cleanup command.

## Automated clean-room proof

Run the offline product smoke with:

```sh
npm --prefix packages/tui run smoke:runner-preferences
```

It builds the local package, launches the real npm shim and Go TUI inside a
temporary root, and then drives the real Settings model, Clarify Job launcher,
and TypeScript methodology bridge with fake Codex CLI and Claude Code
executables. It verifies independent OpenAI and Claude model preferences,
OpenAI Thinking effort, restart persistence, native runtime arguments and
configuration-home environment, run logs, generated artifacts, and filesystem
containment. The build and smoke-test compile enforce `GOPROXY=off`; the run
requires no credentials and makes no provider or network calls.
Before/after metadata checks prove the normal Liner config and Project library
were not created or modified. Evidence is retained under the printed root.

This is distinct from the installed-package release smoke:

```sh
npm --prefix packages/tui run acceptance:go -- release-smoke
```

The release smoke packs and installs a clean consumer package. The runner
preferences smoke proves repository development isolation and AI invocation
behavior.

## Provider preference contract

OpenAI is the provider and Codex CLI is its runtime. Claude is the provider and
Claude Code is its runtime. Backward-compatible configuration keys continue to
use the internal runtime identifiers:

```yaml
provider_preferences:
  codex:
    model: gpt-5.6-sol
    reasoning_effort: max
  claude:
    model: opus
```

Missing fields mean provider defaults. Model precedence is explicit per-phase
override, provider preference, existing built-in phase default, then runtime
default. OpenAI reasoning effort has no per-phase layer: an explicit provider
preference wins, otherwise Codex CLI decides.

Curated model choices have known capability metadata. A custom model ID is
preserved as entered after whitespace trimming. A custom OpenAI model resets
Thinking effort to Model default; choosing an explicit effort afterward is
allowed but compatibility is marked unverified and OpenAI remains
authoritative. Unknown configuration fields and legacy per-phase model maps are
preserved by Settings writes.

Model and Thinking effort overrides apply only to fresh runs. Resumed Codex CLI
or Claude Code sessions retain the model and effort they started with. An
explicit provider rejection is shown as the primary failure and is never
silently replaced with another model or effort. Full private runner logs remain
under the Project's `.liner-runs/` directory for diagnosis.
