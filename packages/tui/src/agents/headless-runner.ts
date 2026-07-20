import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { accessSync, constants, statSync } from "node:fs";
import { homedir } from "node:os";
import { basename, dirname, resolve } from "node:path";
import { detectAgents, resolveSkillPathWithDiagnostics } from "./detect.js";
import { runPhaseWithAgent } from "./runner.js";
import type { AgentDescriptor, AgentId } from "./types.js";
import type { AgentEvent } from "./events.js";
import type { PhaseId } from "../phases.js";
import { readConfig, resolveConfiguredAgent, type UserConfig } from "../config.js";
import { projectFolder, readTape } from "../yaml-io.js";

const AGENT_PHASES = new Set<PhaseId>([
  "framing",
  "candidates",
  "evaluation",
  "quality",
  "synthesis",
  "assembly",
  "improvement",
]);

type AgentChoice = AgentId | "auto";

export type HeadlessArgs = {
  project: string;
  phaseId: PhaseId;
  agent: AgentChoice;
  skillPath?: string;
  resume: boolean;
};

type ParseResult =
  | { ok: true; args: HeadlessArgs }
  | { ok: false; code: number; message: string };

type AgentSelection =
  | { ok: true; agent: AgentDescriptor }
  | { ok: false; code: number; message: string };

export type RunnerPreflight = { ok: true } | { ok: false; message: string };

type RunnerOutcome = "success" | "failed" | "cancelled";

type HeadlessEvent =
  | AgentEvent
  | {
      kind: "runner_start";
      phaseId: PhaseId;
      project: string;
      agent: AgentId;
      skillPath: string;
      resume: boolean;
    }
  | { kind: "runner_error"; message: string }
  | {
      kind: "runner_failure";
      failureKind: "preflight" | "runtime";
      message: string;
      recovery: string;
    }
  | { kind: "runner_diagnostic"; category: string; message: string }
  | { kind: "runner_cancelled"; message: string; recovery: string }
  | { kind: "runner_done"; outcome: RunnerOutcome; code: number | null; logPath?: string };

let stdoutBroken = false;

process.stdout.on("error", (error) => {
  if (isBrokenPipeError(error)) {
    stdoutBroken = true;
    return;
  }
  throw error;
});

export function parseHeadlessArgs(argv: string[]): ParseResult {
  const values = new Map<string, string>();
  let resume = false;

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i] ?? "";
    if (arg === "--help" || arg === "-h") {
      return { ok: false, code: 0, message: usage() };
    }
    if (arg === "--resume") {
      resume = true;
      continue;
    }
    const [name, inlineValue] = splitArg(arg);
    if (!name) {
      return { ok: false, code: 2, message: `Unknown argument: ${arg}` };
    }
    const value = inlineValue ?? argv[++i];
    if (value == null || value.startsWith("--")) {
      return { ok: false, code: 2, message: `Missing value for --${name}` };
    }
    values.set(name, value);
  }

  const project = values.get("project")?.trim();
  if (!project) {
    return { ok: false, code: 2, message: "Missing required --project" };
  }

  const rawPhase = values.get("phase")?.trim();
  if (!rawPhase || !AGENT_PHASES.has(rawPhase as PhaseId)) {
    return {
      ok: false,
      code: 2,
      message:
        `--phase must be one of: ${Array.from(AGENT_PHASES).join(", ")}`,
    };
  }

  const rawAgent = values.get("agent")?.trim() ?? "auto";
  if (rawAgent !== "auto" && rawAgent !== "claude" && rawAgent !== "codex") {
    return { ok: false, code: 2, message: "--agent must be auto, claude, or codex" };
  }

  if (rawPhase === "improvement" && resume) {
    return {
      ok: false,
      code: 2,
      message: "Improve Corpus retries must start a fresh isolated agent session; --resume is not supported",
    };
  }

  const skillPath = values.get("skill-path")?.trim();
  return {
    ok: true,
    args: {
      project,
      phaseId: rawPhase as PhaseId,
      agent: rawAgent,
      skillPath: skillPath || undefined,
      resume,
    },
  };
}

