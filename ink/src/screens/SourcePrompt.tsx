import React from "react";
import { Box, Text, useInput } from "ink";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { HEADING, MUTED, STRUCTURE } from "../colors.js";
import { useApp } from "../store.js";
import type { Tape } from "../types.js";

export function SourcePrompt({
  tape,
  folder,
}: {
  tape: Tape;
  folder: string;
}): React.ReactElement {
  const app = useApp();

  function pasteSources(): void {
    app.navigate({ kind: "sourceInbox", tape, folder });
  }

  function openEditor(): void {
    app.navigate({ kind: "editor", tape, folder });
  }

  function skip(): void {
    app.navigate({ kind: "hub", tape, folder });
  }

  useInput((input, key) => {
    if (key.return || input === "p") pasteSources();
    else if (input === "e") openEditor();
    else if (input === "s" || key.escape) skip();
  });

  return (
    <Box flexDirection="column">
      <Header title="bring your own sources?" subtitle={folder} />

      <Box flexDirection="column" marginBottom={1}>
        <Text color={HEADING}>Do you already have sources you want this mixtape to use?</Text>
        <Text color={MUTED}>
          Add them now so the research and evaluation start from your best material.
        </Text>
      </Box>

      <Box flexDirection="column" marginBottom={1}>
        <Text>
          <Text color={STRUCTURE}>enter</Text> paste a batch of URLs, files, and skills
        </Text>
        <Text>
          <Text color={STRUCTURE}>e</Text> open the full source editor
        </Text>
        <Text>
          <Text color={STRUCTURE}>s</Text> skip for now
        </Text>
      </Box>

      <Text color={MUTED}>
        You can always add more later from the source editor with `p`.
      </Text>

      <Box marginTop={1}>
        <KeyHints
          hints={[
            { key: "enter", label: "paste sources" },
            { key: "e", label: "source editor" },
            { key: "s/esc", label: "skip" },
          ]}
        />
      </Box>
    </Box>
  );
}
