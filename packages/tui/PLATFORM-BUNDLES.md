# Platform Binary Packages

The `linersh` npm package resolves the Python core from optional platform packages:

- `linersh-darwin-arm64`
- `linersh-darwin-x64`
- `linersh-linux-arm64`
- `linersh-linux-x64`
- `linersh-win32-x64`

Each package contains a PyInstaller build of the CLI. The executable must live at the package root as `liner` or `liner.exe`; PyInstaller's `_internal/` directory sits beside it.

## Version Rule

When Python source changes need to ship through npm, publish all five platform packages at the same version as `packages/tui/package.json`, then publish the main `linersh` package.

For TUI-only releases, leave the optional dependency pins on the latest platform package version and publish only `linersh`.

## Build Current Platform

Builds are per-platform. PyInstaller does not cross-compile, so build macOS packages on macOS, Linux packages on Linux, and Windows packages on Windows.

From the repo root:

```sh
python3 scripts/build-platform-package.py
```

The script:

1. Runs PyInstaller through `uv` with Python 3.11 and the `binary` optional dependency group.
2. Builds a one-directory CLI bundle that includes the Python `playwright` package.
3. Writes `packages/platform/linersh-<platform>-<arch>/package.json`.
4. Runs smoke checks:
   - `liner --version`
   - `liner setup-js --help`

## Validate

```sh
python3 scripts/validate-platform-package.py --pack-dry-run
```

The validator checks package metadata, root executable placement, Unix executable bits, `_internal/`, command smoke tests, and optionally `npm pack --dry-run`.

## Local npm Smoke

After building the current platform package:

```sh
python3 scripts/smoke-local-npm-bundle.py
```

This installs the local `packages/tui` package and the generated platform package into a temporary npm project, then verifies:

- `liner --version` resolves through the npm shim to the platform binary.
- `LINER_BIN=/bin/echo liner setup-js` still honors the debugging override.

## GitHub Actions Artifacts

The workflow `.github/workflows/platform-bundles.yml` builds and validates all five platform packages plus the main TUI package.

On pushes to `master`, `main`, or `codex/**`, it builds and uploads short-lived tarball artifacts.

Manual artifact build:

```sh
gh workflow run platform-bundles.yml -f publish_to_npm=false -f npm_tag=next
gh run list --workflow platform-bundles.yml --limit 1
gh run watch <run-id> --exit-status
gh run download <run-id> --dir /tmp/liner-release-artifacts-<version>
```

Manual npm publish from downloaded artifacts:

```sh
cd /tmp/liner-release-artifacts-<version>
npm publish ./linersh-darwin-arm64/*.tgz --tag next --access public
npm publish ./linersh-darwin-x64/*.tgz --tag next --access public
npm publish ./linersh-linux-arm64/*.tgz --tag next --access public
npm publish ./linersh-linux-x64/*.tgz --tag next --access public
npm publish ./linersh-win32-x64/*.tgz --tag next --access public
npm publish ./linersh-tui/*.tgz --tag next --access public
```

Use explicit local paths with `./`; otherwise npm can parse a tarball path as a package or GitHub spec.

First-time publishers may need account-level 2FA for publish writes. npm login email OTPs are not the same as publish 2FA; passkey/authenticator verification may be required during `npm publish`.

## Automated Publish

The same workflow can publish from CI on manual dispatch when `publish_to_npm=true` and an `NPM_TOKEN` repository secret exists. It publishes platform tarballs first, then the main TUI tarball.

Use this only after the manual artifact path is trusted for the release. Manual publish is easier to recover from when npm asks for 2FA or rejects a package.

## Runner Labels

The workflow uses explicit hosted runner labels for each architecture:

- macOS arm64: `macos-15`
- macOS Intel: `macos-15-intel`
- Linux arm64: `ubuntu-24.04-arm`
- Linux x64: `ubuntu-24.04`
- Windows x64: `windows-2025`
