import { afterEach, describe, expect, it } from "vitest";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { resolveSkillPathWithDiagnostics } from "./detect.js";

const originalSkillPath = process.env["LINER_SKILL_PATH"];
const tempDirs: string[] = [];

afterEach(() => {
  if (originalSkillPath === undefined) delete process.env["LINER_SKILL_PATH"];
  else process.env["LINER_SKILL_PATH"] = originalSkillPath;

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
