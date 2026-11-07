#!/usr/bin/env bash
#
# Liner — clean uninstaller.
#
# Removes the `linersh` CLI, the optional Playwright browser, the local
# `~/.liner/` cache + config, and (optionally) any globally installed TUI and
# the repo's dev `.venv`. Your liners are left alone unless you pass
# --purge-library.
#
# Usage:
#   ./scripts/uninstall.sh                # interactive, prompts before each step
#   ./scripts/uninstall.sh --yes          # no prompts
#   ./scripts/uninstall.sh --dry-run      # show what would happen, change nothing
#   ./scripts/uninstall.sh --dev          # also remove the repo's .venv + egg-info
#   ./scripts/uninstall.sh --purge-library    # also delete your liners (asks you to type how many)
#
# --purge-library deletes the liners in your library: $LINER_DIR if set, else
# projects_dir in ~/.liner/config.yaml, else ~/liner/projects, plus the older
# ~/liner-workspace. Only liner folders are deleted, and it always asks you to
# type how many, even with --yes. --purge-workspace is an older name for it.
#
# Compatible with macOS and Linux. Windows users — see README.md "Uninstalling".

set -euo pipefail

ASSUME_YES=0
DRY_RUN=0
DEV_CLEANUP=0
PURGE_LIBRARY=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -y|--yes) ASSUME_YES=1 ;;
    -n|--dry-run) DRY_RUN=1 ;;
    --dev) DEV_CLEANUP=1 ;;
    --purge-library|--purge-workspace) PURGE_LIBRARY=1 ;;
    -h|--help)
      sed -n '2,23p' "$0"
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

OLD_WORKSPACE="$HOME/liner-workspace"

# The library folder, resolved the way Liner does (src/liner/library.py):
# LINER_DIR, then projects_dir in ~/.liner/config.yaml, then ~/liner/projects.
# Read it now, because step 3 can delete ~/.liner. LIBRARY_PROBLEM is set when
# the setting can't be read safely; the purge then refuses instead of guessing.
LIBRARY_DIR=""
LIBRARY_PROBLEM=""
trim() { local value="$1"; value="${value#"${value%%[![:space:]]*}"}"; printf '%s' "${value%"${value##*[![:space:]]}"}"; }
CONFIGURED_DIR=""
read_configured_library() {
  local config="$HOME/.liner/config.yaml" line value
  local pattern='^projects_dir:[[:space:]]*("[^"]*"|'"'"'[^'"'"']*'"'"'|[^"'"'"'#[:space:]][^#]*)[[:space:]]*$'
  [[ -f "$config" ]] || return 0
  line="$(grep -E '^[[:space:]]*["'"'"']?projects_dir["'"'"']?[[:space:]]*:' "$config" 2>/dev/null || true)"
  [[ -n "$line" ]] || return 0
  # YAML reads an empty value, an unquoted ~, or null as no value, and so does
  # Liner, which then uses the default library.
  if [[ "$line" =~ ^projects_dir:[[:space:]]*(~|null|Null|NULL)?[[:space:]]*$ ]]; then
    return 0
  fi
  if [[ "$line" == *$'\n'* ]] || ! [[ "$line" =~ $pattern ]]; then
    LIBRARY_PROBLEM="~/.liner/config.yaml sets projects_dir in a way this script can't read safely"
    return 0
  fi
  value="$(trim "${BASH_REMATCH[1]}")"
  case "$value" in
    \"*\"|\'*\') value="${value:1:${#value}-2}" ;;
  esac
  CONFIGURED_DIR="$value"
}
LIBRARY_DIR="$(trim "${LINER_DIR:-}")"
if [[ -z "$LIBRARY_DIR" ]]; then
  read_configured_library
  LIBRARY_DIR="$CONFIGURED_DIR"
fi
[[ -n "$LIBRARY_DIR" || -n "$LIBRARY_PROBLEM" ]] || LIBRARY_DIR="$HOME/liner/projects"
case "$LIBRARY_DIR" in
  "~") LIBRARY_DIR="$HOME" ;;
  "~/"*) LIBRARY_DIR="$HOME/${LIBRARY_DIR#\~/}" ;;
esac
if [[ -n "$LIBRARY_DIR" && "$LIBRARY_DIR" != /* ]]; then
  LIBRARY_PROBLEM="the library folder setting ($LIBRARY_DIR) isn't an absolute path"
fi
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

# --- 7. Your liners (--purge-library only) -------------------------------
#
# Deletes only folders that are liners (a non-hidden folder holding
# mixtape/tape.yaml, or tape.yaml in older liners), never the folder around
# them, so a library setting that points at the wrong place deletes nothing
# else. An emptied library folder is removed only if nothing else is in it.

if (( PURGE_LIBRARY )); then
  say "7. Your liners (--purge-library)"
  home_real="$(cd "$HOME" && pwd -P)"
  roots=()
  if [[ -n "$LIBRARY_PROBLEM" ]]; then
    warn "  Not deleting anything from the library: $LIBRARY_PROBLEM. Set LINER_DIR to the library's full path and run again."
  else
    roots+=("$LIBRARY_DIR")
  fi
  roots+=("$OLD_WORKSPACE")
  liners=()
  library_roots=()
  for root in "${roots[@]}"; do
    [[ -d "$root" ]] || continue
    resolved="$(cd "$root" 2>/dev/null && pwd -P)" || continue
    if [[ "$home_real/" == "$resolved"/* || "$resolved" == "/" ]]; then
      warn "  Not deleting the folder $root itself: it is / or holds your home folder. Only liners inside it are deleted."
    fi
    seen=0
    for existing in "${library_roots[@]-}"; do [[ "$existing" == "$resolved" ]] && seen=1; done
    (( seen )) && continue
    library_roots+=("$resolved")
    for child in "$resolved"/*; do
      [[ -d "$child" && ! -L "$child" ]] || continue
      [[ "$child" != "$home_real" && "$home_real/" != "$child"/* ]] || continue
      [[ -f "$child/mixtape/tape.yaml" || -f "$child/tape.yaml" ]] || continue
      seen=0
      for existing in "${liners[@]-}"; do [[ "$existing" == "$child" ]] && seen=1; done
      (( seen )) || liners+=("$child")
    done
  done
  if [[ ${#liners[@]} -eq 0 ]]; then
    info "  No liners found. Skipping."
  else
    for liner in "${liners[@]}"; do warn "  $liner"; done
    say "  ${#liners[@]} liner(s) found."
    delete_liners() {
      local liner root
      for liner in "${liners[@]}"; do run rm -rf "$liner"; done
      for root in "${library_roots[@]}"; do
        if [[ -d "$root" && "$home_real/" != "$root"/* && "$root" != "/" ]] && [[ -z "$(ls -A "$root" 2>/dev/null)" ]]; then
          run rmdir "$root" 2>/dev/null || true
        fi
      done
    }
    if (( DRY_RUN )); then
      delete_liners
    else
      reply=""
      printf '  This deletes the liners above and cannot be undone. Type %s to delete them: ' "${#liners[@]}"
      read -r reply </dev/tty 2>/dev/null || reply=""
      if [[ "$reply" == "${#liners[@]}" ]]; then
        delete_liners
        ok "Liners deleted."
      else
        info "  Skipped. Nothing was deleted."
      fi
    fi
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
