// Normalized event stream used by the TUI to render agent progress.
//
// Both Claude (`--output-format stream-json --verbose`) and Codex (`--json`)
// emit newline-delimited JSON. Their schemas differ; per-CLI parsers in this
// directory convert each line into one of the AgentEvent variants below so
// the PhaseRunner UI is agent-agnostic.

export type AgentEvent =
  | { kind: "init"; model?: string; sessionId?: string }
  | { kind: "text"; text: string }
  | { kind: "tool_start"; id: string; name: string; input: unknown }
  | {
      kind: "tool_done";
      id: string;
      ok: boolean;
      /** Single-line preview of the result. */
      preview?: string;
    }
  | {
      kind: "summary";
      ok: boolean;
      durationMs?: number;
      costUsd?: number;
      turns?: number;
      finalText?: string;
    }
  | { kind: "rate_limit"; status: string }
  /**
   * Per-turn token usage emitted by the agent. Claude includes this in every
   * `assistant` event; Codex emits it once in `turn.completed`. Token counts
   * here are *incremental* — the UI should accumulate them.
   */
  | {
      kind: "tokens";
      inputTokens: number;
      outputTokens: number;
      cacheReadTokens: number;
      cacheCreationTokens: number;
    }
  /** A JSON line we couldn't classify. Surfaced dimmed in the UI. */
  | { kind: "raw"; text: string };
