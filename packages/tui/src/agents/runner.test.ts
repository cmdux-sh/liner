import { describe, it, expect } from "vitest";
import { chmodSync, mkdtempSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  buildArgs,
  evaluationArtifactClosurePrompt,
  evaluationClosureIdleMs,
  findLatestCodexSessionId,
  looksLikeModelRejection,
  resumePromptForTask,
  runPhaseWithAgent,
  shouldSurfaceAgentStderr,
} from "./runner.js";
import type { AgentEvent } from "./events.js";
import type { Tape } from "../types.js";

describe("buildArgs", () => {
  it("uses Codex fresh-run workspace options before the stdin prompt", () => {
    expect(buildArgs("codex", "/tmp/project", "/repo/docs/curation-skill", false, "gpt-5")).toEqual([
      "exec",
      "--cd",
      "/tmp/project",
      "--model",
      "gpt-5",
      "--add-dir",
      "/repo/docs/curation-skill",
      "--skip-git-repo-check",
      "-s",
      "workspace-write",
      "--json",
      "-",
    ]);
  });

  it("uses Codex resume options with a recorded session id", () => {
    expect(buildArgs("codex", "/tmp/project", "/repo/docs/curation-skill", true, "gpt-5", "thread-123")).toEqual([
      "exec",
      "resume",
      "--skip-git-repo-check",
      "--json",
      "thread-123",
      "-",
    ]);
  });

  it("keeps --last only as an explicit low-level fallback", () => {
    expect(buildArgs("codex", "/tmp/project", "/repo/docs/curation-skill", true, "gpt-5")).toEqual([
      "exec",
      "resume",
      "--skip-git-repo-check",
      "--json",
      "--last",
      "-",
    ]);
  });

  it("scopes Claude fresh-run tools to the methodology surface", () => {
    expect(buildArgs("claude", "/tmp/project", "/repo/docs/curation-skill", false, "sonnet")).toEqual([
      "-p",
      "--output-format",
      "stream-json",
      "--verbose",
      "--add-dir",
      "/repo/docs/curation-skill",
      "--strict-mcp-config",
      "--dangerously-skip-permissions",
      "--tools",
      "Read,Write,Edit,Glob,Grep,WebFetch",
      "--allowedTools",
      "Read Write Edit Glob Grep WebFetch",
      "--disallowedTools",
      "Bash Task WebSearch ToolSearch",
      "--disable-slash-commands",
      "--model",
      "sonnet",
    ]);
  });

  it("keeps Claude resume runs hardened without changing the session model", () => {
    expect(buildArgs("claude", "/tmp/project", "/repo/docs/curation-skill", true, "sonnet")).toEqual([
      "-p",
      "--output-format",
      "stream-json",
      "--verbose",
      "--add-dir",
      "/repo/docs/curation-skill",
      "--strict-mcp-config",
      "--dangerously-skip-permissions",
      "--tools",
      "Read,Write,Edit,Glob,Grep,WebFetch",
      "--allowedTools",
      "Read Write Edit Glob Grep WebFetch",
      "--disallowedTools",
      "Bash Task WebSearch ToolSearch",
      "--disable-slash-commands",
      "--continue",
    ]);
  });
});

describe("findLatestCodexSessionId", () => {
  it("reads the latest non-resume Codex thread id for a phase", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-runner-"));
    const dir = join(project, ".liner-runs", "candidates");
    mkdirSync(dir, { recursive: true });
    writeFileSync(
      join(dir, "2026-06-14T08-00-00.jsonl"),
      [
        JSON.stringify({ type: "_liner_meta", agent: "codex", resume: false, taskLabel: "candidates" }),
        JSON.stringify({ type: "thread.started", thread_id: "thread-original" }),
        "",
      ].join("\n"),
    );
    writeFileSync(
      join(dir, "2026-06-14T09-00-00.jsonl"),
      [
        JSON.stringify({ type: "_liner_meta", agent: "codex", resume: true, taskLabel: "candidates" }),
        JSON.stringify({ type: "thread.started", thread_id: "thread-resume" }),
        "",
      ].join("\n"),
    );

    expect(findLatestCodexSessionId(project, "candidates")).toBe("thread-original");
  });

  it("ignores non-Codex logs", () => {
    const project = mkdtempSync(join(tmpdir(), "liner-runner-"));
    const dir = join(project, ".liner-runs", "quality");
    mkdirSync(dir, { recursive: true });
    writeFileSync(
      join(dir, "2026-06-14T08-00-00.jsonl"),
      [
        JSON.stringify({ type: "_liner_meta", agent: "claude", resume: false, taskLabel: "quality" }),
        JSON.stringify({ type: "thread.started", thread_id: "thread-claude" }),
        "",
      ].join("\n"),
    );

    expect(findLatestCodexSessionId(project, "quality")).toBeUndefined();
  });
});

