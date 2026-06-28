import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import * as YAML from "yaml";
import {
  countCandidateUrlsFromLonglist,
  normalizeCandidateUrl,
  readCandidateLonglist,
  type LonglistCandidate,
} from "./candidate-longlist.js";
import { validateEvaluation } from "./artifact-validation.js";

export type EvaluationAssemblyResult =
  | { ok: true; path: string; count: number }
  | { ok: false; message: string };

export type EvaluationEnsureResult =
  | { ok: true; assembled: boolean; message?: string }
  | { ok: false; message: string };

export type EvaluationFragmentValidationResult =
  | { ok: true; count: number }
  | { ok: false; message: string };

type DecisionRecord = Record<string, unknown>;

const DECISIONS = new Set(["kept", "trim", "trimmed", "dropped"]);
const JTBD_FITS = new Set(["direct", "bridge", "background"]);
const FETCH_STATUSES = new Set(["readable", "partial", "metadata_only", "unavailable"]);
const USABLE_FETCH_STATUSES = new Set(["readable", "partial"]);
const CONTENT_QUALITIES = new Set(["high", "medium", "low"]);
const USABLE_CONTENT_QUALITIES = new Set(["high", "medium"]);

export function ensureEvaluationArtifact(project: string): EvaluationEnsureResult {
  const finalPath = join(project, "working/03-evaluation.yaml");
  const expectedCandidateCount = countCandidateUrlsFromLonglist(
    join(project, "working/02-candidate-longlist.md"),
  );
  const existing = validateEvaluation(finalPath, { expectedCandidateCount });
  if (existing.ok) return { ok: true, assembled: false };

  const assembled = assembleEvaluationFromFragments(project);
  if (!assembled.ok) {
    return {
      ok: false,
      message: `${existing.message}; fragment assembly failed: ${assembled.message}`,
    };
  }

  const validation = validateEvaluation(finalPath, { expectedCandidateCount });
  if (!validation.ok) {
    return {
      ok: false,
      message: `assembled working/03-evaluation.yaml is invalid: ${validation.message}`,
    };
  }
  return {
    ok: true,
    assembled: true,
    message: `Assembled working/03-evaluation.yaml with ${assembled.count} Evaluation decisions from fragments.`,
  };
}

export function assembleEvaluationFromFragments(project: string): EvaluationAssemblyResult {
  const longlistPath = join(project, "working/02-candidate-longlist.md");
  const candidates = readCandidateLonglist(longlistPath);
  if (candidates.length === 0) {
    return { ok: false, message: "working/02-candidate-longlist.md has no URL candidates" };
  }

  const fragments = readEvaluationFragments(join(project, "working/evaluation-decisions"));
  if (fragments.length === 0) {
    return { ok: false, message: "working/evaluation-decisions has no YAML decision fragments" };
  }

  const byUrl = new Map<string, DecisionRecord>();
  for (const fragment of fragments) {
    const url = stringField(fragment, "url");
    if (!url) return { ok: false, message: "a decision fragment is missing url" };
    const decision = normalizeDecision(stringField(fragment, "decision"));
    if (!DECISIONS.has(decision)) {
      return { ok: false, message: `decision fragment for ${url} has invalid decision: ${decision || "(empty)"}` };
    }
    byUrl.set(normalizeCandidateUrl(url), { ...fragment, decision });
  }

  const missing = candidates.filter((candidate) => !byUrl.has(candidate.url));
  if (missing.length > 0) {
    return {
      ok: false,
      message: `missing decision fragments for ${missing.length} candidates, first: ${missing[0]?.url}`,
    };
  }

  const assembled = candidates.map((candidate) => normalizeDecisionRecord(candidate, byUrl.get(candidate.url)!));
  const outputPath = join(project, "working/03-evaluation.yaml");
  mkdirSync(join(project, "working"), { recursive: true });
  writeFileSync(outputPath, YAML.stringify({ candidates: assembled }), "utf8");
  return { ok: true, path: outputPath, count: assembled.length };
}

export function validateEvaluationFragment(
  path: string,
  candidates: LonglistCandidate[],
): EvaluationFragmentValidationResult {
  const parsed = readEvaluationFragmentFile(path);
  if (!parsed.ok) return parsed;

  const expectedUrls = new Set(candidates.map((candidate) => candidate.url));
  const candidateByUrl = new Map(candidates.map((candidate) => [candidate.url, candidate]));
  const seen = new Set<string>();

  for (const [i, fragment] of parsed.records.entries()) {
    const rawUrl = stringField(fragment, "url");
    if (!rawUrl) return { ok: false, message: `fragment candidates[${i}].url is required` };
    const url = normalizeCandidateUrl(rawUrl);
    if (!expectedUrls.has(url)) {
      return { ok: false, message: `fragment includes unexpected URL: ${url}` };
    }
    if (seen.has(url)) {
      return { ok: false, message: `fragment includes duplicate URL: ${url}` };
    }
    seen.add(url);

    const decision = normalizeDecision(stringField(fragment, "decision"));
    if (!DECISIONS.has(decision)) {
      return {
        ok: false,
        message: `fragment candidate ${url} has invalid decision: ${decision || "(empty)"}`,
      };
    }
    if (decision === "kept" || decision === "trim") {
      const rating = numericField(fragment, "rating");
      if (rating == null || rating < 1 || rating > 5) {
        return { ok: false, message: `fragment candidate ${url} rating must be 1-5 for ${decision}` };
      }
      const jtbdFit = stringField(fragment, "jtbd_fit");
      if (!JTBD_FITS.has(jtbdFit)) {
        return { ok: false, message: `fragment candidate ${url} jtbd_fit is invalid: ${jtbdFit || "(empty)"}` };
      }
      if (!stringField(fragment, "section") && !candidateByUrl.get(url)?.section) {
        return { ok: false, message: `fragment candidate ${url} section is required for ${decision}` };
      }
      if (!stringField(fragment, "note")) {
        return { ok: false, message: `fragment candidate ${url} note is required for ${decision}` };
      }
      const evidence = validateFragmentEvidence(fragment, url, decision);
      if (!evidence.ok) return evidence;
    }
  }

  const missing = candidates.filter((candidate) => !seen.has(candidate.url));
  if (missing.length > 0) {
    return {
      ok: false,
      message: `fragment is missing ${missing.length} candidate(s), first: ${missing[0]?.url}`,
    };
  }

  return { ok: true, count: seen.size };
}

