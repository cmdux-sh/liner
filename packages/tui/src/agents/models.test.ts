import { describe, it, expect } from "vitest";
import { modelForPhase } from "./models.js";

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

  it("downgrades the codex heavy phases by default", () => {
    expect(modelForPhase("codex", "candidates")).toBe("gpt-5-mini");
    expect(modelForPhase("codex", "evaluation")).toBe("gpt-5-mini");
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
});
