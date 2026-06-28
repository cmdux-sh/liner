import { spawn } from "node:child_process";
import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { validatePhaseArtifact } from "./artifact-validation.js";
import { groupCandidateLonglist, readCandidateLonglist } from "./candidate-longlist.js";
import { ensureEvaluationArtifact, validateEvaluationFragment } from "./evaluation-assembly.js";
import { buildEvaluationSectionPrompt, buildPhasePrompt, WORKSPACE_DISCIPLINE } from "./prompts.js";
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
const DEFAULT_EVALUATION_CLOSURE_IDLE_MS = 120_000;

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
  /**
   * Optional continuation prompt used for special resume flows. Normal phase
   * resumes use `resumePromptForTask`; Evaluation artifact closure uses a
   * stricter write-only prompt after stopping an idle turn.
   */
  resumePrompt?: string;
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
  if (args.phaseId === "evaluation" && (!args.resume || hasSectionedEvaluationState(args.project))) {
    return runEvaluationSectionsWithAgent(args);
  }

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
  const handle = runAgentTask({
    agent: args.agent,
    project: args.project,
    skillPath: args.skillPath,
    prompt,
    model,
    taskLabel: args.phaseId,
    resume: args.resume,
    onEvent: args.onEvent,
  });
  return {
    cancel: handle.cancel,
    done: handle.done.then((result) => {
      if (result.code !== 0) return result;
      const validation = ensurePhaseArtifact(args.project, args.phaseId);
      if (validation.ok) {
        if ("message" in validation && validation.message) {
          args.onEvent({ kind: "raw", text: `[liner] ${validation.message}` });
        }
        return result;
      }

      const message = `[liner] Artifact validation failed after ${args.phaseId}: ${validation.message}`;
      args.onEvent({ kind: "text", text: message });
      return {
        code: 1,
        stderr: result.stderr ? `${result.stderr}\n${message}` : message,
      };
    }),
  };
}

function runEvaluationSectionsWithAgent(args: AgentRunArgs): AgentRunHandle {
  if (args.resume) {
    const existingEvaluation = ensureEvaluationArtifact(args.project);
    if (existingEvaluation.ok) {
      const message = existingEvaluation.message || "Evaluation artifact already complete.";
      args.onEvent({ kind: "raw", text: `[liner] ${message}` });
      return { cancel: () => {}, done: Promise.resolve({ code: 0, stderr: "" }) };
    }
  }

  const groups = groupCandidateLonglist(
    readCandidateLonglist(join(args.project, "working/02-candidate-longlist.md")),
  );
  if (groups.length === 0) {
    const message = "[liner] Cannot run Evaluation: working/02-candidate-longlist.md has no URL candidates.";
    args.onEvent({ kind: "raw", text: message });
    return { cancel: () => {}, done: Promise.resolve({ code: 1, stderr: message }) };
  }

  const overrides = readConfig().models?.[args.agent.id];
  const model = modelForPhase(args.agent.id, "evaluation", overrides);
  let cancelCurrent: () => void = () => {};
  let cancelled = false;

  const done = (async (): Promise<{ code: number | null; stderr: string }> => {
    markSectionedEvaluationStarted(args.project, groups.length);
    args.onEvent({
      kind: "raw",
      text: `[liner] Running Evaluation in ${groups.length} section chunk${groups.length === 1 ? "" : "s"}.`,
    });

    for (const group of groups) {
      if (cancelled) return { code: 1, stderr: "[liner] Evaluation cancelled." };
      const fragment = validateEvaluationFragment(join(args.project, group.fragmentPath), group.candidates);
      if (fragment.ok) {
        args.onEvent({
          kind: "raw",
          text:
            `[liner] Evaluation chunk ${group.index}/${group.total} already complete: ` +
            `${group.section} (${fragment.count} candidates).`,
        });
        continue;
      }
      args.onEvent({
        kind: "raw",
        text:
          `[liner] Evaluation chunk ${group.index}/${group.total}: ` +
          `${group.section} (${group.candidates.length} candidates).`,
      });
      const handle = runAgentTask({
        agent: args.agent,
        project: args.project,
        skillPath: args.skillPath,
        prompt: buildEvaluationSectionPrompt({
          project: args.project,
          skillPath: args.skillPath,
          tape: args.tape,
          group,
        }),
        model,
        taskLabel: "evaluation",
        onEvent: args.onEvent,
      });
      cancelCurrent = handle.cancel;
      const result = await handle.done;
      if (result.code !== 0) return result;
    }

    const validation = ensureEvaluationArtifact(args.project);
    if (!validation.ok) {
      const message = `[liner] Artifact validation failed after evaluation: ${validation.message}`;
      args.onEvent({ kind: "text", text: message });
      return { code: 1, stderr: message };
    }
    if (validation.message) {
      args.onEvent({ kind: "raw", text: `[liner] ${validation.message}` });
    }
    return { code: 0, stderr: "" };
  })();

  return {
    cancel: () => {
      cancelled = true;
      cancelCurrent();
    },
    done,
  };
}

