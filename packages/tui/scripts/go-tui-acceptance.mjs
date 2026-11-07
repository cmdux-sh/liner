#!/usr/bin/env node
// Acceptance checks for the Go TUI default-switch runbook.
//
// This does not drive the interactive agent runs. With no subcommand it
// automates the repeatable shim and Go test preflight. The `release-smoke`
// subcommand packs the current platform package plus the main npm package and
// installs both into a clean consumer project.

import { spawn, spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, realpathSync, statSync } from "node:fs";
import { chmod, mkdir, writeFile } from "node:fs/promises";
import { basename, dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { homedir } from "node:os";
import {
  expectedGoVersion,
  expectedShimVersion,
  matchesCanonicalVersion,
} from "./release-version-contract.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");
const repoRoot = resolve(packageDir, "..", "..");
const canonicalVersion = readFileSync(join(repoRoot, "VERSION"), "utf8").trim();
const goTuiDir = join(repoRoot, "packages", "go-tui");
const npmCmd = process.platform === "win32" ? "npm.cmd" : "npm";
const goCmd = process.platform === "win32" ? "go.exe" : "go";
const linerTuiBin = join(packageDir, "bin", process.platform === "win32" ? "liner-tui.exe" : "liner-tui");
const installedLinerBin = process.platform === "win32" ? "liner.cmd" : "liner";
const packagedGoTuiBin = process.platform === "win32" ? "liner-tui.exe" : "liner-tui";
const packageName = "linersh";
const platformPackageName = `linersh-${process.platform}-${process.arch}`;
const shimProbeMarker = "liner-go-tui-shim-probe";

const knownSubcommands = new Set(["release-smoke"]);
const subcommand = knownSubcommands.has(process.argv[2]) ? process.argv[2] : "preflight";
const rawArgs = subcommand === "preflight" ? process.argv.slice(2) : process.argv.slice(3);

let args;
try {
  if (subcommand === "release-smoke") {
    args = parseReleaseArgs(rawArgs);
  } else {
    args = parsePreflightArgs(rawArgs);
  }
} catch (error) {
  console.error(`[go-tui-acceptance] ${error.message}`);
  printHelp();
  process.exit(1);
}

if (args.help) {
  printHelp();
  process.exit(0);
}

const evidenceDir = resolveUserPath(args.dir || process.env.LINER_ACCEPTANCE_DIR || "/tmp/liner-go-live-acceptance");
const stamp = new Date().toISOString().replace(/[:.]/g, "-");

if (subcommand === "release-smoke") {
  await runReleaseSmoke(args, evidenceDir, stamp);
} else {
  await runPreflight(args, evidenceDir, stamp);
}

async function runPreflight(args, evidenceDir, stamp) {
  const reportPath = join(evidenceDir, `go-tui-preflight-${stamp}.md`);
  await mkdir(evidenceDir, { recursive: true });
  const shimProbeBin = await writeGoTuiShimProbe(evidenceDir, stamp);

  const steps = [
    ...(!args.skipBuild ? [{
      name: "Build TUI package",
      cwd: packageDir,
      command: npmCmd,
      commandLabel: "npm run build:package",
      args: ["run", "build:package"],
    }] : []),
    {
      name: "Go TUI binary version",
      cwd: packageDir,
      command: linerTuiBin,
      commandLabel: "./bin/liner-tui --version",
      args: ["--version"],
    },
    {
      name: "npm shim defaults to Go TUI",
      cwd: packageDir,
      command: process.execPath,
      commandLabel: "LINER_GO_TUI_BIN=<probe> node bin/liner.js",
      args: ["bin/liner.js"],
      env: { LINER_GO_TUI_BIN: shimProbeBin, LINER_TUI: "" },
      expectStdoutIncludes: shimProbeMarker,
    },
    {
      name: "npm shim selects Go TUI",
      cwd: packageDir,
      command: process.execPath,
      commandLabel: "LINER_TUI=go node bin/liner.js --version",
      args: ["bin/liner.js", "--version"],
      env: { LINER_TUI: "go" },
    },
    ...(!args.skipGoTests ? [{
      name: "Go TUI tests",
      cwd: goTuiDir,
      command: goCmd,
      commandLabel: "go test ./...",
      args: ["test", "./..."],
    }] : []),
    ...(!args.skipPack ? [{
      name: "npm package dry-run",
      cwd: packageDir,
      command: npmCmd,
      commandLabel: "npm pack --dry-run",
      args: ["pack", "--dry-run"],
    }] : []),
  ];

  console.log(`[go-tui-acceptance] writing evidence to ${reportPath}`);

  const results = [];
  for (const [index, step] of steps.entries()) {
    console.log(`[go-tui-acceptance] ${index + 1}/${steps.length} ${step.name}`);
    const result = applyStepAssertions(step, await runStep(step, args.verbose));
    results.push({ ...step, ...result });
    if (result.ok) {
      console.log(`[go-tui-acceptance] pass: ${step.name} (${result.durationMs}ms)`);
    } else {
      console.error(`[go-tui-acceptance] fail: ${step.name} (${result.durationMs}ms)`);
    }
  }

  const failed = results.filter((result) => !result.ok);
  const metadata = buildMetadata(evidenceDir);

  await writeFile(reportPath, renderPreflightReport(metadata, results), "utf8");
  console.log(`[go-tui-acceptance] report: ${reportPath}`);

  if (failed.length > 0) {
    console.error(`[go-tui-acceptance] ${failed.length} check(s) failed`);
    process.exit(1);
  }

  console.log("[go-tui-acceptance] all checks passed");
}

