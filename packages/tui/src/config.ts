// User-level config persisted to ~/.liner/config.yaml.
//
// The shape stays an object so settings can grow without a migration. Anything
// missing from the file resolves to a safe default on read; the caller does
// not have to undefined-check.

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import * as YAML from "yaml";
import type { AgentDescriptor, AgentId } from "./agents/types.js";
import type { ModelMode, PhaseModelMap, ProviderPreference, ReasoningEffort } from "./agents/models.js";
export type { ProviderPreference, ReasoningEffort } from "./agents/models.js";

export type UserConfig = {
  /**
   * The agent the user chose during setup. `null` means "no preference set"
   * — fall back to detection (and the mid-phase picker if multiple agents
   * are installed).
   */
  agent: AgentId | null;
  /** Durable runner identity. Contains paths only; credentials never live here. */
  runner: AIRunnerProfile | null;
  /** Provider-level preferences applied to fresh runs only. */
  providerPreferences: Partial<Record<AgentId, ProviderPreference>> | null;
  /**
   * Per-agent, per-phase model overrides. `null` (or a missing key) means
   * "use the built-in default map" — see `agents/models.ts`, which downgrades
   * the heavy phases (candidates + evaluation) to a cheaper model. Set a phase
   * here to a flagship model to undo that downgrade, e.g.
   * `models: { claude: { candidates: opus } }`.
   */
  models: Partial<Record<AgentId, PhaseModelMap>> | null;
  /**
   * Whether the first-run onboarding has already offered to install JS
   * rendering support (Playwright's Chromium download). This records the
   * prompt, not whether Chromium is currently installed; compile still detects
   * a missing browser and offers repair when a page needs it.
   */
  jsSetupPrompted: boolean;
};

export type AIRunnerProfile = {
  agent: AgentId;
  executable: string;
  configHome: string;
};

const KNOWN_AGENT_IDS: readonly AgentId[] = ["claude", "codex"];

const EMPTY_CONFIG: UserConfig = {
  agent: null,
  runner: null,
  providerPreferences: null,
  models: null,
  jsSetupPrompted: false,
};

const REASONING_EFFORTS = new Set<ReasoningEffort>([
  "none",
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
]);

function parseProviderPreferences(
  raw: unknown,
): Partial<Record<AgentId, ProviderPreference>> | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const preferences: Partial<Record<AgentId, ProviderPreference>> = {};
  for (const agent of KNOWN_AGENT_IDS) {
    const entry = (raw as Record<string, unknown>)[agent];
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) continue;
    const model = (entry as Record<string, unknown>)["model"];
    const modelMode = (entry as Record<string, unknown>)["model_mode"];
    const effort = (entry as Record<string, unknown>)["reasoning_effort"];
    const preference: ProviderPreference = {
      model: typeof model === "string" && model.trim() ? model.trim() : null,
    };
    if (agent === "codex" && typeof effort === "string" && REASONING_EFFORTS.has(effort.trim() as ReasoningEffort)) {
      preference.reasoningEffort = effort.trim() as ReasoningEffort;
    }
    if (agent === "codex" && (modelMode === "auto" || modelMode === "default")) {
      preference.modelMode = modelMode;
    }
    if (!preference.model && !preference.modelMode && !preference.reasoningEffort) continue;
    preferences[agent] = preference;
  }
  return Object.keys(preferences).length > 0 ? preferences : null;
}

function parseRunner(raw: unknown): AIRunnerProfile | null {
  if (!raw || typeof raw !== "object") return null;
  const value = raw as Record<string, unknown>;
  const agent = value["agent"];
  const executable = value["executable"];
  const configHome = value["config_home"] ?? value["configHome"];
  if (
    (agent !== "claude" && agent !== "codex") ||
    typeof executable !== "string" || !executable.trim() ||
    typeof configHome !== "string" || !configHome.trim()
  ) return null;
  return { agent, executable: executable.trim(), configHome: configHome.trim() };
}

