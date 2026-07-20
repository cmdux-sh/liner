import { existsSync, mkdtempSync, mkdirSync, realpathSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { describe, expect, it } from "vitest";
import { prepareIsolatedLaunch } from "../scripts/dev-isolated.mjs";
import { fingerprintTargets } from "../scripts/isolation-fingerprint.mjs";

const packageDir = join(import.meta.dirname, "..");
const script = join(packageDir, "scripts", "dev-isolated.mjs");

describe("isolated development launch", () => {
  it("uses temporary Liner and provider homes instead of ambient production paths", () => {
    const root = mkdtempSync(join(tmpdir(), "liner-dev-isolated-test-"));
    const productionHome = join(root, "production-home");
    mkdirSync(productionHome, { recursive: true });

    const result = spawnSync(process.execPath, [script, "--dry-run", "--root", join(root, "isolated")], {
      cwd: packageDir,
      encoding: "utf8",
      env: {
        ...process.env,
        HOME: productionHome,
        CODEX_HOME: join(productionHome, ".codex"),
        CLAUDE_CONFIG_DIR: join(productionHome, ".claude"),
      },
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain(join(root, "isolated", "home", ".liner", "config.yaml"));
    expect(result.stdout).toContain(join(root, "isolated", "projects"));
    expect(result.stdout).toContain("temporary empty profile");
    expect(result.stdout).toContain("Cleanup after testing");
    expect(result.stdout).not.toContain(productionHome);
    expect(existsSync(join(productionHome, ".liner"))).toBe(false);
  });

  it("isolates writable caches, removes ambient credentials and binary overrides, and explicitly references profiles", () => {
    const root = mkdtempSync(join(tmpdir(), "liner-dev-isolated-probe-"));
    const isolated = join(root, "isolated");
    const existingCodexHome = join(root, "existing-codex");
    mkdirSync(existingCodexHome, { recursive: true });
    const launch = prepareIsolatedLaunch(
      { root: isolated, codexHome: existingCodexHome, claudeHome: "" },
      {
        ...process.env,
        OPENAI_API_KEY: "production-secret",
        ANTHROPIC_API_KEY: "production-secret",
        CLAUDE_CODE_USE_BEDROCK: "1",
        AWS_PROFILE: "production",
        GOOGLE_APPLICATION_CREDENTIALS: join(root, "production-google-credentials.json"),
        LINER_GO_TUI_BIN: join(root, "wrong-binary"),
        LINER_HEADLESS_RUNNER: join(root, "wrong-runner.js"),
        LINER_BIN: join(root, "wrong-core"),
        LINER_AGENT: "claude",
        LINER_TUI_INITIALIZATION_PROBE: "1",
        XDG_CACHE_HOME: join(root, "production-cache"),
        LOCALAPPDATA: join(root, "production-local-app-data"),
        PLAYWRIGHT_BROWSERS_PATH: join(root, "production-browsers"),
      },
    );

    expect(launch.codexHome).toBe(realpathSync(existingCodexHome));
    expect(launch.env.HOME).toBe(join(launch.root, "home"));
    expect(launch.env.USERPROFILE).toBe(join(launch.root, "home"));
    expect(launch.env.LINER_DIR).toBe(join(launch.root, "projects"));
    expect(launch.env.CLAUDE_CONFIG_DIR).toBe(join(launch.root, "provider-homes", "claude"));
    expect(launch.env.XDG_CACHE_HOME).toBe(join(launch.root, "cache"));
    expect(launch.env.LOCALAPPDATA).toBe(join(launch.root, "local-app-data"));
    expect(launch.env.PLAYWRIGHT_BROWSERS_PATH).toBe(join(launch.root, "cache", "ms-playwright"));
    expect(launch.env.OPENAI_API_KEY).toBeUndefined();
    expect(launch.env.ANTHROPIC_API_KEY).toBeUndefined();
    expect(launch.env.CLAUDE_CODE_USE_BEDROCK).toBeUndefined();
    expect(launch.env.AWS_PROFILE).toBeUndefined();
    expect(launch.env.GOOGLE_APPLICATION_CREDENTIALS).toBeUndefined();
    expect(launch.env.LINER_GO_TUI_BIN).toBeUndefined();
    expect(launch.env.LINER_BIN).toBeUndefined();
    expect(launch.env.LINER_AGENT).toBeUndefined();
    expect(launch.env.LINER_TUI_INITIALIZATION_PROBE).toBeUndefined();
    expect(launch.env.LINER_HEADLESS_RUNNER).toBe(join(packageDir, "dist", "agents", "headless-runner.js"));
  });

  it("refuses reused roots and roots whose requested path is a symlink", () => {
    const parent = mkdtempSync(join(tmpdir(), "liner-dev-isolated-unsafe-"));
    const existing = join(parent, "existing");
    mkdirSync(existing);
    expect(() => prepareIsolatedLaunch({ root: existing, codexHome: "", claudeHome: "" })).toThrow("must not already exist");

    const outside = join(parent, "outside");
    mkdirSync(outside);
    const linked = join(parent, "linked");
    symlinkSync(outside, linked, "dir");
    expect(() => prepareIsolatedLaunch({ root: linked, codexHome: "", claudeHome: "" })).toThrow("must not already exist");
  });

  it("detects changes behind a symlinked production path", () => {
    const parent = mkdtempSync(join(tmpdir(), "liner-isolation-fingerprint-"));
    const productionTarget = join(parent, "production-target");
    mkdirSync(productionTarget);
    const productionLink = join(parent, "production-link");
    symlinkSync(productionTarget, productionLink, "dir");
    const config = join(productionTarget, "config.yaml");
    writeFileSync(config, "provider: openai\n");

    const before = fingerprintTargets([productionLink]);
    writeFileSync(config, "provider: claude\nmodel: opus\n");
    const after = fingerprintTargets([productionLink]);

    expect(after).not.toBe(before);
  });
});
