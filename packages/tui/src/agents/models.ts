// Per-task model and Thinking-effort selection.
//
// OpenAI Auto runs Sol 6.1 at Medium for every task. Claude runs Sonnet 5.5 at
// Medium unless Settings chooses another model or effort. A provider
// preference or hand-edited phase override remains authoritative; choosing
// provider default disables the OpenAI Auto policy.

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

export const AUTO_OPENAI_MODEL = "gpt-6.1-sol";
export const AUTO_OPENAI_EFFORT: ReasoningEffort = "medium";
export const DEFAULT_CLAUDE_MODEL = "claude-sonnet-5-5";
export const DEFAULT_CLAUDE_EFFORT: ReasoningEffort = "medium";

// Claude Code's --effort levels; "none" is OpenAI-only.
export const CLAUDE_REASONING_EFFORTS: ReadonlySet<ReasoningEffort> = new Set([
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
]);

// Claude Code aliases saved by earlier Liner versions, and the models that
// replaced them in Settings.
const LEGACY_CLAUDE_MODELS: Record<string, string> = {
  sonnet: DEFAULT_CLAUDE_MODEL,
  opus: "claude-opus-5-5",
};

export function normalizeProviderModel(agent: AgentId, model: string): string {
  const value = model.trim();
  if (agent !== "claude") return value;
  return LEGACY_CLAUDE_MODELS[value.toLowerCase()] ?? value;
}

// Built-in models when neither a phase override nor a provider preference is
// set. The values are the agent CLIs' own model identifiers.
const DEFAULT_MODELS: Record<AgentId, string | undefined> = {
  claude: DEFAULT_CLAUDE_MODEL,
  codex: undefined,
};

export function defaultModelFor(agent: AgentId): string | undefined {
  return DEFAULT_MODELS[agent];
}

/**
 * Resolve the model for a phase. A user override for that exact phase wins
 * (including an empty string, read as "use the agent default"); otherwise the
 * provider preference; otherwise the built-in model; otherwise undefined →
 * the agent's own model.
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
  const builtin = defaultModelFor(agent);
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
  const autoActive = agent === "codex" && !providerModel && preference?.modelMode !== "default";

  let model: string | undefined;
  let modelSource: RunProfileResolution["modelSource"] = "default";
  if (Object.prototype.hasOwnProperty.call(overrides || {}, taskId)) {
    model = overrides?.[taskId as PhaseId] || undefined;
    modelSource = "phase";
  } else if (providerModel) {
    model = providerModel;
    modelSource = "provider";
  } else if (autoActive) {
    model = AUTO_OPENAI_MODEL;
    modelSource = "auto";
  } else {
    const builtin = defaultModelFor(agent);
    if (builtin) {
      model = builtin;
      modelSource = "builtin";
    }
  }

  const providerEffort = preference?.reasoningEffort?.trim() || undefined;
  if (providerEffort) {
    return { model, reasoningEffort: providerEffort, modelSource, effortSource: "provider" };
  }
  if (agent === "claude") {
    return { model, reasoningEffort: DEFAULT_CLAUDE_EFFORT, modelSource, effortSource: "auto" };
  }
  if (autoActive) {
    return { model, reasoningEffort: AUTO_OPENAI_EFFORT, modelSource, effortSource: "auto" };
  }
  return { model, reasoningEffort: undefined, modelSource, effortSource: "default" };
}
