import React from "react";
import { render } from "ink";
import { App } from "./App.js";

// Clear the terminal (screen + scrollback) before Ink mounts so the TUI
// starts at the top of a blank canvas — no `npm run dev` lines visible above
// the splash. `\x1B[2J` = erase visible viewport, `\x1B[3J` = erase scrollback,
// `\x1B[H` = home cursor.
if (process.stdout.isTTY) {
  process.stdout.write("\x1B[2J\x1B[3J\x1B[H");
}

const { waitUntilExit } = render(<App />);
waitUntilExit().catch((e: unknown) => {
  // eslint-disable-next-line no-console
  console.error(e);
  process.exit(1);
});
