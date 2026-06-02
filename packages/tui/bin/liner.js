#!/usr/bin/env node
// Entry shim for `npx linersh` / `liner`.
//
// No args: launch the Ink TUI from dist/.
// CLI args: forward to the bundled Python core (or LINER_BIN/dev fallback).
//
// This lets the npm package expose one binary name while still supporting
// mechanical commands like `liner setup-js`, `liner compile`, and `liner status`.
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";
import { spawn, spawnSync } from "node:child_process";
import { existsSync, readFileSync, rmSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const packageRoot = join(here, "..");
const argv = process.argv.slice(2);

// `liner --version` used to forward straight to the Python core, so it
// reported the core version (e.g. 0.5.0) regardless of which TUI package was
// installed — confusing when verifying an npm publish. Report the npm/TUI
// version (the thing the user actually installed) and, when resolvable, the
// bundled core version alongside it.
if (argv[0] === "--version" || argv[0] === "-v") {
  printVersion();
  process.exit(0);
}

if (argv[0] === "uninstall") {
  const code = await runUninstall(argv.slice(1));
  process.exit(code);
}

const CLI_COMMANDS = new Set([
  "init",
  "replay",
  "compile",
  "share",
  "import",
  "clone",
  "setup-js",
  "list",
  "cache",
  "manifest",
  "status",
]);

function wantsCli(args) {
  if (args.length === 0) return false;
  const first = args[0];
  return first === "--help" || first === "--version" || first === "-h" || CLI_COMMANDS.has(first);
}

if (wantsCli(argv)) {
  const resolved = resolveCliBinary();
  const child = spawn(resolved.command, [...resolved.args, ...argv], {
    stdio: "inherit",
    env: { ...process.env, LINER_TUI_SHIM: "1" },
  });
  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(code ?? 1);
  });
} else {
  // Enter the terminal's alternate screen buffer before launching Ink.
  //
  // Every Ink screen that changes total render height between frames (each
  // wizard step, each PhaseRunner stream event, each gate transition) was
  // committing older frames to the user's scrollback — producing stacked
  // duplicated headers above the visible content. The alternate buffer has
  // no scrollback by definition, so the bug class is eliminated.
  //
  // Trade-off: on exit, the TUI vanishes and the terminal returns to its
  // pre-launch state. Post-exit context lives in `.liner-runs/` and
  // `MIXTAPE.md`, which are the intended sources of truth anyway.
  //
  // Only enter alt-screen on a real TTY; piped/redirected stdout shouldn't
  // receive raw escape sequences.
  const useAltScreen = process.stdout.isTTY === true;

  if (useAltScreen) {
    process.stdout.write("\x1b[?1049h");
    const restoreScreen = () => {
      process.stdout.write("\x1b[?1049l");
    };
    process.on("exit", restoreScreen);
    process.on("SIGINT", () => {
      restoreScreen();
      process.exit(130);
    });
    process.on("SIGTERM", () => {
      restoreScreen();
      process.exit(143);
    });
    process.on("uncaughtException", (err) => {
      restoreScreen();
      console.error(err);
      process.exit(1);
    });
  }

  await import(join(packageRoot, "dist", "index.js"));
}

function resolveCliBinary() {
  const envBin = process.env.LINER_BIN;
  if (envBin && existsSync(envBin)) {
    return { command: envBin, args: [] };
  }

  const bundled = findBundledBinary();
  if (bundled) return { command: bundled, args: [] };

  const repoVenv = findRepoVenvBinary();
  if (repoVenv) return { command: repoVenv, args: [] };

  const pathHit = findPathBinary();
  if (pathHit) return { command: pathHit, args: [] };

  console.error(
    [
      "Could not find the Liner CLI binary.",
      "",
      "Expected one of:",
      "  - LINER_BIN=/path/to/liner",
      "  - bundled platform package (linersh-<platform>-<arch>)",
      "  - repo-local .venv/bin/liner",
      "  - a separate liner on PATH",
    ].join("\n"),
  );
  process.exit(1);
}

