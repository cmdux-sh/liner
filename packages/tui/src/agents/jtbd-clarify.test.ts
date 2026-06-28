import { describe, expect, it } from "vitest";
import { buildClarifyingQuestionsPrompt, elicitClarifyingQuestions } from "./jtbd-clarify.js";

describe("buildClarifyingQuestionsPrompt", () => {
  it("sharpens the future-agent capability without canned questions or source-lane asks", () => {
    const text = buildClarifyingQuestionsPrompt("Read visual references and turn them into web art direction.");

    expect(text).toContain("future AI agent");
    expect(text).toContain("Ask questions that are specific to this capability");
    expect(text).toContain("runtime inputs and output contract");
    expect(text).not.toContain("What sources");
  });

  it("throws instead of returning fallback questions when no agent is available", async () => {
    await expect(
      elicitClarifyingQuestions({
        jtbd: "Read visual references and turn them into web art direction.",
        agent: null,
        skillPath: "/tmp/skill",
        cwd: "/tmp",
      }),
    ).rejects.toThrow("Clarifying questions require");
  });
});
