import { describe, it, expect } from "vitest";
import { modelForPhase, resolveModelForPhase, resolveRunProfile } from "./models.js";

describe("modelForPhase", () => {
  it("runs every Claude phase on Sonnet 5.5 by default", () => {
    for (const phase of ["candidates", "evaluation", "synthesis", "quality", "framing", "assembly"] as const) {
      expect(modelForPhase("claude", phase)).toBe("claude-sonnet-5-5");
    }
  });

  it("keeps Codex on the CLI default by default because supported model ids vary by account", () => {
    expect(modelForPhase("codex", "candidates")).toBeUndefined();
    expect(modelForPhase("codex", "evaluation")).toBeUndefined();
  });

  it("lets a user override win over the default map", () => {
    expect(modelForPhase("claude", "candidates", { candidates: "opus" })).toBe("opus");
  });

  it("treats an empty-string override as 'use the agent default'", () => {
    expect(modelForPhase("claude", "candidates", { candidates: "" })).toBeUndefined();
  });

  it("an override for one phase doesn't affect another", () => {
    expect(modelForPhase("claude", "evaluation", { candidates: "opus" })).toBe("claude-sonnet-5-5");
  });

  it("still lets users pin a Codex model explicitly", () => {
    expect(modelForPhase("codex", "candidates", { candidates: "gpt-5" })).toBe("gpt-5");
  });

  it("places provider preferences between phase overrides and built-in defaults", () => {
    expect(modelForPhase("claude", "candidates", undefined, "opus")).toBe("opus");
    expect(modelForPhase("claude", "candidates", { candidates: "sonnet" }, "opus")).toBe("sonnet");
    expect(modelForPhase("claude", "candidates", { candidates: "" }, "opus")).toBeUndefined();
  });

  it("identifies only built-in defaults as fallback-compatible", () => {
    expect(resolveModelForPhase("claude", "candidates").source).toBe("builtin");
    expect(resolveModelForPhase("claude", "candidates", undefined, "opus").source).toBe("provider");
    expect(resolveModelForPhase("claude", "candidates", { candidates: "opus" }).source).toBe("phase");
  });
});

describe("resolveRunProfile", () => {
  it("runs every OpenAI Auto task on Sol 6.1 at Medium", () => {
    for (const task of [
      "jtbd-clarify",
      "candidates",
      "evaluation",
      "framing",
      "quality",
      "synthesis",
      "improvement",
      "assembly",
    ] as const) {
      expect(resolveRunProfile("codex", task)).toMatchObject({
        model: "gpt-6.1-sol",
        reasoningEffort: "medium",
        modelSource: "auto",
        effortSource: "auto",
      });
    }
  });

  it("keeps explicit phase, global model, global effort, and provider-default choices authoritative", () => {
    expect(resolveRunProfile("codex", "candidates", { candidates: "phase-model" })).toMatchObject({
      model: "phase-model",
      reasoningEffort: "medium",
      modelSource: "phase",
      effortSource: "auto",
    });
    expect(resolveRunProfile("codex", "candidates", undefined, {
      model: "gpt-5.6-terra",
      reasoningEffort: "xhigh",
    })).toMatchObject({
      model: "gpt-5.6-terra",
      reasoningEffort: "xhigh",
      modelSource: "provider",
      effortSource: "provider",
    });
    expect(resolveRunProfile("codex", "candidates", undefined, { modelMode: "default" })).toMatchObject({
      model: undefined,
      reasoningEffort: undefined,
      modelSource: "default",
      effortSource: "default",
    });
  });

  it("runs Claude on Sonnet 5.5 at Medium unless Settings chooses otherwise", () => {
    expect(resolveRunProfile("claude", "candidates")).toMatchObject({
      model: "claude-sonnet-5-5",
      reasoningEffort: "medium",
      modelSource: "builtin",
      effortSource: "auto",
    });
    expect(resolveRunProfile("claude", "synthesis", undefined, {
      model: "claude-opus-5-5",
      reasoningEffort: "high",
    })).toMatchObject({
      model: "claude-opus-5-5",
      reasoningEffort: "high",
      modelSource: "provider",
      effortSource: "provider",
    });
  });
});
