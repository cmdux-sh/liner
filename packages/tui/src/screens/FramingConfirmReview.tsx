import React, { useMemo, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdin } from "ink";
import { existsSync, readFileSync } from "node:fs";
import { spawn } from "node:child_process";
import { join } from "node:path";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { useApp } from "../store.js";
import { projectFolder } from "../yaml-io.js";
import { resolveEditor } from "../editor.js";
import { readGateState, writeGateState } from "../phases.js";
import { markPhaseComplete } from "../progress.js";
import { MUTED, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import type { Tape } from "../types.js";

/**
 * Methodology-mode pause point between Phase 1 (framing) and Phase 2
 * (candidate discovery). Gives the curator a look at the knowledge map's
 * *shape* before the agent burns budget searching for sources. If the map
 * is lopsided (one of two named domains gets six sections, the other one),
 * fix it here cheaply rather than discovering it three phases later.
 *
 * Quick mode skips this screen entirely; PhaseRunner auto-accepts on advance.
 */
export function FramingConfirmReview({
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
  const mapPath = useMemo(
    () => join(project.path, "working/01-jtbd-and-knowledge-map.md"),
    [project],
  );
  const sections = useMemo<KnowledgeMapSection[]>(
    () => parseKnowledgeMap(mapPath),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [mapPath, tick],
  );

  const totalSubAreas = sections.reduce((n, s) => n + s.subAreas.length, 0);
  const emptySections = sections.filter((s) => s.subAreas.length === 0);

  const [accepted, setAccepted] = useState<boolean>(
    () => readGateState(project).gate0Accepted,
  );

  function acceptAndContinue(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate0Accepted: true });
    markPhaseComplete(folder, "gate0");
    setAccepted(true);
    app.setNotification({
      kind: "info",
      message: "Framing confirmed — continuing to Phase 2 — Candidate discovery.",
    });
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "candidates" });
  }

  function resetGate(): void {
    const state = readGateState(project);
    writeGateState(project, { ...state, gate0Accepted: false });
    setAccepted(false);
    app.setNotification({ kind: "info", message: "Confirmation reset." });
  }

  function openInEditor(): void {
    const editor = resolveEditor();
    setRawMode(false);
    const child = spawn(editor, [mapPath], { stdio: "inherit" });
    child.on("close", () => {
      setRawMode(true);
      setTick((t) => t + 1);
      app.setNotification({ kind: "info", message: "knowledge map updated." });
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
    app.navigate({ kind: "phaseRunner", tape, folder, phaseId: "framing" });
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

  if (sections.length === 0) {
    return (
      <Box flexDirection="column">
        <Header title="confirm framing" subtitle={folder} />
        <Box flexDirection="column" marginBottom={1}>
          <Text color={WARNING}>Knowledge map is empty.</Text>
          <Text color={MUTED}>
            Phase 1 hasn't been run yet (or working/01-jtbd-and-knowledge-map.md
            is still on the placeholder). Press <Text color={STRUCTURE}>r</Text> to run
            Phase 1.
          </Text>
        </Box>
        <KeyHints
          hints={[
            { key: "r", label: "run Phase 1" },
            { key: "e", label: "edit knowledge map ($EDITOR)" },
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
        title="confirm framing"
        subtitle={`${folder}  ·  status: ${accepted ? "confirmed" : "ready"}`}
      />

      <Box marginBottom={1}>
        <LabeledBox
          label={accepted ? "framing confirmed" : "confirm framing"}
          color={accepted ? SUCCESS : STRUCTURE}
        >
          <Box marginBottom={1}>
            <Text bold color={accepted ? SUCCESS : STRUCTURE}>
              {accepted ? "✓ Framing confirmed" : "Phase 1 complete"}
            </Text>
          </Box>
          <Box>
            <Text>The agent's knowledge map has </Text>
            <Text bold>{sections.length}</Text>
            <Text> section{sections.length === 1 ? "" : "s"} and </Text>
            <Text bold>{totalSubAreas}</Text>
            <Text> sub-area{totalSubAreas === 1 ? "" : "s"}.</Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            {sections.map((s) => (
              <Box key={s.title}>
                <Text color={MUTED}>{"  · "}</Text>
                <Text color={STRUCTURE}>{s.title}</Text>
                <Text color={MUTED}>
                  {" — "}
                  {s.subAreas.length} sub-area{s.subAreas.length === 1 ? "" : "s"}
                </Text>
              </Box>
            ))}
          </Box>
          {emptySections.length > 0 ? (
            <Box marginTop={1}>
              <Text color={WARNING} bold>
                ⚠ {emptySections.length} section
                {emptySections.length === 1 ? " has" : "s have"} no sub-areas
              </Text>
              <Text color={MUTED}>
                {" — agent may have left them as placeholders. Edit before accepting?"}
              </Text>
            </Box>
          ) : null}
        </LabeledBox>
      </Box>

      <Box marginBottom={1}>
        <Text color={HEADING}>
          {accepted
            ? "next · continue to Phase 2 — Candidate discovery"
            : "next · accept and continue to Phase 2 — Candidate discovery"}
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
          { key: "e", label: "edit map ($EDITOR)" },
          { key: "r", label: "regen with agent" },
          ...(accepted ? [{ key: "u", label: "un-accept" }] : []),
          { key: "b", label: "back to hub" },
          { key: "q", label: "quit" },
        ]}
      />
    </Box>
  );
}

// ---------------------------------------------------------------------------
// Knowledge-map parsing — permissive, format-tolerant
// ---------------------------------------------------------------------------

type KnowledgeMapSection = {
  title: string;
  /** Bullet points / sub-area lines under this section. */
  subAreas: string[];
};

/**
 * Parse working/01-jtbd-and-knowledge-map.md into sections + sub-area counts.
 * Permissive: looks for `## Section title` headers (the convention) and
 * counts `- bullet` lines underneath as sub-areas. Lines before the first
 * header are ignored (JTBD prose, intro). Returns `[]` when the file doesn't
 * exist or has no headers.
 */
function parseKnowledgeMap(path: string): KnowledgeMapSection[] {
  if (!existsSync(path)) return [];
  const text = readFileSync(path, "utf8");
  const lines = text.split("\n");
  const sections: KnowledgeMapSection[] = [];
  let current: KnowledgeMapSection | null = null;
  for (const raw of lines) {
    const line = raw.trim();
    // `## Section` opens a new section; `#` (top-level title) is ignored.
    const headerMatch = /^##+\s+(.*)$/.exec(line);
    if (headerMatch) {
      // `## Knowledge map` is a wrapper header before the real sections in
      // the methodology template. Skip it if its name is obviously generic.
      const title = headerMatch[1]!.trim();
      const lower = title.toLowerCase();
      if (
        lower === "knowledge map" ||
        lower === "jtbd" ||
        lower === "job-to-be-done"
      ) {
        current = null;
        continue;
      }
      current = { title, subAreas: [] };
      sections.push(current);
      continue;
    }
    if (current && /^[-*]\s+\S/.test(line)) {
      current.subAreas.push(line.replace(/^[-*]\s+/, ""));
    }
  }
  return sections;
}
