import React, { useEffect, useRef, useState } from "react";
import { Box, Text, useApp as useInkApp, useInput, useStdout } from "ink";
import Spinner from "ink-spinner";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { spawn } from "node:child_process";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { MUTED, ERROR, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { useApp } from "../store.js";
import { streamCompile, runSetupJs, type CompileHandle } from "../ipc.js";
import { openFolder } from "../open-folder.js";
import { projectFolder } from "../yaml-io.js";
import { markPhaseComplete } from "../progress.js";
import type {
  CompileEvent,
  CompileResultPayload,
  SourceSpec,
  Tape,
} from "../types.js";

type SourceState = {
  spec: SourceSpec;
  status: "pending" | "running" | "done" | "cached" | "failed";
  title?: string | null;
  message?: string;
  bodyPreview?: string;
  bodyChars?: number;
};

type Phase = "running" | "succeeded" | "partial" | "failed";

export function CompileView({
  tape,
  folder,
  showExisting = false,
}: {
  tape: Tape;
  folder: string;
  showExisting?: boolean;
}): React.ReactElement {
  const app = useApp();
  const ink = useInkApp();
  const { stdout } = useStdout();
  const rows = stdout?.rows ?? 40;
  const columns = stdout?.columns ?? 80;
  const project = projectFolder(folder);
  const [total, setTotal] = useState<number>(tape.sources.length);
  const [sources, setSources] = useState<SourceState[]>(
    tape.sources.map((s) => ({ spec: s, status: "pending" })),
  );
  const [phase, setPhase] = useState<Phase>(showExisting ? "partial" : "running");
  const [result, setResult] = useState<CompileResultPayload | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [runId, setRunId] = useState(0);
  const [hasRequestedRun, setHasRequestedRun] = useState(false);
  const viewExisting = showExisting && !hasRequestedRun;
  const [sourceScroll, setSourceScroll] = useState(0);
  const [setupJsState, setSetupJsState] = useState<"idle" | "running">("idle");
  const [setupJsNote, setSetupJsNote] = useState<string | null>(null);
  const handleRef = useRef<CompileHandle | null>(null);

  useEffect(() => {
    if (viewExisting) {
      const existing = existingCompileResult(tape, folder, project);
      setTotal(existing.summary.total);
      setSources(
        tape.sources.map((spec, index) => {
          const source = existing.sources[index];
          const failed = source ? !source.succeeded : false;
          return {
            spec,
            status: failed ? "failed" : "cached",
            title: source?.title ?? sourceTitleFromSpec(spec),
            message:
              failed
                ? existing.warnings.find((w) => w.url === spec.url)?.message ??
                  "Source unavailable in the last compile."
                : undefined,
            bodyPreview: failed ? undefined : existingSourcePreview(project.sourcesDir, source?.filename),
            bodyChars: failed
              ? undefined
              : existingSourceChars(project.sourcesDir, source?.filename),
          };
        }),
      );
      setPhase(existing.summary.failed === 0 ? "succeeded" : "partial");
      setResult(existing);
      setError(null);
      setSetupJsNote(null);
      setSourceScroll(Math.max(0, existing.summary.total - visibleSourceCount(rows)));
      handleRef.current = null;
      return;
    }

    let cursor = -1;
    // Per-run cancellation flag. If the effect re-runs (folder/runId change)
    // or the component unmounts mid-compile, any in-flight callbacks fired
    // by streamCompile or handle.done.then must short-circuit so they don't
    // setState on a stale render.
    let cancelled = false;
    setSources(tape.sources.map((s) => ({ spec: s, status: "pending" })));
    setPhase("running");
    setResult(null);
    setError(null);
    setSetupJsNote(null);
    setSourceScroll(0);

    const handle = streamCompile({ folder }, (event: CompileEvent) => {
      if (cancelled) return;
      if (event.type === "start") {
        setTotal(event.total);
      } else if (event.type === "source_start") {
        cursor += 1;
        const idx = cursor;
        setSources((prev) => updateAt(prev, idx, { status: "running", spec: event.spec }));
        setSourceScroll(scrollSourceIntoView(idx, rows));
      } else if (event.type === "source_done" || event.type === "source_cached") {
        const idx = cursor;
        setSources((prev) =>
          updateAt(prev, idx, {
            status: event.type === "source_cached" ? "cached" : "done",
            title: event.title,
            bodyPreview: event.body_preview,
            bodyChars: event.body_chars,
          }),
        );
        setSourceScroll(scrollSourceIntoView(idx, rows));
      } else if (event.type === "source_failed") {
        const idx = cursor;
        setSources((prev) =>
          updateAt(prev, idx, {
            status: "failed",
            message: event.message,
          }),
        );
        setSourceScroll(scrollSourceIntoView(idx, rows));
      } else if (event.type === "result") {
        setResult(event.payload);
        setSourceScroll(Math.max(0, event.payload.summary.total - visibleSourceCount(rows)));
        const summary = event.payload.summary;
        if (summary.failed === 0) {
          setPhase("succeeded");
          // Compile is the final progress-advancing phase. A successful
          // compile means the mixtape is ready to use.
          markPhaseComplete(folder, "compile");
        } else if (summary.succeeded === 0) {
          setPhase("failed");
        } else {
          setPhase("partial");
          // A partial compile still writes MIXTAPE.md + sources/ and is usable
          // with warnings. Count it as complete so the hub does not keep
          // presenting an already-built mixtape as "ready to compile."
          markPhaseComplete(folder, "compile");
        }
      }
    });
    handleRef.current = handle;

    handle.done.then(({ code, stderr }) => {
      if (cancelled) return;
      // 0 = full success, 2 = partial, 3 = total failure (per CLI conventions).
      if (code !== 0 && code !== 2 && code !== 3) {
        setError(stderr.trim() || `compile exited with code ${code}`);
        setPhase("failed");
      }
    });

    return () => {
      cancelled = true;
      handle.cancel();
    };
    // runId in deps so pressing `r` re-runs by re-running the effect.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [folder, runId, viewExisting]);

  function copyMixtapeToClipboard(): void {
    let text: string;
    try {
      text = readFileSync(project.mixtapePath, "utf8");
    } catch (e) {
      app.setNotification({
        kind: "error",
        message: `Could not read MIXTAPE.md: ${(e as Error).message}`,
      });
      return;
    }
    const cmd =
      process.platform === "darwin"
        ? { command: "pbcopy", args: [] as string[] }
        : process.platform === "linux"
          ? { command: "xclip", args: ["-selection", "clipboard"] }
          : null;
    if (!cmd) {
      app.setNotification({
        kind: "error",
        message: "No clipboard helper found (need pbcopy/xclip).",
      });
      return;
    }
    const proc = spawn(cmd.command, cmd.args);
    proc.stdin?.end(text, "utf8");
    proc.on("close", (code) => {
      if (code === 0) {
        app.setNotification({ kind: "info", message: "Copied MIXTAPE.md to clipboard." });
      } else {
        app.setNotification({ kind: "error", message: `Clipboard helper exited ${code}` });
      }
    });
    proc.on("error", (e) => {
      app.setNotification({ kind: "error", message: e.message });
    });
  }

  function setupJsAndRetry(): void {
    if (setupJsState === "running") return;
    setSetupJsState("running");
    setSetupJsNote("Installing JS rendering support…");
    void runSetupJs()
      .then(() => {
        setSetupJsNote("JS rendering is ready. Retrying compile…");
        setHasRequestedRun(true);
        setRunId((id) => id + 1);
      })
      .catch((e: Error) => {
        setSetupJsNote(`JS setup failed: ${e.message}`);
      })
      .finally(() => {
        setSetupJsState("idle");
      });
  }

  useInput((input, key) => {
    const maxSourceScroll = Math.max(0, sources.length - visibleSourceCount(rows));
    if (key.upArrow) {
      setSourceScroll((n) => Math.max(0, n - 1));
      return;
    }
    if (key.downArrow) {
      setSourceScroll((n) => Math.min(maxSourceScroll, n + 1));
      return;
    }
    if (key.pageUp) {
      setSourceScroll((n) => Math.max(0, n - Math.max(1, visibleSourceCount(rows) - 1)));
      return;
    }
    if (key.pageDown) {
      setSourceScroll((n) =>
        Math.min(maxSourceScroll, n + Math.max(1, visibleSourceCount(rows) - 1)),
      );
      return;
    }
    if (phase === "running") {
      if (input === "q") {
        ink.exit();
      } else if (key.escape) {
        handleRef.current?.cancel();
        app.navigate({ kind: "hub", tape, folder });
      }
      return;
    }
    const canSetupJs = result ? needsJsSetup(result) : false;
    if (input === "j" && canSetupJs) setupJsAndRetry();
    else if (input === "y") copyMixtapeToClipboard();
    else if (input === "o") openFolder(folder);
    else if (input === "r") {
      setHasRequestedRun(true);
      setRunId((id) => id + 1);
    }
    else if (input === "q") ink.exit();
    else if (input === "b" || key.escape) {
      app.navigate({ kind: "hub", tape, folder });
    }
  });

  // "Starting up" is the window between mount and the first source flipping
  // off `pending`. The Python subprocess imports trafilatura / yt-dlp / etc.
  // on every cold launch and that can take 5-60 seconds on first run, during
  // which nothing visible happens — the user reads that as "frozen." This
  // banner gives them an explicit "yes, work is happening" signal.
  const anyProgress = sources.some((s) => s.status !== "pending");
  const startingUp = phase === "running" && !anyProgress;
  const showSetupJs = result ? needsJsSetup(result) : false;
  const sourceCount = visibleSourceCount(rows);
  const maxSourceScroll = Math.max(0, sources.length - sourceCount);
  const safeSourceScroll = Math.min(sourceScroll, maxSourceScroll);
  const visibleSources = sources.slice(safeSourceScroll, safeSourceScroll + sourceCount);
  const hiddenBefore = safeSourceScroll;
  const hiddenAfter = Math.max(0, sources.length - safeSourceScroll - visibleSources.length);

  return (
    <Box flexDirection="column">
      <Header
        title="compile"
        subtitle={`${folder}  →  ${total} source${total === 1 ? "" : "s"}`}
      />

      {startingUp ? (
        <Box flexDirection="column" marginBottom={1}>
          <Box>
            <Text color={STRUCTURE}>
              <Spinner type="dots" />{" "}
            </Text>
            <Text bold>Starting compile…</Text>
          </Box>
          <Text color={MUTED}>
            {"  "}Spinning up the Python fetcher. Cold start can take up to a minute the first time;
          </Text>
          <Text color={MUTED}>
            {"  "}subsequent runs are seconds. Press <Text color={STRUCTURE}>esc</Text> to cancel.
          </Text>
        </Box>
      ) : null}

      <Box flexDirection="column" marginBottom={1}>
        {hiddenBefore > 0 ? (
          <Text color={MUTED}>
            ↑ {hiddenBefore} earlier source{hiddenBefore === 1 ? "" : "s"} above
          </Text>
        ) : null}
        {visibleSources.map((s, i) => (
          <SourceLine key={safeSourceScroll + i} index={safeSourceScroll + i} state={s} />
        ))}
        {hiddenAfter > 0 ? (
          <Text color={MUTED}>
            ↓ {hiddenAfter} more source{hiddenAfter === 1 ? "" : "s"} below
          </Text>
        ) : null}
      </Box>

      {error ? (
        <Box marginBottom={1}>
          <Text color={ERROR}>{error}</Text>
        </Box>
      ) : null}

      {result ? (
        <Box marginBottom={1}>
          <LabeledBox label="compile result" color={phaseColor(phase)}>
            <Text bold>
              <Text color={phaseColor(phase)}>● {phase}</Text>
              {"  "}
              <Text color={MUTED}>
                {result.summary.succeeded}/{result.summary.total} succeeded
              </Text>
            </Text>
            <Text color={MUTED}>
              wrote {result.mixtape_path}
            </Text>
            <Text color={MUTED}>
              sources/ → {result.sources.length} file{result.sources.length === 1 ? "" : "s"}
            </Text>
            {result.warnings.length > 0 ? (
              <Box flexDirection="column" marginTop={1}>
                <Text color={WARNING}>{result.warnings.length} warning{result.warnings.length === 1 ? "" : "s"}:</Text>
                {result.warnings.slice(0, 4).map((w, i) => (
                  <Text key={i} color={MUTED}>
                    {formatCompileWarningLine(w, columns)}
                  </Text>
                ))}
              </Box>
            ) : null}
            {showSetupJs ? (
              <Box flexDirection="column" marginTop={1}>
                <Text color={STRUCTURE} bold>JS rendering needed</Text>
                <Text color={MUTED}>
                  Some pages require headless Chromium to fetch correctly. Press{" "}
                  <Text color={STRUCTURE}>j</Text> to run{" "}
                  <Text color={STRUCTURE}>liner setup-js --yes</Text> (~150MB).
                </Text>
                <Text color={MUTED}>
                  Liner retries this compile after setup.
                </Text>
              </Box>
            ) : null}
          </LabeledBox>
        </Box>
      ) : null}

      {/* No NextSteps panel here — the result LabeledBox tells the user
          what happened, the KeyHints below tells them what they can do.
          Adding a third box that restates the keys just adds clutter. */}

      {/* Transient notification (e.g. "copied MIXTAPE.md"). Short-lived
          confirmations stay here. */}
      {app.notification ? (
        <Box marginBottom={1}>
          <Text color={app.notification.kind === "error" ? ERROR : SUCCESS}>
            {app.notification.message}
          </Text>
        </Box>
      ) : null}

      {setupJsNote ? (
        <Box marginBottom={1}>
          <Text color={setupJsNote.startsWith("JS setup failed") ? ERROR : SUCCESS}>
            {setupJsState === "running" ? <Spinner type="dots" /> : null}
            {setupJsState === "running" ? " " : ""}
            {setupJsNote}
          </Text>
        </Box>
      ) : null}

      <KeyHints
        hints={
          phase === "running"
            ? [
                { key: "↑↓", label: "scroll sources" },
                { key: "pgup/pgdn", label: "jump sources" },
                { key: "esc", label: "cancel" },
                { key: "q", label: "quit" },
              ]
            : [
                ...(showSetupJs
                  ? [{ key: "j", label: "install JS rendering" }]
                  : []),
                { key: "↑↓", label: "scroll sources" },
                { key: "pgup/pgdn", label: "jump sources" },
                { key: "y", label: "copy MIXTAPE.md" },
                { key: "o", label: "open folder" },
                // Re-running always re-invokes the full compile, but cached
                // successes return instantly — so for a partial/failed run
                // the user-visible effect is "retry the broken ones."
                {
                  key: "r",
                  label: phase === "succeeded" ? "re-run" : "retry failed",
                },
                { key: "b", label: "back" },
                { key: "q", label: "quit" },
              ]
        }
      />
    </Box>
  );
}

function SourceLine({
  index,
  state,
}: {
  index: number;
  state: SourceState;
}): React.ReactElement {
  const icon = state.spec.type === "youtube" ? "▶" : "◌";
  const url = truncate(state.spec.url, 50);
  return (
    <Box flexDirection="column">
      <Box>
        <Text>
          {String(index + 1).padStart(2, " ")}. {icon} {url}
          {"  "}
        </Text>
        <StatusBadge status={state.status} />
      </Box>
      {state.title ? (
        <Box paddingLeft={6}>
          <Text color={MUTED}>{truncate(state.title, 62)}</Text>
        </Box>
      ) : null}
      {state.bodyPreview && state.status !== "failed" ? (
        <Box paddingLeft={6}>
          <Text color={MUTED}>
            “{truncate(state.bodyPreview.replace(/\s+/g, " "), 76)}”
            {state.bodyChars != null
              ? `  (${state.bodyChars.toLocaleString()} chars)`
              : ""}
          </Text>
        </Box>
      ) : null}
      {state.message ? (
        <Box paddingLeft={6}>
          <Text color={ERROR}>{truncate(state.message, 100)}</Text>
        </Box>
      ) : null}
    </Box>
  );
}

function StatusBadge({ status }: { status: SourceState["status"] }): React.ReactElement {
  if (status === "pending") return <Text color={MUTED}>waiting</Text>;
  if (status === "running")
    return (
      <Text color={STRUCTURE}>
        <Spinner type="dots" /> fetching
      </Text>
    );
  if (status === "done") return <Text color={SUCCESS}>✓ fetched</Text>;
  if (status === "cached") return <Text color={SUCCESS}>✓ cached</Text>;
  return <Text color={ERROR}>✗ failed</Text>;
}

function updateAt(
  arr: SourceState[],
  idx: number,
  patch: Partial<SourceState>,
): SourceState[] {
  if (idx < 0 || idx >= arr.length) return arr;
  const next = arr.slice();
  const current = next[idx];
  if (!current) return arr;
  next[idx] = { ...current, ...patch };
  return next;
}

function phaseColor(p: Phase): string {
  if (p === "succeeded") return SUCCESS;
  if (p === "partial") return WARNING;
  return ERROR;
}

function visibleSourceCount(rows: number): number {
  // Compile source rows are usually 2-3 terminal lines once title/preview are
  // present, and long result/warning boxes can easily consume 15+ rows. Keep a
  // generous reservation so the top of the source viewport never clips behind
  // the terminal chrome or Ink's bottom-anchored render.
  return Math.max(2, Math.min(6, Math.floor((rows - 32) / 3)));
}

function scrollSourceIntoView(index: number, rows: number): number {
  return Math.max(0, index - visibleSourceCount(rows) + 1);
}

function existingCompileResult(
  tape: Tape,
  folder: string,
  project: ReturnType<typeof projectFolder>,
): CompileResultPayload {
  const warnings = parseCompilationWarnings(project.mixtapePath);
  const sourceFiles = listExistingSourceFiles(project.sourcesDir);
  const sources = tape.sources.map((spec, index) => {
    const filename = sourceFiles.get(index + 1) ?? "";
    const body = filename ? readExistingSourceFile(project.sourcesDir, filename) : "";
    const unavailable = body.includes("_Source unavailable.");
    return {
      index: index + 1,
      filename,
      path: filename ? `${project.sourcesDir}/${filename}` : "",
      url: spec.type === "local_file" ? spec.citation || spec.path || "" : spec.url,
      type: spec.type,
      section: spec.section ?? null,
      title: existingTitle(body) ?? sourceTitleFromSpec(spec),
      succeeded: Boolean(filename) && !unavailable,
    };
  });
  const succeeded = sources.filter((source) => source.succeeded).length;
  return {
    tape: {
      title: tape.title,
      description: tape.description,
      curator: tape.curator,
      version: tape.version,
      mode: tape.mode ?? null,
      jtbd: tape.jtbd ?? null,
    },
    compiled_at: "",
    folder,
    mixtape_path: project.mixtapePath,
    sources,
    warnings,
    summary: {
      total: sources.length,
      succeeded,
      failed: sources.length - succeeded,
    },
  };
}

function parseCompilationWarnings(
  mixtapePath: string,
): CompileResultPayload["warnings"] {
  if (!existsSync(mixtapePath)) return [];
  const text = readFileSync(mixtapePath, "utf8");
  const notesStart = text.indexOf("\n## Compilation notes");
  if (notesStart === -1) return [];
  return text
    .slice(notesStart)
    .split("\n")
    .flatMap((line) => {
      const match = line.match(/^- \*\*(.*?)\*\* — (.*)$/);
      if (!match) return [];
      return [
        {
          url: match[1]!,
          message: match[2]!,
          severity: match[2]!.toLowerCase().startsWith("failed") ? "error" : "warning",
        },
      ];
    });
}

function listExistingSourceFiles(sourcesDir: string): Map<number, string> {
  const out = new Map<number, string>();
  if (!existsSync(sourcesDir)) return out;
  for (const filename of readdirSync(sourcesDir)) {
    const match = filename.match(/^(\d+)-.*\.md$/);
    if (match) out.set(Number(match[1]), filename);
  }
  return out;
}

function readExistingSourceFile(sourcesDir: string, filename: string | undefined): string {
  if (!filename) return "";
  try {
    return readFileSync(`${sourcesDir}/${filename}`, "utf8");
  } catch {
    return "";
  }
}

function existingTitle(body: string): string | null {
  const first = body.split("\n").find((line) => line.startsWith("# "));
  return first ? first.replace(/^#\s+/, "").trim() || null : null;
}

function existingSourcePreview(sourcesDir: string, filename: string | undefined): string {
  const body = readExistingSourceFile(sourcesDir, filename);
  return body
    .split("\n")
    .filter((line) => line.trim() && !line.startsWith("#") && !line.startsWith("**"))
    .join(" ")
    .replace(/\s+/g, " ")
    .trim();
}

function existingSourceChars(sourcesDir: string, filename: string | undefined): number {
  return existingSourcePreview(sourcesDir, filename).length;
}

function sourceTitleFromSpec(spec: SourceSpec): string {
  if (spec.type === "local_file") return spec.citation || spec.path || "local file";
  return spec.url;
}

function needsJsSetup(result: CompileResultPayload): boolean {
  return result.warnings.some((w) => {
    const msg = w.message.toLowerCase();
    return msg.includes("liner setup-js") ||
      msg.includes("playwright chromium isn't installed") ||
      msg.includes("render: js needs playwright") ||
      msg.includes("js-rendering support");
  });
}

export function formatCompileWarningLine(
  warning: CompileResultPayload["warnings"][number],
  terminalColumns: number,
): string {
  const boxWidth = Math.max(40, terminalColumns - 2);
  const contentWidth = Math.max(24, boxWidth - 6);
  const prefix = `  ${warning.severity === "error" ? "✗" : "⚠"} `;
  const separator = " — ";
  const available = Math.max(12, contentWidth - prefix.length - separator.length);
  const urlWidth = Math.min(50, Math.max(18, Math.floor(available * 0.38)));
  const messageWidth = Math.max(8, available - urlWidth);
  return `${prefix}${truncate(warning.url, urlWidth)}${separator}${truncate(warning.message, messageWidth)}`;
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}
