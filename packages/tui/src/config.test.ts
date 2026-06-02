import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  existsSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { AgentDescriptor } from "./agents/types.js";

// All exercises below talk to ~/.liner/config.yaml. We need to redirect HOME
// to a tmp directory so the test suite doesn't write into the real user's
// dotfiles. Because the path is resolved at import time inside config.ts via
// `homedir()`, we restore os.homedir and re-import the module fresh per test.

let fakeHome: string;
let originalHome: string | undefined;
let originalUserprofile: string | undefined;
let originalLinerAgent: string | undefined;

beforeEach(() => {
  fakeHome = mkdtempSync(join(tmpdir(), "liner-config-test-"));
  originalHome = process.env["HOME"];
  originalUserprofile = process.env["USERPROFILE"];
  originalLinerAgent = process.env["LINER_AGENT"];
  process.env["HOME"] = fakeHome;
  process.env["USERPROFILE"] = fakeHome; // Windows
  delete process.env["LINER_AGENT"];
  vi.resetModules();
});

afterEach(() => {
  rmSync(fakeHome, { recursive: true, force: true });
  if (originalHome !== undefined) process.env["HOME"] = originalHome;
  else delete process.env["HOME"];
  if (originalUserprofile !== undefined) process.env["USERPROFILE"] = originalUserprofile;
  else delete process.env["USERPROFILE"];
  if (originalLinerAgent !== undefined) process.env["LINER_AGENT"] = originalLinerAgent;
  else delete process.env["LINER_AGENT"];
});

async function loadConfigModule(): Promise<typeof import("./config.js")> {
  return await import("./config.js");
}

function fakeAgent(id: "claude" | "codex"): AgentDescriptor {
  return {
    id,
    name: id === "claude" ? "Claude Code" : "OpenAI Codex",
    bin: `/usr/local/bin/${id}`,
  };
}

describe("config", () => {
  it("returns null defaults when the file doesn't exist", async () => {
    const cfg = await loadConfigModule();
    expect(cfg.configExists()).toBe(false);
    expect(cfg.readConfig()).toEqual({
      agent: null,
      models: null,
      jsSetupPrompted: false,
    });
  });

  it("round-trips a written config", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    expect(cfg.configExists()).toBe(true);
    expect(cfg.readConfig().agent).toBe("claude");
  });

  it("falls back to null on malformed YAML rather than throwing", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    // Stomp the file with garbage; the next read must not crash.
    writeFileSync(cfg.configPath(), "::: not yaml :::", "utf8");
    expect(cfg.readConfig()).toEqual({
      agent: null,
      models: null,
      jsSetupPrompted: false,
    });
  });

  it("rejects unknown agent values during read (defense-in-depth)", async () => {
    const cfg = await loadConfigModule();
    // writeConfig creates the parent dir; we then stomp the file with a
    // value that isn't claude or codex. Read should null it out.
    cfg.writeConfig({ agent: "claude" });
    writeFileSync(cfg.configPath(), "agent: gpt-9000\n", "utf8");
    expect(cfg.readConfig().agent).toBeNull();
  });

  it("preserves a hand-editable comment header", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    const text = readFileSync(cfg.configPath(), "utf8");
    expect(text).toContain("# Liner user config");
    expect(text).toContain("agent: codex");
  });

  it("round-trips per-phase model overrides", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude", models: { claude: { candidates: "opus" } } });
    expect(cfg.readConfig().models).toEqual({ claude: { candidates: "opus" } });
  });

  it("round-trips the JS setup onboarding flag", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ jsSetupPrompted: true });
    expect(cfg.readConfig().jsSetupPrompted).toBe(true);
  });

  it("preserves model overrides when only the agent changes", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ models: { claude: { evaluation: "opus" } } });
    cfg.writeConfig({ agent: "codex" });
    const out = cfg.readConfig();
    expect(out.agent).toBe("codex");
    expect(out.models).toEqual({ claude: { evaluation: "opus" } });
    expect(out.jsSetupPrompted).toBe(false);
  });

  it("drops a malformed models block rather than crashing", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    writeFileSync(cfg.configPath(), "agent: claude\nmodels:\n  claude: not-an-object\n", "utf8");
    const out = cfg.readConfig();
    expect(out.agent).toBe("claude");
    expect(out.models).toBeNull();
  });
});

describe("resolveConfiguredAgent priority order", () => {
  it("returns null when nothing is installed", async () => {
    const cfg = await loadConfigModule();
    expect(cfg.resolveConfiguredAgent([])).toBeNull();
  });

  it("honors LINER_AGENT when the named agent is installed", async () => {
    process.env["LINER_AGENT"] = "codex";
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" }); // config says claude...
    const installed = [fakeAgent("claude"), fakeAgent("codex")];
    // ...but the env pin wins.
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("codex");
  });

  it("ignores LINER_AGENT when the named agent isn't installed", async () => {
    process.env["LINER_AGENT"] = "codex";
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    // Codex isn't installed — env pin is invalid, fall through to config.
    const installed = [fakeAgent("claude")];
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("claude");
  });

  it("falls back to the configured agent when no env pin is set", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    const installed = [fakeAgent("claude"), fakeAgent("codex")];
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("codex");
  });

  it("ignores configured agent when it's no longer installed", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    const installed = [fakeAgent("claude")]; // codex uninstalled
    // Single remaining agent picked automatically.
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("claude");
  });

  it("auto-picks the single installed agent when no preference is set", async () => {
    const cfg = await loadConfigModule();
    const installed = [fakeAgent("claude")];
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("claude");
  });

  it("returns null when multiple agents installed and no preference is set", async () => {
    const cfg = await loadConfigModule();
    const installed = [fakeAgent("claude"), fakeAgent("codex")];
    expect(cfg.resolveConfiguredAgent(installed)).toBeNull();
  });

  it("creates the parent directory if it doesn't exist", async () => {
    const cfg = await loadConfigModule();
    // Sanity: the test's fake home starts empty, no .liner/ inside.
    expect(existsSync(join(fakeHome, ".liner"))).toBe(false);
    cfg.writeConfig({ agent: "claude" });
    expect(existsSync(join(fakeHome, ".liner"))).toBe(true);
    expect(cfg.configExists()).toBe(true);
  });
});
