export type AgentId = "claude" | "codex";

export type AgentDescriptor = {
  id: AgentId;
  name: string;
  bin: string;
  /** Exact configuration directory used for auth and agent preferences. */
  configHome?: string;
};

export type AgentRunHandle = {
  cancel: () => void;
  done: Promise<AgentRunResult>;
};

export type AgentRunResult = {
  code: number | null;
  stderr: string;
  logPath?: string;
};
