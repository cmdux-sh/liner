#!/usr/bin/env node
// Copy the curation skill bundle from docs/curation-skill/ into the TUI package
// so it ships inside the published npm tarball.
//
// npm forbids `..` paths in `files`, so the bundle has to physically live
// inside the package directory at publish time. This script is invoked by
// `prepack`. The copied directory is gitignored; docs/curation-skill/ is the
// tracked source.

import { existsSync, mkdirSync, readdirSync, statSync, copyFileSync, rmSync } from "node:fs";
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

function copyRecursive(from, to) {
  mkdirSync(to, { recursive: true });
  for (const entry of readdirSync(from)) {
    const fromPath = join(from, entry);
    const toPath = join(to, entry);
    const s = statSync(fromPath);
    if (s.isDirectory()) {
      copyRecursive(fromPath, toPath);
    } else if (s.isFile()) {
      copyFileSync(fromPath, toPath);
    }
  }
}

copyRecursive(src, dest);
console.log(`copy-skill-bundle: copied ${src} → ${dest}`);
