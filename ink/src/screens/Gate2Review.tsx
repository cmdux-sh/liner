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
import {
  groupEvaluation,
  parseEvaluation,
  parseQualityChecks,
} from "../gates.js";
import { readGateState, writeGateState } from "../phases.js";
import { markPhaseComplete } from "../progress.js";
import type { Tape } from "../types.js";

/**
 * Methodology-mode pause between Phase 5 (quality checks) and Phase 6
 * (synthesis). Like Gate 1, kept minimal: counts + actions. Inspection
 * happens in `$EDITOR`. Quick mode skips this screen entirely.
 */
export function Gate2Review({
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

  const items = useMemo(
    () => parseEvaluation(join(project.path, "working/03-evaluation.yaml")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [project, tick],
  );
  const groups = useMemo(() => groupEvaluation(items), [items]);
  const checks = useMemo(
    () => parseQualityChecks(join(project.path, "working/04-quality-checks.md")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [project, tick],
  );

  const counts = {
    kept: groups.find((g) => g.decision === "kept")?.candidates.length ?? 0,
    trimmed: groups.find((g) => g.decision === "trimmed")?.candidates.length ?? 0,
    dropped: groups.find((g) => g.decision === "dropped")?.candidates.length ?? 0,
  };
  const totalEvaluated = counts.kept + counts.trimmed + counts.dropped;

  const [accepted, setAccepted] = useState<boolean>(
    () => readGateState(project).gate2Accepted,
  );

  function acceptAndContinue(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate2Accepted: true });
    markPhaseComplete(folder, "gate2");
    setAccepted(true);
    app.setNotification({
      kind: "info",
      message: "Evaluation confirmed — continuing to Phase 5 — Synthesis.",
    });
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "synthesis" });
  }

  function resetGate(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate2Accepted: false });
    setAccepted(false);
    app.setNotification({ kind: "info", message: "Confirmation reset." });
  }

  function openInEditor(target: "evaluation" | "quality"): void {
    const editor = resolveEditor();
    const rel =
      target === "evaluation"
        ? "working/03-evaluation.yaml"
        : "working/04-quality-checks.md";
    const full = join(project.path, rel);
    setRawMode(false);
    const child = spawn(editor, [full], { stdio: "inherit" });
    child.on("close", () => {
      setRawMode(true);
      setTick((t) => t + 1);
      app.setNotification({ kind: "info", message: `${rel} updated.` });
    });
    child.on("error", (e: Error) => {
      setRawMode(true);
      app.setNotification({
        kind: "error",
        message: `Could not launch ${editor}: ${e.message}`,
      });
    });
  }

  function regenerateEvaluation(): void {
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "evaluation" });
  }

  useInput((input, key) => {
    if (key.return) acceptAndContinue();
    else if (input === "a") acceptAndContinue();
    else if (input === "u") resetGate();
    else if (input === "e") openInEditor("evaluation");
    else if (input === "k") openInEditor("quality");
    else if (input === "r") regenerateEvaluation();
    else if (input === "R") setTick((t) => t + 1);
    else if (input === "q") ink.exit();
    else if (key.escape || input === "b") {
      app.navigate({ kind: "hub", tape, folder });
    }
  });

  if (totalEvaluated === 0) {
    return (
      <Box flexDirection="column">
        <Header title="confirm evaluation" subtitle={folder} />
        <Box flexDirection="column" marginBottom={1}>
          <Text color={WARNING}>Nothing to review yet.</Text>
          <Text color={MUTED}>
            Phase 4 (Evaluation) hasn't been run. Press{" "}
            <Text color={STRUCTURE}>r</Text> to run it now.
          </Text>
        </Box>
        <KeyHints
          hints={[
            { key: "r", label: "run Phase 4" },
            { key: "e", label: "edit evaluation ($EDITOR)" },
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
        title="confirm evaluation"
        subtitle={`${folder}  ·  status: ${accepted ? "confirmed" : "ready"}`}
      />

      <Box marginBottom={1}>
        <LabeledBox
          label={accepted ? "evaluation confirmed" : "confirm evaluation"}
          color={accepted ? SUCCESS : STRUCTURE}
        >
          <Box marginBottom={1}>
            <Text bold color={accepted ? SUCCESS : STRUCTURE}>
              {accepted ? "✓ Evaluation confirmed" : "Phases 4 + 5 complete"}
            </Text>
          </Box>
          <Box>
            <Text>The agent </Text>
            <Text bold color={SUCCESS}>kept {counts.kept}</Text>
            <Text>, </Text>
            <Text bold color={WARNING}>trimmed {counts.trimmed}</Text>
            <Text>, </Text>
            <Text bold color={ERROR}>dropped {counts.dropped}</Text>
            <Text>.</Text>
          </Box>
          <Box marginTop={1}>
            <Text color={MUTED}>
              Quality checks: {checks.length} finding{checks.length === 1 ? "" : "s"} recorded.
            </Text>
          </Box>
        </LabeledBox>
      </Box>

      <Box marginBottom={1}>
        <Text color={HEADING}>
          {accepted
            ? "next · continue to Phase 5 — Synthesis"
            : "next · accept and continue to Phase 5 — Synthesis"}
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
          { key: "e", label: "evaluation ($EDITOR)" },
          { key: "k", label: "quality checks ($EDITOR)" },
          { key: "r", label: "regen eval" },
          ...(accepted ? [{ key: "u", label: "un-accept" }] : []),
          { key: "b", label: "back to hub" },
          { key: "q", label: "quit" },
        ]}
      />
    </Box>
  );
}
