import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Single source of truth for the TUI version. Read from package.json at
// runtime so the header chip, the boot splash, and `liner --version` can
// never drift from the published package version (they used to be three
// separately-edited string constants — BootSplash silently sat at 0.5.0
// through the whole 0.5.2 cycle). The only number to bump on release is
// package.json's `version`.
//
// At runtime this module is dist/version.js, so `..` resolves to the package
// root; under tsx (dev) it's src/version.ts and `..` is the package root too.
function readVersion(): string {
  try {
    const here = dirname(fileURLToPath(import.meta.url));
    const pkg = JSON.parse(readFileSync(join(here, "..", "package.json"), "utf8")) as {
      version?: unknown;
    };
    return typeof pkg.version === "string" ? pkg.version : "unknown";
  } catch {
    return "unknown";
  }
}

export const VERSION = readVersion();