describe("resumePromptForTask", () => {
  it("asks resumed Candidate discovery to stop search loops and write the long-list", () => {
    const prompt = resumePromptForTask("candidates");

    expect(prompt).toContain("Phase 2 reminder");
    expect(prompt).toContain("Workspace discipline");
    expect(prompt).toContain("Do not run `git diff`");
    expect(prompt).toContain("Ruby 2.6");
    expect(prompt).toContain("stop any open-ended web/search loop");
    expect(prompt).toContain("Write working/02-candidate-longlist.md now");
    expect(prompt).toContain("Do not run more than one final targeted search");
  });

  it("adds the bounded fallback reminder when resuming evaluation", () => {
    const prompt = resumePromptForTask("evaluation");

    expect(prompt).toContain("Continue from where you left off");
    expect(prompt).toContain("do not get stuck chasing unavailable content");
    expect(prompt).toContain("Every candidate still needs a keep/trim/drop decision");
    expect(prompt).toContain("working/evaluation-decisions/");
    expect(prompt).toContain("fetch_status");
    expect(prompt).toContain("content_quality");
    expect(prompt).toContain("content-specific evidence");
  });

  it("asks resumed quality runs to stop search loops and write the report", () => {
    const prompt = resumePromptForTask("quality");

    expect(prompt).toContain("Phase 5 reminder");
    expect(prompt).toContain("stop any open-ended web/search loop");
    expect(prompt).toContain("assign missing jtbd_fit and source kinds");
    expect(prompt).toContain("repair weak curator notes");
    expect(prompt).toContain("working/04-quality-checks.md");
    expect(prompt).toContain("Test 0 core-action fit");
    expect(prompt).toContain("working/05-operating-fit-audit.md");
    expect(prompt).toContain("status: improvement_recommended");
  });

  it("keeps other resumed tasks short", () => {
    expect(resumePromptForTask("synthesis")).not.toContain("Phase 4 reminder");
  });
});

