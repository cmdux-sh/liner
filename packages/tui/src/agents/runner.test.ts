import { describe, it, expect } from "vitest";
import { looksLikeModelRejection } from "./runner.js";

describe("looksLikeModelRejection", () => {
  it("matches common unknown-model phrasings", () => {
    expect(looksLikeModelRejection("Error: unknown model 'gpt-5-mini'", "gpt-5-mini")).toBe(true);
    expect(looksLikeModelRejection("invalid model: sonnet", "sonnet")).toBe(true);
    expect(looksLikeModelRejection("model not found", "whatever")).toBe(true);
    expect(looksLikeModelRejection('{"error":"model_not_found"}', "gpt-5-mini")).toBe(true);
  });

  it("matches when the model id is echoed next to an error token", () => {
    expect(looksLikeModelRejection("API error: gpt-5-mini is unavailable", "gpt-5-mini")).toBe(true);
  });

  it("does not fire on unrelated / auth failures", () => {
    expect(looksLikeModelRejection("", "gpt-5-mini")).toBe(false);
    expect(
      looksLikeModelRejection("Authentication failed: please run `claude login`", "sonnet"),
    ).toBe(false);
    expect(looksLikeModelRejection("rate limit exceeded", "sonnet")).toBe(false);
  });

  it("is case-insensitive", () => {
    expect(looksLikeModelRejection("UNKNOWN MODEL", "x")).toBe(true);
  });
});
