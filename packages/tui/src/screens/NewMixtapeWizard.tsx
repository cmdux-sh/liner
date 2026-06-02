import React, { useEffect, useMemo, useState } from "react";
import { Box, Text, useInput } from "ink";
import Spinner from "ink-spinner";
import TextInput from "ink-text-input";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { useApp } from "../store.js";
import { runInit } from "../ipc.js";
import { readTape, slugify, writeTape, projectFolder } from "../yaml-io.js";
import { writeProgress } from "../progress.js";
import { detectAgents, resolveSkillPath } from "../agents/detect.js";
import { elicitClarifyingQuestions, HARDCODED_QUESTIONS } from "../agents/jtbd-clarify.js";
import { MUTED, CURRENT, ERROR, HEADING, HIGHLIGHT, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import type { JtbdClarification, Mode } from "../types.js";

type Step =
  | "slug"
  | "jtbd"
  | "jtbd-clarify"
  | "title"
  | "description"
  | "curator"
  | "review"
  | "creating";

type Draft = {
  // Mode stays in Draft + tape.yaml for forward/back-compat with existing
  // methodology-mode mixtapes, but new mixtapes always get "quick" — the
  // wizard no longer surfaces the choice. The mode field can come back as
  // a power-user setting once methodology has been validated end-to-end.
  mode: Mode;
  slug: string;
  jtbd: string;
  clarifications: JtbdClarification[];
  title: string;
  description: string;
  curator: string;
};

// 6 input steps: slug, jtbd, jtbd-clarify, title, description, curator.
const TOTAL_STEPS = 6;
const STEP_ORDER: Exclude<Step, "creating" | "review">[] = [
  "slug",
  "jtbd",
  "jtbd-clarify",
  "title",
  "description",
  "curator",
];

/**
 * State for the JTBD-clarify step. The wizard kicks off an agent call to
 * generate questions when this step is entered for the first time; the user
 * then answers each one sequentially. Esc during loading skips ahead with
 * empty clarifications; once questions are loaded, esc on a question goes
 * back to the jtbd step (and discards captured answers — they were specific
 * to that JTBD anyway).
 */
type ClarifyState =
  | { phase: "idle" }
  | { phase: "loading" }
  | { phase: "asking"; questions: string[]; answers: string[]; index: number }
  | { phase: "error"; message: string };

export function NewMixtapeWizard(): React.ReactElement {
  const app = useApp();
  const [step, setStep] = useState<Step>("slug");
  // Draft fields start empty — we don't pre-fill suggestions into draft any
  // more, because that left the previous-step text sitting in the new input.
  // Suggestions are surfaced via `suggestionFor()` as placeholders; on empty
  // submit we accept the suggestion as the value.
  const [draft, setDraft] = useState<Draft>({
    mode: "quick",
    slug: "",
    jtbd: "",
    clarifications: [],
    title: "",
    description: "",
    curator: "",
  });
  const [buffer, setBuffer] = useState("");
  const [error, setError] = useState<string | null>(null);
  // When the user jumps into a field from the review screen (via 1-6), we
  // record that here so the next goForward returns straight to review instead
  // of traversing the rest of the wizard. Reset after the bounce-back.
  const [returningToReview, setReturningToReview] = useState(false);
  // JTBD-clarify sub-state. Idle until the user enters the step; loading while
  // the agent generates questions; asking while the user answers them.
  const [clarifyState, setClarifyState] = useState<ClarifyState>({ phase: "idle" });

  const stepIndex = useMemo(() => {
    if (step === "review") return TOTAL_STEPS;
    if (step === "creating") return TOTAL_STEPS + 1;
    return STEP_ORDER.indexOf(step) + 1;
  }, [step]);

  // When the user enters the JTBD-clarify step for the first time, fire off
  // the agent call to generate questions. On success: store the questions in
  // clarifyState and the user starts answering. On any failure: the helper
  // returns the hardcoded baseline, so the user still gets a clarify pass.
  useEffect(() => {
    if (step !== "jtbd-clarify") return;
    if (clarifyState.phase !== "idle") return;
    let cancelled = false;
    setClarifyState({ phase: "loading" });
    const agents = detectAgents();
    const skillPath = resolveSkillPath();
    void elicitClarifyingQuestions({
      jtbd: draft.jtbd,
      agent: agents[0] ?? null,
      skillPath,
      cwd: app.baseDir,
    }).then((questions) => {
      if (cancelled) return;
      setClarifyState({
        phase: "asking",
        questions,
        answers: new Array<string>(questions.length).fill(""),
        index: 0,
      });
      setBuffer("");
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step]);

  function startEditing(target: Exclude<Step, "creating" | "review" | "jtbd-clarify">): void {
    setBuffer(draft[target as keyof Draft] as string);
    setError(null);
    setReturningToReview(true);
    setStep(target);
  }

  /**
   * Resolve a submitted value, falling back to the per-step suggestion when
   * the user submitted an empty buffer. Lets Enter "accept the default" while
   * a non-empty type-then-Enter overrides it cleanly.
   */
  function resolveSubmit(rawValue: string, currentStep: FieldStepName): string {
    const trimmed = rawValue.trim();
    if (trimmed) return trimmed;
    return suggestionFor(currentStep, draft).trim();
  }

  // Commit one answer in the clarify step and advance to the next question
  // (or to the title step when finished). Empty answers are kept — the user
  // can skip a question they don't have an answer for; the clarification is
  // still recorded so the agent can see "user skipped this one".
  function commitClarifyAnswer(value: string): void {
    if (clarifyState.phase !== "asking") return;
    const trimmed = value.trim();
    const nextAnswers = clarifyState.answers.slice();
    nextAnswers[clarifyState.index] = trimmed;
    const nextIndex = clarifyState.index + 1;
    if (nextIndex >= clarifyState.questions.length) {
      // Last question — fold answers into the draft as JtbdClarification[]
      // and advance to the title step.
      const clarifications: JtbdClarification[] = clarifyState.questions
        .map((q, i) => ({ question: q, answer: nextAnswers[i] ?? "" }))
        // Drop pairs the user skipped entirely — keeping them just adds
        // noise in tape.yaml and in the Phase 1 prompt.
        .filter((c) => c.answer.trim().length > 0);
      setDraft({ ...draft, clarifications });
      setClarifyState({ phase: "idle" });
      setBuffer("");
      goForward();
      return;
    }
    setClarifyState({ ...clarifyState, answers: nextAnswers, index: nextIndex });
    setBuffer("");
  }

  function commitField(value: string): void {
    setError(null);
    if (step === "slug") {
      const resolved = resolveSubmit(value, "slug") || "untitled";
      const cleaned = slugify(resolved);
      const folder = join(app.baseDir, cleaned);
      if (existsSync(folder)) {
        setError(`${cleaned} already exists — pick a different slug.`);
        return;
      }
      setDraft({ ...draft, slug: cleaned });
      goForward();
    } else if (step === "jtbd") {
      const trimmed = resolveSubmit(value, "jtbd");
      if (trimmed.length < 12) {
        setError("Push for a real sentence — what are you trying to do with the AI in this domain?");
        return;
      }
      setDraft({ ...draft, jtbd: trimmed });
      goForward();
    } else if (step === "title") {
      const resolved = resolveSubmit(value, "title");
      if (!resolved) {
        setError("Title is required.");
        return;
      }
      setDraft({ ...draft, title: resolved });
      goForward();
    } else if (step === "description") {
      const resolved = resolveSubmit(value, "description");
      if (!resolved) {
        setError("One-line description helps the AI consuming this mixtape.");
        return;
      }
      setDraft({ ...draft, description: resolved });
      goForward();
    } else if (step === "curator") {
      const resolved = resolveSubmit(value, "curator");
      if (!resolved) {
        setError("Curator name is required.");
        return;
      }
      setDraft({ ...draft, curator: resolved });
      setBuffer("");
      // Curator is the last input step; commit always lands on review,
      // whether we got here from review-edit or the normal forward path.
      setReturningToReview(false);
      setStep("review");
    }
  }

  function goForward(): void {
    // Bounce back to review if the user entered this step via [1-6] on the
    // review screen. The expectation when editing a single field is "fix it
    // and return," not "fix it and walk through every later step again."
    if (returningToReview) {
      setReturningToReview(false);
      setBuffer("");
      setStep("review");
      return;
    }
    const i = STEP_ORDER.indexOf(step as never);
    const next = STEP_ORDER[i + 1];
    // Each forward step starts with a clean buffer; the suggestion shows as
    // a placeholder (TextInput's placeholder prop).
    setBuffer("");
    if (next) setStep(next);
    else setStep("review");
  }

  function goBack(): void {
    if (step === "review") {
      setStep("curator");
      setBuffer(draft.curator);
      return;
    }
    // If the user was editing this field from the review screen, esc returns
    // them straight to review — pairs with the goForward bounce-back.
    if (returningToReview) {
      setReturningToReview(false);
      setBuffer("");
      setStep("review");
      return;
    }
    // Leaving jtbd-clarify in either sub-phase resets clarify state and
    // returns to the JTBD field. Captured answers are dropped — they were
    // generated for the previous JTBD and won't apply to a revised one.
    if (step === "jtbd-clarify") {
      setClarifyState({ phase: "idle" });
      setBuffer(draft.jtbd);
      setStep("jtbd");
      return;
    }
    const i = STEP_ORDER.indexOf(step as never);
    if (i <= 0) {
      // First step → leaving the wizard entirely. back() returns to whatever
      // launched it (splash, browser, …) rather than always landing on browser.
      app.back();
      return;
    }
    const prev = STEP_ORDER[i - 1];
    if (prev) {
      // Going back to a previously-answered step shows the existing answer so
      // the user can tweak rather than retype. The clarify step has its own
      // sub-state so it's not buffer-backed; reset to idle and let the effect
      // re-trigger the agent call on the next forward pass.
      setStep(prev);
      if (prev === "jtbd-clarify") {
        setClarifyState({ phase: "idle" });
        setBuffer("");
      } else {
        setBuffer(draft[prev as keyof Draft] as string);
      }
    }
  }

  async function confirm(): Promise<void> {
    setStep("creating");
    const folder = join(app.baseDir, draft.slug);
    try {
      await runInit(folder);
      // Read the just-scaffolded tape, apply the wizard's fields, write back.
      const { tape, doc } = readTape(join(folder, "tape.yaml"));
      tape.title = draft.title;
      tape.description = draft.description;
      tape.curator = draft.curator;
      tape.mode = draft.mode;
      tape.jtbd = draft.jtbd;
      // Persist JTBD clarifications onto the tape so Phase 1 (framing) can
      // read them. Empty array → null so unaffected tapes stay clean.
      tape.jtbd_clarifications =
        draft.clarifications.length > 0 ? draft.clarifications : null;
      if (draft.mode === "methodology") tape.methodology_version = "2.0";
      writeTape(join(folder, "tape.yaml"), tape, doc);

      // Pre-fill the JTBD into working/01 so the curator doesn't have to copy
      // the same sentence over. The knowledge-map section stays as TODO so the
      // next agent (or the curator) has work to do.
      prefillJtbdIntoWorking01(folder, draft.jtbd);

      // Explicitly initialize the progress cursor to step 0. Without this,
      // the hub falls back to `migrateProgress` which inspects artifact
      // contents — and our JTBD pre-fill above strips the JTBD-section TODO
      // from working/01, which migrateProgress would otherwise misread as
      // "Phase 1 done." Writing step=0 here means the user lands on Framing
      // (the actual current phase) instead of jumping to Gate 0.
      writeProgress(folder, {
        step: 0,
        lastTouched: new Date().toISOString(),
      });

      app.setNotification({ kind: "info", message: `Created ${folder}` });
      await app.refreshProjects();
      app.navigate({ kind: "hub", tape, folder });
    } catch (e) {
      setError((e as Error).message);
      setStep("review");
    }
  }

  const isFieldStep =
    step === "slug" ||
    step === "jtbd" ||
    step === "title" ||
    step === "description" ||
    step === "curator" ||
    (step === "jtbd-clarify" && clarifyState.phase === "asking");

  // Field-editing steps: TextInput owns letters/numbers. Only escape leaks here.
  useInput(
    (_input, key) => {
      if (key.escape) goBack();
    },
    { isActive: isFieldStep },
  );

  // While loading the clarifying questions, the wizard is between states —
  // no TextInput is mounted yet. Esc cancels the wait and bounces back to
  // the JTBD step; the in-flight agent call gets ignored via the effect's
  // cleanup flag (the agent process self-exits, audit log captures it).
  useInput(
    (_input, key) => {
      if (key.escape) goBack();
    },
    { isActive: step === "jtbd-clarify" && clarifyState.phase === "loading" },
  );

  // Review step: full keybinding palette.
  useInput(
    (input, key) => {
      if (step === "review") {
        if (key.return || input === "y") {
          void confirm();
        } else if (key.escape || input === "n") {
          goBack();
        } else if (input === "1") startEditing("slug");
        else if (input === "2") startEditing("jtbd");
        else if (input === "3") startEditing("title");
        else if (input === "4") startEditing("description");
        else if (input === "5") startEditing("curator");
      }
    },
    { isActive: step === "review" },
  );

  return (
    <Box flexDirection="column">
      <Header
        title="new mixtape"
        subtitle={progressLine(step, stepIndex, draft)}
      />

      <StepChip step={step} stepIndex={stepIndex} draft={draft} />

      <Transcript draft={draft} step={step} />

      {(step === "slug" ||
        step === "jtbd" ||
        step === "title" ||
        step === "description" ||
        step === "curator") && (
        <FieldStep
          step={step}
          buffer={buffer}
          suggestion={suggestionFor(step, draft)}
          onChange={setBuffer}
          onSubmit={(v) => commitField(v)}
        />
      )}

      {step === "jtbd-clarify" ? (
        <ClarifyStep
          state={clarifyState}
          buffer={buffer}
          onChange={setBuffer}
          onSubmit={commitClarifyAnswer}
        />
      ) : null}

      {step === "review" ? <ReviewStep draft={draft} /> : null}

      {step === "creating" ? (
        <Box marginTop={1}>
          <Text color={WARNING}>Creating folder…</Text>
        </Box>
      ) : null}

      {error ? (
        <Box marginTop={1}>
          <Text color={ERROR}>{error}</Text>
        </Box>
      ) : null}

      <Box marginTop={1}>
        <KeyHints hints={hintsFor(step)} />
      </Box>
    </Box>
  );
}

function progressLine(step: Step, stepIndex: number, _draft: Draft): string {
  if (step === "review") return `step ${TOTAL_STEPS}/${TOTAL_STEPS} · review and confirm`;
  if (step === "creating") return "creating…";
  const label = stepHeading(step);
  return `step ${stepIndex} of ${TOTAL_STEPS} · ${label}`;
}

function stepHeading(step: Step): string {
  switch (step) {
    case "slug":
      return "folder name";
    case "jtbd":
      return "job-to-be-done";
    case "jtbd-clarify":
      return "sharpen the JTBD";
    case "title":
      return "title";
    case "description":
      return "one-line description";
    case "curator":
      return "your name";
    default:
      return "";
  }
}

function Transcript({ draft, step }: { draft: Draft; step: Step }): React.ReactElement | null {
  const entries: Array<{ key: string; label: string; value: string }> = [];
  if (afterStep(step, "slug") && draft.slug)
    entries.push({ key: "slug", label: "folder", value: draft.slug });
  if (afterStep(step, "jtbd") && draft.jtbd)
    entries.push({ key: "jtbd", label: "jtbd", value: truncate(draft.jtbd, 80) });
  if (afterStep(step, "jtbd-clarify") && draft.clarifications.length > 0)
    entries.push({
      key: "clarifications",
      label: "clarifications",
      value: `${draft.clarifications.length} captured`,
    });
  if (afterStep(step, "title") && draft.title)
    entries.push({ key: "title", label: "title", value: draft.title });
  if (afterStep(step, "description") && draft.description)
    entries.push({ key: "description", label: "description", value: truncate(draft.description, 80) });
  if (afterStep(step, "curator") && draft.curator)
    entries.push({ key: "curator", label: "curator", value: draft.curator });

  if (entries.length === 0) return null;
  return (
    <Box flexDirection="column" marginBottom={1}>
      {entries.map((e) => (
        <Box key={e.key}>
          {/* 13 chars wide: longest label is "description" (11) + 2 chars
              breathing room so the value doesn't butt up against it. */}
          <Text color={MUTED}>{e.label.padEnd(13, " ")}</Text>
          <Text>{e.value}</Text>
        </Box>
      ))}
    </Box>
  );
}

function StepChip({
  step,
  stepIndex,
  draft: _draft,
}: {
  step: Step;
  stepIndex: number;
  draft: Draft;
}): React.ReactElement | null {
  if (step === "creating") return null;
  const total = TOTAL_STEPS;
  // On review, every input step is in the past — no dot is "current".
  const filled = step === "review" ? total + 1 : stepIndex;
  const dots: React.ReactElement[] = [];
  for (let i = 1; i <= total; i++) {
    const isPast = i < filled;
    const isCurrent = i === filled;
    dots.push(
      <Text
        key={i}
        color={isPast ? SUCCESS : isCurrent ? STRUCTURE : MUTED}
        bold={isCurrent}
      >
        {isPast ? "●" : isCurrent ? "●" : "○"}
        {i < total ? " " : ""}
      </Text>,
    );
  }
  return (
    <Box marginBottom={1}>
      <Text backgroundColor={HEADING} color={HIGHLIGHT} bold>
        {` step ${step === "review" ? total : stepIndex} / ${total} `}
      </Text>
      <Text>{"  "}</Text>
      {dots}
    </Box>
  );
}

function afterStep(current: Step, target: Exclude<Step, "creating" | "review">): boolean {
  if (current === "review" || current === "creating") return true;
  const ci = STEP_ORDER.indexOf(current as never);
  const ti = STEP_ORDER.indexOf(target);
  return ci > ti;
}

type FieldStepName = Exclude<Step, "review" | "creating" | "jtbd-clarify">;

/**
 * Render the JTBD-clarify step. Has three visible variants:
 *
 *   loading → spinner + "asking the agent for clarifying questions" line.
 *             Esc cancels and goes back to JTBD.
 *
 *   asking  → shows one question at a time with a TextInput. Each answer
 *             advances to the next question; the last one advances to title.
 *             Enter on an empty input skips the question (and the entry is
 *             dropped from the final clarifications list).
 *
 *   error   → fallback for the helper's promise rejecting; in practice the
 *             helper never throws (it falls back to HARDCODED_QUESTIONS),
 *             but we keep the variant for completeness.
 */
function ClarifyStep({
  state,
  buffer,
  onChange,
  onSubmit,
}: {
  state: ClarifyState;
  buffer: string;
  onChange: (v: string) => void;
  onSubmit: (v: string) => void;
}): React.ReactElement {
  if (state.phase === "idle" || state.phase === "loading") {
    return (
      <Box flexDirection="column" marginBottom={1}>
        <Text color={HEADING} bold>Sharpening the JTBD</Text>
        <Box marginTop={1}>
          <Text color={STRUCTURE}>
            <Spinner type="dots" />{" "}
          </Text>
          <Text>asking the agent for clarifying questions…</Text>
        </Box>
        <Box marginTop={1}>
          <Text color={MUTED}>
            The corpus is only as good as the input. A few targeted questions
            here usually beat one vague JTBD downstream.
          </Text>
        </Box>
        <Box marginTop={1}>
          <Text color={MUTED}>
            Press <Text color={STRUCTURE}>esc</Text> to skip this step.
          </Text>
        </Box>
      </Box>
    );
  }

  if (state.phase === "error") {
    return (
      <Box flexDirection="column" marginBottom={1}>
        <Text color={ERROR}>Could not generate questions: {state.message}</Text>
        <Text color={MUTED}>Press <Text color={STRUCTURE}>esc</Text> to skip back to the JTBD step.</Text>
      </Box>
    );
  }

  const question = state.questions[state.index] ?? "";
  const total = state.questions.length;
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Box>
        <Text color={HEADING} bold>{question}</Text>
      </Box>
      <Box marginTop={1}>
        <Text color={MUTED}>
          question {state.index + 1} of {total} · enter to advance ·
          empty + enter skips this one
        </Text>
      </Box>
      <Box marginTop={1}>
        <Text color={CURRENT}>› </Text>
        <TextInput
          value={buffer}
          onChange={onChange}
          onSubmit={onSubmit}
          placeholder="your answer (or leave blank to skip)"
        />
      </Box>
    </Box>
  );
}

function FieldStep({
  step,
  buffer,
  suggestion,
  onChange,
  onSubmit,
}: {
  step: FieldStepName;
  buffer: string;
  /** Dynamic per-step default. If the user submits empty, this becomes the value. */
  suggestion: string;
  onChange: (v: string) => void;
  onSubmit: (v: string) => void;
}): React.ReactElement {
  // The TextInput placeholder is always a short static example. The dynamic
  // JTBD-derived suggestion (often long-form) gets its own labeled card below
  // so it doesn't visually bleed into the input field.
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text color={HEADING} bold>{questionFor(step)}</Text>
      {helpFor(step) ? (
        <Box marginTop={1}>
          <Text color={MUTED}>{helpFor(step)}</Text>
        </Box>
      ) : null}
      <Box marginTop={1}>
        <Text color={STRUCTURE}>› </Text>
        <TextInput
          value={buffer}
          onChange={onChange}
          onSubmit={onSubmit}
          placeholder={placeholderFor(step)}
        />
      </Box>
      {suggestion && !buffer ? (
        <Box marginTop={1} flexDirection="column">
          <Box>
            <Text color={MUTED}>suggestion: </Text>
            <Text>{truncate(suggestion, 72)}</Text>
          </Box>
          <Text color={MUTED}>
            press <Text color={STRUCTURE}>enter</Text> to accept, or type your own.
          </Text>
        </Box>
      ) : null}
      {step === "slug" && buffer ? (
        <Box marginTop={1}>
          <Text color={MUTED}>→ will create folder: </Text>
          <Text>{slugify(buffer)}</Text>
        </Box>
      ) : null}
    </Box>
  );
}

function ReviewStep({ draft }: { draft: Draft }): React.ReactElement {
  const rows: Array<{ key: string; label: string; value: string }> = [
    { key: "1", label: "folder", value: draft.slug },
    { key: "2", label: "jtbd", value: truncate(draft.jtbd, 80) },
    { key: "3", label: "title", value: draft.title },
    { key: "4", label: "description", value: truncate(draft.description, 80) },
    { key: "5", label: "curator", value: draft.curator },
  ];
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Text color={HEADING} bold>Review:</Text>
      <Box marginTop={1} flexDirection="column">
        {rows.map((r) => (
          <Box key={r.key}>
            <Text color={STRUCTURE}>{`[${r.key}] `}</Text>
            <Text color={MUTED}>{`${r.label}:`.padEnd(14, " ")}</Text>
            <Text>{r.value}</Text>
          </Box>
        ))}
      </Box>
      <Box marginTop={1}>
        <Text>Press </Text>
        <Text color={STRUCTURE}>enter</Text>
        <Text> to create, </Text>
        <Text color={STRUCTURE}>1–6</Text>
        <Text> to edit a field, </Text>
        <Text color={STRUCTURE}>esc</Text>
        <Text> to go back.</Text>
      </Box>
    </Box>
  );
}

