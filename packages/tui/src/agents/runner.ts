import { spawn } from "node:child_process";
import { buildPhasePrompt } from "./prompts.js";
import { LineBuffer, parseLine } from "./parsers.js";
import { closeRunLog, openRunLog } from "./run-log.js";
import { modelForPhase } from "./models.js";
import { readConfig } from "../config.js";
import type { AgentEvent } from "./events.js";
import type { AgentDescriptor, AgentId, AgentRunHandle } from "./types.js";
import type { PhaseId } from "../phases.js";
import type { Tape } from "../types.js";

// How long to wait for the agent's first byte of output before declaring it
// unreachable. Generous enough to cover a cold CLI start + auth refresh, short
// enough that a dead subscription doesn't hang the TUI indefinitely.
const WATCHDOG_MS = 15_000;

export type AgentRunArgs = {
  agent: AgentDescriptor;
  phaseId: PhaseId;
  project: string;
  skillPath: string;
  tape: Tape;
  /**
   * Continue the previous session for this phase if one exists. The agent
   * resumes from its last message in the same conversation, so prior
   * tool-call outputs stay in context — no re-fetching, no wasted tokens.
   * When false (default), start a fresh session with the full SKILL.md
   * prompt.
   */
  resume?: boolean;
  /** Called for each normalized event parsed from the agent's stdout. */
  onEvent: (event: AgentEvent) => void;
};

export type AgentTaskArgs = {
  agent: AgentDescriptor;
  project: string;
  skillPath: string;
  /** The full prompt string sent to the agent on stdin. */
  prompt: string;
  /**
   * Model id/alias to pass to the agent CLI (`--model`). Undefined → the
   * agent's own default model. Applied on fresh runs only; a resumed session
   * keeps the model it started with.
   */
  model?: string;
  /**
   * Short identifier used for the audit-log bucket directory under
   * `.liner-runs/<taskLabel>/`. Methodology phases pass their `PhaseId`; one-off
   * side tasks (JTBD elicitation, etc.) pass a descriptive slug like
   * `"jtbd-clarify"`.
   */
  taskLabel: string;
  /**
   * Same semantics as the phase variant — continue the previous session
   * (cheap, no re-fetch) instead of re-sending the full prompt.
   */
  resume?: boolean;
  /** Called for each normalized event parsed from the agent's stdout. */
  onEvent: (event: AgentEvent) => void;
};

/**
 * Spawn the configured agent CLI with a phase-specific prompt. The agent's
 * newline-delimited JSON stdout is parsed into AgentEvents so the PhaseRunner
 * can render tool calls / streaming text / summary as structured panes.
 *
 * Thin wrapper over `runAgentTask` — builds the per-phase prompt via
 * `buildPhasePrompt` and labels the audit log with the phase id.
 */
export function runPhaseWithAgent(args: AgentRunArgs): AgentRunHandle {
  const prompt = buildPhasePrompt({
    phaseId: args.phaseId,
    project: args.project,
    skillPath: args.skillPath,
    tape: args.tape,
  });
  // Per-phase model: the built-in map downgrades the heavy phases, with the
  // user's config.yaml overrides winning. The compile phase isn't an agent
  // run; gates don't reach here either.
  const overrides = readConfig().models?.[args.agent.id];
  const model = modelForPhase(args.agent.id, args.phaseId, overrides);
  return runAgentTask({
    agent: args.agent,
    project: args.project,
    skillPath: args.skillPath,
    prompt,
    model,
    taskLabel: args.phaseId,
    resume: args.resume,
    onEvent: args.onEvent,
  });
}

/**
 * Run a single agent task with a caller-supplied prompt. Used by methodology
 * phases (via `runPhaseWithAgent`) and by one-off side tasks like the JTBD
 * elicitation step in the new-mixtape wizard. The agent's stdout is parsed
 * the same way (stream JSON → AgentEvent) so callers get tool calls + final
 * text via the same callback interface.
 */
export function runAgentTask(args: AgentTaskArgs): AgentRunHandle {
  // Delegating cancel: a model-rejection fallback re-spawns under the hood, so
  // the handle's cancel must target whichever child is currently live.
  let cancelCurrent: () => void = () => {};
  const done = runAttempt(args, false, (c) => {
    cancelCurrent = c;
  });
  return { cancel: () => cancelCurrent(), done };
}

/**
 * One spawn attempt. On a model-rejection failure — and only on a fresh,
 * model-pinned run — it warns the curator with the exact fix and retries once
 * on the agent's default model, so a renamed/retired/mistyped model id
 * degrades to "ran on the default" instead of a hard failure. `registerCancel`
 * is re-invoked per attempt so the caller's cancel always hits the live child.
 */
