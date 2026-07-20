import React, { useMemo, useState } from "react";
import { Box, Text, useInput, useStdin } from "ink";
import TextInput from "ink-text-input";
import { spawn } from "node:child_process";
import { join, basename } from "node:path";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { useApp } from "../store.js";
import { MUTED, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import {
  projectFolder,
  readSynthesisStatus,
  validateTape,
  writeTape,
  type SynthesisStatus,
} from "../yaml-io.js";
import { resolveEditor } from "../editor.js";
import type { Tape, SourceSpec, Mode } from "../types.js";

type Field = "title" | "description" | "curator" | "jtbd" | null;

export function TapeEditor({
  tape: initialTape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();
  const { setRawMode } = useStdin();
  const project = useMemo(() => projectFolder(folder), [folder]);
  const [tape, setTape] = useState<Tape>(initialTape);
  const [editingField, setEditingField] = useState<Field>(null);
  const [bufferValue, setBufferValue] = useState("");
  const [sourceIndex, setSourceIndex] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [synthesisStatus, setSynthesisStatus] = useState<SynthesisStatus>(() =>
    readSynthesisStatus(project),
  );

  const errors = useMemo(() => validateTape(tape), [tape]);
  const sourcesMissingNotes = useMemo(
    () => tape.sources.filter((s) => !s.note?.trim()).length,
    [tape.sources],
  );

  function startEdit(field: Exclude<Field, null>): void {
    setBufferValue(String(tape[field] ?? ""));
    setEditingField(field);
  }

  function moveSource(from: number, to: number): void {
    if (to < 0 || to >= tape.sources.length) return;
    const next = [...tape.sources];
    const [item] = next.splice(from, 1);
    if (!item) return;
    next.splice(to, 0, item);
    setTape({ ...tape, sources: next });
    setSourceIndex(to);
    setDirty(true);
  }

  function deleteSelected(): void {
    if (tape.sources.length === 0) return;
    const next = tape.sources.filter((_, i) => i !== sourceIndex);
    setTape({ ...tape, sources: next });
    setSourceIndex(Math.max(0, Math.min(sourceIndex, next.length - 1)));
    setDirty(true);
  }

  function toggleMode(): void {
    const next: Mode = tape.mode === "methodology" ? "quick" : "methodology";
    setTape({ ...tape, mode: next });
    setDirty(true);
  }

  function save(): boolean {
    if (errors.length > 0) {
      app.setNotification({
        kind: "error",
        message: `Cannot save: ${errors[0]?.field} — ${errors[0]?.message}`,
      });
      return false;
    }
    try {
      writeTape(project.tapePath, tape);
      setDirty(false);
      app.setNotification({ kind: "info", message: `Saved ${project.tapePath}` });
      return true;
    } catch (e) {
      app.setNotification({ kind: "error", message: (e as Error).message });
      return false;
    }
  }

  function launchSynthesisEditor(): void {
    const editor = resolveEditor();
    // Release raw mode so the external editor gets the TTY cleanly.
    // The editor inherits the TTY and typically clears the screen itself.
    setRawMode(false);
    const child = spawn(editor, [project.synthesisPath], { stdio: "inherit" });
    child.on("close", () => {
      // Restore raw mode for Ink and refresh synthesis status.
      setRawMode(true);
      setSynthesisStatus(readSynthesisStatus(project));
      app.setNotification({ kind: "info", message: "synthesis.md updated." });
    });
    child.on("error", (e: Error) => {
      setRawMode(true);
      app.setNotification({
        kind: "error",
        message: `Could not launch ${editor}: ${e.message}`,
      });
    });
  }

  function compile(): void {
    if (!save()) return;
    if (!synthesisStatus.exists || !synthesisStatus.isReady) {
      app.setNotification({
        kind: "error",
        message:
          "synthesis.md is empty or still a placeholder — press E to write it before compiling.",
      });
      return;
    }
    app.navigate({ kind: "compile", tape, folder });
  }

  // Active only while a meta field is being edited inside TextInput.
  useInput(
    (_input, key) => {
      if (key.escape) setEditingField(null);
    },
    { isActive: editingField !== null },
  );

  // Active only in command mode (nothing being typed into TextInput).
  useInput((input, key) => {
    if (key.escape) {
      if (dirty) {
        app.setNotification({
          kind: "error",
          message: "Unsaved changes — press s to save, or Esc again to discard.",
        });
        setDirty(false);
        return;
      }
      void app.refreshProjects();
      app.navigate({ kind: "hub", tape, folder });
      return;
    }

    if (input === "t") startEdit("title");
    else if (input === "d") startEdit("description");
    else if (input === "u") startEdit("curator");
    else if (input === "g") startEdit("jtbd");
    else if (input === "m") toggleMode();
    else if (input === "E") launchSynthesisEditor();
    else if (input === "a") {
      app.navigate({ kind: "sourceModal", tape, folder, editingIndex: null });
    } else if (input === "p") {
      app.navigate({ kind: "sourceInbox", tape, folder });
    } else if (input === "e") {
      if (tape.sources.length > 0) {
        app.navigate({
          kind: "sourceModal",
          tape,
          folder,
          editingIndex: sourceIndex,
        });
      }
    } else if (input === "x") {
      deleteSelected();
    } else if (key.upArrow || input === "k") {
      setSourceIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === "j") {
      setSourceIndex((i) => Math.min(Math.max(0, tape.sources.length - 1), i + 1));
    } else if (input === "K") {
      moveSource(sourceIndex, sourceIndex - 1);
    } else if (input === "J") {
      moveSource(sourceIndex, sourceIndex + 1);
    } else if (input === "s") {
      save();
    } else if (input === "c") {
      compile();
    }
  }, { isActive: editingField === null });

  if (editingField) {
    return (
      <Box flexDirection="column">
        <Header title={`edit ${editingField}`} />
        <Box>
          <Text color={STRUCTURE}>{editingField}: </Text>
          <TextInput
            value={bufferValue}
            onChange={setBufferValue}
            onSubmit={(v) => {
              setTape((t) => ({ ...t, [editingField]: v }));
              setEditingField(null);
              setDirty(true);
            }}
          />
        </Box>
        <Box marginTop={1}>
          <KeyHints
            hints={[
              { key: "enter", label: "save" },
              { key: "esc", label: "cancel" },
            ]}
          />
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Header title="phase 7 — assembly" subtitle={`${folder}  (${basename(project.tapePath)})`} />

      <Box flexDirection="column" marginBottom={1}>
        <Field label="title" value={tape.title} hint="t" />
        <Field label="description" value={tape.description} hint="d" />
        <Field label="curator" value={tape.curator} hint="u" />
        <Field
          label="mode"
          value={tape.mode ?? "(unset)"}
          hint="m"
          valueColor={tape.mode === "methodology" ? HEADING : undefined}
        />
        <Field label="jtbd" value={tape.jtbd ?? ""} hint="g" placeholder="(no JTBD — press g to set)" />
      </Box>

      <Box marginBottom={1}>
        <Text color={STRUCTURE}>[E] </Text>
        <Text>synthesis: </Text>
        <SynthesisBadge status={synthesisStatus} />
      </Box>

      <Text color={HEADING} bold>
        Sources ({tape.sources.length})
      </Text>
      {tape.sources.length === 0 ? (
        <Text color={MUTED}>  none yet — press `a` to add</Text>
      ) : (
        <Box flexDirection="column" marginBottom={1}>
          {tape.sources.map((s, i) => (
            <SourceRow key={i} source={s} index={i} selected={i === sourceIndex} />
          ))}
        </Box>
      )}

      {errors.length > 0 ? (
        <Box marginBottom={1} flexDirection="column">
          <Text color={WARNING}>
            ⚠ {errors.length} validation issue{errors.length === 1 ? "" : "s"}:
          </Text>
          {errors.slice(0, 3).map((e, i) => (
            <Text key={i} color={MUTED}>
              {"  "}
              {e.field}: {e.message}
            </Text>
          ))}
        </Box>
      ) : null}

      {sourcesMissingNotes > 0 ? (
        <Box marginBottom={1} flexDirection="column">
          <Text color={WARNING}>
            ⚠ {sourcesMissingNotes} source{sourcesMissingNotes === 1 ? "" : "s"} without a curator note
          </Text>
          <Text color={MUTED}>
            {"  "}Notes are load-bearing — the AI uses them to weight sources. See CURATION.md §6.
          </Text>
        </Box>
      ) : null}

      {dirty ? (
        <Box marginBottom={1}>
          <Text color={WARNING}>● unsaved changes</Text>
        </Box>
      ) : null}

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "t/d/u", label: "edit meta" },
          { key: "m", label: "mode" },
          { key: "g", label: "jtbd" },
          { key: "E", label: "synthesis" },
          { key: "↑↓", label: "source" },
          { key: "a/e/x", label: "add/edit/del" },
          { key: "p", label: "paste sources" },
          { key: "K/J", label: "reorder" },
          { key: "s", label: "save" },
          { key: "c", label: "compile" },
          { key: "esc", label: "back" },
        ]}
      />
    </Box>
  );
}

