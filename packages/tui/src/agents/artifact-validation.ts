import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import * as YAML from "yaml";
import { countCandidateUrlsFromLonglist, normalizeCandidateUrl, readCandidateLonglist } from "./candidate-longlist.js";
import type { PhaseId } from "../phases.js";

export type ArtifactValidationResult =
  | { ok: true }
  | { ok: false; message: string };

const DECISIONS = new Set(["kept", "trim", "dropped"]);
const SOURCE_TYPES = new Set(["web", "youtube", "local_file", "skill"]);
const PRIORITIES = new Set(["required", "optional"]);
const KINDS = new Set(["reference", "principle", "prescription", "example"]);
const JTBD_FITS = new Set(["direct", "bridge", "background"]);
const FETCH_STATUSES = new Set(["readable", "partial", "metadata_only", "unavailable"]);
const USABLE_FETCH_STATUSES = new Set(["readable", "partial"]);
const CONTENT_QUALITIES = new Set(["high", "medium", "low"]);
const USABLE_CONTENT_QUALITIES = new Set(["high", "medium"]);

type EvaluationValidationOptions = {
  expectedCandidateCount?: number;
};

type AssemblyValidationOptions = {
  currentTapePath?: string;
  evaluationPath?: string;
  sourceManifestPath?: string;
};

export function validatePhaseArtifact(
  project: string,
  phaseId: PhaseId,
): ArtifactValidationResult {
  switch (phaseId) {
    case "candidates":
      return validateCandidateLonglist(join(project, "working/02-candidate-longlist.md"));
    case "evaluation":
      return validateProjectEvaluation(project);
    case "quality": {
      const quality = validateQualityChecks(join(project, "working/04-quality-checks.md"));
      if (!quality.ok) return quality;
      const evaluation = validateProjectEvaluation(project);
      if (!evaluation.ok) {
        return {
          ok: false,
          message: `working/03-evaluation.yaml is invalid after quality checks: ${evaluation.message}`,
        };
      }
      return quality;
    }
    case "synthesis":
      return validateSynthesis(join(project, "synthesis.md"));
    case "assembly":
      return validateAssemblyDraft(join(project, "working/07-tape-draft.yaml"), {
        currentTapePath: join(project, "tape.yaml"),
        evaluationPath: join(project, "working/03-evaluation.yaml"),
        sourceManifestPath: join(project, "local-sources/sources-manifest.yaml"),
      });
    case "improvement":
      return validateImprovementDelta(join(project, ".liner-runs/improvement/delta.yaml"));
    default:
      return { ok: true };
  }
}

function validateProjectEvaluation(project: string): ArtifactValidationResult {
  return validateEvaluation(join(project, "working/03-evaluation.yaml"), {
    expectedCandidateCount: countCandidateUrlsFromLonglist(
      join(project, "working/02-candidate-longlist.md"),
    ),
  });
}

