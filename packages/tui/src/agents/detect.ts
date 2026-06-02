import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import { existsSync } from "node:fs";
import { homedir } from "node:os";
import { join, dirname, basename } from "node:path";
import { fileURLToPath } from "node:url";
import type { AgentDescriptor, AgentId } from "./types.js";

/**
 * Resolve an agent binary path. Resolution order:
 *
 *   1. Explicit env override (`LINER_CLAUDE_BIN` / `LINER_CODEX_BIN`).
 *      Honored even if the path is missing — we surface the path the user
 *      named so they can see exactly what was tried.
 *   2. `which <name>` against the inherited PATH. Fast and correct for the
 *      common case (user launched the TUI from their normal shell).
 *   3. `bash -lc 'command -v <name>'` against the *login* shell. Costs ~50ms
 *      but sources the user's shell init (.bashrc / .zshrc / fnm / nvm /
 *      asdf), catching version-manager shims that aren't in the TUI's
 *      inherited PATH — e.g. cmux-launched sessions where fnm hasn't run.
 *
 * Returning null means "definitely not installed" — both shells failed and
 * no env override was set.
 */
function resolveBin(name: string, envVar: string): string | null {
  const override = process.env[envVar];
  if (override && override.trim()) return override.trim();

  const direct = spawnSync("which", [name], { encoding: "utf8" });
  if (direct.status === 0) {
    const out = direct.stdout.trim();
    if (out) return out;
  }

  // Fall back to a login shell so shims (fnm / nvm / asdf / pyenv / etc.)
  // get a chance to register. `command -v` is POSIX-portable; `-l` makes
  // bash source the login profile that puts those shims on PATH.
  const viaLoginShell = spawnSync("bash", ["-lc", `command -v ${name}`], {
    encoding: "utf8",
  });
  if (viaLoginShell.status === 0) {
    const out = viaLoginShell.stdout.trim();
    if (out) return out;
  }

  return null;
}

/**
 * Names the env var that overrides binary detection for an agent. Mirrors
 * the LINER_BIN pattern from bin-resolver — explicit > inferred.
 */
export function envVarForAgent(id: AgentId): string {
  return id === "claude" ? "LINER_CLAUDE_BIN" : "LINER_CODEX_BIN";
}

/** Catalog of agents the TUI knows how to drive. Single source of truth so
 *  `detectAgents` and the "not detected" UI in AgentSetup stay in sync. */
export const KNOWN_AGENTS: ReadonlyArray<{ id: AgentId; name: string; envVar: string }> = [
  { id: "claude", name: "Claude Code", envVar: "LINER_CLAUDE_BIN" },
  { id: "codex", name: "OpenAI Codex", envVar: "LINER_CODEX_BIN" },
];

export function detectAgents(): AgentDescriptor[] {
  const found: AgentDescriptor[] = [];
  for (const meta of KNOWN_AGENTS) {
    const bin = resolveBin(meta.id, meta.envVar);
    if (bin) found.push({ id: meta.id, name: meta.name, bin });
  }
  return found;
}

export type SkillPathResolution = {
  path: string | null;
  envPath: string | null;
  searched: string[];
};

/**
 * Find the curating-mixtapes skill bundle on disk.
 *
 * Resolution order:
 *   1. `LINER_SKILL_PATH` env var (must contain SKILL.md).
 *   2. Walk up from this module's own location (covers the npm-installed
 *      case — the skill bundle ships inside the package at
 *      `linersh/cli-update-docs/`, and `import.meta.url` resolves the
 *      symlinks npx puts in front of the package).
 *   3. Walk up from cwd (covers the dev-from-repo case where the user
 *      is somewhere inside the liner working tree).
 *   4. Walk up from `process.argv[1]` (legacy global-install case).
 */
export function resolveSkillPath(): string | null {
  return resolveSkillPathWithDiagnostics().path;
}

export function resolveSkillPathWithDiagnostics(): SkillPathResolution {
  const env = process.env["LINER_SKILL_PATH"];
  const envPath = env && env.trim() ? normalizeSkillPathInput(env.trim()) : null;
  if (envPath && existsSync(join(envPath, "SKILL.md"))) {
    return { path: envPath, envPath, searched: [join(envPath, "SKILL.md")] };
  }

  const candidates = ["cli-update-docs", "docs/skill", "skills/curating-mixtapes"];

  const moduleDir = dirname(fileURLToPath(import.meta.url));
  const packageRoot = nearestPackageRoot(moduleDir);
  const searchFromRoots = [
    moduleDir,
    packageRoot,
    selfPackageRoot(),
    process.cwd(),
    dirname(process.argv[1] ?? ""),
  ].filter((root): root is string => Boolean(root));

  const searched: string[] = envPath ? [join(envPath, "SKILL.md")] : [];
  for (const root of searchFromRoots) {
    const hit = walkUp(root, candidates, searched);
    if (hit) return { path: hit, envPath, searched };
  }
  return { path: null, envPath, searched };
}

function normalizeSkillPathInput(input: string): string {
  const expanded = input === "~"
    ? homedir()
    : input.startsWith("~/")
      ? join(homedir(), input.slice(2))
      : input;
  return basename(expanded) === "SKILL.md" ? dirname(expanded) : expanded;
}

function nearestPackageRoot(start: string): string | null {
  let dir = start;
  for (let i = 0; i < 12; i++) {
    if (existsSync(join(dir, "package.json"))) return dir;
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
  return null;
}

function selfPackageRoot(): string | null {
  try {
    const require = createRequire(import.meta.url);
    return dirname(require.resolve("linersh/package.json"));
  } catch {
    return null;
  }
}

function walkUp(start: string, candidates: readonly string[], searched: string[]): string | null {
  let dir = start;
  for (let i = 0; i < 12; i++) {
    for (const rel of candidates) {
      const guess = join(dir, rel);
      const skillFile = join(guess, "SKILL.md");
      searched.push(skillFile);
      if (existsSync(skillFile)) return guess;
    }
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
  return null;
}
