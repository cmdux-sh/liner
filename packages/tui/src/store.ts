// Minimal global state via React Context. No Zustand dep needed.

import { createContext, useContext } from "react";
import type { Tape, ProjectSummary } from "./types.js";
import type { PhaseId } from "./phases.js";

export type Screen =
  | { kind: "splash" }
  | { kind: "agentSetup" }
  | { kind: "jsSetup" }
  | { kind: "browser" }
  | { kind: "newProject" }
  | { kind: "hub"; tape: Tape; folder: string }
  | { kind: "phaseRunner"; tape: Tape; folder: string; phaseId: PhaseId }
  | { kind: "gate0"; tape: Tape; folder: string }
  | { kind: "gate1"; tape: Tape; folder: string }
  | { kind: "gate2"; tape: Tape; folder: string }
  | { kind: "tapeDraft"; tape: Tape; folder: string }
  | { kind: "editor"; tape: Tape; folder: string }
  | {
      kind: "sourceModal";
      tape: Tape;
      folder: string;
      editingIndex: number | null;
    }
  | { kind: "compile"; tape: Tape; folder: string; showExisting?: boolean }
  | { kind: "manifest"; tape: Tape; folder: string }
  | { kind: "import" };

export type AppState = {
  screen: Screen;
  baseDir: string;
  projects: ProjectSummary[];
  notification: { kind: "info" | "error"; message: string } | null;
};

export type AppActions = {
  navigate: (screen: Screen) => void;
  /**
   * Return to the previous screen. Uses a history stack maintained by
   * `navigate`. If the stack is empty (the user is at the root), falls back
   * to the browser. Screens should prefer `back()` over a hard-coded
   * `navigate({ kind: "browser" })` for their esc handlers so that flows
   * launched from the splash (Import, New mixtape) return to the splash
   * instead of dumping the user on an unfamiliar screen.
   */
  back: () => void;
  refreshProjects: () => Promise<void>;
  setNotification: (n: AppState["notification"]) => void;
};

export const AppContext = createContext<(AppState & AppActions) | null>(null);

export function useApp(): AppState & AppActions {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("AppContext not provided");
  return ctx;
}