function hasSectionedEvaluationState(project: string): boolean {
  if (existsSync(join(project, "working/evaluation-decisions/.liner-sectioned-evaluation.json"))) {
    return true;
  }
  const dir = join(project, "working/evaluation-decisions");
  if (!existsSync(dir)) return false;
  return readdirSync(dir).some((file) => /\.ya?ml$/i.test(file));
}

function markSectionedEvaluationStarted(project: string, chunkCount: number): void {
  const dir = join(project, "working/evaluation-decisions");
  mkdirSync(dir, { recursive: true });
  writeFileSync(
    join(dir, ".liner-sectioned-evaluation.json"),
    JSON.stringify(
      {
        version: 1,
        chunkCount,
        updatedAt: new Date().toISOString(),
      },
      null,
      2,
    ) + "\n",
    "utf8",
  );
}

function ensurePhaseArtifact(
  project: string,
  phaseId: PhaseId,
): { ok: true; message?: string } | { ok: false; message: string } {
  if (phaseId !== "evaluation") return validatePhaseArtifact(project, phaseId);
  return ensureEvaluationArtifact(project);
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
  const done = runAttempt(args, {}, (c) => {
    cancelCurrent = c;
  });
  return { cancel: () => cancelCurrent(), done };
}

type AttemptOptions = {
  modelFallback?: boolean;
  artifactClosure?: boolean;
};

const CLAUDE_METHOD_TOOLS = "Read Write Edit Glob Grep WebFetch";
const CLAUDE_METHOD_BUILTIN_TOOLS = "Read,Write,Edit,Glob,Grep,WebFetch";
const CLAUDE_HARD_DENIED_TOOLS = "Bash Task WebSearch ToolSearch";

/**
 * One spawn attempt. On a model-rejection failure — and only on a fresh,
 * model-pinned run — it warns the curator with the exact fix and retries once
 * on the agent's default model, so a renamed/retired/mistyped model id
 * degrades to "ran on the default" instead of a hard failure. `registerCancel`
 * is re-invoked per attempt so the caller's cancel always hits the live child.
 */
