import React, { useMemo, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdout } from "ink";
import BigText from "ink-big-text";
import Gradient from "ink-gradient";
import { useApp } from "../store.js";
import { MUTED, CURRENT, HEADING, PALETTE, STRUCTURE } from "../colors.js";
import { VERSION } from "../version.js";
import {
  SMALL_CASSETTE_ROWS,
  SMALL_CASSETTE_WIDTH,
} from "./cassette-art.js";

// First-paint splash. An ASCII-art logo (rendered as foreground text in the
// brand color) above a real interactive launch menu. Arrows move the
// selection; Enter activates it; `n` / `i` / `o` are direct hotkeys.
//
// The compact (~66×19) cassette is the default — it fits comfortably in
// every reasonable terminal width and the visual balance against the menu
// rows below is tighter than the maximalist 100×29 version. The full
// CASSETTE_ROWS export still lives in cassette-art.ts for any future
// "show the big one" affordance, but the splash never auto-promotes.
// Skipped via `shouldShowSplash()` when stdout isn't a TTY or
// LINER_NO_SPLASH is set.

type Action = "new" | "import" | "open" | "settings";

type Item = {
  action: Action;
  hotkey: string;
  label: string;
  hint: string;
};

const ITEMS: Item[] = [
  { action: "new", hotkey: "n", label: "New Liner", hint: "open the curation wizard" },
  { action: "import", hotkey: "i", label: "Import Project", hint: "load a .mixtape file" },
  { action: "open", hotkey: "o", label: "Open existing", hint: "browse your mixtapes folder" },
  { action: "settings", hotkey: "s", label: "Settings", hint: "change AI agent" },
];

export function BootSplash(): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { stdout } = useStdout();
  const cols = stdout?.columns ?? 80;
  // Show the compact cassette whenever it fits; fall back to the wordmark
  // on terminals too narrow for it.
  const showCassette = cols >= SMALL_CASSETTE_WIDTH;
  const [selected, setSelected] = useState(0);

  function dispatch(action: Action): void {
    switch (action) {
      case "new":
        app.navigate({ kind: "newProject" });
        return;
      case "import":
        app.navigate({ kind: "import" });
        return;
      case "open":
        app.navigate({ kind: "browser" });
        return;
      case "settings":
        app.navigate({ kind: "agentSetup" });
        return;
    }
  }

  useInput((input, key) => {
    if (key.upArrow || input === "k") {
      setSelected((i) => (i - 1 + ITEMS.length) % ITEMS.length);
    } else if (key.downArrow || input === "j") {
      setSelected((i) => (i + 1) % ITEMS.length);
    } else if (input === "n") {
      dispatch("new");
    } else if (input === "i") {
      dispatch("import");
    } else if (input === "o") {
      dispatch("open");
    } else if (input === "s") {
      dispatch("settings");
    } else if (key.return || input === " ") {
      dispatch(ITEMS[selected]!.action);
    } else if (input === "q") {
      ink.exit();
    }
    // No esc handler: the splash is home. Esc anywhere else walks back to
    // here and stops — esc on home itself is a deliberate no-op.
  });

  return (
    <Box flexDirection="column" paddingY={1}>
      {showCassette ? (
        <CassetteLogo rows={SMALL_CASSETTE_ROWS} />
      ) : (
        <Gradient name="vice">
          <BigText text="liner" font="block" />
        </Gradient>
      )}
      <Box flexDirection="column" paddingLeft={2} marginTop={showCassette ? 1 : -1}>
        <Box>
          <Text color={STRUCTURE} bold>liner</Text>
          <Text color={MUTED}>  ·  curate AI-ready mixtapes from any set of sources</Text>
        </Box>
        <Text color={MUTED}>v{VERSION}</Text>
      </Box>
      <Box flexDirection="column" paddingLeft={2} marginTop={1}>
        <Text color={HEADING} bold>Get started</Text>
        <Box marginTop={1} flexDirection="column">
          {ITEMS.map((item, i) => (
            <MenuRow key={item.action} item={item} active={i === selected} />
          ))}
        </Box>
      </Box>
      <Box marginTop={1} paddingLeft={2}>
        <Text color={MUTED}>↑↓ navigate  ·  </Text>
        <Text color={STRUCTURE}>enter</Text>
        <Text color={MUTED}> select  ·  </Text>
        <Text color={STRUCTURE}>[n]</Text>
        <Text color={MUTED}> </Text>
        <Text color={STRUCTURE}>[i]</Text>
        <Text color={MUTED}> </Text>
        <Text color={STRUCTURE}>[o]</Text>
        <Text color={MUTED}> </Text>
        <Text color={STRUCTURE}>[s]</Text>
        <Text color={MUTED}> direct shortcuts  ·  </Text>
        <Text color={STRUCTURE}>q</Text>
        <Text color={MUTED}> quit</Text>
      </Box>
    </Box>
  );
}

