#!/usr/bin/env bash
#
# Liner — clean uninstaller.
#
# Removes the `linersh` CLI, the optional Playwright browser, the local
# `~/.liner/` cache + config, and (optionally) any globally installed TUI and
# the repo's dev `.venv`. Mixtape workspaces in `$LINER_DIR` / `~/liner-workspace`
# are left alone unless you pass --purge-workspace.
#
# Usage:
#   ./scripts/uninstall.sh                # interactive, prompts before each step
#   ./scripts/uninstall.sh --yes          # no prompts
#   ./scripts/uninstall.sh --dry-run      # show what would happen, change nothing
#   ./scripts/uninstall.sh --dev          # also remove the repo's .venv + egg-info
#   ./scripts/uninstall.sh --purge-workspace  # also delete ~/liner-workspace
#
# Compatible with macOS and Linux. Windows users — see README.md "Uninstalling".

set -euo pipefail

ASSUME_YES=0
DRY_RUN=0
DEV_CLEANUP=0
PURGE_WORKSPACE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -y|--yes) ASSUME_YES=1 ;;
    -n|--dry-run) DRY_RUN=1 ;;
    --dev) DEV_CLEANUP=1 ;;
    --purge-workspace) PURGE_WORKSPACE=1 ;;
    -h|--help)
      sed -n '2,18p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown flag: $1" >&2
      exit 2
      ;;
  esac
  shift
done

# --- Pretty output ----------------------------------------------------------

if [[ -t 1 ]]; then
  C_RED=$'\033[31m'
  C_YEL=$'\033[33m'
  C_GRN=$'\033[32m'
  C_DIM=$'\033[2m'
  C_OFF=$'\033[0m'
else
  C_RED=""; C_YEL=""; C_GRN=""; C_DIM=""; C_OFF=""
fi

say() { printf '%s\n' "$*"; }
info() { printf '%s%s%s\n' "$C_DIM" "$*" "$C_OFF"; }
note() { printf '%s%s%s\n' "$C_YEL" "$*" "$C_OFF"; }
ok() { printf '%s✓%s %s\n' "$C_GRN" "$C_OFF" "$*"; }
warn() { printf '%s!%s %s\n' "$C_RED" "$C_OFF" "$*"; }

confirm() {
  # confirm "Prompt text" → 0 if yes, 1 if no
  local prompt="$1"
  if (( ASSUME_YES )); then
    info "  (auto-yes) $prompt"
    return 0
  fi
  local reply
  printf '%s [y/N] ' "$prompt"
  read -r reply </dev/tty || reply=""
  case "$reply" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

run() {
  # run <cmd> [args...]
  # Honors DRY_RUN; prints what it's about to do.
  if (( DRY_RUN )); then
    printf '%s$ %s%s\n' "$C_DIM" "$*" "$C_OFF"
    return 0
  fi
  "$@"
}

# --- Detection --------------------------------------------------------------

LINER_DIR_OVERRIDE="${LINER_DIR:-}"
DEFAULT_WORKSPACE="$HOME/liner-workspace"
PIPX_VENV_DIR="$HOME/.local/pipx/venvs/linersh"

case "$(uname -s)" in
  Darwin) PLAYWRIGHT_CACHE_DIR="$HOME/Library/Caches/ms-playwright" ;;
  Linux)  PLAYWRIGHT_CACHE_DIR="$HOME/.cache/ms-playwright" ;;
  *)      PLAYWRIGHT_CACHE_DIR="" ;;
esac

# --- Header -----------------------------------------------------------------

say ""
say "Liner uninstaller"
say "================="
if (( DRY_RUN )); then
  note "Dry-run mode — no files will be modified."
fi
say ""

# --- 1. Playwright browser (must run before pipx uninstall) ----------------

say "1. Playwright browser"
if [[ -x "$PIPX_VENV_DIR/bin/playwright" ]]; then
  info "  Found Playwright in $PIPX_VENV_DIR/bin/"
  if confirm "  Remove the Chromium binary Playwright manages?"; then
    if run "$PIPX_VENV_DIR/bin/playwright" uninstall --all; then
      ok "Playwright browser removed."
    else
      warn "playwright uninstall exited non-zero; we'll fall back to removing the cache directory below."
    fi
  else
    info "  Skipped."
  fi
else
  info "  No Playwright install found (the [js] extra was never installed). Skipping."
fi
say ""

# --- 2. The CLI itself ------------------------------------------------------

say "2. Liner CLI"
if command -v pipx >/dev/null 2>&1 && pipx list --short 2>/dev/null | grep -q '^linersh'; then
  if confirm "  Run 'pipx uninstall linersh'?"; then
    run pipx uninstall linersh
    ok "CLI uninstalled."
  else
    info "  Skipped."
  fi
else
  info "  No pipx-installed 'linersh' found. Skipping."
  # Check for pip installs too, just in case.
  if command -v pip >/dev/null 2>&1 && pip show linersh >/dev/null 2>&1; then
    note "  Heads-up: 'pip show linersh' returns a package. If you installed via plain pip,"
    note "  remove it manually: pip uninstall linersh"
  fi
fi
say ""

# --- 3. Local cache + config -----------------------------------------------

