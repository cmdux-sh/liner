// Tee-style audit log of every agent run.
//
// We already get resume-on-failure for free via Claude's session.json (and
// Codex's equivalent), so this log isn't load-bearing for recovery. What it
// *does* give us:
//
//   - A user-readable transcript of past runs they can scroll back through.
//   - Forensic data when something behaves weirdly (rate-limit retry order,
//     unexpected tool calls, parse failures in the JSON stream).
//   - Raw material for a future "replay a run in the UI" feature.
//
// One file per run, namespaced by phaseId, timestamped. The first line is a
// `_liner_meta` envelope describing the run; the body is the agent's raw
// JSONL exactly as we received it; the last line is a `_liner_close` marker
// with the exit code and end time. Everything else (parsed events, derived
// UI state) is reconstructable from those primitives.

import { createWriteStream, mkdirSync } from "node:fs";
import { join } from "node:path";
import type { WriteStream } from "node:fs";
import type { AgentId } from "./types.js";

const RUN_DIR = ".liner-runs";

export type RunLog = {
  /** Absolute path to the JSONL file being written. */
  path: string;
  /** Append a line. A trailing newline is added if absent. */
  write: (line: string) => void;
  /** Close the underlying stream. Safe to call more than once. */
  close: () => void;
};

export type RunLogMeta = {
  /**
   * Bucket label for the audit log directory — usually a PhaseId, but any
   * filesystem-safe slug works (e.g. "jtbd-clarify" for the wizard's
   * elicitation side task). Lives at `.liner-runs/<taskLabel>/<ts>.jsonl`.
   */
  taskLabel: string;
  agent: AgentId;
  resume: boolean;
  /** ISO timestamp set at log creation. */
  startedAt: string;
};

/**
 * Open a new log file for an agent run. The directory is created lazily;
 * no IO until the first write. Caller owns close() in both happy and
 * cancellation paths.
 */
export function openRunLog(
  folder: string,
  taskLabel: string,
  meta: Omit<RunLogMeta, "taskLabel" | "startedAt">,
): RunLog {
  // Defense-in-depth: sanitize taskLabel so a malformed caller can't escape
  // the .liner-runs/ directory with `..` or absolute paths.
  const safeLabel = taskLabel.replace(/[^a-zA-Z0-9_-]/g, "-") || "task";
  const dir = join(folder, RUN_DIR, safeLabel);
  mkdirSync(dir, { recursive: true });
  const startedAt = new Date().toISOString();
  // Filenames need to be filesystem-safe on Windows too — strip colons.
  const stamp = startedAt.replace(/:/g, "-").replace(/\..+/, "");
  const path = join(dir, `${stamp}.jsonl`);
  const stream: WriteStream = createWriteStream(path, { flags: "a" });
  let closed = false;

  const fullMeta: RunLogMeta = { ...meta, taskLabel: safeLabel, startedAt };
  stream.write(
    JSON.stringify({ type: "_liner_meta", ...fullMeta }) + "\n",
    "utf8",
  );

  return {
    path,
    write(line: string) {
      if (closed) return;
      if (!line) return;
      stream.write(line, "utf8");
      if (!line.endsWith("\n")) stream.write("\n", "utf8");
    },
    close() {
      if (closed) return;
      closed = true;
      stream.end();
    },
  };
}

/**
 * Convenience: append a final `_liner_close` marker with exit metadata before
 * the stream is shut. Callers call this from the agent's `close` handler.
 */
export function closeRunLog(log: RunLog, code: number | null, stderrLen: number): void {
  log.write(
    JSON.stringify({
      type: "_liner_close",
      exitCode: code,
      stderrBytes: stderrLen,
      endedAt: new Date().toISOString(),
    }),
  );
  log.close();
}
