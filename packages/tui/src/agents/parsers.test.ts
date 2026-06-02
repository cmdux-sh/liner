import { describe, expect, it } from "vitest";
import { LineBuffer, parseLine } from "./parsers.js";

describe("parseLine — Claude stream-json", () => {
  it("converts system init into an init event", () => {
    const line = JSON.stringify({
      type: "system",
      subtype: "init",
      model: "claude-opus-4-7",
      session_id: "abc-123",
      tools: ["Read", "Write"],
    });
    const events = parseLine("claude", line);
    expect(events).toEqual([
      { kind: "init", model: "claude-opus-4-7", sessionId: "abc-123" },
    ]);
  });

  it("splits an assistant message with mixed content into text + tool_start", () => {
    const line = JSON.stringify({
      type: "assistant",
      message: {
        content: [
          { type: "text", text: "Reading the JTBD now." },
          {
            type: "tool_use",
            id: "toolu_1",
            name: "Read",
            input: { file_path: "working/01-jtbd-and-knowledge-map.md" },
          },
        ],
      },
    });
    const events = parseLine("claude", line);
    expect(events).toHaveLength(2);
    expect(events[0]).toEqual({ kind: "text", text: "Reading the JTBD now." });
    expect(events[1]).toMatchObject({
      kind: "tool_start",
      id: "toolu_1",
      name: "Read",
      input: { file_path: "working/01-jtbd-and-knowledge-map.md" },
    });
  });

  it("extracts per-turn token usage from assistant messages", () => {
    const line = JSON.stringify({
      type: "assistant",
      message: {
        content: [{ type: "text", text: "ok" }],
        usage: {
          input_tokens: 5,
          output_tokens: 12,
          cache_read_input_tokens: 16401,
          cache_creation_input_tokens: 14115,
        },
      },
    });
    const events = parseLine("claude", line);
    // text event + tokens event
    expect(events).toHaveLength(2);
    expect(events[1]).toEqual({
      kind: "tokens",
      inputTokens: 5,
      outputTokens: 12,
      cacheReadTokens: 16401,
      cacheCreationTokens: 14115,
    });
  });

  it("converts a tool_result block into tool_done with a preview", () => {
    const line = JSON.stringify({
      type: "user",
      message: {
        content: [
          {
            type: "tool_result",
            tool_use_id: "toolu_1",
            is_error: false,
            content: [{ type: "text", text: "first line\nsecond line" }],
          },
        ],
      },
    });
    const events = parseLine("claude", line);
    // Preview keeps newlines so the detail panel can render multi-line
    // results; the row display extracts the first line itself.
    expect(events).toEqual([
      { kind: "tool_done", id: "toolu_1", ok: true, preview: "first line\nsecond line" },
    ]);
  });

  it("marks tool_done as failed when is_error is true", () => {
    const line = JSON.stringify({
      type: "user",
      message: {
        content: [
          {
            type: "tool_result",
            tool_use_id: "toolu_x",
            is_error: true,
            content: [{ type: "text", text: "File not found" }],
          },
        ],
      },
    });
    expect(parseLine("claude", line)).toEqual([
      { kind: "tool_done", id: "toolu_x", ok: false, preview: "File not found" },
    ]);
  });

  it("converts a result event into a summary", () => {
    const line = JSON.stringify({
      type: "result",
      subtype: "success",
      is_error: false,
      duration_ms: 12345,
      num_turns: 3,
      total_cost_usd: 0.1234,
      result: "Final summary text.",
    });
    expect(parseLine("claude", line)).toEqual([
      {
        kind: "summary",
        ok: true,
        durationMs: 12345,
        costUsd: 0.1234,
        turns: 3,
        finalText: "Final summary text.",
      },
    ]);
  });

  it("converts a rate_limit_event into a rate_limit event", () => {
    const line = JSON.stringify({
      type: "rate_limit_event",
      rate_limit_info: { status: "allowed" },
    });
    expect(parseLine("claude", line)).toEqual([
      { kind: "rate_limit", status: "allowed" },
    ]);
  });

  it("falls back to a raw event when JSON is unparseable", () => {
    expect(parseLine("claude", "not-json {")).toEqual([
      { kind: "raw", text: "not-json {" },
    ]);
  });

  it("returns nothing for an empty line", () => {
    expect(parseLine("claude", "")).toEqual([]);
    expect(parseLine("claude", "   \n  ")).toEqual([]);
  });
});

