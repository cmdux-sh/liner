#!/usr/bin/env bash
set -euo pipefail

TARGETS=(
  "linersh-darwin-arm64"
  "linersh-darwin-x64"
  "linersh-linux-arm64"
  "linersh-linux-x64"
  "linersh-win32-x64"
  "linersh"
)

WORKFLOW="platform-bundles.yml"
REPO="cmdux-sh/liner"
BRANCH="main"

usage() {
  cat <<'EOF'
Usage: scripts/release-tarballs.sh [--version VERSION] [--dest DIR] [--repo OWNER/REPO] [--branch BRANCH] [--skip-npm-check]

Builds every Liner npm release tarball from the public GitHub Actions workflow,
downloads each artifact into a local versioned directory, deletes the GitHub
Actions artifact, and writes PUBLISH-COMMANDS.md with the exact npm publish
commands Arturo should run.

The script refuses to start unless:
  - the checkout is the public cmdux-sh/liner repo
  - the tracked worktree is clean
  - the current branch matches its upstream
  - package.json, package-lock.json, pyproject.toml, and src/liner/__init__.py agree
  - the target version is not already published for any release package on npm

Examples:
  scripts/release-tarballs.sh --version 1.0.2
  scripts/release-tarballs.sh --version 1.0.3 --dest "$HOME/Desktop/liner-release-1.0.3"
EOF
}

fail() {
  echo "error: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "Missing required command: $1"
}

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[[ -n "$repo_root" ]] || fail "Run this from inside the Liner public checkout."
cd "$repo_root"

