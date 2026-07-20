import { describe, expect, it } from "vitest";

import {
  expectedGoVersion,
  expectedShimVersion,
  matchesCanonicalVersion,
} from "../scripts/release-version-contract.mjs";

describe("release version contract", () => {
  it("accepts aligned package and installed-component versions", () => {
    const version = "1.2.3";

    expect(matchesCanonicalVersion(version, version)).toBe(true);
    expect(matchesCanonicalVersion(expectedShimVersion(version), expectedShimVersion(version))).toBe(
      true,
    );
    expect(matchesCanonicalVersion(expectedGoVersion(version), expectedGoVersion(version))).toBe(true);
  });

  it("rejects drift in packages, the launcher/core pair, and the Go binary", () => {
    const version = "1.2.3";

    expect(matchesCanonicalVersion(version, "1.2.4")).toBe(false);
    expect(
      matchesCanonicalVersion(
        expectedShimVersion(version),
        "liner 1.2.3 (tui)  ·  1.2.4 (core)",
      ),
    ).toBe(false);
    expect(matchesCanonicalVersion(expectedGoVersion(version), "liner-tui 1.2.4")).toBe(false);
  });
});
