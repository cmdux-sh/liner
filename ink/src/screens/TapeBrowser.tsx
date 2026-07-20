import React, { useEffect, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdout } from "ink";
import { rmSync } from "node:fs";
import { join } from "node:path";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { useApp } from "../store.js";
import { MUTED, ON_FILL, ERROR, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { readTape } from "../yaml-io.js";
import { runShare } from "../ipc.js";
import type { ProjectSummary } from "../types.js";

export function TapeBrowser(): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { stdout } = useStdout();
  const [selected, setSelected] = useState(0);
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);
  const [sharing, setSharing] = useState(false);

  // Re-scan the mixtapes folder every time the browser is mounted (i.e. on
  // first paint and on every back-navigation from a child screen). Catches
  // mixtapes added out-of-band — e.g. `liner replay <other>` writing a new
  // folder while the TUI was on another screen, or someone unzipping a
  // .mixtape archive directly into the folder.
  useEffect(() => {
    void app.refreshProjects();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useInput((input, key) => {
    if (confirmDelete !== null) {
      if (input === "y") {
        const target = app.projects[confirmDelete];
        if (target) {
          try {
            rmSync(target.path, { recursive: true, force: true });
            app.setNotification({
              kind: "info",
              message: `Deleted ${target.name}`,
            });
          } catch (e) {
            app.setNotification({
              kind: "error",
              message: `Delete failed: ${(e as Error).message}`,
            });
          }
        }
        setConfirmDelete(null);
        void app.refreshProjects();
      } else if (input === "n" || key.escape) {
        setConfirmDelete(null);
      }
      return;
    }

    if (sharing) return;

    if (key.upArrow || input === "k") {
      if (app.projects.length === 0) return;
      setSelected((s) => Math.max(0, s - 1));
    } else if (key.downArrow || input === "j") {
      if (app.projects.length === 0) return;
      setSelected((s) => Math.min(Math.max(0, app.projects.length - 1), s + 1));
    } else if (key.return || input === "o") {
      const target = app.projects[selected];
      if (!target) return;
      try {
        const { tape } = readTape(join(target.path, "tape.yaml"));
        app.navigate({ kind: "hub", tape, folder: target.path });
      } catch (e) {
        app.setNotification({
          kind: "error",
          message: `Open failed: ${(e as Error).message}`,
        });
      }
    } else if (input === "e") {
      const target = app.projects[selected];
      if (!target) return;
      try {
        const { tape } = readTape(join(target.path, "tape.yaml"));
        app.navigate({ kind: "editor", tape, folder: target.path });
      } catch (e) {
        app.setNotification({
          kind: "error",
          message: `Open failed: ${(e as Error).message}`,
        });
      }
    } else if (input === "n") {
      app.navigate({ kind: "newProject" });
    } else if (input === "c") {
      const target = app.projects[selected];
      if (!target) return;
      try {
        const { tape } = readTape(join(target.path, "tape.yaml"));
        app.navigate({ kind: "compile", tape, folder: target.path });
      } catch (e) {
        app.setNotification({
          kind: "error",
          message: `Open failed: ${(e as Error).message}`,
        });
      }
    } else if (input === "s") {
      const target = app.projects[selected];
      if (!target) return;
      setSharing(true);
      void runShare(target.path)
        .then((archivePath) => {
          app.setNotification({ kind: "info", message: `Shared → ${archivePath}` });
        })
        .catch((e: Error) => {
          app.setNotification({ kind: "error", message: e.message });
        })
        .finally(() => setSharing(false));
    } else if (input === "i") {
      app.navigate({ kind: "import" });
    } else if (input === "d") {
      if (app.projects[selected]) setConfirmDelete(selected);
    } else if (input === "r") {
      void app.refreshProjects();
    } else if (key.escape || input === "b") {
      // Esc / b returns to whatever launched this screen (splash on cold
      // start; whatever was upstream when navigating back via history). The
      // browser was the only top-level screen with no back affordance —
      // matching the rest of the app's convention here.
      app.back();
    } else if (input === "q") {
      ink.exit();
    }
  });

  return (
    <Box flexDirection="column">
      <Header
        title="mixtape browser"
        subtitle={`${app.projects.length} mixtape${app.projects.length === 1 ? "" : "s"} · saved in ${app.baseDir}`}
      />

      {app.projects.length === 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={MUTED}>No mixtapes yet in this folder.</Text>
          <Text color={MUTED}>
            Press <Text color={STRUCTURE}>n</Text> to create your first one,{" "}
            <Text color={STRUCTURE}>i</Text> to import a .mixtape archive, or set{" "}
            <Text color={STRUCTURE}>LINER_DIR</Text> and restart to point at a different folder.
          </Text>
        </Box>
      ) : (
        <Box flexDirection="column" marginBottom={1}>
          <ProjectTable
            projects={app.projects}
            selected={selected}
            termRows={stdout?.rows ?? 24}
            termCols={stdout?.columns ?? 80}
          />
        </Box>
      )}

      {sharing ? (
        <Box marginBottom={1}>
          <Text color={WARNING}>Sharing…</Text>
        </Box>
      ) : null}

      {confirmDelete !== null ? (
        <Box marginBottom={1}>
          <Text color={WARNING}>
            Delete folder {app.projects[confirmDelete]?.name}? ({"y/n"}){" "}
            <Text color={MUTED}>(removes the entire mixtape directory)</Text>
          </Text>
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
          { key: "↑↓", label: "select" },
          { key: "enter", label: "open" },
          { key: "e", label: "edit sources" },
          { key: "n", label: "new" },
          { key: "c", label: "compile" },
          { key: "s", label: "share" },
          { key: "i", label: "import" },
          { key: "d", label: "delete" },
          { key: "r", label: "refresh" },
          { key: "esc", label: "back" },
          { key: "q", label: "quit" },
        ]}
      />
    </Box>
  );
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}

