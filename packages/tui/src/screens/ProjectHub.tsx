import React, { useMemo, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdin } from "ink";
import { spawn } from "node:child_process";
import { existsSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const TAPE_DRAFT_REL = "working/07-tape-draft.yaml";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { useApp } from "../store.js";
import { projectFolder, writeTape } from "../yaml-io.js";
import {
  computePhases,
  everythingComplete,
  statusGlyph,
  statusColor,
  statusLabel,
  type PhaseRecord,
} from "../phases.js";
import {
  migrateProgress,
  progressFileExists,
  readProgress,
  TOTAL_STEPS,
  type Progress,
} from "../progress.js";
import { runShare } from "../ipc.js";
import { openFolder } from "../open-folder.js";
import { detectAgents } from "../agents/detect.js";
import { resolveEditor } from "../editor.js";
import { MUTED, ON_FILL, CURRENT, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import type { Tape, Mode } from "../types.js";

const PREVIEW_LINES = 14;

export function ProjectHub({
  tape: initialTape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { setRawMode } = useStdin();
  const project = useMemo(() => projectFolder(folder), [folder]);
  const [tape, setTape] = useState<Tape>(initialTape);
  const [refreshTick, setRefreshTick] = useState(0);

  // Read or migrate the progress cursor on mount. Migration runs once for
  // projects that pre-date `.liner-progress.json` (e.g. created with an
  // older TUI build); it walks the phase artifacts and sets step to the
  // longest consecutive "done" prefix.
  const [progress, setProgress] = useState<Progress>(() => {
    if (progressFileExists(folder)) return readProgress(folder);
    return migrateProgress(folder, initialTape);
  });
  const compiledArtifactCurrent = useMemo(
    () => isCompiledArtifactCurrent(project),
    [project, refreshTick],
  );
  const displayProgress = useMemo(
    () =>
      compiledArtifactCurrent && progress.step >= TOTAL_STEPS - 1
        ? { ...progress, step: TOTAL_STEPS }
        : progress,
    [compiledArtifactCurrent, progress],
  );
  const compileWarningCount = useMemo(
    () => (compiledArtifactCurrent ? countCompilationNotes(project.mixtapePath) : 0),
    [compiledArtifactCurrent, project.mixtapePath, refreshTick],
  );

  const phases = useMemo(
    () => computePhases(project, tape, displayProgress),
    // refreshTick is intentionally in the dep array so file changes re-derive
    // status. eslint-disable-next-line react-hooks/exhaustive-deps
    [project, tape, displayProgress, refreshTick],
  );

  // Current index IS the progress cursor — clamped so the all-done state
  // (step >= TOTAL_STEPS) still has a sensible "last visible" phase to focus.
  const currentIndex = Math.min(displayProgress.step, phases.length - 1);

  // Single summary view: a status box + the full phase checklist with a
  // detail pane, always shown. `selected` is the highlighted phase; it starts
  // on the current (recommended-next) phase.
  const [selected, setSelected] = useState(currentIndex);

  // Detect once at mount whether any agent CLI is installed. Drives whether
  // enter on agent-eligible phases runs the agent or falls back to $EDITOR.
  const hasAgent = useMemo(() => detectAgents().length > 0, []);

  const focusedPhase = phases[selected];
  const focusedIndex = selected;
  const allDone = everythingComplete(displayProgress);
  const notStarted = displayProgress.step === 0;
  const readyToCompile = !allDone && displayProgress.step === TOTAL_STEPS - 1;
  const nextLabel = phases[currentIndex]?.label ?? "Framing";

  function refresh(): void {
    // Re-read progress in case another screen advanced the cursor (gate
    // accept, phase agent run, compile success). Without this the hub
    // would show stale status until the user navigated away and back.
    setProgress(readProgress(folder));
    setRefreshTick((t) => t + 1);
  }

  function openInEditor(relPath: string, opts: { template?: string } = {}): void {
    const full = join(folder, relPath);
    if (!existsSync(full) && opts.template) {
      writeFileSync(full, opts.template, "utf8");
    }
    const editor = resolveEditor();
    setRawMode(false);
    const child = spawn(editor, [full], { stdio: "inherit" });
    child.on("close", () => {
      setRawMode(true);
      refresh();
      app.setNotification({ kind: "info", message: `${relPath} updated.` });
    });
    child.on("error", (e: Error) => {
      setRawMode(true);
      app.setNotification({
        kind: "error",
        message: `Could not launch ${editor}: ${e.message}`,
      });
    });
  }

  /**
   * The primary action for a phase, invoked by Enter. Designed so the user
   * almost never has to think — the most likely thing happens.
   *
   *   - Agent-eligible phases: run the agent (or fall back to $EDITOR if no
   *     agent CLI is installed).
   *   - Gates: open the review screen (blocked if prerequisites missing).
   *   - Assembly: review the pending draft if one exists; otherwise run agent.
   *   - Compile: dedicated screen.
   */
  function enterPhase(phase: PhaseRecord): void {
    switch (phase.id) {
      case "framing":
      case "candidates":
      case "evaluation":
      case "quality":
      case "synthesis": {
        runAgentOrFallback(phase);
        return;
      }
      case "assembly": {
        if (existsSync(join(folder, TAPE_DRAFT_REL))) {
          app.navigate({ kind: "tapeDraft", tape, folder });
        } else {
          runAgentOrFallback(phase);
        }
        return;
      }
      case "gate0": {
        if (phase.status === "blocked") {
          app.setNotification({
            kind: "error",
            message: "Confirm framing needs Phase 1 (knowledge map) first.",
          });
          return;
        }
        app.navigate({ kind: "gate0", tape, folder });
        return;
      }
      case "gate1": {
        if (phase.status === "blocked") {
          app.setNotification({
            kind: "error",
            message: "Confirm candidates needs Phase 2 (long-list) first.",
          });
          return;
        }
        app.navigate({ kind: "gate1", tape, folder });
        return;
      }
      case "gate2": {
        if (phase.status === "blocked") {
          app.setNotification({
            kind: "error",
            message: "Confirm evaluation needs Phases 4 and 5 first.",
          });
          return;
        }
        app.navigate({ kind: "gate2", tape, folder });
        return;
      }
      case "compile": {
        if (tape.sources.length === 0) {
          app.setNotification({
            kind: "error",
            message: "Add sources in Phase 7 before compiling.",
          });
          return;
        }
        app.navigate({
          kind: "compile",
          tape,
          folder,
          showExisting: compiledArtifactCurrent && existsSync(project.mixtapePath),
        });
        return;
      }
    }
  }

  /**
   * Run the configured agent for `phase` if one is installed, otherwise drop
   * the curator into $EDITOR on the phase's artifact (or the source editor
   * for assembly). Either way Enter "does something useful" — the agent vs.
   * editor decision is hidden from the keymap.
   */
  function runAgentOrFallback(phase: PhaseRecord): void {
    if (hasAgent) {
      app.navigate({ kind: "phaseRunner", tape, folder, phaseId: phase.id });
      return;
    }
    app.setNotification({
      kind: "info",
      message: "No agent installed — opening the artifact in $EDITOR.",
    });
    editPhase(phase);
  }

  /**
   * Explicit "edit by hand" path — Enter's escape hatch, bound to `e`.
   * For assembly, "edit" means the source editor (the existing TapeEditor
   * surface) rather than raw tape.yaml in $EDITOR.
   */
  function editPhase(phase: PhaseRecord): void {
    if (phase.id === "assembly") {
      app.navigate({ kind: "editor", tape, folder });
      return;
    }
    if (phase.artifact) {
      openInEditor(phase.artifact);
      return;
    }
    app.setNotification({
      kind: "error",
      message: `Phase ${phase.number} has no artifact to edit.`,
    });
  }

  function share(): void {
    void runShare(folder)
      .then((archivePath) => {
        app.setNotification({ kind: "info", message: `Shared → ${archivePath}` });
      })
      .catch((e: Error) => {
        app.setNotification({ kind: "error", message: e.message });
      });
  }

  function actOnFocused(): void {
    const phase = focusedPhase;
    if (!phase) return;
    // In TOC mode, block entering anything past the current phase — even if
    // its artifact happens to exist from a prior out-of-order run. The
    // methodology is linear; jumping ahead leaves earlier phases empty and
    // confuses the status display.
    if (focusedIndex > currentIndex) {
      app.setNotification({
        kind: "error",
        message: `Finish ${phases[currentIndex]?.label ?? "the current phase"} first.`,
      });
      return;
    }
    enterPhase(phase);
  }

  useInput((input, key) => {
    if (key.upArrow || input === "k") {
      setSelected((s) => Math.max(0, s - 1));
      return;
    }
    if (key.downArrow || input === "j") {
      setSelected((s) => Math.min(phases.length - 1, s + 1));
      return;
    }
    if (key.return) {
      actOnFocused();
    } else if (input === "e") {
      if (focusedPhase) editPhase(focusedPhase);
    } else if (input === "s") {
      share();
    } else if (input === "p") {
      app.navigate({ kind: "manifest", tape, folder });
    } else if (input === "o") {
      openFolder(folder);
    } else if (key.escape) {
      void app.refreshProjects();
      app.navigate({ kind: "browser" });
    } else if (input === "q") {
      ink.exit();
    }
  });

  return (
    <Box flexDirection="column">
      <Header
        title={tape.title || "untitled mixtape"}
        subtitle={`${folder}${tape.jtbd ? `  ·  ${truncate(tape.jtbd, 60)}` : ""}`}
      />

      <StatusBox
        allDone={allDone}
        notStarted={notStarted}
        readyToCompile={readyToCompile}
        nextLabel={nextLabel}
        sourceCount={tape.sources.length}
        warningCount={compileWarningCount}
      />

      <TocView
        phases={phases}
        selected={selected}
        currentIndex={currentIndex}
        folder={folder}
        tape={tape}
        hasAgent={hasAgent}
      />

      <ProgressBar phases={phases} currentIndex={currentIndex} selected={selected} />

      {app.notification ? (
        <Box marginTop={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      {/* "What Enter does" call-to-action, in HEADING — the one loud line.
          The navbar below stays a uniform dim reference. */}
      <Box marginTop={1}>
        <Text color={HEADING}>
          {allDone
            ? "next · it's ready — share it, open the folder, or revisit any phase"
            : readyToCompile
              ? "done · final mix assembled — press enter to compile MIXTAPE.md"
            : focusedIndex > currentIndex
              ? `next · finish ${nextLabel} first`
              : `next · ${focusedPhase ? primaryActionLabel(focusedPhase, hasAgent, folder) : "select a phase"}`}
        </Text>
      </Box>

      <Box marginTop={0}>
        <KeyHints
          hints={[
            { key: "↑↓", label: "select phase" },
            { key: "enter", label: hasAgent ? "run selected" : "edit selected" },
            { key: "e", label: "edit by hand" },
            { key: "p", label: "process" },
            { key: "s", label: "share" },
            { key: "o", label: "open folder" },
            { key: "esc", label: "back" },
            { key: "q", label: "quit" },
          ]}
        />
      </Box>
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Status box — the one-glance "where is this mixtape" summary at the top.
// ---------------------------------------------------------------------------

function StatusBox({
  allDone,
  notStarted,
  readyToCompile,
  nextLabel,
  sourceCount,
  warningCount,
}: {
  allDone: boolean;
  notStarted: boolean;
  readyToCompile: boolean;
  nextLabel: string;
  sourceCount: number;
  warningCount: number;
}): React.ReactElement {
  // Compiled (allDone) → green "ready to use"; just-started → neutral;
  // mid-flow → yellow "in progress, next: …". The checklist below carries the
  // per-phase detail; this box answers "is it done?" at a glance.
  const { color, glyph, line, sub } = allDone
    ? {
        color: SUCCESS,
        glyph: "✓",
        line:
          warningCount > 0
            ? "Ready to use — compiled with warnings"
            : "Ready to use — compiled",
        sub: `${sourceCount} source${sourceCount === 1 ? "" : "s"} · MIXTAPE.md written${
          warningCount > 0
            ? ` · ${warningCount} warning${warningCount === 1 ? "" : "s"}`
            : ""
        }`,
      }
    : readyToCompile
      ? {
          color: SUCCESS,
          glyph: "✓",
          line: "Methodology complete — ready to compile",
          sub: `final mix assembled · ${sourceCount} source${sourceCount === 1 ? "" : "s"} on tape`,
        }
    : notStarted
      ? {
          color: STRUCTURE,
          glyph: "○",
          line: "Not started",
          sub: `begin with ${nextLabel}`,
        }
      : {
          color: CURRENT,
          glyph: "◐",
          line: "In progress",
          sub: `next: ${nextLabel}`,
        };
  return (
    <Box marginBottom={1}>
      <LabeledBox label="status" color={color}>
        <Text color={color} bold>
          {glyph} {line}
        </Text>
        <Text color={MUTED}>{sub}</Text>
      </LabeledBox>
    </Box>
  );
}

/**
 * Render a phase's sub-step checklist. Done steps get a green ✓ and stay
 * fully colored; pending steps get a dim ○ and lighter text so the eye lands
 * on what's already accomplished. Hints render dim after the main text.
 */
function SubStepList({ steps }: { steps: import("../phases.js").SubStep[] }): React.ReactElement {
  return (
    <Box flexDirection="column">
      {steps.map((step, i) => (
        <Box key={i}>
          <Text color={step.done ? SUCCESS : STRUCTURE} bold={step.done}>
            {step.done ? "✓" : "○"}
          </Text>
          <Text color={!step.done ? MUTED : undefined}>{` ${step.text}`}</Text>
          {step.hint ? (
            <Text color={MUTED}>{`  ·  ${step.hint}`}</Text>
          ) : null}
        </Box>
      ))}
    </Box>
  );
}

// ---------------------------------------------------------------------------
// TOC view — full phase list. Behind `l`.
// ---------------------------------------------------------------------------

function TocView({
  phases,
  selected,
  currentIndex,
  folder,
  tape,
  hasAgent,
}: {
  phases: PhaseRecord[];
  selected: number;
  currentIndex: number;
  folder: string;
  tape: Tape;
  hasAgent: boolean;
}): React.ReactElement {
  const selectedPhase = phases[selected];
  return (
    <Box flexDirection="row" width="100%" marginBottom={1}>
      <Box flexDirection="column" width="38%" minWidth={26} flexShrink={0}>
        {phases.map((phase, i) => (
          <PhaseRow
            key={phase.id}
            phase={phase}
            selected={i === selected}
            future={i > currentIndex}
          />
        ))}
      </Box>
      <Box width={2} flexShrink={0}>
        <Text color={MUTED}>│</Text>
      </Box>
      <Box flexDirection="column" flexGrow={1} minWidth={20}>
        {selectedPhase ? (
          <PhaseDetail
            phase={selectedPhase}
            folder={folder}
            tape={tape}
            hasAgent={hasAgent}
          />
        ) : null}
      </Box>
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Progress bar — dense single-line summary.
// ---------------------------------------------------------------------------

function ProgressBar({
  phases,
  currentIndex,
  selected,
}: {
  phases: PhaseRecord[];
  currentIndex: number;
  selected: number | null;
}): React.ReactElement {
  return (
    <Box flexWrap="wrap" columnGap={1}>
      <Text color={MUTED}>progress: </Text>
      {phases.map((p, i) => {
        const isCurrent = i === currentIndex;
        const isSelected = selected != null && i === selected;
        const isPast = p.status === "complete";
        const color = isPast ? SUCCESS : isCurrent ? STRUCTURE : undefined;
        return (
          <Text
            key={p.id}
            backgroundColor={isSelected ? STRUCTURE : undefined}
            color={isSelected ? ON_FILL : !isPast && !isCurrent ? MUTED : color}
            bold={isCurrent || isSelected}
          >
            {phaseShortLabel(p)}
            {statusGlyph(p.status)}
          </Text>
        );
      })}
    </Box>
  );
}

function phaseShortLabel(phase: PhaseRecord): string {
  // For numeric phases, use the number; for special phases (gates, compile),
  // the `number` field is already a single glyph (↳, →) — pass it through.
  return phase.number;
}

/**
 * Long-form label for the focus card chip + TOC detail header. Numbered phases
 * read "Phase N — Label"; gates and compile (whose `number` is a glyph like
 * `↳` or `→`) read just the label — no "Phase glyph —" awkwardness.
 */
function chipLabelFor(phase: PhaseRecord): string {
  if (/^\d/.test(phase.number)) return `Phase ${phase.number} — ${phase.label}`;
  return phase.label;
}

function PhaseRow({
  phase,
  selected,
  future = false,
}: {
  phase: PhaseRecord;
  selected: boolean;
  /** Locked-out future phase — render dim and treat as blocked on action. */
  future?: boolean;
}): React.ReactElement {
  const statusFg = statusColor(phase.status);
  if (selected) {
    return (
      <Box>
        <Text backgroundColor={STRUCTURE} color={ON_FILL} bold>
          {" ▶ "}
        </Text>
        <Text backgroundColor={STRUCTURE} color={statusFg ?? ON_FILL} bold>
          {statusGlyph(phase.status)}{" "}
        </Text>
        <Text backgroundColor={STRUCTURE} color={ON_FILL}>
          {phase.number.padEnd(7, " ")}
        </Text>
        <Text backgroundColor={STRUCTURE} color={ON_FILL} bold>
          {truncate(phase.label, 22)}
          {future ? " 🔒" : ""}
          {"  "}
        </Text>
      </Box>
    );
  }
  return (
    <Box>
      <Text>{"   "}</Text>
      <Text color={future ? MUTED : statusFg}>
        {statusGlyph(phase.status)}{" "}
      </Text>
      <Text color={MUTED}>{phase.number.padEnd(7, " ")}</Text>
      <Text color={future ? MUTED : undefined}>{truncate(phase.label, 22)}</Text>
    </Box>
  );
}

function PhaseDetail({
  phase,
  folder,
  tape,
  hasAgent,
}: {
  phase: PhaseRecord;
  folder: string;
  tape: Tape;
  hasAgent: boolean;
}): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Box>
        <Text bold>{chipLabelFor(phase)}</Text>
        <Text>  </Text>
        <Text color={statusColor(phase.status)}>{statusGlyph(phase.status)} {statusLabel(phase.status)}</Text>
      </Box>
      <Box marginTop={1}>
        <Text>{phase.summary}</Text>
      </Box>
      {phase.subSteps && phase.subSteps.length > 0 ? (
        <Box flexDirection="column" marginTop={1}>
          <SubStepList steps={phase.subSteps} />
        </Box>
      ) : (
        <Box marginTop={1}>
          <Text color={MUTED}>{phase.detail}</Text>
        </Box>
      )}

      {phase.artifact ? (
        <Box flexDirection="column" marginTop={1}>
          <Text color={MUTED}>artifact:</Text>
          <Text color={MUTED}>  {phase.artifact}</Text>
          <ArtifactPreview folder={folder} relPath={phase.artifact} />
        </Box>
      ) : null}

      {phase.id === "assembly" && tape.sources.length > 0 ? (
        <Box flexDirection="column" marginTop={1}>
          <Text color={MUTED}>sources on tape ({tape.sources.length}):</Text>
          {tape.sources.slice(0, 6).map((s, i) => {
            const label = s.type === "local_file" ? s.citation || s.path || "" : s.url;
            return (
              <Text key={i} color={MUTED}>
                  {i + 1}. {truncate(label, 60)}
              </Text>
            );
          })}
          {tape.sources.length > 6 ? (
            <Text color={MUTED}>  … and {tape.sources.length - 6} more</Text>
          ) : null}
        </Box>
      ) : null}

      {phase.id === "assembly" && existsSync(join(folder, TAPE_DRAFT_REL)) ? (
        <Box marginTop={1}>
          <Text color={WARNING} bold>
            ⚠ draft pending —{" "}
          </Text>
          <Text>
            an agent run wrote{" "}
            <Text color={STRUCTURE}>{TAPE_DRAFT_REL}</Text>. Press{" "}
            <Text color={STRUCTURE}>enter</Text> to review.
          </Text>
        </Box>
      ) : null}

      <Box marginTop={1} flexDirection="column">
        <Box>
          <Text color={STRUCTURE}>enter</Text>
          <Text>  </Text>
          <Text>{primaryActionLabel(phase, hasAgent, folder)}</Text>
        </Box>
        {phase.artifact && phase.id !== "compile" ? (
          <Box>
            <Text color={STRUCTURE}>e    </Text>
            <Text>  edit by hand ({phase.artifact} in $EDITOR)</Text>
          </Box>
        ) : null}
      </Box>
    </Box>
  );
}

function ArtifactPreview({
  folder,
  relPath,
}: {
  folder: string;
  relPath: string;
}): React.ReactElement | null {
  const full = join(folder, relPath);
  if (!existsSync(full)) {
    return (
      <Box marginTop={1}>
        <Text color={MUTED}>  (file not on disk yet)</Text>
      </Box>
    );
  }
  let text: string;
  try {
    text = readFileSync(full, "utf8");
  } catch {
    return null;
  }
  const lines = text.split("\n").slice(0, PREVIEW_LINES);
  if (lines.length === 0) {
    return (
      <Box marginTop={1}>
        <Text color={MUTED}>  (empty)</Text>
      </Box>
    );
  }
  return (
    <Box flexDirection="column" marginTop={1}>
      {lines.map((line, i) => (
        <Text key={i} color={MUTED}>
          {"  "}{truncate(line, 72) || " "}
        </Text>
      ))}
      {text.split("\n").length > PREVIEW_LINES ? (
        <Text color={MUTED}>  …</Text>
      ) : null}
    </Box>
  );
}

/**
 * What Enter does for this phase, expressed as a short human label.
 * Designed to read aloud: "Enter: run Phase 1 with Claude" / "Enter: open Gate 1 review".
 */
function primaryActionLabel(phase: PhaseRecord, hasAgent: boolean, folder: string): string {
  switch (phase.id) {
    case "gate0":
      return "review the knowledge map";
    case "gate1":
      return "review the candidate long-list";
    case "gate2":
      return "review evaluation + quality checks";
    case "assembly":
      if (existsSync(join(folder, TAPE_DRAFT_REL))) {
        return "review the agent's proposed sources";
      }
      return hasAgent
        ? "let the agent populate sources from the keep-list"
        : "open the source editor";
    case "compile":
      if (phase.status === "complete") {
        return "view compile result — copy, share, open folder, or retry";
      }
      return "start compile — fetches sources and writes MIXTAPE.md";
    default:
      return hasAgent
        ? `run Phase ${phase.number} with the agent`
        : `open ${phase.artifact ?? "phase"} in $EDITOR`;
  }
}

function isCompiledArtifactCurrent(project: ReturnType<typeof projectFolder>): boolean {
  if (!existsSync(project.mixtapePath)) return false;
  try {
    const mixtapeMtime = statSync(project.mixtapePath).mtimeMs;
    const tapeMtime = existsSync(project.tapePath) ? statSync(project.tapePath).mtimeMs : 0;
    return mixtapeMtime >= tapeMtime;
  } catch {
    return false;
  }
}

function countCompilationNotes(mixtapePath: string): number {
  try {
    const text = readFileSync(mixtapePath, "utf8");
    const notesStart = text.indexOf("\n## Compilation notes");
    if (notesStart === -1) return 0;
    return text
      .slice(notesStart)
      .split("\n")
      .filter((line) => line.startsWith("- **"))
      .length;
  } catch {
    return 0;
  }
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}
