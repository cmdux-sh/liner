import { describe, it, expect } from "vitest";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  detectSourceType,
  emptyTape,
  isProjectFolder,
  projectFolder,
  readSynthesisStatus,
  readTape,
  slugify,
  validateTape,
  writeTape,
} from "./yaml-io.js";

const TMP = mkdtempSync(join(tmpdir(), "liner-tui-"));

describe("detectSourceType", () => {
  it("recognizes youtube URLs", () => {
    expect(detectSourceType("https://www.youtube.com/watch?v=abc")).toBe("youtube");
    expect(detectSourceType("https://youtu.be/xyz")).toBe("youtube");
  });
  it("defaults to web for plain http(s) URLs", () => {
    expect(detectSourceType("https://example.com")).toBe("web");
  });
  it("treats personal/ paths as local_file", () => {
    expect(detectSourceType("personal/foo.pdf")).toBe("local_file");
    expect(detectSourceType("personal/sub/dir/x.md")).toBe("local_file");
    expect(detectSourceType("local-sources/foo.pdf")).toBe("local_file");
  });
  it("recognizes likely skill URLs", () => {
    expect(detectSourceType("https://github.com/user/repo/tree/main/skills/writing")).toBe("skill");
    expect(detectSourceType("https://github.com/user/repo/blob/main/SKILL.md")).toBe("skill");
  });
});

describe("slugify", () => {
  it("slugifies titles", () => {
    expect(slugify("Mobile Design — Foundations")).toBe("mobile-design-foundations");
    expect(slugify("")).toBe("untitled");
    expect(slugify("///")).toBe("untitled");
  });
  it("truncates long input", () => {
    expect(slugify("a".repeat(200), 30).length).toBeLessThanOrEqual(30);
  });
});

describe("local_file validation", () => {
  it("accepts a well-formed local_file source", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "personal/foo.pdf",
        citation: "Author, 2024",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    expect(validateTape(tape)).toEqual([]);
  });

  it("accepts local-sources/ paths", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "local-sources/foo.pdf",
        citation: "Author, 2024",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    expect(validateTape(tape)).toEqual([]);
  });

  it("requires path", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: null,
        citation: "x",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "sources[0].path")).toBeTruthy();
  });

  it("requires citation", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "personal/foo.md",
        citation: null,
        note: null,
        section: null,
        priority: "required",
      },
    ];
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "sources[0].citation")).toBeTruthy();
  });

  it("rejects paths outside local source folders", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "docs/foo.md",
        citation: "x",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "sources[0].path")).toBeTruthy();
  });

  it("rejects bad extensions", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "personal/payload.exe",
        citation: "x",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "sources[0].path")).toBeTruthy();
  });
});

describe("web render validation", () => {
  it("accepts render: js", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        render: "js",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    expect(validateTape(tape)).toEqual([]);
  });

  it("rejects invalid render value", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        render: "turbo" as any,
        note: null,
        section: null,
        priority: "required",
      },
    ];
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "sources[0].render")).toBeTruthy();
  });
});