function runAttempt(
  args: AgentTaskArgs,
  attempt: AttemptOptions,
  registerCancel: (cancel: () => void) => void,
): Promise<{ code: number | null; stderr: string }> {
  const command = args.agent.bin;
  const resume = args.resume === true;
  const resumeSessionId = args.agent.id === "codex" && resume
    ? findLatestCodexSessionId(args.project, args.taskLabel)
    : undefined;
  if (args.agent.id === "codex" && resume && !resumeSessionId) {
    const message =
      `[liner] Cannot resume Codex ${args.taskLabel}: no prior Codex session id was found ` +
      `in .liner-runs/${args.taskLabel}. Retry this phase from a fresh run instead.`;
    args.onEvent({ kind: "raw", text: message });
    const log = openRunLog(args.project, args.taskLabel, {
      agent: args.agent.id,
      resume,
    });
    log.write(JSON.stringify({ type: "_liner_error", message }));
    closeRunLog(log, 1, message.length);
    return Promise.resolve({ code: 1, stderr: message });
  }
  const cliArgs = buildArgs(args.agent.id, args.project, args.skillPath, resume, args.model, resumeSessionId);

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
    ? args.resumePrompt ?? resumePromptForTask(args.taskLabel)
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

  const closureIdleMs = evaluationClosureIdleMs();
  const canCloseEvaluationArtifact =
    args.taskLabel === "evaluation" &&
    !attempt.artifactClosure &&
    closureIdleMs > 0;
  let evaluationClosureTimer: ReturnType<typeof setTimeout> | undefined;
  let evaluationClosureTimedOut = false;
  let childClosed = false;
  let forcedEvaluationResult: { code: number; message: string } | undefined;
  const activeTools = new Set<string>();
  const clearEvaluationClosureTimer = (): void => {
    if (!evaluationClosureTimer) return;
    clearTimeout(evaluationClosureTimer);
    evaluationClosureTimer = undefined;
  };
  const scheduleEvaluationClosureCheck = (): void => {
    if (!canCloseEvaluationArtifact || cancelled || child.killed || childClosed) return;
    clearEvaluationClosureTimer();
    evaluationClosureTimer = setTimeout(() => {
      if (cancelled || child.killed) return;
      if (activeTools.size > 0) {
        scheduleEvaluationClosureCheck();
        return;
      }
      const validation = ensureEvaluationArtifact(args.project);
      if (validation.ok) {
        forcedEvaluationResult = {
          code: 0,
          message: "[liner] Evaluation artifact is valid; stopping the quiet agent turn.",
        };
        args.onEvent({ kind: "raw", text: forcedEvaluationResult.message });
        log.write(
          JSON.stringify({
            type: "_liner_artifact_closure_complete",
            phase: "evaluation",
            ms: closureIdleMs,
          }),
        );
        child.kill("SIGTERM");
        return;
      }
      evaluationClosureTimedOut = true;
      const idleLabel = closureIdleMs >= 1000 ? `${Math.round(closureIdleMs / 1000)}s` : `${closureIdleMs}ms`;
      const message =
        `[liner] Evaluation has been quiet for ${idleLabel} ` +
        `with no valid working/03-evaluation.yaml. Stopping this turn and resuming once ` +
        `with a write-only artifact closure prompt.`;
      args.onEvent({ kind: "raw", text: message });
      log.write(
        JSON.stringify({
          type: "_liner_artifact_closure_timeout",
          phase: "evaluation",
          ms: closureIdleMs,
          validation: validation.message,
        }),
      );
      child.kill("SIGTERM");
    }, closureIdleMs);
    evaluationClosureTimer.unref?.();
  };
  const noteAgentEvent = (event: AgentEvent): void => {
    if (!canCloseEvaluationArtifact) return;
    if (event.kind === "tool_start" && event.id) {
      activeTools.add(event.id);
      scheduleEvaluationClosureCheck();
      return;
    }
    if (event.kind === "tool_done" && event.id) {
      activeTools.delete(event.id);
      scheduleEvaluationClosureCheck();
      return;
    }
    if (event.kind === "summary") {
      clearEvaluationClosureTimer();
      return;
    }
    scheduleEvaluationClosureCheck();
  };
  if (args.taskLabel === "evaluation" && attempt.artifactClosure && closureIdleMs > 0) {
    clearEvaluationClosureTimer();
    evaluationClosureTimer = setTimeout(() => {
      if (cancelled || child.killed || childClosed) return;
      const validation = ensureEvaluationArtifact(args.project);
      const idleLabel = closureIdleMs >= 1000 ? `${Math.round(closureIdleMs / 1000)}s` : `${closureIdleMs}ms`;
      if (validation.ok) {
        forcedEvaluationResult = {
          code: 0,
          message: `[liner] Evaluation artifact closure produced a valid YAML file within ${idleLabel}; stopping the idle closure turn.`,
        };
        args.onEvent({ kind: "raw", text: forcedEvaluationResult.message });
        log.write(
          JSON.stringify({
            type: "_liner_artifact_closure_complete",
            phase: "evaluation",
            ms: closureIdleMs,
          }),
        );
        child.kill("SIGTERM");
        return;
      }
      forcedEvaluationResult = {
        code: 1,
        message:
          `[liner] Evaluation artifact closure timed out after ${idleLabel}: ${validation.message}. ` +
          "Retry Evaluation fresh or reduce the candidate set before continuing.",
      };
      args.onEvent({ kind: "raw", text: forcedEvaluationResult.message });
      log.write(
        JSON.stringify({
          type: "_liner_artifact_closure_failed",
          phase: "evaluation",
          ms: closureIdleMs,
          validation: validation.message,
        }),
      );
      child.kill("SIGTERM");
    }, closureIdleMs);
    evaluationClosureTimer.unref?.();
  }

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
          noteAgentEvent(event);
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
      // Surface meaningful stderr as raw events so users see install/auth
      // hints inline, while keeping repetitive cosmetic Codex startup warnings
      // out of the live TUI. The full stream is still recorded in the run log.
      for (const line of chunk.split("\n")) {
        const trimmed = line.trim();
        if (trimmed) {
          // Record stderr in the log too — wrapped so it doesn't masquerade
          // as a real JSON event when the log is replayed.
          log.write(JSON.stringify({ type: "_liner_stderr", text: trimmed }));
          if (shouldSurfaceAgentStderr(args.agent.id, trimmed)) {
            args.onEvent({ kind: "raw", text: trimmed });
          }
        }
      }
    });
  }

  return new Promise<{ code: number | null; stderr: string }>((resolve) => {
    child.on("close", (code) => {
      clearTimeout(watchdog);
      clearEvaluationClosureTimer();
      childClosed = true;
      // Flush any trailing partial line.
      for (const line of buffer.flush()) {
        log.write(line);
        for (const event of parseLine(args.agent.id, line)) {
          noteAgentEvent(event);
          args.onEvent(event);
        }
      }
      closeRunLog(log, code, stderr.length);

      if (forcedEvaluationResult) {
        resolve({
          code: forcedEvaluationResult.code,
          stderr: forcedEvaluationResult.code === 0
            ? stderr
            : stderr
              ? `${stderr}\n${forcedEvaluationResult.message}`
              : forcedEvaluationResult.message,
        });
        return;
      }

      if (evaluationClosureTimedOut && !cancelled) {
        args.onEvent({
          kind: "raw",
          text: "[liner] Resuming Evaluation once to close working/03-evaluation.yaml. No more source fetching in this pass.",
        });
        resolve(
          runAttempt(
            {
              ...args,
              resume: true,
              resumePrompt: evaluationArtifactClosurePrompt(),
            },
            { artifactClosure: true },
            registerCancel,
          ),
        );
        return;
      }

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
        !attempt.modelFallback &&
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
        resolve(runAttempt({ ...args, model: undefined }, { modelFallback: true }, registerCancel));
        return;
      }

      resolve({ code, stderr });
    });
    child.on("error", (e) => {
      clearTimeout(watchdog);
      clearEvaluationClosureTimer();
      args.onEvent({ kind: "raw", text: `[runner error] ${e.message}` });
      log.write(JSON.stringify({ type: "_liner_error", message: e.message }));
      closeRunLog(log, 1, stderr.length);
      resolve({ code: 1, stderr: e.message });
    });
  });
}

