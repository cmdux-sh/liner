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
let originalCodexBin: string | undefined;
let originalCodexHome: string | undefined;
let originalNativeCodexHome: string | undefined;

beforeEach(() => {
  fakeHome = mkdtempSync(join(tmpdir(), "liner-config-test-"));
  originalHome = process.env["HOME"];
  originalUserprofile = process.env["USERPROFILE"];
  originalLinerAgent = process.env["LINER_AGENT"];
  originalCodexBin = process.env["LINER_CODEX_BIN"];
  originalCodexHome = process.env["LINER_CODEX_HOME"];
  originalNativeCodexHome = process.env["CODEX_HOME"];
  process.env["HOME"] = fakeHome;
  process.env["USERPROFILE"] = fakeHome; // Windows
  delete process.env["LINER_AGENT"];
  delete process.env["LINER_CODEX_BIN"];
  delete process.env["LINER_CODEX_HOME"];
  delete process.env["CODEX_HOME"];
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
  if (originalCodexBin !== undefined) process.env["LINER_CODEX_BIN"] = originalCodexBin;
  else delete process.env["LINER_CODEX_BIN"];
  if (originalCodexHome !== undefined) process.env["LINER_CODEX_HOME"] = originalCodexHome;
  else delete process.env["LINER_CODEX_HOME"];
  if (originalNativeCodexHome !== undefined) process.env["CODEX_HOME"] = originalNativeCodexHome;
  else delete process.env["CODEX_HOME"];
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
      runner: null,
      providerPreferences: null,
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
      runner: null,
      providerPreferences: null,
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

  it("round-trips independent provider model preferences", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      providerPreferences: {
        codex: { model: "gpt-5.6-sol" },
        claude: { model: "sonnet" },
      },
    });
    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: "gpt-5.6-sol" },
      claude: { model: "sonnet" },
    });
    expect(readFileSync(cfg.configPath(), "utf8")).toContain("provider_preferences:");
  });

  it("round-trips OpenAI reasoning effort without clearing its model", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      providerPreferences: {
        codex: { model: "gpt-5.6-sol", reasoningEffort: "max" },
      },
    });
    cfg.writeConfig({
      providerPreferences: {
        codex: { reasoningEffort: "high" },
      },
    });

    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: "gpt-5.6-sol", reasoningEffort: "high" },
    });
    const saved = readFileSync(cfg.configPath(), "utf8");
    expect(saved).toContain("reasoning_effort: high");
  });

  it("persists Auto separately from provider-default and fixed OpenAI models", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      providerPreferences: {
        codex: { model: "gpt-5.6-sol", reasoningEffort: "max" },
      },
    });

    cfg.writeConfig({
      providerPreferences: {
        codex: { model: null, modelMode: "auto", reasoningEffort: null },
      },
    });
    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: null, modelMode: "auto" },
    });
    let saved = readFileSync(cfg.configPath(), "utf8");
    expect(saved).toContain("model_mode: auto");
    expect(saved).not.toContain("model: gpt-5.6-sol");

    cfg.writeConfig({
      providerPreferences: {
        codex: { model: null, modelMode: "default" },
      },
    });
    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: null, modelMode: "default" },
    });

    cfg.writeConfig({
      providerPreferences: {
        codex: { model: "gpt-5.6-terra", modelMode: null },
      },
    });
    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: "gpt-5.6-terra" },
    });
    saved = readFileSync(cfg.configPath(), "utf8");
    expect(saved).not.toMatch(/^\s+model_mode:/m);
  });

  it("keeps a valid OpenAI model when reasoning effort is malformed", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    writeFileSync(
      cfg.configPath(),
      "agent: codex\nprovider_preferences:\n  codex:\n    model: gpt-5.6-terra\n    reasoning_effort: turbo\n",
      "utf8",
    );

    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: "gpt-5.6-terra" },
    });
  });

  it("reads an effort-only OpenAI preference", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    writeFileSync(
      cfg.configPath(),
      "agent: codex\nprovider_preferences:\n  codex:\n    reasoning_effort: xhigh\n",
      "utf8",
    );

    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: null, reasoningEffort: "xhigh" },
    });
  });

  it("updates one provider preference without deleting siblings or unknown fields", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    writeFileSync(
      cfg.configPath(),
      [
        "agent: codex",
        "provider_preferences:",
        "  codex:",
        "    model: old-openai-model",
        "    effort: high",
        "  claude:",
        "    model: opus",
        "  future_provider:",
        "    future_field: keep-me",
        "",
      ].join("\n"),
      "utf8",
    );

    cfg.writeConfig({ providerPreferences: { codex: { model: "gpt-5.6-terra" } } });

    expect(cfg.readConfig().providerPreferences).toEqual({
      codex: { model: "gpt-5.6-terra" },
      claude: { model: "opus" },
    });
    const saved = readFileSync(cfg.configPath(), "utf8");
    for (const expected of ["model: gpt-5.6-terra", "effort: high", "model: opus", "future_field: keep-me"]) {
      expect(saved).toContain(expected);
    }
  });

  it("drops only malformed provider model entries", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    writeFileSync(
      cfg.configPath(),
      "agent: codex\nrunner:\n  agent: codex\n  executable: /saved/codex\n  config_home: /saved/home\nprovider_preferences:\n  codex:\n    model: '   '\n  claude:\n    model: opus\n",
      "utf8",
    );
    const out = cfg.readConfig();
    expect(out.runner?.executable).toBe("/saved/codex");
    expect(out.providerPreferences).toEqual({ claude: { model: "opus" } });
  });

  it("round-trips the JS setup onboarding flag", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ jsSetupPrompted: true });
    expect(cfg.readConfig().jsSetupPrompted).toBe(true);
  });

  it("round-trips the durable AI runner profile", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      runner: {
        agent: "codex",
        executable: "/opt/liner/bin/codex",
        configHome: "/opt/liner/codex-home",
      },
    });

    expect(cfg.readConfig().runner).toEqual({
      agent: "codex",
      executable: "/opt/liner/bin/codex",
      configHome: "/opt/liner/codex-home",
    });
    expect(readFileSync(cfg.configPath(), "utf8")).not.toContain("auth");
  });

  it("drops a malformed runner profile and preserves unrelated settings on write", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    writeFileSync(
      cfg.configPath(),
      "agent: claude\nrunner:\n  agent: codex\n  executable: 42\ncustom_field: keep-me\n",
      "utf8",
    );

    expect(cfg.readConfig().runner).toBeNull();
    cfg.writeConfig({ jsSetupPrompted: true });
    const text = readFileSync(cfg.configPath(), "utf8");
    expect(text).toContain("custom_field: keep-me");
    expect(text).toContain("jsSetupPrompted: true");
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

  it("fails closed when LINER_AGENT names an unavailable runner", async () => {
    process.env["LINER_AGENT"] = "codex";
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "claude" });
    const installed = [fakeAgent("claude")];
    expect(cfg.resolveConfiguredAgent(installed)).toBeNull();
  });

  it("falls back to the configured agent when no env pin is set", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({ agent: "codex" });
    const installed = [fakeAgent("claude"), fakeAgent("codex")];
    expect(cfg.resolveConfiguredAgent(installed)?.id).toBe("codex");
  });

  it("uses the persisted executable instead of the same provider found on PATH", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      runner: {
        agent: "codex",
        executable: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
    });

    const resolved = cfg.resolveConfiguredAgent([fakeAgent("codex")]);
    expect(resolved).toMatchObject({
      id: "codex",
      bin: "/saved/bin/codex",
      configHome: "/saved/codex-home",
    });
  });

  it("lets explicit path overrides win without rewriting the saved profile", async () => {
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      runner: {
        agent: "codex",
        executable: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
    });
    const before = readFileSync(cfg.configPath(), "utf8");
    process.env["LINER_CODEX_BIN"] = "/override/bin/codex";
    process.env["LINER_CODEX_HOME"] = "/override/codex-home";

    expect(cfg.resolveConfiguredAgent([fakeAgent("codex")])).toMatchObject({
      bin: "/override/bin/codex",
      configHome: "/override/codex-home",
    });
    expect(readFileSync(cfg.configPath(), "utf8")).toBe(before);
  });

  it("layers a matching LINER_AGENT over the saved paths before PATH detection", async () => {
    process.env["LINER_AGENT"] = "codex";
    const cfg = await loadConfigModule();
    const config = {
      agent: null,
      runner: {
        agent: "codex",
        executable: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
      providerPreferences: null,
      models: null,
      jsSetupPrompted: false,
    } satisfies import("./config.js").UserConfig;

    expect(cfg.resolveConfiguredAgent([], config)).toMatchObject({
      id: "codex",
      bin: "/saved/bin/codex",
      configHome: "/saved/codex-home",
    });
  });

  it("lets the native config-home override win over the saved home", async () => {
    process.env["CODEX_HOME"] = "/native/codex-home";
    const cfg = await loadConfigModule();
    cfg.writeConfig({
      runner: {
        agent: "codex",
        executable: "/saved/bin/codex",
        configHome: "/saved/codex-home",
      },
    });

    expect(cfg.resolveConfiguredAgent([])).toMatchObject({
      bin: "/saved/bin/codex",
      configHome: "/native/codex-home",
    });
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
