# Liner NPM Release Checklist

This is the v1 npm release path for the Go TUI package shape. Do not use the
old Ink-era tarball flow.

The install command remains one package for users:

```sh
npx linersh
npm install -g linersh
```

Behind that, npm installs the matching optional platform package for the
machine. Each platform package contains both native binaries:

- `liner` or `liner.exe`: Python core CLI bundle
- `liner-tui` or `liner-tui.exe`: Go TUI bundle

## Current Active Package Shape

- `packages/tui` remains the `linersh` npm package shell.
- `bin/liner.js` launches the Go TUI and forwards CLI subcommands.
- `packages/go-tui` is the active terminal UI implementation.
- `scripts/build-platform-package.py` builds the current platform package with
  both native binaries.
- `docs/curation-skill` is copied into `packages/tui/cli-update-docs` during
  packaging so the headless runner can find the bundled methodology files.
- The old React/Ink package shape is historical and should not be published.

## Release Order

1. Bump the release version everywhere:
   - `packages/tui/package.json`
   - `packages/tui/package-lock.json`
   - `pyproject.toml`
   - `src/liner/__init__.py`
   - user-facing docs that name the current release
2. Run the verification suite and commit the version/release changes.
3. Push the release commit to the public `cmdux-sh/liner` repository.
4. Generate tarballs from the public repo with the release helper:

   ```sh
   VERSION=1.0.2
   scripts/release-tarballs.sh --version "$VERSION"
   ```

   The helper is the default path. It verifies the public repo, checks that all
   version files match, confirms the branch is pushed to its upstream, confirms
   the target version is absent on npm, runs the public GitHub Actions bundle
   workflow once per target, downloads every tarball into a local versioned
   folder, deletes the GitHub artifact after each download, writes checksums,
   and writes `PUBLISH-COMMANDS.md`.
5. Give Arturo the generated `PUBLISH-COMMANDS.md` code block verbatim, including
   the absolute local tarball path. Do not hand-write or summarize the publish
   commands from memory.
6. Publish every `linersh-<platform>-<arch>` package for the release version.
7. Publish the main `linersh` package after the platform packages exist.
8. Verify the registry and run a clean external-user smoke test.

Never reuse an npm version. If any package in the release set already exists on
npm, stop, bump the version, regenerate the tarballs, and publish the new
version.

Minimum checks before publishing:

- Python test suite from the repo root.
- Go tests from `packages/go-tui`.
- TypeScript typecheck, tests, and package build from `packages/tui`.
- `npm run acceptance:go -- release-smoke` from `packages/tui`.
- Explicit validation that `liner` opens the Go TUI by default.
- `npm audit --omit=dev` and full `npm audit` for `packages/tui`.

Clean-user smoke pattern:

```sh
tmp=$(mktemp -d)
home="$tmp/home"
cache="$tmp/npm-cache"
projects="$tmp/projects"
mkdir -p "$home" "$cache" "$projects"

HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes linersh@latest --version
HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes linersh@latest

rm -rf "$tmp"
```

Hard-reset smoke pattern for a real machine or a throwaway user account:

```sh
npx --yes linersh@latest uninstall --yes || true
liner uninstall --yes || true

npm uninstall -g linersh || true
npm uninstall -g \
  linersh-darwin-arm64 \
  linersh-darwin-x64 \
  linersh-linux-arm64 \
  linersh-linux-x64 \
  linersh-win32-x64 || true

rm -rf ~/.npm/_npx
rm -rf ~/.liner
rm -rf ~/Library/Caches/ms-playwright
rm -rf ~/.cache/ms-playwright
npm cache clean --force
```

After the reset, test as an external user with an explicit version:

```sh
VERSION=1.0.2
tmp=$(mktemp -d)
home="$tmp/home"
cache="$tmp/npm-cache"
projects="$tmp/projects"
mkdir -p "$home" "$cache" "$projects"

HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes "linersh@$VERSION" --version
HOME="$home" npm_config_cache="$cache" LINER_DIR="$projects" npx --yes "linersh@$VERSION"

rm -rf "$tmp"
```

Use the hard reset only when it is acceptable to remove local Liner config,
npx caches, global installs, and Playwright Chromium. The isolated temporary
`HOME` pattern is safer for repeatable verification because it leaves the real
machine alone.

For non-isolated one-shot installs, use `npx --yes linersh@latest uninstall --yes`
to remove Liner's local cache, Playwright's Chromium cache, and npm's `_npx`
execution cache. For global installs, also run `npm uninstall -g linersh`.

Historical release artifacts from the cleanup are outside the repo at:

`/Users/arturo/Documents/Projects/0-archive/liner-archive/2026-06-25-root-cleanup/package-artifacts/`
