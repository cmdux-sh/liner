import { afterEach, describe, expect, it } from "vitest";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  configHomeForAgent,
  detectAgents,
  resolveSkillPathWithDiagnostics,
} from "./detect.js";

const originalSkillPath = process.env["LINER_SKILL_PATH"];
const originalCodexBin = process.env["LINER_CODEX_BIN"];
const originalCodexHome = process.env["LINER_CODEX_HOME"];
const originalClaudeHome = process.env["LINER_CLAUDE_HOME"];
const originalNativeClaudeHome = process.env["CLAUDE_CONFIG_DIR"];
const tempDirs: string[] = [];

afterEach(() => {
  if (originalSkillPath === undefined) delete process.env["LINER_SKILL_PATH"];
  else process.env["LINER_SKILL_PATH"] = originalSkillPath;
  if (originalCodexBin === undefined) delete process.env["LINER_CODEX_BIN"];
  else process.env["LINER_CODEX_BIN"] = originalCodexBin;
  if (originalCodexHome === undefined) delete process.env["LINER_CODEX_HOME"];
  else process.env["LINER_CODEX_HOME"] = originalCodexHome;
  if (originalClaudeHome === undefined) delete process.env["LINER_CLAUDE_HOME"];
  else process.env["LINER_CLAUDE_HOME"] = originalClaudeHome;
  if (originalNativeClaudeHome === undefined) delete process.env["CLAUDE_CONFIG_DIR"];
  else process.env["CLAUDE_CONFIG_DIR"] = originalNativeClaudeHome;

  for (const dir of tempDirs.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
});

function tempSkillBundle(): string {
  const dir = mkdtempSync(join(tmpdir(), "liner-skill-test-"));
  tempDirs.push(dir);
  writeFileSync(join(dir, "SKILL.md"), "# Test skill\n", "utf8");
  return dir;
}

describe("resolveSkillPathWithDiagnostics", () => {
  it("honors LINER_SKILL_PATH when it points at a skill directory", () => {
    const dir = tempSkillBundle();
    process.env["LINER_SKILL_PATH"] = dir;

    const resolved = resolveSkillPathWithDiagnostics();

    expect(resolved.path).toBe(dir);
    expect(resolved.envPath).toBe(dir);
  });

  it("accepts LINER_SKILL_PATH pointing directly at SKILL.md", () => {
    const dir = tempSkillBundle();
    process.env["LINER_SKILL_PATH"] = join(dir, "SKILL.md");

    const resolved = resolveSkillPathWithDiagnostics();

    expect(resolved.path).toBe(dir);
    expect(resolved.envPath).toBe(dir);
  });

  it("falls back to bundled locations when LINER_SKILL_PATH is stale", () => {
    process.env["LINER_SKILL_PATH"] = join(tmpdir(), "missing-liner-skill");

    const resolved = resolveSkillPathWithDiagnostics();

    expect(resolved.path).toEqual(expect.stringMatching(/(?:curation-skill|cli-update-docs)/));
    expect(resolved.searched[0]).toBe(join(tmpdir(), "missing-liner-skill", "SKILL.md"));
  });
});

describe("AI runner detection", () => {
  it("uses explicit executable and config-home overrides", () => {
    process.env["LINER_CODEX_BIN"] = "/opt/liner/codex";
    process.env["LINER_CODEX_HOME"] = "/opt/liner/codex-home";

    expect(detectAgents()).toContainEqual({
      id: "codex",
	  name: "OpenAI",
      bin: "/opt/liner/codex",
      configHome: "/opt/liner/codex-home",
    });
  });

  it("uses the standard agent config home when no Liner override is set", () => {
    delete process.env["LINER_CLAUDE_HOME"];
    delete process.env["CLAUDE_CONFIG_DIR"];
    expect(configHomeForAgent("claude")).toBe(join(process.env["HOME"] || process.env["USERPROFILE"] || "", ".claude"));
  });
});
