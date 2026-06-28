#!/usr/bin/env node
// Deterministic headless-runner double for Go TUI acceptance smokes.
//
// It intentionally fails a fresh Framing run, succeeds when retried with
// --resume, then fails Candidate discovery so the smoke stops quickly after the
// retry path is proven.

import { appendFileSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const args = parseArgs(process.argv.slice(2));
const project = args.project;
const phase = args.phase;
const resume = args.resume;

if (!project || !phase) {
  console.error("go-tui-fake-headless-runner requires --project and --phase");
  process.exit(2);
}

const threadId = `fake-${phase}-thread`;
const runDir = join(project, ".liner-runs", phase);
mkdirSync(runDir, { recursive: true });
mkdirSync(join(project, "working"), { recursive: true });

const stamp = new Date().toISOString().replace(/[:.]/g, "-");
const runPath = join(runDir, `${stamp}-${resume ? "resume" : "fresh"}.jsonl`);

writeRunLog({ type: "_liner_meta", agent: "fake", phaseId: phase, resume });
writeRunLog({ type: "thread.started", thread_id: threadId });
writeRunLog({ type: "item.started", item: { type: "agent_message" } });

emit({ kind: "runner_start", phaseId: phase, agent: "fake", resume });
emit({ kind: "text", text: `${resume ? "Resumed" : "Fresh"} fake ${phase} run.` });

if (phase === "framing" && !resume) {
  emit({ kind: "runner_error", message: "Intentional fake Framing failure for retry acceptance." });
  writeRunLog({ type: "_liner_close", exitCode: 42 });
  console.error("Intentional fake Framing failure.");
  process.exit(42);
}

if (phase === "framing" && resume) {
  writeFileSync(
    join(project, "working", "01-jtbd-and-knowledge-map.md"),
    [
      "# JTBD and Knowledge Map",
      "",
      "Fake Framing artifact written by the retry acceptance runner.",
      "",
      "## 1. Retry proof",
      "",
      "- The fresh run failed.",
      "- The resumed run wrote this artifact.",
      "",
    ].join("\n"),
    "utf8",
  );
  writeRunLog({ type: "item.completed", item: { type: "agent_message" } });
  writeRunLog({ type: "_liner_close", exitCode: 0 });
  emit({ kind: "runner_done", code: 0 });
  process.exit(0);
}

if (phase === "candidates") {
  emit({ kind: "runner_error", message: "Stopping after retry proof; Candidate discovery intentionally fails." });
  writeRunLog({ type: "_liner_close", exitCode: 43 });
  console.error("Intentional fake Candidate stop after retry proof.");
  process.exit(43);
}

emit({ kind: "runner_error", message: `Unexpected fake phase: ${phase}` });
writeRunLog({ type: "_liner_close", exitCode: 44 });
process.exit(44);

function emit(event) {
  process.stdout.write(`${JSON.stringify(event)}\n`);
}

function writeRunLog(event) {
  appendFileSync(runPath, `${JSON.stringify(event)}\n`, "utf8");
}

function parseArgs(raw) {
  const parsed = { project: "", phase: "", resume: false };
  for (let i = 0; i < raw.length; i += 1) {
    const arg = raw[i];
    if (arg === "--project") {
      i += 1;
      parsed.project = raw[i] || "";
    } else if (arg === "--phase") {
      i += 1;
      parsed.phase = raw[i] || "";
    } else if (arg === "--resume") {
      parsed.resume = true;
    }
  }
  return parsed;
}
