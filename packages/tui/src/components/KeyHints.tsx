import React from "react";
import { Box, Text } from "ink";
import { MUTED, STRUCTURE } from "../colors.js";

export type Hint = { key: string; label: string };

/**
 * Renders the bottom navbar of keyboard hints — a uniform, dim row of
 * reference bindings. The "what to press next" call-to-action lives in a
 * separate description line above the navbar on each screen (not in this
 * component), so the navbar stays a reference and doesn't compete with the
 * description for attention.
 */
export function KeyHints({ hints }: { hints: Hint[] }): React.ReactElement {
  return (
    <Box flexWrap="wrap" columnGap={2}>
      {hints.map((h) => (
        <Text key={h.key} color={MUTED}>
          <Text color={STRUCTURE}>{h.key}</Text> {h.label}
        </Text>
      ))}
    </Box>
  );
}
