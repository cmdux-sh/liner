import React, { useEffect, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput } from "ink";
import Spinner from "ink-spinner";
import { spawn } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { LabeledBox } from "../components/LabeledBox.js";
import { KeyHints } from "../components/KeyHints.js";
import { Header } from "../components/Header.js";
import { resolveBinary, BinaryNotFoundError } from "../bin-resolver.js";
import { useApp } from "../store.js";
import { MUTED, STRUCTURE, SUCCESS, WARNING, ERROR } from "../colors.js";
import type { Tape } from "../types.js";

// Surface for the .liner-runs/ rollup. Shells out to `liner manifest` so the
// TUI never duplicates the Python parser, then reads the produced JSON and
// renders a per-run table + totals + domain frequency.
//
// Refresh on every mount — manifests are cheap to regenerate (single JSONL
// walk) and a stale view is worse than a 200ms loading spinner.

type RunRow = {
  task_label: string;
  agent: string;
  model: string | null;
  started_at: string;
  duration_s: number | null;
  num_turns: number;
  exit_code: number | null;
  tokens: { input: number; output: number; cache_read: number; cache_create: number };
  cost_usd: number | null;
  tools: Record<string, number>;
  fetches: string[];
};

type Manifest = {
  generated_at: string;
  mixtape: { title?: string; jtbd?: string; path?: string };
  totals: {
    runs: number;
    tool_calls: number;
    fetches: number;
    cost_usd: number | null;
    tokens: { input: number; output: number; cache_read: number; cache_create: number };
  };
  agents_used: string[];
  models_used: string[];
  domains: { domain: string; count: number }[];
  runs: RunRow[];
};

type LoadState =
  | { kind: "loading" }
  | { kind: "ready"; manifest: Manifest }
  | { kind: "empty" }
  | { kind: "error"; message: string };