function questionFor(step: FieldStepName): string {
  switch (step) {
    case "slug":
      return "What's the folder name?";
    case "jtbd":
      return "What's the job-to-be-done?";
    case "title":
      return "What should the title be?";
    case "description":
      return "Give it a one-line description.";
    case "curator":
      return "Who are you (curator name)?";
  }
}

function helpFor(step: FieldStepName): string {
  switch (step) {
    case "slug":
      return "Lowercase letters, numbers, hyphens. Becomes the project folder under your mixtapes directory.";
    case "jtbd":
      return 'A single Job Story — not a topic. Form: "When [circumstance], I want [motivation], so I can [outcome]." All three slots required, no tool/vendor names, one job per slot. "Mobile design" is a topic; a tight JTBD scopes the circumstance, motivation, and outcome.';
    case "title":
      return "Short human-readable name. Shown in browsers and at the top of MIXTAPE.md.";
    case "description":
      return "One sentence. Helps the AI consuming this context know what the mixtape is for.";
    case "curator":
      return "Your name or handle. Goes into tape.yaml.";
  }
}

function placeholderFor(step: FieldStepName): string {
  switch (step) {
    case "slug":
      return "mobile-design-foundations";
    case "jtbd":
      return "When I'm designing... I want to... so I can...";
    case "title":
      return "Mobile design foundations";
    case "description":
      return "Designing iOS apps that feel native and distinctive";
    case "curator":
      return "Your name";
  }
}

