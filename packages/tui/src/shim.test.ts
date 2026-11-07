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
      `liner ${canonicalVersion} (tui) · ${canonicalVersion} (core)`,
    );
  });

  it("names the closest command for a typo instead of opening the navigator", () => {
    const dir = mkdtempSync(join(tmpdir(), "liner-unknown-"));
    const probe = join(dir, "liner-tui-probe");
    writeFileSync(probe, "#!/bin/sh\necho NAVIGATOR-OPENED\n", "utf8");
    chmodSync(probe, 0o755);

    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), "compil"], {
      cwd: packageRoot,
      encoding: "utf8",
      env: { ...process.env, LINER_GO_TUI_BIN: probe, LINER_TUI: "" },
    });

    expect(result.status).toBe(2);
    expect(result.stdout).not.toContain("NAVIGATOR-OPENED");
    expect(result.stderr).toContain("Unknown command: compil");
    expect(result.stderr).toContain("Did you mean `liner compile`?");
  });

  it("offers no guess for an unrelated word", () => {
    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), "banana"], {
      cwd: packageRoot,
      encoding: "utf8",
    });

    expect(result.status).toBe(2);
    expect(result.stderr).toContain("Unknown command: banana");
    expect(result.stderr).not.toContain("Did you mean");
  });

  it("shows help for help flags anywhere and for `liner help`", () => {
    for (const args of [["--help", "compile"], ["-h", "status"], ["help"]]) {
      const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), ...args], {
        cwd: packageRoot,
        encoding: "utf8",
      });
      expect(result.status).toBe(0);
      expect(result.stdout).toContain("Open the navigator");
    }
  });

  it("suggests commands whatever their case, and help flags", () => {
    for (const [typed, wanted] of [["COMPILE", "compile"], ["--hlep", "--help"]]) {
      const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), typed], {
        cwd: packageRoot,
        encoding: "utf8",
      });
      expect(result.status).toBe(2);
      expect(result.stderr).toContain(`Did you mean \`liner ${wanted}\`?`);
    }
  });

  it("describes the navigator in --help", () => {
    const result = spawnSync(process.execPath, [join(packageRoot, "bin", "liner.js"), "--help"], {
      cwd: packageRoot,
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("Open the navigator");
    expect(result.stdout).not.toContain("Go TUI");
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
