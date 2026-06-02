import type { AgentEvent } from "./events.js";
import type { AgentId } from "./types.js";

/**
 * Parse a single newline-delimited JSON line into zero or more normalized events.
 *
 * Both Claude's stream-json and Codex's --json emit one JSON object per line.
 * Each object can produce multiple AgentEvents (an assistant message with
 * mixed text + tool_use content blocks becomes one `text` + one or more
 * `tool_start` events).
 */
export function parseLine(agent: AgentId, line: string): AgentEvent[] {
  const trimmed = line.trim();
  if (!trimmed) return [];
  let obj: unknown;
  try {
    obj = JSON.parse(trimmed);
  } catch {
    return [{ kind: "raw", text: trimmed }];
  }
  if (agent === "claude") return parseClaudeObject(obj);
  if (agent === "codex") return parseCodexObject(obj);
  return [{ kind: "raw", text: trimmed }];
}

// ---------------------------------------------------------------------------
// Claude stream-json
// ---------------------------------------------------------------------------
//
// Observed shapes (from `claude -p --output-format stream-json --verbose`):
//
//   { type: "system", subtype: "init", model, session_id, tools, ... }
//   { type: "assistant", message: { content: [{type:"text"|"tool_use",...}], ... } }
//   { type: "user", message: { content: [{type:"tool_result", tool_use_id, content, is_error}] } }
//   { type: "rate_limit_event", rate_limit_info: { status, ... } }
//   { type: "result", subtype: "success"|"error", duration_ms, num_turns, result, total_cost_usd, is_error }

function parseClaudeObject(obj: unknown): AgentEvent[] {
  if (!obj || typeof obj !== "object") return [];
  const o = obj as Record<string, unknown>;
  const type = String(o["type"] ?? "");

  if (type === "system" && o["subtype"] === "init") {
    return [
      {
        kind: "init",
        model: typeof o["model"] === "string" ? (o["model"] as string) : undefined,
        sessionId:
          typeof o["session_id"] === "string"
            ? (o["session_id"] as string)
            : undefined,
      },
    ];
  }

  if (type === "assistant") {
    const message = (o["message"] ?? {}) as Record<string, unknown>;
    const content = Array.isArray(message["content"]) ? (message["content"] as unknown[]) : [];
    const out: AgentEvent[] = [];
    for (const block of content) {
      const b = (block ?? {}) as Record<string, unknown>;
      const blockType = String(b["type"] ?? "");
      if (blockType === "text") {
        const text = String(b["text"] ?? "");
        if (text) out.push({ kind: "text", text });
      } else if (blockType === "tool_use") {
        out.push({
          kind: "tool_start",
          id: String(b["id"] ?? ""),
          name: String(b["name"] ?? ""),
          input: b["input"] ?? null,
        });
      }
    }
    // Claude emits per-turn usage on every assistant message. Treat it as an
    // incremental tokens event so the UI can accumulate a live counter.
    const usage = (message["usage"] ?? {}) as Record<string, unknown>;
    if (usage && Object.keys(usage).length > 0) {
      out.push({
        kind: "tokens",
        inputTokens: numericOrZero(usage["input_tokens"]),
        outputTokens: numericOrZero(usage["output_tokens"]),
        cacheReadTokens: numericOrZero(usage["cache_read_input_tokens"]),
        cacheCreationTokens: numericOrZero(usage["cache_creation_input_tokens"]),
      });
    }
    return out;
  }

  if (type === "user") {
    const message = (o["message"] ?? {}) as Record<string, unknown>;
    const content = Array.isArray(message["content"]) ? (message["content"] as unknown[]) : [];
    const out: AgentEvent[] = [];
    for (const block of content) {
      const b = (block ?? {}) as Record<string, unknown>;
      if (b["type"] !== "tool_result") continue;
      const id = String(b["tool_use_id"] ?? "");
      const isError = Boolean(b["is_error"]);
      out.push({
        kind: "tool_done",
        id,
        ok: !isError,
        preview: extractToolResultPreview(b["content"]),
      });
    }
    return out;
  }

  if (type === "rate_limit_event") {
    const info = (o["rate_limit_info"] ?? {}) as Record<string, unknown>;
    return [{ kind: "rate_limit", status: String(info["status"] ?? "rate_limited") }];
  }

  if (type === "result") {
    const isError = Boolean(o["is_error"]);
    return [
      {
        kind: "summary",
        ok: !isError,
        durationMs:
          typeof o["duration_ms"] === "number" ? (o["duration_ms"] as number) : undefined,
        costUsd:
          typeof o["total_cost_usd"] === "number"
            ? (o["total_cost_usd"] as number)
            : undefined,
        turns: typeof o["num_turns"] === "number" ? (o["num_turns"] as number) : undefined,
        finalText: typeof o["result"] === "string" ? (o["result"] as string) : undefined,
      },
    ];
  }

  return [{ kind: "raw", text: JSON.stringify(o) }];
}