export function validateImprovementDelta(path: string): ArtifactValidationResult {
  const parsed = parseYaml(path, "improvement delta");
  if (!parsed.ok) return parsed;
  if (!isRecord(parsed.value)) {
    return { ok: false, message: ".liner-runs/improvement/delta.yaml is not a YAML mapping" };
  }
  if (stringField(parsed.value, "contract") !== "liner.improvement_delta" || parsed.value["version"] !== 1) {
    return { ok: false, message: ".liner-runs/improvement/delta.yaml has an unsupported contract or version" };
  }
  if (!stringField(parsed.value, "summary")) {
    return { ok: false, message: ".liner-runs/improvement/delta.yaml summary is required" };
  }
  const removals = parsed.value["removals"];
  const replacements = parsed.value["replacements"];
  if (!Array.isArray(removals) || removals.length !== 0 || !Array.isArray(replacements) || replacements.length !== 0) {
    return { ok: false, message: "Improve Corpus requires separate explicit curator intent for removals or replacements" };
  }
  const additions = parsed.value["additions"];
  if (!Array.isArray(additions) || additions.length === 0) {
    return { ok: false, message: ".liner-runs/improvement/delta.yaml must contain at least one focused addition" };
  }
  for (let i = 0; i < additions.length; i++) {
    const source = additions[i];
    if (!isRecord(source)) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}] is not a mapping` };
    }
    const type = stringField(source, "type");
    if (!SOURCE_TYPES.has(type)) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}].type is invalid: ${type || "(empty)"}` };
    }
    if ((type === "web" || type === "youtube") && !stringField(source, "url")) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}].url is required` };
    }
    if (type === "local_file" && !stringField(source, "path")) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}].path is required` };
    }
    if (type === "skill" && !stringField(source, "path") && !stringField(source, "url")) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}] needs path or url` };
    }
    const priority = stringField(source, "priority");
    const kind = stringField(source, "kind");
    if (!PRIORITIES.has(priority) || !KINDS.has(kind)) {
      return { ok: false, message: `.liner-runs/improvement/delta.yaml additions[${i}] needs valid priority and kind` };
    }
  }
  return { ok: true };
}

export function validateCandidateLonglist(path: string): ArtifactValidationResult {
  if (!existsSync(path)) {
    return { ok: false, message: `${path} is missing` };
  }
  const body = readFileSync(path, "utf8");
  if (/(^|\n)\s*(?:[-*]\s*)?TODO\s*(?:[—:-]|$)/i.test(body)) {
    return { ok: false, message: "working/02-candidate-longlist.md still contains TODO placeholders" };
  }
  if (body.includes("https://example.com/...")) {
    return { ok: false, message: "working/02-candidate-longlist.md still contains the example URL placeholder" };
  }
  const candidates = readCandidateLonglist(path);
  if (candidates.length === 0) {
    return { ok: false, message: "working/02-candidate-longlist.md has no URL candidates" };
  }
  return { ok: true };
}

export function validateEvaluation(
  path: string,
  options: EvaluationValidationOptions = {},
): ArtifactValidationResult {
  const parsed = parseYaml(path, "evaluation");
  if (!parsed.ok) return parsed;
  if (!isRecord(parsed.value)) {
    return { ok: false, message: "working/03-evaluation.yaml is not a YAML mapping" };
  }

  const candidates = parsed.value["candidates"];
  if (!Array.isArray(candidates) || candidates.length === 0) {
    return { ok: false, message: "working/03-evaluation.yaml has no candidates list" };
  }
  if (
    options.expectedCandidateCount != null &&
    candidates.length !== options.expectedCandidateCount
  ) {
    return {
      ok: false,
      message:
        `working/03-evaluation.yaml has ${candidates.length} candidates, ` +
        `but working/02-candidate-longlist.md has ${options.expectedCandidateCount} URL candidates`,
    };
  }

  for (let i = 0; i < candidates.length; i++) {
    const candidate = candidates[i];
    if (!isRecord(candidate)) {
      return { ok: false, message: `working/03-evaluation.yaml candidates[${i}] is not a mapping` };
    }
    const url = stringField(candidate, "url");
    if (!url) {
      return { ok: false, message: `working/03-evaluation.yaml candidates[${i}].url is required` };
    }
    const decision = stringField(candidate, "decision");
    if (!DECISIONS.has(decision)) {
      return {
        ok: false,
        message: `working/03-evaluation.yaml candidates[${i}].decision is invalid: ${decision || "(empty)"}`,
      };
    }

    if (decision === "kept" || decision === "trim") {
      const rating = numericField(candidate, "rating");
      if (rating == null || rating < 1 || rating > 5) {
        return {
          ok: false,
          message: `working/03-evaluation.yaml candidates[${i}].rating must be 1-5 for ${decision}`,
        };
      }
      const jtbdFit = stringField(candidate, "jtbd_fit");
      if (!JTBD_FITS.has(jtbdFit)) {
        return {
          ok: false,
          message: `working/03-evaluation.yaml candidates[${i}].jtbd_fit is invalid: ${jtbdFit || "(empty)"}`,
        };
      }
      if (!stringField(candidate, "section")) {
        return {
          ok: false,
          message: `working/03-evaluation.yaml candidates[${i}].section is required for ${decision}`,
        };
      }
      if (!stringField(candidate, "note")) {
        return {
          ok: false,
          message: `working/03-evaluation.yaml candidates[${i}].note is required for ${decision}`,
        };
      }
      const evidence = validateEvidenceFields(candidate, `working/03-evaluation.yaml candidates[${i}]`, decision);
      if (!evidence.ok) return evidence;
    }
  }

  return { ok: true };
}

export function validateQualityChecks(path: string): ArtifactValidationResult {
  if (!existsSync(path)) {
    return { ok: false, message: `${path} is missing` };
  }
  const body = readFileSync(path, "utf8");
  if (/(^|\n)\s*(?:[-*]\s*)?TODO\s*(?:[—:-]|$)/i.test(body)) {
    return { ok: false, message: "working/04-quality-checks.md still contains TODO placeholders" };
  }
  const required = [
    "## Test 0",
    "## Test 1",
    "## Test 2",
    "## Test 3",
    "## Test 4",
    "### Perspectives audit",
    "## Test 5",
    "Distribution:",
    "## Test 6",
    "## Test 7",
    "Source-role fit",
    "## Test 8",
    "Capability-pattern fit",
  ];
  for (const marker of required) {
    if (!body.includes(marker)) {
      return { ok: false, message: `working/04-quality-checks.md is missing ${marker}` };
    }
  }
  return { ok: true };
}

export function validateSynthesis(path: string): ArtifactValidationResult {
  if (!existsSync(path)) {
    return { ok: false, message: `${path} is missing` };
  }
  const body = readFileSync(path, "utf8");
  if (body.includes("Replace this placeholder") || /(^|\n)\s*(?:[-*]\s*)?TODO\s*(?:[—:-]|$)/i.test(body)) {
    return { ok: false, message: "synthesis.md still contains placeholder text" };
  }
  const required = ["## Generative rules", "## Stances this corpus takes"];
  for (const marker of required) {
    if (!body.includes(marker)) {
      return { ok: false, message: `synthesis.md is missing ${marker}` };
    }
  }
  return { ok: true };
}

export function validateAssemblyDraft(
  path: string,
  options: AssemblyValidationOptions = {},
): ArtifactValidationResult {
  const parsed = parseYaml(path, "assembly draft");
  if (!parsed.ok) return parsed;
  if (!isRecord(parsed.value)) {
    return { ok: false, message: "working/07-tape-draft.yaml is not a YAML mapping" };
  }

  const sources = parsed.value["sources"];
  if (!Array.isArray(sources) || sources.length === 0) {
    return { ok: false, message: "working/07-tape-draft.yaml has no sources list" };
  }

  const draftSources: Record<string, unknown>[] = [];
  for (let i = 0; i < sources.length; i++) {
    const source = sources[i];
    if (!isRecord(source)) {
      return { ok: false, message: `working/07-tape-draft.yaml sources[${i}] is not a mapping` };
    }
    draftSources.push(source);
    const type = stringField(source, "type") || "web";
    if (!SOURCE_TYPES.has(type)) {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}].type is invalid: ${type}`,
      };
    }
    if (type === "local_file") {
      if (!stringField(source, "path")) {
        return {
          ok: false,
          message: `working/07-tape-draft.yaml sources[${i}].path is required for local_file`,
        };
      }
    } else if (type === "skill") {
      if (!stringField(source, "path") && !stringField(source, "url")) {
        return {
          ok: false,
          message: `working/07-tape-draft.yaml sources[${i}].path or url is required for skill`,
        };
      }
    } else if (!stringField(source, "url")) {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}].url is required for ${type}`,
      };
    }
    const priority = stringField(source, "priority");
    if (!PRIORITIES.has(priority)) {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}].priority is invalid: ${priority || "(empty)"}`,
      };
    }
    const kind = stringField(source, "kind");
    if (!KINDS.has(kind)) {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}].kind is invalid: ${kind || "(empty)"}`,
      };
    }
    if (!stringField(source, "section")) {
      return { ok: false, message: `working/07-tape-draft.yaml sources[${i}].section is required` };
    }
    if (!stringField(source, "note")) {
      return { ok: false, message: `working/07-tape-draft.yaml sources[${i}].note is required` };
    }
  }

  if (options.currentTapePath) {
    const preservation = validateExistingCustomSources(draftSources, options.currentTapePath);
    if (!preservation.ok) return preservation;
  }
  const activeManifestKeys = options.sourceManifestPath
    ? activeManifestSourceKeys(options.sourceManifestPath)
    : new Set<string>();
  if (options.sourceManifestPath) {
    const preservation = validateActiveManifestSources(draftSources, options.sourceManifestPath);
    if (!preservation.ok) return preservation;
  }
  if (options.evaluationPath) {
    const evidence = validateDraftSourcesAgainstEvaluation(draftSources, options.evaluationPath, activeManifestKeys);
    if (!evidence.ok) return evidence;
  }

  return { ok: true };
}

function validateEvidenceFields(
  candidate: Record<string, unknown>,
  label: string,
  decision: string,
): ArtifactValidationResult {
  const fetchStatus = stringField(candidate, "fetch_status");
  if (!FETCH_STATUSES.has(fetchStatus)) {
    return {
      ok: false,
      message: `${label}.fetch_status must be readable, partial, metadata_only, or unavailable for ${decision}`,
    };
  }
  if (!USABLE_FETCH_STATUSES.has(fetchStatus)) {
    return {
      ok: false,
      message: `${label}.fetch_status cannot be ${fetchStatus} for ${decision}; unavailable or metadata-only candidates must be dropped`,
    };
  }

  const contentQuality = stringField(candidate, "content_quality");
  if (!CONTENT_QUALITIES.has(contentQuality)) {
    return {
      ok: false,
      message: `${label}.content_quality must be high, medium, or low for ${decision}`,
    };
  }
  if (!USABLE_CONTENT_QUALITIES.has(contentQuality)) {
    return {
      ok: false,
      message: `${label}.content_quality cannot be low for ${decision}; low-quality content must be dropped`,
    };
  }

  const evidence = stringListField(candidate, "evidence");
  if (evidence.length < 2) {
    return {
      ok: false,
      message: `${label}.evidence needs at least two content-specific bullets for ${decision}`,
    };
  }
  return { ok: true };
}

function validateDraftSourcesAgainstEvaluation(
  draftSources: Record<string, unknown>[],
  evaluationPath: string,
  protectedSourceKeys: Set<string> = new Set(),
): ArtifactValidationResult {
  const parsed = parseYaml(evaluationPath, "evaluation");
  if (!parsed.ok) return parsed;
  if (!isRecord(parsed.value)) {
    return { ok: false, message: "working/03-evaluation.yaml is not a YAML mapping for assembly evidence checks" };
  }
  const candidates = parsed.value["candidates"];
  if (!Array.isArray(candidates)) {
    return { ok: false, message: "working/03-evaluation.yaml has no candidates list for assembly evidence checks" };
  }

  const byUrl = new Map<string, Record<string, unknown>>();
  for (const candidate of candidates) {
    if (!isRecord(candidate)) continue;
    const url = stringField(candidate, "url");
    if (!url) continue;
    byUrl.set(normalizeCandidateUrl(url), candidate);
  }

  for (let i = 0; i < draftSources.length; i++) {
    const source = draftSources[i]!;
    const type = stringField(source, "type") || "web";
    if (type !== "web" && type !== "youtube") continue;
    const url = stringField(source, "url");
    const key = customSourceKey(source);
    if (key && protectedSourceKeys.has(key)) continue;
    const candidate = byUrl.get(normalizeCandidateUrl(url));
    if (!candidate) {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}] has no evaluation evidence for ${url}`,
      };
    }
    const decision = stringField(candidate, "decision");
    if (decision !== "kept" && decision !== "trim") {
      return {
        ok: false,
        message: `working/07-tape-draft.yaml sources[${i}] includes ${url}, but evaluation decision is ${decision || "(empty)"}`,
      };
    }
    const evidence = validateEvidenceFields(candidate, `working/03-evaluation.yaml candidate for ${url}`, decision);
    if (!evidence.ok) return evidence;
  }

  return { ok: true };
}

