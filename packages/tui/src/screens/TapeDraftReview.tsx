import React, { useEffect, useMemo, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput } from "ink";
import TextInput from "ink-text-input";
import { existsSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import * as YAML from "yaml";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { useApp } from "../store.js";
import { detectSourceType, projectFolder, writeTape, readTape } from "../yaml-io.js";
import { markPhaseComplete } from "../progress.js";
import { MUTED, CURRENT, ERROR, HEADING, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import type { SourceSpec, Tape } from "../types.js";

const DRAFT_PATH_REL = "working/07-tape-draft.yaml";
const VIEWPORT = 14;

type ParseResult =
  | { ok: true; sources: SourceSpec[] }
  | { ok: false; error: string };

export function TapeDraftReview({
  tape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const project = useMemo(() => projectFolder(folder), [folder]);
  const draftPath = useMemo(() => join(project.path, DRAFT_PATH_REL), [project]);

  const [tick, setTick] = useState(0);
  const parsed = useMemo<ParseResult>(
    () => loadDraftSources(draftPath),
    // tick re-derives. eslint-disable-next-line react-hooks/exhaustive-deps
    [draftPath, tick],
  );

  const draftSources = parsed.ok ? parsed.sources : [];
  const sectionCounts = useMemo(() => groupBySection(draftSources), [draftSources]);

  const [cursor, setCursor] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);
  // Inline "add my own source" prompt. Off by default; opened with `+`.
  // Kept as a small URL-only input so the curator's "and here's one of mine"
  // moment doesn't require leaving the review surface. The user can refine
  // note/section/kind later via $EDITOR on tape.yaml.
  const [addingUrl, setAddingUrl] = useState<string | null>(null);
  // Set of source URLs (or path|citation for local_file) that the curator has
  // UN-checked. Everything not in this set is kept. We track exclusions rather
  // than inclusions so freshly-loaded drafts default to "all checked" without
  // an init step, and the set stays small when most sources are accepted.
  const [excluded, setExcluded] = useState<Set<string>>(() => new Set());

  // When the draft sources list changes (e.g. refresh via R), drop any
  // exclusion keys that no longer match a real source.
  useEffect(() => {
    if (!parsed.ok) return;
    const validKeys = new Set(draftSources.map(sourceKey));
    setExcluded((prev) => {
      let changed = false;
      const next = new Set<string>();
      for (const k of prev) {
        if (validKeys.has(k)) next.add(k);
        else changed = true;
      }
      return changed ? next : prev;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [parsed.ok ? draftSources.length : 0, tick]);

  const keptCount = draftSources.filter((s) => !excluded.has(sourceKey(s))).length;
  const droppedCount = draftSources.length - keptCount;

  function moveCursor(delta: number): void {
    if (draftSources.length === 0) return;
    const next = Math.max(0, Math.min(draftSources.length - 1, cursor + delta));
    setCursor(next);
    if (next < scrollTop) setScrollTop(next);
    else if (next >= scrollTop + VIEWPORT) setScrollTop(next - VIEWPORT + 1);
  }

  function toggleCursor(): void {
    const source = draftSources[cursor];
    if (!source) return;
    const key = sourceKey(source);
    setExcluded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  function setAllChecked(checked: boolean): void {
    if (checked) {
      setExcluded(new Set());
    } else {
      setExcluded(new Set(draftSources.map(sourceKey)));
    }
  }

  function accept(): void {
    if (!parsed.ok) {
      app.setNotification({ kind: "error", message: `Draft is invalid: ${parsed.error}` });
      return;
    }
    const kept = parsed.sources.filter((s) => !excluded.has(sourceKey(s)));
    if (kept.length === 0) {
      app.setNotification({
        kind: "error",
        message: "Nothing checked — press space to keep at least one source, or esc to back out.",
      });
      return;
    }
    try {
      // Re-read tape from disk so we don't clobber any changes the curator
      // made since this screen was mounted.
      const { tape: current, doc } = readTape(project.tapePath);
      const next: Tape = { ...current, sources: kept };
      writeTape(project.tapePath, next, doc);
      // Clean up the draft after a successful merge. The user has committed
      // to a specific subset; the draft has served its purpose.
      try {
        unlinkSync(draftPath);
      } catch {
        // non-fatal; user can delete it manually if it lingers
      }
      // Phase 7 is "done" the moment the curator accepts a non-empty subset.
      markPhaseComplete(folder, "assembly");
      const dropped = parsed.sources.length - kept.length;
      const droppedNote = dropped > 0 ? `, dropped ${dropped}` : "";
      app.setNotification({
        kind: "info",
        message: `Wrote ${kept.length} source${kept.length === 1 ? "" : "s"} to tape.yaml${droppedNote}.`,
      });
      void app.refreshProjects();
      // Go straight to compile — the curator's act of pressing enter on
      // this screen IS the decision to start compiling. Routing through
      // the hub would force a second confirmation for no new information.
      app.navigate({ kind: "compile", tape: next, folder });
    } catch (e) {
      app.setNotification({ kind: "error", message: `Merge failed: ${(e as Error).message}` });
    }
  }

  function commitNewUrl(rawUrl: string): void {
    const url = rawUrl.trim();
    if (!url) {
      setAddingUrl(null);
      return;
    }
    try {
      appendUrlToDraft(draftPath, url);
      setAddingUrl(null);
      setTick((t) => t + 1);
      app.setNotification({ kind: "info", message: `Added ${url}` });
    } catch (e) {
      app.setNotification({
        kind: "error",
        message: `Could not add source: ${(e as Error).message}`,
      });
    }
  }

  useInput((input, key) => {
    // While the add-url prompt is open, hand all input to TextInput.
    // The only key we intercept here is esc, to cancel the prompt cleanly.
    if (addingUrl !== null) {
      if (key.escape) setAddingUrl(null);
      return;
    }
    if (key.upArrow || input === "k") moveCursor(-1);
    else if (key.downArrow || input === "j") moveCursor(1);
    else if (key.pageUp) moveCursor(-VIEWPORT);
    else if (key.pageDown) moveCursor(VIEWPORT);
    else if (input === " " || input === "x") toggleCursor();
    else if (input === "A") setAllChecked(true); // shift-A: check all
    else if (input === "N") setAllChecked(false); // shift-N: uncheck all
    else if (input === "+") setAddingUrl(""); // open inline URL prompt
    else if (key.return) accept();
    else if (input === "R") setTick((t) => t + 1);
    else if (input === "q") ink.exit();
    else if (key.escape || input === "b") app.navigate({ kind: "hub", tape, folder });
  });

  // Empty-state: draft file missing entirely.
  if (!existsSync(draftPath)) {
    return (
      <Box flexDirection="column">
        <Header title="review proposed sources" subtitle={folder} />
        <Box flexDirection="column" marginBottom={1}>
          <Text color={WARNING}>No draft to review.</Text>
          <Text color={MUTED}>
            Expected <Text color={STRUCTURE}>{DRAFT_PATH_REL}</Text> — Phase 7 hasn't been run with an
            agent yet (or the draft was already accepted).
          </Text>
        </Box>
        <KeyHints
          hints={[
            { key: "b", label: "back to hub" },
            { key: "R", label: "refresh" },
          ]}
        />
      </Box>
    );
  }

  if (!parsed.ok) {
    return (
      <Box flexDirection="column">
        <Header title="review proposed sources" subtitle={folder} />
        <Box flexDirection="column" marginBottom={1}>
          <Text color={ERROR}>Could not parse the draft.</Text>
          <Text color={MUTED}>{parsed.error}</Text>
          <Box marginTop={1}>
            <Text color={MUTED}>
              Open <Text color={STRUCTURE}>{DRAFT_PATH_REL}</Text> in an editor and fix the YAML, or
              press <Text color={STRUCTURE}>d</Text> to discard and re-run Phase 7.
            </Text>
          </Box>
        </Box>
        <KeyHints
          hints={[
            { key: "d", label: "discard" },
            { key: "R", label: "refresh" },
            { key: "b", label: "back" },
          ]}
        />
      </Box>
    );
  }

  const visible = draftSources.slice(scrollTop, scrollTop + VIEWPORT);
  const cursorSource = draftSources[cursor];

  return (
    <Box flexDirection="column">
      <Header
        title="pick what goes into the tape"
        subtitle={`${folder}  ·  current tape: ${tape.sources.length}`}
      />

      <Box flexDirection="column" marginBottom={1}>
        <Text color={MUTED}>
          The agent proposed{" "}
          <Text color={STRUCTURE}>{draftSources.length}</Text>
          {" "}source{draftSources.length === 1 ? "" : "s"}. Use{" "}
          <Text color={STRUCTURE}>↑↓</Text> to move,{" "}
          <Text color={STRUCTURE}>space</Text> to toggle,{" "}
          <Text color={STRUCTURE}>[+]</Text> to add one of your own,{" "}
          <Text color={STRUCTURE}>enter</Text> to write the checked subset into tape.yaml and compile.
        </Text>
        <Text color={MUTED}>
          by section: {sectionCounts.map((s) => `${s.section} (${s.count})`).join("  ·  ")}
        </Text>
      </Box>

      <Box marginBottom={1}>
        <LabeledBox
          label={`proposed sources  ·  ${keptCount} keep  ·  ${droppedCount} drop`}
          color={STRUCTURE}
          paddingX={1}
        >
          {visible.map((s, i) => {
            const idx = scrollTop + i;
            const isCursor = idx === cursor;
            const isExcluded = excluded.has(sourceKey(s));
            return (
              <Box key={idx}>
                <Text color={isCursor ? CURRENT : undefined} bold={isCursor}>
                  {isCursor ? "▸ " : "  "}
                </Text>
                <Text color={isExcluded ? ERROR : SUCCESS} bold>
                  {isExcluded ? "[ ]" : "[x]"}
                </Text>
                <Text color={MUTED}>{"  "}{String(idx + 1).padStart(2, " ")}.</Text>
                <Text>{" "}</Text>
                <Text color={isExcluded ? MUTED : undefined}>{iconFor(s)}</Text>
                <Text color={MUTED}>{s.section ? `  [${truncate(s.section, 14)}]` : "  [-]"}</Text>
                {s.kind ? <Text color={MUTED}>{`  ${kindChip(s.kind)}`}</Text> : null}
                <Text color={isExcluded ? MUTED : undefined}>{"  " + truncate(labelFor(s), 50)}</Text>
                {s.priority === "optional" ? <Text color={MUTED}>{"  opt"}</Text> : null}
              </Box>
            );
          })}
          {draftSources.length > VIEWPORT ? (
            <Box marginTop={1}>
              <Text color={MUTED}>
                {`showing ${scrollTop + 1}–${Math.min(scrollTop + VIEWPORT, draftSources.length)} of ${draftSources.length}`}
              </Text>
            </Box>
          ) : null}
        </LabeledBox>
      </Box>

      {addingUrl !== null ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={HEADING} bold>Add your own source</Text>
          <Box marginTop={1}>
            <Text color={STRUCTURE}>URL: </Text>
            <TextInput
              value={addingUrl}
              onChange={setAddingUrl}
              onSubmit={commitNewUrl}
            />
          </Box>
          <Text color={MUTED}>
            {"  "}Type a URL and press <Text color={STRUCTURE}>enter</Text> to append it as a checked source.
            {" "}<Text color={STRUCTURE}>esc</Text> cancels. You can edit notes/section/kind later in tape.yaml.
          </Text>
        </Box>
      ) : null}

      {cursorSource && cursorSource.note?.trim() ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={ERROR}>curator note</Text>
          {cursorSource.note
            .split("\n")
            .slice(0, 4)
            .map((line, i) => (
              <Text key={i} color={ERROR}>
                {"  " + truncate(line.trim(), 90)}
              </Text>
            ))}
        </Box>
      ) : null}

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      {droppedCount > 0 ? (
        <Box marginBottom={1}>
          <Text color={WARNING}>
            {`${droppedCount} source${droppedCount === 1 ? "" : "s"} will be dropped on enter. `}
          </Text>
          <Text color={MUTED}>(toggle back with space)</Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "↑↓", label: "move" },
          { key: "space", label: "toggle" },
          { key: "+", label: "add own" },
          { key: "enter", label: `accept ${keptCount}` },
          { key: "A", label: "all" },
          { key: "N", label: "none" },
          { key: "esc", label: "go back" },
          { key: "q", label: "quit" },
        ]}
      />
    </Box>
  );
}

/**
 * Stable identity key for a draft source. URL works for web/youtube; for
 * local_file we combine path + citation so two distinct local files don't
 * collide on an empty url.
 */
function sourceKey(s: SourceSpec): string {
  if (s.type === "local_file") {
    return `local|${s.path ?? ""}|${s.citation ?? ""}`;
  }
  return `url|${s.url}`;
}

function loadDraftSources(path: string): ParseResult {
  if (!existsSync(path)) return { ok: false, error: `${path} not on disk` };
  let raw: unknown;
  try {
    raw = YAML.parse(readFileSync(path, "utf8"));
  } catch (e) {
    return { ok: false, error: (e as Error).message };
  }
  if (!raw || typeof raw !== "object") {
    return { ok: false, error: "draft is empty or not a YAML mapping" };
  }
  const sources = (raw as Record<string, unknown>)["sources"];
  if (!Array.isArray(sources)) {
    return { ok: false, error: "draft has no `sources:` list at the top level" };
  }

  const parsed: SourceSpec[] = [];
  for (let i = 0; i < sources.length; i++) {
    const entry = sources[i] as Record<string, unknown> | null;
    if (!entry || typeof entry !== "object") {
      return { ok: false, error: `sources[${i}] is not a mapping` };
    }
    const type = String(entry["type"] ?? "web") as SourceSpec["type"];
    if (type !== "web" && type !== "youtube" && type !== "local_file") {
      return { ok: false, error: `sources[${i}].type is invalid: ${type}` };
    }
    parsed.push({
      type,
      url: String(entry["url"] ?? ""),
      note: typeof entry["note"] === "string" ? (entry["note"] as string) : null,
      section: typeof entry["section"] === "string" ? (entry["section"] as string) : null,
      priority:
        (entry["priority"] as SourceSpec["priority"] | undefined) === "optional"
          ? "optional"
          : "required",
      render: (entry["render"] as SourceSpec["render"] | null | undefined) ?? null,
      path: typeof entry["path"] === "string" ? (entry["path"] as string) : null,
      citation:
        typeof entry["citation"] === "string" ? (entry["citation"] as string) : null,
      kind:
        typeof entry["kind"] === "string"
          ? ((entry["kind"] as string) as SourceSpec["kind"])
          : null,
    });
  }

  return { ok: true, sources: parsed };
}

function groupBySection(sources: SourceSpec[]): { section: string; count: number }[] {
  const map = new Map<string, number>();
  for (const s of sources) {
    const key = s.section?.trim() || "(unsectioned)";
    map.set(key, (map.get(key) ?? 0) + 1);
  }
  return Array.from(map.entries()).map(([section, count]) => ({ section, count }));
}

function kindChip(kind: SourceSpec["kind"]): string {
  switch (kind) {
    case "reference":
      return "[ref]";
    case "principle":
      return "[pri]";
    case "prescription":
      return "[pre]";
    case "example":
      return "[ex]";
    default:
      return "";
  }
}

function iconFor(source: SourceSpec): string {
  if (source.type === "youtube") return "▶";
  if (source.type === "local_file") return "▣";
  if (source.render === "js") return "◍";
  if (source.render === "server") return "◎";
  return "◌";
}

function labelFor(source: SourceSpec): string {
  if (source.type === "local_file") {
    return source.citation || source.path || "(unset local file)";
  }
  return source.url || "(no url)";
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}

/**
 * Append a curator-added URL to the working draft yaml file. Type is
 * auto-detected (youtube vs. web). Note/section are left empty for the
 * curator to refine later via tape.yaml.
 *
 * Re-reads + serializes the file so the new entry survives a TUI restart.
 * Skips silently if the URL is already present — re-adding the same URL is
 * harmless and prevents duplicates if the curator pastes twice.
 */
function appendUrlToDraft(draftPath: string, url: string): void {
  if (!existsSync(draftPath)) {
    throw new Error(`draft not on disk: ${draftPath}`);
  }
  const raw = YAML.parse(readFileSync(draftPath, "utf8")) as Record<string, unknown> | null;
  if (!raw || typeof raw !== "object") {
    throw new Error("draft yaml has no top-level mapping");
  }
  const sources = Array.isArray(raw["sources"]) ? (raw["sources"] as unknown[]) : [];
  for (const existing of sources) {
    if (existing && typeof existing === "object") {
      const eu = (existing as Record<string, unknown>)["url"];
      if (typeof eu === "string" && eu.trim() === url) {
        return; // already present; no-op
      }
    }
  }
  const detected = detectSourceType(url);
  const entry: Record<string, unknown> = {
    type: detected,
    url,
    note: "Added by curator.",
    section: "curator-added",
    priority: "required",
  };
  sources.push(entry);
  raw["sources"] = sources;
  writeFileSync(draftPath, YAML.stringify(raw), "utf8");
}