describe("evaluation artifact closure", () => {
  it("asks resumed Evaluation to write the YAML without more fetching", () => {
    const prompt = evaluationArtifactClosurePrompt();

    expect(prompt).toContain("artifact closure only");
    expect(prompt).toContain("Workspace discipline");
    expect(prompt).toContain("prefer Node.js or Python");
    expect(prompt).toContain("Do not fetch, search, or read external sources");
    expect(prompt).toContain("working/evaluation-decisions/");
    expect(prompt).toContain("fetch_status");
    expect(prompt).toContain("Do not keep sources from URL/title/search snippets/model memory alone");
  });

  it("uses a configurable idle timeout", () => {
    expect(evaluationClosureIdleMs({ LINER_EVALUATION_CLOSURE_IDLE_MS: "25" })).toBe(25);
    expect(evaluationClosureIdleMs({ LINER_EVALUATION_CLOSURE_IDLE_MS: "0" })).toBe(0);
    expect(evaluationClosureIdleMs({ LINER_EVALUATION_CLOSURE_IDLE_MS: "-1" })).toBe(120_000);
    expect(evaluationClosureIdleMs({ LINER_EVALUATION_CLOSURE_IDLE_MS: "nope" })).toBe(120_000);
  });

  it("stops an idle Evaluation turn and resumes once with a write-only closure prompt", async () => {
    const previousIdleMs = process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"];
    process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"] = "1200";
    try {
      const project = mkdtempSync(join(tmpdir(), "liner-eval-closure-"));
      const skillPath = join(project, "skill");
      mkdirSync(join(project, "working"), { recursive: true });
      mkdirSync(skillPath, { recursive: true });
      writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
      writeFileSync(
        join(project, "working/02-candidate-longlist.md"),
        [
          "## foundations",
          "",
          "- https://example.com/a — Example A — Useful primary candidate.",
          "",
        ].join("\n"),
      );
      writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");

      const agentBin = join(project, "fake-agent.mjs");
      writeFileSync(
        agentBin,
        [
          "#!/usr/bin/env node",
          "import { appendFileSync, mkdirSync, writeFileSync } from 'node:fs';",
          "const chunks = [];",
          "process.stdin.setEncoding('utf8');",
          "process.stdin.on('data', (chunk) => chunks.push(chunk));",
          "process.stdin.on('end', () => {",
          "  const input = chunks.join('');",
          "  const resume = process.argv.includes('--continue');",
          "  appendFileSync('invocations.jsonl', JSON.stringify({ resume, input }) + '\\n');",
          "  if (!resume) {",
          "    console.log(JSON.stringify({ type: 'system', subtype: 'init', session_id: 'fake-session', model: 'fake' }));",
          "    console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'I am ready to write the evaluation artifact.' }] } }));",
          "    setInterval(() => {}, 1000);",
          "    return;",
          "  }",
          "  if (!input.includes('Do not fetch, search, or read external sources')) {",
          "    console.error('missing closure prompt');",
          "    process.exit(4);",
          "  }",
          "  mkdirSync('working', { recursive: true });",
          "  writeFileSync('working/03-evaluation.yaml', `candidates:\\n  - url: https://example.com/a\\n    title: Example A\\n    decision: kept\\n    rating: 4\\n    jtbd_fit: direct\\n    fetch_status: readable\\n    content_quality: high\\n    evidence:\\n      - The source shows the core idea with a concrete example.\\n      - It names the limitation that keeps the fixture bounded.\\n    section: foundations\\n    rationale: Strong enough for the JTBD.\\n    note: |\\n      Role: Anchor source. Value: Shows the core idea. Limitations: Small fixture.\\n`);",
          "  console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'Done with Phase 4.' }] } }));",
          "  console.log(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, duration_ms: 1, num_turns: 1, result: 'Done with Phase 4.', total_cost_usd: 0 }));",
          "});",
          "",
        ].join("\n"),
      );
      chmodSync(agentBin, 0o755);

      const events: AgentEvent[] = [];
      const tape: Tape = {
        title: "Terminal craft",
        description: "",
        version: 1,
        curator: "Arturo",
        sources: [],
        mode: "quick",
        jtbd: "Help a design engineer build polished terminal interfaces.",
      };
      const handle = runPhaseWithAgent({
        agent: { id: "claude", name: "Fake Claude", bin: agentBin },
        phaseId: "evaluation",
        project,
        skillPath,
        tape,
        onEvent: (event) => events.push(event),
      });

      const result = await handle.done;

      expect(result.code).toBe(0);
      expect(readFileSync(join(project, "working/03-evaluation.yaml"), "utf8")).toContain("decision: kept");
      const invocations = readFileSync(join(project, "invocations.jsonl"), "utf8")
        .trim()
        .split("\n")
        .map((line) => JSON.parse(line) as { resume: boolean; input: string });
      expect(invocations).toHaveLength(2);
      expect(invocations[0]?.resume).toBe(false);
      expect(invocations[1]?.resume).toBe(true);
      expect(invocations[1]?.input).toContain("working/evaluation-decisions/");
      expect(events.some((event) => event.kind === "raw" && event.text.includes("write-only artifact closure prompt"))).toBe(true);
    } finally {
      if (previousIdleMs == null) delete process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"];
      else process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"] = previousIdleMs;
    }
  }, 7000);

  it("assembles Evaluation fragments after a successful agent exit", async () => {
    const project = mkdtempSync(join(tmpdir(), "liner-eval-fragment-run-"));
    const skillPath = join(project, "skill");
    mkdirSync(join(project, "working/evaluation-decisions"), { recursive: true });
    mkdirSync(skillPath, { recursive: true });
    writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
    writeFileSync(
      join(project, "working/02-candidate-longlist.md"),
      [
        "## Foundations",
        "- **Example A** - https://example.com/a",
        "- **Example B** - https://example.com/b",
      ].join("\n"),
    );
    writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");

    const agentBin = join(project, "fake-fragment-agent.mjs");
    writeFileSync(
      agentBin,
      [
        "#!/usr/bin/env node",
        "import { mkdirSync, writeFileSync } from 'node:fs';",
        "process.stdin.resume();",
        "process.stdin.on('end', () => {",
        "  mkdirSync('working/evaluation-decisions', { recursive: true });",
        "  writeFileSync('working/evaluation-decisions/01-foundations.yaml', `candidates:\\n  - url: https://example.com/a\\n    decision: kept\\n    rating: 5\\n    jtbd_fit: direct\\n    fetch_status: readable\\n    content_quality: high\\n    evidence:\\n      - The source demonstrates the action with a concrete example.\\n      - It names a limitation that keeps the fixture scoped.\\n    rationale: Strong fit.\\n    note: |\\n      Role: Anchor. Value: Useful model. Limitations: Fixture.\\n  - url: https://example.com/b\\n    decision: dropped\\n    rationale: Duplicate.\\n`);",
        "  console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'Fragments written.' }] } }));",
        "  console.log(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, duration_ms: 1, num_turns: 1, result: 'Done with Phase 4.', total_cost_usd: 0 }));",
        "});",
        "",
      ].join("\n"),
    );
    chmodSync(agentBin, 0o755);

    const events: AgentEvent[] = [];
    const tape: Tape = {
      title: "Terminal craft",
      description: "",
      version: 1,
      curator: "Arturo",
      sources: [],
      mode: "quick",
      jtbd: "Help a design engineer build polished terminal interfaces.",
    };
    const result = await runPhaseWithAgent({
      agent: { id: "claude", name: "Fake Claude", bin: agentBin },
      phaseId: "evaluation",
      project,
      skillPath,
      tape,
      onEvent: (event) => events.push(event),
    }).done;

    expect(result.code).toBe(0);
    const output = readFileSync(join(project, "working/03-evaluation.yaml"), "utf8");
    expect(output).toContain("https://example.com/a");
    expect(output).toContain("https://example.com/b");
    expect(events.some((event) => event.kind === "raw" && event.text.includes("Assembled working/03-evaluation.yaml"))).toBe(true);
  });

  it("runs fresh Evaluation as section-scoped chunks", async () => {
    const project = mkdtempSync(join(tmpdir(), "liner-eval-sectioned-"));
    const skillPath = join(project, "skill");
    mkdirSync(join(project, "working/evaluation-decisions"), { recursive: true });
    mkdirSync(skillPath, { recursive: true });
    writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
    writeFileSync(
      join(project, "working/02-candidate-longlist.md"),
      [
        "## Foundations",
        "- **Example A** - https://example.com/a",
        "## Craft",
        "- **Example B** - https://example.com/b",
      ].join("\n"),
    );
    writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");

    const agentBin = join(project, "fake-section-agent.mjs");
    writeFileSync(
      agentBin,
      [
        "#!/usr/bin/env node",
        "import { appendFileSync, mkdirSync, writeFileSync } from 'node:fs';",
        "const chunks = [];",
        "process.stdin.setEncoding('utf8');",
        "process.stdin.on('data', (chunk) => chunks.push(chunk));",
        "process.stdin.on('end', () => {",
        "  const input = chunks.join('');",
        "  const fragment = input.match(/`(working\\/evaluation-decisions\\/[^`]+)`/)?.[1];",
        "  const url = input.includes('https://example.com/a') ? 'https://example.com/a' : 'https://example.com/b';",
        "  appendFileSync('section-invocations.jsonl', JSON.stringify({ fragment, url }) + '\\n');",
        "  mkdirSync('working/evaluation-decisions', { recursive: true });",
        "  writeFileSync(fragment, `candidates:\\n  - url: ${url}\\n    decision: dropped\\n    rationale: Fixture drop.\\n`);",
        "  console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'Fragment written.' }] } }));",
        "  console.log(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, duration_ms: 1, num_turns: 1, result: 'Done with Phase 4.', total_cost_usd: 0 }));",
        "});",
      ].join("\n"),
    );
    chmodSync(agentBin, 0o755);

    const tape: Tape = {
      title: "Terminal craft",
      description: "",
      version: 1,
      curator: "Arturo",
      sources: [],
      mode: "quick",
      jtbd: "Help a design engineer build polished terminal interfaces.",
    };
    const events: AgentEvent[] = [];

    const result = await runPhaseWithAgent({
      agent: { id: "claude", name: "Fake Claude", bin: agentBin },
      phaseId: "evaluation",
      project,
      skillPath,
      tape,
      onEvent: (event) => events.push(event),
    }).done;

    expect(result.code).toBe(0);
    const invocations = readFileSync(join(project, "section-invocations.jsonl"), "utf8").trim().split("\n");
    expect(invocations).toHaveLength(2);
    expect(readFileSync(join(project, "working/03-evaluation.yaml"), "utf8")).toContain("https://example.com/b");
    expect(events.some((event) => event.kind === "raw" && event.text.includes("Evaluation chunk 1/2"))).toBe(true);
    expect(events.some((event) => event.kind === "raw" && event.text.includes("Evaluation chunk 2/2"))).toBe(true);
  });

  it("resumes section-scoped Evaluation by skipping complete fragments", async () => {
    const project = mkdtempSync(join(tmpdir(), "liner-eval-section-resume-"));
    const skillPath = join(project, "skill");
    mkdirSync(join(project, "working/evaluation-decisions"), { recursive: true });
    mkdirSync(skillPath, { recursive: true });
    writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
    writeFileSync(
      join(project, "working/02-candidate-longlist.md"),
      [
        "## Foundations",
        "- **Example A** - https://example.com/a",
        "## Craft",
        "- **Example B** - https://example.com/b",
      ].join("\n"),
    );
    writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");
    writeFileSync(
      join(project, "working/evaluation-decisions/01-foundations.yaml"),
      [
        "candidates:",
        "  - url: https://example.com/a",
        "    decision: kept",
        "    rating: 5",
        "    jtbd_fit: direct",
        "    fetch_status: readable",
        "    content_quality: high",
        "    evidence:",
        "      - The source demonstrates the action with a concrete example.",
        "      - It names a limitation that keeps the fixture scoped.",
        "    note: |",
        "      Role: Anchor. Value: Strong fit. Limitations: Fixture.",
      ].join("\n"),
    );

    const agentBin = join(project, "fake-section-resume-agent.mjs");
    writeFileSync(
      agentBin,
      [
        "#!/usr/bin/env node",
        "import { appendFileSync, mkdirSync, writeFileSync } from 'node:fs';",
        "const chunks = [];",
        "process.stdin.setEncoding('utf8');",
        "process.stdin.on('data', (chunk) => chunks.push(chunk));",
        "process.stdin.on('end', () => {",
        "  const input = chunks.join('');",
        "  const fragment = input.match(/`(working\\/evaluation-decisions\\/[^`]+)`/)?.[1];",
        "  appendFileSync('section-invocations.jsonl', JSON.stringify({ fragment, resume: process.argv.includes('--continue'), input }) + '\\n');",
        "  mkdirSync('working/evaluation-decisions', { recursive: true });",
        "  writeFileSync(fragment, `candidates:\\n  - url: https://example.com/b\\n    decision: dropped\\n    rationale: Fixture drop.\\n`);",
        "  console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'Fragment written.' }] } }));",
        "  console.log(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, duration_ms: 1, num_turns: 1, result: 'Done with Phase 4.', total_cost_usd: 0 }));",
        "});",
      ].join("\n"),
    );
    chmodSync(agentBin, 0o755);

    const tape: Tape = {
      title: "Terminal craft",
      description: "",
      version: 1,
      curator: "Arturo",
      sources: [],
      mode: "quick",
      jtbd: "Help a design engineer build polished terminal interfaces.",
    };
    const events: AgentEvent[] = [];

    const result = await runPhaseWithAgent({
      agent: { id: "claude", name: "Fake Claude", bin: agentBin },
      phaseId: "evaluation",
      project,
      skillPath,
      tape,
      resume: true,
      onEvent: (event) => events.push(event),
    }).done;

    expect(result.code).toBe(0);
    const invocations = readFileSync(join(project, "section-invocations.jsonl"), "utf8")
      .trim()
      .split("\n")
      .map((line) => JSON.parse(line) as { fragment: string; resume: boolean; input: string });
    expect(invocations).toHaveLength(1);
    expect(invocations[0]?.fragment).toBe("working/evaluation-decisions/02-craft.yaml");
    expect(invocations[0]?.resume).toBe(false);
    expect(invocations[0]?.input).toContain("https://example.com/b");
    expect(invocations[0]?.input).not.toContain("https://example.com/a");
    expect(events.some((event) => event.kind === "raw" && event.text.includes("Evaluation chunk 1/2 already complete"))).toBe(true);
    expect(readFileSync(join(project, "working/03-evaluation.yaml"), "utf8")).toContain("https://example.com/b");
  });

  it("keeps legacy Evaluation resume when no sectioned state exists", async () => {
    const project = mkdtempSync(join(tmpdir(), "liner-eval-legacy-resume-"));
    const skillPath = join(project, "skill");
    mkdirSync(join(project, "working"), { recursive: true });
    mkdirSync(skillPath, { recursive: true });
    writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
    writeFileSync(
      join(project, "working/02-candidate-longlist.md"),
      ["## Foundations", "- **Example A** - https://example.com/a"].join("\n"),
    );
    writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");

    const agentBin = join(project, "fake-legacy-resume-agent.mjs");
    writeFileSync(
      agentBin,
      [
        "#!/usr/bin/env node",
        "import { appendFileSync, writeFileSync } from 'node:fs';",
        "const chunks = [];",
        "process.stdin.setEncoding('utf8');",
        "process.stdin.on('data', (chunk) => chunks.push(chunk));",
        "process.stdin.on('end', () => {",
        "  const input = chunks.join('');",
        "  appendFileSync('legacy-invocations.jsonl', JSON.stringify({ resume: process.argv.includes('--continue'), input }) + '\\n');",
        "  writeFileSync('working/03-evaluation.yaml', `candidates:\\n  - url: https://example.com/a\\n    decision: dropped\\n    rationale: Fixture drop.\\n`);",
        "  console.log(JSON.stringify({ type: 'assistant', message: { content: [{ type: 'text', text: 'Done with Phase 4.' }] } }));",
        "  console.log(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, duration_ms: 1, num_turns: 1, result: 'Done with Phase 4.', total_cost_usd: 0 }));",
        "});",
      ].join("\n"),
    );
    chmodSync(agentBin, 0o755);

    const tape: Tape = {
      title: "Terminal craft",
      description: "",
      version: 1,
      curator: "Arturo",
      sources: [],
      mode: "quick",
      jtbd: "Help a design engineer build polished terminal interfaces.",
    };

    const result = await runPhaseWithAgent({
      agent: { id: "claude", name: "Fake Claude", bin: agentBin },
      phaseId: "evaluation",
      project,
      skillPath,
      tape,
      resume: true,
      onEvent: () => {},
    }).done;

    expect(result.code).toBe(0);
    const invocations = readFileSync(join(project, "legacy-invocations.jsonl"), "utf8")
      .trim()
      .split("\n")
      .map((line) => JSON.parse(line) as { resume: boolean; input: string });
    expect(invocations).toHaveLength(1);
    expect(invocations[0]?.resume).toBe(true);
    expect(invocations[0]?.input).toContain("Continue from where you left off");
  });

  it("fails cleanly when the write-only closure pass also stalls", async () => {
    const previousIdleMs = process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"];
    process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"] = "100";
    try {
      const project = mkdtempSync(join(tmpdir(), "liner-eval-closure-fail-"));
      const skillPath = join(project, "skill");
      mkdirSync(join(project, "working"), { recursive: true });
      mkdirSync(skillPath, { recursive: true });
      writeFileSync(join(skillPath, "SKILL.md"), "# Fake skill\n");
      writeFileSync(
        join(project, "working/02-candidate-longlist.md"),
        ["## foundations", "- https://example.com/a — Example A"].join("\n"),
      );
      writeFileSync(join(project, "working/03-evaluation.yaml"), "candidates: []\n");

      const agentBin = join(project, "fake-stalling-agent.mjs");
      writeFileSync(
        agentBin,
        [
          "#!/usr/bin/env node",
          "const chunks = [];",
          "process.stdin.setEncoding('utf8');",
          "process.stdin.on('data', (chunk) => chunks.push(chunk));",
          "process.stdin.on('end', () => {",
          "  const resume = process.argv.includes('--continue');",
          "  if (!resume) {",
          "    console.log(JSON.stringify({ type: 'system', subtype: 'init', session_id: 'fake-session', model: 'fake' }));",
          "  }",
          "  setInterval(() => {}, 1000);",
          "});",
          "",
        ].join("\n"),
      );
      chmodSync(agentBin, 0o755);

      const events: AgentEvent[] = [];
      const tape: Tape = {
        title: "Terminal craft",
        description: "",
        version: 1,
        curator: "Arturo",
        sources: [],
        mode: "quick",
        jtbd: "Help a design engineer build polished terminal interfaces.",
      };

      const result = await runPhaseWithAgent({
        agent: { id: "claude", name: "Fake Claude", bin: agentBin },
        phaseId: "evaluation",
        project,
        skillPath,
        tape,
        onEvent: (event) => events.push(event),
      }).done;

      expect(result.code).toBe(1);
      expect(result.stderr).toContain("Evaluation artifact closure timed out");
      expect(events.some((event) => event.kind === "raw" && event.text.includes("closure timed out"))).toBe(true);
    } finally {
      if (previousIdleMs == null) delete process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"];
      else process.env["LINER_EVALUATION_CLOSURE_IDLE_MS"] = previousIdleMs;
    }
  }, 5000);
});

