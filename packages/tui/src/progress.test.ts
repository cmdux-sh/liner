import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, existsSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  PHASE_ORDER,
  TOTAL_STEPS,
  markPhaseComplete,
  migrateProgress,
  progressFileExists,
  readProgress,
  writeProgress,
} from "./progress.js";
import type { Tape } from "./types.js";

function makeProject(): string {
  const dir = mkdtempSync(join(tmpdir(), "liner-progress-"));
  mkdirSync(join(dir, "working"), { recursive: true });
  return dir;
}

function emptyTape(): Tape {
  return {
    title: "test",
    description: "",
    version: 1,
    curator: "",
    sources: [],
    tags: [],
    mode: "quick",
    jtbd: null,
  };
}

describe("progress.ts", () => {
  let dir: string;
  beforeEach(() => {
    dir = makeProject();
  });
  afterEach(() => {
    rmSync(dir, { recursive: true, force: true });
  });

  it("starts at step 0 when no file exists", () => {
    expect(progressFileExists(dir)).toBe(false);
    const p = readProgress(dir);
    expect(p.step).toBe(0);
  });

  it("round-trips through writeProgress", () => {
    writeProgress(dir, { step: 4, lastTouched: "2026-05-19T22:00:00.000Z" });
    expect(progressFileExists(dir)).toBe(true);
    const p = readProgress(dir);
    expect(p.step).toBe(4);
    expect(p.lastTouched).toBe("2026-05-19T22:00:00.000Z");
  });

  it("clamps invalid step values on read", () => {
    writeFileSync(join(dir, ".liner-progress.json"), JSON.stringify({ step: 999 }), "utf8");
    expect(readProgress(dir).step).toBe(TOTAL_STEPS);
  });

  it("defaults to step 0 on malformed JSON", () => {
    writeFileSync(join(dir, ".liner-progress.json"), "not json", "utf8");
    expect(readProgress(dir).step).toBe(0);
  });

  it("markPhaseComplete advances only when caller is at the expected phase", () => {
    expect(readProgress(dir).step).toBe(0);
    // Trying to mark a later phase done from step 0 is a no-op — out of order.
    markPhaseComplete(dir, "candidates");
    expect(readProgress(dir).step).toBe(0);
    // The actual current phase advances cleanly. Step through the full
    // prefix: framing → gate0 → candidates.
    markPhaseComplete(dir, "framing");
    expect(readProgress(dir).step).toBe(1);
    markPhaseComplete(dir, "gate0");
    expect(readProgress(dir).step).toBe(2);
    markPhaseComplete(dir, "candidates");
    expect(readProgress(dir).step).toBe(3);
  });

  it("markPhaseComplete is idempotent at the same phase", () => {
    markPhaseComplete(dir, "framing");
    markPhaseComplete(dir, "framing");
    markPhaseComplete(dir, "framing");
    expect(readProgress(dir).step).toBe(1);
  });

  it("never advances past TOTAL_STEPS", () => {
    writeProgress(dir, { step: TOTAL_STEPS, lastTouched: "x" });
    markPhaseComplete(dir, "compile");
    expect(readProgress(dir).step).toBe(TOTAL_STEPS);
  });

  it("migrateProgress sets step = longest consecutive done prefix", () => {
    // Populate only working/01 with real content. Framing should count;
    // candidates shouldn't (placeholder).
    writeFileSync(
      join(dir, "working/01-jtbd-and-knowledge-map.md"),
      "# JTBD and knowledge map\n\nWhen I'm testing the progress migration, I want a valid JTBD string in the tape, so I can verify the parser accepts it.\n",
      "utf8",
    );
    const tape: Tape = { ...emptyTape(), jtbd: "When I'm testing the progress migration, I want a valid JTBD string in the tape, so I can verify the parser accepts it." };
    const p = migrateProgress(dir, tape);
    expect(p.step).toBe(1);
    expect(progressFileExists(dir)).toBe(true);
  });

  it("migrateProgress ignores out-of-order artifacts", () => {
    // tape.yaml has sources (Assembly artifact present) but earlier phases
    // are empty. Should still resolve step = 0.
    const tape: Tape = {
      ...emptyTape(),
      sources: [
        { type: "web", url: "https://example.com", priority: "required" },
      ],
    };
    const p = migrateProgress(dir, tape);
    expect(p.step).toBe(0);
  });

  it("PHASE_ORDER and TOTAL_STEPS stay in sync", () => {
    expect(TOTAL_STEPS).toBe(PHASE_ORDER.length);
  });

  it("write produces stable JSON", () => {
    writeProgress(dir, { step: 3, lastTouched: "2026-05-19T22:00:00.000Z" });
    const text = readFileSync(join(dir, ".liner-progress.json"), "utf8");
    expect(text).toContain('"step": 3');
    expect(text.endsWith("\n")).toBe(true);
  });

  it("existsSync sanity (smoke for test environment)", () => {
    // Confirms our temp dir is real and writable — guards against the suite
    // accidentally pointing at a stale fixture path.
    expect(existsSync(dir)).toBe(true);
  });
});