function Field({
  label,
  value,
  hint,
  placeholder,
  valueColor,
}: {
  label: string;
  value: string;
  hint: string;
  placeholder?: string;
  valueColor?: string;
}): React.ReactElement {
  const empty = !value || value.trim() === "";
  return (
    <Box>
      <Text color={STRUCTURE}>[{hint}] </Text>
      <Text>{label}: </Text>
      {empty ? (
        <Text color={MUTED}>{placeholder ?? "(empty)"}</Text>
      ) : (
        <Text bold color={valueColor}>
          {value}
        </Text>
      )}
    </Box>
  );
}

function SynthesisBadge({ status }: { status: SynthesisStatus }): React.ReactElement {
  if (!status.exists) {
    return <Text color={ERROR}>✗ missing (compile will fail)</Text>;
  }
  if (status.isPlaceholder) {
    return <Text color={WARNING}>⚠ placeholder — replace before compile</Text>;
  }
  if (status.charCount === 0) {
    return <Text color={WARNING}>⚠ empty</Text>;
  }
  return (
    <Text color={SUCCESS}>
      ✓ {status.charCount.toLocaleString()} chars
    </Text>
  );
}

function SourceRow({
  source,
  index,
  selected,
}: {
  source: SourceSpec;
  index: number;
  selected: boolean;
}): React.ReactElement {
  const icon = iconFor(source);
  const sectionLabel = source.section ? `[${source.section}] ` : "";
  const opt = source.priority === "optional" ? " (optional)" : "";
  const label = labelFor(source);
  return (
    <Box>
      <Text color={selected ? STRUCTURE : undefined}>
        {selected ? "▶ " : "  "}
        {index + 1}. {icon} {sectionLabel}
        {truncate(label, 60)}
        {opt}
      </Text>
    </Box>
  );
}

function iconFor(source: SourceSpec): string {
  if (source.type === "youtube") return "▶";
  if (source.type === "local_file") return "▣";
  if (source.type === "skill") return "✦";
  if (source.render === "js") return "◍";
  if (source.render === "server") return "◎";
  // null/undefined: auto-fallback (default). Same glyph as the plain web case
  // from before this change — visually unchanged for the vast majority of sources.
  return "◌";
}

function labelFor(source: SourceSpec): string {
  if (source.type === "local_file") {
    return source.citation || source.path || "(unset local file)";
  }
  if (source.type === "skill") {
    return source.path || source.url || "(unset skill)";
  }
  return source.url;
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}