function validateActiveManifestSources(
  draftSources: Record<string, unknown>[],
  sourceManifestPath: string,
): ArtifactValidationResult {
  const manifest = activeManifestSources(sourceManifestPath);
  if (manifest.length === 0) return { ok: true };

  const draftKeys = new Set(
    draftSources
      .map(customSourceKey)
      .filter((key): key is string => key !== null),
  );
  const missing = manifest.filter((source) => {
    const key = customSourceKey(source);
    return key !== null && !draftKeys.has(key);
  });

  if (missing.length > 0) {
    return {
      ok: false,
      message: `working/07-tape-draft.yaml dropped active custom source: ${customSourceLabel(missing[0]!)}`,
    };
  }

  return { ok: true };
}

function activeManifestSourceKeys(sourceManifestPath: string): Set<string> {
  return new Set(
    activeManifestSources(sourceManifestPath)
      .map(customSourceKey)
      .filter((key): key is string => key !== null),
  );
}

function activeManifestSources(sourceManifestPath: string): Record<string, unknown>[] {
  if (!existsSync(sourceManifestPath)) return [];
  const parsed = parseYaml(sourceManifestPath, "source manifest");
  if (!parsed.ok || !isRecord(parsed.value)) return [];
  const items = parsed.value["sources"];
  if (!Array.isArray(items)) return [];

  const sources: Record<string, unknown>[] = [];
  for (const item of items) {
    if (!isRecord(item)) continue;
    if (item["active"] === false) continue;
    const source = item["source"];
    if (isRecord(source)) sources.push(source);
  }
  return sources;
}

