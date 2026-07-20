import { beforeEach, describe, expect, it, vi } from "vitest";
import { chmodSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { buildClarifyingQuestionsPrompt, elicitClarifyingQuestions } from "./jtbd-clarify.js";

const readConfigMock = vi.hoisted(() => vi.fn<() => any>(() => ({
  providerPreferences: { codex: { model: "gpt-5.6-sol", reasoningEffort: "max" } },
})));

vi.mock("../config.js", () => ({ readConfig: readConfigMock }));

beforeEach(() => {
  readConfigMock.mockReturnValue({
    providerPreferences: { codex: { model: "gpt-5.6-sol", reasoningEffort: "max" } },
  });
});

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

  it("uses the Luna High Auto model policy for Clarify Job", async () => {
    readConfigMock.mockReturnValue({
      providerPreferences: { codex: { model: null, modelMode: "auto" } },
    });
    const cwd = mkdtempSync(join(tmpdir(), "liner-clarify-auto-"));
    const runner = join(cwd, "fake-codex");
    writeFileSync(
      runner,
      [
        "#!/bin/sh",
        'printf "%s\\n" "$@" > "$PWD/args"',
        `echo '${JSON.stringify({ type: "item.completed", item: { id: "item_0", type: "agent_message", text: '["What is the input?","What should the output contain?"]' } })}'`,
        "",
      ].join("\n"),
    );
    chmodSync(runner, 0o755);

    await expect(elicitClarifyingQuestions({
      jtbd: "Help an agent research a narrow capability.",
      agent: { id: "codex", name: "OpenAI", bin: runner },
      skillPath: cwd,
      cwd,
    })).resolves.toHaveLength(2);

    const args = readFileSync(join(cwd, "args"), "utf8");
    expect(args).toContain("--model\ngpt-5.6-luna");
    expect(args).toContain('model_reasoning_effort="high"');
  });

  it("surfaces an explicit model rejection without retrying or substituting", async () => {
    const cwd = mkdtempSync(join(tmpdir(), "liner-clarify-model-"));
    const runner = join(cwd, "fake-codex");
    writeFileSync(
      runner,
      [
        "#!/bin/sh",
        'printf "x" >> "$PWD/attempts"',
        'echo "unknown model gpt-5.6-sol" >&2',
        "exit 1",
        "",
      ].join("\n"),
    );
    chmodSync(runner, 0o755);

    await expect(
      elicitClarifyingQuestions({
        jtbd: "Read visual references and turn them into web art direction.",
        agent: { id: "codex", name: "OpenAI", bin: runner },
        skillPath: cwd,
        cwd,
      }),
    ).rejects.toThrow(
      '[liner] OpenAI rejected configured model "gpt-5.6-sol". Choose another model in Settings; Liner did not substitute another model.',
    );
    expect(readFileSync(join(cwd, "attempts"), "utf8")).toBe("x");
  });

  it("surfaces an explicit Thinking effort rejection without retrying or substituting", async () => {
    const cwd = mkdtempSync(join(tmpdir(), "liner-clarify-effort-"));
    const runner = join(cwd, "fake-codex");
    writeFileSync(
      runner,
      [
        "#!/bin/sh",
        'printf "x" >> "$PWD/attempts"',
        'echo "invalid value max for model_reasoning_effort" >&2',
        "exit 1",
        "",
      ].join("\n"),
    );
    chmodSync(runner, 0o755);

    await expect(
      elicitClarifyingQuestions({
        jtbd: "Read visual references and turn them into web art direction.",
        agent: { id: "codex", name: "OpenAI", bin: runner },
        skillPath: cwd,
        cwd,
      }),
    ).rejects.toThrow(
      '[liner] OpenAI rejected configured Thinking effort "max". Choose another effort in Settings; Liner did not substitute another effort or model.',
    );
    expect(readFileSync(join(cwd, "attempts"), "utf8")).toBe("x");
  });
});
