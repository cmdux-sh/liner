import { mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { describe, expect, it } from "vitest";
import { countCandidateUrlsFromLonglist } from "./candidate-longlist.js";
import {
  validateAssemblyDraft,
  validateCandidateLonglist,
  validateEvaluation,
  validatePhaseArtifact,
  validateQualityChecks,
  validateSynthesis,
} from "./artifact-validation.js";

const TMP = mkdtempSync(join(tmpdir(), "liner-artifact-validation-"));

function write(path: string, contents: string): string {
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, contents, "utf8");
  return path;
}

describe("validateEvaluation", () => {
  it("accepts kept, trim, and dropped candidates with the expected required fields", () => {
    const path = write(
      join(TMP, "evaluation-valid.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: readable",
        "    content_quality: high",
        "    evidence:",
        "      - The article names a concrete decision workflow.",
        "      - The source includes a worked example: shows tradeoffs.",
        "    section: Foundations",
        "    note: Useful.",
        "  - url: https://example.com/b",
        "    decision: trim",
        "    rating: '3'",
        "    jtbd_fit: bridge",
        "    fetch_status: partial",
        "    content_quality: medium",
        "    evidence:",
        "      - The transcript excerpt defines the supporting concept.",
        "      - The source names a limitation relevant to this JTBD.",
        "    section: Support",
        "    note: Optional.",
        "  - url: https://example.com/c",
        "    decision: dropped",
        "    rationale: Not relevant.",
      ].join("\n"),
    );

    expect(validateEvaluation(path)).toEqual({ ok: true });
  });

  it("rejects malformed YAML", () => {
    const path = write(
      join(TMP, "evaluation-invalid-yaml.yaml"),
      "candidates:\n  - title: Wizards: Definition and Design Recommendations\n",
    );

    const result = validateEvaluation(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("could not be parsed");
  });

  it("rejects kept candidates without a curator note", () => {
    const path = write(
      join(TMP, "evaluation-missing-note.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    section: Foundations",
      ].join("\n"),
    );

    const result = validateEvaluation(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("note is required");
  });

  it("rejects kept candidates without content evidence", () => {
    const path = write(
      join(TMP, "evaluation-missing-evidence.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: readable",
        "    content_quality: high",
        "    section: Foundations",
        "    note: Useful.",
      ].join("\n"),
    );

    const result = validateEvaluation(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("evidence");
  });

  it("rejects kept metadata-only or low-quality candidates", () => {
    const metadataOnly = write(
      join(TMP, "evaluation-metadata-only.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: metadata_only",
        "    content_quality: high",
        "    evidence:",
        "      - Search result says it might be useful.",
        "      - The title looks relevant.",
        "    section: Foundations",
        "    note: Useful.",
      ].join("\n"),
    );
    const metadataResult = validateEvaluation(metadataOnly);
    expect(metadataResult.ok).toBe(false);
    if (!metadataResult.ok) expect(metadataResult.message).toContain("metadata_only");

    const lowQuality = write(
      join(TMP, "evaluation-low-quality.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/b",
        "    decision: trim",
        "    rating: 3",
        "    jtbd_fit: bridge",
        "    fetch_status: readable",
        "    content_quality: low",
        "    evidence:",
        "      - The article repeats generic advice.",
        "      - The only example is a shallow listicle item.",
        "    section: Support",
        "    note: Optional.",
      ].join("\n"),
    );
    const lowQualityResult = validateEvaluation(lowQuality);
    expect(lowQualityResult.ok).toBe(false);
    if (!lowQualityResult.ok) expect(lowQualityResult.message).toContain("content_quality");
  });

  it("rejects kept candidates without a JTBD-fit label", () => {
    const path = write(
      join(TMP, "evaluation-missing-jtbd-fit.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    section: Foundations",
        "    note: Useful.",
      ].join("\n"),
    );

    const result = validateEvaluation(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("jtbd_fit");
  });
});

describe("countCandidateUrlsFromLonglist", () => {
  it("counts unique URL candidates and ignores trailing punctuation", () => {
    const path = write(
      join(TMP, "candidate-longlist.md"),
      [
        "## foundations",
        "- https://example.com/a — A",
        "- https://example.com/b). — B",
        "- https://example.com/a — duplicate",
      ].join("\n"),
    );

    expect(countCandidateUrlsFromLonglist(path)).toBe(2);
  });
});

describe("validateQualityChecks", () => {
  it("accepts a completed quality report with all required audit sections", () => {
    const path = write(
      join(TMP, "quality-valid.md"),
      [
        "# Quality checks",
        "",
        "## Test 0 — Core-action fit",
        "Distribution: 3 direct / 2 bridge / 1 background",
        "",
        "## Test 1 — Redundancy",
        "No material overlap.",
        "",
        "## Test 2 — Coverage",
        "Every knowledge-map section has kept sources.",
        "",
        "## Test 3 — Disagreement",
        "Counterpoint is included.",
        "",
        "## Test 4 — Framing-gap",
        "",
        "### Perspectives audit",
        "- Accessibility-first viewpoint — stance-represented (Source 3, verified).",
        "",
        "## Test 5 — Source-kind balance",
        "",
        "Distribution: 4 reference / 12 principle / 10 prescription / 5 example",
        "",
        "## Test 6 — Note-quality",
        "",
        "Checked: 31 kept/trim notes",
        "Repaired: 0 notes",
        "",
        "## Test 7 — Source-role fit",
        "",
        "Required roles:",
        "- Core-action method — minimum: 2; current: 3; status: pass; evidence: Source A, Source B, Source C",
        "",
        "## Test 8 — Capability-pattern fit",
        "",
        "Pattern: none",
      ].join("\n"),
    );

    expect(validateQualityChecks(path)).toEqual({ ok: true });
  });

  it("rejects placeholder quality reports", () => {
    const path = write(
      join(TMP, "quality-placeholder.md"),
      [
        "# Quality checks",
        "",
        "## Redundancy test",
        "",
        "TODO — any two sources making essentially the same point?",
      ].join("\n"),
    );

    const result = validateQualityChecks(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("TODO");
  });

  it("does not reject ordinary todo-domain prose as a placeholder", () => {
    const path = write(
      join(TMP, "quality-todo-prose.md"),
      [
        "# Quality checks",
        "",
        "## Test 0 — Core-action fit",
        "Distribution: 2 direct / 1 bridge / 0 background",
        "",
        "## Test 1 — Redundancy",
        "One example source is a todo app tutorial, but it is not duplicate.",
        "",
        "## Test 2 — Coverage",
        "Every section is represented.",
        "",
        "## Test 3 — Disagreement",
        "Tradeoffs are named.",
        "",
        "## Test 4 — Framing-gap",
        "",
        "### Perspectives audit",
        "- Beginner viewpoint — concerns covered by Source 2.",
        "",
        "## Test 5 — Source-kind balance",
        "",
        "Distribution: 2 reference / 3 principle / 4 prescription / 1 example",
        "",
        "## Test 6 — Note-quality",
        "",
        "Checked: 10 kept/trim notes",
        "Repaired: 0 notes",
        "",
        "## Test 7 — Source-role fit",
        "",
        "Required roles:",
        "- Worked example — minimum: 2; current: 2; status: pass; evidence: Source A, Source B",
        "",
        "## Test 8 — Capability-pattern fit",
        "",
        "Pattern: none",
      ].join("\n"),
    );

    expect(validateQualityChecks(path)).toEqual({ ok: true });
  });

  it("rejects quality reports that skip the note-quality check", () => {
    const path = write(
      join(TMP, "quality-missing-note-quality.md"),
      [
        "# Quality checks",
        "",
        "## Test 0 — Core-action fit",
        "Distribution: 2 direct / 1 bridge / 0 background",
        "",
        "## Test 1 — Redundancy",
        "No material overlap.",
        "",
        "## Test 2 — Coverage",
        "Every section is represented.",
        "",
        "## Test 3 — Disagreement",
        "Tradeoffs are named.",
        "",
        "## Test 4 — Framing-gap",
        "",
        "### Perspectives audit",
        "- Beginner viewpoint — concerns covered by Source 2.",
        "",
        "## Test 5 — Source-kind balance",
        "",
        "Distribution: 2 reference / 3 principle / 4 prescription / 1 example",
      ].join("\n"),
    );

    const result = validateQualityChecks(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("## Test 6");
  });

  it("rejects quality reports that skip the source-role fit check", () => {
    const path = write(
      join(TMP, "quality-missing-source-role-fit.md"),
      [
        "# Quality checks",
        "",
        "## Test 0 — Core-action fit",
        "Distribution: 2 direct / 1 bridge / 0 background",
        "",
        "## Test 1 — Redundancy",
        "No material overlap.",
        "",
        "## Test 2 — Coverage",
        "Every section is represented.",
        "",
        "## Test 3 — Disagreement",
        "Tradeoffs are named.",
        "",
        "## Test 4 — Framing-gap",
        "",
        "### Perspectives audit",
        "- Beginner viewpoint — concerns covered by Source 2.",
        "",
        "## Test 5 — Source-kind balance",
        "",
        "Distribution: 2 reference / 3 principle / 4 prescription / 1 example",
        "",
        "## Test 6 — Note-quality",
        "",
        "Checked: 10 kept/trim notes",
        "Repaired: 0 notes",
      ].join("\n"),
    );

    const result = validateQualityChecks(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("## Test 7");
  });

  it("rejects quality reports that skip the capability-pattern fit check", () => {
    const path = write(
      join(TMP, "quality-missing-capability-pattern-fit.md"),
      [
        "# Quality checks",
        "",
        "## Test 0 — Core-action fit",
        "Distribution: 2 direct / 1 bridge / 0 background",
        "",
        "## Test 1 — Redundancy",
        "No material overlap.",
        "",
        "## Test 2 — Coverage",
        "Every section is represented.",
        "",
        "## Test 3 — Disagreement",
        "Tradeoffs are named.",
        "",
        "## Test 4 — Framing-gap",
        "",
        "### Perspectives audit",
        "- Beginner viewpoint — concerns covered by Source 2.",
        "",
        "## Test 5 — Source-kind balance",
        "",
        "Distribution: 2 reference / 3 principle / 4 prescription / 1 example",
        "",
        "## Test 6 — Note-quality",
        "",
        "Checked: 10 kept/trim notes",
        "Repaired: 0 notes",
        "",
        "## Test 7 — Source-role fit",
        "",
        "Required roles:",
        "- Worked example — minimum: 2; current: 2; status: pass; evidence: Source A, Source B",
      ].join("\n"),
    );

    const result = validateQualityChecks(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("## Test 8");
  });
});

describe("validateSynthesis", () => {
  it("accepts synthesis with required operating sections", () => {
    const path = write(
      join(TMP, "synthesis-valid.md"),
      [
        "# Terminal craft synthesis",
        "",
        "The corpus frames terminal interfaces as repeated work surfaces.",
        "",
        "## Generative rules",
        "",
        "- Prefer one obvious next action.",
        "- Keep machine output separate from human guidance.",
        "",
        "## Stances this corpus takes",
        "",
        "- Dense is good when structure is honest.",
      ].join("\n"),
    );

    expect(validateSynthesis(path)).toEqual({ ok: true });
  });

  it("rejects synthesis without generative rules", () => {
    const path = write(
      join(TMP, "synthesis-missing-rules.md"),
      [
        "# Terminal craft synthesis",
        "",
        "The corpus frames terminal interfaces as repeated work surfaces.",
        "",
        "## Stances this corpus takes",
        "",
        "- Dense is good when structure is honest.",
      ].join("\n"),
    );

    const result = validateSynthesis(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("## Generative rules");
  });

  it("rejects placeholder synthesis", () => {
    const path = write(
      join(TMP, "synthesis-placeholder.md"),
      "# Synthesis\n\nReplace this placeholder with the curator's distilled view.\n",
    );

    const result = validateSynthesis(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("placeholder");
  });
});

describe("validateAssemblyDraft", () => {
  it("accepts web, local_file, and skill draft sources with kind metadata", () => {
    const path = write(
      join(TMP, "draft-valid.yaml"),
      [
        "sources:",
        "  - type: web",
        "    url: https://example.com/a",
        "    section: Foundations",
        "    priority: required",
        "    kind: principle",
        "    note: Useful.",
        "  - type: local_file",
        "    path: local-sources/book.md",
        "    section: Local",
        "    priority: optional",
        "    kind: reference",
        "    note: Book notes.",
        "  - type: skill",
        "    path: terminal-ui",
        "    section: Skill boundaries",
        "    priority: required",
        "    kind: reference",
        "    note: Imported as reference material, not active instructions.",
      ].join("\n"),
    );

    expect(validateAssemblyDraft(path)).toEqual({ ok: true });
  });

  it("rejects drafts that drop existing local and skill sources", () => {
    const project = join(TMP, "project-with-custom-sources");
    write(
      join(project, "tape.yaml"),
      [
        "title: Custom",
        "sources:",
        "  - type: local_file",
        "    path: local-sources/book.md",
        "    citation: Local book",
        "    priority: required",
        "  - type: skill",
        "    path: terminal-ui",
        "    priority: required",
      ].join("\n"),
    );
    write(
      join(project, "working/07-tape-draft.yaml"),
      [
        "sources:",
        "  - type: web",
        "    url: https://example.com/a",
        "    section: Foundations",
        "    priority: required",
        "    kind: principle",
        "    note: Useful.",
      ].join("\n"),
    );

    const result = validatePhaseArtifact(project, "assembly");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("dropped existing custom source");
  });

  it("rejects drafts that drop active custom sources from the local source manifest", () => {
    const project = join(TMP, "project-with-active-manifest-source");
    write(join(project, "tape.yaml"), "sources: []\n");
    write(join(project, "working/03-evaluation.yaml"), "candidates: []\n");
    write(
      join(project, "local-sources/sources-manifest.yaml"),
      [
        "sources:",
        "  - active: true",
        "    source:",
        "      type: local_file",
        "      path: local-sources/recovered/video.md",
        "      citation: Recovered video",
        "      priority: required",
      ].join("\n"),
    );
    write(
      join(project, "working/07-tape-draft.yaml"),
      [
        "sources:",
        "  - type: web",
        "    url: https://example.com/a",
        "    section: Foundations",
        "    priority: required",
        "    kind: principle",
        "    note: Useful.",
      ].join("\n"),
    );

    const result = validatePhaseArtifact(project, "assembly");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("dropped active custom source");
  });

  it("allows active custom URL sources from the manifest without evaluation evidence", () => {
    const project = join(TMP, "project-with-active-custom-url");
    write(join(project, "tape.yaml"), "sources: []\n");
    write(join(project, "working/03-evaluation.yaml"), "candidates: []\n");
    write(
      join(project, "local-sources/sources-manifest.yaml"),
      [
        "sources:",
        "  - active: true",
        "    source:",
        "      type: youtube",
        "      url: https://www.youtube.com/watch?v=customsource",
        "      priority: required",
        "      kind: principle",
      ].join("\n"),
    );
    write(
      join(project, "working/07-tape-draft.yaml"),
      [
        "sources:",
        "  - type: youtube",
        "    url: https://www.youtube.com/watch?v=customsource",
        "    section: Custom sources",
        "    priority: required",
        "    kind: principle",
        "    note: Curator-selected source.",
      ].join("\n"),
    );

    expect(validatePhaseArtifact(project, "assembly")).toEqual({ ok: true });
  });

  it("rejects draft sources without a kind", () => {
    const path = write(
      join(TMP, "draft-missing-kind.yaml"),
      [
        "sources:",
        "  - type: web",
        "    url: https://example.com/a",
        "    section: Foundations",
        "    priority: required",
        "    note: Useful.",
      ].join("\n"),
    );

    const result = validateAssemblyDraft(path);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("kind is invalid");
  });

  it("rejects draft URL sources that lack usable evaluation evidence", () => {
    const project = join(TMP, "project-with-unevidenced-draft");
    write(
      join(project, "working/03-evaluation.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: metadata_only",
        "    content_quality: high",
        "    evidence:",
        "      - Search result says it might be useful.",
        "      - The title looks relevant.",
        "    section: Foundations",
        "    note: Useful.",
      ].join("\n"),
    );
    write(
      join(project, "working/07-tape-draft.yaml"),
      [
        "sources:",
        "  - type: web",
        "    url: https://example.com/a",
        "    section: Foundations",
        "    priority: required",
        "    kind: principle",
        "    note: Useful.",
      ].join("\n"),
    );
    write(join(project, "tape.yaml"), "sources: []\n");

    const result = validatePhaseArtifact(project, "assembly");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("metadata_only");
  });
});

describe("validatePhaseArtifact", () => {
  it("rejects placeholder candidate long-lists", () => {
    const project = join(TMP, "project-with-placeholder-candidates");
    write(
      join(project, "working/02-candidate-longlist.md"),
      [
        "# Candidate long-list",
        "",
        "## Section: foundations",
        "",
        "- [ ] https://example.com/...",
      ].join("\n"),
    );

    const result = validatePhaseArtifact(project, "candidates");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("example URL placeholder");
  });

  it("accepts candidate long-lists with real URL candidates", () => {
    const path = write(
      join(TMP, "candidate-longlist.md"),
      [
        "# Candidate long-list",
        "",
        "## Section: foundations",
        "",
        "- https://example.com/source-a — Source A — Useful for the JTBD.",
      ].join("\n"),
    );

    expect(validateCandidateLonglist(path)).toEqual({ ok: true });
  });

  it("routes non-structured phases to success", () => {
    expect(validatePhaseArtifact(TMP, "framing")).toEqual({ ok: true });
  });

  it("routes synthesis through the synthesis validator", () => {
    const project = join(TMP, "project-with-placeholder-synthesis");
    write(
      join(project, "synthesis.md"),
      "# Synthesis\n\nReplace this placeholder with the curator's distilled view.\n",
    );

    const result = validatePhaseArtifact(project, "synthesis");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("placeholder");
  });

  it("routes quality through the quality report validator", () => {
    const project = join(TMP, "project-with-placeholder-quality");
    write(
      join(project, "working/04-quality-checks.md"),
      [
        "# Quality checks",
        "",
        "## Redundancy test",
        "TODO — placeholder.",
      ].join("\n"),
    );

    const result = validatePhaseArtifact(project, "quality");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("TODO");
  });

  it("requires one evaluation decision per longlist URL", () => {
    const project = join(TMP, "project-with-longlist");
    write(
      join(project, "working/02-candidate-longlist.md"),
      [
        "## foundations",
        "- https://example.com/a — A",
        "- https://example.com/b — B",
      ].join("\n"),
    );
    write(
      join(project, "working/03-evaluation.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: dropped",
        "    rationale: Too narrow.",
      ].join("\n"),
    );

    const result = validatePhaseArtifact(project, "evaluation");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("has 1 candidates");
  });
});