/** Max chars retained for a tool result preview. Long enough to carry the
 *  page title + a few lines of context for a fetched URL; short enough that
 *  storing it for every tool call doesn't bloat memory.
 */
const RESULT_PREVIEW_MAX = 600;

function extractToolResultPreview(content: unknown): string | undefined {
  let text: string | undefined;
  if (typeof content === "string") text = content;
  else if (Array.isArray(content)) {
    // Claude's tool_result content is either a string (legacy) or an array of
    // {type:"text",text:...} blocks. Concatenate text blocks so multi-block
    // results (rare) still surface a useful preview.
    const texts: string[] = [];
    for (const block of content) {
      const b = (block ?? {}) as Record<string, unknown>;
      if (b["type"] === "text" && typeof b["text"] === "string") {
        texts.push(b["text"] as string);
      }
    }
    text = texts.join("\n");
  }
  if (!text) return undefined;
  // Keep newlines so the UI's detail panel can render multi-line previews.
  // The PhaseRunner row display extracts the first line for the single-line
  // table view.
  return text.trim().slice(0, RESULT_PREVIEW_MAX);
}

function numericOrZero(v: unknown): number {
  return typeof v === "number" && Number.isFinite(v) ? v : 0;
}

function firstLine(s: string): string {
  const trimmed = s.trim();
  const nl = trimmed.indexOf("\n");
  return nl < 0 ? trimmed : trimmed.slice(0, nl);
}

// ---------------------------------------------------------------------------
// Codex --json
// ---------------------------------------------------------------------------
//
// Observed shapes (from `codex exec --json -`):
//
//   { type: "thread.started", thread_id }
//   { type: "turn.started" }
//   { type: "item.started", item: { id, type: "agent_message"|"tool_call"|..., ... } }
//   { type: "item.completed", item: { id, type, ... } }
//   { type: "turn.completed", usage: {...} }
//
// Codex's item.type for tool invocations varies by Codex version; we treat any
// non-agent_message item that has an `id` and `name` (or `command`) as a tool.

/**
 * Reshape a Codex `item` into the (name, input) pair the rest of the TUI
 * expects. Centralizes the per-item-type translation so the parser stays a
 * thin dispatcher and the detail panel doesn't have to know about Codex's
 * field names.
 *
 * For `command_execution` specifically: the name becomes "bash" (short,
 * fits the row chip) and the command goes into `{ command }` so
 * `formatInputDetail` renders it via its existing "command: …" branch.
 * Without this the tool name was the full shell command — useless for
 * scanning the list — and the input was a plain string, which the formatter
 * skipped entirely.
 */
function normalizeCodexToolInput(
  item: Record<string, unknown>,
  itemType: string,
): { name: string; input: unknown } {
  if (itemType === "command_execution") {
    const command = typeof item["command"] === "string" ? (item["command"] as string) : "";
    return { name: "bash", input: command ? { command } : null };
  }
  // Generic fallback for other item types Codex emits — preserve whatever
  // name/input fields are present so we don't lose information.
  const name = String(item["name"] ?? itemType);
  const input =
    item["input"] != null
      ? item["input"]
      : typeof item["command"] === "string"
        ? { command: item["command"] }
        : null;
  return { name, input };
}

