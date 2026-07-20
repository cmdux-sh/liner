#!/usr/bin/env node

import { spawn } from "node:child_process";
import {
  accessSync,
  constants,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  realpathSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");

export function parseIsolatedArgs(argv) {
  const args = { root: "", codexHome: "", claudeHome: "", dryRun: false, help: false };
  for (let index = 0; index < argv.length; index++) {
    const value = argv[index];
    if (value === "--help" || value === "-h") {
      args.help = true;
      continue;
    }
    if (value === "--dry-run") {
      args.dryRun = true;
      continue;
    }
    const key = value === "--root" ? "root" : value === "--codex-home" ? "codexHome" : value === "--claude-home" ? "claudeHome" : "";
    if (!key) throw new Error(`Unknown option: ${value}`);
    const next = argv[++index];
    if (!next || next.startsWith("--")) throw new Error(`Missing value for ${value}`);
    args[key] = next;
  }
  return args;
}

export function prepareIsolatedLaunch(args, inheritedEnv = process.env) {
  const root = createOwnedRoot(args.root);
  const home = join(root, "home");
  const projects = join(root, "projects");
  const temporaryCodexHome = join(root, "provider-homes", "codex");
  const temporaryClaudeHome = join(root, "provider-homes", "claude");
  const cache = join(root, "cache");
  const configHome = join(root, "config");
  const dataHome = join(root, "data");
  const localAppData = join(root, "local-app-data");
  const roamingAppData = join(root, "roaming-app-data");
  const temporary = join(root, "tmp");
  const playwrightBrowsers = join(cache, "ms-playwright");
  for (const path of [home, projects, temporaryCodexHome, temporaryClaudeHome, cache, configHome, dataHome, localAppData, roamingAppData, temporary, playwrightBrowsers]) {
    mkdirSync(path, { recursive: true });
    requireManagedDirectory(root, path);
  }

  const codexHome = args.codexHome ? requireDirectory(args.codexHome, "Codex CLI configuration home") : temporaryCodexHome;
  const claudeHome = args.claudeHome ? requireDirectory(args.claudeHome, "Claude Code configuration home") : temporaryClaudeHome;
  const env = sanitizedEnvironment(inheritedEnv);
  Object.assign(env, {
    HOME: home,
    USERPROFILE: home,
    LINER_DIR: projects,
    LINER_TUI: "go",
    LINER_HEADLESS_RUNNER: join(packageDir, "dist", "agents", "headless-runner.js"),
    LINER_CODEX_HOME: codexHome,
    CODEX_HOME: codexHome,
    LINER_CLAUDE_HOME: claudeHome,
    CLAUDE_CONFIG_DIR: claudeHome,
    XDG_CACHE_HOME: cache,
    XDG_CONFIG_HOME: configHome,
    XDG_DATA_HOME: dataHome,
    LOCALAPPDATA: localAppData,
    APPDATA: roamingAppData,
    TMPDIR: temporary,
    TMP: temporary,
    TEMP: temporary,
    NPM_CONFIG_CACHE: join(cache, "npm"),
    PLAYWRIGHT_BROWSERS_PATH: playwrightBrowsers,
  });
  return {
    root,
    home,
    projects,
    config: join(home, ".liner", "config.yaml"),
    codexHome,
    claudeHome,
    codexHomeSource: args.codexHome ? "existing profile (referenced in place)" : "temporary empty profile",
    claudeHomeSource: args.claudeHome ? "existing profile (referenced in place)" : "temporary empty profile",
    env,
  };
}

function sanitizedEnvironment(inheritedEnv) {
  const env = {};
  for (const [key, value] of Object.entries(inheritedEnv)) {
    if (!isProviderOrLinerOverride(key)) env[key] = value;
  }
  return env;
}

function isProviderOrLinerOverride(key) {
  const normalized = key.toUpperCase();
  return normalized.startsWith("LINER_") ||
    normalized.startsWith("ANTHROPIC_") ||
    normalized.startsWith("CLAUDE_") ||
    normalized.startsWith("CODEX_") ||
    normalized.startsWith("OPENAI_") ||
    normalized.startsWith("AWS_") ||
    normalized.startsWith("AZURE_") ||
    normalized.startsWith("GCLOUD_") ||
    normalized.startsWith("GCP_") ||
    normalized.startsWith("GOOGLE_") ||
    normalized.startsWith("CLOUDSDK_");
}

function createOwnedRoot(requestedRoot) {
  if (!requestedRoot) {
    const root = realpathSync(mkdtempSync(join(tmpdir(), "liner-development-")));
    writeRootMarker(root);
    return root;
  }
  const root = resolve(requestedRoot);
  if (existsSync(root)) {
    throw new Error(`Isolated root must not already exist: ${root}`);
  }
  mkdirSync(root);
  if (lstatSync(root).isSymbolicLink()) {
    throw new Error(`Isolated root must not be a symbolic link: ${root}`);
  }
  const canonical = realpathSync(root);
  writeRootMarker(canonical);
  return canonical;
}

function writeRootMarker(root) {
  writeFileSync(join(root, ".liner-isolated-root"), "Created by npm run dev:isolated.\n", { flag: "wx" });
}

function requireManagedDirectory(root, path) {
  if (lstatSync(path).isSymbolicLink()) {
    throw new Error(`Managed isolation path must not be a symbolic link: ${path}`);
  }
  const canonicalRoot = realpathSync(root);
  const canonicalPath = realpathSync(path);
  const prefix = canonicalRoot.endsWith("/") || canonicalRoot.endsWith("\\") ? canonicalRoot : canonicalRoot + (process.platform === "win32" ? "\\" : "/");
  if (canonicalPath !== canonicalRoot && !canonicalPath.startsWith(prefix)) {
    throw new Error(`Managed isolation path escaped its root: ${path}`);
  }
}

export function formatIsolationSummary(launch) {
  return [
    "Liner isolated development launch",
    `  Isolated root:       ${launch.root}`,
    `  Liner HOME:         ${launch.home}`,
    `  Liner config:       ${launch.config}`,
    `  Project library:    ${launch.projects}`,
    `  OpenAI / Codex CLI: ${launch.codexHome} (${launch.codexHomeSource})`,
    `  Claude Code:        ${launch.claudeHome} (${launch.claudeHomeSource})`,
    "",
    "Existing provider profiles are referenced only when explicitly selected; credentials are never copied.",
    `Cleanup after testing: ${cleanupCommand(launch.root)}`,
  ].join("\n");
}

export async function main(argv = process.argv.slice(2)) {
  let args;
  try {
    args = parseIsolatedArgs(argv);
  } catch (error) {
    console.error(`[dev:isolated] ${error.message}`);
    console.error(usage());
    return 2;
  }
  if (args.help) {
    console.log(usage());
    return 0;
  }

  let launch;
  try {
    launch = prepareIsolatedLaunch(args);
  } catch (error) {
    console.error(`[dev:isolated] ${error.message}`);
    return 2;
  }
  console.log(formatIsolationSummary(launch));
  if (args.dryRun) return 0;

  return await new Promise((resolveExit) => {
    const child = spawn(process.execPath, [join(packageDir, "bin", "liner.js")], {
      cwd: packageDir,
      env: launch.env,
      stdio: "inherit",
    });
    child.on("error", (error) => {
      console.error(`[dev:isolated] Could not launch Liner: ${error.message}`);
      resolveExit(1);
    });
    child.on("exit", (code, signal) => {
      if (signal) {
        process.kill(process.pid, signal);
        return;
      }
      resolveExit(code ?? 1);
    });
  });
}

function requireDirectory(path, label) {
  const resolved = resolve(path);
  try {
    if (!statSync(resolved).isDirectory()) throw new Error("not a directory");
    accessSync(resolved, constants.R_OK);
  } catch {
    throw new Error(`${label} is not a readable directory: ${resolved}`);
  }
  return realpathSync(resolved);
}

export function cleanupCommand(path) {
  if (process.platform === "win32") {
    return `powershell -NoProfile -Command "Remove-Item -LiteralPath '${path.replaceAll("'", "''")}' -Recurse -Force"`;
  }
  return `rm -rf '${path.replaceAll("'", `'\\''`)}'`;
}

function usage() {
  return [
    "Usage: npm --prefix packages/tui run dev:isolated -- [options]",
    "",
    "Options:",
    "  --codex-home <path>  Explicitly reference an existing Codex CLI profile.",
    "  --claude-home <path> Explicitly reference an existing Claude Code profile.",
    "  --root <path>        Create a new named isolated root; the path must not exist.",
    "  --dry-run            Prepare and print the isolation boundary without launching.",
  ].join("\n");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.exitCode = await main();
}