export function ProcessManifest({
  folder,
  tape,
}: {
  folder: string;
  tape: Tape;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const [state, setState] = useState<LoadState>({ kind: "loading" });

  function refresh(): void {
    setState({ kind: "loading" });
    let bin;
    try {
      bin = resolveBinary();
    } catch (e) {
      if (e instanceof BinaryNotFoundError) {
        setState({ kind: "error", message: e.message });
        return;
      }
      setState({ kind: "error", message: (e as Error).message });
      return;
    }

    const proc = spawn(bin.command, [...bin.args, "manifest", folder], { stdio: "pipe" });
    let stderr = "";
    proc.stderr?.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    proc.on("error", (e) => {
      setState({ kind: "error", message: e.message });
    });
    proc.on("close", (code) => {
      if (code !== 0) {
        setState({
          kind: "error",
          message: `liner manifest exited ${code}\n${stderr}`.trim(),
        });
        return;
      }
      const path = join(folder, "process.json");
      if (!existsSync(path)) {
        setState({ kind: "empty" });
        return;
      }
      try {
        const m = JSON.parse(readFileSync(path, "utf8")) as Manifest;
        if ((m.totals?.runs ?? 0) === 0) {
          setState({ kind: "empty" });
          return;
        }
        setState({ kind: "ready", manifest: m });
      } catch (e) {
        setState({ kind: "error", message: `Could not parse process.json: ${(e as Error).message}` });
      }
    });
  }

  useEffect(() => {
    refresh();
  }, [folder]);

  useInput((input, key) => {
    if (key.escape) {
      app.navigate({ kind: "hub", tape, folder });
    } else if (input === "q") {
      ink.exit();
    } else if (input === "r") {
      refresh();
    }
  });

  return (
    <Box flexDirection="column">
      <Header title="Process manifest" subtitle={folder} />
      <Box marginTop={1} flexDirection="column">
        {state.kind === "loading" ? (
          <Box>
            <Text color={STRUCTURE}>
              <Spinner type="dots" />
            </Text>
            <Text> Rolling up .liner-runs/ …</Text>
          </Box>
        ) : state.kind === "empty" ? (
          <LabeledBox label="No agent runs yet" color={WARNING}>
            <Text color={MUTED}>
              Run a phase from the project hub (or any agent task) to populate
              this view. The manifest aggregates everything written to
              .liner-runs/ — tokens, tool calls, and fetched URLs.
            </Text>
          </LabeledBox>
        ) : state.kind === "error" ? (
          <LabeledBox label="Manifest failed" color={ERROR}>
            <Text color={ERROR}>{state.message}</Text>
          </LabeledBox>
        ) : (
          <ManifestView manifest={state.manifest} />
        )}
      </Box>

      <Box marginTop={1}>
        <KeyHints
          hints={[
            { key: "r", label: "refresh" },
            { key: "esc", label: "back" },
            { key: "q", label: "quit" },
          ]}
        />
      </Box>
    </Box>
  );
}

function ManifestView({ manifest }: { manifest: Manifest }): React.ReactElement {
  const { totals, agents_used, models_used, runs, domains } = manifest;
  return (
    <Box flexDirection="column" gap={1}>
      <LabeledBox label="Totals" color={STRUCTURE}>
        <Box flexDirection="column">
          <Row k="Runs" v={String(totals.runs)} />
          <Row k="Tool calls" v={String(totals.tool_calls)} />
          <Row k="Fetches" v={String(totals.fetches)} />
          <Row
            k="Tokens"
            v={
              `in ${fmt(totals.tokens.input)} · out ${fmt(totals.tokens.output)} · ` +
              `cache_read ${fmt(totals.tokens.cache_read)} · cache_create ${fmt(totals.tokens.cache_create)}`
            }
          />
          <Row k="Agents" v={agents_used.join(", ") || "—"} />
          <Row k="Models" v={models_used.join(", ") || "—"} />
          <Row k="Generated" v={manifest.generated_at} />
        </Box>
      </LabeledBox>

      <LabeledBox label={`Runs (${runs.length})`} color={STRUCTURE}>
        <Box flexDirection="column">
          <RunsHeader />
          {runs.map((r, i) => (
            <RunRowView key={i} row={r} />
          ))}
        </Box>
      </LabeledBox>

      {domains.length > 0 ? (
        <LabeledBox label={`Domains fetched (${domains.length})`} color={STRUCTURE}>
          <Box flexDirection="column">
            {domains.map((d, i) => (
              <Box key={i}>
                <Text>{pad(d.domain, 32)}</Text>
                <Text color={STRUCTURE}>{String(d.count).padStart(4, " ")}</Text>
              </Box>
            ))}
          </Box>
        </LabeledBox>
      ) : null}
    </Box>
  );
}

function Row({ k, v }: { k: string; v: string }): React.ReactElement {
  return (
    <Box>
      <Text color={MUTED}>{pad(k, 14)}</Text>
      <Text>{v}</Text>
    </Box>
  );
}

function RunsHeader(): React.ReactElement {
  return (
    <Box>
      <Text bold>{pad("Task", 14)}</Text>
      <Text bold>{pad("Model", 22)}</Text>
      <Text bold>{padRight("Dur", 7)}</Text>
      <Text bold>{padRight("Turns", 6)}</Text>
      <Text bold>{padRight("Tools", 6)}</Text>
      <Text bold>{padRight("Fetch", 6)}</Text>
      <Text bold>{padRight("Tokens (in/out)", 22)}</Text>
      <Text bold>Exit</Text>
    </Box>
  );
}

function RunRowView({ row }: { row: RunRow }): React.ReactElement {
  const tools = Object.values(row.tools).reduce((a, b) => a + b, 0);
  const exitColor = row.exit_code === 0 ? SUCCESS : row.exit_code == null ? STRUCTURE : ERROR;
  return (
    <Box>
      <Text>{pad(row.task_label, 14)}</Text>
      <Text color={MUTED}>{pad(row.model ?? "?", 22)}</Text>
      <Text>{padRight(row.duration_s != null ? `${row.duration_s.toFixed(0)}s` : "—", 7)}</Text>
      <Text>{padRight(String(row.num_turns), 6)}</Text>
      <Text>{padRight(String(tools), 6)}</Text>
      <Text>{padRight(String(row.fetches.length), 6)}</Text>
      <Text>{padRight(`${fmt(row.tokens.input)}/${fmt(row.tokens.output)}`, 22)}</Text>
      <Text color={exitColor}>{row.exit_code != null ? String(row.exit_code) : "—"}</Text>
    </Box>
  );
}

function fmt(n: number): string {
  if (n < 1000) return String(n);
  return n.toLocaleString("en-US");
}

function pad(s: string, w: number): string {
  if (s.length >= w) return s.slice(0, w - 1) + " ";
  return s + " ".repeat(w - s.length);
}

function padRight(s: string, w: number): string {
  if (s.length >= w) return s.slice(0, w - 1) + " ";
  return " ".repeat(w - s.length - 1) + s + " ";
}