export function resumePromptForTask(taskLabel: string): string {
  const base =
    "Continue from where you left off. Pick up the same task; do not re-do tool calls you already completed." +
    `\n\n${WORKSPACE_DISCIPLINE}`;
  if (taskLabel === "candidates") {
    return `${base}\n\nPhase 2 reminder: stop any open-ended web/search loop. If working/05-operating-fit-audit.md exists with status: improvement_recommended, treat this as a focused improvement pass and search for the missing source roles named there. Otherwise use the verified URLs, curator-provided tape sources, titles, and reasons already gathered in this conversation. Write working/02-candidate-longlist.md now, grouped by the knowledge-map sections, replacing the placeholder completely. Do not run more than one final targeted search for a genuinely empty section; if a section is still thin, document the best verified candidates you have and stop after writing the file.`;
  }
  if (taskLabel === "evaluation") {
    return `${base}\n\nPhase 4 reminder: do not get stuck chasing unavailable content. For any candidate whose content is still unavailable, make at most one more recovery attempt, then evaluate from verified metadata, the Phase 2 reason, and any partial content. Every candidate still needs a keep/trim/drop decision. Kept/trim entries need rating, jtbd_fit (direct/bridge/background), section, rationale, fetch_status (readable/partial only), content_quality (high/medium only), at least two content-specific evidence bullets, and the three-part curator note. Do not keep sources from URL/title/search snippets/model memory alone. Prefer section-sized fragments under working/evaluation-decisions/ so Liner can assemble working/03-evaluation.yaml; if you write the final file directly, do not say you are ready to write the artifact before writing it.`;
  }
  if (taskLabel === "quality") {
    return `${base}\n\nPhase 5 reminder: stop any open-ended web/search loop. Quality is a bounded audit, not fresh discovery. Use the existing kept/trim decisions in working/03-evaluation.yaml, assign missing jtbd_fit and source kinds from the title/note/section/source type, repair weak curator notes from existing evidence, spend no more external search attempts, and write working/04-quality-checks.md now with Test 0 core-action fit, Tests 1-6, Perspectives audit, the kind distribution, and the note-quality check. If a material source-role gap remains, write working/05-operating-fit-audit.md with status: improvement_recommended and concrete search lanes for a focused second pass; do not call the corpus ready with a limitation.`;
  }
  return base;
}

