import { existsSync, lstatSync, readdirSync, realpathSync } from "node:fs";
import { join, relative } from "node:path";

export function fingerprintTargets(targets) {
  const rows = [];
  for (const target of targets) fingerprint(target, target, rows, new Set());
  return JSON.stringify(rows);
}

function fingerprint(base, path, rows, visitedDirectories) {
  if (!existsSync(path)) {
    rows.push([base, relative(base, path), "missing"]);
    return;
  }
  const stat = lstatSync(path);
  const label = relative(base, path);
  rows.push([base, label, stat.isDirectory() ? "dir" : stat.isSymbolicLink() ? "link" : "file", stat.size, stat.mtimeMs]);
  if (stat.isSymbolicLink()) {
    let canonical;
    try {
      canonical = realpathSync(path);
    } catch {
      rows.push([base, label, "broken-link"]);
      return;
    }
    rows.push([base, label, "link-target", canonical]);
    fingerprint(base, canonical, rows, visitedDirectories);
    return;
  }
  if (!stat.isDirectory()) return;
  const canonicalDirectory = realpathSync(path);
  if (visitedDirectories.has(canonicalDirectory)) {
    rows.push([base, label, "directory-cycle", canonicalDirectory]);
    return;
  }
  visitedDirectories.add(canonicalDirectory);
  for (const name of readdirSync(path).sort()) {
    fingerprint(base, join(path, name), rows, visitedDirectories);
  }
}
