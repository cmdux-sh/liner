# Isolated development

Use the isolated development launch when testing Liner from this repository:

```sh
npm --prefix packages/tui run dev:isolated
```

The command builds the navigator, creates one
temporary root, prints every writable Liner path, and launches the normal npm
shim. The temporary root contains:

- `home/.liner/config.yaml`: the development Liner profile;
- `projects/`: the development Project library;
- `provider-homes/codex/`: an empty Codex CLI configuration home;
- `provider-homes/claude/`: an empty Claude Code configuration home.

The root is retained when Liner exits so failures can be inspected. The launch
prints the platform-appropriate cleanup command. It sets `HOME`, Windows
`USERPROFILE`, XDG and Windows application/cache directories, the temporary
directory, npm cache, Playwright browser path, and `LINER_DIR`, so writable
runtime state cannot fall through to the normal user profile. The navigator
doesn't create Projects: your agent's `liner research` writes them under
`LINER_DIR`, so point the agent at the same `LINER_DIR` to keep test liners in
the temporary root. Ambient provider credential variables are removed.
That includes alternate cloud-backend selectors and AWS, Azure, and Google
credential variables. Ambient `LINER_*` overrides are also removed; the launch
lets the local shim
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

## Installed-package release smoke

```sh
npm --prefix packages/tui run acceptance:go -- release-smoke
```

It packs and installs a clean consumer package, then checks that the launcher opens
the navigator and forwards Core commands.
