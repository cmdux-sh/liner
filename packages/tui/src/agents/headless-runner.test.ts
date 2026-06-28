import { homedir } from "node:os";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import {
  parseHeadlessArgs,
  resolveHeadlessSkillPath,
  selectHeadlessAgent,
  writeHeadlessEvent,
  type HeadlessArgs,
} from "./headless-runner.js";
import type { AgentDescriptor } from "./types.js";
import type { UserConfig } from "../config.js";

const noPreference: UserConfig = {
  agent: null,
  models: null,
  jsSetupPrompted: false,
};

describe("parseHeadlessArgs", () => {
  it("parses the minimum project and phase flags", () => {
    const parsed = parseHeadlessArgs(["--project", "/tmp/tape", "--phase", "framing"]);

    expect(parsed.ok).toBe(true);
    expect((parsed as { ok: true; args: HeadlessArgs }).args).toMatchObject({
      project: "/tmp/tape",
      phaseId: "framing",
      agent: "auto",
      resume: false,
    });
  });

  it("parses inline values and resume", () => {
    const parsed = parseHeadlessArgs([
      "--project=/tmp/tape",
      "--phase=evaluation",
      "--agent=codex",
      "--skill-path=/tmp/skill",
      "--resume",
    ]);

    expect(parsed.ok).toBe(true);
    expect((parsed as { ok: true; args: HeadlessArgs }).args).toMatchObject({
      project: "/tmp/tape",
      phaseId: "evaluation",
      agent: "codex",
      skillPath: "/tmp/skill",
      resume: true,
    });
  });

  it("rejects gate and compile phases because the agent runner does not own them", () => {
    const parsed = parseHeadlessArgs(["--project", "/tmp/tape", "--phase", "compile"]);

    if (parsed.ok) throw new Error("expected parse failure");
    expect(parsed.message).toContain("--phase must be one of");
  });
});

describe("resolveHeadlessSkillPath", () => {
  it("normalizes an explicit relative skill path against the runner cwd", () => {
    expect(resolveHeadlessSkillPath("docs/curation-skill")).toBe(
      resolve(process.cwd(), "docs/curation-skill"),
    );
  });

  it("accepts a SKILL.md file path and returns the containing directory", () => {
    expect(resolveHeadlessSkillPath("/tmp/curating-mixtapes/SKILL.md")).toBe(
      "/tmp/curating-mixtapes",
    );
  });

  it("expands home-relative paths", () => {
    expect(resolveHeadlessSkillPath("~/skills/curating-mixtapes")).toBe(
      join(homedir(), "skills/curating-mixtapes"),
    );
  });
});

describe("selectHeadlessAgent", () => {
  const claude: AgentDescriptor = { id: "claude", name: "Claude Code", bin: "claude" };
  const codex: AgentDescriptor = { id: "codex", name: "OpenAI Codex", bin: "codex" };

  it("uses the only installed agent in auto mode", () => {
    const selected = selectHeadlessAgent("auto", [claude], noPreference);

    expect(selected).toEqual({ ok: true, agent: claude });
  });

  it("honors the configured agent when multiple agents are installed", () => {
    const selected = selectHeadlessAgent("auto", [claude, codex], {
      ...noPreference,
      agent: "codex",
    });

    expect(selected).toEqual({ ok: true, agent: codex });
  });

  it("requires an explicit choice when multiple agents are installed with no preference", () => {
    const selected = selectHeadlessAgent("auto", [claude, codex], noPreference);

    if (selected.ok) throw new Error("expected selection failure");
    expect(selected.message).toContain("Multiple agents found");
  });

  it("reports when an explicitly requested agent is missing", () => {
    const selected = selectHeadlessAgent("codex", [claude], noPreference);

    if (selected.ok) throw new Error("expected selection failure");
    expect(selected.message).toContain("codex agent was requested");
  });
});

describe("writeHeadlessEvent", () => {
  it("writes JSONL events", () => {
    const lines: string[] = [];

    const ok = writeHeadlessEvent(
      { kind: "runner_error", message: "nope" },
      (line) => lines.push(line),
    );

    expect(ok).toBe(true);
    expect(lines).toEqual(['{"kind":"runner_error","message":"nope"}\n']);
  });

  it("treats broken stdout pipes as a clean stop signal", () => {
    const error = new Error("write EPIPE") as Error & { code: string };
    error.code = "EPIPE";

    const ok = writeHeadlessEvent(
      { kind: "runner_error", message: "nope" },
      () => {
        throw error;
      },
    );

    expect(ok).toBe(false);
  });

  it("rethrows non-pipe write errors", () => {
    const error = new Error("disk on fire") as Error & { code: string };
    error.code = "NOPE";

    expect(() =>
      writeHeadlessEvent(
        { kind: "runner_error", message: "nope" },
        () => {
          throw error;
        },
      ),
    ).toThrow("disk on fire");
  });
});