export function evaluationArtifactClosurePrompt(): string {
  return [
    "Continue Phase 4, but switch to artifact closure only.",
    "",
    WORKSPACE_DISCIPLINE,
    "",
    "Do not fetch, search, or read external sources in this pass. Use the evidence, metadata, Phase 2 rationale, and partial content already gathered in this conversation.",
    "Write complete section-sized YAML fragments under working/evaluation-decisions/ now, or write working/03-evaluation.yaml directly if it is small enough.",
    "Every URL candidate in working/02-candidate-longlist.md must appear exactly once across those fragments or in the final file.",
    "Use decision: dropped for uncertain, duplicate, unavailable, or metadata-only candidates unless the prior evidence clearly supports kept or trim.",
    "For kept/trim entries include rating, jtbd_fit (direct/bridge/background), fetch_status (readable/partial only), content_quality (high/medium only), at least two content-specific evidence bullets, section, rationale, and the three-part curator note. For dropped entries include URL, title, decision, section if known, and one rationale sentence.",
    "Do not keep sources from URL/title/search snippets/model memory alone; if evidence is not already in the conversation, mark the candidate dropped in this closure pass.",
    "After writing fragments, Liner will assemble working/03-evaluation.yaml; if you write the final file yourself, parse/check it before the required Phase 4 final message.",
  ].join("\n");
}

export function evaluationClosureIdleMs(env: NodeJS.ProcessEnv = process.env): number {
  const raw = env["LINER_EVALUATION_CLOSURE_IDLE_MS"];
  if (raw == null || raw.trim() === "") return DEFAULT_EVALUATION_CLOSURE_IDLE_MS;
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed < 0) return DEFAULT_EVALUATION_CLOSURE_IDLE_MS;
  return parsed;
}

