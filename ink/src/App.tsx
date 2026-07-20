import React, { useCallback, useEffect, useRef, useState } from "react";
import { Box, Text } from "ink";
import { AppContext, type AppState, type Screen } from "./store.js";
import { MUTED, ERROR } from "./colors.js";
import { TapeBrowser } from "./screens/TapeBrowser.js";
import { TapeEditor } from "./screens/TapeEditor.js";
import { SourceModal } from "./screens/SourceModal.js";
import { SourcePrompt } from "./screens/SourcePrompt.js";
import { SourceInbox } from "./screens/SourceInbox.js";
import { CompileView } from "./screens/CompileView.js";
import { NewMixtapeWizard } from "./screens/NewMixtapeWizard.js";
import { ProjectHub } from "./screens/ProjectHub.js";
import { PhaseRunner } from "./screens/PhaseRunner.js";
import { FramingConfirmReview } from "./screens/FramingConfirmReview.js";
import { Gate1Review } from "./screens/Gate1Review.js";
import { Gate2Review } from "./screens/Gate2Review.js";
import { TapeDraftReview } from "./screens/TapeDraftReview.js";
import { ImportPrompt } from "./screens/ImportPrompt.js";
import { AgentSetup } from "./screens/AgentSetup.js";
import { JsSetup } from "./screens/JsSetup.js";
import { ProcessManifest } from "./screens/ProcessManifest.js";
import { BootSplash, shouldShowSplash } from "./screens/BootSplash.js";
import { configExists, readConfig } from "./config.js";
import { detectAgents } from "./agents/detect.js";
import { listProjects } from "./ipc.js";
import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";

const NOTIFICATION_MS = 3500;
/** Cap on the back stack — deeper history isn't useful and just costs memory. */
const HISTORY_LIMIT = 10;

export function App(): React.ReactElement {
  const baseDir = useRef<string>(resolveBaseDir()).current;
  // Decide what the user sees first. Four states, in priority order:
  //
  //   1. Agent setup — first launch on a machine with multiple agents
  //      installed AND no config file yet. Force a choice before the splash
  //      so the methodology phases don't hit the mid-phase picker.
  //   2. JS setup — one-time onboarding for Playwright/Chromium support.
  //   3. Splash — the normal greeting (cassette art + launch menu).
  //   4. Browser — fallback when stdout isn't a TTY or LINER_NO_SPLASH is
  //      set (CI runs, narrow terminals).
  //
  // Single-agent installs skip step 1: there's nothing to pick. Repeat runs
  // skip it too because config.yaml exists.
  const initialScreen: Screen = resolveInitialScreen();
  const [screen, setScreen] = useState<Screen>(initialScreen);
  const [history, setHistory] = useState<Screen[]>([]);
  const [projects, setProjects] = useState<AppState["projects"]>([]);
  const [notification, setNotification] = useState<AppState["notification"]>(null);
  const [bootError, setBootError] = useState<string | null>(null);

  // navigate(next) moves to `next` and maintains the back stack.
  //
  // Dedupe rule: if `next` is already somewhere in the back stack, we REWIND
  // to it (drop everything above it) instead of pushing a duplicate. This is
  // what stops esc loops: several screens hardcode `navigate({kind:"hub"})` /
  // `navigate({kind:"browser"})` for their "go back" action instead of
  // back(), which would otherwise push A→B→A→B forever (e.g. hub esc →
  // browser, browser esc → back to hub, repeat). Rewinding collapses the
  // cycle so esc always makes progress toward home (splash).
  //
  // Side effect: clears the terminal scrollback when the user moves between
  // meaningfully different surfaces. Done before setScreen so Ink's next
  // render lands on a fresh canvas.
  const navigate = useCallback((next: Screen) => {
    setScreen((current) => {
      if (process.stdout.isTTY && shouldClearOnNavigate(current, next)) {
        process.stdout.write("\x1B[2J\x1B[3J\x1B[H");
      }
      const nextKey = JSON.stringify(next);
      setHistory((h) => {
        const existingIdx = h.findIndex((s) => JSON.stringify(s) === nextKey);
        if (existingIdx !== -1) return h.slice(0, existingIdx);
        return [...h, current].slice(-HISTORY_LIMIT);
      });
      return next;
    });
  }, []);

  // back() pops the history stack and navigates to the prior screen. When
  // the stack is empty (root) it falls back to the splash (home). We pop
  // INSIDE setScreen so the two updates batch and Ink renders once.
  const back = useCallback(() => {
    setHistory((h) => {
      if (h.length === 0) {
        // Empty history = user is at the root. Land on home (splash), not
        // the browser — home is the terminal screen for esc, and esc on home
        // is a no-op (see BootSplash), so navigation stops there instead of
        // looping home ↔ browser.
        if (process.stdout.isTTY) process.stdout.write("\x1B[2J\x1B[3J\x1B[H");
        setScreen({ kind: "splash" });
        return h;
      }
      const prev = h[h.length - 1]!;
      if (process.stdout.isTTY) process.stdout.write("\x1B[2J\x1B[3J\x1B[H");
      setScreen(prev);
      return h.slice(0, -1);
    });
  }, []);

  const refreshProjects = useCallback(async () => {
    try {
      const result = await listProjects(baseDir);
      setProjects(result);
    } catch (e) {
      setBootError((e as Error).message);
    }
  }, [baseDir]);

  useEffect(() => {
    void refreshProjects();
  }, [refreshProjects]);

  useEffect(() => {
    if (!notification) return;
    const timer = setTimeout(() => setNotification(null), NOTIFICATION_MS);
    return () => clearTimeout(timer);
  }, [notification]);

  if (bootError) {
    return (
      <Box flexDirection="column">
        <Text color={ERROR}>{bootError}</Text>
        <Text color={MUTED}>Press Ctrl+C to exit.</Text>
      </Box>
    );
  }

  const ctx = {
    screen,
    baseDir,
    projects,
    notification,
    navigate,
    back,
    refreshProjects,
    setNotification,
  };

  return (
    <AppContext.Provider value={ctx}>
      <ScreenRoot screen={screen} />
    </AppContext.Provider>
  );
}

