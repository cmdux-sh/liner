# Platform Binary Packages

Platform packages are the native-binary side of the `linersh` npm install.
They are optional dependencies of the main package, so users still install one
thing while npm selects the package that matches their OS and CPU.

The `linersh` npm package resolves native binaries from optional platform packages:

- `linersh-darwin-arm64`
- `linersh-darwin-x64`
- `linersh-linux-arm64`
- `linersh-linux-x64`
- `linersh-win32-x64`

Each package contains:

- a PyInstaller build of the CLI at package root as `liner` or `liner.exe`
- a Go TUI build at package root as `liner-tui` or `liner-tui.exe`
- PyInstaller's `_internal/` directory beside the Python CLI executable

## Version Rule

Python or Go TUI changes should ship through all five platform packages at the
same version as `packages/tui/package.json`, followed by the main `linersh`
package.

For future TUI-only releases, leave the optional dependency pins on the latest
platform package version and publish only `linersh`.

## Build Current Platform

Builds are per-platform. PyInstaller does not cross-compile, so build macOS packages on macOS, Linux packages on Linux, and Windows packages on Windows.

From the repo root:

```sh
python3 scripts/build-platform-package.py
```

The script:

1. Runs PyInstaller through `uv` with Python 3.11 and the `binary` optional dependency group.
2. Builds a one-directory CLI bundle that includes the Python `playwright` package.
3. Builds the Go TUI binary for the current platform.
4. Writes `packages/platform/linersh-<platform>-<arch>/package.json`.
4. Runs smoke checks:
   - `liner --version`
   - `liner setup-js --help`
   - `liner-tui --version`

## Validate

```sh
python3 scripts/validate-platform-package.py --pack-dry-run
```

The validator checks package metadata, root executable placement, Unix
executable bits, `_internal/`, command smoke tests, and optionally
`npm pack --dry-run`.

## Local npm Smoke

After building the current platform package:

```sh
python3 scripts/smoke-local-npm-bundle.py
```

This installs the local `packages/tui` package and the generated platform
package into a temporary npm project, then verifies:

- `liner --version` resolves through the npm shim to the platform binary.
- `liner` with no args resolves the Go TUI from the platform package.
- `LINER_BIN=/bin/echo liner setup-js` still honors the debugging override.

## Public GitHub Actions Artifacts

The workflow `.github/workflows/platform-bundles.yml` is intended to run from
the public `cmdux-sh/liner` repository. It builds one selected target at a time
so the release can stay under GitHub Free artifact storage limits.

Manual trigger examples:

```sh
gh workflow run platform-bundles.yml --repo cmdux-sh/liner --ref main -f target=linersh-darwin-arm64
gh workflow run platform-bundles.yml --repo cmdux-sh/liner --ref main -f target=linersh-linux-x64
gh workflow run platform-bundles.yml --repo cmdux-sh/liner --ref main -f target=linersh
```

Download each artifact immediately, delete it from GitHub Actions, then run the
next target. Publish manually only after all five platform package tarballs and
the main `linersh` tarball have been downloaded and inspected.

## Automated Publish

Automated npm publish is not enabled. Publish manually only after the platform
package artifacts and clean consumer smoke pass.

## Historical 0.5.5 Artifact Handoff

The 0.5.5 release includes Python core changes, so all platform packages were rebuilt.

- GitHub Actions workflow: latest post-release-prep `Platform Bundles` run on `master`
- Local artifact directory: `/tmp/liner-release-artifacts-0.5.5`
- Local smoke result: `liner 0.5.5 (tui)  ·  0.5.5 (core)`

Artifacts:

```text
linersh-darwin-arm64-0.5.5.tgz
linersh-darwin-x64-0.5.5.tgz
linersh-linux-arm64-0.5.5.tgz
linersh-linux-x64-0.5.5.tgz
linersh-win32-x64-0.5.5.tgz
linersh-0.5.5.tgz
```

Publish details are paused in `packages/tui/PUBLISH-CHECKLIST.md`.

## Historical Runner Labels

The old artifact workflow used explicit hosted runner labels for each
architecture:

- macOS arm64: `macos-15`
- macOS Intel: `macos-15-intel`
- Linux arm64: `ubuntu-24.04-arm`
- Linux x64: `ubuntu-24.04`
- Windows x64: `windows-2025`
