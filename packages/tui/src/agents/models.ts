// Per-phase model selection.
//
// Candidate discovery + evaluation are ~65% of a cycle's token cost, and their
// work is fetch/triage-heavy rather than taste-heavy — so by default we run
// those two phases on a cheaper model (Sonnet for Claude, gpt-5-mini for
// Codex) and leave framing/quality/synthesis/assembly on the agent's own
// default, where curator judgment matters most. On a subscription the win is
// rate-limit quota, not dollars.
//
// A phase absent from the map runs on the agent's default model (no --model
// flag). Users override per agent + phase in ~/.liner/config.yaml — set a
// phase back to a flagship model to undo the downgrade.

import type { AgentId } from "./types.js";
import type { PhaseId } from "../phases.js";

export type PhaseModelMap = Partial<Record<PhaseId, string>>;

// Default downgrades, applied unless the user overrides them in config. The
// values are the agent CLIs' own model identifiers/aliases: Claude accepts the
// `sonnet` alias; Codex takes a model id via -m/--model.
const DEFAULT_MODELS: Record<AgentId, PhaseModelMap> = {
  claude: { candidates: "sonnet", evaluation: "sonnet" },
  codex: { candidates: "gpt-5-mini", evaluation: "gpt-5-mini" },
};

export function defaultModelMap(agent: AgentId): PhaseModelMap {
  return DEFAULT_MODELS[agent] ?? {};
}

/**
 * Resolve the model for a phase. A user override for that exact phase wins
 * (including an empty string, read as "use the agent default"); otherwise the
 * default downgrade map applies; otherwise undefined → the agent's own model.
 */
export function modelForPhase(
  agent: AgentId,
  phaseId: PhaseId,
  overrides?: PhaseModelMap,
): string | undefined {
  if (overrides && Object.prototype.hasOwnProperty.call(overrides, phaseId)) {
    return overrides[phaseId] || undefined;
  }
  return defaultModelMap(agent)[phaseId];
}
