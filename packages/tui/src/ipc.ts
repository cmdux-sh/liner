// Subprocess spawning + NDJSON event stream parsing.

import { spawn, type ChildProcess } from "node:child_process";
import { createInterface } from "node:readline";
import { resolveBinary } from "./bin-resolver.js";
import type { CompileEvent, ProjectSummary } from "./types.js";

export type CompileOptions = {
  /** Project folder path (the mixtape folder containing tape.yaml). */
  folder: string;
  cookies?: string;
  noCache?: boolean;
  skipOptional?: boolean;
  maxTranscriptLength?: number;
  includeSections?: string[];
  excludeSections?: string[];
};

export type CompileHandle = {
  child: ChildProcess;
  cancel: () => void;
  done: Promise<{ code: number | null; stderr: string }>;
};

/** Spawn `liner compile --emit-events` and call `onEvent` for each NDJSON line. */
export function streamCompile(
  opts: CompileOptions,
  onEvent: (event: CompileEvent) => void,
): CompileHandle {
  const bin = resolveBinary();
  const args = [...bin.args, "compile", opts.folder, "--emit-events"];
  if (opts.cookies) args.push("--cookies", opts.cookies);
  if (opts.noCache) args.push("--no-cache");
  if (opts.skipOptional) args.push("--skip-optional");
  if (opts.maxTranscriptLength != null) {
    args.push("--max-transcript-length", String(opts.maxTranscriptLength));
  }
  if (opts.includeSections && opts.includeSections.length > 0) {
    args.push("--include-sections", opts.includeSections.join(","));
  }
  if (opts.excludeSections && opts.excludeSections.length > 0) {
    args.push("--exclude-sections", opts.excludeSections.join(","));
  }

  const child = spawn(bin.command, args, {
    stdio: ["ignore", "pipe", "pipe"],
    env: { ...process.env, PYTHONUNBUFFERED: "1" },
  });

  if (child.stdout) {
    const rl = createInterface({ input: child.stdout });
    rl.on("line", (line: string) => {
      const trimmed = line.trim();
      if (!trimmed) return;
      try {
        const event = JSON.parse(trimmed) as CompileEvent;
        onEvent(event);
      } catch {
        // ignore non-JSON noise; surface via stderr collector
      }
    });
  }

  let stderr = "";
  if (child.stderr) {
    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });
  }

  const done = new Promise<{ code: number | null; stderr: string }>((resolve) => {
    child.on("close", (code) => resolve({ code, stderr }));
  });

  return {
    child,
    cancel: () => {
      if (!child.killed) child.kill("SIGTERM");
    },
    done,
  };
}

/** Run `liner list --json --dir <dir>` and return project folder records. */
export async function listProjects(
  dir: string,
  recursive = false,
): Promise<ProjectSummary[]> {
  const bin = resolveBinary();
  const args = [...bin.args, "list", "--json", "--dir", dir];
  if (recursive) args.push("--recursive");
  const result = await runCapture(bin.command, args);
  if (result.code !== 0) {
    throw new Error(`liner list failed (${result.code}): ${result.stderr}`);
  }
  if (!result.stdout.trim()) return [];
  return JSON.parse(result.stdout) as ProjectSummary[];
}

/** Run `liner init <folder>` to scaffold a project folder. */
export async function runInit(folder: string, opts: { force?: boolean } = {}): Promise<void> {
  const bin = resolveBinary();
  const args = [...bin.args, "init", folder];
  if (opts.force) args.push("--force");
  const result = await runCapture(bin.command, args);
  if (result.code !== 0) {
    throw new Error(result.stderr.trim() || `liner init failed with code ${result.code}`);
  }
}

export type ShareOptions = {
  noWorkingNotes?: boolean;
  noSourceContent?: boolean;
  noPersonal?: boolean;
  minimal?: boolean;
  out?: string;
};

/** Run `liner share <folder>` and return the archive path. */
export async function runShare(
  folder: string,
  opts: ShareOptions = {},
): Promise<string> {
  const bin = resolveBinary();
  const args = [...bin.args, "share", folder];
  if (opts.noWorkingNotes) args.push("--no-working-notes");
  if (opts.noSourceContent) args.push("--no-source-content");
  if (opts.noPersonal) args.push("--no-personal");
  if (opts.minimal) args.push("--minimal");
  if (opts.out) args.push("--out", opts.out);
  const result = await runCapture(bin.command, args);
  if (result.code !== 0) {
    throw new Error(result.stderr.trim() || `liner share failed with code ${result.code}`);
  }
  // CLI prints "Wrote <path> (N entries)" to stderr (Rich console -> stderr).
  const match = (result.stderr + result.stdout).match(/Wrote\s+(\S+)/);
  if (match && match[1]) return match[1];
  if (opts.out) return opts.out;
  // Fallback: derive default location.
  return folder.replace(/\/+$/, "") + ".mixtape";
}

/** Run `liner setup-js --yes` to install/download Playwright Chromium support. */
export async function runSetupJs(): Promise<void> {
  const bin = resolveBinary();
  const result = await runCapture(bin.command, [...bin.args, "setup-js", "--yes"]);
  if (result.code !== 0) {
    throw new Error(
      (result.stderr + result.stdout).trim() ||
        `liner setup-js failed with code ${result.code}`,
    );
  }
}

export type ImportOptions = {
  noRefetch?: boolean;
  cookies?: string;
};

/** Run `liner import <archive> <dest>`. */
export async function runImport(
  archive: string,
  destination: string,
  opts: ImportOptions = {},
): Promise<void> {
  const bin = resolveBinary();
  const args = [...bin.args, "import", archive, destination];
  if (opts.noRefetch) args.push("--no-refetch");
  if (opts.cookies) args.push("--cookies", opts.cookies);
  const result = await runCapture(bin.command, args);
  if (result.code !== 0) {
    throw new Error(result.stderr.trim() || `liner import failed with code ${result.code}`);
  }
}

function runCapture(
  command: string,
  args: string[],
): Promise<{ code: number | null; stdout: string; stderr: string }> {
  return new Promise((resolve) => {
    const child = spawn(command, args, { stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout?.on("data", (c: Buffer) => (stdout += c.toString("utf8")));
    child.stderr?.on("data", (c: Buffer) => (stderr += c.toString("utf8")));
    child.on("close", (code) => resolve({ code, stdout, stderr }));
  });
}
