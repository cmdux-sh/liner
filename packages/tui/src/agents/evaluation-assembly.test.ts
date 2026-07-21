import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import {
  assembleEvaluationFromFragments,
  ensureEvaluationArtifact,
  validateEvaluationFragment,
} from "./evaluation-assembly.js";

function projectFixture(): string {
  const project = mkdtempSync(join(tmpdir(), "liner-eval-assembly-"));
  mkdirSync(join(project, "working/evaluation-decisions"), { recursive: true });
  writeFileSync(
    join(project, "working/02-candidate-longlist.md"),
    [
      "## 1. Foundations",
      "- **Source A** - https://example.com/a",
      "  - Reason: Strong foundation.",
      "- **Source B** - https://example.com/b",
      "  - Reason: Too narrow.",
    ].join("\n"),
  );
  writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");
  return project;
}

describe("assembleEvaluationFromFragments", () => {
  it("assembles section fragments into the final evaluation YAML in longlist order", () => {
    const project = projectFixture();
    writeFileSync(
      join(project, "working/evaluation-decisions/01-foundations.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/b",
        "    decision: dropped",
        "    rationale: Too narrow for the JTBD.",
        "  - url: https://example.com/a",
        "    decision: trimmed",
        "    rating: '4'",
        "    jtbd_fit: bridge",
        "    fetch_status: readable",
        "    content_quality: medium",
        "    evidence:",
        "      - The source defines the model with a concrete example.",
        "      - It names a limitation: keeps the source selective.",
        "    rationale: Useful but not exhaustive.",
        "    note: |",
        "      Role: Foundation. Value: Establishes the model. Limitations: Narrow example set.",
      ].join("\n"),
    );

    const result = assembleEvaluationFromFragments(project);

    expect(result).toEqual({
      ok: true,
      path: join(project, "working/03-evaluation.yaml"),
      count: 2,
    });
    const output = readFileSync(join(project, "working/03-evaluation.yaml"), "utf8");
    expect(output.indexOf("https://example.com/a")).toBeLessThan(output.indexOf("https://example.com/b"));
    expect(output).toContain("title: Source A");
    expect(output).toContain("decision: trim");
    expect(output).toContain("jtbd_fit: bridge");
    expect(output).toContain("fetch_status: readable");
    expect(output).toContain("content_quality: medium");
    expect(output).toContain("The source defines the model");
    expect(output).toContain("It names a limitation: keeps the source selective");
    expect(output).toContain("section: Foundations");
  });

  it("fails when a longlist candidate has no fragment decision", () => {
    const project = projectFixture();
    writeFileSync(
      join(project, "working/evaluation-decisions/01-foundations.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: dropped",
      ].join("\n"),
    );

    const result = assembleEvaluationFromFragments(project);

    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("missing decision fragments");
  });

  it("matches clean fragment URLs to bold Markdown links from the longlist", () => {
    const project = projectFixture();
    writeFileSync(
      join(project, "working/02-candidate-longlist.md"),
      [
        "## 1. Evidence foundations",
        "1. **[PROV-DM: The PROV Data Model](https://www.w3.org/TR/prov-dm/)**  ",
      ].join("\n"),
    );
    writeFileSync(
      join(project, "working/evaluation-decisions/01-foundations.yaml"),
      [
        "candidates:",
        "  - url: https://www.w3.org/TR/prov-dm/",
        "    decision: dropped",
        "    rationale: Fixture decision.",
      ].join("\n"),
    );

    expect(assembleEvaluationFromFragments(project)).toEqual({
      ok: true,
      path: join(project, "working/03-evaluation.yaml"),
      count: 1,
    });
  });
});

describe("ensureEvaluationArtifact", () => {
  it("assembles fragments when the final evaluation file is still a placeholder", () => {
    const project = projectFixture();
    writeFileSync(
      join(project, "working/evaluation-decisions/01-foundations.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: readable",
        "    content_quality: high",
        "    evidence:",
        "      - The source shows the core action in a worked example.",
        "      - It names the quality bar the JTBD needs.",
        "    rationale: Direct fit.",
        "    note: |",
        "      Role: Anchor. Value: Strong principles. Limitations: Brief.",
        "  - url: https://example.com/b",
        "    decision: dropped",
        "    rationale: Duplicate.",
      ].join("\n"),
    );

    const result = ensureEvaluationArtifact(project);

    expect(result.ok).toBe(true);
    if (result.ok) expect(result.assembled).toBe(true);
    expect(readFileSync(join(project, "working/03-evaluation.yaml"), "utf8")).toContain("decision: kept");
  });
});

describe("validateEvaluationFragment", () => {
  it("accepts a complete section fragment and rejects partial fragments", () => {
    const project = projectFixture();
    const fragmentPath = join(project, "working/evaluation-decisions/01-foundations.yaml");
    writeFileSync(
      fragmentPath,
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: readable",
        "    content_quality: high",
        "    evidence:",
        "      - The source shows the core action in a worked example.",
        "      - It names the quality bar the JTBD needs.",
        "    note: |",
        "      Role: Anchor. Value: Strong fit. Limitations: Fixture.",
        "  - url: https://example.com/b",
        "    decision: dropped",
        "    rationale: Too narrow.",
      ].join("\n"),
    );

    expect(validateEvaluationFragment(fragmentPath, [
      { url: "https://example.com/a", section: "Foundations" },
      { url: "https://example.com/b", section: "Foundations" },
    ])).toEqual({ ok: true, count: 2 });

    const partialPath = join(project, "working/evaluation-decisions/02-foundations.yaml");
    writeFileSync(
      partialPath,
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
      ].join("\n"),
    );

    const result = validateEvaluationFragment(partialPath, [
      { url: "https://example.com/a", section: "Foundations" },
    ]);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("note is required");
  });
});
