import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { describe, expect, it } from "vitest";
import { groupCandidateLonglist, readCandidateLonglist } from "./candidate-longlist.js";

function write(path: string, contents: string): string {
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, contents, "utf8");
  return path;
}

describe("readCandidateLonglist", () => {
  it("preserves title, section, URL order, and reason from the longlist", () => {
    const root = mkdtempSync(join(tmpdir(), "liner-longlist-"));
    const path = write(
      join(root, "working/02-candidate-longlist.md"),
      [
        "# Candidate long-list",
        "",
        "## 1. Foundations",
        "",
        "- **Source A** - https://example.com/a",
        "  - Reason: Strong foundation.",
        "- **Source B** - https://example.com/b).",
      ].join("\n"),
    );

    expect(readCandidateLonglist(path)).toEqual([
      {
        url: "https://example.com/a",
        title: "Source A",
        section: "Foundations",
        reason: "Strong foundation.",
      },
      {
        url: "https://example.com/b",
        title: "Source B",
        section: "Foundations",
      },
    ]);
  });

  it("keeps level-three candidate titles inside their level-two knowledge-map section", () => {
    const root = mkdtempSync(join(tmpdir(), "liner-heading-longlist-"));
    const path = write(
      join(root, "working/02-candidate-longlist.md"),
      [
        "# Candidate long-list",
        "",
        "## 1. Foundations",
        "",
        "### Source A",
        "",
        "- URL: https://example.com/a",
        "- Candidate reason: Strong foundation.",
        "",
        "### Source B",
        "",
        "- URL: https://example.com/b",
        "- Candidate reason: Useful counterpoint.",
      ].join("\n"),
    );

    const candidates = readCandidateLonglist(path);
    expect(candidates).toEqual([
      {
        url: "https://example.com/a",
        title: "Source A",
        section: "Foundations",
        reason: "Strong foundation.",
      },
      {
        url: "https://example.com/b",
        title: "Source B",
        section: "Foundations",
        reason: "Useful counterpoint.",
      },
    ]);
    expect(groupCandidateLonglist(candidates)).toMatchObject([
      {
        index: 1,
        total: 1,
        section: "Foundations",
        candidates,
      },
    ]);
  });
});

describe("groupCandidateLonglist", () => {
  it("turns the captured 47-candidate, eight-section shape into eight evaluation groups", () => {
    const sectionSizes = [6, 6, 6, 6, 6, 6, 6, 5];
    const candidates = sectionSizes.flatMap((size, sectionIndex) =>
      Array.from({ length: size }, (_, candidateIndex) => ({
        url: `https://example.com/${sectionIndex + 1}/${candidateIndex + 1}`,
        section: `Research section ${sectionIndex + 1}`,
      })),
    );

    const groups = groupCandidateLonglist(candidates);
    expect(candidates).toHaveLength(47);
    expect(groups).toHaveLength(8);
    expect(groups.map((group) => group.candidates.length)).toEqual(sectionSizes);
    expect(groups.every((group) => group.total === 8)).toBe(true);
  });

  it("groups by section and splits large sections into stable fragment paths", () => {
    const candidates = Array.from({ length: 3 }, (_, i) => ({
      url: `https://example.com/${i}`,
      section: "Terminal Judgment",
    }));

    expect(groupCandidateLonglist(candidates, 2)).toMatchObject([
      {
        index: 1,
        total: 2,
        section: "Terminal Judgment",
        fragmentPath: "working/evaluation-decisions/01-terminal-judgment-1.yaml",
        candidates: candidates.slice(0, 2),
      },
      {
        index: 2,
        total: 2,
        section: "Terminal Judgment",
        fragmentPath: "working/evaluation-decisions/02-terminal-judgment-2.yaml",
        candidates: candidates.slice(2),
      },
    ]);
  });

  it("treats invalid group sizes as one candidate per fragment", () => {
    const candidates = [
      { url: "https://example.com/a", section: "Foundations" },
      { url: "https://example.com/b", section: "Foundations" },
    ];

    expect(groupCandidateLonglist(candidates, 0).map((group) => group.candidates)).toEqual([
      [candidates[0]],
      [candidates[1]],
    ]);
  });
});
