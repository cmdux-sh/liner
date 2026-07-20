import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdout } from "ink";
import Spinner from "ink-spinner";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { NextSteps } from "../components/NextSteps.js";
import { MUTED, CURRENT, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { useApp } from "../store.js";
import {
  detectAgents,
  resolveSkillPathWithDiagnostics,
  type SkillPathResolution,
} from "../agents/detect.js";
import { resolveConfiguredAgent } from "../config.js";
import { runPhaseWithAgent } from "../agents/runner.js";
import type { AgentDescriptor, AgentRunHandle } from "../agents/types.js";
import type { AgentEvent } from "../agents/events.js";
import {
  computePhases,
  readGateState,
  writeGateState,
  type PhaseId,
  type PhaseRecord,
} from "../phases.js";
import { markPhaseComplete, readProgress } from "../progress.js";
import { projectFolder } from "../yaml-io.js";
import type { Screen } from "../store.js";
import type { Tape } from "../types.js";

type Stage =
  | { kind: "picking_agent"; agents: AgentDescriptor[]; selectedIndex: number }
  | { kind: "missing_agent" }
  | { kind: "missing_skill" }
  | { kind: "running"; agent: AgentDescriptor }
  | { kind: "done"; agent: AgentDescriptor; code: number | null };

type ToolCall = {
  id: string;
  name: string;
  /** Raw input object from the agent stream — used by the detail panel. */
  input?: unknown;
  /** One-line summary derived from `input` — used by the table row. */
  inputPreview: string;
  status: "running" | "ok" | "failed";
  /** Possibly multi-line preview (up to ~600 chars) from the parser. */
  resultPreview?: string;
};

type TokenCounts = {
  input: number;
  output: number;
  cacheRead: number;
  cacheCreation: number;
};

// Tool list viewport. Always shows VIEWPORT rows; arrow keys move the cursor
// and scroll the window when the cursor hits an edge. Replaces the previous
// "press t to expand / collapse" toggle — keeps the layout perfectly stable
// regardless of how many tool calls the agent made.
const TOOLS_VIEWPORT = 8;
const FINAL_RESPONSE_VIEWPORT = 7;
const FINAL_RESPONSE_INSPECT_VIEWPORT = 3;

export function PhaseRunner({
  tape,
  folder,
  phaseId,
}: {
  tape: Tape;
  folder: string;
  phaseId: PhaseId;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { stdout } = useStdout();
  const project = useMemo(() => projectFolder(folder), [folder]);
  const phase = useMemo(
    () => computePhases(project, tape, readProgress(folder)).find((p) => p.id === phaseId) ?? null,
    [project, tape, phaseId, folder],
  );
  const skillResolution = useMemo(() => resolveSkillPathWithDiagnostics(), []);
  const skillPath = skillResolution.path;
  const initial = useMemo(() => initialStage(skillPath), [skillPath]);
  const [stage, setStage] = useState<Stage>(initial.stage);

  // Structured state derived from AgentEvent stream.
  const [tools, setTools] = useState<ToolCall[]>([]);
  const [finalText, setFinalText] = useState<string>("");
  // Refs mirror state for the handle.done.then() closure.
  const finalTextRef = useRef<string>("");
  const currentTextRef = useRef<string>("");
  const [stats, setStats] = useState<{
    events: number;
    cost?: number;
    durationMs?: number;
    startedAt?: number;
    tokens: TokenCounts;
  }>({ events: 0, tokens: zeroTokens() });
  // Index into the full `tools` array. Drives the cursor in the done-state
  // tool list and the detail panel below it. Snaps to the last tool when the
  // run finishes.
  const [toolCursor, setToolCursor] = useState(0);
  // First tool index visible in the viewport. The arrow handlers maintain
  // the invariant that `toolCursor` is always inside [scrollTop, scrollTop +
  // VIEWPORT) by scrolling when the cursor hits an edge.
  const [toolScrollTop, setToolScrollTop] = useState(0);
  // Inspect mode is a modal toggle pressed with `t`. When OFF (default), the
  // tool list is just a static viewport showing the tail end of the call
  // history — no cursor, no detail panel, arrow keys don't do anything.
  // When ON, arrows scroll the cursor through the full list and a detail
  // panel renders below showing the selected call's full input + result.
  // Press `t` again to exit. This keeps the default done view compact and
  // makes the inspection ritual explicit instead of "you happen to be in
  // arrow mode all the time."
  const [inspectMode, setInspectMode] = useState(false);
  const [finalScrollTop, setFinalScrollTop] = useState(0);
  const handleRef = useRef<AgentRunHandle | null>(null);
  const renderedFinalText = finalText || currentTextRef.current;
  const finalLineCount = renderedFinalText ? renderedFinalText.split("\n").length : 0;
  const finalViewportLines = finalResponseViewportLines(stdout?.rows ?? 24, inspectMode);

  const handleEvent = useCallback((event: AgentEvent): void => {
    setStats((s) => ({ ...s, events: s.events + 1 }));

    switch (event.kind) {
      case "init":
        return;
      case "text":
        // Track the latest assistant text for the "final response" pane on
        // the done state. We don't render it live anymore — that was the
        // source of the scrollback laddering bug on long runs.
        currentTextRef.current = currentTextRef.current
          ? currentTextRef.current + "\n\n" + event.text
          : event.text;
        return;
      case "tool_start":
        setTools((prev) => [
          ...prev,
          {
            id: event.id,
            name: event.name,
            input: event.input,
            inputPreview: summarizeInput(event.input),
            status: "running",
          },
        ]);
        return;
      case "tool_done":
        setTools((prev) =>
          prev.map((t) =>
            t.id === event.id
              ? {
                  ...t,
                  status: event.ok ? "ok" : "failed",
                  resultPreview: event.preview,
                }
              : t,
          ),
        );
        return;
      case "tokens":
        setStats((s) => ({
          ...s,
          tokens: {
            input: s.tokens.input + event.inputTokens,
            output: s.tokens.output + event.outputTokens,
            cacheRead: s.tokens.cacheRead + event.cacheReadTokens,
            cacheCreation: s.tokens.cacheCreation + event.cacheCreationTokens,
          },
        }));
        return;
      case "summary":
        setStats((s) => ({
          ...s,
          cost: event.costUsd ?? s.cost,
          durationMs: event.durationMs ?? s.durationMs,
        }));
        if (event.finalText) {
          setFinalText(event.finalText);
          finalTextRef.current = event.finalText;
        }
        return;
      case "rate_limit":
        // Surface rate limits via a notification so they don't get lost in the
        // event stream, but don't dedicate UI space to them in the compact view.
        app.setNotification({
          kind: "error",
          message: `Rate limit: ${event.status}`,
        });
        return;
      case "raw":
        // Quietly drop. Anything important we already surfaced; raw stderr
        // junk was making the screen flicker.
        return;
    }
  }, [app]);

  function launch(agent: AgentDescriptor, opts: { resume?: boolean } = {}): void {
    if (!skillPath) {
      setStage({ kind: "missing_skill" });
      return;
    }
    setStage({ kind: "running", agent });
    setTools([]);
    setFinalText("");
    setFinalScrollTop(0);
    currentTextRef.current = "";
    finalTextRef.current = "";
    setStats({ events: 0, tokens: zeroTokens(), startedAt: Date.now() });
    const handle = runPhaseWithAgent({
      agent,
      phaseId,
      project: folder,
      skillPath,
      tape,
      resume: opts.resume,
      onEvent: handleEvent,
    });
    handleRef.current = handle;
    handle.done.then(({ code }) => {
      // Clear the screen as we flip running → done. The running view and the
      // done view differ in height, and on terminals that keep scrollback in
      // the alternate buffer the old running frames otherwise linger above
      // the done view (stacked duplicate headers). Clearing once on the
      // transition lands the done view on a fresh canvas.
      if (process.stdout.isTTY) process.stdout.write("\x1B[2J\x1B[3J\x1B[H");
      setStage({ kind: "done", agent, code });
      // Surface the agent's final-line summary as a notification so it
      // survives navigation. The full final response stays on this screen.
      const summary = finalTextRef.current || currentTextRef.current;
      if (summary) {
        const oneLine = summary.split("\n").find((l) => l.trim().length > 0) ?? "";
        if (oneLine) {
          const truncated = oneLine.length > 140 ? oneLine.slice(0, 139) + "…" : oneLine;
          app.setNotification({
            kind: code === 0 ? "info" : "error",
            message: `${phase?.label ?? phaseId}: ${truncated}`,
          });
        }
      }
      // On clean success, advance the progress cursor by one. Assembly is
      // the exception — its "real" completion lands in TapeDraftReview when
      // the curator accepts the proposed sources. Compile and the gates are
      // their own screens and advance themselves.
      if (code === 0 && phaseId !== "assembly") {
        markPhaseComplete(folder, phaseId);
      }
      // Phase 7 special-case: a successful assembly run wrote
      // working/07-tape-draft.yaml. Jump straight into the draft-review.
      if (code === 0 && phaseId === "assembly") {
        app.navigate({ kind: "tapeDraft", tape, folder });
      }
    });
  }

  function cancel(): void {
    handleRef.current?.cancel();
  }

  useEffect(() => {
    if (initial.autoLaunch) {
      launch(initial.autoLaunch);
    }
    return () => {
      handleRef.current?.cancel();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // When entering the done state, park the tool cursor on the most recent
  // call so the detail panel below the table immediately shows something
  // useful (the last thing the agent did before finishing). Snap the scroll
  // window so the cursor sits at the bottom of the viewport.
  useEffect(() => {
    if (stage.kind === "done" && tools.length > 0) {
      const cursor = tools.length - 1;
      setToolCursor(cursor);
      setToolScrollTop(Math.max(0, cursor - TOOLS_VIEWPORT + 1));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stage.kind]);

  function advance(): void {
    if (stage.kind !== "done") return;
    const next = nextScreenFor(phaseId, tape, folder);
    if (next) app.navigate(next);
    else app.navigate({ kind: "hub", tape, folder });
  }

  useInput((input, key) => {
    if (stage.kind === "picking_agent") {
      if (key.upArrow || input === "k") {
        setStage({ ...stage, selectedIndex: Math.max(0, stage.selectedIndex - 1) });
      } else if (key.downArrow || input === "j") {
        setStage({
          ...stage,
          selectedIndex: Math.min(stage.agents.length - 1, stage.selectedIndex + 1),
        });
      } else if (key.return) {
        const agent = stage.agents[stage.selectedIndex];
        if (agent) launch(agent);
      } else if (input === "q") {
        ink.exit();
      } else if (key.escape) {
        app.navigate({ kind: "hub", tape, folder });
      }
      return;
    }
    if (stage.kind === "missing_agent" || stage.kind === "missing_skill") {
      if (input === "q") {
        ink.exit();
      } else if (key.escape || input === "b") {
        app.navigate({ kind: "hub", tape, folder });
      }
      return;
    }
    if (stage.kind === "running") {
      if (key.escape || input === "c") cancel();
      return;
    }
    if (stage.kind === "done") {
      const failed = stage.code !== 0;

      // `t` is the modal toggle. Always available, regardless of mode.
      if (input === "t") {
        setInspectMode((m) => {
          // Entering inspect mode: snap cursor + window to the last call so
          // the detail panel renders something useful from the start.
          if (!m && tools.length > 0) {
            const cursor = tools.length - 1;
            setToolCursor(cursor);
            setToolScrollTop(Math.max(0, cursor - TOOLS_VIEWPORT + 1));
          }
          return !m;
        });
        return;
      }

      if (inspectMode) {
        // Inspect mode: arrows scroll, esc / b / t exit, everything else
        // (enter to advance, r to re-run) is intentionally locked so the
        // user doesn't accidentally move forward while scanning calls.
        if (key.upArrow || input === "k") {
          if (tools.length === 0) return;
          setToolCursor((c) => {
            const next = Math.max(0, c - 1);
            setToolScrollTop((s) => (next < s ? next : s));
            return next;
          });
        } else if (key.downArrow || input === "j") {
          if (tools.length === 0) return;
          setToolCursor((c) => {
            const next = Math.min(tools.length - 1, c + 1);
            setToolScrollTop((s) =>
              next >= s + TOOLS_VIEWPORT ? next - TOOLS_VIEWPORT + 1 : s,
            );
            return next;
          });
        } else if (key.pageUp) {
          if (tools.length === 0) return;
          setToolCursor((c) => {
            const next = Math.max(0, c - TOOLS_VIEWPORT);
            setToolScrollTop(Math.max(0, next));
            return next;
          });
        } else if (key.pageDown) {
          if (tools.length === 0) return;
          setToolCursor((c) => {
            const next = Math.min(tools.length - 1, c + TOOLS_VIEWPORT);
            setToolScrollTop(Math.max(0, next - TOOLS_VIEWPORT + 1));
            return next;
          });
        } else if (key.escape) {
          // Esc inside inspect mode = exit the mode (does NOT bail to hub).
          setInspectMode(false);
        }
        return;
      }

      // Default done mode: arrows/page keys scroll the final response. Normal
      // terminal scrollback is unavailable in the alternate screen, and mouse
      // wheels arrive as arrow keys in many terminals; make those inputs move
      // the readable response instead of doing nothing.
      if (key.upArrow || input === "k") {
        setFinalScrollTop((s) =>
          Math.max(0, Math.min(s, Math.max(0, finalLineCount - finalViewportLines)) - 1),
        );
        return;
      }
      if (key.downArrow || input === "j") {
        setFinalScrollTop((s) =>
          Math.min(
            Math.max(0, finalLineCount - finalViewportLines),
            Math.min(s, Math.max(0, finalLineCount - finalViewportLines)) + 1,
          ),
        );
        return;
      }
      if (key.pageUp) {
        setFinalScrollTop((s) =>
          Math.max(
            0,
            Math.min(s, Math.max(0, finalLineCount - finalViewportLines)) -
              finalViewportLines,
          ),
        );
        return;
      }
      if (key.pageDown) {
        setFinalScrollTop((s) =>
          Math.min(
            Math.max(0, finalLineCount - finalViewportLines),
            s + finalViewportLines,
          ),
        );
        return;
      }

      // Default mode: enter advances, b/esc backs to hub, r retries on fail.
      if (key.return) {
        // On failure, Enter = resume (cheap continuation, no re-fetching).
        // On success, Enter = advance to the next logical screen.
        if (failed) launch(stage.agent, { resume: true });
        else advance();
      } else if (input === "r" && failed) {
        // Failure-only escape hatch: re-run from scratch.
        launch(stage.agent);
      } else if (input === "q") {
        ink.exit();
      } else if (key.escape || input === "b") {
        app.navigate({ kind: "hub", tape, folder });
      }
      return;
    }
  });

  return (
    <Box flexDirection="column">
      <Header
        title={`phase ${phase?.number ?? "?"} — ${phase?.label ?? phaseId}`}
        subtitle={subtitleFor(stage, skillPath)}
      />

      {stage.kind === "picking_agent" ? (
        <PickingAgentView stage={stage} />
      ) : null}

      {stage.kind === "missing_agent" ? <MissingAgentView /> : null}
      {stage.kind === "missing_skill" ? (
        <MissingSkillView resolution={skillResolution} />
      ) : null}

      {stage.kind === "running" ? (
        <RunningView
          agent={stage.agent}
          phase={phase}
          phaseId={phaseId}
          tools={tools}
          stats={stats}
        />
      ) : null}

      {stage.kind === "done" ? (
        <DoneView
          agent={stage.agent}
          code={stage.code}
          phaseId={phaseId}
          phase={phase}
          tape={tape}
          folder={folder}
          tools={tools}
          finalText={renderedFinalText}
          finalScrollTop={finalScrollTop}
          finalViewportLines={finalViewportLines}
          stats={stats}
          toolCursor={toolCursor}
          toolScrollTop={toolScrollTop}
          inspectMode={inspectMode}
        />
      ) : null}

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      <KeyHints hints={hintsFor(stage, inspectMode)} />
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Sub-views
// ---------------------------------------------------------------------------

function PickingAgentView({
  stage,
}: {
  stage: { kind: "picking_agent"; agents: AgentDescriptor[]; selectedIndex: number };
}): React.ReactElement {
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text>Multiple agents available — pick one:</Text>
      <Box marginTop={1} flexDirection="column">
        {stage.agents.map((a, i) => (
          <Box key={a.id}>
            <Text color={i === stage.selectedIndex ? STRUCTURE : undefined}>
              {i === stage.selectedIndex ? "▶ " : "  "}
              {a.name}
            </Text>
            <Text color={MUTED}>{"  "}{a.bin}</Text>
          </Box>
        ))}
      </Box>
      <Box marginTop={1}>
        <Text color={MUTED}>
          Tip: pick once in the splash menu (<Text color={STRUCTURE}>[s]</Text> Settings) to skip
          this prompt forever — or set <Text color={STRUCTURE}>LINER_AGENT=claude</Text> in your shell.
        </Text>
      </Box>
    </Box>
  );
}

function MissingAgentView(): React.ReactElement {
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text color={ERROR}>No agent CLI found on PATH.</Text>
      <Box marginTop={1} flexDirection="column">
        <Text>Liner can drive the methodology with either:</Text>
        <Text>  - Claude Code (<Text color={STRUCTURE}>claude</Text>) — install: <Text color={STRUCTURE}>npm install -g @anthropic-ai/claude-code</Text></Text>
        <Text>  - OpenAI Codex (<Text color={STRUCTURE}>codex</Text>) — see https://github.com/openai/codex</Text>
      </Box>
    </Box>
  );
}

function MissingSkillView({
  resolution,
}: {
  resolution: SkillPathResolution;
}): React.ReactElement {
  const checked = resolution.searched.slice(0, 5);
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text color={ERROR}>Couldn't find the curating-mixtapes skill bundle.</Text>
      <Box marginTop={1} flexDirection="column">
        <Text color={MUTED}>
          Set <Text color={STRUCTURE}>LINER_SKILL_PATH</Text> to the directory containing SKILL.md.
        </Text>
        {resolution.envPath ? (
          <Text color={MUTED}>
            Current <Text color={STRUCTURE}>LINER_SKILL_PATH</Text>: {resolution.envPath}
          </Text>
        ) : null}
        {checked.length > 0 ? (
          <Box marginTop={1} flexDirection="column">
            <Text color={MUTED}>Checked:</Text>
            {checked.map((path) => (
              <Text key={path} color={MUTED}>
                {"  - "}
                {path}
              </Text>
            ))}
          </Box>
        ) : null}
      </Box>
    </Box>
  );
}

/**
 * Compact running view. Two lines + a bordered status block — fits any
 * terminal height. The full tool list and final response wait for the done
 * state so the live view never overflows the viewport.
 */
function RunningView({
  agent,
  phase,
  phaseId,
  tools,
  stats,
}: {
  agent: AgentDescriptor;
  phase: PhaseRecord | null;
  phaseId: PhaseId;
  tools: ToolCall[];
  stats: { events: number; tokens: TokenCounts; startedAt?: number };
}): React.ReactElement {
  const latest = tools[tools.length - 1];
  const liveDuration = stats.startedAt
    ? formatDuration(Date.now() - stats.startedAt)
    : "";
  const totalTokens =
    stats.tokens.input +
    stats.tokens.output +
    stats.tokens.cacheRead +
    stats.tokens.cacheCreation;
  const phaseLabel = phase
    ? `Phase ${phase.number} — ${phase.label}`
    : `Phase ${phaseId}`;

  return (
    <Box marginBottom={1}>
      <LabeledBox label={phaseLabel} color={CURRENT} paddingX={1} paddingY={1}>
        <Box>
          <Text color={CURRENT}>
            <Spinner type="dots" />{" "}
          </Text>
          <Text>{agent.name}</Text>
          <Text color={MUTED}>{"   ·   running"}</Text>
        </Box>
        <Box marginTop={1}>
          <Text color={MUTED}>
            {stats.events} event{stats.events === 1 ? "" : "s"}
            {"  ·  "}
            {tools.length} tool{tools.length === 1 ? "" : "s"}
            {"  ·  "}
            {formatTokens(totalTokens)} tokens
            {liveDuration ? `  ·  ${liveDuration}` : ""}
          </Text>
        </Box>
        {latest ? (
          <Box marginTop={1}>
            <Text color={MUTED}>latest: </Text>
            <Text color={latest.status === "ok" ? SUCCESS : latest.status === "failed" ? ERROR : STRUCTURE}>
              {latest.status === "ok" ? "✓" : latest.status === "failed" ? "✗" : "⟳"}{" "}
            </Text>
            <Text>{latest.name.replace(/^mcp__[^_]+__/, "")}</Text>
            <Text color={MUTED}>  {previewify(latest.inputPreview, 60)}</Text>
          </Box>
        ) : (
          <Box marginTop={1}>
            <Text color={MUTED}>warming up…</Text>
          </Box>
        )}
      </LabeledBox>
    </Box>
  );
}

function DoneView({
  agent,
  code,
  phaseId,
  phase,
  tape,
  folder,
  tools,
  finalText,
  finalScrollTop,
  finalViewportLines,
  stats,
  toolCursor,
  toolScrollTop,
  inspectMode,
}: {
  agent: AgentDescriptor;
  code: number | null;
  phaseId: PhaseId;
  phase: PhaseRecord | null;
  tape: Tape;
  folder: string;
  tools: ToolCall[];
  finalText: string;
  finalScrollTop: number;
  finalViewportLines: number;
  stats: { events: number; tokens: TokenCounts; cost?: number; durationMs?: number };
  toolCursor: number;
  toolScrollTop: number;
  inspectMode: boolean;
}): React.ReactElement {
  // Fixed-size scroll viewport. Cursor index drives both the highlighted
  // row in the list and the content of the detail panel below.
  const visibleStart = Math.max(0, Math.min(tools.length, toolScrollTop));
  const visibleEnd = Math.min(tools.length, visibleStart + TOOLS_VIEWPORT);
  const visibleTools = tools.slice(visibleStart, visibleEnd);
  const hiddenAbove = visibleStart;
  const hiddenBelow = Math.max(0, tools.length - visibleEnd);
  const clampedCursor = Math.max(0, Math.min(tools.length - 1, toolCursor));
  const selectedTool = tools[clampedCursor];
  const totalTokens =
    stats.tokens.input +
    stats.tokens.output +
    stats.tokens.cacheRead +
    stats.tokens.cacheCreation;
  const next = nextScreenFor(phaseId, tape, folder);
  const nextLabel = nextScreenLabel(phaseId, tape, folder);

  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Status header */}
      <Box marginBottom={1}>
        <Text color={code === 0 ? SUCCESS : ERROR} bold>
          {code === 0 ? "✓ done" : `✗ exited with code ${code}`}
        </Text>
        <Text color={MUTED}>
          {"   "}
          {agent.name}
          {"  ·  "}
          {stats.events} events
          {"  ·  "}
          {tools.length} tools
          {"  ·  "}
          {formatTokens(totalTokens)} tokens
          {stats.durationMs ? `  ·  ${formatDuration(stats.durationMs)}` : ""}
          {/* No dollar figure: Claude Code / Codex runs go through the user's
              subscription, not metered API billing, so a $ amount would be
              invented. Token count is real; dollars are not. */}
        </Text>
      </Box>

      {/* Final response from the agent. Border + chip color tracks whether
          the run succeeded — SUCCESS on a clean exit, ERROR otherwise. */}
      {finalText ? (
        <Box marginBottom={1}>
          <LabeledBox
            label={finalResponseLabel(finalText, finalScrollTop, finalViewportLines)}
            color={code === 0 ? SUCCESS : ERROR}
          >
            <MarkdownBody
              text={finalText}
              maxLines={finalViewportLines}
              scrollTop={finalScrollTop}
              color={code === 0 ? SUCCESS : ERROR}
            />
          </LabeledBox>
        </Box>
      ) : null}

      {/* Tool list — fixed-size scroll viewport. When NOT in inspect mode,
          the chip is neutral cyan and rows have no cursor highlight. When
          inspecting, the chip turns yellow (matches the detail panel below)
          and shows the cursor position. Pad to TOOLS_VIEWPORT rows so the
          box height never changes as the cursor moves through the calls. */}
      {tools.length > 0 ? (
        <Box marginBottom={1}>
          <LabeledBox
            label={
              inspectMode
                ? `inspecting tool calls  ·  ${clampedCursor + 1} / ${tools.length}`
                : `tool calls  ·  ${tools.length} total  ·  press t to inspect`
            }
            color={inspectMode ? CURRENT : STRUCTURE}
            paddingX={1}
          >
            {hiddenAbove > 0 ? (
              <Text color={MUTED}>
                {`  ↑ ${hiddenAbove} earlier call${hiddenAbove === 1 ? "" : "s"} above`}
              </Text>
            ) : (
              <Text color={MUTED}>{"  "}</Text>
            )}
            {Array.from({ length: TOOLS_VIEWPORT }).map((_, idx) => {
              const tool = visibleTools[idx];
              if (!tool) return <Text key={`pad-${idx}`}>{" "}</Text>;
              return (
                <ToolCallRow
                  key={tool.id}
                  call={tool}
                  active={inspectMode && visibleStart + idx === clampedCursor}
                />
              );
            })}
            {hiddenBelow > 0 ? (
              <Text color={MUTED}>
                {`  ↓ ${hiddenBelow} more call${hiddenBelow === 1 ? "" : "s"} below`}
              </Text>
            ) : (
              <Text color={MUTED}>{"  "}</Text>
            )}
          </LabeledBox>
        </Box>
      ) : null}

      {/* Inspect-mode instructions. Appears only when the user has engaged
          the tool inspector — keeps the default done view uncluttered while
          making the modal nature of inspect mode obvious. */}
      {inspectMode ? (
        <Box marginBottom={1}>
          <Text color={CURRENT} bold>▸ inspect mode  </Text>
          <Text color={MUTED}>
            {"use "}
          </Text>
          <Text color={STRUCTURE}>↑↓</Text>
          <Text color={MUTED}>{" to move, "}</Text>
          <Text color={STRUCTURE}>pgup/pgdn</Text>
          <Text color={MUTED}>{" to jump, "}</Text>
          <Text color={STRUCTURE}>t</Text>
          <Text color={MUTED}>{" or "}</Text>
          <Text color={STRUCTURE}>esc</Text>
          <Text color={MUTED}>{" to exit"}</Text>
        </Box>
      ) : null}

      {/* Detail panel for the cursor-selected tool call. Up/down arrows in
          the done state move the cursor; this card always reflects the
          currently-highlighted row. Shows the full input + result, not the
          row's truncated single-line preview. */}
      {selectedTool && inspectMode ? (
        <Box marginBottom={1}>
          <LabeledBox
            label={`detail · ${friendlyToolName(selectedTool.name)}`}
            color={CURRENT}
            paddingY={1}
          >
            <ToolDetailBody call={selectedTool} />
          </LabeledBox>
        </Box>
      ) : null}

      {/* Single routing line. KeyHints below carries the key labels; this
          line carries the routing information they don't — where enter
          actually takes you, or what "try again" does after a failure. */}
      <Box marginBottom={1}>
        <Text color={HEADING}>
          {code === 0
            ? `next · ${nextLabel}${next ? "" : " — back to the hub"}`
            : "next · enter resumes from where the agent stopped (no re-fetching)"}
        </Text>
      </Box>
    </Box>
  );
}

function ToolCallRow({
  call,
  active = false,
}: {
  call: ToolCall;
  /** True when this is the cursor-selected row in the done view. */
  active?: boolean;
}): React.ReactElement | null {
  // Guard: every now and then the agent stream emits a tool_start with no
  // name and no input. Skip those rows so they don't punch holes in the box.
  if (!call.name && !call.inputPreview) return null;

  const glyph = call.status === "ok" ? "✓" : call.status === "failed" ? "✗" : "⟳";
  const glyphColor =
    call.status === "ok" ? SUCCESS : call.status === "failed" ? ERROR : STRUCTURE;
  const friendlyName = friendlyToolName(call.name);

  // The "effect" string leads the row — what the tool produced, not what we
  // sent it. For successful calls that's the result preview; for failures
  // it's the cleaned-up error; for in-flight calls it's a status line.
  const effect = effectFor(call);
  // The input/path becomes a dim trailing chip — the where, not the what.
  const inputChip = call.inputPreview ? `  [${previewify(call.inputPreview, 40)}]` : "";

  return (
    <Box>
      <Text color={active ? CURRENT : undefined} bold={active}>
        {active ? "▸ " : "  "}
      </Text>
      <Text color={glyphColor} bold>
        {glyph}{" "}
      </Text>
      <Text color={active ? CURRENT : STRUCTURE} bold={active}>
        {friendlyName.padEnd(13, " ")}
      </Text>
      <Text color={call.status === "failed" ? ERROR : undefined}>
        {truncate(effect, 54)}
      </Text>
      {inputChip ? <Text color={MUTED}>{inputChip}</Text> : null}
    </Box>
  );
}

// Fixed counts so the detail panel always occupies the same vertical space.
// Switching between tool calls never causes the layout below the box to shift.
// Empty rows are rendered with a single space so Ink keeps them as real
// lines (and not collapsed whitespace).
const DETAIL_INPUT_LINES = 3;
const DETAIL_RESULT_LINES = 6;

/**
 * Detail panel body for the currently-selected tool call. Shows the full
 * input (path/URL/command/JSON object) and the un-truncated result preview.
 * The total rendered height is constant: status row + input header + 3 input
 * lines + result header + 6 result lines + spacers. Padding empty rows when
 * content is shorter keeps the surrounding layout (NextSteps, KeyHints)
 * anchored as the user moves the cursor through different tools.
 */
function ToolDetailBody({ call }: { call: ToolCall }): React.ReactElement {
  const statusColor =
    call.status === "ok" ? SUCCESS : call.status === "failed" ? ERROR : STRUCTURE;
  const rawInputLines = formatInputDetail(call.name, call.input);
  const inputLines = padLines(rawInputLines, DETAIL_INPUT_LINES);
  const inputTruncated = rawInputLines.length > DETAIL_INPUT_LINES;

  const resultText = call.resultPreview ? cleanResultPreview(call.resultPreview) : "";
  const rawResultLines = resultText ? resultText.split("\n") : [];
  const resultLines = padLines(rawResultLines, DETAIL_RESULT_LINES);
  const resultTruncated = rawResultLines.length > DETAIL_RESULT_LINES;

  return (
    <Box flexDirection="column">
      <Box>
        <Text color={statusColor} bold>
          {call.status === "ok" ? "✓" : call.status === "failed" ? "✗" : "⟳"}{" "}
        </Text>
        <Text bold>{friendlyToolName(call.name)}</Text>
        <Text color={MUTED}>  ·  {call.status}</Text>
      </Box>

      <Box flexDirection="column" marginTop={1}>
        <Text color={HEADING} bold>
          input
          {inputTruncated ? (
            <Text color={MUTED}>{` (showing first ${DETAIL_INPUT_LINES} of ${rawInputLines.length})`}</Text>
          ) : null}
        </Text>
        {inputLines.map((line, i) => (
          <Text key={`in-${i}`} color={MUTED}>{"  " + (line || " ")}</Text>
        ))}
      </Box>

      <Box flexDirection="column" marginTop={1}>
        <Text color={HEADING} bold>
          result
          {resultTruncated ? (
            <Text color={MUTED}>{` (showing first ${DETAIL_RESULT_LINES} of ${rawResultLines.length})`}</Text>
          ) : null}
        </Text>
        {resultLines.map((line, i) => (
          <Text key={`res-${i}`} color={MUTED}>{"  " + (line || " ")}</Text>
        ))}
      </Box>
    </Box>
  );
}

/**
 * Truncate `lines` to `target` rows, padding with empty strings when shorter.
 * Used to keep the detail panel's render height constant regardless of how
 * much content the selected tool produced.
 */
function padLines(lines: string[], target: number): string[] {
  if (lines.length >= target) return lines.slice(0, target);
  return [...lines, ...new Array<string>(target - lines.length).fill("")];
}

/**
 * Multi-line detail for a tool's input, formatted per tool type. Returns the
 * full path/URL/command (no truncation) plus secondary fields where they
 * exist. Falls back to a pretty-printed JSON dump for unknown shapes.
 */
function formatInputDetail(name: string, input: unknown): string[] {
  if (input == null) return [];
  // String inputs (e.g. a bare shell command Codex emits) still deserve to
  // show up in the detail panel — fall back to a single line keyed by the
  // tool's natural label rather than rendering nothing.
  if (typeof input === "string") {
    return input ? [`${name === "bash" ? "command" : "input"}: ${input}`] : [];
  }
  if (typeof input !== "object") return [];
  const o = input as Record<string, unknown>;
  const tool = name.replace(/^mcp__[^_]+__/, "");
  const lines: string[] = [];
  const fieldOrder = [
    ["url", "url"],
    ["file_path", "path"],
    ["path", "path"],
    ["query", "query"],
    ["pattern", "pattern"],
    ["command", "command"],
    ["prompt", "prompt"],
    ["description", "description"],
  ] as const;
  const known = new Set<string>();
  for (const [field, label] of fieldOrder) {
    const v = o[field];
    if (typeof v === "string" && v) {
      lines.push(`${label}: ${v}`);
      known.add(field);
    }
  }
  // Anything else in the input — fall back to a one-line JSON dump per key.
  // For TodoWrite specifically the `todos` array is the meaningful payload;
  // render it line-by-line so the curator can read the plan.
  if (tool === "TodoWrite" && Array.isArray(o["todos"])) {
    lines.push("todos:");
    for (const todo of o["todos"] as unknown[]) {
      const t = (todo ?? {}) as Record<string, unknown>;
      const status = String(t["status"] ?? "?");
      const content = String(t["content"] ?? "");
      const glyph = status === "completed" ? "✓" : status === "in_progress" ? "▸" : "○";
      lines.push(`  ${glyph} ${content}`);
    }
    known.add("todos");
  }
  // Any remaining fields go as one-liner key: value JSON-ish entries.
  for (const [k, v] of Object.entries(o)) {
    if (known.has(k)) continue;
    if (v == null) continue;
    const s = typeof v === "string" ? v : JSON.stringify(v);
    if (!s) continue;
    lines.push(`${k}: ${s.slice(0, 200)}`);
  }
  return lines;
}

/**
 * One-line "what happened" string for a tool call. Falls back through the
 * available signals so a row never goes empty:
 *  - result preview (post-call), with `<tool_use_error>` wrappers stripped
 *  - "fetching…" / "writing…" / etc. for in-flight calls
 *  - the raw input preview if nothing else is available
 */
function effectFor(call: ToolCall): string {
  if (call.status === "running") {
    return inFlightVerb(call.name);
  }
  if (call.resultPreview) {
    // First line only for the row display; the detail panel shows the full
    // multi-line preview when the row is selected.
    return firstLineOf(cleanResultPreview(call.resultPreview));
  }
  // Nothing reported — fall back to a tool-typed default so the row stays
  // informative rather than displaying an empty string.
  if (call.status === "ok") return defaultDoneFor(call.name);
  if (call.status === "failed") return "failed without an error message";
  return "";
}

function firstLineOf(s: string): string {
  const nl = s.indexOf("\n");
  return nl < 0 ? s : s.slice(0, nl);
}

/**
 * Strip Claude's `<tool_use_error>...</tool_use_error>` envelope and the
 * "File has not been read yet" preamble that surfaces frequently. Surfacing
 * the bare message gives the curator a fighting chance at understanding what
 * went wrong.
 */
function cleanResultPreview(s: string): string {
  let out = s.trim();
  // Strip XML-style envelope.
  const m = /<tool_use_error>\s*(.*?)\s*<\/tool_use_error>/is.exec(out);
  if (m && m[1]) out = m[1];
  // Strip common HTTP-error preambles that don't carry useful information
  // for someone watching the stream.
  out = out.replace(/^Error:\s+/i, "");
  return out;
}

/** Human-friendly title for a tool name. */
function friendlyToolName(raw: string): string {
  const stripped = raw.replace(/^mcp__[^_]+__/, "");
  switch (stripped) {
    case "WebSearch":
      return "Web search";
    case "WebFetch":
      return "Web fetch";
    case "Read":
      return "Read file";
    case "Write":
      return "Write file";
    case "Edit":
      return "Edit file";
    case "Glob":
      return "Find files";
    case "Grep":
      return "Search code";
    case "TodoWrite":
      return "Update plan";
    case "Bash":
      return "Shell";
    default:
      return stripped;
  }
}

function inFlightVerb(raw: string): string {
  const n = raw.replace(/^mcp__[^_]+__/, "");
  switch (n) {
    case "WebSearch":
      return "searching the web…";
    case "WebFetch":
      return "fetching…";
    case "Read":
      return "reading…";
    case "Write":
      return "writing…";
    case "Edit":
      return "editing…";
    case "Glob":
      return "globbing…";
    case "Grep":
      return "searching…";
    case "TodoWrite":
      return "updating the plan…";
    case "Bash":
      return "running…";
    default:
      return "working…";
  }
}

function defaultDoneFor(raw: string): string {
  const n = raw.replace(/^mcp__[^_]+__/, "");
  switch (n) {
    case "WebSearch":
      return "search complete";
    case "WebFetch":
      return "fetched";
    case "Read":
      return "read";
    case "Write":
      return "written";
    case "Edit":
      return "edited";
    case "Glob":
      return "matched";
    case "Grep":
      return "matched";
    case "TodoWrite":
      return "plan updated";
    case "Bash":
      return "executed";
    default:
      return "done";
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type InitialState = { stage: Stage; autoLaunch: AgentDescriptor | null };

function initialStage(skillPath: string | null): InitialState {
  if (!skillPath) return { stage: { kind: "missing_skill" }, autoLaunch: null };
  const agents = detectAgents();
  if (agents.length === 0) return { stage: { kind: "missing_agent" }, autoLaunch: null };

  // Resolution order is centralized in resolveConfiguredAgent:
  //   LINER_AGENT env > ~/.liner/config.yaml > single-installed > null.
  // If it returns a non-null agent, we auto-launch and skip the picker. If
  // it returns null (multiple agents, no config, no env hint), we render
  // the picker as before.
  const resolved = resolveConfiguredAgent(agents);
  if (resolved) {
    return {
      stage: {
        kind: "picking_agent",
        agents,
        selectedIndex: agents.indexOf(resolved),
      },
      autoLaunch: resolved,
    };
  }
  return {
    stage: { kind: "picking_agent", agents, selectedIndex: 0 },
    autoLaunch: null,
  };
}

function subtitleFor(stage: Stage, skillPath: string | null): string {
  if (stage.kind === "running") return `agent: ${stage.agent.name}`;
  if (stage.kind === "done") return `agent: ${stage.agent.name}  ·  exit ${stage.code}`;
  if (stage.kind === "picking_agent") return skillPath ? `skill: ${skillPath}` : "";
  return "";
}

function hintsFor(
  stage: Stage,
  inspectMode: boolean = false,
): Array<{ key: string; label: string }> {
  if (stage.kind === "picking_agent") {
    return [
      { key: "↑↓", label: "pick" },
      { key: "enter", label: "launch" },
      { key: "esc", label: "cancel" },
      { key: "q", label: "quit" },
    ];
  }
  if (stage.kind === "running") {
    return [{ key: "esc", label: "cancel run" }];
  }
  if (stage.kind === "done") {
    // Hints split by mode: when inspecting tool calls, only the scroll keys
    // and the exit keys matter; enter/b are intentionally locked so the user
    // doesn't accidentally leave inspect mode by advancing.
    if (inspectMode) {
      return [
        { key: "↑↓", label: "move cursor" },
        { key: "pgup/pgdn", label: "jump page" },
        { key: "t", label: "exit inspect" },
        { key: "esc", label: "exit inspect" },
      ];
    }
    const failed = stage.code !== 0;
    return [
      { key: "↑↓", label: "scroll response" },
      { key: "pgup/pgdn", label: "jump" },
      { key: "enter", label: failed ? "try again" : "continue" },
      { key: "t", label: "inspect tool calls" },
      { key: "b", label: "back to hub" },
      { key: "q", label: "quit" },
    ];
  }
  return [{ key: "b", label: "back" }];
}

/**
 * After a phase run completes, what's the obvious next screen? Returns null
 * when we should just go back to the hub.
 *
 * Mode-aware: quick mode skips the gate review screens (per CURATION.md v2.0
 * — "Quick mode defaults them to continue"). We auto-accept the gate state
 * file as a side effect here so the hub's progress strip still shows the
 * gate as ✓ complete.
 */
function nextScreenFor(phaseId: PhaseId, tape: Tape, folder: string): Screen | null {
  const isQuick = (tape.mode ?? "quick") !== "methodology";
  const project = projectFolder(folder);
  switch (phaseId) {
    case "framing":
      if (isQuick) {
        // Quick mode skips the gate. Auto-accept gate0 + advance the cursor
        // so the progress strip stays consistent (no skipped step).
        const state = readGateState(project);
        writeGateState(project, { ...state, gate0Accepted: true });
        markPhaseComplete(folder, "gate0");
        return { kind: "phaseRunner", tape, folder, phaseId: "candidates" };
      }
      return { kind: "gate0", tape, folder };
    case "candidates":
      if (isQuick) {
        // Auto-accept Gate 1 — both the legacy gate-state file (so other
        // surfaces still recognise it as accepted) and the progress cursor.
        const state = readGateState(project);
        writeGateState(project, { ...state, gate1Accepted: true });
        markPhaseComplete(folder, "gate1");
        return { kind: "phaseRunner", tape, folder, phaseId: "evaluation" };
      }
      return { kind: "gate1", tape, folder };
    case "evaluation":
      return { kind: "phaseRunner", tape, folder, phaseId: "quality" };
    case "quality":
      if (isQuick) {
        const state = readGateState(project);
        writeGateState(project, { ...state, gate2Accepted: true });
        markPhaseComplete(folder, "gate2");
        return { kind: "phaseRunner", tape, folder, phaseId: "synthesis" };
      }
      return { kind: "gate2", tape, folder };
    case "synthesis":
      return { kind: "phaseRunner", tape, folder, phaseId: "assembly" };
    case "assembly":
      // Handled in the .then() — auto-navigates to tape draft. If somehow we
      // land on done without a draft, go back to the hub.
      return null;
    default:
      return null;
  }
}

function nextScreenLabel(phaseId: PhaseId, tape: Tape, _folder: string): string {
  const isQuick = (tape.mode ?? "quick") !== "methodology";
  switch (phaseId) {
    case "framing":
      return isQuick
        ? "continue to Phase 2 — Candidate discovery (framing auto-confirmed in quick mode)"
        : "continue to Confirm framing — review the knowledge map";
    case "candidates":
      return isQuick
        ? "continue to Phase 3 — Evaluation (candidates auto-confirmed in quick mode)"
        : "continue to Confirm candidates — review the long-list";
    case "evaluation":
      return "continue to Phase 4 — Quality checks";
    case "quality":
      return isQuick
        ? "continue to Phase 5 — Synthesis (evaluation auto-confirmed in quick mode)"
        : "continue to Confirm evaluation — review keep-list + checks";
    case "synthesis":
      return "continue to Phase 6 — Assembly";
    case "assembly":
      return "review the proposed sources";
    default:
      return "back to the hub";
  }
}

function summarizeInput(input: unknown): string {
  if (!input || typeof input !== "object") return "";
  const o = input as Record<string, unknown>;
  if (typeof o["file_path"] === "string") return o["file_path"] as string;
  if (typeof o["path"] === "string") return o["path"] as string;
  if (typeof o["url"] === "string") return o["url"] as string;
  if (typeof o["pattern"] === "string") return o["pattern"] as string;
  if (typeof o["command"] === "string") return o["command"] as string;
  if (typeof o["query"] === "string") return o["query"] as string;
  try {
    return JSON.stringify(o).slice(0, 80);
  } catch {
    return "";
  }
}

function formatTokens(n: number): string {
  if (n < 1_000) return String(n);
  if (n < 1_000_000) return `${(n / 1_000).toFixed(1)}k`;
  return `${(n / 1_000_000).toFixed(2)}M`;
}

function formatDuration(ms: number): string {
  if (ms < 60_000) return `${(ms / 1000).toFixed(0)}s`;
  const minutes = Math.floor(ms / 60_000);
  const seconds = Math.floor((ms % 60_000) / 1000);
  return `${minutes}m${seconds.toString().padStart(2, "0")}s`;
}

function zeroTokens(): TokenCounts {
  return { input: 0, output: 0, cacheRead: 0, cacheCreation: 0 };
}

function finalResponseViewportLines(termRows: number, inspectMode: boolean): number {
  if (inspectMode) return FINAL_RESPONSE_INSPECT_VIEWPORT;
  // Keep enough space for status, tool calls, next-action line, and key hints.
  // The body lines still may wrap, so this is intentionally conservative.
  return Math.max(3, Math.min(FINAL_RESPONSE_VIEWPORT, termRows - 34));
}

function finalResponseLabel(text: string, scrollTop: number, viewportLines: number): string {
  const total = text.split("\n").length;
  const maxScroll = Math.max(0, total - viewportLines);
  const start = Math.min(Math.max(0, scrollTop), maxScroll);
  const end = Math.min(total, start + viewportLines);
  return total > viewportLines
    ? `final response  ·  lines ${start + 1}-${end} / ${total}`
    : "final response";
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}

/**
 * Right-anchored truncation for file paths. The basename is the load-bearing
 * part of any path the agent reads/writes; keeping it visible is much more
 * useful than seeing "/Users/me/Docu…" without the actual filename.
 */
function truncatePath(s: string, n: number): string {
  if (s.length <= n) return s;
  return "…" + s.slice(-(n - 1));
}

/**
 * Heuristic: looks like a filesystem path? Used to pick path-style vs.
 * plain truncation for the tool-call preview line.
 */
function looksLikePath(s: string): boolean {
  return s.startsWith("/") || s.startsWith("./") || s.startsWith("../") ||
    /^[A-Za-z]:[\\/]/.test(s) ||
    (s.includes("/") && !s.includes(" ") && !s.includes("://"));
}

/**
 * Truncate a value smartly: file paths keep their basename, URLs and other
 * strings get standard right-trim ellipsis.
 */
function previewify(s: string, n: number): string {
  if (!s) return "";
  if (looksLikePath(s)) return truncatePath(s, n);
  return truncate(s, n);
}

export type _ForReExport = PhaseRecord; // keep PhaseRecord import alive

// ---------------------------------------------------------------------------
// Markdown rendering for the agent's final response.
//
// The agent produces a small markdown blob with `**heading:**` lines, `-` bullets,
// and inline `` `code` `` / `[bracketed paths]`. Rendering it raw shows the
// asterisks as literal characters, which looks unfinished. This renderer:
//
//   - Strips ** and renders the bold runs with weight + a heading color for
//     start-of-line headings (the ones ending with `:`).
//   - Renders `-` / `*` bullets with a cyan `•` glyph and indents the rest.
//   - Renders ``backtick code`` runs in cyan.
//   - Detects trailing `[paths/with/slashes]` and dims them so the verb-phrase
//     leads.
//
// Intentionally not a full markdown engine — just enough to make the agent's
// output legible. If the format grows beyond this, swap in `marked-terminal`
// or `ink-markdown`.
// ---------------------------------------------------------------------------

function MarkdownBody({
  text,
  maxLines = 18,
  scrollTop = 0,
  color,
}: {
  text: string;
  maxLines?: number;
  scrollTop?: number;
  /** Body-text color — set to the box color so prose matches the box. The
      bullet markers and inline code stay STRUCTURE (orange) regardless. */
  color?: string;
}): React.ReactElement {
  const lines = text.split("\n");
  const maxScroll = Math.max(0, lines.length - maxLines);
  const start = Math.min(Math.max(0, scrollTop), maxScroll);
  const shown = lines.slice(start, start + maxLines);
  const hiddenAbove = start;
  const hiddenBelow = Math.max(0, lines.length - (start + shown.length));
  return (
    <Box flexDirection="column">
      {hiddenAbove > 0 ? (
        <Text color={MUTED}>
          ↑ {hiddenAbove} earlier line{hiddenAbove === 1 ? "" : "s"}
        </Text>
      ) : null}
      {shown.map((line, i) => (
        <MarkdownLine key={start + i} line={line} color={color} />
      ))}
      {hiddenBelow > 0 ? (
        <Box marginTop={1}>
          <Text color={MUTED}>
            ↓ {hiddenBelow} more line{hiddenBelow === 1 ? "" : "s"} · use ↑↓ or pgup/pgdn
          </Text>
        </Box>
      ) : null}
    </Box>
  );
}

function MarkdownLine({ line, color }: { line: string; color?: string }): React.ReactElement {
  // Empty line → one-row spacer (keep paragraphs visible).
  if (!line.trim()) {
    return <Text> </Text>;
  }

  // Bullet line: -  / *  prefix. The marker stays orange (STRUCTURE); the
  // rest of the bullet text takes the body color.
  const bullet = /^(\s*)([-*])\s+(.*)$/.exec(line);
  if (bullet) {
    const [, indent, , rest] = bullet;
    return (
      <Box>
        <Text>{indent}</Text>
        <Text color={STRUCTURE}>• </Text>
        <InlineSegments text={rest ?? ""} color={color} />
      </Box>
    );
  }

  return (
    <Box>
      <InlineSegments text={line} headingDetect color={color} />
    </Box>
  );
}

type Seg =
  | { kind: "plain"; text: string }
  | { kind: "bold"; text: string }
  | { kind: "code"; text: string };

/**
 * Tokenize a single line by `**bold**` and `` `code` `` runs.
 * `**` and backticks themselves are dropped. Unclosed markers fall back to
 * plain text so we never lose content.
 */
function tokenize(s: string): Seg[] {
  const out: Seg[] = [];
  let i = 0;
  let buf = "";
  const flush = (): void => {
    if (buf) {
      out.push({ kind: "plain", text: buf });
      buf = "";
    }
  };
  while (i < s.length) {
    if (s[i] === "*" && s[i + 1] === "*") {
      const end = s.indexOf("**", i + 2);
      if (end > 0) {
        flush();
        out.push({ kind: "bold", text: s.slice(i + 2, end) });
        i = end + 2;
        continue;
      }
    }
    if (s[i] === "`") {
      const end = s.indexOf("`", i + 1);
      if (end > 0) {
        flush();
        out.push({ kind: "code", text: s.slice(i + 1, end) });
        i = end + 1;
        continue;
      }
    }
    buf += s[i];
    i++;
  }
  flush();
  return out;
}

/**
 * Render the inline segments of a line. If `headingDetect` is set and the
 * first segment is a bold-ending-with-`:`, treat it as a section heading and
 * give the bold run a cyan tint — so the agent's `**Wrote:**` style ledes
 * carry the eye.
 */
function InlineSegments({
  text,
  headingDetect = false,
  color,
}: {
  text: string;
  headingDetect?: boolean;
  color?: string;
}): React.ReactElement {
  const segs = tokenize(text);
  const firstBold = segs[0]?.kind === "bold" && segs[0].text.trim().endsWith(":");
  return (
    <>
      {segs.map((seg, i) => renderSeg(seg, i, headingDetect && firstBold && i === 0, color))}
    </>
  );
}

function renderSeg(
  seg: Seg,
  key: number,
  isHeading: boolean,
  color?: string,
): React.ReactElement {
  if (seg.kind === "code") {
    // Inline code keeps the structural orange regardless of body color.
    return (
      <Text key={key} color={STRUCTURE}>
        {seg.text}
      </Text>
    );
  }
  if (seg.kind === "bold") {
    return (
      <Text key={key} bold color={color}>
        {seg.text}
      </Text>
    );
  }
  // Plain text — dim `[bracketed paths]` so they recede.
  return renderPlainWithBrackets(seg.text, key, color);
}

/**
 * Plain text, with `[paths/like/this.md]` substrings rendered dim. The
 * heuristic: a `[...]` run that contains at least one `/` or `.` is treated
 * as a path/citation. Anything else is rendered as-is.
 */
function renderPlainWithBrackets(text: string, key: number, color?: string): React.ReactElement {
  const parts: React.ReactElement[] = [];
  let i = 0;
  let cursor = 0;
  let k = 0;
  while (i < text.length) {
    if (text[i] === "[") {
      const close = text.indexOf("]", i + 1);
      if (close > 0) {
        const inside = text.slice(i + 1, close);
        if (inside.includes("/") || inside.includes(".")) {
          if (i > cursor) {
            parts.push(<Text key={`${key}-${k++}`} color={color}>{text.slice(cursor, i)}</Text>);
          }
          parts.push(
            <Text key={`${key}-${k++}`} color={MUTED}>
              [{inside}]
            </Text>,
          );
          i = close + 1;
          cursor = i;
          continue;
        }
      }
    }
    i++;
  }
  if (cursor < text.length) {
    parts.push(<Text key={`${key}-${k++}`} color={color}>{text.slice(cursor)}</Text>);
  }
  return <React.Fragment key={key}>{parts}</React.Fragment>;
}
