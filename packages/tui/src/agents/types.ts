export type AgentId = "claude" | "codex";

export type AgentDescriptor = {
  id: AgentId;
  name: string;
  bin: string;
};

export type AgentRunHandle = {
  cancel: () => void;
  done: Promise<{ code: number | null; stderr: string }>;
};