function parseCodexObject(obj: unknown): AgentEvent[] {
  if (!obj || typeof obj !== "object") return [];
  const o = obj as Record<string, unknown>;
  const type = String(o["type"] ?? "");

  if (type === "thread.started") {
    return [
      {
        kind: "init",
        sessionId:
          typeof o["thread_id"] === "string" ? (o["thread_id"] as string) : undefined,
      },
    ];
  }

  if (type === "turn.started") return [];

  if (type === "item.started") {
    const item = (o["item"] ?? {}) as Record<string, unknown>;
    const itemType = String(item["type"] ?? "");
    if (itemType === "agent_message") return [];
    const { name, input } = normalizeCodexToolInput(item, itemType);
    return [
      {
        kind: "tool_start",
        id: String(item["id"] ?? ""),
        name,
        input,
      },
    ];
  }

  if (type === "item.completed") {
    const item = (o["item"] ?? {}) as Record<string, unknown>;
    const itemType = String(item["type"] ?? "");
    if (itemType === "agent_message") {
      const text = String(item["text"] ?? "");
      if (!text) return [];
      return [{ kind: "text", text }];
    }
    // Tool result.
    const id = String(item["id"] ?? "");
    // Codex marks the process "completed" even when the underlying command
    // exited non-zero — so we also have to consult exit_code. Without this
    // the UI showed "failed without an error message" for half of the calls
    // (parser said ok, but resultPreview was empty because we were reading
    // the wrong output field too).
    const statusLower = String(item["status"] ?? "").toLowerCase();
    const exitCode = item["exit_code"];
    const isError =
      Boolean(item["is_error"]) ||
      statusLower === "error" ||
      statusLower === "failed" ||
      (typeof exitCode === "number" && exitCode !== 0);
    // Codex's command_execution puts the captured stdout/stderr in
    // `aggregated_output`. Other (future) item types may use `output`;
    // fall back to that for forward-compat.
    const rawOutput =
      typeof item["aggregated_output"] === "string"
        ? (item["aggregated_output"] as string)
        : typeof item["output"] === "string"
          ? (item["output"] as string)
          : "";
    return [
      {
        kind: "tool_done",
        id,
        ok: !isError,
        preview: rawOutput ? rawOutput.trim().slice(0, RESULT_PREVIEW_MAX) : undefined,
      },
    ];
  }

  if (type === "turn.completed") {
    const usage = (o["usage"] ?? {}) as Record<string, unknown>;
    const out: AgentEvent[] = [];
    if (usage && Object.keys(usage).length > 0) {
      out.push({
        kind: "tokens",
        inputTokens: numericOrZero(usage["input_tokens"]),
        outputTokens: numericOrZero(usage["output_tokens"]),
        cacheReadTokens: numericOrZero(usage["cached_input_tokens"]),
        cacheCreationTokens: 0,
      });
    }
    out.push({
      kind: "summary",
      ok: true,
      turns:
        typeof usage["turns"] === "number" ? (usage["turns"] as number) : undefined,
    });
    return out;
  }

  return [{ kind: "raw", text: JSON.stringify(o) }];
}

// ---------------------------------------------------------------------------
// Line buffering — chunks from a child's stdout may split a JSON line.
// ---------------------------------------------------------------------------

export class LineBuffer {
  private remainder = "";

  push(chunk: string): string[] {
    const combined = this.remainder + chunk;
    const lines = combined.split("\n");
    // The last element is whatever came after the final \n (could be empty).
    // Hold it for the next push so we don't try to parse a partial JSON line.
    this.remainder = lines.pop() ?? "";
    return lines;
  }

  flush(): string[] {
    const out = this.remainder ? [this.remainder] : [];
    this.remainder = "";
    return out;
  }
}