export function buildArgs(
  id: AgentId,
  project: string,
  skillPath: string,
  resume: boolean,
  model?: string,
  resumeSessionId?: string,
): string[] {
  if (id === "claude") {
    // Stream-json requires --verbose. --include-partial-messages would give
    // us mid-message chunking but the cost is ~10× more events; we let the
    // assistant message arrive as one block per turn instead.
    //
    // Tool scoping keeps the agent's reach bounded even with permissions
    // skipped. --tools constrains Claude's built-in tool surface shown in the
    // init stream, while --allowedTools/--disallowedTools guard permission
    // checks and any globally-provided names that may appear despite the
    // built-in list. The methodology needs local file reads/writes plus
    // targeted fetches; it should not shell out, launch subagents, or web-search.
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
      "--tools",
      CLAUDE_METHOD_BUILTIN_TOOLS,
      "--allowedTools",
      CLAUDE_METHOD_TOOLS,
      "--disallowedTools",
      CLAUDE_HARD_DENIED_TOOLS,
      "--disable-slash-commands",
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
      // Codex's `exec resume` subcommand has a narrower option set than fresh
      // exec: there is no --cd or --sandbox flag here. Use Liner's recorded
      // session id instead of --last so a resume cannot pick up unrelated
      // Codex work from the user's machine.
      const sessionArg = resumeSessionId ? [resumeSessionId] : ["--last"];
      return [
        "exec",
        "resume",
        "--skip-git-repo-check",
        "--json",
        ...sessionArg,
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

export function findLatestCodexSessionId(project: string, taskLabel: string): string | undefined {
  const dir = join(project, ".liner-runs", taskLabel.replace(/[^a-zA-Z0-9_-]/g, "-") || "task");
  if (!existsSync(dir)) return undefined;
  const files = readdirSync(dir)
    .filter((name) => name.endsWith(".jsonl"))
    .map((name) => join(dir, name))
    .filter((path) => {
      try {
        return statSync(path).isFile();
      } catch {
        return false;
      }
    })
    .sort((a, b) => {
      try {
        return statSync(b).mtimeMs - statSync(a).mtimeMs;
      } catch {
        return 0;
      }
    });

  const fallback: string[] = [];
  for (const file of files) {
    const candidate = codexSessionIdFromRunLog(file, false);
    if (candidate) return candidate;
    const resumedCandidate = codexSessionIdFromRunLog(file, true);
    if (resumedCandidate) fallback.push(resumedCandidate);
  }
  return fallback[0];
}

function codexSessionIdFromRunLog(path: string, allowResumeLog: boolean): string | undefined {
  let metaAgent = "";
  let metaResume = false;
  try {
    for (const line of readFileSync(path, "utf8").split("\n")) {
      if (!line.trim()) continue;
      let value: unknown;
      try {
        value = JSON.parse(line);
      } catch {
        continue;
      }
      if (!value || typeof value !== "object") continue;
      const record = value as Record<string, unknown>;
      if (record.type === "_liner_meta") {
        metaAgent = typeof record.agent === "string" ? record.agent : "";
        metaResume = record.resume === true;
        continue;
      }
      if (record.type !== "thread.started") continue;
      if (metaAgent !== "codex") return undefined;
      if (metaResume && !allowResumeLog) return undefined;
      return typeof record.thread_id === "string" && record.thread_id.trim() !== ""
        ? record.thread_id
        : undefined;
    }
  } catch {
    return undefined;
  }
  return undefined;
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

export function shouldSurfaceAgentStderr(agent: AgentId, line: string): boolean {
  const value = line.trim();
  if (!value) return false;
  if (agent !== "codex") return true;
  return !isNoisyCodexStderr(value);
}

function isNoisyCodexStderr(line: string): boolean {
  const value = line.toLowerCase();
  return (
    isCosmeticCodexStartupWarning(line) ||
    isCodexGlobalSkillLoadWarning(value) ||
    isNoisyCodexMcpTransport(value) ||
    value.includes("rmcp::transport::auth") ||
    (value.includes("oauth") && (value.includes("mcp") || value.includes("connector"))) ||
    value.includes("write_stdin failed: stdin is closed")
  );
}

function isCodexGlobalSkillLoadWarning(value: string): boolean {
  return value.includes("failed to load skill");
}

function isNoisyCodexMcpTransport(value: string): boolean {
  return value.includes("rmcp::transport::worker") && value.includes("transport channel closed");
}

function isCosmeticCodexStartupWarning(line: string): boolean {
  return (
    line.includes("codex_core_plugins::manifest: ignoring interface.defaultPrompt") ||
    line.includes("codex_core_skills::loader: ignoring interface.icon_small") ||
    line.includes("codex_core_skills::loader: ignoring interface.icon_large")
  );
}
