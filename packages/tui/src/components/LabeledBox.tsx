import React from "react";
import { Box, Text, useStdout } from "ink";
import { ON_FILL, STRUCTURE } from "../colors.js";

// Charm/Lipgloss-style card: a bordered box with a chip on the top border line.
// Inspired by the Daytona TUI screenshots — the chip provides a clear card label
// that doesn't waste a content line.
//
// Implementation: Ink 5 lets us toggle each side of a border independently
// (borderTop={false}). We render the top border ourselves as a manual row of
// characters with the chip embedded, then let Ink draw the side + bottom
// borders normally. The corner glyphs must match between the manual top row
// and Ink's border style — both use the same "round" or "single" set.

const ROUND = {
  topLeft: "╭",
  topRight: "╮",
  h: "─",
} as const;

const SINGLE = {
  topLeft: "┌",
  topRight: "┐",
  h: "─",
} as const;

const DOUBLE = {
  topLeft: "╔",
  topRight: "╗",
  h: "═",
} as const;

type BorderStyle = "round" | "single" | "double";

type Props = {
  /** Chip text. Padded with one space on each side automatically. */
  label: string;
  /**
   * Border color. Defaults to STRUCTURE (cyan) — see colors.ts. Override only
   * when the card itself is REPORTING a terminal state (SUCCESS / ERROR /
   * WARNING). Borders don't move with progress; only outcome.
   */
  color?: string;
  /**
   * Chip background. Defaults to the same color as the border so the chip
   * reads as part of the same frame — a solid section of the top border line
   * rather than a floating island. Override to CURRENT (yellowBright) on the
   * "you are here" card, SUCCESS (green) on cards reporting done, or ERROR
   * (red) on cards reporting fail. Passing `undefined` keeps the matched-to-
   * border behavior; passing an explicit value (including "white") opts out.
   */
  labelBg?: string;
  /** Chip foreground. Defaults to black so the chip reads well over any bg. */
  labelColor?: string;
  borderStyle?: BorderStyle;
  /** Explicit width in cells. Falls back to terminal columns minus margin. */
  width?: number;
  /** Inner horizontal padding (cells). Default 2 — matches the Daytona look. */
  paddingX?: number;
  /**
   * Inner vertical padding (rows). Default 1 so the chip on the top border
   * never butts directly against the first content row — especially when the
   * chip has a colored background (yellow / green / red) and would otherwise
   * crowd the text underneath.
   */
  paddingY?: number;
  children?: React.ReactNode;
};

export function LabeledBox({
  label,
  color = STRUCTURE,
  labelBg,
  labelColor = ON_FILL,
  borderStyle = "round",
  width,
  paddingX = 2,
  paddingY = 1,
  children,
}: Props): React.ReactElement {
  // Default chip bg = border color, so the chip looks like a solid section
  // of the top border. Callers can opt out by passing a different bg.
  const resolvedLabelBg = labelBg ?? color;
  const { stdout } = useStdout();
  const cols = stdout?.columns ?? 80;
  // Default width: terminal minus a 2-cell right margin so the box doesn't
  // butt against the edge of the pane. Floor at 40 so things don't collapse
  // in very narrow terminals.
  const w = Math.max(40, width ?? cols - 2);
  const chars = borderStyle === "round" ? ROUND : borderStyle === "double" ? DOUBLE : SINGLE;

  // Truncate the label if it would overflow the top row.
  // Budget: 2 corners + 2 leading dashes + " label " + at least 1 filler dash.
  const maxLabelLen = Math.max(4, w - 2 - 2 - 2 - 1);
  const safeLabel = label.length > maxLabelLen ? label.slice(0, maxLabelLen - 1) + "…" : label;
  const labelText = ` ${safeLabel} `;

  const leadDashes = 2;
  const filler = Math.max(1, w - 2 - leadDashes - labelText.length);

  return (
    <Box flexDirection="column" width={w}>
      {/* Manual top row: corner + lead + chip + filler + corner */}
      <Box width={w}>
        <Text color={color}>
          {chars.topLeft + chars.h.repeat(leadDashes)}
        </Text>
        <Text backgroundColor={resolvedLabelBg} color={labelColor} bold>
          {labelText}
        </Text>
        <Text color={color}>
          {chars.h.repeat(filler) + chars.topRight}
        </Text>
      </Box>
      {/* Ink-drawn sides + bottom. No top border so it joins with our manual row. */}
      <Box
        borderStyle={borderStyle}
        borderColor={color}
        borderTop={false}
        flexDirection="column"
        width={w}
        paddingX={paddingX}
        paddingY={paddingY}
      >
        {children}
      </Box>
    </Box>
  );
}
