import { describe, expect, it } from "vitest";
import { existsSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  ensureLocalSourceFolders,
  importSourceInboxText,
  parseSourceInboxTokens,
} from "./source-inbox.js";

describe("parseSourceInboxTokens", () => {
  it("extracts URLs from messy pasted text", () => {
    expect(
      parseSourceInboxTokens(`
        - https://example.com/article
        2. https://youtu.be/abc,
        https://github.com/user/repo/tree/main/skills/writing
      `),
    ).toEqual([
      "https://example.com/article",
      "https://youtu.be/abc",
      "https://github.com/user/repo/tree/main/skills/writing",
    ]);
  });

  it("splits atomic space-separated values", () => {
    expect(parseSourceInboxTokens("terminal-ui local-sources/report.pdf")).toEqual([
      "terminal-ui",
      "local-sources/report.pdf",
    ]);
  });
});

describe("importSourceInboxText", () => {
  it("ensures the local source folders exist", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-inbox-folders-"));

    ensureLocalSourceFolders(project);

    expect(existsSync(join(project, "local-sources"))).toBe(true);
    expect(existsSync(join(project, "local-sources", "captured"))).toBe(true);
    expect(existsSync(join(project, "local-sources", "skills"))).toBe(true);
  });

  it("imports web, youtube, skill name, and GitHub skill URL", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-inbox-"));
    const result = importSourceInboxText(
      "https://example.com https://youtu.be/abc terminal-ui https://github.com/user/repo/tree/main/skills/writing",
      project,
    );
    expect(result.warnings).toEqual([]);
    expect(result.sources.map((source) => source.type)).toEqual([
      "web",
      "youtube",
      "skill",
      "skill",
    ]);
    expect(result.sources[2]?.path).toBe("terminal-ui");
    expect(result.sources[3]?.url).toContain("github.com");
  });

  it("copies absolute local files into local-sources", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-inbox-project-"));
    const sourceDir = mkdtempSync(join(tmpdir(), "liner-inbox-source-"));
    const file = join(sourceDir, "paper.md");
    writeFileSync(file, "# Paper\n\nUseful.", "utf8");

    const result = importSourceInboxText(file, project);

    expect(result.sources).toHaveLength(1);
    expect(result.sources[0]?.type).toBe("local_file");
    expect(result.sources[0]?.path).toBe("local-sources/paper.md");
    expect(existsSync(join(project, "local-sources", "paper.md"))).toBe(true);
  });

  it("captures pasted article text as a local source", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-inbox-capture-"));
    const result = importSourceInboxText(
      [
        "A Useful Article About Source Capture",
        "",
        "This article explains why rendered article text is useful for authenticated sources.",
        "It includes enough real prose to be treated as captured source content.",
        "The curator copied it from a browser session they already had permission to access.",
      ].join("\n"),
      project,
    );

    expect(result.warnings).toEqual([]);
    expect(result.sources).toHaveLength(1);
    expect(result.sources[0]?.type).toBe("local_file");
    expect(result.sources[0]?.path).toMatch(/^local-sources\/captured\/.+\.md$/);
    expect(existsSync(join(project, result.sources[0]?.path ?? ""))).toBe(true);
  });

  it("captures multiple pasted article blocks as separate local sources", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-inbox-multi-capture-"));
    const result = importSourceInboxText(
      [
        "First Article About Design Systems",
        "This first article has enough prose to be captured as one local source.",
        "It discusses interaction timing, motion, and how designers make judgment calls.",
        "--- source ---",
        "Second Article About Writing Voice",
        "This second article has enough prose to become another local source.",
        "It includes examples, constraints, and preferences for how the user wants to write.",
      ].join("\n"),
      project,
    );

    expect(result.warnings).toEqual([]);
    expect(result.sources).toHaveLength(2);
    expect(result.sources.map((source) => source.type)).toEqual(["local_file", "local_file"]);
    expect(result.sources[0]?.path).not.toBe(result.sources[1]?.path);
  });
});
