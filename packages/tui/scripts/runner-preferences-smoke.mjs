#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { chmodSync, copyFileSync, existsSync, mkdirSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import * as YAML from "yaml";
import { cleanupCommand, prepareIsolatedLaunch, formatIsolationSummary } from "./dev-isolated.mjs";
import { fingerprintTargets } from "./isolation-fingerprint.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");
const repoRoot = resolve(packageDir, "..", "..");
const goTuiDir = join(repoRoot, "packages", "go-tui");
const rootArg = process.argv.indexOf("--root");
const requestedRoot = rootArg >= 0 && process.argv[rootArg + 1] ? resolve(process.argv[rootArg + 1]) : "";
const productionHome = homedir();
const productionTargets = productionLinerTargets(productionHome);
const before = fingerprintTargets(productionTargets);
const launch = prepareIsolatedLaunch({ root: requestedRoot, codexHome: "", claudeHome: "" });
const root = launch.root;
const offlineBuildEnv = { ...process.env, GOPROXY: "off" };
console.log("[runner-preferences-smoke] Building local package with GOPROXY=off.");
const buildResult = spawnSync(
  process.platform === "win32" ? "npm.cmd" : "npm",
  ["run", "build:package"],
  { cwd: packageDir, env: offlineBuildEnv, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] },
);
if (buildResult.status !== 0) fail("offline local package build", buildResult);

const toolsDir = join(root, "smoke-tools");
mkdirSync(toolsDir, { recursive: true });
const smokeTestBin = join(toolsDir, process.platform === "win32" ? "runner-preferences-smoke.test.exe" : "runner-preferences-smoke.test");
const compileResult = spawnSync(
  process.platform === "win32" ? "go.exe" : "go",
  ["test", "-c", "-o", smokeTestBin, "./internal/app"],
  { cwd: goTuiDir, env: offlineBuildEnv, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] },
);
if (compileResult.status !== 0) fail("offline smoke-test compile", compileResult);

const fakeBinDir = join(root, "fake-provider-bin");
mkdirSync(fakeBinDir, { recursive: true });
const codexBin = join(fakeBinDir, process.platform === "win32" ? "codex.exe" : "codex");
const claudeBin = join(fakeBinDir, process.platform === "win32" ? "claude.exe" : "claude");
for (const providerBin of [codexBin, claudeBin]) {
  copyFileSync(smokeTestBin, providerBin);
  chmodSync(providerBin, 0o755);
}

const env = {
  ...launch.env,
  GOPROXY: "off",
  LINER_RUNNER_PREFERENCES_SMOKE_ROOT: root,
  LINER_HEADLESS_RUNNER: join(packageDir, "dist", "agents", "headless-runner.js"),
  LINER_SKILL_PATH: join(repoRoot, "docs", "curation-skill"),
  LINER_GO_TUI_BIN: join(packageDir, "bin", process.platform === "win32" ? "liner-tui.exe" : "liner-tui"),
  LINER_CODEX_BIN: codexBin,
  LINER_CLAUDE_BIN: claudeBin,
};

console.log(formatIsolationSummary(launch));
console.log("\n[runner-preferences-smoke] Launching the real local shim and Go TUI inside the isolated root.");
const launchResult = launchTUIAndQuit(env);
if (launchResult.status !== 0) fail("isolated shim/TUI launch", launchResult);

console.log("[runner-preferences-smoke] Driving Settings, restart, Clarify Job, and methodology with offline fake CLIs.");
const testResult = spawnSync(
  smokeTestBin,
  ["-test.run", "^TestRunnerPreferencesCleanRoomSmoke$", "-test.count=1", "-test.v"],
  { cwd: goTuiDir, env, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] },
);
if (testResult.status !== 0) fail("clean-room product smoke", testResult);
process.stdout.write(testResult.stdout);

const after = fingerprintTargets(productionTargets);
if (before !== after) {
  console.error("[runner-preferences-smoke] Production Liner paths changed during the isolated smoke.");
  console.error("The isolated root was retained for diagnosis: " + root);
  process.exit(1);
}

console.log("[runner-preferences-smoke] PASS: normal Liner config and Project library metadata are unchanged.");
console.log("[runner-preferences-smoke] Evidence retained under: " + root);
console.log("[runner-preferences-smoke] Cleanup: " + cleanupCommand(root));

function launchTUIAndQuit(environment) {
  const shim = join(packageDir, "bin", "liner.js");
  if (process.platform === "darwin") {
    return spawnSync("/bin/sh", ["-c", 'printf q | script -q /dev/null "$0" "$1"', process.execPath, shim], {
      cwd: packageDir,
      env: environment,
      encoding: "utf8",
    });
  }
  if (process.platform !== "win32") {
    const command = `${shellQuote(process.execPath)} ${shellQuote(shim)}`;
    return spawnSync("script", ["-q", "-c", command, "/dev/null"], {
      cwd: packageDir,
      env: environment,
      input: "q",
      encoding: "utf8",
    });
  }
  // Windows lacks a standard-library ConPTY bridge in Node. The initialization
  // probe still traverses the no-argument npm-shim -> local Go-TUI seam and
  // constructs the real app model before exiting. The product smoke below
  // drives the interactive Settings model and runner bridges.
  return spawnSync(process.execPath, [shim], {
    cwd: packageDir,
    env: { ...environment, LINER_TUI_INITIALIZATION_PROBE: "1" },
    encoding: "utf8",
  });
}

function productionLinerTargets(home) {
  const config = join(home, ".liner", "config.yaml");
  const targets = new Set([config, join(home, "liner", "projects")]);
  if (existsSync(config)) {
    try {
      const raw = YAML.parse(readFileSync(config, "utf8"));
      if (raw && typeof raw === "object" && typeof raw.projects_dir === "string" && raw.projects_dir.trim()) {
        const configured = raw.projects_dir.trim().replace(/^~(?=[/\\]|$)/, home);
        targets.add(resolve(configured));
      }
    } catch {
      // A malformed production config is still fingerprinted as a file. The
      // smoke never needs to parse it in order to keep its own writes isolated.
    }
  }
  return Array.from(targets);
}

function fail(label, result) {
  console.error(`[runner-preferences-smoke] Failed: ${label}`);
  if (result.stdout) process.stderr.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);
  console.error("The isolated root was retained for diagnosis: " + root);
  process.exit(1);
}

function shellQuote(value) {
  return `'${value.replaceAll("'", `'\\''`)}'`;
}
