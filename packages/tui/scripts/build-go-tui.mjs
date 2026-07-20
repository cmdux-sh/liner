#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");
const repoRoot = resolve(packageDir, "..", "..");
const goTuiDir = join(repoRoot, "packages", "go-tui");
const output = join(packageDir, "bin", process.platform === "win32" ? "liner-tui.exe" : "liner-tui");
const version = readFileSync(join(repoRoot, "VERSION"), "utf8").trim();

if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)) {
  console.error(`Invalid canonical Liner version: ${JSON.stringify(version)}`);
  process.exit(1);
}

const result = spawnSync(
  process.platform === "win32" ? "go.exe" : "go",
  ["build", "-trimpath", "-ldflags", `-X main.version=${version}`, "-o", output, "./cmd/liner-tui"],
  { cwd: goTuiDir, stdio: "inherit" },
);

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
