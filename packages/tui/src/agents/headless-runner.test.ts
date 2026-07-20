import { homedir, tmpdir } from "node:os";
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  classifyRunnerFailure,
  parseHeadlessArgs,
  preflightAgent,
  resolveHeadlessSkillPath,
  selectHeadlessAgent,
  writeHeadlessEvent,
  type HeadlessArgs,
} from "./headless-runner.js";
import type { AgentDescriptor } from "./types.js";
import type { UserConfig } from "../config.js";

const noPreference: UserConfig = {
  agent: null,
  runner: null,
  providerPreferences: null,
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

  it("accepts the dedicated incremental improvement phase", () => {
    const parsed = parseHeadlessArgs([
      "--project=/tmp/tape",
      "--phase=improvement",
      "--agent=codex",
    ]);

    expect(parsed.ok).toBe(true);
    expect((parsed as { ok: true; args: HeadlessArgs }).args.phaseId).toBe("improvement");
  });

  it("rejects improvement resume so every retry gets a fresh filesystem sandbox", () => {
    const parsed = parseHeadlessArgs([
      "--project=/tmp/tape",
      "--phase=improvement",
      "--agent=claude",
      "--resume",
    ]);

    expect(parsed.ok).toBe(false);
    if (parsed.ok) throw new Error("expected parse failure");
    expect(parsed.message).toContain("fresh isolated agent session");
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

  it("uses a persisted runner profile in headless mode", () => {
    const selected = selectHeadlessAgent("auto", [codex], {
      ...noPreference,
      runner: {
        agent: "codex",
        executable: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
    });

    expect(selected).toEqual({
      ok: true,
      agent: {
        id: "codex",
        name: "OpenAI",
        bin: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
    });
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

describe("preflightAgent", () => {
  const roots: string[] = [];

  afterEach(() => {
    for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true });
  });

  function fakeRunner(body: string): { bin: string; configHome: string } {
    const root = mkdtempSync(join(tmpdir(), "liner-runner-preflight-"));
    roots.push(root);
    const bin = join(root, "codex");
    const configHome = join(root, "config");
    mkdirSync(configHome);
    writeFileSync(bin, `#!/bin/sh\n${body}\n`, "utf8");
    chmodSync(bin, 0o755);
    return { bin, configHome };
  }

  it("accepts an identified, capable, authenticated runner", () => {
    const runner = fakeRunner('if [ "$1" = "--version" ]; then echo "codex-cli 1.0"; elif [ "$1" = "--help" ]; then echo "Usage: codex exec"; elif [ "$1" = "login" ]; then echo "Logged in"; else exit 1; fi');

    expect(preflightAgent({ id: "codex", name: "OpenAI Codex", ...runner })).toEqual({ ok: true });
  });

  it("passes the saved Codex config home to every preflight command", () => {
    const runner = fakeRunner("");
    writeFileSync(
      runner.bin,
      `#!/bin/sh
if [ "$CODEX_HOME" != "${runner.configHome}" ]; then
  echo "wrong CODEX_HOME: $CODEX_HOME" >&2
  exit 41
fi
if [ "$1" = "--version" ]; then echo "codex-cli 1.0"; elif [ "$1" = "--help" ]; then echo "Usage: codex exec"; elif [ "$1" = "login" ]; then echo "Logged in"; else exit 1; fi
`,
      "utf8",
    );

    expect(preflightAgent({ id: "codex", name: "OpenAI Codex", ...runner })).toEqual({ ok: true });
  });

  it("returns exact auth remediation before a run can start", () => {
    const runner = fakeRunner('if [ "$1" = "--version" ]; then echo "codex-cli 1.0"; elif [ "$1" = "--help" ]; then echo "Usage: codex exec"; else echo "Not logged in" >&2; exit 1; fi');

    const result = preflightAgent({ id: "codex", name: "OpenAI Codex", ...runner });
    expect(result).toEqual({
      ok: false,
      message: `OpenAI authentication is not ready. Run: ${runner.bin} login`,
    });
  });

  it("rejects an executable that does not identify as the selected runner", () => {
    const runner = fakeRunner('echo "Claude Code 9.0"');

    expect(preflightAgent({ id: "codex", name: "OpenAI Codex", ...runner })).toEqual({
      ok: false,
      message: `Codex CLI executable identity check failed: ${runner.bin}. Update Settings or LINER_CODEX_BIN.`,
    });
  });

  it("rejects a CLI that lacks the headless capability Liner drives", () => {
    const runner = fakeRunner('if [ "$1" = "--version" ]; then echo "codex-cli 1.0"; else echo "Usage: codex interactive"; fi');

    expect(preflightAgent({ id: "codex", name: "OpenAI Codex", ...runner })).toEqual({
      ok: false,
      message: `Codex CLI is not a supported headless runner: ${runner.bin}. Upgrade the CLI and retry.`,
    });
  });

  it("rejects an inaccessible config home with the exact profile fix", () => {
    const runner = fakeRunner('echo "codex-cli 1.0"');
    const missing = join(runner.configHome, "missing");

    expect(preflightAgent({
      id: "codex",
      name: "OpenAI Codex",
      bin: runner.bin,
      configHome: missing,
    })).toEqual({
      ok: false,
      message: `OpenAI config home is not accessible: ${missing}. Update Settings or LINER_CODEX_HOME.`,
    });
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

describe("classifyRunnerFailure", () => {
  it("keeps the fatal version error primary and separates integration warnings", () => {
    expect(classifyRunnerFailure([
      "WARN codex_core_skills::loader: failed to load skill /tmp/SKILL.md",
      "ERROR rmcp::transport::worker: MCP connector unavailable",
      "Error: Codex CLI version 0.9 is unsupported; version 1.0 or newer is required.",
    ].join("\n"))).toEqual({
      message: "Codex CLI version 0.9 is unsupported; version 1.0 or newer is required.",
      recovery: "Upgrade the configured AI runner, then retry this phase.",
      diagnostics: [
        "WARN codex_core_skills::loader: failed to load skill /tmp/SKILL.md",
        "ERROR rmcp::transport::worker: MCP connector unavailable",
      ],
    });
  });

  it("returns no primary cause when stderr contains diagnostics only", () => {
    expect(classifyRunnerFailure("WARN integration: optional connector unavailable")).toEqual({
      message: "",
      recovery: "",
      diagnostics: ["WARN integration: optional connector unavailable"],
    });
  });

  it("promotes a required integration error when it is the stopping cause", () => {
    expect(classifyRunnerFailure("ERROR rmcp::transport::worker: required MCP connector unavailable")).toEqual({
      message: "rmcp::transport::worker: required MCP connector unavailable",
      recovery: "Repair the required AI runner integration, then retry this phase.",
      diagnostics: [],
    });
  });

  it("ranks actionable authentication details above an earlier generic error", () => {
    expect(classifyRunnerFailure([
      "Error: request failed",
      "Authentication token expired; run codex login.",
    ].join("\n"))).toEqual({
      message: "Authentication token expired; run codex login.",
      recovery: "Authenticate the configured AI runner, then retry this phase.",
      diagnostics: [],
    });
  });

  it("keeps integration errors diagnostic when the run succeeds", () => {
    expect(classifyRunnerFailure(
      "ERROR rmcp::transport::worker: MCP connector unavailable",
      false,
    )).toEqual({
      message: "",
      recovery: "",
      diagnostics: ["ERROR rmcp::transport::worker: MCP connector unavailable"],
    });
  });

  it("uses generic exit fallback for stderr without a failure signal", () => {
    expect(classifyRunnerFailure("info: shutting down")).toEqual({
      message: "",
      recovery: "",
      diagnostics: [],
    });
  });
});