function readRawConfig(): Record<string, unknown> {
  if (!existsSync(CONFIG_PATH)) return {};
  try {
    const raw = YAML.parse(readFileSync(CONFIG_PATH, "utf8")) as unknown;
    return raw && typeof raw === "object" && !Array.isArray(raw)
      ? raw as Record<string, unknown>
      : {};
  } catch {
    return {};
  }
}

/**
 * Best-effort parse of the `models` config block. Anything that isn't a
 * `{ agent: { phase: string } }` shape is dropped silently — a malformed
 * override should never crash the TUI or block a run, just fall back to the
 * default map.
 */
function parseModels(
  raw: unknown,
): Partial<Record<AgentId, PhaseModelMap>> | null {
  if (!raw || typeof raw !== "object") return null;
  const out: Partial<Record<AgentId, PhaseModelMap>> = {};
  for (const agentId of KNOWN_AGENT_IDS) {
    const block = (raw as Record<string, unknown>)[agentId];
    if (!block || typeof block !== "object") continue;
    const phaseMap: PhaseModelMap = {};
    for (const [phase, model] of Object.entries(block as Record<string, unknown>)) {
      if (typeof model === "string") phaseMap[phase as keyof PhaseModelMap] = model;
    }
    if (Object.keys(phaseMap).length > 0) out[agentId] = phaseMap;
  }
  return Object.keys(out).length > 0 ? out : null;
}

const CONFIG_DIR = join(homedir(), ".liner");
const CONFIG_PATH = join(CONFIG_DIR, "config.yaml");

export function configPath(): string {
  return CONFIG_PATH;
}

export function configExists(): boolean {
  return existsSync(CONFIG_PATH);
}

/**
 * Read the user-level config. Returns a fully-populated config with `null`
 * defaults for any field that's missing or malformed — callers don't have
 * to undefined-check. Never throws; a corrupt config file degrades to "no
 * preference set" rather than crashing the TUI.
 */
export function readConfig(): UserConfig {
  if (!existsSync(CONFIG_PATH)) {
    return { ...EMPTY_CONFIG };
  }
  try {
    const raw = YAML.parse(readFileSync(CONFIG_PATH, "utf8")) as
      | Record<string, unknown>
      | null;
    if (!raw || typeof raw !== "object") {
      return { ...EMPTY_CONFIG };
    }
    const agent = raw["agent"];
    const validAgent: AgentId | null =
      agent === "claude" || agent === "codex" ? agent : null;
    return {
      agent: validAgent,
      runner: parseRunner(raw["runner"]),
      providerPreferences: parseProviderPreferences(raw["provider_preferences"]),
      models: parseModels(raw["models"]),
      jsSetupPrompted: raw["jsSetupPrompted"] === true,
    };
  } catch {
    return { ...EMPTY_CONFIG };
  }
}

/**
 * Persist a config patch, merged over whatever is already on disk. Callers
 * that only touch one field (e.g. the agent picker) leave the rest — including
 * hand-edited model overrides — intact.
 */
