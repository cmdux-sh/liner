import { spawn } from "node:child_process";

/**
 * Open a folder in the OS file manager. Cross-platform, best-effort, never
 * throws — opening the folder is a convenience, not load-bearing.
 *
 *   macOS   → `open <path>`
 *   Windows → `explorer <path>`
 *   Linux   → `xdg-open <path>`
 *
 * Spawned detached + unref'd so it doesn't tie the file manager's lifetime to
 * the TUI, and stdio is ignored so the opener can't corrupt the terminal.
 */
export function openFolder(path: string): boolean {
  const cmd =
    process.platform === "darwin"
      ? "open"
      : process.platform === "win32"
        ? "explorer"
        : "xdg-open";
  try {
    const child = spawn(cmd, [path], { detached: true, stdio: "ignore" });
    child.unref();
    return true;
  } catch {
    return false;
  }
}
