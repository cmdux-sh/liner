#!/bin/sh
# Install this checkout as a local test copy of `liner` and `liner-tui`.
#
#   scripts/install-local-copy.sh
#
# LINER_LOCAL_ENV is a Python virtualenv that already has Liner installed
# (default ~/.local/share/liner-test). LINER_LOCAL_BIN is where `liner-tui`
# goes (default ~/.local/bin).
#
# The new `liner-tui` is moved into place rather than copied over the old one:
# macOS kills a signed binary whose file was overwritten in place.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
env_dir=${LINER_LOCAL_ENV:-"$HOME/.local/share/liner-test"}
bin_dir=${LINER_LOCAL_BIN:-"$HOME/.local/bin"}

if [ ! -x "$env_dir/bin/python" ]; then
  echo "No virtualenv at $env_dir. Set LINER_LOCAL_ENV." >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go isn't on PATH, so liner-tui can't be built. Nothing was installed." >&2
  exit 1
fi

"$env_dir/bin/python" -m pip install --quiet --no-deps --force-reinstall "$repo"

mkdir -p "$bin_dir"
staged=$(mktemp "$bin_dir/.liner-tui.XXXXXX")
trap 'rm -f "$staged"' EXIT
(cd "$repo/packages/go-tui" &&
  go build -trimpath -ldflags "-X main.version=$(cat "$repo/VERSION")" -o "$staged" ./cmd/liner-tui)
chmod 755 "$staged"
mv -f "$staged" "$bin_dir/liner-tui"
trap - EXIT

"$env_dir/bin/liner" --version
"$bin_dir/liner-tui" --version
commit=$(git -C "$repo" rev-parse --short HEAD 2>/dev/null || echo "an unknown commit")
if [ -n "$(git -C "$repo" status --porcelain 2>/dev/null || true)" ]; then
  commit="$commit with uncommitted changes"
fi
echo "Installed from $commit in $repo."