export function selectHeadlessAgent(
  choice: AgentChoice,
  installed: AgentDescriptor[],
  config: UserConfig = readConfig(),
): AgentSelection {
  if (choice !== "auto") {
    const envBin = process.env[choice === "claude" ? "LINER_CLAUDE_BIN" : "LINER_CODEX_BIN"]?.trim();
    if (!envBin && config.runner?.agent === choice) {
      const envHome =
        process.env[choice === "claude" ? "LINER_CLAUDE_HOME" : "LINER_CODEX_HOME"]?.trim() ||
        process.env[choice === "claude" ? "CLAUDE_CONFIG_DIR" : "CODEX_HOME"]?.trim();
      return {
        ok: true,
        agent: {
          id: choice,
          name: choice === "claude" ? "Claude" : "OpenAI",
          bin: config.runner.executable,
          configHome: envHome || config.runner.configHome,
        },
      };
    }
    const agent = installed.find((candidate) => candidate.id === choice);
    if (agent) return { ok: true, agent };
    return {
      ok: false,
      code: 2,
      message: `${choice} agent was requested, but it was not found on PATH.`,
    };
  }

  const envChoice = process.env["LINER_AGENT"]?.trim().toLowerCase();
  const configured = resolveConfiguredAgent(installed, config);
  if (configured) return { ok: true, agent: configured };
  if (envChoice === "claude" || envChoice === "codex") {
    return {
      ok: false,
      code: 2,
      message: `${envChoice} is selected by LINER_AGENT, but its executable is unavailable. Set ${envChoice === "claude" ? "LINER_CLAUDE_BIN" : "LINER_CODEX_BIN"} or update LINER_AGENT.`,
    };
  }
  if (installed.length === 0) {
    return {
      ok: false,
      code: 2,
      message: "No supported agent found on PATH. Install the Claude Code or Codex CLI.",
    };
  }
  return {
    ok: false,
    code: 2,
    message:
      `Multiple agents found (${installed.map((a) => a.id).join(", ")}). ` +
      `Pass --agent or set LINER_AGENT.`,
  };
}

export function preflightAgent(agent: AgentDescriptor): RunnerPreflight {
  const providerLabel = agent.id === "claude" ? "Claude" : "OpenAI";
  const cliLabel = agent.id === "claude" ? "Claude Code" : "Codex CLI";
  const homeEnv = agent.id === "claude" ? "LINER_CLAUDE_HOME" : "LINER_CODEX_HOME";
  const configHome = agent.configHome;
  if (!configHome || !directoryReadable(configHome)) {
    return {
      ok: false,
      message: `${providerLabel} config home is not accessible: ${configHome || "(not configured)"}. Update Settings or ${homeEnv}.`,
    };
  }

  const env = agentEnvironment(agent);
  const version = spawnSync(agent.bin, ["--version"], { encoding: "utf8", env, timeout: 10_000 });
  const identity = `${version.stdout || ""}\n${version.stderr || ""}`.toLowerCase();
  if (version.status !== 0 || !identity.includes(agent.id === "claude" ? "claude" : "codex")) {
    return {
      ok: false,
      message: `${cliLabel} executable identity check failed: ${agent.bin}. Update Settings or ${agent.id === "claude" ? "LINER_CLAUDE_BIN" : "LINER_CODEX_BIN"}.`,
    };
  }
  const capability = spawnSync(agent.bin, ["--help"], { encoding: "utf8", env, timeout: 10_000 });
  const help = `${capability.stdout || ""}\n${capability.stderr || ""}`.toLowerCase();
  const supportsHeadless = agent.id === "codex" ? /\bexec\b/.test(help) : /(?:^|\s)(?:-p|--print)(?:\s|,|$)/m.test(help);
  if (capability.status !== 0 || !supportsHeadless) {
    return {
      ok: false,
      message: `${cliLabel} is not a supported headless runner: ${agent.bin}. Upgrade the CLI and retry.`,
    };
  }
  const authArgs = agent.id === "claude" ? ["auth", "status"] : ["login", "status"];
  const auth = spawnSync(agent.bin, authArgs, { encoding: "utf8", env, timeout: 10_000 });
  if (auth.status !== 0) {
    return {
      ok: false,
      message: `${providerLabel} authentication is not ready. Run: ${agent.bin} ${agent.id === "claude" ? "auth login" : "login"}`,
    };
  }
  return { ok: true };
}

function directoryReadable(path: string): boolean {
  try {
    if (!statSync(path).isDirectory()) return false;
    accessSync(path, constants.R_OK);
    return true;
  } catch {
    return false;
  }
}