describe("yaml round-trip with new fields", () => {
  it("round-trips a local_file source", () => {
    const path = join(TMP, "local-roundtrip.yaml");
    const tape = emptyTape("me");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "local_file",
        url: "",
        path: "personal/chapter.pdf",
        citation: "Doe, 2024",
        note: "the foundational chapter",
        section: "foundations",
        priority: "required",
      },
    ];
    writeTape(path, tape);
    const { tape: reread } = readTape(path);
    expect(reread.sources[0]?.type).toBe("local_file");
    expect(reread.sources[0]?.path).toBe("personal/chapter.pdf");
    expect(reread.sources[0]?.citation).toBe("Doe, 2024");
  });

  it("round-trips render: js on a web source", () => {
    const path = join(TMP, "render-roundtrip.yaml");
    const tape = emptyTape("me");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        render: "js",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    writeTape(path, tape);
    const { tape: reread } = readTape(path);
    expect(reread.sources[0]?.render).toBe("js");
  });

  it("round-trips render: server (explicit opt-out of auto-fallback)", () => {
    const path = join(TMP, "render-server.yaml");
    const tape = emptyTape("me");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        render: "server",
        note: null,
        section: null,
        priority: "required",
      },
    ];
    writeTape(path, tape);
    const text = readFileSync(path, "utf8");
    expect(text).toContain("render: server");
    const { tape: reread } = readTape(path);
    expect(reread.sources[0]?.render).toBe("server");
  });

  it("omits render when null (auto default)", () => {
    const path = join(TMP, "render-auto.yaml");
    const tape = emptyTape("me");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        render: null,
        note: null,
        section: null,
        priority: "required",
      },
    ];
    writeTape(path, tape);
    const text = readFileSync(path, "utf8");
    expect(text).not.toContain("render:");
    const { tape: reread } = readTape(path);
    expect(reread.sources[0]?.render).toBeFalsy();
  });
});

describe("validateTape", () => {
  it("flags missing fields", () => {
    const tape = emptyTape();
    tape.title = "";
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "title")).toBeTruthy();
    expect(errs.find((e) => e.field === "sources")).toBeTruthy();
  });

  it("rejects invalid modes", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      { type: "web", url: "https://example.com", note: null, section: null, priority: "required" },
    ];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (tape as any).mode = "turbo";
    const errs = validateTape(tape);
    expect(errs.find((e) => e.field === "mode")).toBeTruthy();
  });

  it("accepts a complete tape", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.mode = "methodology";
    tape.jtbd = "When I'm bootstrapping a design knowledge base for AI-assisted critique, I want to assemble a foundational corpus of mobile design references, so I can compile mixtapes for specific design questions without restarting from zero.";
    tape.sources = [
      { type: "web", url: "https://example.com", note: null, section: null, priority: "required" },
    ];
    expect(validateTape(tape)).toEqual([]);
  });
});

describe("yaml round-trip", () => {
  it("writes a tape and reads it back identically", () => {
    const path = join(TMP, "round.yaml");
    const tape = emptyTape("me");
    tape.title = "Round Trip";
    tape.description = "yes";
    tape.mode = "quick";
    tape.jtbd = "Test JTBD survival.";
    tape.sources = [
      {
        type: "youtube",
        url: "https://www.youtube.com/watch?v=aircAruvnKk",
        note: "test note",
        section: "intro",
        priority: "required",
      },
    ];
    writeTape(path, tape);
    const { tape: reread } = readTape(path);
    expect(reread.title).toBe("Round Trip");
    expect(reread.mode).toBe("quick");
    expect(reread.jtbd).toBe("Test JTBD survival.");
    expect(reread.sources).toHaveLength(1);
    expect(reread.sources[0]?.url).toBe("https://www.youtube.com/watch?v=aircAruvnKk");
    expect(reread.sources[0]?.section).toBe("intro");
  });

  it("preserves user comments on existing tapes", () => {
    const path = join(TMP, "with-comments.yaml");
    writeFileSync(
      path,
      `# This is my favorite tape — keep this comment!
title: Existing
description: has comments
version: 1
curator: me
sources:
  - type: web
    url: https://example.com
`,
      "utf8",
    );
    const { tape, doc } = readTape(path);
    tape.title = "Edited";
    writeTape(path, tape, doc);
    const after = readFileSync(path, "utf8");
    expect(after).toContain("# This is my favorite tape");
    expect(after).toContain("title: Edited");
  });
});

