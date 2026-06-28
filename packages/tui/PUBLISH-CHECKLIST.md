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

1. Confirm the repo is clean and all version files agree.
2. Sync the release commit to the public `cmdux-sh/liner` repository.
3. Build each release tarball one at a time from the public repo's manual
   `Platform Bundles` workflow:
   - `linersh-darwin-arm64`
   - `linersh-darwin-x64`
   - `linersh-linux-arm64`
   - `linersh-linux-x64`
   - `linersh-win32-x64`
   - `linersh`
4. Download each tarball immediately, then delete the GitHub Actions artifact
   before building the next target.
5. Publish every `linersh-<platform>-<arch>` package for the release version.
6. Publish the main `linersh` package after the platform packages exist.
7. Smoke a clean consumer install with `npx linersh --version` and `npx linersh`
   on at least macOS arm64, Linux x64, and Windows x64.
8. Run clean-user smoke tests with temporary `HOME`, `npm_config_cache`, and
   `LINER_DIR` values. Remove the temp directory when done so the verification
   leaves no npm cache, Liner config, Playwright cache, or test project behind.

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

For non-isolated one-shot installs, use `npx --yes linersh@latest uninstall --yes`
to remove Liner's local cache, Playwright's Chromium cache, and npm's `_npx`
execution cache. For global installs, also run `npm uninstall -g linersh`.

Historical release artifacts from the cleanup are outside the repo at:

`/Users/arturo/Documents/Projects/0-archive/liner-archive/2026-06-25-root-cleanup/package-artifacts/`
