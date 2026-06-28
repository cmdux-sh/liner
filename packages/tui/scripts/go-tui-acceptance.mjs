#!/usr/bin/env node
// Acceptance checks for the Go TUI default-switch runbook.
//
// This does not drive the interactive agent runs. With no subcommand it
// automates the repeatable shim and Go test preflight. The `release-smoke`
// subcommand packs the current platform package plus the main npm package and
// installs both into a clean consumer project. With
// `verify-project`, it audits artifacts produced by a live/manual run.

import { spawn, spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { chmod, mkdir, writeFile } from "node:fs/promises";
import { basename, dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { homedir } from "node:os";
import * as YAML from "yaml";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");
const repoRoot = resolve(packageDir, "..", "..");
const goTuiDir = join(repoRoot, "packages", "go-tui");
const npmCmd = process.platform === "win32" ? "npm.cmd" : "npm";
const goCmd = process.platform === "win32" ? "go.exe" : "go";
const linerTuiBin = join(packageDir, "bin", process.platform === "win32" ? "liner-tui.exe" : "liner-tui");
const installedLinerBin = process.platform === "win32" ? "liner.cmd" : "liner";
const packagedGoTuiBin = process.platform === "win32" ? "liner-tui.exe" : "liner-tui";
const packageName = "linersh";
const platformPackageName = `linersh-${process.platform}-${process.arch}`;
const shimProbeMarker = "liner-go-tui-shim-probe";

const knownSubcommands = new Set(["release-smoke", "verify-project"]);
const subcommand = knownSubcommands.has(process.argv[2]) ? process.argv[2] : "preflight";
const rawArgs = subcommand === "preflight" ? process.argv.slice(2) : process.argv.slice(3);

let args;
try {
  if (subcommand === "verify-project") {
    args = parseVerifyArgs(rawArgs);
  } else if (subcommand === "release-smoke") {
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

if (subcommand === "verify-project") {
  await runProjectVerification(args, evidenceDir, stamp);
} else if (subcommand === "release-smoke") {
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

async function runProjectVerification(args, evidenceDir, stamp) {
  const project = resolveUserPath(args.project);
  const reportPath = join(evidenceDir, `go-tui-project-${basename(project)}-${args.expect}-${stamp}.md`);
  await mkdir(evidenceDir, { recursive: true });

  console.log(`[go-tui-acceptance] verifying ${project}`);
  console.log(`[go-tui-acceptance] writing evidence to ${reportPath}`);

  const results = verifyProject(project, args.expect);
  await writeFile(reportPath, renderProjectReport(buildMetadata(evidenceDir), project, args.expect, results), "utf8");
  console.log(`[go-tui-acceptance] report: ${reportPath}`);

  const failed = results.filter((result) => !result.ok);
  if (failed.length > 0) {
    for (const result of failed) {
      console.error(`[go-tui-acceptance] fail: ${result.name} — ${result.detail}`);
    }
    process.exit(1);
  }

  console.log("[go-tui-acceptance] project verification passed");
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
  console.log(`[go-tui-acceptance] 1/8 ${platformBuildStep.name}`);
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
    console.log(`[go-tui-acceptance] 2/8 ${platformPackStep.name}`);
    const platformPackResult = await runStep(platformPackStep, args.verbose);
    commandResults.push({ ...platformPackStep, ...platformPackResult });
    logStepResult(platformPackStep.name, platformPackResult);

    if (platformPackResult.ok) {
      const parsed = parseNpmPackJson(platformPackResult.stdout);
      addCheck(parsed.ok, "platform npm pack emitted parseable JSON", parsed.ok ? parsed.detail : parsed.message);
      if (parsed.ok) {
        platformTarballPath = resolve(packDir, parsed.value.filename);
        addCheck(existsSync(platformTarballPath), "packed platform tarball exists", platformTarballPath);
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
  console.log(`[go-tui-acceptance] 3/8 ${packStep.name}`);
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
    console.log(`[go-tui-acceptance] 4/8 ${initStep.name}`);
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
    console.log(`[go-tui-acceptance] 5/8 ${installStep.name}`);
    const installResult = await runStep(installStep, args.verbose);
    commandResults.push({ ...installStep, ...installResult });
    logStepResult(installStep.name, installResult);

    const installedPackageDir = join(consumerDir, "node_modules", packageName);
    const installedPlatformDir = join(consumerDir, "node_modules", platformPackageName);
    const installedShimPath = join(installedPackageDir, "bin", "liner.js");
    const installedGoPath = join(installedPlatformDir, packagedGoTuiBin);
    const installedBinPath = join(consumerDir, "node_modules", ".bin", installedLinerBin);
    addCheck(existsSync(installedPackageDir), "installed package directory exists", installedPackageDir);
    addCheck(existsSync(installedPlatformDir), "installed platform package directory exists", installedPlatformDir);
    addCheck(existsSync(installedShimPath), "installed npm shim exists", installedShimPath);
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
      console.log(`[go-tui-acceptance] 6/8 ${shimStep.name}`);
      const shimResult = await runStep(shimStep, args.verbose);
      commandResults.push({ ...shimStep, ...shimResult });
      logStepResult(shimStep.name, shimResult);

      const defaultStep = {
        name: "Installed liner shim defaults to Go TUI",
        cwd: consumerDir,
        command: installedBinPath,
        commandLabel: "LINER_GO_TUI_BIN=<probe> node_modules/.bin/liner",
        args: [],
        env: { LINER_GO_TUI_BIN: shimProbeBin, LINER_TUI: "" },
        expectStdoutIncludes: shimProbeMarker,
      };
      console.log(`[go-tui-acceptance] 7/8 ${defaultStep.name}`);
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
      console.log(`[go-tui-acceptance] 8/8 ${goStep.name}`);
      const goResult = await runStep(goStep, args.verbose);
      commandResults.push({ ...goStep, ...goResult });
      logStepResult(goStep.name, goResult);
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

function parseVerifyArgs(raw) {
  const parsed = {
    dir: "",
    expect: "methodology-draft",
    help: false,
    project: "",
  };
  for (let i = 0; i < raw.length; i += 1) {
    const arg = raw[i];
    if (arg === "--help" || arg === "-h") parsed.help = true;
    else if (arg === "--project") {
      i += 1;
      parsed.project = raw[i] || "";
    } else if (arg.startsWith("--project=")) {
      parsed.project = arg.slice("--project=".length);
    } else if (arg === "--expect") {
      i += 1;
      parsed.expect = raw[i] || "";
    } else if (arg.startsWith("--expect=")) {
      parsed.expect = arg.slice("--expect=".length);
    } else if (arg === "--dir") {
      i += 1;
      parsed.dir = raw[i] || "";
    } else if (arg.startsWith("--dir=")) {
      parsed.dir = arg.slice("--dir=".length);
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  if (!parsed.help && parsed.project.trim() === "") {
    throw new Error("verify-project requires --project <path>");
  }
  const expects = new Set(["methodology-draft", "framing-complete", "claude-framing", "candidate-resume", "evaluation-active", "evaluation-complete", "failure-retry", "accepted-draft", "compiled", "local-compiled", "imported-compiled", "custom-sources", "custom-compiled"]);
  if (!parsed.help && !expects.has(parsed.expect)) {
    throw new Error(`Unknown --expect value: ${parsed.expect}`);
  }
  return parsed;
}

function printHelp() {
  console.log(`Usage:
  npm run acceptance:go -- [options]
  npm run acceptance:go -- release-smoke [--dir <path>] [--verbose]
  npm run acceptance:go -- verify-project --project <path> [--expect methodology-draft|framing-complete|claude-framing|candidate-resume|evaluation-active|evaluation-complete|failure-retry|accepted-draft|compiled|local-compiled|imported-compiled|custom-sources|custom-compiled]

Options:
  --dir <path>       Evidence directory. Defaults to LINER_ACCEPTANCE_DIR or /tmp/liner-go-live-acceptance.
  --skip-build       Do not run npm run build:package.
  --skip-go-tests    Do not run go test ./...
  --skip-pack        Do not run npm pack --dry-run.
  --verbose          Stream command output while still writing the report.
  -h, --help         Show this help.

verify-project options:
  --project <path>   Project folder to audit.
  --expect <state>   methodology-draft, framing-complete, claude-framing, candidate-resume, evaluation-active, evaluation-complete, failure-retry, accepted-draft, compiled, local-compiled, imported-compiled, custom-sources, or custom-compiled. Defaults to methodology-draft.
  --dir <path>       Evidence directory for the project verification report.

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

function addPackFileChecks(summary, addCheck) {
  const files = Array.isArray(summary.files) ? summary.files.filter(isRecord) : [];
  const paths = new Set(files.map((file) => (typeof file.path === "string" ? file.path : "")));
  const requiredPaths = [
    "bin/liner.js",
    "dist/agents/headless-runner.js",
    "cli-update-docs/SKILL.md",
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

function verifyProject(project, expect) {
  const checks = [];
  const add = (ok, name, detail) => checks.push({ ok, name, detail });
  const exists = (rel) => existsSync(join(project, rel));

  add(existsSync(project) && statSync(project).isDirectory(), "project directory exists", project);
  const tape = readYaml(join(project, "tape.yaml"));
  add(tape.ok, "tape.yaml parses", tape.ok ? "YAML parsed" : tape.message);
  if (tape.ok) {
    add(isRecord(tape.value), "tape.yaml is a mapping", isRecord(tape.value) ? "mapping" : typeof tape.value);
  }
  const tapeValue = tape.ok && isRecord(tape.value) ? tape.value : {};
  if (tape.ok && isRecord(tape.value)) {
    add(typeof tapeValue.title === "string" && tapeValue.title.trim() !== "", "tape title exists", tapeValue.title || "(empty)");
    add(Array.isArray(tapeValue.sources), "tape sources list exists", `sources: ${Array.isArray(tapeValue.sources) ? tapeValue.sources.length : "missing"}`);
  }

  if (
    expect !== "custom-sources" &&
    expect !== "imported-compiled" &&
    expect !== "framing-complete" &&
    expect !== "claude-framing" &&
    expect !== "candidate-resume" &&
    expect !== "failure-retry" &&
    expect !== "evaluation-active" &&
    expect !== "evaluation-complete"
  ) {
    for (const artifact of ["synthesis.md"]) {
      add(exists(artifact), `${artifact} exists`, join(project, artifact));
    }
  }

  if (
    expect !== "local-compiled" &&
    expect !== "imported-compiled" &&
    expect !== "custom-sources" &&
    expect !== "framing-complete" &&
    expect !== "claude-framing" &&
    expect !== "candidate-resume" &&
    expect !== "failure-retry" &&
    expect !== "evaluation-active" &&
    expect !== "evaluation-complete"
  ) {
    for (const artifact of [
      "working/01-jtbd-and-knowledge-map.md",
      "working/02-candidate-longlist.md",
      "working/03-evaluation.yaml",
      "working/04-quality-checks.md",
    ]) {
      add(exists(artifact), `${artifact} exists`, join(project, artifact));
    }

    const evaluation = readYaml(join(project, "working/03-evaluation.yaml"));
    const evaluationValue = evaluation.ok && isRecord(evaluation.value) ? evaluation.value : {};
    const candidateCount = Array.isArray(evaluationValue.candidates) ? evaluationValue.candidates.length : 0;
    add(candidateCount > 0, "evaluation candidates exist", `candidates: ${candidateCount}`);

    const phases = ["framing", "candidates", "evaluation", "quality", "synthesis", "assembly"];
    for (const phase of phases) {
      const logs = listFiles(join(project, ".liner-runs", phase), ".jsonl");
      add(logs.length > 0, `.liner-runs/${phase} log exists`, logs.length > 0 ? logs.map((file) => relative(project, file)).join(", ") : "missing");
    }
  }

  if (expect === "methodology-draft") {
    const draft = readYaml(join(project, "working/07-tape-draft.yaml"));
    const draftValue = draft.ok && isRecord(draft.value) ? draft.value : {};
    const draftSources = Array.isArray(draftValue.sources) ? draftValue.sources.length : 0;
    add(draft.ok, "assembly draft parses", draft.ok ? "YAML parsed" : draft.message);
    add(draftSources > 0, "assembly draft has sources", `sources: ${draftSources}`);
    if (tape.ok && isRecord(tape.value) && Array.isArray(tapeValue.sources)) {
      add(tapeValue.sources.length === 0, "tape sources are still unaccepted", `sources: ${tapeValue.sources.length}`);
    }
    add(!exists("MIXTAPE.md"), "MIXTAPE.md not compiled yet", exists("MIXTAPE.md") ? "present" : "absent");
  }

  if (expect === "framing-complete") {
    verifyFramingComplete(project, exists, add);
  }

  if (expect === "claude-framing") {
    verifyFramingComplete(project, exists, add);
    verifyClaudeFraming(project, add);
  }

  if (expect === "candidate-resume") {
    verifyFramingComplete(project, exists, add);
    verifyCandidateResume(project, add);
  }

  if (expect === "failure-retry") {
    verifyFailureRetry(project, exists, add);
  }

  if (expect === "evaluation-active") {
    verifyFramingComplete(project, exists, add);
    verifyCandidatesComplete(project, add);
    verifyEvaluationActive(project, exists, add);
  }

  if (expect === "evaluation-complete") {
    verifyFramingComplete(project, exists, add);
    verifyCandidatesComplete(project, add);
    verifyEvaluationComplete(project, exists, add);
  }

  if (expect === "custom-sources") {
    verifyCustomSources(project, tapeValue, add);
    add(!exists("working/07-tape-draft.yaml"), "assembly draft not created yet", exists("working/07-tape-draft.yaml") ? "present" : "absent");
    add(!exists("MIXTAPE.md"), "MIXTAPE.md not compiled yet", exists("MIXTAPE.md") ? "present" : "absent");
  }

  if (expect === "custom-compiled") {
    verifyCustomSources(project, tapeValue, add);
    verifyCustomCompiled(project, add);
  }

  if (expect === "accepted-draft" || expect === "compiled" || expect === "local-compiled" || expect === "imported-compiled" || expect === "custom-compiled") {
    if (tape.ok && isRecord(tape.value) && Array.isArray(tapeValue.sources)) {
      add(tapeValue.sources.length > 0, "accepted sources exist in tape.yaml", `sources: ${tapeValue.sources.length}`);
    }
    add(!exists("working/07-tape-draft.yaml"), "accepted draft removed", exists("working/07-tape-draft.yaml") ? "still present" : "absent");
  }

  if (expect === "compiled" || expect === "local-compiled" || expect === "imported-compiled" || expect === "custom-compiled") {
    add(exists("MIXTAPE.md"), "MIXTAPE.md exists", join(project, "MIXTAPE.md"));
    add(listFiles(join(project, "sources"), ".md").length > 0, "compiled source files exist", `${listFiles(join(project, "sources"), ".md").length} markdown files`);
  }

  if (expect === "imported-compiled") {
    verifyImportedCompiled(project, tapeValue, add);
  }

  return checks;
}

function verifyImportedCompiled(project, tapeValue, add) {
  add(!basename(project).endsWith(".mixtape"), "project is an extracted folder path", basename(project));
  add(existsSync(join(project, "working")), "working artifacts were imported", join(project, "working"));
  add(existsSync(join(project, "local-sources")), "local-sources folder exists after import", join(project, "local-sources"));
  if (existsSync(join(project, ".liner-progress.json"))) {
    const progress = readJson(join(project, ".liner-progress.json"));
    add(progress.ok, ".liner-progress.json parses when present", progress.ok ? "JSON parsed" : progress.message);
  }
  const tapeSources = Array.isArray(tapeValue.sources) ? tapeValue.sources.filter(isRecord) : [];
  const sourceFiles = listFiles(join(project, "sources"), ".md");
  add(sourceFiles.length >= Math.min(tapeSources.length, 1), "compiled files survived no-refetch import", `source files: ${sourceFiles.length}, tape sources: ${tapeSources.length}`);
}

function verifyCustomSources(project, tapeValue, add) {
  const tapeSources = Array.isArray(tapeValue.sources) ? tapeValue.sources.filter(isRecord) : [];
  add(tapeSources.length > 0, "active custom sources accepted in tape.yaml", `sources: ${tapeSources.length}`);
  const customTapeSources = tapeSources.filter((source) => source.type === "local_file" || source.type === "skill");

  const types = new Set(tapeSources.map((source) => source.type).filter((type) => typeof type === "string"));
  for (const type of ["web", "youtube", "local_file", "skill"]) {
    add(types.has(type), `tape.yaml includes ${type} source`, types.has(type) ? "present" : `types: ${Array.from(types).join(", ") || "none"}`);
  }

  const localPaths = tapeSources
    .map((source) => (typeof source.path === "string" ? source.path : ""))
    .filter((sourcePath) => sourcePath.startsWith("local-sources/"));
  add(localPaths.length > 0, "local source paths exist in tape.yaml", localPaths.length > 0 ? localPaths.join(", ") : "missing");
  for (const sourcePath of localPaths.slice(0, 8)) {
    add(existsSync(join(project, sourcePath)), `${sourcePath} exists`, join(project, sourcePath));
  }
  const capturedPaths = localPaths.filter((sourcePath) => sourcePath.startsWith("local-sources/captured/"));
  add(capturedPaths.length > 0, "captured article source accepted", capturedPaths.length > 0 ? capturedPaths.join(", ") : "missing");
  const copiedLocalPaths = localPaths.filter((sourcePath) => !sourcePath.startsWith("local-sources/captured/") && !sourcePath.startsWith("local-sources/skills/"));
  add(copiedLocalPaths.length > 0, "copied local file source accepted", copiedLocalPaths.length > 0 ? copiedLocalPaths.join(", ") : "missing");

  const manifest = readYaml(join(project, "local-sources", "sources-manifest.yaml"));
  add(manifest.ok, "local-sources/sources-manifest.yaml parses", manifest.ok ? "YAML parsed" : manifest.message);
  const manifestValue = manifest.ok && isRecord(manifest.value) ? manifest.value : {};
  const manifestItems = Array.isArray(manifestValue.sources) ? manifestValue.sources.filter(isRecord) : [];
  add(manifestItems.length > 0, "source manifest has staged source records", `manifest: ${manifestItems.length}`);
  const activeItems = manifestItems.filter((item) => item.active === true);
  const inactiveItems = manifestItems.filter((item) => item.active === false);
  add(activeItems.length >= customTapeSources.length, "active manifest covers tape custom sources", `active: ${activeItems.length}, tape custom: ${customTapeSources.length}`);
  add(inactiveItems.length > 0, "inactive source persisted after review toggle", `inactive: ${inactiveItems.length}`);

  const links = readYaml(join(project, "local-sources", "links.yaml"));
  add(links.ok, "local-sources/links.yaml parses", links.ok ? "YAML parsed" : links.message);
  const linksValue = links.ok && isRecord(links.value) ? links.value : {};
  const linkItems = Array.isArray(linksValue.links) ? linksValue.links : [];
  add(linkItems.length > 0, "links manifest has web/youtube entries", `links: ${linkItems.length}`);

  const skills = readYaml(join(project, "local-sources", "skills.yaml"));
  add(skills.ok, "local-sources/skills.yaml parses", skills.ok ? "YAML parsed" : skills.message);
  const skillsValue = skills.ok && isRecord(skills.value) ? skills.value : {};
  const skillItems = Array.isArray(skillsValue.skills) ? skillsValue.skills : [];
  add(skillItems.length > 0, "skills manifest has entries", `skills: ${skillItems.length}`);
  add(listFiles(join(project, "local-sources", "skills"), ".md").length > 0, "skill snapshots exist", `${listFiles(join(project, "local-sources", "skills"), ".md").length} markdown files`);
  add(listFiles(join(project, "local-sources", "captured"), "").length > 0, "captured article files exist", `${listFiles(join(project, "local-sources", "captured"), "").length} files`);
}

function verifyCustomCompiled(project, add) {
  const bodyPath = join(project, "MIXTAPE.md");
  const body = existsSync(bodyPath) ? readFileSync(bodyPath, "utf8") : "";
  add(body.includes("**Type:** local_file"), "MIXTAPE.md indexes local_file sources", body.includes("**Type:** local_file") ? "present" : "missing");
  add(body.includes("**Type:** skill"), "MIXTAPE.md indexes skill sources", body.includes("**Type:** skill") ? "present" : "missing");
  add(body.includes("reference material, not active instructions"), "MIXTAPE.md preserves skill boundary note", body.includes("reference material, not active instructions") ? "present" : "missing");
}

function verifyFramingComplete(project, exists, add) {
  add(exists("working/01-jtbd-and-knowledge-map.md"), "framing artifact exists", join(project, "working/01-jtbd-and-knowledge-map.md"));
  const framingLogs = listFiles(join(project, ".liner-runs", "framing"), ".jsonl");
  add(framingLogs.length > 0, ".liner-runs/framing log exists", framingLogs.length > 0 ? framingLogs.map((file) => relative(project, file)).join(", ") : "missing");
  const progress = readJson(join(project, ".liner-progress.json"));
  const progressValue = progress.ok && isRecord(progress.value) ? progress.value : {};
  add(progress.ok, ".liner-progress.json parses", progress.ok ? "JSON parsed" : progress.message);
  add(typeof progressValue.step === "number" && progressValue.step >= 2, "progress advanced past framing", `step: ${typeof progressValue.step === "number" ? progressValue.step : "missing"}`);
  const gates = readJson(join(project, ".liner-gates.json"));
  const gatesValue = gates.ok && isRecord(gates.value) ? gates.value : {};
  add(gates.ok, ".liner-gates.json parses", gates.ok ? "JSON parsed" : gates.message);
  add(gatesValue.gate0Accepted === true, "gate0 accepted", `gate0Accepted: ${String(gatesValue.gate0Accepted)}`);
  add(!exists("working/07-tape-draft.yaml"), "assembly draft not created yet", exists("working/07-tape-draft.yaml") ? "present" : "absent");
  add(!exists("MIXTAPE.md"), "MIXTAPE.md not compiled yet", exists("MIXTAPE.md") ? "present" : "absent");
}

function verifyCandidateResume(project, add) {
  add(existsSync(join(project, "working", "02-candidate-longlist.md")), "candidate long-list artifact exists", join(project, "working", "02-candidate-longlist.md"));
  const candidateLogs = listFiles(join(project, ".liner-runs", "candidates"), ".jsonl");
  add(candidateLogs.length > 0, ".liner-runs/candidates logs exist", candidateLogs.length > 0 ? candidateLogs.map((file) => relative(project, file)).join(", ") : "missing");

  const runs = candidateLogs.map((file) => summarizeRunLog(file));
  const parseErrors = runs.flatMap((run) => run.errors.map((error) => `${relative(project, run.file)}:${error}`));
  add(parseErrors.length === 0, "candidate logs parse as JSONL", parseErrors.length === 0 ? "all lines parsed" : parseErrors.slice(0, 3).join("; "));

  const freshRuns = runs.filter((run) => run.agent === "codex" && run.resume === false && run.threadId);
  const freshThreadIds = new Set(freshRuns.map((run) => run.threadId));
  add(freshThreadIds.size > 0, "fresh Codex candidate thread recorded", freshThreadIds.size > 0 ? Array.from(freshThreadIds).join(", ") : "missing");

  const resumeRuns = runs.filter((run) => run.agent === "codex" && run.resume === true && run.threadId);
  add(resumeRuns.length > 0, "Codex candidate resume log exists", resumeRuns.length > 0 ? resumeRuns.map((run) => relative(project, run.file)).join(", ") : "missing");

  const matchingResumeRuns = resumeRuns.filter((run) => freshThreadIds.has(run.threadId));
  add(
    matchingResumeRuns.length > 0,
    "resume used recorded candidate thread id",
    matchingResumeRuns.length > 0
      ? matchingResumeRuns.map((run) => `${relative(project, run.file)} -> ${run.threadId}`).join(", ")
      : `fresh: ${Array.from(freshThreadIds).join(", ") || "none"}; resume: ${resumeRuns.map((run) => run.threadId).join(", ") || "none"}`,
  );

  const resumedWorkEvents = matchingResumeRuns.reduce((count, run) => count + run.workEvents, 0);
  add(resumedWorkEvents > 0, "resumed candidate run emitted work events", `work events: ${resumedWorkEvents}`);
}

function verifyClaudeFraming(project, add) {
  const framingLogs = listFiles(join(project, ".liner-runs", "framing"), ".jsonl");
  const runs = framingLogs.map((file) => summarizeClaudeRunLog(file));
  add(runs.length > 0, "Claude framing logs exist", runs.length > 0 ? runs.map((run) => relative(project, run.file)).join(", ") : "missing");

  const parseErrors = runs.flatMap((run) => run.errors.map((error) => `${relative(project, run.file)}:${error}`));
  add(parseErrors.length === 0, "Claude framing logs parse as JSONL", parseErrors.length === 0 ? "all lines parsed" : parseErrors.slice(0, 3).join("; "));

  const claudeRuns = runs.filter((run) => run.agent === "claude");
  add(claudeRuns.length > 0, "framing ran with Claude", claudeRuns.length > 0 ? claudeRuns.map((run) => relative(project, run.file)).join(", ") : "missing Claude metadata");

  const cleanRuns = claudeRuns.filter((run) => run.closeExitCode === 0 && run.resultOk === true);
  add(cleanRuns.length > 0, "Claude framing run closed successfully", cleanRuns.length > 0 ? cleanRuns.map((run) => relative(project, run.file)).join(", ") : "missing successful Claude close/result");

  const initRuns = claudeRuns.filter((run) => run.sessionId && run.model && run.version);
  add(initRuns.length > 0, "Claude init recorded model and session", initRuns.length > 0 ? initRuns.map((run) => `${run.model} · ${run.sessionId} · ${run.version}`).join(", ") : "missing init metadata");

  const isolatedRuns = claudeRuns.filter((run) => run.mcpServers === 0);
  add(isolatedRuns.length > 0, "Claude run loaded no MCP servers", isolatedRuns.length > 0 ? isolatedRuns.map((run) => `${relative(project, run.file)} mcp_servers=0`).join(", ") : claudeRuns.map((run) => `${relative(project, run.file)} mcp_servers=${run.mcpServers}`).join(", "));

  const expectedInitTools = ["Edit", "Glob", "Grep", "Read", "WebFetch", "Write"];
  const scopedToolRuns = claudeRuns.filter((run) => sameStringSet(run.initTools, expectedInitTools));
  add(
    scopedToolRuns.length > 0,
    "Claude init tool surface is methodology-scoped",
    scopedToolRuns.length > 0
      ? scopedToolRuns.map((run) => `${relative(project, run.file)} tools=${run.initTools.join("/")}`).join(", ")
      : claudeRuns.map((run) => `${relative(project, run.file)} tools=${run.initTools.join("/") || "(none)"}`).join(", "),
  );

  const forbiddenInitTools = new Set(["Bash", "Task", "ToolSearch", "WebSearch"]);
  const forbiddenToolRuns = claudeRuns.filter((run) => run.initTools.some((tool) => forbiddenInitTools.has(tool)));
  add(
    forbiddenToolRuns.length === 0,
    "Claude init excludes shell/search/subagent tools",
    forbiddenToolRuns.length === 0
      ? "Bash/Task/ToolSearch/WebSearch absent"
      : forbiddenToolRuns.map((run) => `${relative(project, run.file)} tools=${run.initTools.join("/")}`).join(", "),
  );

  const readWriteRuns = claudeRuns.filter((run) => run.toolNames.includes("Read") && run.toolNames.includes("Write"));
  add(readWriteRuns.length > 0, "Claude emitted Read and Write tool events", readWriteRuns.length > 0 ? readWriteRuns.map((run) => `${relative(project, run.file)} tools=${run.toolNames.join("/")}`).join(", ") : "missing Read/Write");
}

function verifyFailureRetry(project, exists, add) {
  add(exists("working/01-jtbd-and-knowledge-map.md"), "framing artifact written after retry", join(project, "working/01-jtbd-and-knowledge-map.md"));

  const framingRuns = listFiles(join(project, ".liner-runs", "framing"), ".jsonl").map((file) => summarizeRunLog(file));
  add(framingRuns.length >= 2, "fresh and resumed framing logs exist", framingRuns.length > 0 ? framingRuns.map((run) => relative(project, run.file)).join(", ") : "missing");
  const parseErrors = framingRuns.flatMap((run) => run.errors.map((error) => `${relative(project, run.file)}:${error}`));
  add(parseErrors.length === 0, "framing retry logs parse as JSONL", parseErrors.length === 0 ? "all lines parsed" : parseErrors.slice(0, 3).join("; "));

  const failedFreshRuns = framingRuns.filter((run) => run.resume === false && typeof run.closeExitCode === "number" && run.closeExitCode !== 0);
  add(failedFreshRuns.length > 0, "fresh framing attempt failed", failedFreshRuns.length > 0 ? failedFreshRuns.map((run) => `${relative(project, run.file)} exit ${run.closeExitCode}`).join(", ") : "missing failed fresh run");

  const resumedRuns = framingRuns.filter((run) => run.resume === true && run.closeExitCode === 0);
  add(resumedRuns.length > 0, "retry used resume and succeeded", resumedRuns.length > 0 ? resumedRuns.map((run) => relative(project, run.file)).join(", ") : "missing successful resumed run");

  const failedThreads = new Set(failedFreshRuns.map((run) => run.threadId).filter(Boolean));
  const resumedThreads = new Set(resumedRuns.map((run) => run.threadId).filter(Boolean));
  const sharedThreads = Array.from(resumedThreads).filter((threadId) => failedThreads.has(threadId));
  add(sharedThreads.length > 0, "retry resumed the recorded framing thread", sharedThreads.length > 0 ? sharedThreads.join(", ") : `fresh: ${Array.from(failedThreads).join(", ") || "none"}; resume: ${Array.from(resumedThreads).join(", ") || "none"}`);

  const progress = readJson(join(project, ".liner-progress.json"));
  const progressValue = progress.ok && isRecord(progress.value) ? progress.value : {};
  add(progress.ok, ".liner-progress.json parses", progress.ok ? "JSON parsed" : progress.message);
  add(typeof progressValue.step === "number" && progressValue.step >= 2, "progress advanced after retry", `step: ${typeof progressValue.step === "number" ? progressValue.step : "missing"}`);

  const gates = readJson(join(project, ".liner-gates.json"));
  const gatesValue = gates.ok && isRecord(gates.value) ? gates.value : {};
  add(gates.ok, ".liner-gates.json parses", gates.ok ? "JSON parsed" : gates.message);
  add(gatesValue.gate0Accepted === true, "gate0 accepted after retry", `gate0Accepted: ${String(gatesValue.gate0Accepted)}`);
}

function verifyCandidatesComplete(project, add) {
  const longlistPath = join(project, "working", "02-candidate-longlist.md");
  add(existsSync(longlistPath), "candidate long-list artifact exists", longlistPath);
  const body = existsSync(longlistPath) ? readFileSync(longlistPath, "utf8") : "";
  const sectionCount = countMatches(body, /^## \d+\./gm);
  const candidateCount = countMatches(body, /^- \*\*/gm);
  const urlCount = countMatches(body, /https?:\/\//g);
  add(sectionCount > 0, "candidate long-list has sections", `sections: ${sectionCount}`);
  add(candidateCount > 0, "candidate long-list has candidates", `candidates: ${candidateCount}`);
  add(urlCount >= candidateCount, "candidate entries include URLs", `urls: ${urlCount}, candidates: ${candidateCount}`);
  add(!body.includes("TODO: Run Phase 2"), "candidate placeholder replaced", body.includes("TODO: Run Phase 2") ? "placeholder still present" : "placeholder absent");

  const candidateRuns = listFiles(join(project, ".liner-runs", "candidates"), ".jsonl").map((file) => summarizeRunLog(file));
  const cleanRuns = candidateRuns.filter((run) => run.closeExitCode === 0);
  add(cleanRuns.length > 0, "candidate run closed successfully", cleanRuns.length > 0 ? cleanRuns.map((run) => relative(project, run.file)).join(", ") : "missing exitCode 0 close");
}

function verifyEvaluationActive(project, exists, add) {
  const progress = readJson(join(project, ".liner-progress.json"));
  const progressValue = progress.ok && isRecord(progress.value) ? progress.value : {};
  add(progress.ok, ".liner-progress.json parses", progress.ok ? "JSON parsed" : progress.message);
  add(typeof progressValue.step === "number" && progressValue.step >= 4, "progress advanced to evaluation", `step: ${typeof progressValue.step === "number" ? progressValue.step : "missing"}`);

  const gates = readJson(join(project, ".liner-gates.json"));
  const gatesValue = gates.ok && isRecord(gates.value) ? gates.value : {};
  add(gates.ok, ".liner-gates.json parses", gates.ok ? "JSON parsed" : gates.message);
  add(gatesValue.gate1Accepted === true, "gate1 accepted", `gate1Accepted: ${String(gatesValue.gate1Accepted)}`);

  const evaluation = readYaml(join(project, "working", "03-evaluation.yaml"));
  add(evaluation.ok, "evaluation artifact parses", evaluation.ok ? "YAML parsed" : evaluation.message);

  const evaluationRuns = listFiles(join(project, ".liner-runs", "evaluation"), ".jsonl").map((file) => summarizeRunLog(file));
  add(evaluationRuns.length > 0, ".liner-runs/evaluation log exists", evaluationRuns.length > 0 ? evaluationRuns.map((run) => relative(project, run.file)).join(", ") : "missing");
  const evaluationThreadRuns = evaluationRuns.filter((run) => run.threadId);
  add(evaluationThreadRuns.length > 0, "evaluation thread recorded", evaluationThreadRuns.length > 0 ? evaluationThreadRuns.map((run) => run.threadId).join(", ") : "missing");
  const evaluationWorkEvents = evaluationRuns.reduce((count, run) => count + run.workEvents, 0);
  add(evaluationWorkEvents > 0, "evaluation emitted work events", `work events: ${evaluationWorkEvents}`);

  add(!exists("working/07-tape-draft.yaml"), "assembly draft not created yet", exists("working/07-tape-draft.yaml") ? "present" : "absent");
  add(!exists("MIXTAPE.md"), "MIXTAPE.md not compiled yet", exists("MIXTAPE.md") ? "present" : "absent");
}

function verifyEvaluationComplete(project, exists, add) {
  verifyEvaluationActive(project, exists, add);

  const longlistUrls = candidateUrlsFromLonglist(join(project, "working", "02-candidate-longlist.md"));
  add(longlistUrls.length > 0, "candidate URL count is available", `URL candidates: ${longlistUrls.length}`);

  const evaluation = readYaml(join(project, "working", "03-evaluation.yaml"));
  const evaluationValue = evaluation.ok && isRecord(evaluation.value) ? evaluation.value : {};
  const candidates = Array.isArray(evaluationValue.candidates) ? evaluationValue.candidates.filter(isRecord) : [];
  add(candidates.length === longlistUrls.length, "evaluation covers every longlist URL", `evaluation: ${candidates.length}, longlist: ${longlistUrls.length}`);

  const expectedUrls = new Set(longlistUrls);
  const seenUrls = new Set();
  const invalidRows = [];
  for (const [index, candidate] of candidates.entries()) {
    const url = normalizeUrl(typeof candidate.url === "string" ? candidate.url : "");
    if (!url) invalidRows.push(`candidates[${index}] missing url`);
    else if (!expectedUrls.has(url)) invalidRows.push(`candidates[${index}] unexpected url ${url}`);
    else if (seenUrls.has(url)) invalidRows.push(`candidates[${index}] duplicate url ${url}`);
    else seenUrls.add(url);

    const decision = typeof candidate.decision === "string" ? candidate.decision.trim() : "";
    if (!["kept", "trim", "dropped"].includes(decision)) {
      invalidRows.push(`candidates[${index}] invalid decision ${decision || "(empty)"}`);
    }
    if (decision === "kept" || decision === "trim") {
      const rating = typeof candidate.rating === "number" ? candidate.rating : Number(candidate.rating);
      if (!Number.isFinite(rating) || rating < 1 || rating > 5) {
        invalidRows.push(`candidates[${index}] invalid rating`);
      }
      if (typeof candidate.section !== "string" || candidate.section.trim() === "") {
        invalidRows.push(`candidates[${index}] missing section`);
      }
      if (typeof candidate.note !== "string" || candidate.note.trim() === "") {
        invalidRows.push(`candidates[${index}] missing note`);
      }
    }
  }
  const missingUrls = longlistUrls.filter((url) => !seenUrls.has(url));
  add(missingUrls.length === 0, "evaluation has no missing longlist URLs", missingUrls.length === 0 ? "none" : missingUrls.slice(0, 3).join(", "));
  add(invalidRows.length === 0, "evaluation decisions are structurally valid", invalidRows.length === 0 ? "all rows valid" : invalidRows.slice(0, 5).join("; "));

  const fragmentFiles = listFiles(join(project, "working", "evaluation-decisions"), ".yaml");
  add(fragmentFiles.length > 0, "evaluation decision fragments exist", fragmentFiles.length > 0 ? fragmentFiles.map((file) => relative(project, file)).join(", ") : "missing");

  const evaluationRuns = listFiles(join(project, ".liner-runs", "evaluation"), ".jsonl").map((file) => summarizeRunLog(file));
  const parseErrors = evaluationRuns.flatMap((run) => run.errors.map((error) => `${relative(project, run.file)}:${error}`));
  add(parseErrors.length === 0, "evaluation logs parse as JSONL", parseErrors.length === 0 ? "all lines parsed" : parseErrors.slice(0, 3).join("; "));
  const cleanRuns = evaluationRuns.filter((run) => run.closeExitCode === 0);
  add(cleanRuns.length > 0, "evaluation run closed successfully", cleanRuns.length > 0 ? cleanRuns.map((run) => relative(project, run.file)).join(", ") : "missing exitCode 0 close");
}

function summarizeRunLog(file) {
  const records = readJsonl(file);
  const meta = records.events.find((event) => isRecord(event) && event.type === "_liner_meta") || {};
  const thread = records.events.find((event) => isRecord(event) && event.type === "thread.started") || {};
  const close = records.events.find((event) => isRecord(event) && event.type === "_liner_close") || {};
  const workEvents = records.events.filter((event) => {
    if (!isRecord(event)) return false;
    if (event.type === "item.started" || event.type === "item.completed") return true;
    if (event.type === "agent_message" || event.type === "tool_call") return true;
    const item = isRecord(event.item) ? event.item : {};
    return item.type === "agent_message" || item.type === "command_execution" || item.type === "tool_call";
  }).length;

  return {
    agent: typeof meta.agent === "string" ? meta.agent : "",
    closeExitCode: typeof close.exitCode === "number" ? close.exitCode : null,
    errors: records.errors,
    file,
    resume: typeof meta.resume === "boolean" ? meta.resume : null,
    threadId: typeof thread.thread_id === "string" ? thread.thread_id : "",
    workEvents,
  };
}

function summarizeClaudeRunLog(file) {
  const records = readJsonl(file);
  const meta = records.events.find((event) => isRecord(event) && event.type === "_liner_meta") || {};
  const init = records.events.find((event) => isRecord(event) && event.type === "system" && event.subtype === "init") || {};
  const close = records.events.find((event) => isRecord(event) && event.type === "_liner_close") || {};
  const result = records.events.find((event) => isRecord(event) && event.type === "result") || {};
  const toolNames = new Set();
  for (const event of records.events) {
    if (!isRecord(event) || event.type !== "assistant") continue;
    const message = isRecord(event.message) ? event.message : {};
    const content = Array.isArray(message.content) ? message.content : [];
    for (const block of content) {
      if (!isRecord(block) || block.type !== "tool_use") continue;
      if (typeof block.name === "string" && block.name.trim() !== "") {
        toolNames.add(block.name.trim());
      }
    }
  }
  const mcpServers = Array.isArray(init.mcp_servers) ? init.mcp_servers.length : null;
  const initTools = Array.isArray(init.tools)
    ? init.tools.filter((tool) => typeof tool === "string").map((tool) => tool.trim()).filter(Boolean).sort()
    : [];

  return {
    agent: typeof meta.agent === "string" ? meta.agent : "",
    closeExitCode: typeof close.exitCode === "number" ? close.exitCode : null,
    errors: records.errors,
    file,
    mcpServers,
    model: typeof init.model === "string" ? init.model : "",
    resultOk: result.type === "result" ? result.is_error === false && result.subtype === "success" : null,
    sessionId: typeof init.session_id === "string" ? init.session_id : "",
    initTools,
    toolNames: Array.from(toolNames).sort(),
    version: typeof init.claude_code_version === "string" ? init.claude_code_version : "",
  };
}

function sameStringSet(actual, expected) {
  if (actual.length !== expected.length) return false;
  const actualSet = new Set(actual);
  return expected.every((item) => actualSet.has(item));
}

function readJsonl(path) {
  if (!existsSync(path)) return { events: [], errors: [`${path} is missing`] };
  const events = [];
  const errors = [];
  const lines = readFileSync(path, "utf8").split(/\r?\n/);
  lines.forEach((line, index) => {
    if (line.trim() === "") return;
    try {
      events.push(JSON.parse(line));
    } catch (error) {
      errors.push(`line ${index + 1}: ${error instanceof Error ? error.message : String(error)}`);
    }
  });
  return { events, errors };
}

function countMatches(text, pattern) {
  return Array.from(text.matchAll(pattern)).length;
}

function candidateUrlsFromLonglist(path) {
  if (!existsSync(path)) return [];
  const seen = new Set();
  const urls = [];
  for (const line of readFileSync(path, "utf8").split(/\r?\n/)) {
    const match = line.match(/https?:\/\/[^\s"'<>]+/);
    const url = normalizeUrl(match?.[0] || "");
    if (!url || seen.has(url)) continue;
    seen.add(url);
    urls.push(url);
  }
  return urls;
}

function normalizeUrl(url) {
  return url.trim().replace(/[)\],.;:]+$/g, "");
}

function readYaml(path) {
  if (!existsSync(path)) return { ok: false, message: `${path} is missing` };
  try {
    return { ok: true, value: YAML.parse(readFileSync(path, "utf8")) };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function readJson(path) {
  if (!existsSync(path)) return { ok: false, message: `${path} is missing` };
  try {
    return { ok: true, value: JSON.parse(readFileSync(path, "utf8")) };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function listFiles(dir, extension) {
  if (!existsSync(dir)) return [];
  const files = [];
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const stat = statSync(path);
    if (stat.isDirectory()) {
      files.push(...listFiles(path, extension));
    } else if (stat.isFile() && (extension === "" || path.endsWith(extension))) {
      files.push(path);
    }
  }
  return files.sort();
}

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
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

function renderProjectReport(metadata, project, expect, results) {
  const status = results.every((result) => result.ok) ? "PASS" : "FAIL";
  const lines = [
    "# Go TUI Project Verification",
    "",
    `Status: ${status}`,
    `Date: ${metadata.date}`,
    `Branch: ${metadata.branch}`,
    `Commit: ${metadata.commit}`,
    `Project: \`${project}\``,
    `Expectation: \`${expect}\``,
    `Evidence dir: \`${metadata.evidenceDir}\``,
    "",
    "This report audits project artifacts after a live/manual acceptance run. It does not prove keyboard navigation by itself.",
    "",
    "## Checks",
    "",
    ...results.map((result) => `- ${result.ok ? "[x]" : "[ ]"} ${result.name}: ${result.detail}`),
    "",
  ];
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