say "3. Liner local cache + config (~/.liner/)"
if [[ -d "$HOME/.liner" ]]; then
  size=$(du -sh "$HOME/.liner" 2>/dev/null | awk '{print $1}')
  if confirm "  Delete ~/.liner (cache.db, config.toml; about ${size:-unknown})?"; then
    run rm -rf "$HOME/.liner"
    ok "Local cache removed."
  else
    info "  Skipped."
  fi
else
  info "  ~/.liner does not exist. Skipping."
fi
say ""

# --- 4. Leftover Playwright cache ------------------------------------------

say "4. Playwright browser cache (leftover)"
if [[ -n "$PLAYWRIGHT_CACHE_DIR" && -d "$PLAYWRIGHT_CACHE_DIR" ]]; then
  size=$(du -sh "$PLAYWRIGHT_CACHE_DIR" 2>/dev/null | awk '{print $1}')
  if confirm "  Delete $PLAYWRIGHT_CACHE_DIR (about ${size:-unknown})?"; then
    run rm -rf "$PLAYWRIGHT_CACHE_DIR"
    ok "Playwright cache removed."
  else
    info "  Skipped."
  fi
else
  info "  No leftover Playwright cache directory. Skipping."
fi
say ""

# --- 5. Globally installed TUI (npm) ---------------------------------------

say "5. Globally installed TUI (npm)"
if command -v npm >/dev/null 2>&1; then
  if npm list -g --depth=0 linersh >/dev/null 2>&1; then
    if confirm "  Run 'npm uninstall -g linersh'?"; then
      run npm uninstall -g linersh
      ok "Global TUI uninstalled."
    else
      info "  Skipped."
    fi
  else
    info "  No global 'linersh' npm install found. Skipping."
  fi
else
  info "  npm not on PATH. Skipping."
fi

# npx package cache (used when you ran `npx linersh`)
NPX_DIR=""
if [[ -d "$HOME/.npm/_npx" ]]; then
  NPX_DIR="$HOME/.npm/_npx"
fi
if [[ -n "$NPX_DIR" ]]; then
  if confirm "  Clear npx package cache at $NPX_DIR? (affects only npx-cached copies of any package, not just linersh)"; then
    run rm -rf "$NPX_DIR"
    ok "npx cache cleared."
  else
    info "  Skipped."
  fi
fi
say ""

# --- 6. Dev install (repo .venv) -------------------------------------------

if (( DEV_CLEANUP )); then
  say "6. Repo dev install (--dev)"
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  repo_root="$(cd "$script_dir/.." && pwd)"
  if [[ -d "$repo_root/.venv" ]]; then
    if confirm "  Remove $repo_root/.venv?"; then
      run rm -rf "$repo_root/.venv"
      ok "Dev venv removed."
    else
      info "  Skipped."
    fi
  else
    info "  No .venv in repo root. Skipping."
  fi
  if [[ -d "$repo_root/src/linersh.egg-info" ]]; then
    if confirm "  Remove $repo_root/src/linersh.egg-info?"; then
      run rm -rf "$repo_root/src/linersh.egg-info"
      ok "egg-info removed."
    else
      info "  Skipped."
    fi
  fi
  say ""
fi

# --- 7. Mixtape workspace (--purge-workspace only) -------------------------

if (( PURGE_WORKSPACE )); then
  say "7. Mixtape workspace (--purge-workspace)"
  workspaces=()
  [[ -d "$DEFAULT_WORKSPACE" ]] && workspaces+=("$DEFAULT_WORKSPACE")
  if [[ -n "$LINER_DIR_OVERRIDE" && -d "$LINER_DIR_OVERRIDE" && "$LINER_DIR_OVERRIDE" != "$DEFAULT_WORKSPACE" ]]; then
    workspaces+=("$LINER_DIR_OVERRIDE")
  fi
  if [[ ${#workspaces[@]} -eq 0 ]]; then
    info "  No workspaces found. Skipping."
  else
    for ws in "${workspaces[@]}"; do
      size=$(du -sh "$ws" 2>/dev/null | awk '{print $1}')
      warn "  This will delete your mixtape data: $ws (about ${size:-unknown})"
      if confirm "  Really delete $ws?"; then
        run rm -rf "$ws"
        ok "Workspace removed."
      else
        info "  Skipped."
      fi
    done
  fi
  say ""
fi

# --- Verification -----------------------------------------------------------

say "Verification"
say "------------"
if command -v liner >/dev/null 2>&1; then
  note "  'liner' is still on PATH at $(command -v liner)"
  note "  (may be a stale shim; try 'hash -r' or open a new terminal)"
else
  ok "'liner' is not on PATH."
fi
if [[ -d "$HOME/.liner" ]]; then
  note "  ~/.liner still exists."
else
  ok "~/.liner is gone."
fi
if [[ -n "$PLAYWRIGHT_CACHE_DIR" && -d "$PLAYWRIGHT_CACHE_DIR" ]]; then
  note "  $PLAYWRIGHT_CACHE_DIR still exists."
else
  ok "Playwright cache is clear."
fi

say ""
say "Done. To reinstall:"
say "  pipx install linersh             # CLI only"
say "  pipx install 'linersh[js]'       # CLI with headless-browser support"
say "  npm install -g linersh           # TUI"
say ""
