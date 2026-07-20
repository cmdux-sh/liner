// Per-task model and Thinking-effort selection.
//
// OpenAI Auto uses Luna/high for research-heavy tasks and Sol/medium for
// judgment-heavy tasks. Claude retains its existing Sonnet downgrades for
// Candidate discovery and Evaluation. A provider preference or hand-edited
// phase override remains authoritative; choosing provider default disables
// the OpenAI Auto policy.

import type { AgentId } from "./types.js";
import type { PhaseId } from "../phases.js";

export type PhaseModelMap = Partial<Record<PhaseId, string>>;
export type ModelMode = "auto" | "default";
export type ModelTaskId = PhaseId | "jtbd-clarify";
export type ReasoningEffort = "none" | "low" | "medium" | "high" | "xhigh" | "max";
export type ProviderPreference = {
  model?: string | null;
  reasoningEffort?: ReasoningEffort | null;
  modelMode?: ModelMode | null;
};
export type ModelResolution = {
  model: string | undefined;
  source: "phase" | "provider" | "builtin" | "default";
};
export type RunProfileResolution = {
  model: string | undefined;
  reasoningEffort: string | undefined;
  modelSource: ModelResolution["source"] | "auto";
  effortSource: "provider" | "auto" | "default";
};

const AUTO_CODEX_POLICY: Partial<Record<ModelTaskId, { model: string; reasoningEffort: string }>> = {
  "jtbd-clarify": { model: "gpt-5.6-luna", reasoningEffort: "high" },
  candidates: { model: "gpt-5.6-luna", reasoningEffort: "high" },
  evaluation: { model: "gpt-5.6-luna", reasoningEffort: "high" },
  framing: { model: "gpt-5.6-sol", reasoningEffort: "medium" },
  quality: { model: "gpt-5.6-sol", reasoningEffort: "medium" },
  synthesis: { model: "gpt-5.6-sol", reasoningEffort: "medium" },
  improvement: { model: "gpt-5.6-sol", reasoningEffort: "medium" },
  assembly: { model: "gpt-5.6-sol", reasoningEffort: "medium" },
};

// Default downgrades, applied unless the user overrides them in config. The
// values are the agent CLIs' own model identifiers/aliases.
const DEFAULT_MODELS: Record<AgentId, PhaseModelMap> = {
  claude: { candidates: "sonnet", evaluation: "sonnet" },
  codex: {},
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
  providerModel?: string | null,
): string | undefined {
  return resolveModelForPhase(agent, phaseId, overrides, providerModel).model;
}

export function resolveModelForPhase(
  agent: AgentId,
  phaseId: PhaseId,
  overrides?: PhaseModelMap,
  providerModel?: string | null,
): ModelResolution {
  if (overrides && Object.prototype.hasOwnProperty.call(overrides, phaseId)) {
    return { model: overrides[phaseId] || undefined, source: "phase" };
  }
  if (providerModel?.trim()) {
    return { model: providerModel.trim(), source: "provider" };
  }
  const builtin = defaultModelMap(agent)[phaseId];
  if (builtin) return { model: builtin, source: "builtin" };
  return { model: undefined, source: "default" };
}

/** Resolve the complete provider invocation profile for one Liner AI task. */
export function resolveRunProfile(
  agent: AgentId,
  taskId: ModelTaskId,
  overrides?: PhaseModelMap,
  preference?: ProviderPreference,
): RunProfileResolution {
  const providerModel = preference?.model?.trim() || undefined;
  const autoProfile = agent === "codex" ? AUTO_CODEX_POLICY[taskId] : undefined;
  const autoActive = agent === "codex" && !providerModel && preference?.modelMode !== "default";

  let model: string | undefined;
  let modelSource: RunProfileResolution["modelSource"] = "default";
  if (Object.prototype.hasOwnProperty.call(overrides || {}, taskId)) {
    model = overrides?.[taskId as PhaseId] || undefined;
    modelSource = "phase";
  } else if (providerModel) {
    model = providerModel;
    modelSource = "provider";
  } else if (autoActive && autoProfile) {
    model = autoProfile.model;
    modelSource = "auto";
  } else {
    const builtin = defaultModelMap(agent)[taskId as PhaseId];
    if (builtin) {
      model = builtin;
      modelSource = "builtin";
    }
  }

  if (agent !== "codex") {
    return { model, reasoningEffort: undefined, modelSource, effortSource: "default" };
  }
  const providerEffort = preference?.reasoningEffort?.trim() || undefined;
  if (providerEffort) {
    return { model, reasoningEffort: providerEffort, modelSource, effortSource: "provider" };
  }
  if (autoActive && autoProfile) {
    return { model, reasoningEffort: autoProfile.reasoningEffort, modelSource, effortSource: "auto" };
  }
  return { model, reasoningEffort: undefined, modelSource, effortSource: "default" };
}