function validateExistingCustomSources(
  draftSources: Record<string, unknown>[],
  currentTapePath: string,
): ArtifactValidationResult {
  const parsed = parseYaml(currentTapePath, "tape");
  if (!parsed.ok) {
    return {
      ok: false,
      message: `tape.yaml could not be parsed for custom-source preservation: ${parsed.message}`,
    };
  }
  if (!isRecord(parsed.value)) {
    return { ok: false, message: "tape.yaml is not a YAML mapping for custom-source preservation" };
  }
  const currentSources = parsed.value["sources"];
  if (!Array.isArray(currentSources)) return { ok: true };

  const currentCustomSources = currentSources
    .filter(isRecord)
    .filter((source) => {
      const type = stringField(source, "type");
      return type === "local_file" || type === "skill";
    });
  if (currentCustomSources.length === 0) return { ok: true };

  const draftKeys = new Set(
    draftSources
      .map(customSourceKey)
      .filter((key): key is string => key !== null),
  );
  const missing = currentCustomSources.filter((source) => {
    const key = customSourceKey(source);
    return key !== null && !draftKeys.has(key);
  });

  if (missing.length > 0) {
    return {
      ok: false,
      message: `working/07-tape-draft.yaml dropped existing custom source: ${customSourceLabel(missing[0]!)}`,
    };
  }

  return { ok: true };
}