function runAttempt(
  args: AgentTaskArgs,
  isModelFallback: boolean,
  registerCancel: (cancel: () => void) => void,
): Promise<{ code: number | null; stderr: string }> {
  const command = args.agent.bin;
  const resume = args.resume === true;
  const cliArgs = buildArgs(args.agent.id, args.project, args.skillPath, resume, args.model);

  const child = spawn(command, cliArgs, {
    stdio: ["pipe", "pipe", "pipe"],
    cwd: args.project,
    env: {
      ...process.env,
      // Disable extended/interleaved thinking for the spawned Claude Code.
      // A long multi-turn tool loop with thinking on hits a hard API 400 once
      // Claude Code auto-compacts the conversation: a prior `thinking` block
      // gets modified on replay and the API rejects the next turn
      // ("`thinking` or `redacted_thinking` blocks ... cannot be modified").
      // It surfaces both on fresh runs (~25+ turns) and instantly on resume
      // (the resume replays the already-corrupted message). Turning thinking
      // off removes the blocks entirely, so the failure class can't occur.
      // Only Claude reads this; Codex ignores it.
      ...(args.agent.id === "claude" ? { MAX_THINKING_TOKENS: "0" } : {}),
    },
  });

  let cancelled = false;
  registerCancel(() => {
    cancelled = true;
    if (!child.killed) child.kill("SIGTERM");
  });

  // Accumulated tail of the agent's own output (stdout + stderr). Used only to
  // recognize a model-rejection error for the fallback; capped so a long run
  // doesn't retain the whole transcript.
  let outputTail = "";
  const noteForDetection = (chunk: string): void => {
    outputTail = (outputTail + chunk).slice(-4000);
  };

  // On resume, the previous session already contains the prompt + all prior
  // tool-call results in context. Send a tiny continuation nudge instead of
  // re-shipping the full prompt — that's the whole point of resume (cheaper,
  // no re-fetching).
  const stdinPayload = resume
    ? "Continue from where you left off. Pick up the same task; do not re-do tool calls you already completed."
    : args.prompt;
  child.stdin?.write(stdinPayload, "utf8");
  child.stdin?.end();

  // Tee-style audit log of the raw JSONL stream. Not load-bearing (Claude's
  // session.json already covers resume) — this gives us a per-run transcript
  // the curator can scroll back through and a forensic record when things
  // misbehave.
  const log = openRunLog(args.project, args.taskLabel, {
    agent: args.agent.id,
    resume,
  });

  // Agent-unavailability watchdog. A dead subscription / revoked auth / model
  // deprecation can leave the agent process launched but mute — no stream
  // events, no stderr, no exit — and the TUI would otherwise spin forever. If
  // nothing comes back within the window, surface a clear error and kill the
  // process so the run lands in the failed state (where retry / switch-agent
  // live). Any output at all — stdout or stderr — clears the watchdog.
  let sawOutput = false;
  const watchdog = setTimeout(() => {
    if (sawOutput || child.killed) return;
    const name = args.agent.id === "claude" ? "Claude Code" : "Codex";
    args.onEvent({
      kind: "raw",
      text:
        `[liner] No response from ${name} after ${WATCHDOG_MS / 1000}s. ` +
        `It may not be signed in, or its subscription/quota may be exhausted. ` +
        `Try running \`${args.agent.id}\` on its own to check, then retry or switch agents.`,
    });
    log.write(JSON.stringify({ type: "_liner_watchdog_timeout", ms: WATCHDOG_MS }));
    child.kill("SIGTERM");
  }, WATCHDOG_MS);
  watchdog.unref?.();
  const noteOutput = (): void => {
    if (sawOutput) return;
    sawOutput = true;
    clearTimeout(watchdog);
  };

  const buffer = new LineBuffer();
  if (child.stdout) {
    child.stdout.setEncoding("utf8");
    child.stdout.on("data", (chunk: string) => {
      noteOutput();
      noteForDetection(chunk);
      const lines = buffer.push(chunk);
      for (const line of lines) {
        log.write(line);
        for (const event of parseLine(args.agent.id, line)) {
          args.onEvent(event);
        }
      }
    });
  }

  let stderr = "";
  if (child.stderr) {
    child.stderr.setEncoding("utf8");
    child.stderr.on("data", (chunk: string) => {
      noteOutput();
      noteForDetection(chunk);
      stderr += chunk;
      // Surface stderr as raw events so users see install/auth hints inline.
      for (const line of chunk.split("\n")) {
        const trimmed = line.trim();
        if (trimmed) {
          args.onEvent({ kind: "raw", text: trimmed });
          // Record stderr in the log too — wrapped so it doesn't masquerade
          // as a real JSON event when the log is replayed.
          log.write(JSON.stringify({ type: "_liner_stderr", text: trimmed }));
        }
      }
    });
  }

  return new Promise<{ code: number | null; stderr: string }>((resolve) => {
    child.on("close", (code) => {
      clearTimeout(watchdog);
      // Flush any trailing partial line.
      for (const line of buffer.flush()) {
        log.write(line);
        for (const event of parseLine(args.agent.id, line)) {
          args.onEvent(event);
        }
      }
      closeRunLog(log, code, stderr.length);

      // Model-rejection fallback. If a fresh, model-pinned run failed because
      // the agent didn't recognize the model (renamed / retired / mistyped in
      // config), warn the curator with the exact fix, then retry once on the
      // agent's default model. Resumes, cancellations, and already-fallen-back
      // runs are exempt — and we only claim "model rejected" when the output
      // plausibly says so, so a real auth/quota failure surfaces normally.
      if (
        code !== 0 &&
        !cancelled &&
        !resume &&
        !isModelFallback &&
        args.model &&
        looksLikeModelRejection(outputTail, args.model)
      ) {
        args.onEvent({
          kind: "raw",
          text:
            `[liner] ${args.agent.name} rejected the model "${args.model}" set for ` +
            `this phase — retrying on the default model. To fix: set ` +
            `models.${args.agent.id}.${args.taskLabel} to a valid model id in ` +
            `~/.liner/config.yaml, or remove it to keep the default.`,
        });
        log.write(
          JSON.stringify({ type: "_liner_model_fallback", model: args.model }),
        );
        resolve(runAttempt({ ...args, model: undefined }, true, registerCancel));
        return;
      }

      resolve({ code, stderr });
    });
    child.on("error", (e) => {
      clearTimeout(watchdog);
      args.onEvent({ kind: "raw", text: `[runner error] ${e.message}` });
      log.write(JSON.stringify({ type: "_liner_error", message: e.message }));
      closeRunLog(log, 1, stderr.length);
      resolve({ code: 1, stderr: e.message });
    });
  });
}

