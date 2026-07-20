import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const canonicalVersion = readFileSync(resolve(packageRoot, "..", "..", "VERSION"), "utf8").trim();

describe("liner npm shim", () => {
  it("passes the bundled headless runner path to the Go TUI", () => {
    if (process.platform === "win32") return;

    const dir = mkdtempSync(join(tmpdir(), "liner-shim-"));
    const probe = join(dir, "liner-tui-probe");
    const headlessRunner = join(packageRoot, "dist", "agents", "headless-runner.js");
    const createdHeadlessRunner = !existsSync(headlessRunner);
    writeFileSync(probe, '#!/bin/sh\nprintf "%s\\n" "$LINER_HEADLESS_RUNNER"\n', "utf8");
    chmodSync(probe, 0o755);
    if (createdHeadlessRunner) {
      mkdirSync(dirname(headlessRunner), { recursive: true });
      writeFileSync(headlessRunner, "// Test fixture for bundled runner resolution.\n", "utf8");
    }

    try {
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
      expect(result.stdout.trim()).toBe(headlessRunner);
    } finally {
      if (createdHeadlessRunner) rmSync(headlessRunner, { force: true });
    }
  });

  it.each(["--help", "-h"])("owns top-level %s help at the installed boundary", (flag) => {
    if (process.platform === "win32") return;

    const dir = mkdtempSync(join(tmpdir(), "liner-help-"));
    const probe = join(dir, "liner-core-probe");
    writeFileSync(probe, '#!/bin/sh\nprintf "CORE_PROBE %s\\n" "$*"\nexit 23\n', "utf8");
    chmodSync(probe, 0o755);

    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), flag], {
      cwd: packageRoot,
      encoding: "utf8",
      env: { ...process.env, LINER_BIN: probe },
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("");
    expect(result.stdout).toContain("liner with no arguments opens the Go TUI");
    expect(result.stdout).toContain("Common workflow commands:");
    expect(result.stdout).toContain("Advanced maintenance commands:");
    for (const row of [
      "  init       Create a Liner Project",
      "  compile    Build MIXTAPE.md from a Liner Project",
      "  share      Create a shareable .mixtape file",
      "  import     Import a .mixtape file as a Liner Project",
      "  list       List Liner Projects",
      "  status     Show Liner Project status",
      "  setup-js   Set up JavaScript rendering support",
      "  uninstall  Remove local Liner state and caches",
      "  --version  Show installed launcher and core versions (-v)",
      "  replay     Clone project inputs with lineage for later comparison",
      "  clone      Fetch or copy a project recipe",
      "  skills     Find installed skills that can be used as Sources",
      "  sources    Plan and apply Source maintenance",
      "  adapters   Manage optional Maintenance Adapters",
      "  project    Inspect, plan, and apply Project maintenance",
      "  cache      Inspect or clear the Source cache",
      "  manifest   Build or inspect a Source manifest",
    ]) {
      expect(result.stdout).toContain(row);
    }
    expect(result.stdout.indexOf("Common workflow commands:")).toBeLessThan(
      result.stdout.indexOf("Advanced maintenance commands:"),
    );
    expect(result.stdout).not.toContain("CORE_PROBE");
  });

  it.each(["compile", "project", "sources", "adapters"])(
    "forwards %s command help to the owning core implementation",
    (command) => {
      if (process.platform === "win32") return;

      const dir = mkdtempSync(join(tmpdir(), "liner-command-help-"));
      const probe = join(dir, "liner-core-probe");
      writeFileSync(
        probe,
        '#!/bin/sh\nprintf "stdout:%s\\n" "$*"\nprintf "stderr:%s\\n" "$*" >&2\nexit 23\n',
        "utf8",
      );
      chmodSync(probe, 0o755);

      const result = spawnSync(
        process.execPath,
        [join(packageRoot, "bin", "liner.js"), command, "--help"],
        {
          cwd: packageRoot,
          encoding: "utf8",
          env: { ...process.env, LINER_BIN: probe },
        },
      );

      expect(result.status).toBe(23);
      expect(result.stdout).toBe(`stdout:${command} --help\n`);
      expect(result.stderr).toBe(`stderr:${command} --help\n`);
    },
  );

  it("reports the canonical launcher and core release together", () => {
    if (process.platform === "win32") return;

    const dir = mkdtempSync(join(tmpdir(), "liner-version-"));
    const probe = join(dir, "liner-core-probe");
    writeFileSync(probe, `#!/bin/sh\nprintf "liner ${canonicalVersion}\\n"\n`, "utf8");
    chmodSync(probe, 0o755);

    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), "--version"], {
      cwd: packageRoot,
      encoding: "utf8",
      env: { ...process.env, LINER_BIN: probe },
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("");
    expect(result.stdout.trim()).toBe(
      `liner ${canonicalVersion} (tui)  ·  ${canonicalVersion} (core)`,
    );
  });
});