function customSourceKey(source: Record<string, unknown>): string | null {
  const type = stringField(source, "type");
  if (type === "web" || type === "youtube") {
    const url = stringField(source, "url");
    if (!url) return null;
    return `${type}|${normalizeCandidateUrl(url)}`;
  }
  if (type === "local_file") {
    const sourcePath = stringField(source, "path");
    if (!sourcePath) return null;
    return `local_file|${sourcePath}|${stringField(source, "citation")}`;
  }
  if (type === "skill") {
    const sourcePath = stringField(source, "path");
    const url = stringField(source, "url");
    if (!sourcePath && !url) return null;
    return `skill|${sourcePath}|${url}`;
  }
  return null;
}

function customSourceLabel(source: Record<string, unknown>): string {
  const type = stringField(source, "type") || "unknown";
  return [type, stringField(source, "path") || stringField(source, "url") || stringField(source, "citation") || "(missing id)"]
    .join(" ");
}

function parseYaml(
  path: string,
  label: string,
): { ok: true; value: unknown } | { ok: false; message: string } {
  if (!existsSync(path)) {
    return { ok: false, message: `${path} is missing` };
  }
  try {
    return { ok: true, value: YAML.parse(readFileSync(path, "utf8")) };
  } catch (error) {
    return {
      ok: false,
      message: `${label} YAML could not be parsed: ${error instanceof Error ? error.message : String(error)}`,
    };
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function stringField(record: Record<string, unknown>, key: string): string {
  const value = record[key];
  return typeof value === "string" ? value.trim() : "";
}

function numericField(record: Record<string, unknown>, key: string): number | null {
  const value = record[key];
  if (typeof value === "number") return value;
  if (typeof value !== "string") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function stringListField(record: Record<string, unknown>, key: string): string[] {
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
