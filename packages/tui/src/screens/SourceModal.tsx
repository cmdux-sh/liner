import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import TextInput from "ink-text-input";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { useApp } from "../store.js";
import { MUTED, ERROR, STRUCTURE, SUCCESS } from "../colors.js";
import { detectSourceType } from "../yaml-io.js";
import type { Tape, SourceSpec, Priority, SourceType, Render } from "../types.js";
import { LocalFilePicker } from "./LocalFilePicker.js";

type WebField = "url" | "render" | "note" | "section" | "priority";
type LocalField = "path" | "citation" | "note" | "section" | "priority";
type FieldKey = WebField | LocalField;

const WEB_FIELD_ORDER: WebField[] = ["url", "render", "note", "section", "priority"];
const LOCAL_FIELD_ORDER: LocalField[] = ["path", "citation", "note", "section", "priority"];

export function SourceModal({
  tape,
  folder,
  editingIndex,
}: {
  tape: Tape;
  folder: string;
  editingIndex: number | null;
}): React.ReactElement {
  const app = useApp();
  const initial: SourceSpec =
    editingIndex !== null && tape.sources[editingIndex]
      ? (tape.sources[editingIndex] as SourceSpec)
      : {
          type: "web",
          url: "",
          note: "",
          section: "",
          priority: "required",
          render: null,
          path: null,
          citation: null,
        };

  const [draft, setDraft] = useState<SourceSpec>(initial);
  const fieldOrder = draft.type === "local_file" ? LOCAL_FIELD_ORDER : WEB_FIELD_ORDER;
  const [activeField, setActiveField] = useState<FieldKey>(fieldOrder[0] ?? "url");
  const [editing, setEditing] = useState(true);
  const [buffer, setBuffer] = useState(stringValue(initial, fieldOrder[0] ?? "url"));
  const [picking, setPicking] = useState(false);

  function commit(): void {
    if (draft.type === "local_file") {
      if (!draft.path?.trim()) {
        app.setNotification({ kind: "error", message: "path is required for local_file" });
        return;
      }
      if (!draft.citation?.trim()) {
        app.setNotification({ kind: "error", message: "citation is required for local_file" });
        return;
      }
    } else {
      if (!draft.url?.trim()) {
        app.setNotification({ kind: "error", message: "URL is required" });
        return;
      }
    }

    const finalDraft: SourceSpec =
      draft.type === "local_file"
        ? {
            ...draft,
            url: "",
            path: (draft.path ?? "").trim(),
            citation: (draft.citation ?? "").trim(),
            note: draft.note?.trim() || null,
            section: draft.section?.trim() || null,
            render: null,
          }
        : {
            ...draft,
            type: detectSourceType(draft.url) as SourceType,
            url: draft.url.trim(),
            path: null,
            citation: null,
            note: draft.note?.trim() || null,
            section: draft.section?.trim() || null,
          };

    const next = [...tape.sources];
    if (editingIndex !== null) {
      next[editingIndex] = finalDraft;
    } else {
      next.push(finalDraft);
    }
    app.navigate({ kind: "editor", tape: { ...tape, sources: next }, folder });
  }

  function cancel(): void {
    app.navigate({ kind: "editor", tape, folder });
  }

  function moveField(delta: number): void {
    const idx = fieldOrder.indexOf(activeField as never);
    const order = fieldOrder as readonly FieldKey[];
    const next = order[(idx + delta + order.length) % order.length];
    if (!next) return;
    setActiveField(next);
    setBuffer(stringValue(draft, next));
    setEditing(true);
  }

  function toggleType(): void {
    const newType: SourceType = draft.type === "local_file" ? "web" : "local_file";
    const updated: SourceSpec = { ...draft, type: newType };
    setDraft(updated);
    const newOrder = newType === "local_file" ? LOCAL_FIELD_ORDER : WEB_FIELD_ORDER;
    const firstField = newOrder[0] ?? "url";
    setActiveField(firstField);
    setBuffer(stringValue(updated, firstField));
    setEditing(false);
  }

  function toggleRender(): void {
    if (draft.type !== "web") return;
    // Cycle: auto (null) → js → server → auto. The auto state mirrors the
    // CLI's compile-time default: try server-rendered first, fall back to
    // Playwright if the page is JS-walled.
    let next: Render | null;
    if (draft.render === null || draft.render === undefined) {
      next = "js";
    } else if (draft.render === "js") {
      next = "server";
    } else {
      next = null;
    }
    setDraft({ ...draft, render: next });
  }

  function applyBuffer(field: FieldKey, value: string): SourceSpec {
    if (field === "priority") {
      const p: Priority = value.toLowerCase().startsWith("o") ? "optional" : "required";
      return { ...draft, priority: p };
    }
    if (field === "render") {
      const v = value.trim().toLowerCase();
      if (v === "js" || v === "j") return { ...draft, render: "js" };
      if (v === "server" || v === "s") return { ...draft, render: "server" };
      // Anything else — including empty string and "auto" — resolves to null,
      // which is the CLI's auto-fallback default.
      return { ...draft, render: null };
    }
    if (field === "path") return { ...draft, path: value };
    if (field === "citation") return { ...draft, citation: value };
    return { ...draft, [field]: value } as SourceSpec;
  }

  function openPicker(): void {
    setEditing(false);
    setPicking(true);
  }

  // Editing mode: TextInput owns letters/numbers. Only tab + escape leak to us.
  useInput(
    (_input, key) => {
      if (key.tab) {
        const updated = applyBuffer(activeField, buffer);
        setDraft(updated);
        moveField(key.shift ? -1 : 1);
        return;
      }
      if (key.escape) setEditing(false);
    },
    { isActive: editing && !picking },
  );

  // Command mode: full keybinding palette.
  useInput(
    (input, key) => {
      if (key.escape) cancel();
      else if (input === "s") commit();
      else if (input === "T") toggleType();
      else if (input === "R") toggleRender();
      else if (key.return) {
        if (draft.type === "local_file" && activeField === "path") {
          openPicker();
          return;
        }
        setBuffer(stringValue(draft, activeField));
        setEditing(true);
      } else if (key.upArrow || input === "k") moveField(-1);
      else if (key.downArrow || input === "j" || key.tab) moveField(1);
      else if (input === "p") {
        const updated: SourceSpec = {
          ...draft,
          priority: draft.priority === "required" ? "optional" : "required",
        };
        setDraft(updated);
      }
    },
    { isActive: !editing && !picking },
  );

  if (picking) {
    return (
      <LocalFilePicker
        projectFolder={folder}
        initialPath={draft.path}
        onPick={(relPath) => {
          setDraft({ ...draft, path: relPath });
          setPicking(false);
          // After picking, advance to citation if it's empty.
          if (!draft.citation?.trim()) {
            setActiveField("citation");
            setBuffer("");
            setEditing(true);
          }
        }}
        onCancel={() => setPicking(false)}
      />
    );
  }

  const detectedTypeForWeb: SourceType = draft.url
    ? (detectSourceType(draft.url) as SourceType)
    : "web";
  const headerSubtitle =
    draft.type === "local_file"
      ? "type: local_file"
      : `detected type: ${detectedTypeForWeb}${draft.render ? `  ·  render: ${draft.render}` : ""}`;

  return (
    <Box flexDirection="column">
      <Header
        title={editingIndex !== null ? `edit source #${editingIndex + 1}` : "add source"}
        subtitle={headerSubtitle}
      />

      <Box flexDirection="column" marginBottom={1}>
        {fieldOrder.map((field) => (
          <FieldRow
            key={field}
            field={field}
            value={editing && field === activeField ? buffer : stringValue(draft, field)}
            active={field === activeField}
            editing={editing && field === activeField}
            onChange={setBuffer}
            onSubmit={(v) => {
              const updated = applyBuffer(field, v);
              setDraft(updated);
              setEditing(false);
            }}
            placeholder={placeholderFor(field, draft)}
          />
        ))}
      </Box>

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      <KeyHints
        hints={
          editing
            ? [
                { key: "enter", label: "next field" },
                { key: "tab", label: "next/prev" },
                { key: "esc", label: "stop editing" },
              ]
            : [
                { key: "↑↓", label: "field" },
                { key: "enter", label: draft.type === "local_file" && activeField === "path" ? "pick file" : "edit" },
                { key: "T", label: "toggle type" },
                ...(draft.type === "web" ? [{ key: "R", label: "render" }] : []),
                { key: "p", label: "priority" },
                { key: "s", label: "save" },
                { key: "esc", label: "cancel" },
              ]
        }
      />
    </Box>
  );
}