function hintsFor(step: Step): Array<{ key: string; label: string }> {
  if (step === "review")
    return [
      { key: "enter", label: "create" },
      { key: "1–5", label: "edit field" },
      { key: "esc", label: "go back" },
    ];
  if (step === "creating") return [];
  return [
    { key: "enter", label: "next" },
    { key: "esc", label: "go back" },
  ];
}

/**
 * Per-step suggestion — the value Enter accepts when the user submits an
 * empty buffer. Stays empty for steps that have no sensible default (slug,
 * jtbd). For title/description, derives from the user's JTBD answer.
 */
function suggestionFor(step: FieldStepName, draft: Draft): string {
  switch (step) {
    case "slug":
      return "";
    case "jtbd":
      return "";
    case "title":
      return suggestTitleFromJtbd(draft.jtbd);
    case "description":
      return suggestDescriptionFromJtbd(draft.jtbd);
    case "curator":
      return defaultCurator();
  }
}

function suggestTitleFromJtbd(jtbd: string): string {
  // Strip leading helper-verb phrases so the title reads as a noun phrase, not
  // an instruction. Then take only the first clause (up to the first sentence
  // boundary or comma) so a long, rambling JTBD still produces a tight title.
  const stripped = jtbd
    .replace(/^(help me|help us|i need (?:help )?(?:to )?|i want to|i'm trying to|i am trying to|let me|allow me to)\s+/i, "")
    .replace(/\.+\s*$/, "")
    .trim();
  if (!stripped) return "";
  const firstClause = stripped.split(/[,.;]/)[0]?.trim() ?? stripped;
  // If the first clause is still long, it's not a title — bail rather than
  // surface a bad suggestion.
  if (firstClause.length > 60) return "";
  return firstClause[0]!.toUpperCase() + firstClause.slice(1);
}

function suggestDescriptionFromJtbd(jtbd: string): string {
  // Take the first sentence only; descriptions are one line.
  const firstSentence = jtbd.split(/(?<=[.!?])\s+/)[0]?.trim() ?? "";
  const cleaned = firstSentence.replace(/\.+\s*$/, "").trim();
  if (!cleaned) return "";
  if (cleaned.length > 100) return "";
  return cleaned[0]!.toUpperCase() + cleaned.slice(1) + ".";
}

function defaultCurator(): string {
  return (
    process.env["LINER_CURATOR"] ||
    process.env["USER"] ||
    process.env["USERNAME"] ||
    ""
  );
}

function prefillJtbdIntoWorking01(folder: string, jtbd: string): void {
  const path = join(folder, "working", "01-jtbd-and-knowledge-map.md");
  if (!existsSync(path)) return;
  const text = readFileSync(path, "utf8");
  // Replace the TODO with the curator's JTBD; leave the knowledge-map section
  // as-is so the agent or curator still has explicit work to do.
  const updated = text.replace(
    /TODO — a single specific sentence\. Not the topic — the use case\. Examples:[\s\S]*?- "[^"]*"\s*\n- "[^"]*"\s*\n/,
    `${jtbd}\n\n_Set via the new-mixtape wizard. Revise here if your understanding sharpens during research._\n\n`,
  );
  if (updated !== text) {
    const _ = projectFolder; // keep import live for tree-shaking sanity
    void _;
    writeFileSync(path, updated, "utf8");
  }
}

function truncate(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}
