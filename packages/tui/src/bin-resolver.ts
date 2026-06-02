// Locate the `liner` binary to spawn.
//
// Resolution order:
//   1. LINER_BIN env var (explicit override)
//   2. Bundled platform-specific binary from optionalDependencies
//      (e.g. linersh-darwin-arm64/liner) — the production path
//   3. Repo-local `.venv/bin/liner` discovered by walking up from this file
//      (dev convenience — when you're hacking on the Python source in a
//      checkout, the local venv should always win over any global install
//      so your edits actually run.)
//   4. `liner` on $PATH (e.g. `pipx install linersh` or an activated venv)
//
// Branches (1) and (2) are the only paths that matter for the published npm
// package — neither dev path applies there (no `.venv` to walk up to). (3)
// before (4) is the dev affordance: previously PATH beat .venv, which meant
// a developer with a pipx-installed `liner` would silently run the global
// version instead of their local Python edits.

import { existsSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const __filename = fileURLToPath(import.meta.url);

export type ResolvedBinary = {
  command: string;
  args: string[];
  source: "env" | "bundled" | "path" | "repo-venv";
};

export function resolveBinary(): ResolvedBinary {
  // 1. Explicit env override
  const envBin = process.env["LINER_BIN"];
  if (envBin && existsSync(envBin)) {
    return { command: envBin, args: [], source: "env" };
  }

  // 2. Bundled binary via optionalDependencies (production path)
  const target = `linersh-${process.platform}-${process.arch}`;
  try {
    const pkgJson = require.resolve(`${target}/package.json`);
    const baseDir = pkgJson.replace(/[\\/]package\.json$/, "");
    const bundled = join(baseDir, exeName());
    if (existsSync(bundled)) {
      return { command: bundled, args: [], source: "bundled" };
    }
  } catch {
    // fall through
  }

  // 3. Repo-local venv (dev convenience). Walk up from this file until we
  //    hit a `.venv/bin/liner` or the filesystem root. Prefer this over a
  //    global pipx install so dev edits to the Python source take effect
  //    immediately without juggling `LINER_BIN`.
  const repoVenvBin = findRepoVenvBinary();
  if (repoVenvBin) {
    return { command: repoVenvBin, args: [], source: "repo-venv" };
  }

  // 4. PATH lookup (works when user has pipx-installed liner globally or
  //    activated an unrelated venv).
  const probe = spawnSync(process.platform === "win32" ? "where" : "which", ["liner"]);
  if (probe.status === 0) {
    return { command: "liner", args: [], source: "path" };
  }

  throw new BinaryNotFoundError(target);
}

function exeName(): string {
  return process.platform === "win32" ? "liner.exe" : "liner";
}

function findRepoVenvBinary(): string | null {
  const venvBinDir = process.platform === "win32" ? "Scripts" : "bin";
  let dir = dirname(__filename);
  // Cap the walk at ~10 levels so we never blow up on weird filesystems.
  for (let i = 0; i < 10; i += 1) {
    const candidate = join(dir, ".venv", venvBinDir, exeName());
    if (existsSync(candidate)) return candidate;
    const parent = dirname(dir);
    if (parent === dir) break; // hit filesystem root
    dir = parent;
  }
  return null;
}

export class BinaryNotFoundError extends Error {
  constructor(target: string) {
    super(
      `Could not find the liner binary.\n` +
        `  • No LINER_BIN env var set.\n` +
        `  • No bundled binary for ${target} (optionalDependency missing).\n` +
        `  • \`liner\` not on PATH.\n` +
        `  • No repo-local \`.venv/bin/liner\` found by walking up from this file.\n\n` +
        `Fix one of:\n` +
        `  • Install the Python core globally: \`pipx install linersh\`\n` +
        `  • Activate the repo's venv: \`source .venv/bin/activate\` (then re-run)\n` +
        `  • Point at a specific binary: \`LINER_BIN=/path/to/liner npm run dev\``,
    );
    this.name = "BinaryNotFoundError";
  }
}