function FieldRow({
  field,
  value,
  active,
  editing,
  onChange,
  onSubmit,
  placeholder,
}: {
  field: FieldKey;
  value: string;
  active: boolean;
  editing: boolean;
  onChange: (v: string) => void;
  onSubmit: (v: string) => void;
  placeholder: string;
}): React.ReactElement {
  const arrow = active ? "▶ " : "  ";
  const label = field.padEnd(9, " ");
  return (
    <Box>
      <Text color={active ? STRUCTURE : undefined}>
        {arrow}
        {label}
      </Text>
      {editing ? (
        <TextInput value={value} onChange={onChange} onSubmit={onSubmit} placeholder={placeholder} />
      ) : (
        <Text>{value || <Text color={MUTED}>{placeholder}</Text>}</Text>
      )}
    </Box>
  );
}

function stringValue(s: SourceSpec, field: FieldKey): string {
  if (field === "priority") return s.priority;
  if (field === "render") return s.render ?? "auto";
  if (field === "path") return s.path ?? "";
  if (field === "citation") return s.citation ?? "";
  if (field === "url") return s.url ?? "";
  const v = s[field as "note" | "section"];
  return v == null ? "" : String(v);
}

function placeholderFor(field: FieldKey, draft: SourceSpec): string {
  if (field === "url") {
    const detected = draft.url ? detectSourceType(draft.url) : "web";
    return detected === "youtube"
      ? "https://www.youtube.com/watch?v=…"
      : "https://…";
  }
  if (field === "render") return "auto | js | server  (auto = try server, fall back to JS)";
  if (field === "path") return "personal/foo.pdf  (press enter to pick a file)";
  if (field === "citation") return "Author, Title, Publication, Date";
  if (field === "note") return "why this source matters";
  if (field === "section") return "optional group name (e.g. foundations)";
  return "required | optional";
}
