export function matchesCanonicalVersion(expected, actual) {
  return actual === expected;
}

export function expectedShimVersion(version) {
  return `liner ${version} (tui)  ·  ${version} (core)`;
}

export function expectedGoVersion(version) {
  return `liner-tui ${version}`;
}
