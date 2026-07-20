import React from "react";
import { Box, Text } from "ink";
import { MUTED, ON_FILL, STRUCTURE } from "../colors.js";
import { VERSION } from "../version.js";

export function Header({
  title,
  subtitle,
}: {
  title: string;
  subtitle?: string;
}): React.ReactElement {
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Box>
        <Text backgroundColor={STRUCTURE} color={ON_FILL} bold>
          {" ▶ liner "}
        </Text>
        <Text>{"  "}</Text>
        <Text bold color={STRUCTURE}>
          {title}
        </Text>
        <Text color={MUTED}>{"  ·  v" + VERSION}</Text>
      </Box>
      {subtitle ? (
        <Box marginTop={0}>
          <Text color={MUTED} wrap="truncate">{subtitle}</Text>
        </Box>
      ) : null}
    </Box>
  );
}