describe("projectFolder helpers", () => {
  it("exposes canonical paths", () => {
    const p = projectFolder("/tmp/demo");
    expect(p.rootPath).toBe("/tmp/demo");
    expect(p.path).toBe("/tmp/demo/mixtape");
    expect(p.tapePath).toBe("/tmp/demo/mixtape/tape.yaml");
    expect(p.synthesisPath).toBe("/tmp/demo/mixtape/synthesis.md");
    expect(p.mixtapePath).toBe("/tmp/demo/mixtape/MIXTAPE.md");
    expect(p.sourcesDir).toBe("/tmp/demo/mixtape/sources");
    expect(p.workingDir).toBe("/tmp/demo/mixtape/working");
  });

  it("isProjectFolder reflects presence of tape.yaml", () => {
    const empty = join(TMP, "empty-dir");
    mkdirSync(empty, { recursive: true });
    expect(isProjectFolder(empty)).toBe(false);

    writeFileSync(join(empty, "tape.yaml"), "title: x\n", "utf8");
    expect(isProjectFolder(empty)).toBe(true);
  });

  it("keeps legacy root tape folders readable", () => {
    const legacy = join(TMP, "legacy-root");
    mkdirSync(legacy, { recursive: true });
    writeFileSync(join(legacy, "tape.yaml"), "title: x\n", "utf8");
    const p = projectFolder(legacy);
    expect(p.path).toBe(legacy);
    expect(p.tapePath).toBe(join(legacy, "tape.yaml"));
  });

  it("lets the v2 project marker win over a legacy root tape", () => {
    const root = join(TMP, "v2-marker-with-legacy-root");
    mkdirSync(root, { recursive: true });
    writeFileSync(join(root, "liner.yaml"), "version: 2\nartifact: liner\nmixtape: mixtape\n", "utf8");
    writeFileSync(join(root, "tape.yaml"), "title: x\n", "utf8");

    const p = projectFolder(root);
    expect(p.path).toBe(join(root, "mixtape"));
    expect(p.tapePath).toBe(join(root, "mixtape", "tape.yaml"));
  });
});

describe("readSynthesisStatus", () => {
  it("reports missing synthesis", () => {
    const dir = join(TMP, "no-syn");
    mkdirSync(dir, { recursive: true });
    const status = readSynthesisStatus(projectFolder(dir));
    expect(status.exists).toBe(false);
    expect(status.isReady).toBe(false);
  });

  it("detects placeholder text", () => {
    const dir = join(TMP, "placeholder-syn");
    mkdirSync(join(dir, "mixtape"), { recursive: true });
    writeFileSync(
      join(dir, "mixtape", "synthesis.md"),
      "# Synthesis\n\nReplace this placeholder with the curator's distilled view.\n",
      "utf8",
    );
    const status = readSynthesisStatus(projectFolder(dir));
    expect(status.exists).toBe(true);
    expect(status.isPlaceholder).toBe(true);
    expect(status.isReady).toBe(false);
  });

  it("recognizes a real synthesis", () => {
    const dir = join(TMP, "real-syn");
    mkdirSync(join(dir, "mixtape"), { recursive: true });
    writeFileSync(
      join(dir, "mixtape", "synthesis.md"),
      "This is a real synthesis with substantive content from the curator.",
      "utf8",
    );
    const status = readSynthesisStatus(projectFolder(dir));
    expect(status.exists).toBe(true);
    expect(status.isReady).toBe(true);
    expect(status.charCount).toBeGreaterThan(20);
  });
});

// ---------------------------------------------------------------------------
// New v0.4 tape fields — round-trip + validation
// ---------------------------------------------------------------------------

