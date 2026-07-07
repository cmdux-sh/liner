import { chmodSync, mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

describe("liner npm shim", () => {
  it("passes the bundled headless runner path to the Go TUI", () => {
    if (process.platform === "win32") return;

    const dir = mkdtempSync(join(tmpdir(), "liner-shim-"));
    const probe = join(dir, "liner-tui-probe");
    writeFileSync(probe, '#!/bin/sh\nprintf "%s\\n" "$LINER_HEADLESS_RUNNER"\n', "utf8");
    chmodSync(probe, 0o755);
    const bundledRunner = join(packageRoot, "dist", "agents", "headless-runner.js");
    mkdirSync(dirname(bundledRunner), { recursive: true });
    writeFileSync(bundledRunner, "export {};\n", "utf8");

    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js")], {
      cwd: packageRoot,
      encoding: "utf8",
      env: {
        ...process.env,
        LINER_GO_TUI_BIN: probe,
        LINER_HEADLESS_RUNNER: "",
        LINER_TUI: "",
      },
    });

    expect(result.status).toBe(0);
    expect(result.stdout.trim()).toBe(bundledRunner);
  });
});
