// A single source of truth for "how far has the curator walked through the
// methodology" — replaces the swiss-cheese per-artifact status detection
// that produced check-marks out of order whenever a previous run or
// hand-edit had touched a later artifact first.
//
// The file is dead-simple: one integer plus a timestamp.
//
//   { "step": 4, "lastTouched": "2026-05-19T22:00:00.000Z" }
//
// `step` is the count of completed phases. With 10 phases in PHASE_ORDER,
// step ranges 0..10:
//
//   0  → nothing done; user is currently on PHASE_ORDER[0] (Framing)
//   N  → N phases done; user is currently on PHASE_ORDER[N]
//   10 → everything done (Compile is the last phase; compiled = complete)

import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { projectFolder } from "./yaml-io.js";
import type { Tape } from "./types.js";
import type { PhaseId } from "./phases.js";

const PROGRESS_FILE = ".liner-progress.json";

/** Walk order. Position in this array IS the step value the user is "on". */
export const PHASE_ORDER: readonly PhaseId[] = [
  "framing",
  "gate0",
  "candidates",
  "gate1",
  "evaluation",
  "quality",
  "gate2",
  "synthesis",
  "assembly",
  "compile",
];

export const TOTAL_STEPS = PHASE_ORDER.length;

export type Progress = {
  step: number;
  lastTouched: string;
};

export function readProgress(folder: string): Progress {
  const path = join(folder, PROGRESS_FILE);
  if (!existsSync(path)) {
    return { step: 0, lastTouched: new Date().toISOString() };
  }
  try {
    const raw = JSON.parse(readFileSync(path, "utf8"));
    return {
      step:
        typeof raw.step === "number"
          ? Math.max(0, Math.min(TOTAL_STEPS, Math.floor(raw.step)))
          : 0,
      lastTouched:
        typeof raw.lastTouched === "string"
          ? raw.lastTouched
          : new Date().toISOString(),
    };
  } catch {
    return { step: 0, lastTouched: new Date().toISOString() };
  }
}

export function writeProgress(folder: string, state: Progress): void {
  const path = join(folder, PROGRESS_FILE);
  writeFileSync(path, JSON.stringify(state, null, 2) + "\n", "utf8");
}

export function progressFileExists(folder: string): boolean {
  return existsSync(join(folder, PROGRESS_FILE));
}

/**
 * Advance the progress cursor when a phase completes. Idempotent: calling
 * twice for the same phase is a no-op. Calling for a phase that isn't the
 * current one is also a no-op (we don't skip ahead silently).
 */
export function markPhaseComplete(folder: string, phaseId: PhaseId): Progress {
  const current = readProgress(folder);
  const expected = PHASE_ORDER.indexOf(phaseId);
  if (expected === -1) return current;
  if (current.step !== expected) {
    // Out of sync — caller is asking us to advance from somewhere we aren't.
    // Don't move the cursor. Caller should reconcile before retrying.
    return current;
  }
  const next: Progress = {
    step: Math.min(TOTAL_STEPS, current.step + 1),
    lastTouched: new Date().toISOString(),
  };
  writeProgress(folder, next);
  return next;
}

/**
 * One-time inference for projects that predate progress.json. Walks
 * PHASE_ORDER and finds the longest consecutive prefix of "done" phases,
 * then commits that as the starting step.
 *
 * "Done" here uses the same artifact heuristics that drove status in the
 * old per-phase model — so a project where the agent ran Phase 1 and Phase 2
 * before this rewrite migrates to step = 2 cleanly. Out-of-order artifacts
 * (Phase 7 done but Phase 2 empty) are ignored — the consecutive-prefix
 * rule reflects what the curator has *actually walked through*, not what
 * files happen to be on disk.
 */
export function migrateProgress(folder: string, tape: Tape): Progress {
  const project = projectFolder(folder);
  const gateState = readGateStateInline(folder);

  let step = 0;
  for (const phaseId of PHASE_ORDER) {
    if (!isPhaseComplete(phaseId, folder, project, tape, gateState)) break;
    step++;
  }

  const state: Progress = {
    step,
    lastTouched: new Date().toISOString(),
  };
  writeProgress(folder, state);
  return state;
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

type GateState = {
  gate0Accepted: boolean;
  gate1Accepted: boolean;
  gate2Accepted: boolean;
};

function readGateStateInline(folder: string): GateState {
  // Cheap re-read — kept local to avoid a circular import with phases.ts.
  const path = join(folder, ".liner-gates.json");
  if (!existsSync(path)) {
    return { gate0Accepted: false, gate1Accepted: false, gate2Accepted: false };
  }
  try {
    const raw = JSON.parse(readFileSync(path, "utf8"));
    return {
      gate0Accepted: Boolean(raw.gate0Accepted),
      gate1Accepted: Boolean(raw.gate1Accepted),
      gate2Accepted: Boolean(raw.gate2Accepted),
    };
  } catch {
    return { gate0Accepted: false, gate1Accepted: false, gate2Accepted: false };
  }
}

function isPhaseComplete(
  phaseId: PhaseId,
  folder: string,
  project: ReturnType<typeof projectFolder>,
  tape: Tape,
  gates: GateState,
): boolean {
  switch (phaseId) {
    case "framing":
      return artifactHasRealContent(join(folder, "working/01-jtbd-and-knowledge-map.md")) &&
        Boolean(tape.jtbd?.trim());
    case "gate0":
      return gates.gate0Accepted;
    case "candidates":
      return artifactHasRealContent(join(folder, "working/02-candidate-longlist.md"));
    case "gate1":
      return gates.gate1Accepted;
    case "evaluation":
      return artifactHasRealContent(join(folder, "working/03-evaluation.yaml"));
    case "quality":
      return artifactHasRealContent(join(folder, "working/04-quality-checks.md"));
    case "gate2":
      return gates.gate2Accepted;
    case "synthesis": {
      if (!existsSync(project.synthesisPath)) return false;
      const text = readFileSync(project.synthesisPath, "utf8");
      return text.trim().length > 0 && !text.includes("Replace this placeholder");
    }
    case "assembly":
      return tape.sources.length > 0;
    case "compile":
      return existsSync(project.mixtapePath);
  }
}

const PLACEHOLDER_MARKERS = [
  "TODO — a single specific sentence",
  // Knowledge-map section placeholder — survives the wizard's JTBD pre-fill
  // (which only strips the JTBD-specific TODO above) and gets removed by
  // Phase 1 when the agent writes real sections. Without this marker, the
  // freshly-pre-filled working/01 would falsely register as "Framing done."
  "TODO — Phase 1 replaces this",
  "Quantity over precision",
  "candidates: []",
  "Run each test deliberately",
];

function artifactHasRealContent(path: string): boolean {
  if (!existsSync(path)) return false;
  const text = readFileSync(path, "utf8");
  const trimmed = text.trim();
  if (trimmed.length === 0) return false;
  for (const marker of PLACEHOLDER_MARKERS) {
    if (text.includes(marker)) return false;
  }
  return true;
}
