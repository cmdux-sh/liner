// Phase model for the curating-mixtapes methodology.
//
// Status is driven by a single progress cursor (see progress.ts), not by
// per-artifact filesystem checks. Detail strings still pull from the
// filesystem for context (source counts, character counts, etc.) but they
// no longer drive the ✓ / ◐ / ○ badge — the cursor does. This eliminates
// the swiss-cheese view where a later phase showed complete while earlier
// ones were empty just because some artifact happened to be on disk.

import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { ProjectFolder } from "./yaml-io.js";
import type { Tape } from "./types.js";
import type { Progress } from "./progress.js";
import { PHASE_ORDER, TOTAL_STEPS } from "./progress.js";

export type PhaseId =
  | "framing"
  | "gate0"
  | "candidates"
  | "gate1"
  // Phase 3 (Fetching) deliberately omitted — fetching is implicit at compile
  // time in Liner; surfacing it as a row was actively misleading. The
  // methodology numbering survives in artifact paths only (working/03-*).
  | "evaluation"
  | "quality"
  | "gate2"
  | "synthesis"
  | "assembly"
  | "compile";

export type PhaseStatus =
  | "not_started"
  | "in_progress"
  | "needs_review"
  | "complete"
  | "blocked";

/**
 * One actionable piece of work inside a phase, with a visible done/not-done
 * marker. Mostly a no-op (single artifact = single sub-step that doesn't need
 * its own row), but valuable for phases like Framing where the curator
 * supplies the capability goal via the wizard and the agent supplies the
 * capability brief + knowledge map later — distinct units, only one of which
 * is done at this point.
 */
export type SubStep = {
  /** True when this piece is finished. Drives the ✓ vs ○ glyph + color. */
  done: boolean;
  /** Imperative, terse — "Goal set", "Capability brief drafted". */
  text: string;
  /** Optional secondary detail rendered dim after `text`. */
  hint?: string;
};

export type PhaseRecord = {
  id: PhaseId;
  number: string;
  label: string;
  summary: string;
  artifact: string | null;
  status: PhaseStatus;
  detail: string;
  /**
   * Optional checklist of finer-grained work inside this phase. When present,
   * the focus card renders it below the summary so the curator sees what's
   * already done (the goal they typed in the wizard) and what's still
   * pending (the capability brief / knowledge map). Omit when the phase has only one unit of
   * work; the existing `detail` line covers that case fine.
   */
  subSteps?: SubStep[];
};

// ---------------------------------------------------------------------------
// Gate state — separate file because gates can be accepted independently of
// step (e.g., methodology-mode user re-accepts an existing gate).
// ---------------------------------------------------------------------------

export type GateState = {
  /** Confirm framing — between Phase 1 and Phase 2. */
  gate0Accepted: boolean;
  /** Confirm candidates — between Phase 2 and Phase 3 (Evaluation). */
  gate1Accepted: boolean;
  /** Confirm evaluation — between Phase 4 (Quality) and Phase 5 (Synthesis). */
  gate2Accepted: boolean;
};

const GATE_FILE = ".liner-gates.json";

export function readGateState(folder: ProjectFolder): GateState {
  const path = join(folder.path, GATE_FILE);
  if (!existsSync(path)) {
    return { gate0Accepted: false, gate1Accepted: false, gate2Accepted: false };
  }
  try {
    const parsed = JSON.parse(readFileSync(path, "utf8")) as Partial<GateState>;
    return {
      gate0Accepted: Boolean(parsed.gate0Accepted),
      gate1Accepted: Boolean(parsed.gate1Accepted),
      gate2Accepted: Boolean(parsed.gate2Accepted),
    };
  } catch {
    return { gate0Accepted: false, gate1Accepted: false, gate2Accepted: false };
  }
}

export function writeGateState(folder: ProjectFolder, state: GateState): void {
  const path = join(folder.path, GATE_FILE);
  writeFileSync(path, JSON.stringify(state, null, 2), "utf8");
}

// ---------------------------------------------------------------------------
// Phase list construction
// ---------------------------------------------------------------------------

