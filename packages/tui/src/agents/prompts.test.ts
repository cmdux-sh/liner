import { describe, expect, it } from "vitest";
import type { Tape } from "../types.js";
import { buildEvaluationSectionPrompt, buildPhasePrompt } from "./prompts.js";

const tape: Tape = {
  title: "Terminal craft",
  description: "",
  version: 1,
  curator: "Arturo",
  sources: [],
  mode: "quick",
  jtbd: "Help a design engineer build polished terminal interfaces.",
};

describe("buildPhasePrompt", () => {
  it("tells agents to treat mixtape folders as artifact folders, not repos", () => {
    const prompt = buildPhasePrompt({
      phaseId: "framing",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
    });

    expect(prompt).toContain("Workspace discipline");
    expect(prompt).toContain("not a git worktree");
    expect(prompt).toContain("Do not run `git diff`");
    expect(prompt).toContain("prefer Node.js or Python");
    expect(prompt).toContain("Ruby 2.6");
    expect(prompt).toContain("Array#tally");
  });

  it("frames Phase 1 as a capability brief derived from the user's AI-agent goal", () => {
    const prompt = buildPhasePrompt({
      phaseId: "framing",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
    });

    expect(prompt).toContain("AI-agent goal: Help a design engineer build polished terminal interfaces.");
    expect(prompt).toContain("Capability Brief");
    expect(prompt).toContain("research lanes");
    expect(prompt).toContain("Required source roles");
    expect(prompt).toContain("minimum kept/trim sources");
    expect(prompt).toContain("The user should not have to know research lanes");
    expect(prompt).toContain("Capability pattern: reference-translation");
    expect(prompt).toContain("runtime output contract");
  });

  it("bounds Phase 2 candidate discovery so custom-source runs write the long-list", () => {
    const prompt = buildPhasePrompt({
      phaseId: "candidates",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape: {
        ...tape,
        sources: [{ type: "web", url: "https://example.com/source", priority: "required" }],
      },
    });

    expect(prompt).toContain("Bounded search discipline");
    expect(prompt).toContain("seed the long-list with those URLs");
    expect(prompt).toContain("Count coverage by source roles and capability-pattern lanes");
    expect(prompt).toContain("continue past 30 when needed");
    expect(prompt).toContain("usually topping out around 40–60");
    expect(prompt).toContain("Replace the placeholder entirely");
    expect(prompt).toContain("Find sources for the capability, not just the topic");
    expect(prompt).toContain("Cover the **Required source roles**");
    expect(prompt).toContain("Role gaps");
    expect(prompt).toContain("status: improvement_recommended");
    expect(prompt).toContain("focused second pass");
    expect(prompt).toContain("source ecology");
    expect(prompt).toContain("input/reference domains outside the target output medium");
  });

  it("bounds Phase 4 fetch recovery so evaluation writes decisions under flaky network access", () => {
    const prompt = buildPhasePrompt({
      phaseId: "evaluation",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
    });

    expect(prompt).toContain("Bounded fetch discipline");
    expect(prompt).toContain("Chunked decision discipline");
    expect(prompt).toContain("working/evaluation-decisions/");
    expect(prompt).toContain("After two failed retrieval attempts total");
    expect(prompt).toContain("Every candidate still needs a decision in the YAML");
    expect(prompt).toContain("jtbd_fit");
    expect(prompt).toContain("direct | bridge | background");
    expect(prompt).toContain("source_role");
    expect(prompt).toContain("Source-role discipline");
    expect(prompt).toContain("Evidence contract");
    expect(prompt).toContain("fetch_status: readable");
    expect(prompt).toContain("content_quality: high");
    expect(prompt).toContain("Search snippets, titles, and model memory do not count");
    expect(prompt).toContain("reference-translation");
    expect(prompt).toContain("Target-medium implementation mechanics");
  });

  it("bounds Phase 5 quality checks so missing kinds do not trigger open-ended search", () => {
    const prompt = buildPhasePrompt({
      phaseId: "quality",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
    });

    expect(prompt).toContain("Bounded quality discipline");
    expect(prompt).toContain("Missing `kind` metadata is not itself a reason to backfill");
    expect(prompt).toContain("Search budget for Phase 5");
    expect(prompt).toContain("Do not exceed four external search/fetch attempts");
    expect(prompt).toContain("On resume after a paused/cancelled Quality run, do not continue a search loop");
    expect(prompt).toContain("Run the core-action test");
    expect(prompt).toContain("Core-action fit rules");
    expect(prompt).toContain("Note-quality");
    expect(prompt).toContain("repair it in `working/03-evaluation.yaml`");
    expect(prompt).toContain("Operating-fit audit");
    expect(prompt).toContain("Source-role fit rules");
    expect(prompt).toContain("Do not write \"light but not absent\"");
    expect(prompt).toContain("Capability-pattern fit rules");
    expect(prompt).toContain("## Test 8 — Capability-pattern fit");
    expect(prompt).toContain("working/05-operating-fit-audit.md");
    expect(prompt).toContain("do not write \"ready with limitation");
  });

  it("preserves custom local and skill sources during Assembly", () => {
    const prompt = buildPhasePrompt({
      phaseId: "assembly",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape: {
        ...tape,
        sources: [
          { type: "local_file", url: "", path: "local-sources/book.md", citation: "Book", priority: "required" },
          { type: "skill", url: "", path: "terminal-ui", priority: "required" },
        ],
      },
    });

    expect(prompt).toContain("web | youtube | local_file | skill");
    expect(prompt).toContain("Preserve every existing `local_file` and `skill` source");
    expect(prompt).toContain("Preserve every `active: true` source from `local-sources/sources-manifest.yaml`");
    expect(prompt).toContain("Do not silently convert `local_file` or `skill` sources into `web` sources");
    expect(prompt).toContain("Include them only when they already exist in tape.yaml or as active entries");
    expect(prompt).toContain("fetch_status: readable|partial");
    expect(prompt).toContain("content_quality: high|medium");
    expect(prompt).toContain("stop and repair `working/03-evaluation.yaml`");
  });

  it("requires operating sections in synthesis", () => {
    const prompt = buildPhasePrompt({
      phaseId: "synthesis",
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
    });

    expect(prompt).toContain("Required operating sections");
    expect(prompt).toContain("## Generative rules");
    expect(prompt).toContain("## Stances this corpus takes");
    expect(prompt).toContain("if you remember nothing else");
  });
});

describe("buildEvaluationSectionPrompt", () => {
  it("scopes Evaluation to one candidate group and one fragment", () => {
    const prompt = buildEvaluationSectionPrompt({
      project: "/tmp/terminal-craft",
      skillPath: "/tmp/skill",
      tape,
      group: {
        index: 2,
        total: 4,
        section: "Foundations",
        slug: "foundations",
        fragmentPath: "working/evaluation-decisions/02-foundations.yaml",
        candidates: [
          {
            url: "https://example.com/a",
            title: "Source A",
            section: "Foundations",
            reason: "Strong fit.",
          },
        ],
      },
    });

    expect(prompt).toContain("Evaluation chunk 2 of 4");
    expect(prompt).toContain("Do not run `git diff`");
    expect(prompt).toContain("working/evaluation-decisions/02-foundations.yaml");
    expect(prompt).toContain("Do not evaluate candidates from any other section");
    expect(prompt).toContain("Phase 2 reason: Strong fit.");
    expect(prompt).toContain("at least two content-specific evidence bullets");
  });
});