// ---------------------------------------------------------------------------
// Columnar project list. Inspired by the Daytona `daytona list` view —
// a real table with a header row, fixed-width columns, and a selected row
// highlighted with cyan background.
// ---------------------------------------------------------------------------

type ColumnKey = "marker" | "folder" | "title" | "sources" | "mode" | "touched";

const COLUMN_ORDER: ColumnKey[] = ["marker", "folder", "title", "sources", "mode", "touched"];
const COLUMN_LABELS: Record<ColumnKey, string> = {
  marker: "",
  folder: "Folder",
  title: "Title",
  sources: "Sources",
  mode: "Mode",
  touched: "Last touched",
};

/**
 * Column widths sized to the terminal width. The fixed columns
 * (marker/sources/mode/touched) take a constant slice; folder and title split
 * whatever's left. Sizing to the terminal — instead of the old hard-coded
 * 91-column total — is what stops rows from wrapping on a narrow window. A
 * wrapped row doubles its height and bleeds its selection background across
 * lines (the garbled selected-row artifact).
 */
function computeWidths(cols: number): Record<ColumnKey, number> {
  const marker = 2;
  const sources = 8;
  const mode = 6;
  const touched = 12;
  const fixed = marker + sources + mode + touched;
  const remaining = Math.max(20, cols - fixed - 1);
  const folder = Math.min(22, Math.max(10, Math.floor(remaining * 0.45)));
  const title = Math.min(40, Math.max(10, remaining - folder));
  return { marker, folder, title, sources, mode, touched };
}