function ScreenRoot({ screen }: { screen: Screen }): React.ReactElement {
  switch (screen.kind) {
    case "splash":
      return <BootSplash />;
    case "agentSetup":
      return <AgentSetup />;
    case "jsSetup":
      return <JsSetup />;
    case "browser":
      return <TapeBrowser />;
    case "newProject":
      return <NewMixtapeWizard />;
    case "hub":
      return <ProjectHub tape={screen.tape} folder={screen.folder} />;
    case "phaseRunner":
      // `key` forces React to unmount and remount when phaseId changes —
      // PhaseRunner's auto-launch effect has empty deps and only fires on
      // initial mount, so without the key, navigating Phase 2 → Phase 4 (both
      // "phaseRunner" kind) leaves the component dormant: same DOM, no
      // agent launch. The remount is also what resets the tool list,
      // finalText, and stats from the previous phase.
      return (
        <PhaseRunner
          key={screen.phaseId}
          tape={screen.tape}
          folder={screen.folder}
          phaseId={screen.phaseId}
        />
      );
    case "gate0":
      return <FramingConfirmReview tape={screen.tape} folder={screen.folder} />;
    case "gate1":
      return <Gate1Review tape={screen.tape} folder={screen.folder} />;
    case "gate2":
      return <Gate2Review tape={screen.tape} folder={screen.folder} />;
    case "tapeDraft":
      return <TapeDraftReview tape={screen.tape} folder={screen.folder} />;
    case "import":
      return <ImportPrompt />;
    case "editor":
      return <TapeEditor tape={screen.tape} folder={screen.folder} />;
    case "sourcePrompt":
      return <SourcePrompt tape={screen.tape} folder={screen.folder} />;
    case "sourceInbox":
      return <SourceInbox tape={screen.tape} folder={screen.folder} />;
    case "sourceModal":
      return (
        <SourceModal
          tape={screen.tape}
          folder={screen.folder}
          editingIndex={screen.editingIndex}
        />
      );
    case "compile":
      return (
        <CompileView
          key={`${screen.folder}:${screen.showExisting ? "view" : "run"}`}
          tape={screen.tape}
          folder={screen.folder}
          showExisting={screen.showExisting}
        />
      );
    case "manifest":
      return <ProcessManifest tape={screen.tape} folder={screen.folder} />;
  }
}

/**
 * Pick the screen the user lands on at startup. See the inline doc in
 * `App()` for the priority order.
 */
function resolveInitialScreen(): Screen {
  // If no splash (non-TTY / narrow / opted-out), don't show setup either —
  // the user is probably in CI or scripting and just wants the browser.
  if (!shouldShowSplash()) return { kind: "browser" };
  // First-run agent setup only when we have a real choice to surface. One
  // agent (or zero) has nothing to pick, so go straight to the JS-rendering
  // setup prompt.
  if (!configExists()) {
    const installed = detectAgents();
    if (installed.length >= 2) return { kind: "agentSetup" };
    return { kind: "jsSetup" };
  }
  if (!readConfig().jsSetupPrompted) return { kind: "jsSetup" };
  return { kind: "splash" };
}

/**
 * Decide whether navigation `prev → next` should clear the terminal. We clear
 * on any kind change, and on phaseRunner re-entry with a different phaseId so
 * "entering Phase 3" feels like starting fresh, not stacking on top of Phase 2.
 */
function shouldClearOnNavigate(prev: Screen, next: Screen): boolean {
  if (prev.kind !== next.kind) return true;
  if (prev.kind === "phaseRunner" && next.kind === "phaseRunner") {
    return prev.phaseId !== next.phaseId;
  }
  return false;
}

function resolveBaseDir(): string {
  const override = process.env["LINER_DIR"];
  const dir = override || join(process.cwd(), "mixtapes");
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  return dir;
}