/**
 * Render the ASCII cassette logo with every non-space character painted in
 * a randomly-chosen palette color. The randomization is computed once per
 * mount (via useMemo) — re-randomizing on every re-render would flicker.
 *
 * For perf we group consecutive characters that landed on the same color
 * into runs so each row ships a handful of `Text` nodes instead of one per
 * character — Ink mounts a separate element per `<Text>`, and 66 elements
 * × 19 rows × every re-paint adds up.
 *
 * Spaces are emitted as their own color-less run so we don't burn nodes
 * coloring invisible cells.
 */
function CassetteLogo({ rows }: { rows: readonly string[] }): React.ReactElement {
  const colored = useMemo(() => rows.map(colorizeRow), [rows]);
  return (
    <Box flexDirection="column">
      {colored.map((runs, rowIdx) => (
        <Box key={rowIdx}>
          {runs.map((run, runIdx) => (
            <Text key={runIdx} color={run.color}>
              {run.chars}
            </Text>
          ))}
        </Box>
      ))}
    </Box>
  );
}

type Run = { color: string | undefined; chars: string };

// Logo coloring mode:
//   "solid"  — every glyph in the brand orange (STRUCTURE). Current default.
//   "random" — each glyph gets a random palette color (the original look).
// Flip this to "random" to bring back the multicolor cassette.
const LOGO_COLOR_MODE: "solid" | "random" = "solid";

function glyphColor(): string {
  return LOGO_COLOR_MODE === "solid"
    ? STRUCTURE
    : PALETTE[Math.floor(Math.random() * PALETTE.length)]!;
}

function colorizeRow(row: string): Run[] {
  const runs: Run[] = [];
  for (const ch of row) {
    // Spaces don't need coloring — collapse them into uncolored runs so we
    // don't allocate a color per blank cell.
    const color = ch === " " ? undefined : glyphColor();
    const tail = runs[runs.length - 1];
    if (tail && tail.color === color) {
      tail.chars += ch;
    } else {
      runs.push({ color, chars: ch });
    }
  }
  return runs;
}

function MenuRow({ item, active }: { item: Item; active: boolean }): React.ReactElement {
  // ASCII ">" is guaranteed single-cell. The unicode ▶ glyph renders
  // double-wide in many terminal fonts, which made the inactive rows shift
  // every time the user moved the selection. Using a fixed two-char prefix
  // ("> " or "  ") keeps every row anchored at the same column.
  return (
    <Box>
      <Text color={active ? CURRENT : undefined} bold={active}>
        {active ? "> " : "  "}
      </Text>
      <Text color={STRUCTURE}>{`[${item.hotkey}]`}</Text>
      <Text>{"  "}</Text>
      <Text color={active ? CURRENT : undefined} bold={active}>
        {item.label.padEnd(18, " ")}
      </Text>
      <Text color={MUTED}>{item.hint}</Text>
    </Box>
  );
}

export function shouldShowSplash(): boolean {
  if (process.env["LINER_NO_SPLASH"]) return false;
  if (!process.stdout.isTTY) return false;
  // Narrow terminals (< 60 cols) can't render the big-text wordmark well —
  // skip rather than show a clipped logo.
  const cols = process.stdout.columns ?? 80;
  if (cols < 60) return false;
  return true;
}