function readEvaluationFragments(dir: string): DecisionRecord[] {
  if (!existsSync(dir)) return [];
  const records: DecisionRecord[] = [];
  for (const name of readdirSync(dir).filter((file) => /\.ya?ml$/i.test(file)).sort()) {
    const path = join(dir, name);
    const parsed = readEvaluationFragmentFile(path);
    if (parsed.ok) records.push(...parsed.records);
  }
  return records;
}

function readEvaluationFragmentFile(
  path: string,
): { ok: true; records: DecisionRecord[] } | { ok: false; message: string } {
  if (!existsSync(path)) return { ok: false, message: `${path} is missing` };
  let parsed: unknown;
  try {
    parsed = YAML.parse(readFileSync(path, "utf8"));
  } catch (error) {
    return {
      ok: false,
      message: `fragment YAML could not be parsed: ${error instanceof Error ? error.message : String(error)}`,
    };
  }
  if (Array.isArray(parsed)) {
    return { ok: true, records: parsed.filter(isRecord) };
  }
  if (!isRecord(parsed)) return { ok: false, message: "fragment is not a YAML mapping or list" };
  const candidates = parsed["candidates"];
  if (Array.isArray(candidates)) {
    return { ok: true, records: candidates.filter(isRecord) };
  }
  if (stringField(parsed, "url")) {
    return { ok: true, records: [parsed] };
  }
  return { ok: false, message: "fragment has no candidates list" };
}

function normalizeDecisionRecord(
  candidate: { url: string; title?: string; section?: string },
  fragment: DecisionRecord,
): Record<string, unknown> {
  const decision = normalizeDecision(stringField(fragment, "decision"));
  const out: Record<string, unknown> = {
    url: candidate.url,
    title: stringField(fragment, "title") || candidate.title,
    decision,
    section: stringField(fragment, "section") || candidate.section,
  };
  const rationale = stringField(fragment, "rationale");
  if (rationale) out["rationale"] = rationale;
  if (decision === "kept" || decision === "trim") {
    const rating = numericField(fragment, "rating");
    if (rating != null) out["rating"] = rating;
    const jtbdFit = stringField(fragment, "jtbd_fit");
    if (jtbdFit) out["jtbd_fit"] = jtbdFit;
    const note = stringField(fragment, "note");
    if (note) out["note"] = note;
    const fetchStatus = stringField(fragment, "fetch_status");
    if (fetchStatus) out["fetch_status"] = fetchStatus;
    const contentQuality = stringField(fragment, "content_quality");
    if (contentQuality) out["content_quality"] = contentQuality;
    const evidence = stringListField(fragment, "evidence");
    if (evidence.length > 0) out["evidence"] = evidence;
  }
  return Object.fromEntries(Object.entries(out).filter(([, value]) => value != null && value !== ""));
}

function validateFragmentEvidence(
  fragment: DecisionRecord,
  url: string,
  decision: string,
): EvaluationFragmentValidationResult {
  const fetchStatus = stringField(fragment, "fetch_status");
  if (!FETCH_STATUSES.has(fetchStatus)) {
    return { ok: false, message: `fragment candidate ${url} fetch_status is required for ${decision}` };
  }
  if (!USABLE_FETCH_STATUSES.has(fetchStatus)) {
    return {
      ok: false,
      message: `fragment candidate ${url} fetch_status cannot be ${fetchStatus} for ${decision}`,
    };
  }
  const contentQuality = stringField(fragment, "content_quality");
  if (!CONTENT_QUALITIES.has(contentQuality)) {
    return { ok: false, message: `fragment candidate ${url} content_quality is required for ${decision}` };
  }
  if (!USABLE_CONTENT_QUALITIES.has(contentQuality)) {
    return {
      ok: false,
      message: `fragment candidate ${url} content_quality cannot be low for ${decision}`,
    };
  }
  const evidence = stringListField(fragment, "evidence");
  if (evidence.length < 2) {
    return {
      ok: false,
      message: `fragment candidate ${url} evidence needs at least two content-specific bullets for ${decision}`,
    };
  }
  return { ok: true, count: 1 };
}

function normalizeDecision(decision: string): string {
  return decision === "trimmed" ? "trim" : decision;
}

function isRecord(value: unknown): value is DecisionRecord {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function stringField(record: DecisionRecord, key: string): string {
  const value = record[key];
  return typeof value === "string" ? value.trim() : "";
}

function numericField(record: DecisionRecord, key: string): number | null {
  const value = record[key];
  if (typeof value === "number") return value;
  if (typeof value !== "string") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function stringListField(record: DecisionRecord, key: string): string[] {
  const value = record[key];
  if (!Array.isArray(value)) return [];
  return value
    .map(yamlListItemToString)
    .filter((item) => item !== "");
}

function yamlListItemToString(item: unknown): string {
  if (typeof item === "string") return item.trim();
  if (!isRecord(item)) return "";
  const entries = Object.entries(item);
  if (entries.length !== 1) return "";
  const [key, value] = entries[0]!;
  if (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean") {
    return "";
  }
  return `${key}: ${String(value)}`.trim();
}
