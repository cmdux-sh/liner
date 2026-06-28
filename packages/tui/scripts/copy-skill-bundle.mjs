#!/usr/bin/env node
// Copy the curation skill bundle from docs/curation-skill/ into the TUI package
// so it ships inside the published npm tarball.
//
// npm forbids `..` paths in `files`, so the bundle has to physically live
// inside the package directory at publish time. This script is invoked by
// `prepack`. The copied directory is gitignored; docs/curation-skill/ is the
// tracked source.

import { existsSync, mkdirSync, copyFileSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(here, "..");
const repoRoot = resolve(packageDir, "..", "..");

const src = join(repoRoot, "docs", "curation-skill");
const dest = join(packageDir, "cli-update-docs");

if (!existsSync(src)) {
  console.error(`copy-skill-bundle: source not found at ${src}`);
  process.exit(1);
}

if (existsSync(dest)) rmSync(dest, { recursive: true, force: true });

const shippedFiles = [
  "README.md",
  "SKILL.md",
  "source-finding-tactics.md",
  "source-quality-hierarchy.md",
  "curator-notes.md",
  "quality-check-tests.md",
  "synthesis-guidance.md",
];

mkdirSync(dest, { recursive: true });
for (const file of shippedFiles) {
  const fromPath = join(src, file);
  if (!existsSync(fromPath)) {
    console.error(`copy-skill-bundle: missing required bundle file ${fromPath}`);
    process.exit(1);
  }
  copyFileSync(fromPath, join(dest, file));
}

console.error(`copy-skill-bundle: copied ${shippedFiles.length} files from ${src} → ${dest}`);