async function runReleaseSmoke(args, evidenceDir, stamp) {
  const reportPath = join(evidenceDir, `go-tui-release-smoke-${stamp}.md`);
  const workDir = join(evidenceDir, `release-smoke-${stamp}`);
  const packDir = join(workDir, "pack");
  const consumerDir = join(workDir, "consumer");
  const commandResults = [];
  const checks = [];
  const addCheck = (ok, name, detail) => checks.push({ ok, name, detail });

  await mkdir(packDir, { recursive: true });
  await mkdir(consumerDir, { recursive: true });
  const shimProbeBin = await writeGoTuiShimProbe(workDir, stamp);
  const platformOutDir = join(workDir, "platform");
  const platformPackageDir = join(platformOutDir, platformPackageName);

  console.log(`[go-tui-acceptance] writing evidence to ${reportPath}`);
  console.log(`[go-tui-acceptance] release smoke workspace: ${workDir}`);

  const platformBuildStep = {
    name: "Build local platform package",
    cwd: repoRoot,
    command: process.platform === "win32" ? "python" : "python3",
    commandLabel: `python3 scripts/build-platform-package.py --out-dir ${platformOutDir}`,
    args: ["scripts/build-platform-package.py", "--out-dir", platformOutDir],
  };
  console.log(`[go-tui-acceptance] 1/12 ${platformBuildStep.name}`);
  const platformBuildResult = await runStep(platformBuildStep, args.verbose);
  commandResults.push({ ...platformBuildStep, ...platformBuildResult });
  logStepResult(platformBuildStep.name, platformBuildResult);

  let platformTarballPath = "";
  if (platformBuildResult.ok) {
    const platformPackStep = {
      name: "Pack local platform package",
      cwd: platformPackageDir,
      command: npmCmd,
      commandLabel: `npm pack --json --pack-destination ${packDir}`,
      args: ["pack", "--json", "--pack-destination", packDir],
    };
    console.log(`[go-tui-acceptance] 2/12 ${platformPackStep.name}`);
    const platformPackResult = await runStep(platformPackStep, args.verbose);
    commandResults.push({ ...platformPackStep, ...platformPackResult });
    logStepResult(platformPackStep.name, platformPackResult);

    if (platformPackResult.ok) {
      const parsed = parseNpmPackJson(platformPackResult.stdout);
      addCheck(parsed.ok, "platform npm pack emitted parseable JSON", parsed.ok ? parsed.detail : parsed.message);
      if (parsed.ok) {
        platformTarballPath = resolve(packDir, parsed.value.filename);
        addCheck(existsSync(platformTarballPath), "packed platform tarball exists", platformTarballPath);
        addCheck(
          matchesCanonicalVersion(canonicalVersion, parsed.value.version),
          "platform package version matches canonical release",
          `${parsed.value.version || "missing"} vs ${canonicalVersion}`,
        );
        addPlatformPackFileChecks(parsed.value, addCheck);
      }
    } else {
      addCheck(false, "platform npm pack completed", `exit: ${platformPackResult.exitCode === null ? "not started" : platformPackResult.exitCode}`);
    }
  } else {
    addCheck(false, "platform package build completed", `exit: ${platformBuildResult.exitCode === null ? "not started" : platformBuildResult.exitCode}`);
  }

  const packStep = {
    name: "Pack npm release artifact",
    cwd: packageDir,
    command: npmCmd,
    commandLabel: `npm pack --json --pack-destination ${packDir}`,
    args: ["pack", "--json", "--pack-destination", packDir],
  };
  console.log(`[go-tui-acceptance] 3/12 ${packStep.name}`);
  const packResult = await runStep(packStep, args.verbose);
  commandResults.push({ ...packStep, ...packResult });
  logStepResult(packStep.name, packResult);

  let tarballPath = "";
  let packSummary = null;
  if (packResult.ok) {
    const parsed = parseNpmPackJson(packResult.stdout);
    addCheck(parsed.ok, "npm pack emitted parseable JSON", parsed.ok ? parsed.detail : parsed.message);
    if (parsed.ok) {
      packSummary = parsed.value;
      tarballPath = resolve(packDir, packSummary.filename);
      addCheck(existsSync(tarballPath), "packed tarball exists", tarballPath);
      addCheck(
        matchesCanonicalVersion(canonicalVersion, packSummary.version),
        "main package version matches canonical release",
        `${packSummary.version || "missing"} vs ${canonicalVersion}`,
      );
      addPackFileChecks(packSummary, addCheck);
    }
  } else {
    addCheck(false, "npm pack completed", `exit: ${packResult.exitCode === null ? "not started" : packResult.exitCode}`);
  }

  if (tarballPath && existsSync(tarballPath)) {
    const initStep = {
      name: "Create clean consumer package",
      cwd: consumerDir,
      command: npmCmd,
      commandLabel: "npm init -y",
      args: ["init", "-y"],
    };
    console.log(`[go-tui-acceptance] 4/12 ${initStep.name}`);
    const initResult = await runStep(initStep, args.verbose);
    commandResults.push({ ...initStep, ...initResult });
    logStepResult(initStep.name, initResult);

    const installStep = {
      name: "Install packed artifact",
      cwd: consumerDir,
      command: npmCmd,
      commandLabel: `npm install ${platformTarballPath} ${tarballPath} --ignore-scripts --no-audit --no-fund`,
      args: ["install", platformTarballPath, tarballPath, "--ignore-scripts", "--no-audit", "--no-fund"],
    };
    console.log(`[go-tui-acceptance] 5/12 ${installStep.name}`);
    const installResult = await runStep(installStep, args.verbose);
    commandResults.push({ ...installStep, ...installResult });
    logStepResult(installStep.name, installResult);

    const installedPackageDir = join(consumerDir, "node_modules", packageName);
    const installedPlatformDir = join(consumerDir, "node_modules", platformPackageName);
    const installedShimPath = join(installedPackageDir, "bin", "liner.js");
    const installedCorePath = join(installedPlatformDir, process.platform === "win32" ? "liner.exe" : "liner");
    const installedGoPath = join(installedPlatformDir, packagedGoTuiBin);
    const installedBinPath = join(consumerDir, "node_modules", ".bin", installedLinerBin);
    addCheck(existsSync(installedPackageDir), "installed package directory exists", installedPackageDir);
    addCheck(existsSync(installedPlatformDir), "installed platform package directory exists", installedPlatformDir);
    addCheck(existsSync(installedShimPath), "installed npm shim exists", installedShimPath);
    addCheck(existsSync(installedCorePath), "installed platform Core binary exists", installedCorePath);
    addCheck(existsSync(installedGoPath), "installed platform Go TUI binary exists", installedGoPath);
    addCheck(existsSync(installedBinPath), "installed liner bin link exists", installedBinPath);

    if (installResult.ok) {
      const shimStep = {
        name: "Installed liner shim reports version",
        cwd: consumerDir,
        command: installedBinPath,
        commandLabel: "node_modules/.bin/liner --version",
        args: ["--version"],
      };
      console.log(`[go-tui-acceptance] 6/12 ${shimStep.name}`);
      const shimResult = await runStep(shimStep, args.verbose);
      commandResults.push({ ...shimStep, ...shimResult });
      logStepResult(shimStep.name, shimResult);
      const expectedShimOutput = expectedShimVersion(canonicalVersion);
      addCheck(
        shimResult.ok && matchesCanonicalVersion(expectedShimOutput, shimResult.stdout.trim()),
        "installed launcher and core versions match canonical release",
        `${shimResult.stdout.trim() || "missing"} vs ${expectedShimOutput}`,
      );

      const defaultStep = {
        name: "Installed liner shim defaults to Go TUI",
        cwd: consumerDir,
        command: installedBinPath,
        commandLabel: "LINER_GO_TUI_BIN=<probe> node_modules/.bin/liner",
        args: [],
        env: { LINER_GO_TUI_BIN: shimProbeBin, LINER_TUI: "" },
        expectStdoutIncludes: shimProbeMarker,
      };
      console.log(`[go-tui-acceptance] 7/12 ${defaultStep.name}`);
      const defaultResult = applyStepAssertions(defaultStep, await runStep(defaultStep, args.verbose));
      commandResults.push({ ...defaultStep, ...defaultResult });
      logStepResult(defaultStep.name, defaultResult);

      const goStep = {
        name: "Installed Go TUI binary reports version",
        cwd: consumerDir,
        command: installedGoPath,
        commandLabel: `node_modules/${platformPackageName}/${packagedGoTuiBin} --version`,
        args: ["--version"],
      };
      console.log(`[go-tui-acceptance] 8/12 ${goStep.name}`);
      const goResult = await runStep(goStep, args.verbose);
      commandResults.push({ ...goStep, ...goResult });
      logStepResult(goStep.name, goResult);
      const expectedGoOutput = expectedGoVersion(canonicalVersion);
      addCheck(
        goResult.ok && matchesCanonicalVersion(expectedGoOutput, goResult.stdout.trim()),
        "installed Go TUI version matches canonical release",
        `${goResult.stdout.trim() || "missing"} vs ${expectedGoOutput}`,
      );

      // The launcher only forwards listed commands, so an agent's commands must reach Core.
      const agentHome = join(workDir, "agent-home");
      await mkdir(agentHome, { recursive: true });
      const agentSteps = [
        { name: "Installed CLI answers liner catalog", args: ["catalog", "--root", agentHome, "--json"], expectStdoutIncludes: '"liner.catalog"' },
        { name: "Installed CLI answers liner library", args: ["library", "--root", agentHome], expectStdoutIncludes: '"liner.library_view"' },
        { name: "Installed CLI answers liner handoff", args: ["handoff", "start"], expectStdoutIncludes: '"liner.handoff"' },
      ];
      for (const agentStep of agentSteps) {
        const step = {
          ...agentStep,
          cwd: consumerDir,
          command: installedBinPath,
          commandLabel: `node_modules/.bin/liner ${agentStep.args.join(" ")}`,
        };
        const result = applyStepAssertions(step, await runStep(step, args.verbose));
        commandResults.push({ ...step, ...result });
        logStepResult(step.name, result);
      }

      const adapterHome = join(workDir, "adapter-home");
      const adapterSkillDir = join(adapterHome, ".codex", "skills", "liner-maintenance");
      const adapterUserNote = join(adapterSkillDir, "NOTES.md");
      await mkdir(adapterSkillDir, { recursive: true });
      await writeFile(adapterUserNote, "User-owned adapter note.\n", "utf8");
      const adapterSteps = [
        {
          name: "Installed CLI installs optional maintenance adapter",
          args: ["adapters", "install", "codex", "--home", adapterHome, "--yes", "--json"],
          expectStdoutIncludes: '"action": "install"',
        },
        {
          name: "Installed CLI inspects optional maintenance adapter",
          args: ["adapters", "inspect", "codex", "--home", adapterHome, "--json"],
          expectStdoutIncludes: '"status": "current"',
        },
        {
          name: "Installed CLI replays current adapter update",
          args: ["adapters", "update", "codex", "--home", adapterHome, "--json"],
          expectStdoutIncludes: '"replayed": true',
        },
        {
          name: "Installed CLI removes optional maintenance adapter",
          args: ["adapters", "remove", "codex", "--home", adapterHome, "--yes", "--json"],
          expectStdoutIncludes: '"action": "remove"',
        },
      ];
      for (const [index, adapterStep] of adapterSteps.entries()) {
        const step = {
          ...adapterStep,
          cwd: consumerDir,
          command: installedBinPath,
          commandLabel: `node_modules/.bin/liner ${adapterStep.args.join(" ")}`,
        };
        console.log(`[go-tui-acceptance] ${index + 9}/12 ${step.name}`);
        const result = applyStepAssertions(step, await runStep(step, args.verbose));
        commandResults.push({ ...step, ...result });
        logStepResult(step.name, result);
        if (index === 0) {
          const installedSkillPath = join(adapterSkillDir, "SKILL.md");
          const installedSkillBody = existsSync(installedSkillPath) ? readFileSync(installedSkillPath, "utf8") : "";
          addCheck(
            installedSkillBody.includes("liner project guidance"),
            "installed optional skill routes to CLI guidance",
            installedSkillBody.includes("liner project guidance") ? "routing present" : "routing missing",
          );
        }
      }
      addCheck(
        existsSync(adapterUserNote) && readFileSync(adapterUserNote, "utf8") === "User-owned adapter note.\n",
        "optional skill update/remove preserves user-owned sibling content",
        existsSync(adapterUserNote) ? "preserved" : "removed",
      );
    }
  }

  const metadata = buildMetadata(evidenceDir);
  await writeFile(reportPath, renderReleaseSmokeReport(metadata, workDir, tarballPath, commandResults, checks), "utf8");
  console.log(`[go-tui-acceptance] report: ${reportPath}`);

  const failedCommands = commandResults.filter((result) => !result.ok);
  const failedChecks = checks.filter((result) => !result.ok);
  if (failedCommands.length > 0 || failedChecks.length > 0) {
    console.error(`[go-tui-acceptance] ${failedCommands.length + failedChecks.length} release smoke check(s) failed`);
    process.exit(1);
  }

  console.log("[go-tui-acceptance] release smoke passed");
}