function agentEnvironment(agent: AgentDescriptor): NodeJS.ProcessEnv {
  if (!agent.configHome) return process.env;
  return {
    ...process.env,
    [agent.id === "claude" ? "CLAUDE_CONFIG_DIR" : "CODEX_HOME"]: agent.configHome,
  };
}

export async function main(argv: string[] = process.argv.slice(2)): Promise<number> {
  const parsed = parseHeadlessArgs(argv);
  if (!parsed.ok) {
    if (parsed.code === 0) {
      process.stdout.write(parsed.message + "\n");
    } else {
      emitRunnerFailure("preflight", parsed.message);
    }
    return parsed.code;
  }

  try {
    const args = parsed.args;
    const skillPath = resolveHeadlessSkillPath(args.skillPath);
    if (!skillPath) {
      emitRunnerFailure("preflight", "Could not find the curating-mixtapes skill bundle. Pass --skill-path.");
      return 2;
    }

    const selection = selectHeadlessAgent(args.agent, detectAgents());
    if (!selection.ok) {
      emitRunnerFailure("preflight", selection.message);
      return selection.code;
    }

    const preflight = preflightAgent(selection.agent);
    if (!preflight.ok) {
      emitRunnerFailure("preflight", preflight.message);
      return 2;
    }

    const folder = projectFolder(args.project);
    const { tape } = readTape(folder.tapePath);
    const handle = runPhaseWithAgent({
      agent: selection.agent,
      phaseId: args.phaseId,
      project: folder.path,
      skillPath,
      tape,
      resume: args.resume,
      onEvent: emit,
    });

    emit({
      kind: "runner_start",
      phaseId: args.phaseId,
      project: folder.path,
      agent: selection.agent.id,
      skillPath,
      resume: args.resume,
    });

    let cancelled = false;
    const cancel = (): void => {
      cancelled = true;
      handle.cancel();
    };
    process.once("SIGINT", cancel);
    process.once("SIGTERM", cancel);
    const result = await handle.done;
    process.off("SIGINT", cancel);
    process.off("SIGTERM", cancel);
    const failure = classifyRunnerFailure(result.stderr, result.code !== 0);
    for (const diagnostic of failure.diagnostics) {
      emit({
        kind: "runner_diagnostic",
        category: runnerDiagnosticCategory(diagnostic),
        message: diagnostic,
      });
    }
    if (cancelled) {
      emit({
        kind: "runner_cancelled",
        message: "AI run cancelled.",
        recovery: "Retry this phase when ready, or return to the project.",
      });
    } else if (result.code !== 0 && failure.message) {
      emit({
        kind: "runner_failure",
        failureKind: "runtime",
        message: failure.message,
        recovery: failure.recovery,
      });
    }
    emit({
      kind: "runner_done",
      outcome: cancelled ? "cancelled" : result.code === 0 ? "success" : "failed",
      code: result.code,
      logPath: result.logPath,
    });
    return result.code ?? 1;
  } catch (error) {
    emitRunnerFailure("runtime", error instanceof Error ? error.message : String(error));
    return 1;
  }
}

function emitRunnerFailure(failureKind: "preflight" | "runtime", message: string): void {
  emit({
    kind: "runner_failure",
    failureKind,
    message,
    recovery: runnerRecovery(message),
  });
}

export function classifyRunnerFailure(stderr: string, failed = true): {
  message: string;
  recovery: string;
  diagnostics: string[];
} {
  const lines = stderr
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  if (!failed) {
    return {
      message: "",
      recovery: "",
      diagnostics: lines.filter(isRunnerDiagnostic),
    };
  }
  const candidates = lines.filter((line) =>
    !isNonfatalRunnerDiagnostic(line) && runnerFailureScore(line) > 0);
  const primary = candidates.reduce((best, line) =>
    runnerFailureScore(line) > runnerFailureScore(best) ? line : best, "");
  const diagnostics = lines.filter((line) => isRunnerDiagnostic(line) && line !== primary);
  const message = primary.replace(/^(?:fatal|error)(?:\s*:\s*|\s+)/i, "").trim();
  return {
    message,
    recovery: message ? runnerRecovery(message) : "",
    diagnostics,
  };
}

