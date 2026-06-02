import React, { useMemo, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdin } from "ink";
import { spawn } from "node:child_process";
import { join } from "node:path";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { useApp } from "../store.js";
import { MUTED, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { projectFolder } from "../yaml-io.js";
import { resolveEditor } from "../editor.js";
import { parseLonglist, type LonglistSection } from "../gates.js";
import { readGateState, writeGateState } from "../phases.js";
import { markPhaseComplete } from "../progress.js";
import type { Tape } from "../types.js";

/**
 * Methodology-mode pause point between Phase 2 and Phase 3. Deliberately
 * minimal: a summary of what the agent proposed, plus three actions. The
 * actual reading of the longlist happens in `$EDITOR` — Ink-rendered
 * markdown wasn't going to compete with a real text editor.
 *
 * Quick mode skips this screen entirely (PhaseRunner auto-accepts the gate
 * and routes straight to Phase 4).
 */
export function Gate1Review({
  tape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { setRawMode } = useStdin();
  const project = useMemo(() => projectFolder(folder), [folder]);

  const [tick, setTick] = useState(0);
  const sections = useMemo<LonglistSection[]>(
    () => parseLonglist(join(project.path, "working/02-candidate-longlist.md")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [project, tick],
  );

  const totalCandidates = useMemo(
    () => sections.reduce((n, s) => n + s.candidates.length, 0),
    [sections],
  );

  const [accepted, setAccepted] = useState<boolean>(
    () => readGateState(project).gate1Accepted,
  );

  function acceptAndContinue(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate1Accepted: true });
    markPhaseComplete(folder, "gate1");
    setAccepted(true);
    app.setNotification({
      kind: "info",
      message: "Candidates confirmed — continuing to Phase 3 — Evaluation.",
    });
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "evaluation" });
  }

  function resetGate(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate1Accepted: false });
    setAccepted(false);
    app.setNotification({ kind: "info", message: "Confirmation reset." });
  }

  function openInEditor(): void {
    const editor = resolveEditor();
    const target = join(project.path, "working/02-candidate-longlist.md");
    setRawMode(false);
    const child = spawn(editor, [target], { stdio: "inherit" });
    child.on("close", () => {
      setRawMode(true);
      setTick((t) => t + 1);
      app.setNotification({ kind: "info", message: "longlist updated." });
    });
    child.on("error", (e: Error) => {
      setRawMode(true);
      app.setNotification({
        kind: "error",
        message: `Could not launch ${editor}: ${e.message}`,
      });
    });
  }

  function regenerateWithAgent(): void {
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "candidates" });
  }

  useInput((input, key) => {
    if (key.return) acceptAndContinue();
    else if (input === "a") acceptAndContinue();
    else if (input === "u") resetGate();
    else if (input === "e") openInEditor();
    else if (input === "r") regenerateWithAgent();
    else if (input === "R") setTick((t) => t + 1);
    else if (input === "q") ink.exit();
    else if (key.escape || input === "b") {
      app.navigate({ kind: "hub", tape, folder });
    }
  });

  if (totalCandidates === 0) {
    return (
      <Box flexDirection="column">
        <Header title="confirm candidates" subtitle={folder} />
        <Box flexDirection="column" marginBottom={1}>
          <Text color={WARNING}>No candidates yet.</Text>
          <Text color={MUTED}>
            Phase 2 hasn't been run (or the longlist file still has the init
            placeholder). Press <Text color={STRUCTURE}>r</Text> to run Phase 2.
          </Text>
        </Box>
        <KeyHints
          hints={[
            { key: "r", label: "run Phase 2" },
            { key: "e", label: "edit longlist ($EDITOR)" },
            { key: "b", label: "back to hub" },
          { key: "q", label: "quit" },
          ]}
        />
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Header
        title="confirm candidates"
        subtitle={`${folder}  ·  status: ${accepted ? "confirmed" : "ready"}`}
      />

      {/* Summary card — the only content the user needs to read here. */}
      <Box marginBottom={1}>
        <LabeledBox
          label={accepted ? "candidates confirmed" : "confirm candidates"}
          color={accepted ? SUCCESS : STRUCTURE}
        >
          <Box marginBottom={1}>
            <Text bold color={accepted ? SUCCESS : STRUCTURE}>
              {accepted ? "✓ Candidates confirmed" : "Phase 2 complete"}
            </Text>
          </Box>
          <Box>
            <Text>The agent proposed </Text>
            <Text bold>{totalCandidates}</Text>
            <Text> candidate{totalCandidates === 1 ? "" : "s"} across </Text>
            <Text bold>{sections.length}</Text>
            <Text> section{sections.length === 1 ? "" : "s"}:</Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            {sections.map((s) => (
              <Text key={s.title} color={MUTED}>
                {"  · "}
                <Text color={STRUCTURE}>{s.title}</Text>
                <Text>{" — "}{s.candidates.length}</Text>
              </Text>
            ))}
          </Box>
        </LabeledBox>
      </Box>

      <Box marginBottom={1}>
        <Text color={HEADING}>
          {accepted
            ? "next · continue to Phase 3 — Evaluation"
            : "next · accept and continue to Phase 3 — Evaluation"}
        </Text>
      </Box>

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "enter", label: "accept + continue" },
          { key: "e", label: "open longlist ($EDITOR)" },
          { key: "r", label: "regen with agent" },
          ...(accepted ? [{ key: "u", label: "un-accept" }] : []),
          { key: "b", label: "back to hub" },
          { key: "q", label: "quit" },
        ]}
      />
    </Box>
  );
}
