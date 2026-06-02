// Read/write tape YAML, preserving user comments where possible.
//
// Uses the `yaml` package's Document API so block comments survive round-trips.

import { readFileSync, writeFileSync, existsSync, mkdirSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import * as YAML from "yaml";
import type {
  Tape,
  SourceSpec,
  SourceType,
  Priority,
  Mode,
  Render,
  SourceKind,
  JtbdClarification,
} from "./types.js";

export type ValidationError = { field: string; message: string };

const ALLOWED_TYPES: ReadonlySet<string> = new Set<SourceType>([
  "youtube",
  "web",
  "local_file",
]);
const ALLOWED_PRIORITIES: ReadonlySet<string> = new Set<Priority>(["required", "optional"]);
const ALLOWED_MODES: ReadonlySet<string> = new Set<Mode>(["quick", "methodology"]);
const ALLOWED_RENDER: ReadonlySet<string> = new Set<Render>(["server", "js"]);
const ALLOWED_KINDS: ReadonlySet<string> = new Set<SourceKind>([
  "reference",
  "principle",
  "prescription",
  "example",
]);
const ALLOWED_LOCAL_EXTENSIONS: ReadonlySet<string> = new Set([
  ".md",
  ".txt",
  ".html",
  ".htm",
  ".pdf",
]);

export type ProjectFolder = {
  path: string;
  tapePath: string;
  synthesisPath: string;
  mixtapePath: string;
  sourcesDir: string;
  workingDir: string;
};

export function projectFolder(path: string): ProjectFolder {
  return {
    path,
    tapePath: join(path, "tape.yaml"),
    synthesisPath: join(path, "synthesis.md"),
    mixtapePath: join(path, "MIXTAPE.md"),
    sourcesDir: join(path, "sources"),
    workingDir: join(path, "working"),
  };
}

export type SynthesisStatus = {
  exists: boolean;
  charCount: number;
  /** True if the file looks like the placeholder shipped by `liner init`. */
  isPlaceholder: boolean;
  /** True if synthesis exists and has meaningful (non-placeholder, non-empty) content. */
  isReady: boolean;
};

export function readSynthesisStatus(folder: ProjectFolder): SynthesisStatus {
  if (!existsSync(folder.synthesisPath)) {
    return { exists: false, charCount: 0, isPlaceholder: false, isReady: false };
  }
  const text = readFileSync(folder.synthesisPath, "utf8");
  const trimmed = text.trim();
  const isPlaceholder = trimmed.includes("Replace this placeholder");
  return {
    exists: true,
    charCount: trimmed.length,
    isPlaceholder,
    isReady: trimmed.length > 0 && !isPlaceholder,
  };
}

export function readTape(path: string): { tape: Tape; doc: YAML.Document } {
  const text = readFileSync(path, "utf8");
  const doc = YAML.parseDocument(text);
  const raw = doc.toJSON() as Record<string, unknown> | null;
  if (raw == null || typeof raw !== "object") {
    throw new Error(`${path}: file is empty or not a YAML mapping`);
  }
  const tape = normalizeTape(raw);
  return { tape, doc };
}

export function writeTape(path: string, tape: Tape, doc?: YAML.Document): void {
  const out = doc ?? YAML.parseDocument("");
  applyTapeToDocument(out, tape);
  ensureDir(dirname(path));
  writeFileSync(path, String(out), "utf8");
}

export function emptyTape(curator = ""): Tape {
  return {
    title: "New mixtape",
    description: "",
    version: 1,
    curator,
    sources: [],
    tags: [],
    mode: "quick",
    jtbd: null,
  };
}

// --- Validation -------------------------------------------------------------

export function validateTape(tape: Tape): ValidationError[] {
  const errs: ValidationError[] = [];
  if (!tape.title?.trim()) errs.push({ field: "title", message: "required" });
  if (!tape.description?.trim()) errs.push({ field: "description", message: "required" });
  if (!tape.curator?.trim()) errs.push({ field: "curator", message: "required" });
  if (tape.version !== 1) errs.push({ field: "version", message: "must be 1" });
  if (!tape.sources?.length) errs.push({ field: "sources", message: "at least one required" });
  if (tape.mode != null && !ALLOWED_MODES.has(tape.mode)) {
    errs.push({ field: "mode", message: "must be quick or methodology" });
  }
  tape.sources?.forEach((src, i) => {
    const prefix = `sources[${i}]`;
    if (!ALLOWED_TYPES.has(src.type)) {
      errs.push({ field: `${prefix}.type`, message: "must be youtube, web, or local_file" });
    }
    if (!ALLOWED_PRIORITIES.has(src.priority)) {
      errs.push({
        field: `${prefix}.priority`,
        message: "must be required or optional",
      });
    }
    if (src.type === "local_file") {
      if (!src.path?.trim()) {
        errs.push({ field: `${prefix}.path`, message: "required for local_file" });
      } else {
        const path = src.path.trim();
        if (path.startsWith("/")) {
          errs.push({ field: `${prefix}.path`, message: "must be relative, not absolute" });
        } else if (!path.startsWith("personal/")) {
          errs.push({ field: `${prefix}.path`, message: "must start with personal/" });
        } else if (path.split("/").includes("..")) {
          errs.push({ field: `${prefix}.path`, message: "must not contain .. segments" });
        } else {
          const ext = extOf(path);
          if (!ALLOWED_LOCAL_EXTENSIONS.has(ext)) {
            errs.push({
              field: `${prefix}.path`,
              message: `unsupported extension ${ext}; allowed: .md/.txt/.html/.htm/.pdf`,
            });
          }
        }
      }
      if (!src.citation?.trim()) {
        errs.push({ field: `${prefix}.citation`, message: "required for local_file" });
      }
    } else {
      // youtube / web
      if (!src.url?.trim()) {
        errs.push({ field: `${prefix}.url`, message: "required" });
      } else if (!isLikelyUrl(src.url)) {
        errs.push({ field: `${prefix}.url`, message: "not a valid URL" });
      }
      if (src.render != null) {
        if (src.type !== "web") {
          errs.push({ field: `${prefix}.render`, message: "only valid on web sources" });
        } else if (!ALLOWED_RENDER.has(src.render)) {
          errs.push({ field: `${prefix}.render`, message: "must be server or js" });
        }
      }
    }
    if (src.kind != null && !ALLOWED_KINDS.has(src.kind)) {
      errs.push({
        field: `${prefix}.kind`,
        message: "must be reference, principle, prescription, or example",
      });
    }
  });
  return errs;
}

function extOf(p: string): string {
  const dot = p.lastIndexOf(".");
  return dot < 0 ? "" : p.slice(dot).toLowerCase();
}

export function detectSourceType(input: string): SourceType {
  const trimmed = input.trim();
  if (!trimmed) return "web";
  const lower = trimmed.toLowerCase();
  // Path-like input → local_file
  if (lower.startsWith("personal/") || (!lower.includes("://") && lower.includes("/"))) {
    return "local_file";
  }
  if (
    lower.includes("youtube.com/") ||
    lower.includes("youtu.be/") ||
    lower.includes("youtube.com/shorts/")
  ) {
    return "youtube";
  }
  return "web";
}

/** Slugify a string for use as a folder name. Mirrors Python `project.slugify`. */
export function slugify(text: string, maxLength = 60): string {
  if (!text) return "untitled";
  const normalized = text
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  if (!normalized) return "untitled";
  const truncated = normalized.slice(0, maxLength).replace(/-+$/g, "");
  return truncated || "untitled";
}

export function isProjectFolder(path: string): boolean {
  try {
    return statSync(path).isDirectory() && existsSync(join(path, "tape.yaml"));
  } catch {
    return false;
  }
}

// --- Internals --------------------------------------------------------------

function normalizeTape(raw: Record<string, unknown>): Tape {
  const sourcesRaw = (raw["sources"] as unknown[] | undefined) ?? [];
  const sources: SourceSpec[] = sourcesRaw.map((s) => {
    const o = (s ?? {}) as Record<string, unknown>;
    return {
      type: (o["type"] as SourceType) ?? "web",
      url: String(o["url"] ?? ""),
      note: (o["note"] as string | null | undefined) ?? null,
      section: (o["section"] as string | null | undefined) ?? null,
      priority: (o["priority"] as Priority | undefined) ?? "required",
      render: (o["render"] as Render | null | undefined) ?? null,
      path: (o["path"] as string | null | undefined) ?? null,
      citation: (o["citation"] as string | null | undefined) ?? null,
      kind: (o["kind"] as SourceKind | null | undefined) ?? null,
    };
  });
  // jtbd_clarifications round-trips as a list of {question, answer} objects.
  // Permissive parse: skip entries that don't have both fields as strings.
  const clarRaw = raw["jtbd_clarifications"];
  let clarifications: JtbdClarification[] | null = null;
  if (Array.isArray(clarRaw)) {
    clarifications = [];
    for (const entry of clarRaw) {
      const o = (entry ?? {}) as Record<string, unknown>;
      const q = o["question"];
      const a = o["answer"];
      if (typeof q === "string" && typeof a === "string") {
        clarifications.push({ question: q, answer: a });
      }
    }
    if (clarifications.length === 0) clarifications = null;
  }
  return {
    title: String(raw["title"] ?? ""),
    description: String(raw["description"] ?? ""),
    version: Number(raw["version"] ?? 1),
    curator: String(raw["curator"] ?? ""),
    sources,
    tags: Array.isArray(raw["tags"]) ? (raw["tags"] as string[]) : [],
    created: (raw["created"] as string | null | undefined) ?? null,
    updated: (raw["updated"] as string | null | undefined) ?? null,
    license: (raw["license"] as string | null | undefined) ?? null,
    homepage: (raw["homepage"] as string | null | undefined) ?? null,
    mode: (raw["mode"] as Mode | null | undefined) ?? null,
    jtbd: (raw["jtbd"] as string | null | undefined) ?? null,
    jtbd_clarifications: clarifications,
    methodology_version: (raw["methodology_version"] as string | null | undefined) ?? null,
    parent: (raw["parent"] as string | null | undefined) ?? null,
  };
}

function applyTapeToDocument(doc: YAML.Document, tape: Tape): void {
  const today = new Date().toISOString().slice(0, 10);
  doc.set("title", tape.title);
  doc.set("description", tape.description);
  doc.set("version", tape.version);
  doc.set("curator", tape.curator);
  if (tape.mode) doc.set("mode", tape.mode);
  else doc.delete("mode");
  if (tape.jtbd && tape.jtbd.trim()) doc.set("jtbd", tape.jtbd.trim());
  else doc.delete("jtbd");
  // jtbd_clarifications — emit as a list of mappings so the YAML reads
  // naturally. Omit when empty/null so unaffected tapes stay clean.
  if (tape.jtbd_clarifications && tape.jtbd_clarifications.length > 0) {
    doc.set(
      "jtbd_clarifications",
      doc.createNode(
        tape.jtbd_clarifications.map((c) => ({
          question: c.question,
          answer: c.answer,
        })),
      ),
    );
  } else {
    doc.delete("jtbd_clarifications");
  }
  if (tape.methodology_version) doc.set("methodology_version", tape.methodology_version);
  // parent — written when present, omitted otherwise. Round-trips through
  // `liner replay` (sets parent on the clone) and through TUI edits.
  if (tape.parent && tape.parent.trim()) doc.set("parent", tape.parent.trim());
  else doc.delete("parent");
  if (tape.tags && tape.tags.length > 0) {
    doc.set("tags", tape.tags);
  } else {
    doc.delete("tags");
  }
  if (tape.created) doc.set("created", tape.created);
  if (tape.license) doc.set("license", tape.license);
  if (tape.homepage) doc.set("homepage", tape.homepage);
  doc.set("updated", today);

  const sourcesNode = doc.createNode(
    tape.sources.map((s) => {
      const obj: Record<string, unknown> = { type: s.type };
      if (s.type === "local_file") {
        if (s.path) obj["path"] = s.path;
        if (s.citation) obj["citation"] = s.citation;
      } else {
        if (s.url) obj["url"] = s.url;
        // Explicit `render` field is preserved (server or js). Absence is
        // significant — it means the auto-fallback default. We persist
        // whichever the curator chose, including `server` for library tapes.
        if (s.render) obj["render"] = s.render;
      }
      if (s.note) obj["note"] = s.note;
      if (s.section) obj["section"] = s.section;
      if (s.priority && s.priority !== "required") obj["priority"] = s.priority;
      if (s.kind) obj["kind"] = s.kind;
      return obj;
    }),
  );
  doc.set("sources", sourcesNode);
}

function isLikelyUrl(s: string): boolean {
  try {
    const u = new URL(s);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

function ensureDir(d: string): void {
  if (!existsSync(d)) mkdirSync(d, { recursive: true });
}