describe("tape format additions (jtbd_clarifications, parent, kind)", () => {
  function writeAndRead(tapeYaml: string): ReturnType<typeof readTape> {
    const dir = join(TMP, `roundtrip-${Math.random().toString(36).slice(2)}`);
    mkdirSync(dir, { recursive: true });
    const path = join(dir, "tape.yaml");
    writeFileSync(path, tapeYaml, "utf8");
    return readTape(path);
  }

  it("round-trips jtbd_clarifications", () => {
    const yaml = [
      "title: T",
      "description: d",
      "version: 1",
      "curator: c",
      "jtbd: When I'm designing a developer CLI from scratch, I want to follow established interaction patterns from well-loved tools, so I can ship something that feels familiar in the first 30 seconds.",
      "jtbd_clarifications:",
      "  - question: How weighted are CLI and TUI?",
      "    answer: Both equally.",
      "  - question: Quality anchors?",
      "    answer: ripgrep, fzf",
      "sources:",
      "  - type: web",
      "    url: https://example.com",
      "",
    ].join("\n");
    const { tape } = writeAndRead(yaml);
    expect(tape.jtbd_clarifications).toHaveLength(2);
    expect(tape.jtbd_clarifications?.[0]?.question).toBe(
      "How weighted are CLI and TUI?",
    );
    expect(tape.jtbd_clarifications?.[1]?.answer).toBe("ripgrep, fzf");
  });

  it("treats jtbd_clarifications as optional", () => {
    const yaml = [
      "title: T",
      "description: d",
      "version: 1",
      "curator: c",
      "sources:",
      "  - type: web",
      "    url: https://example.com",
      "",
    ].join("\n");
    const { tape } = writeAndRead(yaml);
    expect(tape.jtbd_clarifications).toBeNull();
  });

  it("skips malformed jtbd_clarifications entries silently", () => {
    const yaml = [
      "title: T",
      "description: d",
      "version: 1",
      "curator: c",
      "jtbd_clarifications:",
      "  - question: Real question",
      "    answer: Real answer",
      "  - { broken: shape }",  // missing question/answer keys
      "sources:",
      "  - type: web",
      "    url: https://example.com",
      "",
    ].join("\n");
    const { tape } = writeAndRead(yaml);
    expect(tape.jtbd_clarifications).toHaveLength(1);
  });

  it("round-trips parent", () => {
    const yaml = [
      "title: T",
      "description: d",
      "version: 1",
      "curator: c",
      "parent: /tmp/source-folder",
      "sources:",
      "  - type: web",
      "    url: https://example.com",
      "",
    ].join("\n");
    const { tape } = writeAndRead(yaml);
    expect(tape.parent).toBe("/tmp/source-folder");
  });

  it("round-trips source kind", () => {
    const yaml = [
      "title: T",
      "description: d",
      "version: 1",
      "curator: c",
      "sources:",
      "  - type: web",
      "    url: https://example.com",
      "    kind: principle",
      "  - type: web",
      "    url: https://example.org",
      "    kind: reference",
      "",
    ].join("\n");
    const { tape } = writeAndRead(yaml);
    expect(tape.sources[0]?.kind).toBe("principle");
    expect(tape.sources[1]?.kind).toBe("reference");
  });

  it("rejects invalid source kinds during validation", () => {
    const tape = emptyTape("tester");
    tape.title = "T";
    tape.description = "d";
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        priority: "required",
        kind: "principal" as unknown as "principle", // typo
      },
    ];
    const errs = validateTape(tape);
    const kindErr = errs.find((e) => e.field === "sources[0].kind");
    expect(kindErr).toBeDefined();
  });

  it("write+read preserves all three new fields together", () => {
    const dir = join(TMP, "all-three");
    mkdirSync(dir, { recursive: true });
    const path = join(dir, "tape.yaml");
    const tape = emptyTape("tester");
    tape.title = "Composite";
    tape.description = "d";
    tape.parent = "/tmp/parent";
    tape.jtbd_clarifications = [
      { question: "Q1?", answer: "A1" },
      { question: "Q2?", answer: "A2" },
    ];
    tape.sources = [
      {
        type: "web",
        url: "https://example.com",
        priority: "required",
        kind: "prescription",
      },
    ];
    writeTape(path, tape);
    const { tape: out } = readTape(path);
    expect(out.parent).toBe("/tmp/parent");
    expect(out.jtbd_clarifications).toHaveLength(2);
    expect(out.sources[0]?.kind).toBe("prescription");
  });
});
