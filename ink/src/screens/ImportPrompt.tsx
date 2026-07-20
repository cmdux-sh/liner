import React, { useMemo, useState } from "react";
import { Box, Text, useInput } from "ink";
import TextInput from "ink-text-input";
import { resolve, isAbsolute, join, basename } from "node:path";
import { existsSync, readdirSync, statSync } from "node:fs";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { useApp } from "../store.js";
import { MUTED, ERROR, HEADING, STRUCTURE, SUCCESS } from "../colors.js";
import { runImport } from "../ipc.js";
import { readTape } from "../yaml-io.js";

type Detected = { name: string; path: string; sizeKb: number };

/** Scan a directory for `*.mixtape` files (non-recursive). Never throws. */
function scanFolder(dir: string): Detected[] {
  try {
    return readdirSync(dir)
      .filter((f) => f.endsWith(".mixtape"))
      .map((f) => {
        const full = join(dir, f);
        let sizeKb = 0;
        try {
          sizeKb = Math.max(1, Math.round(statSync(full).size / 1024));
        } catch {
          /* ignore */
        }
        return { name: f, path: full, sizeKb };
      })
      .sort((a, b) => a.name.localeCompare(b.name));
  } catch {
    return [];
  }
}

export function ImportPrompt(): React.ReactElement {
  const app = useApp();
  const cwd = useMemo(() => process.cwd(), []);
  const [detected, setDetected] = useState<Detected[]>(() => scanFolder(cwd));
  // "list" when there are detected files to pick from; "path" to type a path.
  const [mode, setMode] = useState<"list" | "path">(() =>
    scanFolder(cwd).length > 0 ? "list" : "path",
  );
  const [selected, setSelected] = useState(0);
  const [pathInput, setPathInput] = useState("");
  const [busy, setBusy] = useState(false);

  function rescan(): void {
    const next = scanFolder(cwd);
    setDetected(next);
    setSelected(0);
    if (next.length === 0) setMode("path");
    app.setNotification({
      kind: "info",
      message:
        next.length === 0
          ? "No .mixtape files in this folder."
          : `Found ${next.length} .mixtape file${next.length === 1 ? "" : "s"}.`,
    });
  }

  async function importFrom(path: string): Promise<void> {
    if (!existsSync(path)) {
      app.setNotification({ kind: "error", message: `Not found: ${path}` });
      return;
    }
    setBusy(true);
    try {
      // No auto-refetch — the archive carries the sources; compile later if needed.
      await runImport(path, app.baseDir, { noRefetch: true });
      await app.refreshProjects();
      // Land on the imported mixtape's hub if we can resolve it (archives made
      // by `liner share` extract to <baseDir>/<archive-basename>/). Otherwise
      // fall back to the browser.
      const folderName = basename(path).replace(/\.mixtape$/, "");
      const importedFolder = join(app.baseDir, folderName);
      if (existsSync(join(importedFolder, "tape.yaml"))) {
        const { tape } = readTape(join(importedFolder, "tape.yaml"));
        app.navigate({ kind: "hub", tape, folder: importedFolder });
      } else {
        app.navigate({ kind: "browser" });
      }
    } catch (e) {
      app.setNotification({ kind: "error", message: (e as Error).message });
      setBusy(false);
    }
  }

  useInput((input, key) => {
    if (busy) return;
    if (mode === "list") {
      if (key.upArrow || input === "k") {
        setSelected((s) => Math.max(0, s - 1));
      } else if (key.downArrow || input === "j") {
        setSelected((s) => Math.min(detected.length - 1, s + 1));
      } else if (key.return) {
        const pick = detected[selected];
        if (pick) void importFrom(pick.path);
      } else if (input === "r") {
        rescan();
      } else if (input === "p") {
        setMode("path");
      } else if (key.escape) {
        app.back();
      }
      return;
    }
    // path mode — the TextInput owns most keys; esc returns to the list (or back).
    if (key.escape) {
      if (detected.length > 0) setMode("list");
      else app.back();
    }
  });

  function submitPath(value: string): void {
    const trimmed = value.trim();
    if (!trimmed) {
      app.setNotification({ kind: "error", message: "Type a path to a .mixtape file." });
      return;
    }
    const path = isAbsolute(trimmed) ? trimmed : resolve(cwd, trimmed);
    void importFrom(path);
  }

  return (
    <Box flexDirection="column">
      <Header title="Import Project" subtitle={`will extract into ${join(app.baseDir)}`} />

      {detected.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={MUTED}>Found in this folder ({cwd}):</Text>
          {detected.map((d, i) => {
            const active = mode === "list" && i === selected;
            return (
              <Box key={d.path}>
                <Text color={active ? STRUCTURE : MUTED}>{active ? "▶ " : "  "}</Text>
                <Text color={active ? undefined : MUTED}>
                  {`${i + 1}. ${d.name}`}
                </Text>
                <Text color={MUTED}>{`   ${d.sizeKb} KB`}</Text>
              </Box>
            );
          })}
        </Box>
      ) : (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={MUTED}>No .mixtape files found in this folder ({cwd}).</Text>
          <Text color={MUTED}>Drop a .mixtape here and press `r` to re-scan — or type a path below.</Text>
        </Box>
      )}

      {mode === "path" ? (
        <Box marginBottom={1}>
          <Text color={STRUCTURE}>path: </Text>
          <TextInput
            value={pathInput}
            onChange={setPathInput}
            onSubmit={submitPath}
            placeholder={join("~", "Downloads", "thing.mixtape")}
          />
        </Box>
      ) : null}

      <Box marginBottom={1}>
        <Text color={MUTED}>Sources aren't refetched — press `c` on the imported folder to compile.</Text>
      </Box>

      {busy ? (
        <Box marginBottom={1}>
          <Text color={SUCCESS}>Extracting…</Text>
        </Box>
      ) : null}

      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      <Box marginTop={1}>
        <Text color={HEADING}>
          {mode === "list"
            ? "next · enter to import the selected mixtape"
            : "next · type a path and press enter to import"}
        </Text>
      </Box>

      <Box>
        <KeyHints
          hints={
            mode === "list"
              ? [
                  { key: "↑↓", label: "select" },
                  { key: "enter", label: "import" },
                  { key: "r", label: "re-scan folder" },
                  { key: "p", label: "type a path" },
                  { key: "esc", label: "back" },
                ]
              : [
                  { key: "enter", label: "import path" },
                  { key: "esc", label: detected.length > 0 ? "back to list" : "back" },
                ]
          }
        />
      </Box>
    </Box>
  );
}
