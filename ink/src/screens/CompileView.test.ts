import { describe, expect, it } from "vitest";
import { formatCompileWarningLine } from "./CompileView.js";

describe("formatCompileWarningLine", () => {
  const warning = {
    severity: "warning" as const,
    url: "https://example.com/articles/very/long/path/with/many/segments",
    message:
      "paywall: HTTP 402 text/html body preview shows a subscription interstitial before the article body",
  };

  it("fits warning rows inside the compile result box at narrow widths", () => {
    const columns = 60;
    const line = formatCompileWarningLine(warning, columns);

    expect(line.length).toBeLessThanOrEqual(columns - 8);
    expect(line).toContain("https://example.c…");
    expect(line).toContain("paywall:");
  });

  it("uses more of the available line at wider widths", () => {
    const narrow = formatCompileWarningLine(warning, 60);
    const wide = formatCompileWarningLine(warning, 118);

    expect(wide.length).toBeGreaterThan(narrow.length);
    expect(wide.length).toBeLessThanOrEqual(110);
    expect(wide).toContain("subscription");
  });
});