function parsePreflightArgs(raw) {
  const parsed = {
    dir: "",
    help: false,
    skipBuild: false,
    skipGoTests: false,
    skipPack: false,
    verbose: false,
  };
  for (let i = 0; i < raw.length; i += 1) {
    const arg = raw[i];
    if (arg === "--help" || arg === "-h") parsed.help = true;
    else if (arg === "--skip-build") parsed.skipBuild = true;
    else if (arg === "--skip-go-tests") parsed.skipGoTests = true;
    else if (arg === "--skip-pack") parsed.skipPack = true;
    else if (arg === "--verbose") parsed.verbose = true;
    else if (arg === "--dir") {
      i += 1;
      parsed.dir = raw[i] || "";
    } else if (arg.startsWith("--dir=")) {
      parsed.dir = arg.slice("--dir=".length);
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  return parsed;
}

function parseReleaseArgs(raw) {
  const parsed = {
    dir: "",
    help: false,
    verbose: false,
  };
  for (let i = 0; i < raw.length; i += 1) {
    const arg = raw[i];
    if (arg === "--help" || arg === "-h") parsed.help = true;
    else if (arg === "--verbose") parsed.verbose = true;
    else if (arg === "--dir") {
      i += 1;
      parsed.dir = raw[i] || "";
    } else if (arg.startsWith("--dir=")) {
      parsed.dir = arg.slice("--dir=".length);
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  return parsed;
}

function printHelp() {
  console.log(`Usage:
  npm run acceptance:go -- [options]
  npm run acceptance:go -- release-smoke [--dir <path>] [--verbose]

Options:
  --dir <path>       Evidence directory. Defaults to LINER_ACCEPTANCE_DIR or /tmp/liner-go-live-acceptance.
  --skip-build       Do not run npm run build:package.
  --skip-go-tests    Do not run go test ./...
  --skip-pack        Do not run npm pack --dry-run.
  --verbose          Stream command output while still writing the report.
  -h, --help         Show this help.

release-smoke options:
  --dir <path>       Evidence directory for the release smoke workspace and report.
  --verbose          Stream command output while still writing the report.
`);
}

function resolveUserPath(value) {
  if (value === "~") return homedir();
  if (value.startsWith("~/")) return join(homedir(), value.slice(2));
  return resolve(value);
}

function runStep(step, verbose) {
  const started = Date.now();
  return new Promise((resolveStep) => {
    const child = spawn(step.command, step.args, {
      cwd: step.cwd,
      env: { ...process.env, ...(step.env || {}) },
      shell: false,
    });
    let stdout = "";
    let stderr = "";

    child.stdout?.on("data", (chunk) => {
      const text = chunk.toString();
      stdout += text;
      if (verbose) process.stdout.write(text);
    });
    child.stderr?.on("data", (chunk) => {
      const text = chunk.toString();
      stderr += text;
      if (verbose) process.stderr.write(text);
    });
    child.on("error", (error) => {
      resolveStep({
        ok: false,
        exitCode: null,
        signal: null,
        durationMs: Date.now() - started,
        stdout,
        stderr,
        error: error.message,
      });
    });
    child.on("close", (exitCode, signal) => {
      resolveStep({
        ok: exitCode === 0,
        exitCode,
        signal,
        durationMs: Date.now() - started,
        stdout,
        stderr,
        error: "",
      });
    });
  });
}

function applyStepAssertions(step, result) {
  if (!result.ok || !step.expectStdoutIncludes) return result;
  if (result.stdout.includes(step.expectStdoutIncludes)) return result;
  return {
    ...result,
    ok: false,
    error: `stdout did not include ${JSON.stringify(step.expectStdoutIncludes)}`,
  };
}

function logStepResult(name, result) {
  if (result.ok) {
    console.log(`[go-tui-acceptance] pass: ${name} (${result.durationMs}ms)`);
  } else {
    console.error(`[go-tui-acceptance] fail: ${name} (${result.durationMs}ms)`);
  }
}

async function writeGoTuiShimProbe(parentDir, stamp) {
  const probeDir = join(parentDir, `shim-probe-${stamp}`);
  await mkdir(probeDir, { recursive: true });
  if (process.platform === "win32") {
    const probePath = join(probeDir, "liner-tui-probe.cmd");
    await writeFile(probePath, `@echo off\r\necho ${shimProbeMarker}\r\n`, "utf8");
    return probePath;
  }

  const probePath = join(probeDir, "liner-tui-probe");
  await writeFile(probePath, `#!/bin/sh\nprintf '%s\\n' '${shimProbeMarker}'\n`, "utf8");
  await chmod(probePath, 0o755);
  return probePath;
}

function parseNpmPackJson(stdout) {
  const text = stdout.trim();
  if (text === "") return { ok: false, message: "npm pack produced no stdout" };
  const start = text.indexOf("[");
  const end = text.lastIndexOf("]");
  if (start === -1 || end === -1 || end < start) {
    return { ok: false, message: "npm pack stdout did not contain a JSON array" };
  }
  try {
    const value = JSON.parse(text.slice(start, end + 1));
    if (!Array.isArray(value) || value.length === 0 || !isRecord(value[0])) {
      return { ok: false, message: "npm pack JSON did not contain a package summary" };
    }
    const summary = value[0];
    if (typeof summary.filename !== "string" || summary.filename.trim() === "") {
      return { ok: false, message: "npm pack JSON did not contain a filename" };
    }
    return {
      ok: true,
      value: summary,
      detail: `${summary.filename} (${Array.isArray(summary.files) ? summary.files.length : "unknown"} files)`,
    };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function addPackFileChecks(summary, addCheck) {
  const files = Array.isArray(summary.files) ? summary.files.filter(isRecord) : [];
  const paths = new Set(files.map((file) => (typeof file.path === "string" ? file.path : "")));
  const requiredPaths = [
    "bin/liner.js",
    "package.json",
  ];
  for (const requiredPath of requiredPaths) {
    addCheck(paths.has(requiredPath), `tarball includes ${requiredPath}`, paths.has(requiredPath) ? "present" : "missing");
  }
  addCheck(!paths.has(`bin/${packagedGoTuiBin}`), `tarball excludes bin/${packagedGoTuiBin}`, paths.has(`bin/${packagedGoTuiBin}`) ? "present" : "absent");
}

function addPlatformPackFileChecks(summary, addCheck) {
  const files = Array.isArray(summary.files) ? summary.files.filter(isRecord) : [];
  const paths = new Set(files.map((file) => (typeof file.path === "string" ? file.path : "")));
  const requiredPaths = [
    process.platform === "win32" ? "liner.exe" : "liner",
    packagedGoTuiBin,
    "_internal/",
    "package.json",
  ];
  for (const requiredPath of requiredPaths) {
    const present = requiredPath.endsWith("/")
      ? Array.from(paths).some((path) => path.startsWith(requiredPath))
      : paths.has(requiredPath);
    addCheck(present, `platform tarball includes ${requiredPath}`, present ? "present" : "missing");
  }
}

function captureText(command, args, cwd) {
  const result = spawnSync(command, args, { cwd, encoding: "utf8" });
  if (result.status !== 0) return "unknown";
  return (result.stdout || "").trim() || "unknown";
}

function buildMetadata(evidenceDir) {
  return {
    date: new Date().toISOString(),
    repoRoot,
    packageDir,
    evidenceDir,
    branch: captureText("git", ["branch", "--show-current"], repoRoot),
    commit: captureText("git", ["rev-parse", "--short", "HEAD"], repoRoot),
  };
}

function renderPreflightReport(metadata, results) {
  const status = results.every((result) => result.ok) ? "PASS" : "FAIL";
  const lines = [
    "# Go TUI Acceptance Preflight",
    "",
    `Status: ${status}`,
    `Date: ${metadata.date}`,
    `Branch: ${metadata.branch}`,
    `Commit: ${metadata.commit}`,
    `Repo: \`${metadata.repoRoot}\``,
    `Package: \`${metadata.packageDir}\``,
    `Evidence dir: \`${metadata.evidenceDir}\``,
    "",
    "This report covers package and shim preflight only. It does not replace manual live agent passes documented from `docs/tui/GO_TERMINAL_AGENT_HANDOFF_2026-06-16.md`.",
    "",
    "## Summary",
    "",
    ...results.map((result) => `- ${result.ok ? "[x]" : "[ ]"} ${result.name} (${result.durationMs}ms)`),
    "",
    "## Commands",
    "",
  ];

  results.forEach((result, index) => {
    lines.push(
      `### ${index + 1}. ${result.name}`,
      "",
      `- cwd: \`${result.cwd}\``,
      `- command: \`${result.commandLabel}\``,
      `- exit: ${result.exitCode === null ? "not started" : result.exitCode}`,
      `- signal: ${result.signal || "none"}`,
      `- duration: ${result.durationMs}ms`,
    );
    if (result.error) lines.push(`- error: \`${result.error}\``);
    lines.push("", "stdout:", "", fenced(result.stdout), "", "stderr:", "", fenced(result.stderr), "");
  });

  return `${lines.join("\n")}\n`;
}

function renderReleaseSmokeReport(metadata, workDir, tarballPath, commandResults, checks) {
  const status = commandResults.every((result) => result.ok) && checks.every((result) => result.ok) ? "PASS" : "FAIL";
  const lines = [
    "# Go TUI Release Smoke",
    "",
    `Status: ${status}`,
    `Date: ${metadata.date}`,
    `Branch: ${metadata.branch}`,
    `Commit: ${metadata.commit}`,
    `Repo: \`${metadata.repoRoot}\``,
    `Package: \`${metadata.packageDir}\``,
    `Evidence dir: \`${metadata.evidenceDir}\``,
    `Workspace: \`${workDir}\``,
    `Tarball: \`${tarballPath || "not created"}\``,
    "",
    "This report verifies the packed npm artifact from a clean consumer install. It complements the source-tree preflight and live Go TUI project runs.",
    "",
    "## Checks",
    "",
    ...checks.map((result) => `- ${result.ok ? "[x]" : "[ ]"} ${result.name}: ${result.detail}`),
    "",
    "## Commands",
    "",
  ];

  commandResults.forEach((result, index) => {
    lines.push(
      `### ${index + 1}. ${result.name}`,
      "",
      `- cwd: \`${result.cwd}\``,
      `- command: \`${result.commandLabel}\``,
      `- exit: ${result.exitCode === null ? "not started" : result.exitCode}`,
      `- signal: ${result.signal || "none"}`,
      `- duration: ${result.durationMs}ms`,
    );
    if (result.error) lines.push(`- error: \`${result.error}\``);
    lines.push("", "stdout:", "", fenced(result.stdout), "", "stderr:", "", fenced(result.stderr), "");
  });

  return `${lines.join("\n")}\n`;
}

function fenced(value) {
  const text = value.trim();
  if (text === "") return "_(empty)_";
  return `\`\`\`text\n${text.replaceAll("```", "`\\`\\`")}\n\`\`\``;
}
