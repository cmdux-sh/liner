import React, { useEffect, useState } from "react";
import { join } from "node:path";
import { Box, Text, useInput } from "ink";
import TextInput from "ink-text-input";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { ERROR, HEADING, MUTED, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { projectFolder, readTape, writeTape } from "../yaml-io.js";
import { useApp } from "../store.js";
import { ensureLocalSourceFolders, importSourceInboxText } from "../source-inbox.js";
import type { SourceSpec, Tape } from "../types.js";

export function SourceInbox({
  tape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();
  const [value, setValue] = useState("");
  const [imported, setImported] = useState<SourceSpec[]>([]);
  const [warnings, setWarnings] = useState<string[]>([]);

  useEffect(() => {
    try {
      ensureLocalSourceFolders(folder);
    } catch (e) {
      app.setNotification({ kind: "error", message: (e as Error).message });
    }
  }, [app, folder]);

  function importNow(input: string): void {
    const result = importSourceInboxText(input, folder);
    if (result.sources.length === 0) {
      setWarnings(result.warnings.length > 0 ? result.warnings : ["No sources recognized."]);
      return;
    }
    const nextTape = { ...tape, sources: [...tape.sources, ...result.sources] };
    try {
      const project = projectFolder(folder);
      const { doc } = readTape(project.tapePath);
      writeTape(project.tapePath, nextTape, doc);
    } catch (e) {
      app.setNotification({ kind: "error", message: (e as Error).message });
      return;
    }
    setImported(result.sources);
    setWarnings(result.warnings);
    const skipped = result.warnings.length;
    app.setNotification({
      kind: "info",
      message: `Imported and saved ${result.sources.length} source${result.sources.length === 1 ? "" : "s"}${skipped > 0 ? `, skipped ${skipped}` : ""}.`,
    });
    app.navigate({ kind: "editor", tape: nextTape, folder });
  }

  useInput((_input, key) => {
    if (key.escape) {
      app.navigate({ kind: "editor", tape, folder });
      return;
    }
  });

  return (
    <Box flexDirection="column">
      <Header title="source inbox" subtitle="paste URLs, files, skills, or article text" />

      <Box flexDirection="column" marginBottom={1}>
        <Text color={MUTED}>
          Paste one source per line, separate URLs/paths with spaces, or paste article text.
        </Text>
        <Text color={MUTED}>
          Article text is saved to local-sources/captured/. Local files are copied into local-sources/.
        </Text>
        <Text color={MUTED}>Drop local files here: {join(folder, "local-sources")}</Text>
        <Text color={MUTED}>For multiple pasted articles, separate each article with: --- source ---</Text>
      </Box>

      <Box marginBottom={1}>
        <Text color={STRUCTURE}>sources: </Text>
        <TextInput
          value={value}
          onChange={setValue}
          onSubmit={importNow}
          placeholder="https://...  ~/Desktop/paper.pdf  terminal-ui  or paste article text"
        />
      </Box>

      <Box flexDirection="column" marginBottom={1}>
        <Text color={HEADING}>Recognizes</Text>
        <Text color={MUTED}>  web URLs, YouTube URLs, GitHub skill URLs</Text>
        <Text color={MUTED}>  absolute file paths, local-sources/... paths</Text>
        <Text color={MUTED}>  installed skill names such as terminal-ui</Text>
        <Text color={MUTED}>  pasted website/article text</Text>
      </Box>

      {warnings.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={WARNING}>Warnings</Text>
          {warnings.slice(0, 4).map((warning) => (
            <Text key={warning} color={MUTED}>
              {"  "}
              {warning}
            </Text>
          ))}
        </Box>
      ) : null}

      {imported.length > 0 ? (
        <Box flexDirection="column" marginBottom={1}>
          <Text color={SUCCESS}>Imported {imported.length} sources</Text>
        </Box>
      ) : null}

      {app.notification?.kind === "error" ? (
        <Box marginBottom={1}>
          <Text color={ERROR}>{app.notification.message}</Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "enter", label: "import" },
          { key: "esc", label: "cancel" },
        ]}
      />
    </Box>
  );
}
