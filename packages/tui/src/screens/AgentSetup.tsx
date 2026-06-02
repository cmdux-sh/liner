import React, { useMemo, useState } from "react";
import { Box, Text, useInput } from "ink";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { useApp } from "../store.js";
import { detectAgents, KNOWN_AGENTS } from "../agents/detect.js";
import { readConfig, writeConfig } from "../config.js";
import { MUTED, CURRENT, ERROR, HEADING, STRUCTURE, SUCCESS } from "../colors.js";
import type { AgentDescriptor } from "../agents/types.js";

// One-time picker for "which AI agent should drive the methodology phases?"
// — runs as the first screen on a fresh install when both Claude Code and
// Codex are present, and is reachable later via the splash Settings entry.
//
// Persists the choice to ~/.liner/config.yaml (via the config module). The
// LINER_AGENT env var still overrides — power users / CI keep the escape
// hatch. The mid-phase picker becomes dead code in practice but stays bound
// as a defensive fallback (e.g. config file disappears, env var names a
// non-installed agent).

/**
 * Source of truth for what the screen says about each agent. Kept inline so
 * the copy lives with the screen that shows it, rather than in agents/types
 * (which is where the cold type lives).
 */
const AGENT_BLURBS: Record<string, { tagline: string; install: string }> = {
  claude: {
    tagline: "Anthropic's Claude Code — strong at long-context tasks and at writing prose.",
    install: "npm install -g @anthropic-ai/claude-code",
  },
  codex: {
    tagline: "OpenAI Codex — fast for code-heavy phases, snappy on shorter prompts.",
    install: "see https://github.com/openai/codex",
  },
};