describe("parseLine — Codex --json", () => {
  it("converts thread.started into an init event", () => {
    const line = JSON.stringify({ type: "thread.started", thread_id: "thread-1" });
    expect(parseLine("codex", line)).toEqual([
      { kind: "init", sessionId: "thread-1" },
    ]);
  });

  it("converts an agent_message item into a text event", () => {
    const line = JSON.stringify({
      type: "item.completed",
      item: { id: "item_0", type: "agent_message", text: "Hello world." },
    });
    expect(parseLine("codex", line)).toEqual([
      { kind: "text", text: "Hello world." },
    ]);
  });

  it("packages a command_execution start as a structured bash input", () => {
    // Codex's most common tool. Pre-fix the name was the entire shell command
    // ('/bin/zsh -lc "sed -n …"'), which was useless as a row label, and the
    // input was a bare string the detail panel skipped entirely. Now: name
    // is "bash" and input is { command } so formatInputDetail renders it.
    const line = JSON.stringify({
      type: "item.started",
      item: {
        id: "item_1",
        type: "command_execution",
        command: "/bin/zsh -lc \"sed -n '1,5p' README.md\"",
      },
    });
    expect(parseLine("codex", line)).toEqual([
      {
        kind: "tool_start",
        id: "item_1",
        name: "bash",
        input: { command: "/bin/zsh -lc \"sed -n '1,5p' README.md\"" },
      },
    ]);
  });

  it("reads aggregated_output (not output) on command_execution completed", () => {
    // Real-world regression: parser was reading `item.output`, but Codex's
    // command_execution puts captured stdout/stderr in `aggregated_output`.
    // Result: every completed bash call had an empty preview, and the UI
    // fell back to "failed without an error message" for failures.
    const line = JSON.stringify({
      type: "item.completed",
      item: {
        id: "item_1",
        type: "command_execution",
        command: "echo hi",
        aggregated_output: "hi\n",
        exit_code: 0,
        status: "completed",
      },
    });
    expect(parseLine("codex", line)).toEqual([
      { kind: "tool_done", id: "item_1", ok: true, preview: "hi" },
    ]);
  });

  it("treats non-zero exit_code as failure even when status is 'completed'", () => {
    // Codex marks the *process* completed regardless of the command's exit
    // code. Without checking exit_code we'd render a clean ✓ on something
    // that actually failed (e.g. `sed` against a missing file).
    const line = JSON.stringify({
      type: "item.completed",
      item: {
        id: "item_1",
        type: "command_execution",
        command: "sed -n '1p' /no/such/file",
        aggregated_output: "sed: /no/such/file: No such file or directory",
        exit_code: 1,
        status: "completed",
      },
    });
    expect(parseLine("codex", line)).toEqual([
      {
        kind: "tool_done",
        id: "item_1",
        ok: false,
        preview: "sed: /no/such/file: No such file or directory",
      },
    ]);
  });

  it("falls back to legacy output field when aggregated_output is absent", () => {
    // Forward-compat: an older or alternate Codex shape that uses `output`
    // should still produce a preview rather than going dark.
    const line = JSON.stringify({
      type: "item.completed",
      item: { id: "x", type: "shell_call", status: "success", output: "ok" },
    });
    expect(parseLine("codex", line)).toEqual([
      { kind: "tool_done", id: "x", ok: true, preview: "ok" },
    ]);
  });
});

describe("LineBuffer", () => {
  it("yields complete lines and holds the partial tail", () => {
    const buf = new LineBuffer();
    expect(buf.push('{"a":1}\n{"b":')).toEqual(['{"a":1}']);
    expect(buf.push('2}\n{"c":3}\n')).toEqual(['{"b":2}', '{"c":3}']);
    expect(buf.flush()).toEqual([]);
  });

  it("flushes the trailing partial line on close", () => {
    const buf = new LineBuffer();
    buf.push("partial without newline");
    expect(buf.flush()).toEqual(["partial without newline"]);
    expect(buf.flush()).toEqual([]);
  });
});