// Phase `number` is the user-visible step indicator in the TUI. We renumber
// the methodology's 1-8 (with Phase 3 = Fetching) into sequential 1-7 for
// the curator — fetching happens implicitly at compile time, so the user
// shouldn't see a "Phase 4 — Evaluation" come right after "Phase 2 — Candidate
// discovery" and wonder where Phase 3 went. The methodology numbering is
// still preserved in the artifact filenames (working/01-, 02-, 03-, 04-, 05-)
// — those follow the spec's canonical naming.
const PHASE_TEMPLATES: Record<PhaseId, Omit<PhaseRecord, "status" | "detail">> = {
  framing: {
    id: "framing",
    number: "1",
    label: "Framing",
    summary: "Define the capability brief and sketch a knowledge map.",
    artifact: "working/01-jtbd-and-knowledge-map.md",
  },
  gate0: {
    id: "gate0",
    number: "↳",
    label: "Confirm framing",
    summary: "Review the knowledge map before any candidate search.",
    artifact: null,
  },
  candidates: {
    id: "candidates",
    number: "2",
    label: "Candidate discovery",
    summary: "Long-list of candidate sources (URLs only — no fetching).",
    artifact: "working/02-candidate-longlist.md",
  },
  gate1: {
    id: "gate1",
    number: "↳",
    label: "Confirm candidates",
    summary: "Review the long-list before any fetching.",
    artifact: null,
  },
  evaluation: {
    id: "evaluation",
    number: "3",
    label: "Evaluation",
    summary: "Keep / trim / drop with curator notes.",
    artifact: "working/03-evaluation.yaml",
  },
  quality: {
    id: "quality",
    number: "4",
    label: "Quality checks",
    summary: "Redundancy, coverage, disagreement, framing-gap tests.",
    artifact: "working/04-quality-checks.md",
  },
  gate2: {
    id: "gate2",
    number: "↳",
    label: "Confirm evaluation",
    summary: "Review keep-list and findings before synthesis.",
    artifact: null,
  },
  synthesis: {
    id: "synthesis",
    number: "5",
    label: "Synthesis",
    summary: "Curator's distilled understanding (800–2000 words).",
    artifact: "synthesis.md",
  },
  assembly: {
    id: "assembly",
    number: "6",
    label: "Assembly",
    summary: "Write tape.yaml — sources with curator notes, sections, order.",
    artifact: "tape.yaml",
  },
  compile: {
    id: "compile",
    number: "→",
    label: "Compile",
    summary: "Final mix is assembled. Compile fetches sources and writes MIXTAPE.md.",
    artifact: null,
  },
};

/**
 * Compute the phase list for display. Status comes from `progress.step`;
 * detail strings are informational and pull from the filesystem for context.
 */
export function computePhases(
  folder: ProjectFolder,
  tape: Tape,
  progress: Progress,
): PhaseRecord[] {
  return PHASE_ORDER.map((phaseId, index) => {
    const template = PHASE_TEMPLATES[phaseId];
    const status = statusForIndex(index, progress.step);
    const detail = detailFor(phaseId, status, folder, tape);
    const subSteps = subStepsFor(phaseId, status, folder, tape);
    return subSteps
      ? { ...template, status, detail, subSteps }
      : { ...template, status, detail };
  });
}

/**
 * Build the optional sub-step checklist for a phase. Returns undefined for
 * phases that have one unit of work (most of them). Currently only Framing
 * uses this — the wizard supplies the capability goal up front, but the
 * capability brief and knowledge map are the agent's job, so we want both shown
 * even though the phase isn't done yet.
 */
function subStepsFor(
  phaseId: PhaseId,
  status: PhaseStatus,
  folder: ProjectFolder,
  tape: Tape,
): SubStep[] | undefined {
  if (phaseId !== "framing") return undefined;

  const jtbdSet = Boolean(tape.jtbd?.trim());
  const clarsCount = tape.jtbd_clarifications?.length ?? 0;
  const chars = readChars(join(folder.path, "working/01-jtbd-and-knowledge-map.md"));
  // Knowledge map is "drafted" once the placeholder sentinel is gone AND
  // the file has substantial content. We approximate via the
  // artifact-has-real-content rule the progress migration uses.
  const mapDrafted = status === "complete";

  const steps: SubStep[] = [];
  steps.push({
    done: jtbdSet,
    text: "Goal set",
    hint: jtbdSet ? undefined : "wizard or hand-edit working/01",
  });
  if (clarsCount > 0) {
    steps.push({
      done: true,
      text: "Clarifications captured",
      hint: `${clarsCount} answered`,
    });
  }
  steps.push({
    done: mapDrafted,
    text: "Capability brief drafted",
    hint: mapDrafted
      ? `${chars.toLocaleString()} chars`
      : status === "in_progress"
        ? "press enter to draft"
        : undefined,
  });
  return steps;
}

function statusForIndex(index: number, step: number): PhaseStatus {
  if (index < step) return "complete";
  if (index === step) return "in_progress";
  return "not_started";
}

