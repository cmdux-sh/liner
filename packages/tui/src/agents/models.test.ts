import { describe, it, expect } from "vitest";
import { modelForPhase, resolveModelForPhase, resolveRunProfile } from "./models.js";

describe("modelForPhase", () => {
  it("downgrades the heavy phases by default (claude → sonnet)", () => {
    expect(modelForPhase("claude", "candidates")).toBe("sonnet");
    expect(modelForPhase("claude", "evaluation")).toBe("sonnet");
  });

  it("leaves taste-heavy phases on the agent default (undefined)", () => {
    expect(modelForPhase("claude", "synthesis")).toBeUndefined();
    expect(modelForPhase("claude", "quality")).toBeUndefined();
    expect(modelForPhase("claude", "framing")).toBeUndefined();
    expect(modelForPhase("claude", "assembly")).toBeUndefined();
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
    expect(modelForPhase("claude", "evaluation", { candidates: "opus" })).toBe("sonnet");
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
  it("routes OpenAI Auto tasks through the approved Luna and Sol tiers", () => {
    for (const task of ["jtbd-clarify", "candidates", "evaluation"] as const) {
      expect(resolveRunProfile("codex", task)).toMatchObject({
        model: "gpt-5.6-luna",
        reasoningEffort: "high",
        modelSource: "auto",
        effortSource: "auto",
      });
    }

    for (const task of ["framing", "quality", "synthesis", "improvement", "assembly"] as const) {
      expect(resolveRunProfile("codex", task)).toMatchObject({
        model: "gpt-5.6-sol",
        reasoningEffort: "medium",
        modelSource: "auto",
        effortSource: "auto",
      });
    }
  });

  it("keeps explicit phase, global model, global effort, and provider-default choices authoritative", () => {
    expect(resolveRunProfile("codex", "candidates", { candidates: "phase-model" })).toMatchObject({
      model: "phase-model",
      reasoningEffort: "high",
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

  it("leaves Claude routing and thinking behavior unchanged", () => {
    expect(resolveRunProfile("claude", "candidates")).toMatchObject({
      model: "sonnet",
      reasoningEffort: undefined,
      modelSource: "builtin",
      effortSource: "default",
    });
  });
});
