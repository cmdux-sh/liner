import {
  chmodSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
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

  describe("without a bundled platform package", () => {
    // npm installs the launcher outside any repo and puts it on PATH through a
    // symlink. These copies have no platform package and no repo .venv. A
    // copy given a marker file records every time it runs.
    function writeLauncher(dir: string, marker?: string): string {
      mkdirSync(dir, { recursive: true });
      const target = join(dir, "liner.js");
      let source = readFileSync(join(packageRoot, "bin", "liner.js"), "utf8");
      if (marker) {
        source = source.replace(
          /^(#!.*\n)/,
          `$1import { appendFileSync as recordRun } from "node:fs";\nrecordRun(${JSON.stringify(marker)}, "ran\\n");\n`,
        );
      }
      writeFileSync(target, source, "utf8");
      chmodSync(target, 0o755);
      return target;
    }

    function installedLauncher(name: string, marker?: string): { root: string; bin: string } {
      const root = mkdtempSync(join(tmpdir(), `liner-${name}-`));
      writeLauncher(join(root, "pkg", "bin"), marker);
      copyFileSync(join(packageRoot, "package.json"), join(root, "pkg", "package.json"));
      mkdirSync(join(root, "bin"), { recursive: true });
      symlinkSync(join("..", "pkg", "bin", "liner.js"), join(root, "bin", "liner"));
      return { root, bin: join(root, "bin") };
    }

    function separateCli(root: string): string {
      const core = join(root, "core");
      mkdirSync(core, { recursive: true });
      writeFileSync(join(core, "liner"), '#!/bin/sh\nprintf "CORE_PROBE %s\\n" "$*"\nexit 23\n', "utf8");
      chmodSync(join(core, "liner"), 0o755);
      return core;
    }

    function runLauncher(
      launcherBin: string,
      pathDirs: string[],
      args: string[],
      extraEnv: Record<string, string> = {},
    ) {
      const env: Record<string, string> = {
        HOME: join(launcherBin, "..", "home"),
        PATH: [...pathDirs, "/usr/bin", "/bin", dirname(process.execPath)].join(":"),
        ...extraEnv,
      };
      return spawnSync(join(launcherBin, "liner"), args, { encoding: "utf8", env, timeout: 15_000 });
    }

    it("stops with a clear error instead of starting itself again", () => {
      if (process.platform === "win32") return;
      const launcher = installedLauncher("no-cli");
      try {
        const result = runLauncher(launcher.bin, [launcher.bin], ["status"]);
        expect(result.error).toBeUndefined();
        expect(result.status).toBe(1);
        expect(result.stderr).toContain("Could not find the Liner CLI binary.");
      } finally {
        rmSync(launcher.root, { recursive: true, force: true });
      }
    });

    it("uses a separate Liner CLI later on PATH", () => {
      if (process.platform === "win32") return;
      const launcher = installedLauncher("path-cli");
      const core = separateCli(launcher.root);
      try {
        const result = runLauncher(launcher.bin, [launcher.bin, core], ["status"]);
        expect(result.error).toBeUndefined();
        expect(result.status).toBe(23);
        expect(result.stdout).toBe("CORE_PROBE status\n");
      } finally {
        rmSync(launcher.root, { recursive: true, force: true });
      }
    });

    it("never runs another installed launcher", () => {
      if (process.platform === "win32") return;
      const first = installedLauncher("first");
      const marker = join(first.root, "second-launcher-ran");
      const second = installedLauncher("second", marker);
      try {
        const result = runLauncher(first.bin, [first.bin, second.bin], ["status"]);
        expect(result.error).toBeUndefined();
        expect(result.status).toBe(1);
        expect(existsSync(marker)).toBe(false);
      } finally {
        rmSync(first.root, { recursive: true, force: true });
        rmSync(second.root, { recursive: true, force: true });
      }
    });

    it("skips npm wrapper scripts that run a Liner launcher, in any layout", () => {
      if (process.platform === "win32") return;
      // A local install's node_modules/.bin wrapper (npm writes these instead of
      // symlinks on Windows, as liner.cmd/liner.ps1, and where symlinks fail).
      const launcher = installedLauncher("wrapper");
      const modules = join(launcher.root, "project", "node_modules");
      const marker = join(launcher.root, "wrapped-launcher-ran");
      writeLauncher(join(modules, "linersh", "bin"), marker);
      mkdirSync(join(modules, ".bin"), { recursive: true });
      writeFileSync(
        join(modules, ".bin", "liner"),
        '#!/bin/sh\nbasedir=$(dirname "$0")\nexec node "$basedir/../linersh/bin/liner.js" "$@"\n',
        "utf8",
      );
      chmodSync(join(modules, ".bin", "liner"), 0o755);
      const core = separateCli(launcher.root);
      try {
        const result = runLauncher(launcher.bin, [join(modules, ".bin"), core], ["status"]);
        expect(result.error).toBeUndefined();
        expect(result.status).toBe(23);
        expect(result.stdout).toBe("CORE_PROBE status\n");
        expect(existsSync(marker)).toBe(false);
      } finally {
        rmSync(launcher.root, { recursive: true, force: true });
      }
    });

    it("turns the PATH fallback off once launchers are nested", () => {
      if (process.platform === "win32") return;
      const launcher = installedLauncher("depth");
      const core = separateCli(launcher.root);
      try {
        const result = runLauncher(launcher.bin, [launcher.bin, core], ["status"], {
          LINER_LAUNCHER_DEPTH: "2",
        });
        expect(result.error).toBeUndefined();
        expect(result.status).toBe(1);
        expect(result.stdout).not.toContain("CORE_PROBE");
        expect(result.stderr).toContain("Could not find the Liner CLI binary.");
      } finally {
        rmSync(launcher.root, { recursive: true, force: true });
      }
    });
  });
});
