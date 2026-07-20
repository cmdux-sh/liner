import React from "react";
import { Box, Text } from "ink";
import { MUTED, STRUCTURE } from "../colors.js";

export type NextStep = {
  /** Single-line label or instruction. */
  label: string;
  /** Optional inline hint shown dim-colored after the label. */
  hint?: string;
};

/**
 * Bordered callout listing what to do next. Modeled on `clack`'s `p.note()`.
 * Use after a wizard completes, a compile succeeds, or any other moment
 * where the user just finished a discrete unit of work.
 */
export function NextSteps({
  title = "Next steps",
  items,
  color = STRUCTURE,
}: {
  title?: string;
  items: NextStep[];
  color?: string;
}): React.ReactElement {
  return (
    <Box
      flexDirection="column"
      borderStyle="round"
      borderColor={color}
      paddingX={2}
      paddingY={0}
      marginBottom={1}
    >
      <Box marginBottom={1}>
        <Text color={color} bold>
          {title}
        </Text>
      </Box>
      {items.map((item, i) => (
        <Box key={i}>
          <Text color={color}>{`${i + 1}. `}</Text>
          <Text>{item.label}</Text>
          {item.hint ? (
            <Text color={MUTED}>{"  " + item.hint}</Text>
          ) : null}
        </Box>
      ))}
    </Box>
  );
}
