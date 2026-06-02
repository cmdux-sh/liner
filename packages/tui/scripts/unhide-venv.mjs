#!/usr/bin/env node
// Defensive: clear macOS UF_HIDDEN flag on the repo venv before each dev run.
//
// macOS's File Provenance / Spotlight subsystem sometimes marks `.venv/` and
// every file inside as hidden. Python 3.14's `site.addpackage()` silently
// skips hidden `.pth` files (the trace only fires under PYTHONVERBOSE=1),
// which breaks editable installs invisibly.
//
// This script is idempotent and silent on non-macOS / no-venv setups.
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

if (process.platform !== "darwin") process.exit(0);

const __filename = fileURLToPath(import.meta.url);
const venv = findVenv(dirname(__filename));
if (!venv) process.exit(0);

// `chflags -R nohidden <venv>` — no-op for already-unhidden files.
const result = spawnSync("chflags", ["-R", "nohidden", venv], { stdio: "ignore" });
if (result.status !== 0) {
  // Don't fail the dev run on this — log a hint and continue.
  console.warn(`[unhide-venv] chflags exited ${result.status} on ${venv}; continuing anyway.`);
}

function findVenv(startDir) {
  let dir = startDir;
  for (let i = 0; i < 10; i += 1) {
    const candidate = join(dir, ".venv");
    if (existsSync(candidate)) return candidate;
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
  return null;
}