function isRunnerDiagnostic(line: string): boolean {
  const value = line.toLowerCase();
  return (
    /(?:^|\s)warn(?:ing)?(?:\s|:)/i.test(line) ||
    value.includes("failed to load skill") ||
    value.includes("codex_core_skills::loader") ||
    value.includes("rmcp::") ||
    value.includes("mcp connector") ||
    value.includes("optional connector") ||
    value.includes("integration:")
  );
}

function isNonfatalRunnerDiagnostic(line: string): boolean {
  const value = line.toLowerCase();
  return /(?:^|\s)warn(?:ing)?(?:\s|:)/i.test(line) || value.includes("optional connector");
}

function runnerFailureScore(line: string): number {
  const value = line.toLowerCase();
  if (!value) return -1;
  if (/\b(?:auth(?:entication)?|login|token expired|logged in)\b/.test(value)) return 100;
  if (/\b(?:version|unsupported|upgrade)\b/.test(value)) return 90;
  if (value.includes("required") && isRunnerDiagnostic(line)) return 85;
  if (/\b(?:config home|executable|not found|missing|denied|invalid|timed? out)\b/.test(value)) return 80;
  if (/\bfatal\b/.test(value)) return 70;
  if (/\b(?:error|failed|failure)\b/.test(value)) return 40;
  return 0;
}

function runnerDiagnosticCategory(line: string): string {
  const value = line.toLowerCase();
  if (value.includes("skill")) return "skill";
  if (value.includes("mcp") || value.includes("connector")) return "mcp";
  if (value.includes("integration")) return "integration";
  return "warning";
}

function runnerRecovery(message: string): string {
  const value = message.toLowerCase();
  if (value.includes("version") || value.includes("unsupported") || value.includes("upgrade")) {
    return "Upgrade the configured AI runner, then retry this phase.";
  }
  if (value.includes("auth") || value.includes("login") || value.includes("logged in")) {
    return "Authenticate the configured AI runner, then retry this phase.";
  }
  if (value.includes("config home") || value.includes("settings") || value.includes("executable")) {
    return "Update the AI runner profile in Settings, then retry this phase.";
  }
  if (value.includes("required") && (value.includes("mcp") || value.includes("connector") || value.includes("integration"))) {
    return "Repair the required AI runner integration, then retry this phase.";
  }
  return "Retry this phase. If it fails again, inspect the full runner log.";
}

export function resolveHeadlessSkillPath(explicit?: string): string | null {
  if (!explicit) return resolveSkillPathWithDiagnostics().path;

  const expanded = explicit === "~"
    ? homedir()
    : explicit.startsWith("~/")
      ? resolve(homedir(), explicit.slice(2))
      : explicit;
  const directory = basename(expanded) === "SKILL.md" ? dirname(expanded) : expanded;
  return resolve(directory);
}

function splitArg(arg: string): [string, string | undefined] | [null, undefined] {
  if (!arg.startsWith("--")) return [null, undefined];
  const body = arg.slice(2);
  const eq = body.indexOf("=");
  if (eq === -1) return [body, undefined];
  return [body.slice(0, eq), body.slice(eq + 1)];
}

function emit(event: HeadlessEvent): void {
  if (stdoutBroken) return;
  if (!writeHeadlessEvent(event)) stdoutBroken = true;
}

export function writeHeadlessEvent(
  event: HeadlessEvent,
  write: (line: string) => unknown = (line) => process.stdout.write(line),
): boolean {
  try {
    write(JSON.stringify(event) + "\n");
    return true;
  } catch (error) {
    if (isBrokenPipeError(error)) return false;
    throw error;
  }
}

export function isBrokenPipeError(error: unknown): boolean {
  return Boolean(
    error &&
      typeof error === "object" &&
      "code" in error &&
      (error as { code?: unknown }).code === "EPIPE",
  );
}

function usage(): string {
  return [
    "Usage: node dist/agents/headless-runner.js --project <path> --phase <phase> [options]",
    "",
    "Options:",
    "  --agent auto|claude|codex  Agent to run. Defaults to auto.",
    "  --skill-path <path>        Directory containing SKILL.md.",
    "  --resume                   Continue the last agent session for this project.",
    "  -h, --help                 Show this help.",
  ].join("\n");
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  void main().then((code) => {
    process.exitCode = code;
  });
}
