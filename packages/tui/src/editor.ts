// Centralized $EDITOR resolution.
//
// The default fallback used to be `vi`, which is the classic Unix tradition
// but a hostile UX for anyone who hasn't learned vim. Nano is on every macOS
// install, is on most Linux distros, and crucially shows its keybindings at
// the bottom of the screen — a user who has never seen it before can still
// figure out save (^O) and exit (^X).
//
// $VISUAL beats $EDITOR per the de-facto convention; the user explicitly set
// either one if they prefer it.

import { existsSync } from "node:fs";

const FALLBACK_CANDIDATES = ["nano", "pico", "vi"] as const;

/**
 * Resolve which editor to spawn for in-tui $EDITOR launches. Honors the
 * user's $VISUAL / $EDITOR if set; otherwise probes for newbie-friendly
 * editors before falling back to vi.
 */
export function resolveEditor(): string {
  const explicit = (process.env["VISUAL"] || process.env["EDITOR"] || "").trim();
  if (explicit) return explicit;
  for (const candidate of FALLBACK_CANDIDATES) {
    if (binaryOnPath(candidate)) return candidate;
  }
  return "vi";
}

/** True when a binary lookups in the user's PATH. */
function binaryOnPath(name: string): boolean {
  const pathEnv = process.env["PATH"] || "";
  const sep = process.platform === "win32" ? ";" : ":";
  for (const dir of pathEnv.split(sep)) {
    if (!dir) continue;
    if (existsSync(`${dir}/${name}`)) return true;
  }
  return false;
}