export function AgentSetup(): React.ReactElement {
  const app = useApp();
  const installed = useMemo(() => detectAgents(), []);
  const config = useMemo(() => readConfig(), []);
  // Agents the TUI knows about but didn't find on this machine. Surfaced as
  // a footer so a user with (e.g.) Codex installed via fnm but unreachable
  // from the TUI's PATH has a concrete unblock path rather than just "not
  // detected." The KNOWN_AGENTS catalog is the single source of truth.
  const undetected = useMemo(
    () =>
      KNOWN_AGENTS.filter((meta) => !installed.some((a) => a.id === meta.id)),
    [installed],
  );

  // Where the cursor lands by default: the currently-configured agent if
  // it's installed, otherwise the first one. New installs start at index 0.
  const initialIndex = useMemo(() => {
    if (config.agent) {
      const idx = installed.findIndex((a) => a.id === config.agent);
      if (idx >= 0) return idx;
    }
    return 0;
  }, [installed, config.agent]);
  const [cursor, setCursor] = useState(initialIndex);
  const [error, setError] = useState<string | null>(null);

  /**
   * Return to wherever launched this screen. If the JS-rendering onboarding
   * has not been shown yet, continue the first-run flow there before home.
   */
  function done(): void {
    if (!config.jsSetupPrompted) {
      app.navigate({ kind: "jsSetup" });
      return;
    }
    app.back();
  }

  function pick(agent: AgentDescriptor): void {
    try {
      writeConfig({ agent: agent.id });
      app.setNotification({
        kind: "info",
        message: `Agent set to ${agent.name}. Change anytime from the splash menu.`,
      });
      done();
    } catch (e) {
      setError(`Could not write config: ${(e as Error).message}`);
    }
  }

  function clearChoice(): void {
    try {
      writeConfig({ agent: null });
      app.setNotification({
        kind: "info",
        message: "Cleared agent preference. The mid-phase picker will appear when needed.",
      });
      done();
    } catch (e) {
      setError(`Could not write config: ${(e as Error).message}`);
    }
  }

  useInput((input, key) => {
    if (installed.length === 0) {
      if (key.escape || input === "b") done();
      return;
    }
    if (key.upArrow || input === "k") {
      setCursor((c) => Math.max(0, c - 1));
    } else if (key.downArrow || input === "j") {
      setCursor((c) => Math.min(installed.length - 1, c + 1));
    } else if (key.return) {
      // Enter accepts; space is intentionally NOT an accept key — it's
      // reserved for toggle semantics elsewhere, so we keep one meaning per key.
      const agent = installed[cursor];
      if (agent) pick(agent);
    } else if (input === "u" && config.agent) {
      clearChoice();
    } else if (key.escape || input === "b") {
      done();
    }
  });

  // Zero-agent state: instructions, no picker. User can't proceed without
  // an agent for methodology phases — but they can still browse compiled
  // mixtapes, so we let them through to the browser.
  if (installed.length === 0) {
    return (
      <Box flexDirection="column">
        <Header title="choose your AI agent" subtitle="first-run setup" />
        <Box marginBottom={1}>
          <LabeledBox label="no agent installed" color={ERROR}>
            <Text color={ERROR} bold>
              Neither Claude Code nor Codex was found on your PATH.
            </Text>
            <Box marginTop={1} flexDirection="column">
              <Text>Liner can run the methodology with either:</Text>
              <Box marginTop={1} flexDirection="column">
                <Text>
                  {"  "}
                  <Text color={STRUCTURE} bold>Claude Code</Text>
                  <Text color={MUTED}>{" — "}{AGENT_BLURBS.claude!.install}</Text>
                </Text>
                <Text>
                  {"  "}
                  <Text color={STRUCTURE} bold>OpenAI Codex</Text>
                  <Text color={MUTED}>{" — "}{AGENT_BLURBS.codex!.install}</Text>
                </Text>
              </Box>
            </Box>
            <Box marginTop={1}>
              <Text color={MUTED}>
                You can still browse compiled mixtapes without an agent — just
                press <Text color={STRUCTURE}>esc</Text> to continue.
              </Text>
            </Box>
          </LabeledBox>
        </Box>

        <KeyHints hints={[{ key: "esc", label: "skip to browser" }]} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Header
        title="choose your AI agent"
        subtitle={
          config.agent
            ? `current: ${config.agent} · change anytime`
            : "first-run setup · pick once, change anytime"
        }
      />

      <Box marginBottom={1} flexDirection="column">
        <Text color={HEADING} bold>Which agent should drive the methodology?</Text>
        <Box marginTop={1}>
          <Text color={MUTED}>
            Liner spawns the agent CLI for each methodology phase (framing,
            candidate discovery, evaluation, …). Both options run the same
            skill bundle and produce the same artifacts — pick the one you
            already use, or default to whichever is faster on your hardware.
            You can change this later from the splash menu's Settings entry.
          </Text>
        </Box>
      </Box>

      <Box marginBottom={1}>
        <LabeledBox label="installed agents" color={STRUCTURE}>
          {installed.map((agent, i) => {
            const active = i === cursor;
            const blurb = AGENT_BLURBS[agent.id];
            const isConfigured = config.agent === agent.id;
            return (
              <Box key={agent.id} flexDirection="column" marginBottom={i === installed.length - 1 ? 0 : 1}>
                <Box>
                  <Text color={active ? CURRENT : undefined} bold={active}>
                    {active ? "> " : "  "}
                  </Text>
                  <Text color={STRUCTURE}>{`[${i + 1}]`}</Text>
                  <Text>{"  "}</Text>
                  <Text color={active ? CURRENT : undefined} bold={active}>
                    {agent.name}
                  </Text>
                  {isConfigured ? (
                    <Text color={SUCCESS}>{"  ·  current"}</Text>
                  ) : null}
                </Box>
                {blurb ? (
                  <Box paddingLeft={6}>
                    <Text color={MUTED}>{blurb.tagline}</Text>
                  </Box>
                ) : null}
                <Box paddingLeft={6}>
                  <Text color={MUTED}>{agent.bin}</Text>
                </Box>
              </Box>
            );
          })}
        </LabeledBox>
      </Box>

      <Box marginBottom={1}>
        <Text color={STRUCTURE}>enter</Text>
        <Text color={MUTED}>{` → pick ${installed[cursor]?.name ?? "agent"} and continue`}</Text>
      </Box>

      {undetected.length > 0 ? (
        <Box marginBottom={1}>
          <LabeledBox label="not detected" color={STRUCTURE}>
            <Text color={MUTED}>
              {undetected.length === 1
                ? `${undetected[0]!.name} isn't on Liner's PATH right now.`
                : `${undetected.map((u) => u.name).join(" and ")} aren't on Liner's PATH right now.`}
              {" "}If you have one installed but it's not showing up — usually
              fnm / nvm / asdf shims that aren't loaded in the launcher's
              environment — try one of:
            </Text>
            <Box marginTop={1} flexDirection="column">
              <Text color={MUTED}>
                {"  • "}Restart the TUI from a shell where{" "}
                <Text color={STRUCTURE}>which {undetected[0]!.id}</Text>{" "}
                returns a path.
              </Text>
              <Text color={MUTED}>
                {"  • "}Set{" "}
                <Text color={STRUCTURE}>{undetected[0]!.envVar}=/full/path/to/{undetected[0]!.id}</Text>{" "}
                in your shell config (or pass it inline:{" "}
                <Text color={STRUCTURE}>{undetected[0]!.envVar}=… npm run dev</Text>).
              </Text>
              <Text color={MUTED}>
                {"  • "}If you don't have it installed yet:{" "}
                <Text color={STRUCTURE}>{AGENT_BLURBS[undetected[0]!.id]?.install ?? "(install instructions vary)"}</Text>
              </Text>
            </Box>
          </LabeledBox>
        </Box>
      ) : null}

      {error ? (
        <Box marginBottom={1}>
          <Text color={ERROR}>{error}</Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "↑↓", label: "select" },
          { key: "enter", label: "pick + save" },
          ...(config.agent ? [{ key: "u", label: "clear preference" }] : []),
          { key: "esc", label: "skip" },
        ]}
      />
    </Box>
  );
}