function ProjectTable({
  projects,
  selected,
  termRows,
  termCols,
}: {
  projects: ProjectSummary[];
  selected: number;
  termRows: number;
  termCols: number;
}): React.ReactElement {
  const widths = computeWidths(termCols);

  // Vertical viewport. Render at most `maxRows` rows, windowed around the
  // selection, so the whole screen never exceeds the terminal height. When a
  // screen renders taller than the viewport the terminal scrolls and Ink can
  // no longer erase the lines that scrolled off the top — so it reprints the
  // header on every keystroke (the stacked-header laddering). Capping the
  // height keeps every redraw in place. ~12 rows are reserved for chrome
  // (header, column row, JTBD line, notifications, key-hint navbar).
  const maxRows = Math.max(3, termRows - 12);
  const total = projects.length;
  let start = 0;
  if (total > maxRows) {
    start = Math.min(Math.max(0, selected - Math.floor(maxRows / 2)), total - maxRows);
  }
  const end = Math.min(total, start + maxRows);
  const visible = projects.slice(start, end);
  const hiddenAbove = start;
  const hiddenBelow = total - end;

  const selectedJtbd = projects[selected]?.jtbd;

  return (
    <Box flexDirection="column">
      <Box>
        {COLUMN_ORDER.map((key) => (
          <Text key={key} color={MUTED}>
            {pad(COLUMN_LABELS[key], widths[key])}
          </Text>
        ))}
      </Box>
      {hiddenAbove > 0 ? (
        <Text color={MUTED}>{`  ↑ ${hiddenAbove} more above`}</Text>
      ) : null}
      {visible.map((project, idx) => (
        <ProjectRow
          key={project.path}
          project={project}
          isSelected={start + idx === selected}
          widths={widths}
        />
      ))}
      {hiddenBelow > 0 ? (
        <Text color={MUTED}>{`  ↓ ${hiddenBelow} more below`}</Text>
      ) : null}
      {selectedJtbd ? (
        <Box marginTop={1}>
          <Text color={MUTED} wrap="truncate">{`JTBD: ${selectedJtbd}`}</Text>
        </Box>
      ) : null}
    </Box>
  );
}

function ProjectRow({
  project,
  isSelected,
  widths,
}: {
  project: ProjectSummary;
  isSelected: boolean;
  widths: Record<ColumnKey, number>;
}): React.ReactElement {
  const values = renderValues(project, isSelected, widths);
  // Render the whole row as a single truncating line rather than a flex row of
  // separate cells. A single line lets Ink truncate the overflow instead of
  // wrapping it onto a second line — no height surprise, and no selection
  // background bleeding across wrapped lines.
  if (isSelected) {
    const line = COLUMN_ORDER.map((key) => pad(values[key], widths[key])).join("");
    return (
      <Text wrap="truncate" backgroundColor={STRUCTURE} color={ON_FILL} bold>
        {line}
      </Text>
    );
  }
  return (
    <Text wrap="truncate">
      <Text color={STRUCTURE}>{pad(values.marker, widths.marker)}</Text>
      <Text color={MUTED}>{pad(values.folder, widths.folder)}</Text>
      <Text bold>{pad(values.title, widths.title)}</Text>
      <Text>{pad(values.sources, widths.sources)}</Text>
      <Text color={MUTED}>{pad(values.mode, widths.mode)}</Text>
      <Text color={MUTED}>{pad(values.touched, widths.touched)}</Text>
    </Text>
  );
}

function renderValues(
  p: ProjectSummary,
  isSelected: boolean,
  widths: Record<ColumnKey, number>,
): Record<ColumnKey, string> {
  return {
    marker: isSelected ? "▶ " : "  ",
    folder: truncate(p.name, widths.folder - 1),
    title: truncate(p.title || "(untitled)", widths.title - 1),
    sources: String(p.source_count),
    mode: p.mode ?? "—",
    touched: relativeTime(p.modified_iso),
  };
}

function pad(s: string, w: number): string {
  if (s.length >= w) return s.slice(0, w);
  return s + " ".repeat(w - s.length);
}

/**
 * "Xm ago" / "Xh ago" / "Xd ago" — falls back to the bare date string if the
 * timestamp can't be parsed or is more than a year old. Matches the
 * Daytona-style "10 min ago" cadence shown in the daytona list screenshot.
 */
function relativeTime(iso: string): string {
  if (!iso) return "—";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "—";
  const seconds = Math.floor((Date.now() - t) / 1000);
  if (seconds < 45) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  if (days < 30) return `${Math.floor(days / 7)}w ago`;
  if (days < 365) return `${Math.floor(days / 30)}mo ago`;
  return iso.slice(0, 10);
}