export function writeConfig(patch: Partial<UserConfig>): void {
  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true });
  }
  const merged = readRawConfig();
  if (patch.agent !== undefined) merged["agent"] = patch.agent;
  if (patch.runner !== undefined) {
    merged["runner"] = patch.runner === null ? null : {
      agent: patch.runner.agent,
      executable: patch.runner.executable,
      config_home: patch.runner.configHome,
    };
  }
  if (patch.providerPreferences !== undefined) {
    if (patch.providerPreferences === null) {
      merged["provider_preferences"] = null;
    } else {
      const existing = merged["provider_preferences"];
      const preferences =
        existing && typeof existing === "object" && !Array.isArray(existing)
          ? (existing as Record<string, unknown>)
          : {};
      for (const [agent, preference] of Object.entries(patch.providerPreferences)) {
        const current = preferences[agent];
        const entry =
          current && typeof current === "object" && !Array.isArray(current)
            ? (current as Record<string, unknown>)
            : {};
        if (preference && Object.prototype.hasOwnProperty.call(preference, "model")) {
          const model = preference.model?.trim() || "";
          if (model) entry["model"] = model;
          else delete entry["model"];
        }
        if (agent === "codex" && preference && Object.prototype.hasOwnProperty.call(preference, "modelMode")) {
          const modelMode = preference.modelMode;
          if (modelMode) entry["model_mode"] = modelMode;
          else delete entry["model_mode"];
        }
        if (agent === "codex" && preference && Object.prototype.hasOwnProperty.call(preference, "reasoningEffort")) {
          const effort = preference.reasoningEffort;
          if (effort) entry["reasoning_effort"] = effort;
          else delete entry["reasoning_effort"];
        }
        if (Object.keys(entry).length > 0) preferences[agent] = entry;
        else delete preferences[agent];
      }
      merged["provider_preferences"] = preferences;
    }
  }
  if (patch.models !== undefined) merged["models"] = patch.models;
  if (patch.jsSetupPrompted !== undefined) merged["jsSetupPrompted"] = patch.jsSetupPrompted;
  // Stable, hand-editable output. Comment at the top so curators who poke at
  // the file know what's there and how to extend it.
  const body = [
    "# Liner user config — generated by the TUI.",
    "# Hand-edit if you know what you want; the TUI rewrites supported",
    "# settings when they change.",
    "#",
    "# models: per-agent, per-phase model overrides. The heavy phases",
    "#   (candidates, evaluation) default to a cheaper model — set them back to",
    "#   a flagship model here to undo that, e.g.:",
    "#     models:",
    "#       claude:",
    "#         candidates: opus",
    "#         evaluation: opus",
    "# provider_preferences.codex.model_mode: auto routes each fresh task",
    "#   through Liner's built-in model and Thinking effort policy; default",
    "#   leaves both choices to the OpenAI provider unless explicitly pinned.",
    "# jsSetupPrompted: true after the first-run JS rendering prompt has shown.",
    "",
    YAML.stringify(merged),
  ].join("\n");
  writeFileSync(CONFIG_PATH, body, "utf8");
}

/**
 * Pick which agent to use. Returns null when no agents are available at all.
 * Priority order:
 *   1. LINER_AGENT env var (escape hatch for CI / power users)
 *   2. The configured agent in ~/.liner/config.yaml, IF it's installed
 *   3. The single installed agent (when only one is available)
 *   4. null — caller should show a picker
 */
export function resolveConfiguredAgent(
  installed: AgentDescriptor[],
  config: UserConfig = readConfig(),
): AgentDescriptor | null {
  const envPin = (process.env["LINER_AGENT"] || "").toLowerCase();
  if (envPin) {
    if (config.runner?.agent === envPin) {
      return descriptorFromProfile(config.runner);
    }
    const pinned = installed.find((a) => a.id === envPin);
    if (pinned) return pinned;
    // An explicit but invalid override must fail closed. Falling through to
    // a saved runner would violate the advertised precedence and hide the
    // broken automation configuration.
    return null;
  }

  if (config.runner) {
    return descriptorFromProfile(config.runner);
  }

  if (installed.length === 0) return null;

  if (config.agent) {
    const configured = installed.find((a) => a.id === config.agent);
    if (configured) return configured;
    // The configured agent isn't installed anymore (user uninstalled it).
    // Fall through; the caller will either pick the remaining one or
    // re-prompt setup.
  }

  if (installed.length === 1) {
    return installed[0]!;
  }

  return null;
}

function descriptorFromProfile(profile: AIRunnerProfile): AgentDescriptor {
  const isClaude = profile.agent === "claude";
  const executable = process.env[isClaude ? "LINER_CLAUDE_BIN" : "LINER_CODEX_BIN"]?.trim();
  const configHome =
    process.env[isClaude ? "LINER_CLAUDE_HOME" : "LINER_CODEX_HOME"]?.trim() ||
    process.env[isClaude ? "CLAUDE_CONFIG_DIR" : "CODEX_HOME"]?.trim();
  return {
    id: profile.agent,
    name: isClaude ? "Claude" : "OpenAI",
    bin: executable || profile.executable,
    configHome: configHome || profile.configHome,
  };
}
