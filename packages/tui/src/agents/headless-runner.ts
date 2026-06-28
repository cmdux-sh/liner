import { fileURLToPath } from "node:url";
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
  | { kind: "runner_done"; code: number | null; stderr?: string };

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
    const agent = installed.find((candidate) => candidate.id === choice);
    if (agent) return { ok: true, agent };
    return {
      ok: false,
      code: 2,
      message: `${choice} agent was requested, but it was not found on PATH.`,
    };
  }

  const configured = resolveConfiguredAgent(installed, config);
  if (configured) return { ok: true, agent: configured };
  if (installed.length === 0) {
    return {
      ok: false,
      code: 2,
      message: "No supported agent found on PATH. Install Claude Code or Codex.",
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

export async function main(argv: string[] = process.argv.slice(2)): Promise<number> {
  const parsed = parseHeadlessArgs(argv);
  if (!parsed.ok) {
    if (parsed.code === 0) {
      process.stdout.write(parsed.message + "\n");
    } else {
      emit({ kind: "runner_error", message: parsed.message });
    }
    return parsed.code;
  }

  try {
    const args = parsed.args;
    const skillPath = resolveHeadlessSkillPath(args.skillPath);
    if (!skillPath) {
      emit({
        kind: "runner_error",
        message: "Could not find the curating-mixtapes skill bundle. Pass --skill-path.",
      });
      return 2;
    }

    const selection = selectHeadlessAgent(args.agent, detectAgents());
    if (!selection.ok) {
      emit({ kind: "runner_error", message: selection.message });
      return selection.code;
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

    const cancel = (): void => handle.cancel();
    process.once("SIGINT", cancel);
    process.once("SIGTERM", cancel);
    const result = await handle.done;
    process.off("SIGINT", cancel);
    process.off("SIGTERM", cancel);
    emit({
      kind: "runner_done",
      code: result.code,
      stderr: result.stderr || undefined,
    });
    return result.code ?? 1;
  } catch (error) {
    emit({
      kind: "runner_error",
      message: error instanceof Error ? error.message : String(error),
    });
    return 1;
  }
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