function detailFor(
  phaseId: PhaseId,
  status: PhaseStatus,
  folder: ProjectFolder,
  tape: Tape,
): string {
  switch (phaseId) {
    case "framing": {
      const jtbdSet = Boolean(tape.jtbd?.trim());
      const chars = readChars(join(folder.path, "working/01-jtbd-and-knowledge-map.md"));
      if (status === "complete") {
        return `goal set · capability brief drafted (${chars.toLocaleString()} chars)`;
      }
      if (status === "in_progress") {
        return jtbdSet
          ? `goal set — press enter to draft the capability brief`
          : `no AI-agent goal yet — press enter to set one`;
      }
      return "not yet";
    }
    case "gate0": {
      if (status === "complete") return "accepted";
      if (status === "in_progress")
        return tape.mode === "methodology"
          ? "press enter to review the knowledge map"
          : "auto-accept on advance (quick mode)";
      return "not yet";
    }
    case "candidates": {
      const chars = readChars(join(folder.path, "working/02-candidate-longlist.md"));
      if (status === "complete") return `${chars.toLocaleString()} chars in long-list`;
      if (status === "in_progress") return "press enter to start researching candidate sources";
      return "not yet";
    }
    case "gate1": {
      if (status === "complete") return "accepted";
      if (status === "in_progress")
        return tape.mode === "methodology"
          ? "press enter to review and accept"
          : "auto-accept on advance (quick mode)";
      return "not yet";
    }
    case "evaluation": {
      const chars = readChars(join(folder.path, "working/03-evaluation.yaml"));
      if (status === "complete") return `${chars.toLocaleString()} chars · keep/trim/drop set`;
      if (status === "in_progress") return "press enter to run evaluation";
      return "not yet";
    }
    case "quality": {
      const chars = readChars(join(folder.path, "working/04-quality-checks.md"));
      if (status === "complete") return `${chars.toLocaleString()} chars in findings`;
      if (status === "in_progress") return "press enter to run the four quality tests";
      return "not yet";
    }
    case "gate2": {
      if (status === "complete") return "accepted";
      if (status === "in_progress")
        return tape.mode === "methodology"
          ? "press enter to review and accept"
          : "auto-accept on advance (quick mode)";
      return "not yet";
    }
    case "synthesis": {
      const chars = readChars(folder.synthesisPath);
      if (status === "complete") return `synthesis written (${chars.toLocaleString()} chars)`;
      if (status === "in_progress") return "press enter to draft the synthesis";
      return "not yet";
    }
    case "assembly": {
      const n = tape.sources.length;
      if (status === "complete")
        return `${n} source${n === 1 ? "" : "s"} on tape`;
      if (status === "in_progress") return "press enter — agent proposes a draft to review";
      return "not yet";
    }
    case "compile": {
      if (status === "complete") return "MIXTAPE.md on disk";
      if (status === "in_progress")
        return "press enter — write MIXTAPE.md and sources/";
      return "not yet";
    }
  }
}

function readChars(path: string): number {
  if (!existsSync(path)) return 0;
  try {
    return readFileSync(path, "utf8").trim().length;
  } catch {
    return 0;
  }
}

// ---------------------------------------------------------------------------
// Status display helpers (used by all hub UI bits)
// ---------------------------------------------------------------------------

export function statusGlyph(status: PhaseStatus): string {
  switch (status) {
    case "complete":
      return "✓";
    case "in_progress":
      return "▶";
    case "needs_review":
      return "⚠";
    case "blocked":
      return "·";
    case "not_started":
      return "○";
  }
}

export function statusColor(status: PhaseStatus): string | undefined {
  switch (status) {
    case "complete":
      return "green";
    case "in_progress":
      return "cyan";
    case "needs_review":
      return "magenta";
    case "blocked":
      return undefined;
    case "not_started":
      return undefined;
  }
}

export function statusLabel(status: PhaseStatus): string {
  switch (status) {
    case "complete":
      return "complete";
    case "in_progress":
      return "current";
    case "needs_review":
      return "needs review";
    case "blocked":
      return "blocked";
    case "not_started":
      return "not started";
  }
}

/**
 * True when every corpus-build phase is complete. Compile is the last Ink
 * fallback phase, so this means MIXTAPE.md is ready; the Go TUI owns the
 * Operating Layer and Project Complete flow.
 */
export function everythingComplete(progress: Progress): boolean {
  return progress.step >= TOTAL_STEPS;
}