version=""
dest=""
skip_npm_check=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || fail "--version requires a value"
      version="$2"
      shift 2
      ;;
    --dest)
      [[ $# -ge 2 ]] || fail "--dest requires a value"
      dest="$2"
      shift 2
      ;;
    --repo)
      [[ $# -ge 2 ]] || fail "--repo requires a value"
      REPO="$2"
      shift 2
      ;;
    --branch)
      [[ $# -ge 2 ]] || fail "--branch requires a value"
      BRANCH="$2"
      shift 2
      ;;
    --skip-npm-check)
      skip_npm_check=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "Unknown argument: $1"
      ;;
  esac
done

require_cmd git
require_cmd gh
require_cmd jq
require_cmd npm
require_cmd node
require_cmd python3
require_cmd shasum
require_cmd tar

package_version="$(node -p "require('./packages/tui/package.json').version")"
if [[ -z "$version" ]]; then
  version="$package_version"
fi
if [[ -z "$dest" ]]; then
  dest="$(cd .. && pwd)/release-tarballs-$version"
fi

remote_url="$(git remote get-url origin 2>/dev/null || true)"
[[ "$remote_url" == *"cmdux-sh/liner"* ]] || fail "origin must be the public cmdux-sh/liner repo, got: $remote_url"

current_branch="$(git branch --show-current)"
[[ "$current_branch" == "$BRANCH" ]] || fail "Expected branch $BRANCH, got $current_branch"

if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
  git status --short --untracked-files=no >&2
  fail "Tracked worktree changes are present. Commit and push before building release tarballs."
fi

upstream="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)"
[[ -n "$upstream" ]] || fail "Current branch has no upstream. Push/set upstream before releasing."
read -r behind ahead < <(git rev-list --left-right --count "$upstream"...HEAD)
[[ "$behind" == "0" && "$ahead" == "0" ]] || fail "Branch is not aligned with $upstream (behind=$behind ahead=$ahead). Push/pull before releasing."

head_sha="$(git rev-parse HEAD)"

echo "Verifying release version $version..."
[[ "$package_version" == "$version" ]] || fail "packages/tui/package.json is $package_version, expected $version"

python3 - "$version" <<'PY'
import pathlib
import re
import sys
try:
    import tomllib
except ModuleNotFoundError:
    import tomli as tomllib

expected = sys.argv[1]
pyproject = tomllib.loads(pathlib.Path("pyproject.toml").read_text())
py_version = pyproject["project"]["version"]
if py_version != expected:
    raise SystemExit(f"pyproject.toml is {py_version}, expected {expected}")

init_text = pathlib.Path("src/liner/__init__.py").read_text()
match = re.search(r'__version__\s*=\s*"([^"]+)"', init_text)
if not match:
    raise SystemExit("src/liner/__init__.py has no __version__")
init_version = match.group(1)
if init_version != expected:
    raise SystemExit(f"src/liner/__init__.py is {init_version}, expected {expected}")
PY

lock_version="$(node -p "const p=require('./packages/tui/package-lock.json'); p.version")"
lock_root_version="$(node -p "const p=require('./packages/tui/package-lock.json'); p.packages[''].version")"
[[ "$lock_version" == "$version" ]] || fail "packages/tui/package-lock.json version is $lock_version, expected $version"
[[ "$lock_root_version" == "$version" ]] || fail "packages/tui/package-lock.json root package is $lock_root_version, expected $version"

if [[ "$skip_npm_check" == "0" ]]; then
  echo "Verifying $version is not already published on npm..."
  for pkg in "${TARGETS[@]}"; do
    if npm view "$pkg@$version" version >/dev/null 2>&1; then
      fail "$pkg@$version already exists on npm. Bump the version before generating tarballs."
    fi
  done
else
  echo "Skipping npm registry absence check."
fi

mkdir -p "$dest"

find_run_id() {
  local started_at="$1"
  for _ in {1..60}; do
    local run_id
    run_id="$(
      gh run list \
        --repo "$REPO" \
        --workflow "$WORKFLOW" \
        --branch "$BRANCH" \
        --event workflow_dispatch \
        --commit "$head_sha" \
        --limit 20 \
        --json databaseId,createdAt,headSha,status,url \
      | jq -r --arg started "$started_at" '
          map(select(.createdAt >= $started))
          | sort_by(.createdAt)
          | reverse
          | .[0].databaseId // empty
        '
    )"
    if [[ -n "$run_id" ]]; then
      echo "$run_id"
      return 0
    fi
    sleep 5
  done
  return 1
}

download_target() {
  local target="$1"
  local target_dir="$dest/$target"
  rm -rf "$target_dir"
  mkdir -p "$target_dir"

  echo
  echo "Dispatching $target..."
  local started_at
  started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  gh workflow run "$WORKFLOW" --repo "$REPO" --ref "$BRANCH" -f "target=$target" >/dev/null

  local run_id
  run_id="$(find_run_id "$started_at")" || fail "Could not find the GitHub Actions run for $target."
  echo "Run: https://github.com/$REPO/actions/runs/$run_id"

  gh run watch "$run_id" --repo "$REPO" --exit-status
  gh run download "$run_id" --repo "$REPO" --name "$target" --dir "$target_dir"

  local tgz
  tgz="$(find "$target_dir" -type f -name "*.tgz" -print -quit)"
  [[ -n "$tgz" ]] || fail "No .tgz downloaded for $target"

  local packed_name
  local packed_version
  packed_name="$(tar -xOf "$tgz" package/package.json | jq -r '.name')"
  packed_version="$(tar -xOf "$tgz" package/package.json | jq -r '.version')"
  [[ "$packed_name" == "$target" ]] || fail "$tgz has package name $packed_name, expected $target"
  [[ "$packed_version" == "$version" ]] || fail "$tgz has version $packed_version, expected $version"

  echo "Downloaded and verified: $tgz"

  local artifact_ids
  artifact_ids="$(
    gh api "repos/$REPO/actions/runs/$run_id/artifacts" \
      --jq ".artifacts[] | select(.name == \"$target\") | .id"
  )"
  if [[ -n "$artifact_ids" ]]; then
    while IFS= read -r artifact_id; do
      [[ -z "$artifact_id" ]] && continue
      gh api -X DELETE "repos/$REPO/actions/artifacts/$artifact_id" >/dev/null
      echo "Deleted GitHub Actions artifact $artifact_id for $target"
    done <<< "$artifact_ids"
  fi
}

for target in "${TARGETS[@]}"; do
  download_target "$target"
done

(
  cd "$dest"
  find . -type f -name "*.tgz" -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS.txt
)

publish_file="$dest/PUBLISH-COMMANDS.md"
cat > "$publish_file" <<EOF
# Liner $version npm publish commands

Generated from public repo \`$REPO\` at commit \`$head_sha\`.

Tarballs are saved at:

\`$dest\`

Run the platform package publishes first. Publish the main \`linersh\` package last.

\`\`\`sh
cd "$dest"

npm publish ./linersh-darwin-arm64/linersh-darwin-arm64-$version.tgz
npm publish ./linersh-darwin-x64/linersh-darwin-x64-$version.tgz
npm publish ./linersh-linux-arm64/linersh-linux-arm64-$version.tgz
npm publish ./linersh-linux-x64/linersh-linux-x64-$version.tgz
npm publish ./linersh-win32-x64/linersh-win32-x64-$version.tgz

npm publish ./linersh/linersh-$version.tgz
\`\`\`

After publish, verify:

\`\`\`sh
npm view linersh@$version version
tmp=\$(mktemp -d)
HOME="\$tmp/home" npm_config_cache="\$tmp/npm-cache" LINER_DIR="\$tmp/projects" npx --yes linersh@$version --version
rm -rf "\$tmp"
\`\`\`
EOF

echo
echo "Release tarballs saved to: $dest"
echo "Checksums: $dest/SHA256SUMS.txt"
echo "Publish commands: $publish_file"
echo
sed -n '/^```sh$/,/^```$/p' "$publish_file"
