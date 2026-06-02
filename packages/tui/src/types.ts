// Shared TS types that mirror the Python core's tape + event shapes.

export type SourceType = "youtube" | "web" | "local_file";
export type Priority = "required" | "optional";
export type Mode = "quick" | "methodology";
export type Render = "server" | "js";

/**
 * What role this source plays in the corpus. The synthesis prompt weights
 * sources differently by kind, and the consuming AI uses it to decide which
 * to deep-read. Optional — sources without a kind render unstyled.
 *
 *   reference   — specs / standards cited but not deep-read (POSIX, RFCs).
 *   principle   — read for stance / framing (essays, manifestos).
 *   prescription— read for concrete rules (style guides, conventions).
 *   example     — read for taste / illustration (case studies, tool lists).
 */
export type SourceKind = "reference" | "principle" | "prescription" | "example";

export type SourceSpec = {
  type: SourceType;
  /** Empty string for local_file sources; use `path` instead. */
  url: string;
  note?: string | null;
  section?: string | null;
  priority: Priority;
  /** web sources only: server (default, omitted) or js (requires linersh[js]). */
  render?: Render | null;
  /** local_file sources only: relative path under personal/. */
  path?: string | null;
  /** local_file sources only: human-readable provenance. */
  citation?: string | null;
  /** Role this source plays in the corpus — see SourceKind. */
  kind?: SourceKind | null;
};

/**
 * Single Q&A pair from the JTBD elicitation step. The wizard generates 3-4
 * targeted questions after the user types their raw JTBD and stores the
 * answers here so Phase 1 (framing) can weight the knowledge map by the
 * scope splits / quality anchors / audience the user named.
 */
export type JtbdClarification = { question: string; answer: string };

export type Tape = {
  title: string;
  description: string;
  version: number;
  curator: string;
  sources: SourceSpec[];
  tags?: string[];
  created?: string | null;
  updated?: string | null;
  license?: string | null;
  homepage?: string | null;
  mode?: Mode | null;
  jtbd?: string | null;
  /** Sharpening Q&A captured by the wizard's JTBD-clarify step. */
  jtbd_clarifications?: JtbdClarification[] | null;
  methodology_version?: string | null;
  /**
   * When this tape was created by `liner replay <other>`, the path of the
   * source folder it was cloned from. Records the v1→v2 lineage for replays.
   */
  parent?: string | null;
};

/** A mixtape project folder as reported by `liner list --json`. */
export type ProjectSummary = {
  path: string;
  name: string;
  title: string;
  description: string;
  curator: string;
  mode: Mode | null;
  jtbd: string | null;
  tags: string[];
  source_count: number;
  modified_iso: string;
};

// --- Event stream from `liner compile --emit-events` ------------------------

export type StartEvent = { type: "start"; total: number };
export type SourceStartEvent = { type: "source_start"; spec: SourceSpec };
export type SourceDoneEvent = {
  type: "source_done" | "source_cached";
  url: string;
  title: string | null;
  author: string | null;
  published_at: string | null;
  duration_seconds: number | null;
  body_chars: number;
  body_preview: string;
  metadata: Record<string, unknown>;
};
export type SourceFailedEvent = {
  type: "source_failed";
  url: string;
  message: string;
  severity: "warning" | "error";
};
export type FinishEvent = { type: "finish" };
export type ResultEvent = { type: "result"; payload: CompileResultPayload };

export type CompileEvent =
  | StartEvent
  | SourceStartEvent
  | SourceDoneEvent
  | SourceFailedEvent
  | FinishEvent
  | ResultEvent;

export type CompiledSourceRecord = {
  index: number;
  filename: string;
  path: string;
  url: string;
  type: SourceType;
  section: string | null;
  title: string | null;
  succeeded: boolean;
};

export type CompileResultPayload = {
  tape: Record<string, unknown> & {
    title?: string;
    description?: string;
    curator?: string;
    version?: number;
    mode?: Mode | null;
    jtbd?: string | null;
  };
  compiled_at: string;
  folder: string;
  mixtape_path: string;
  sources: CompiledSourceRecord[];
  warnings: Array<{ url: string; message: string; severity: "warning" | "error" }>;
  summary: { total: number; succeeded: number; failed: number };
};