async function runUninstall(args) {
  if (args.includes("--help") || args.includes("-h")) {
    console.log(
      [
        "Usage: liner uninstall [--yes] [--dry-run]",
        "",
        "Remove local Liner traces for a clean reinstall/test:",
        "  - ~/.liner config and source cache",
        "  - Playwright's Chromium browser cache",
        "  - npm's npx execution cache (_npx)",
        "",
        "This does not delete your mixtape project folders.",
        "For global installs, also run: npm uninstall -g linersh",
      ].join("\n"),
    );
    return 0;
  }

  const known = new Set(["--yes", "-y", "--dry-run"]);
  const unknown = args.filter((arg) => !known.has(arg));
  if (unknown.length > 0) {
    console.error(`Unknown option: ${unknown.join(", ")}`);
    console.error("Run `liner uninstall --help` for usage.");
    return 1;
  }

  const yes = args.includes("--yes") || args.includes("-y");
  const dryRun = args.includes("--dry-run");
  const targets = uninstallTargets();

  console.error("liner uninstall will remove:");
  for (const target of targets) {
    console.error(`  - ${target.label}: ${target.path}`);
  }
  console.error("");
  console.error("It will not delete your mixtape project folders.");

  if (!yes && !dryRun) {
    if (!process.stdin.isTTY) {
      console.error("Refusing to run without a TTY prompt. Re-run with --yes.");
      return 1;
    }
    const { createInterface } = await import("node:readline/promises");
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    const answer = (await rl.question("Continue? [y/N] ")).trim().toLowerCase();
    rl.close();
    if (answer !== "y" && answer !== "yes") {
      console.error("Cancelled.");
      return 1;
    }
  }

  let failed = false;
  for (const target of targets) {
    if (dryRun) {
      console.error(`[dry-run] would remove ${target.path}`);
      continue;
    }
    try {
      rmSync(target.path, { recursive: true, force: true });
      console.error(`Removed ${target.path}`);
    } catch (err) {
      failed = true;
      console.error(`Failed to remove ${target.path}: ${err.message}`);
    }
  }

  if (dryRun) {
    console.error("Dry run complete.");
    return 0;
  }
  if (failed) return 1;
  console.error("Liner local state and caches removed.");
  console.error("For a global install, finish with: npm uninstall -g linersh");
  return 0;
}

function uninstallTargets() {
  const home = homedir();
  return [
    {
      label: "Liner config and source cache",
      path: join(home, ".liner"),
    },
    {
      label: "Playwright Chromium cache",
      path: playwrightCachePath(home),
    },
    {
      label: "npm npx execution cache",
      path: join(npmCachePath(home), "_npx"),
    },
  ];
}

function npmCachePath(home) {
  const npm = spawnSync("npm", ["config", "get", "cache"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "ignore"],
  });
  const configured = npm.status === 0 ? npm.stdout.trim() : "";
  if (configured) return configured;
  if (process.platform === "win32") {
    return join(process.env.LOCALAPPDATA || join(home, "AppData", "Local"), "npm-cache");
  }
  return join(home, ".npm");
}

function playwrightCachePath(home) {
  if (process.platform === "darwin") {
    return join(home, "Library", "Caches", "ms-playwright");
  }
  if (process.platform === "win32") {
    return join(process.env.LOCALAPPDATA || join(home, "AppData", "Local"), "ms-playwright");
  }
  return join(process.env.XDG_CACHE_HOME || join(home, ".cache"), "ms-playwright");
}

function findBundledBinary() {
  const require = createRequire(import.meta.url);
  const target = `linersh-${process.platform}-${process.arch}`;
  try {
    const pkgJson = require.resolve(`${target}/package.json`);
    const baseDir = pkgJson.replace(/[\\/]package\.json$/, "");
    const candidate = join(baseDir, exeName());
    return existsSync(candidate) ? candidate : null;
  } catch {
    return null;
  }
}

function findRepoVenvBinary() {
  const venvBinDir = process.platform === "win32" ? "Scripts" : "bin";
  let dir = packageRoot;
  for (let i = 0; i < 12; i += 1) {
    const candidate = join(dir, ".venv", venvBinDir, exeName());
    if (existsSync(candidate)) return candidate;
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}

function findPathBinary() {
  const probe = spawnSync(process.platform === "win32" ? "where" : "which", ["liner"], {
    encoding: "utf8",
  });
  if (probe.status !== 0) return null;
  const hits = probe.stdout
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
  const thisShim = fileURLToPath(import.meta.url);
  return hits.find((hit) => hit !== thisShim) ?? null;
}

function exeName() {
  return process.platform === "win32" ? "liner.exe" : "liner";
}

function readTuiVersion() {
  try {
    const pkg = JSON.parse(readFileSync(join(packageRoot, "package.json"), "utf8"));
    return typeof pkg.version === "string" ? pkg.version : "unknown";
  } catch {
    return "unknown";
  }
}

// Like resolveCliBinary() but non-fatal: returns null instead of exiting when
// no core binary is found, so `--version` still prints the TUI version on a
// host where the bundled core is missing (e.g. Windows pre-unblock).
function tryResolveCliCommand() {
  const envBin = process.env.LINER_BIN;
  if (envBin && existsSync(envBin)) return envBin;
  return findBundledBinary() || findRepoVenvBinary() || findPathBinary() || null;
}

function printVersion() {
  const tui = readTuiVersion();
  let coreVer = null;
  const core = tryResolveCliCommand();
  if (core) {
    const r = spawnSync(core, ["--version"], { encoding: "utf8" });
    if (r.status === 0 && typeof r.stdout === "string") {
      coreVer = r.stdout.trim().replace(/^liner\s+/i, "") || null;
    }
  }
  console.log(coreVer ? `liner ${tui} (tui)  ·  ${coreVer} (core)` : `liner ${tui} (tui)`);
}
