import React, { useMemo, useState } from "react";
import { Box, Text, useInput } from "ink";
import TextInput from "ink-text-input";
import { existsSync, mkdirSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { KeyHints } from "../components/KeyHints.js";
import { MUTED, HEADING, STRUCTURE } from "../colors.js";

const SUPPORTED_EXTENSIONS = new Set([".md", ".txt", ".html", ".htm", ".pdf"]);

type FileEntry = {
  /** Path relative to the project folder, e.g. "personal/foo.pdf". */
  relPath: string;
  /** Absolute filesystem path. */
  absPath: string;
  basename: string;
  size: number;
  ext: string;
  supported: boolean;
};

export function LocalFilePicker({
  projectFolder,
  initialPath,
  onPick,
  onCancel,
}: {
  projectFolder: string;
  initialPath?: string | null;
  onPick: (relPath: string) => void;
  onCancel: () => void;
}): React.ReactElement {
  const personalDir = join(projectFolder, "personal");
  // Lazily ensure personal/ exists.
  if (!existsSync(personalDir)) {
    mkdirSync(personalDir, { recursive: true });
  }

  const entries = useMemo<FileEntry[]>(() => listEntries(personalDir, projectFolder), [
    personalDir,
    projectFolder,
  ]);

  const initialIndex = useMemo(() => {
    if (!initialPath) return 0;
    const i = entries.findIndex((e) => e.relPath === initialPath);
    return i >= 0 ? i : 0;
  }, [entries, initialPath]);

  const [selected, setSelected] = useState(initialIndex);
  const [pasteMode, setPasteMode] = useState(false);
  const [pasteValue, setPasteValue] = useState(initialPath ?? "personal/");

  useInput((input, key) => {
    if (pasteMode) {
      if (key.escape) setPasteMode(false);
      return;
    }
    if (key.escape) {
      onCancel();
      return;
    }
    if (key.upArrow || input === "k") {
      if (entries.length > 0) setSelected((s) => Math.max(0, s - 1));
    } else if (key.downArrow || input === "j") {
      if (entries.length > 0) setSelected((s) => Math.min(entries.length - 1, s + 1));
    } else if (key.return) {
      const entry = entries[selected];
      if (!entry) return;
      if (!entry.supported) return;
      onPick(entry.relPath);
    } else if (input === "n") {
      setPasteMode(true);
    }
  });

  if (pasteMode) {
    return (
      <Box flexDirection="column">
        <Text color={HEADING} bold>
          New local file — paste a path relative to the project folder
        </Text>
        <Text color={MUTED}>{personalDir}/...</Text>
        <Box marginTop={1}>
          <Text color={STRUCTURE}>path: </Text>
          <TextInput
            value={pasteValue}
            onChange={setPasteValue}
            onSubmit={(v) => {
              const cleaned = v.trim();
              if (!cleaned) return;
              onPick(cleaned);
            }}
            placeholder="personal/foo.pdf"
          />
        </Box>
        <Box marginTop={1}>
          <KeyHints
            hints={[
              { key: "enter", label: "use this path" },
              { key: "esc", label: "back" },
            ]}
          />
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Text color={HEADING} bold>
        Select a file from personal/
      </Text>
      <Text color={MUTED}>{personalDir}</Text>

      {entries.length === 0 ? (
        <Box marginTop={1} flexDirection="column">
          <Text color={MUTED}>(empty — put files into personal/ first, or press `n` to paste a path)</Text>
        </Box>
      ) : (
        <Box flexDirection="column" marginTop={1} marginBottom={1}>
          {entries.map((entry, i) => {
            const isSelected = i === selected;
            const tag = entry.supported ? entry.ext.slice(1) : `${entry.ext.slice(1)} (unsupported)`;
            return (
              <Box key={entry.relPath}>
                <Text color={isSelected ? STRUCTURE : entry.supported ? undefined : undefined}>
                  {isSelected ? "▶ " : "  "}
                  {entry.basename}
                </Text>
                <Text color={MUTED}>
                  {"  "}
                  {formatSize(entry.size)} · {tag}
                </Text>
              </Box>
            );
          })}
        </Box>
      )}

      <KeyHints
        hints={[
          { key: "↑↓", label: "select" },
          { key: "enter", label: "use" },
          { key: "n", label: "paste path" },
          { key: "esc", label: "cancel" },
        ]}
      />
    </Box>
  );
}

function listEntries(personalDir: string, projectFolder: string): FileEntry[] {
  if (!existsSync(personalDir)) return [];
  const out: FileEntry[] = [];
  for (const name of readdirSync(personalDir)) {
    const abs = join(personalDir, name);
    try {
      const stat = statSync(abs);
      if (!stat.isFile()) continue;
      const ext = extOf(name);
      out.push({
        relPath: relative(projectFolder, abs),
        absPath: abs,
        basename: name,
        size: stat.size,
        ext,
        supported: SUPPORTED_EXTENSIONS.has(ext),
      });
    } catch {
      // skip
    }
  }
  return out.sort((a, b) => a.basename.localeCompare(b.basename));
}

function extOf(name: string): string {
  const dot = name.lastIndexOf(".");
  return dot < 0 ? "" : name.slice(dot).toLowerCase();
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
