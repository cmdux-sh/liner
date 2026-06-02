// Permissive parsers for the Gate 1 / Gate 2 review screens.
//
// These read working/* artifacts produced by the agent (or the curator by hand)
// and produce structured data the TUI can render. They are deliberately lenient
// — the artifacts are markdown/yaml meant for human eyes, not a strict schema.

import { readFileSync, existsSync } from "node:fs";
import * as YAML from "yaml";

// ---------------------------------------------------------------------------
// Gate 1 — candidate long-list
// ---------------------------------------------------------------------------

export type LonglistCandidate = {
  /** Raw URL extracted from the bullet (first https?:// match). May be empty if the bullet has no URL. */
  url: string;
  /** Everything else on the bullet line, with the URL and any leading checkbox markers stripped. */
  label: string;
};

export type LonglistSection = {
  title: string;
  candidates: LonglistCandidate[];
};

/** Parse a candidate long-list markdown file produced in Phase 2. */
export function parseLonglist(path: string): LonglistSection[] {
  if (!existsSync(path)) return [];
  const text = readFileSync(path, "utf8");
  const sections: LonglistSection[] = [];
  let current: LonglistSection | null = null;

  for (const rawLine of text.split("\n")) {
    const line = rawLine.replace(/\r$/, "");
    // Section header (## ...)
    const headingMatch = line.match(/^##\s+(.+?)\s*$/);
    if (headingMatch) {
      // Drop the literal "Section:" prefix the template uses, so the displayed
      // header reads as just the section name.
      const title = (headingMatch[1] ?? "").replace(/^Section:\s*/i, "");
      current = { title, candidates: [] };
      sections.push(current);
      continue;
    }
    if (!current) continue;
    // Bullet (- or *) followed by an optional checkbox.
    const bulletMatch = line.match(/^\s*[-*]\s+(.*)$/);
    if (!bulletMatch) continue;
    let rest = (bulletMatch[1] ?? "").trim();
    rest = rest.replace(/^\[[ xX]\]\s*/, "");
    const urlMatch = rest.match(/https?:\/\/\S+/);
    const url = urlMatch ? urlMatch[0].replace(/[.,;)\]]+$/, "") : "";
    const label = url ? rest.replace(url, "").replace(/\s+/g, " ").trim() : rest;
    current.candidates.push({ url, label });
  }

  // Filter out empty sections (e.g. the intro paragraph before any heading).
  return sections.filter((s) => s.candidates.length > 0);
}

// ---------------------------------------------------------------------------
// Gate 2 — evaluation + quality checks
// ---------------------------------------------------------------------------

export type EvaluationDecision = "kept" | "trim" | "trimmed" | "dropped";

export type EvaluationCandidate = {
  url: string;
  title: string;
  decision: EvaluationDecision | string;
  rating: number | null;
  section: string;
  rationale: string;
  note: string;
};

export function parseEvaluation(path: string): EvaluationCandidate[] {
  if (!existsSync(path)) return [];
  let parsed: unknown;
  try {
    parsed = YAML.parse(readFileSync(path, "utf8"));
  } catch {
    return [];
  }
  if (!parsed || typeof parsed !== "object") return [];
  const raw = (parsed as Record<string, unknown>)["candidates"];
  if (!Array.isArray(raw)) return [];
  return raw.map((entry) => {
    const o = (entry ?? {}) as Record<string, unknown>;
    return {
      url: String(o["url"] ?? ""),
      title: String(o["title"] ?? ""),
      decision: String(o["decision"] ?? ""),
      rating: typeof o["rating"] === "number" ? (o["rating"] as number) : null,
      section: String(o["section"] ?? ""),
      rationale: String(o["rationale"] ?? ""),
      note: String(o["note"] ?? ""),
    };
  });
}

export type QualityCheck = {
  test: string;       // "Redundancy" / "Coverage" / "Disagreement" / "Framing-gap"
  finding: string;    // body text after the heading
};

export function parseQualityChecks(path: string): QualityCheck[] {
  if (!existsSync(path)) return [];
  const text = readFileSync(path, "utf8");
  const checks: QualityCheck[] = [];
  let currentTest: string | null = null;
  let buffer: string[] = [];

  function flush(): void {
    if (currentTest) {
      checks.push({ test: currentTest, finding: buffer.join("\n").trim() });
    }
    buffer = [];
  }

  for (const rawLine of text.split("\n")) {
    const line = rawLine.replace(/\r$/, "");
    // ## Redundancy test  → testName = "Redundancy"
    const heading = line.match(/^##\s+(.+?)\s+test\s*$/i);
    if (heading) {
      flush();
      currentTest = (heading[1] ?? "").trim();
      continue;
    }
    if (currentTest) buffer.push(line);
  }
  flush();
  return checks.filter((c) => c.finding.length > 0);
}

export type EvaluationGroup = {
  decision: "kept" | "trimmed" | "dropped";
  candidates: EvaluationCandidate[];
};

export function groupEvaluation(items: EvaluationCandidate[]): EvaluationGroup[] {
  const kept: EvaluationCandidate[] = [];
  const trimmed: EvaluationCandidate[] = [];
  const dropped: EvaluationCandidate[] = [];
  for (const c of items) {
    const d = String(c.decision || "").toLowerCase();
    if (d === "kept" || d === "keep") kept.push(c);
    else if (d === "trim" || d === "trimmed") trimmed.push(c);
    else dropped.push(c);
  }
  // Sort kept by descending rating so the strongest reads first.
  kept.sort((a, b) => (b.rating ?? 0) - (a.rating ?? 0));
  trimmed.sort((a, b) => (b.rating ?? 0) - (a.rating ?? 0));
  return [
    { decision: "kept", candidates: kept },
    { decision: "trimmed", candidates: trimmed },
    { decision: "dropped", candidates: dropped },
  ];
}
