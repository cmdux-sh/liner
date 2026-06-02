// JTBD elicitation — generate 3-4 clarifying questions targeted at the
// user's just-typed JTBD, so the downstream curation has a sharper input.
//
// Runs as a side task at wizard time, not as a methodology phase. The agent
// answers in JSON; we accumulate streamed text and parse once the run
// completes. If anything goes wrong (no agent installed, agent crashed,
// response can't be parsed) the wizard falls back to the hardcoded questions
// below — they're the same canonical set the recommendation system uses when
// the agent path is unavailable. Better to ask predictable questions than
// to skip the step entirely.

import { runAgentTask } from "./runner.js";
import type { AgentDescriptor } from "./types.js";
import type { AgentEvent } from "./events.js";

/**
 * Canonical fallback questions when the agent path is unavailable or fails.
 * Designed to cover the four axes the recommendation in the plan called out:
 *   1. Scope splits (the "and"/"or" boundary problem)
 *   2. Output kinds expected
 *   3. Quality anchors the curator can name
 *   4. Audience / autonomy mode
 */
export const HARDCODED_QUESTIONS: readonly string[] = [
  "If your JTBD names multiple domains (e.g. X and Y), are they equally weighted, or is one primary?",
  "What specific kinds of output should this mixtape help produce — flag descriptions, errors, help text, longer docs, all of the above?",
  "Name 1–2 tools, projects, or authors whose work in this domain you admire. These become quality anchors for source discovery.",
  "Is this corpus for an AI assisting you interactively, or for an AI working autonomously?",
];

export type ElicitArgs = {
  jtbd: string;
  agent: AgentDescriptor | null;
  /** Skill bundle path. When null, falls back to hardcoded questions. */
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
 * Generate clarifying questions for a raw JTBD. Returns the agent's
 * questions on success, or the hardcoded baseline if anything prevents a
 * clean agent answer. Never throws — failure modes degrade to fallback.
 */
export async function elicitClarifyingQuestions(args: ElicitArgs): Promise<string[]> {
  if (!args.agent || !args.skillPath) {
    return [...HARDCODED_QUESTIONS];
  }

  const prompt = buildPrompt(args.jtbd);

  // Accumulate every text + tool-result event into one buffer; the response
  // will be in there somewhere. The agent's `summary.finalText` lands at the
  // end and is usually the cleanest source of the JSON.
  let finalText = "";
  const textChunks: string[] = [];

  const handle = runAgentTask({
    agent: args.agent,
    project: args.cwd,
    skillPath: args.skillPath,
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
    clearTimeout(timeout);
    if (timedOut || code !== 0) {
      return [...HARDCODED_QUESTIONS];
    }
    const body = finalText || textChunks.join("\n");
    const questions = parseQuestions(body);
    if (questions.length === 0) {
      return [...HARDCODED_QUESTIONS];
    }
    // Clamp to a sane range: at least 2, at most 6. The wizard step renders
    // them one at a time, so very long question lists fatigue the user.
    return questions.slice(0, 6);
  } catch {
    clearTimeout(timeout);
    return [...HARDCODED_QUESTIONS];
  }
}

function buildPrompt(jtbd: string): string {
  return `You are helping a curator sharpen a just-typed JTBD for an AI-curated mixtape.

# The JTBD
${jtbd}

# Your task

Generate 3-4 targeted questions whose answers will make the downstream curation produce a sharper corpus. Cover the following four axes — one question per axis, in order:

1. **Scope splits** the user named ("and"/"or" boundaries). If the JTBD names two domains, ask whether they're equally weighted.
2. **Output kinds** — what specific things will this mixtape help produce (e.g. flag descriptions, errors, designs, prose, etc.)? Force a concrete answer.
3. **Quality anchors** — name 1-2 tools/projects/authors whose work in this domain the curator admires. These anchor source discovery.
4. **Audience / autonomy** — is the corpus for an AI assisting interactively, or for an AI working autonomously?

If the JTBD doesn't have a scope split (axis 1), ask a different sharpening question instead — something about the JTBD's most ambiguous word or phrase.

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