describe("looksLikeModelRejection", () => {
  it("matches common unknown-model phrasings", () => {
    expect(looksLikeModelRejection("Error: unknown model 'gpt-5-mini'", "gpt-5-mini")).toBe(true);
    expect(looksLikeModelRejection("invalid model: sonnet", "sonnet")).toBe(true);
    expect(looksLikeModelRejection("model not found", "whatever")).toBe(true);
    expect(looksLikeModelRejection('{"error":"model_not_found"}', "gpt-5-mini")).toBe(true);
  });

  it("matches when the model id is echoed next to an error token", () => {
    expect(looksLikeModelRejection("API error: gpt-5-mini is unavailable", "gpt-5-mini")).toBe(true);
  });

  it("does not fire on unrelated / auth failures", () => {
    expect(looksLikeModelRejection("", "gpt-5-mini")).toBe(false);
    expect(
      looksLikeModelRejection("Authentication failed: please run `claude login`", "sonnet"),
    ).toBe(false);
    expect(looksLikeModelRejection("rate limit exceeded", "sonnet")).toBe(false);
  });

  it("is case-insensitive", () => {
    expect(looksLikeModelRejection("UNKNOWN MODEL", "x")).toBe(true);
  });
});

describe("shouldSurfaceAgentStderr", () => {
  it("hides repetitive cosmetic Codex startup warnings", () => {
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "2026-06-14T05:48:42Z WARN codex_core_plugins::manifest: ignoring interface.defaultPrompt[0]: prompt must be at most 128 characters",
      ),
    ).toBe(false);
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "2026-06-14T05:48:45Z WARN codex_core_skills::loader: ignoring interface.icon_small: icon path with '..' must resolve under plugin assets/",
      ),
    ).toBe(false);
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "2026-06-14T05:48:45Z WARN codex_core_skills::loader: ignoring interface.icon_large: icon path with '..' must resolve under plugin assets/",
      ),
    ).toBe(false);
  });

  it("hides noisy global Codex skill and connector stderr", () => {
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "ERROR codex_core::session::session: failed to load skill /path/SKILL.md: invalid YAML",
      ),
    ).toBe(false);
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "ERROR rmcp::transport::worker: worker quit with fatal: Transport channel closed, when Client HTTP request (HttpRequest)",
      ),
    ).toBe(false);
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "ERROR rmcp::transport::worker: worker quit with fatal: Transport channel closed, when Auth (AuthorizationRequired)",
      ),
    ).toBe(false);
  });

  it("hides noisy Codex connector/router stderr", () => {
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "WARN rmcp::transport::auth: Token refresh not possible, re-authorization required.",
      ),
    ).toBe(false);
    expect(
      shouldSurfaceAgentStderr("codex", "WARN connector OAuth refresh failed for optional MCP server"),
    ).toBe(false);
    expect(shouldSurfaceAgentStderr("codex", "router: write_stdin failed: stdin is closed")).toBe(false);
  });

  it("keeps Codex auth errors visible", () => {
    expect(
      shouldSurfaceAgentStderr(
        "codex",
        "ERROR codex auth failed: run `codex login`",
      ),
    ).toBe(true);
  });

  it("keeps Claude stderr visible and drops blank lines", () => {
    expect(shouldSurfaceAgentStderr("claude", "Authentication failed")).toBe(true);
    expect(shouldSurfaceAgentStderr("claude", "  ")).toBe(false);
  });
});
