// Capability elicitation — generate 3-4 clarifying questions targeted at what
// the user wants this Liner to help a future AI agent do.
//
// Runs as a side task at wizard time, not as a methodology phase. The agent
// answers in JSON; we accumulate streamed text and parse once the run
// completes. If the agent fails, the caller should surface the failure and let
// the user retry; Liner should not hide the problem behind canned questions.

import { runAgentTask } from "./runner.js";
import type { AgentDescriptor } from "./types.js";
import type { AgentEvent } from "./events.js";

export type ElicitArgs = {
  jtbd: string;
  agent: AgentDescriptor | null;
  /** Skill bundle path. Required for agent-backed question generation. */
  skillPath: string | null;
  /**
   * Working directory the agent runs under. Doesn't need to be a project
   * folder; baseDir works. The audit log lands under
   * `<cwd>/.liner-runs/jtbd-clarify/`.
   */
  cwd: string;
  /** Per-run timeout in ms. Default 60s. */
  timeoutMs?: number;
};

/**
 * Generate clarifying questions for a raw capability goal. Returns the agent's
 * questions on success. Throws after retry when no agent is available, the
 * agent crashes or times out, or the answer cannot be parsed.
 */
export async function elicitClarifyingQuestions(args: ElicitArgs): Promise<string[]> {
  if (!args.agent || !args.skillPath) {
    throw new Error("Clarifying questions require an available Claude Code or Codex agent and skill bundle");
  }

  const prompt = buildClarifyingQuestionsPrompt(args.jtbd);
  const failures: string[] = [];
  for (let attempt = 1; attempt <= 2; attempt++) {
    try {
      return await runClarifyAttempt(args, prompt);
    } catch (error) {
      failures.push(error instanceof Error ? error.message : String(error));
    }
  }
  throw new Error(`Could not generate clarification questions after retry: ${failures.join("; retry: ")}`);
}

async function runClarifyAttempt(args: ElicitArgs, prompt: string): Promise<string[]> {
  // Accumulate every text + tool-result event into one buffer; the response
  // will be in there somewhere. The agent's `summary.finalText` lands at the
  // end and is usually the cleanest source of the JSON.
  let finalText = "";
  const textChunks: string[] = [];

  const handle = runAgentTask({
    agent: args.agent!,
    project: args.cwd,
    skillPath: args.skillPath!,
    prompt,
    taskLabel: "jtbd-clarify",
    onEvent: (event: AgentEvent) => {
      if (event.kind === "text") {
        textChunks.push(event.text);
      } else if (event.kind === "summary" && event.finalText) {
        finalText = event.finalText;
      }
    },
  });

  const timeoutMs = args.timeoutMs ?? 60_000;
  let timedOut = false;
  const timeout = setTimeout(() => {
    timedOut = true;
    handle.cancel();
  }, timeoutMs);

  try {
    const { code } = await handle.done;
    if (timedOut) {
      throw new Error(`${args.agent!.name} did not return clarification questions before timeout`);
    }
    if (code !== 0) {
      throw new Error(`${args.agent!.name} exited with code ${code}`);
    }
    const body = finalText || textChunks.join("\n");
    const questions = parseQuestions(body);
    if (questions.length === 0) {
      throw new Error(`${args.agent!.name} did not return a JSON array of questions`);
    }
    // Clamp to a sane range: at least 2, at most 6. The wizard step renders
    // them one at a time, so very long question lists fatigue the user.
    return questions.slice(0, 6);
  } finally {
    clearTimeout(timeout);
  }
}

export function buildClarifyingQuestionsPrompt(jtbd: string): string {
  return `You are helping a curator sharpen a plain-language capability goal for a Liner project.

# The capability goal
${jtbd}

# Your task

Generate 3-4 targeted questions whose answers will help Liner build a hyper-specific, source-grounded resource for a future AI agent. The user should not have to name research lanes, source categories, or formal JTBD syntax. Ask only what is needed to infer those.

Ask questions that are specific to this capability. Do not use a generic checklist. Prefer questions that clarify:

- what future AI sessions should produce, decide, critique, translate, or prevent
- the narrow niche or situation where the resource should be excellent
- domain constraints, risk boundaries, regulated contexts, or things the agent must avoid
- quality anchors only when the user already hinted at examples, people, products, or styles they care about
- whether the future agent should ask follow-up questions at runtime or act autonomously from the request and corpus

If the goal involves images, moodboards, references, examples, inspiration, style, art direction, visual language, or translating one medium/domain into another output, include at least one question that clarifies the runtime inputs and output contract. Ask what kinds of references the future agent may receive and what the caller needs back (for example: observations, interpretation, carry-forward rules, implementation vocabulary, avoid-list, or clarification questions). This is not asking the user to name research sources; it is clarifying the job the corpus must support.

If the goal names multiple domains, ask how they relate in the capability. Do not ask the user "what sources should we gather" or "what should Liner extract"; Liner infers that from the capability.

# Output format (strict)

Respond with ONLY a JSON array of strings, each string being one question. No prose before or after. No markdown code fences. Example:

["First question?", "Second question?", "Third question?"]

Do not use any tools. Do not write to disk. Just output the JSON array.`;
}

/**
 * Extract a JSON array of question strings from the agent's response body.
 * Permissive — accepts the array nested in a code fence, surrounded by
 * prose, or as a top-level value. Returns an empty array on any failure.
 */
function parseQuestions(body: string): string[] {
  if (!body) return [];
  // Strip common code-fence wrappers first.
  const fenceMatch = body.match(/```(?:json)?\s*([\s\S]*?)```/);
  const candidate = fenceMatch ? fenceMatch[1]! : body;
  // Find the first [ ... ] that parses cleanly as an array.
  const arrayMatch = candidate.match(/\[\s*(?:[\s\S]*?)\s*\]/);
  const json = arrayMatch ? arrayMatch[0] : candidate.trim();
  let parsed: unknown;
  try {
    parsed = JSON.parse(json);
  } catch {
    return [];
  }
  if (!Array.isArray(parsed)) return [];
  const out: string[] = [];
  for (const item of parsed) {
    if (typeof item === "string" && item.trim()) {
      out.push(item.trim());
    } else if (item && typeof item === "object" && "question" in item) {
      // Accept {question: "..."} shape too — some agents like the structure.
      const q = (item as Record<string, unknown>)["question"];
      if (typeof q === "string" && q.trim()) out.push(q.trim());
    }
  }
  return out;
}