function buildArgs(
  id: AgentId,
  project: string,
  skillPath: string,
  resume: boolean,
  model?: string,
): string[] {
  if (id === "claude") {
    // Stream-json requires --verbose. --include-partial-messages would give
    // us mid-message chunking but the cost is ~10× more events; we let the
    // assistant message arrive as one block per turn instead.
    //
    // Tool whitelist keeps the agent's reach scoped even with permissions
    // skipped — Read/Write/Edit/Glob/Grep/WebFetch is exactly what the
    // methodology needs and nothing more (no Bash, no Task).
    //
    // `--continue` (resume mode) picks up the most recent session in the
    // current working directory. Since we spawn with cwd=project, "most
    // recent session" is the one we started for this phase. The previous
    // conversation — prompt, tool calls, partial work — stays in context.
    //
    // `--strict-mcp-config` with no `--mcp-config` loads zero MCP servers, so
    // the methodology agent doesn't inherit the user's global connectors
    // (Gmail, Drive, etc.). Those are irrelevant to curation, bloat the
    // context window, and widen the agent's reach for no benefit.
    const base = [
      "-p",
      "--output-format",
      "stream-json",
      "--verbose",
      "--add-dir",
      skillPath,
      "--strict-mcp-config",
      "--dangerously-skip-permissions",
      "--allowedTools",
      "Read Write Edit Glob Grep WebFetch",
    ];
    // Apply the per-phase model on fresh runs only — a resumed session keeps
    // the model it was started with, and `--model` alongside `--continue`
    // wouldn't change the already-established conversation's model.
    if (model && !resume) base.push("--model", model);
    if (resume) base.push("--continue");
    return base;
  }
  if (id === "codex") {
    if (resume) {
      // Codex's `exec resume` subcommand resumes the most recent thread.
      return [
        "exec",
        "resume",
        "--last",
        "--cd",
        project,
        "--skip-git-repo-check",
        "-s",
        "workspace-write",
        "--json",
        "-",
      ];
    }
    return [
      "exec",
      "--cd",
      project,
      ...(model ? ["--model", model] : []),
      "--add-dir",
      skillPath,
      "--skip-git-repo-check",
      "-s",
      "workspace-write",
      "--json",
      "-",
    ];
  }
  return [];
}

/**
 * Heuristic: did this failure come from the agent not recognizing the model?
 * Matches the common "unknown/invalid model" phrasings both CLIs emit, plus
 * the offending model id echoed next to an error token. Deliberately
 * conservative — a false negative just means no auto-fallback (the normal
 * error still surfaces); we only want to claim "model rejected" when it
 * plausibly was, so genuine auth/quota failures aren't mislabeled.
 */
export function looksLikeModelRejection(output: string, model: string): boolean {
  const t = output.toLowerCase();
  if (!t) return false;
  const phrases = [
    "unknown model",
    "invalid model",
    "model not found",
    "model_not_found",
    "no such model",
    "unrecognized model",
    "unsupported model",
    "not a valid model",
    "model does not exist",
    "is not a supported model",
  ];
  if (phrases.some((p) => t.includes(p))) return true;
  // The model id echoed back alongside a generic error/invalid token.
  const m = model.toLowerCase();
  return (
    m.length > 0 &&
    t.includes(m) &&
    (t.includes("error") || t.includes("invalid") || t.includes("unknown") || t.includes("not found"))
  );
}

