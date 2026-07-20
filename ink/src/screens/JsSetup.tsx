import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import Spinner from "ink-spinner";
import { Header } from "../components/Header.js";
import { KeyHints } from "../components/KeyHints.js";
import { LabeledBox } from "../components/LabeledBox.js";
import { CURRENT, ERROR, MUTED, STRUCTURE, SUCCESS, WARNING } from "../colors.js";
import { writeConfig } from "../config.js";
import { runSetupJs } from "../ipc.js";
import { useApp } from "../store.js";

type SetupState = "idle" | "running" | "failed";

export function JsSetup(): React.ReactElement {
  const app = useApp();
  const [state, setState] = useState<SetupState>("idle");
  const [message, setMessage] = useState<string | null>(null);

  function goHome(notification?: string): void {
    try {
      writeConfig({ jsSetupPrompted: true });
      if (notification) app.setNotification({ kind: "info", message: notification });
      app.navigate({ kind: "splash" });
    } catch (e) {
      setState("failed");
      setMessage(`Could not write config: ${(e as Error).message}`);
    }
  }

  function install(): void {
    if (state === "running") return;
    setState("running");
    setMessage("Downloading Playwright Chromium. This can take a few minutes on first run.");
    void runSetupJs()
      .then(() => {
        goHome("JS rendering is ready for pages that need a browser.");
      })
      .catch((e: Error) => {
        setState("failed");
        setMessage(`JS rendering setup failed: ${e.message}`);
      });
  }

  useInput((input, key) => {
    if (state === "running") return;
    if (key.return || input === "y") install();
    else if (input === "s" || input === "b" || key.escape) {
      goHome("Skipped JS rendering setup. You can run liner setup-js later.");
    }
  });

  return (
    <Box flexDirection="column">
      <Header title="set up JS rendering" subtitle="first-run setup · optional but useful" />

      <Box marginBottom={1}>
        <LabeledBox label="browser-backed sources" color={STRUCTURE}>
          <Text>
            {
              "Liner can fetch normal web pages right away. Some sources, especially app docs and interactive sites, only reveal useful text after JavaScript runs in a browser."
            }
          </Text>
          <Box marginTop={1} flexDirection="column">
            <Text color={MUTED}>
              {
                "Installing JS rendering downloads Playwright's headless Chromium browser, about 150MB on first run. After that, compiles can recover from \"requires JavaScript\" pages instead of keeping tiny stubs."
              }
            </Text>
            <Text color={MUTED}>
              {
                "You can skip this now. If a future compile needs it, Liner will prompt again from the compile results screen."
              }
            </Text>
          </Box>
        </LabeledBox>
      </Box>

      <Box marginBottom={1}>
        <LabeledBox label="recommendation" color={state === "failed" ? ERROR : WARNING}>
          <Text color={CURRENT} bold>
            Press enter to install JS rendering now.
          </Text>
          <Box marginTop={1}>
            <Text color={MUTED}>
              {
                "Liner will run "
              }
              <Text color={STRUCTURE}>liner setup-js --yes</Text>
              {
                ". The command is safe to repeat; when Chromium is already present, it exits quickly."
              }
            </Text>
          </Box>
        </LabeledBox>
      </Box>

      {message ? (
        <Box marginBottom={1}>
          <Text color={state === "failed" ? ERROR : state === "running" ? WARNING : SUCCESS}>
            {state === "running" ? <Spinner type="dots" /> : null}
            {state === "running" ? " " : ""}
            {message}
          </Text>
        </Box>
      ) : null}

      <KeyHints
        hints={[
          { key: "enter/y", label: "install" },
          { key: "s", label: "skip" },
          { key: "esc", label: "skip" },
        ]}
      />
    </Box>
  );
}
