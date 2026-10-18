# Liner website performance and accessibility audit

Date: 2026-10-17. Audit owner: performance/accessibility worker; implementation owner: parent agent.

## Scope and evidence boundaries

- Checkout: `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site`.
- Starting revision: `f7dc02e119952aa017077d670b392e74a3c4a498`. Baseline findings below precede the parent's implementation. Retests cover the uncommitted working changes observed during this audit, not a published revision.
- Current checkout preview: http://liner-site-copy-20261016-site-1ce1ccb23419.localhost:8765/ . Parent verified Watcher ownership, matching checkout, no stale build and no leftover servers before testing.
- Production baseline: https://www.liner.sh/ and https://www.liner.sh/docs/install/ . Production measurements describe the deployed baseline; they do not prove the updated working copy's production performance.
- Read the website product/design context, all ten docs source routes, landing/changelog source, shared layout/header/commands/file trees, font declarations and motion/WebGL code.
- Tools: Chrome DevTools MCP Lighthouse, performance traces and insights, accessibility snapshots, keyboard actions, viewport/reduced-motion emulation, computed layout and instrumented WebGL draw counts. No dependencies installed and no servers started or stopped by this worker.
- Skills applied: `chrome-devtools`, `a11y-debugging`, `debug-optimize-lcp`, `jakub-interfaces`, `better-accessibility`, `good-css`.

## Performance measurements

Single observations, not field percentiles or a statistically controlled benchmark. No CrUX field data was returned. Lighthouse's MCP audit excludes performance; the timing values below come from traces.

| Surface and environment | LCP | CLS | Relevant observations |
| --- | ---: | ---: | --- |
| Baseline local homepage, unthrottled, cache bypass | 152 ms | 0.00 | Local development server; not a production speed claim. TTFB 9 ms, render delay 143 ms. |
| Deployed homepage, 390 × 844 CSS pixels, DPR 3, Slow 4G, CPU 4×, cache bypass | 1,520 ms | 0.00 | Hero paragraph was LCP. TTFB 28 ms, render delay 1,492 ms. |
| Deployed install docs, same mobile throttling | 1,548 ms | 0.0164 | Rounded trace summary displayed CLS 0.02. Shift cluster affected command snippet and following content; tool did not establish its cause. |

Actionable baseline work:

1. **Continuous rendering:** WebGL drew 60 frames in a one-second observation with reduced motion enabled, and another 60 with the canvas over 8,000 pixels above the viewport. Resize/layout measurements also ran every frame. The parent added preference, visibility and pause guards, a 30 fps ceiling, a 1.5 DPR ceiling and resize-only geometry work. Retest measured 23 draws in one second while visible at 1440 × 1000/DPR 2, zero paused, zero offscreen and zero with reduced motion. Backing width 864 / CSS width 576 confirms the DPR cap. These counts demonstrate work avoidance, not a battery or GPU benchmark.
2. **Marquee layout stall:** production trace attributed 124 ms forced reflow to `hydrateMarquees` → `fill`, which rewrote markup and immediately read `scrollWidth`. The parent removed this hydration loop. Source inspection confirms its removal; no equivalent post-build production trace was run.
3. **Analytics cost:** production PostHog accounted for 140.1 kB transfer and 120 ms main-thread work in the throttled trace. This is a measured optional optimization, not a release blocker or evidence that PostHog caused the LCP delay. The source skips analytics on local and Chrome-Lighthouse user agents, so Lighthouse results do not include the normal deployed analytics path.
4. **Fonts and critical path:** local variable WOFF2 assets were 48,432 bytes for Inter and 31,340 bytes for JetBrains Mono; both use `font-display: swap`. Homepage preloads both. Production CSS transferred about 6.3 kB (24.4 kB decoded). The trace's longest critical chain was document → stylesheet, 1,303 ms. No image-heavy hero optimization is needed: LCP was text.

## Accessibility findings and disposition

| Severity at baseline | Location | Before | Retested disposition | Criterion or user impact |
| --- | --- | --- | --- | --- |
| High | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/index.astro`, maintenance section; `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/styles/global.css` | At 320 CSS pixels, the grid's minimum content width expanded maintenance text to 336 pixels, ending at x=360; body overflow clipping hid words. A screenshot visually confirmed the right edge was cut off despite document width remaining 320. | Parent constrained minimum width and command overflow. Retest found no ordinary text/control boxes outside the viewport at 320, 768 and 1440 pixels. | WCAG 1.4.10 reflow; hiding overflow does not establish readable content. |
| High | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/index.astro:288`; `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/SiteHeader.astro`; shared styles | WebGL ignored reduced motion; pulse animations continued on home/docs/changelog; infinite motion had no pause control. | Zero WebGL draw activity and zero CSS animations during reduced-motion retest. Visible pause/resume control stopped animation; offscreen rendering stopped. Docs shared-header pulse was removed too. | WCAG 2.2.2 pause/stop/hide and explicit product reduced-motion requirement. |
| Medium | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/CommandSnippet.astro`; `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/scripts/copy-command.ts` | Missing clipboard still displayed COPIED; write rejection was not handled. Fixed accessible name concealed the visual state change and no status region existed. | Simulated rejected promise produced RETRY plus visible manual-copy guidance and live status. Successful mock recorded exact `npx linersh`, displayed COPIED and announced “Command copied.” Button width stayed 92 pixels in both states. | WCAG 4.1.3 status messages; truthful operation feedback. Successful OS clipboard contents were not inspected. |
| Medium | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/index.astro`; `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/changelog.astro` | Main landmark contained only the homepage hero; six narrative sections sat outside it. Home/changelog lacked a skip link. | All seven homepage sections now belong to main; skip links added. Docs already supplied a working skip link. | WCAG 1.3.1 relationships and 2.4.1 bypass repeated blocks. |
| Low | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/index.astro` | Accessibility tree repeated each marquee phrase four times due to authored and JavaScript duplication. | Parent removed JS duplication and marked redundant decorative tracks hidden from assistive technology. | Redundant screen-reader traversal; core claims remain in ordinary page content. |
| Low | `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/CommandSnippet.astro:19` | Production Pagefind excerpts contained “CopyCopied”. | Parent marked controls/status `data-pagefind-ignore`. Source verified; regenerated search results need the final built-site check. | Search results should describe commands rather than control labels. |

## Verification coverage

- **Automated baseline:** deployed homepage Lighthouse mobile accessibility, best practices and SEO all 100. Local homepage, install docs and changelog accessibility/best practices all 100.
- **Automated after changes:** local homepage, troubleshooting docs and changelog again scored 100 accessibility and best practices. SEO remained 92 only because the Astro development toolbar includes a “Learn more” link to https://docs.astro.build/en/reference/cli-reference/#astro-preferences . This is a development artifact, not an application link finding. No automated accessibility failures were returned.
- **320-pixel route coverage:** home, changelog and all ten docs routes: https://www.liner.sh/docs/ , https://www.liner.sh/docs/install/ , https://www.liner.sh/docs/build-a-mixtape/ , https://www.liner.sh/docs/maintenance/ , https://www.liner.sh/docs/mixtapes/ , https://www.liner.sh/docs/tape-format/ , https://www.liner.sh/docs/sharing/ , https://www.liner.sh/docs/cli/ , https://www.liner.sh/docs/tui/ , https://www.liner.sh/docs/troubleshooting/ . Checks ran on their corresponding current-checkout preview URLs. Ordinary docs text and file-tree headers did not clip. Code/table regions retain intended horizontal scrolling.
- **Desktop/tablet:** updated homepage's text/control geometry fitted at 1440 and 768 CSS pixels. A 640-pixel viewport with root font size set to 200% also had no detected text/control overflow. This was a text-size stress test, not actual browser zoom acceptance.
- **Keyboard and semantics:** docs first Tab reaches visible Skip to content with browser outline. Tab to Search, Enter opens its modal, and Escape restores focus to Search. Production search automatically focused its textbox; typing “install” returned seven results; Tab remained in the dialog on Clear, with visible computed focus outline. This is a partial keyboard walk, not exhaustive full-cycle trapping verification. Accessible snapshots expose named headings, links, copy buttons, menu and search controls. Shared command pre elements now have `tabindex="0"` for keyboard scrolling.
- **Motion and clipboard:** behavior counts and mocked clipboard outcomes are detailed above. No writes to user clipboard were required by the failure/success mock tests.
- **Console:** no production homepage errors/warnings/issues returned after the throttled trace.
- **Source hygiene:** `git diff --check` passed during audit handoff. This worker changed only this audit document.

Temporary Lighthouse JSON reports are retained by the tool at:

- Updated homepage: `/var/folders/ts/v1hjn1nj2697d02j9j6649780000gn/T/chrome-devtools-mcp-VDuwpE/report.json`
- Updated troubleshooting docs: `/var/folders/ts/v1hjn1nj2697d02j9j6649780000gn/T/chrome-devtools-mcp-RdTpoL/report.json`
- Updated changelog: `/var/folders/ts/v1hjn1nj2697d02j9j6649780000gn/T/chrome-devtools-mcp-0bM4Xe/report.json`

## Remaining checks and decision

No confirmed high-severity issue remains in the inspected working state. **Approve the inspected scope**, subject to final integration checks after the parent's remaining copy/layout edits.

Not verified: actual browser 200% zoom; physical VoiceOver/NVDA reading and announcement order; complete keyboard walks through every docs menu, table, code region and search result; real-device GPU/battery usage; background-tab visibility behavior; field INP/Core Web Vitals; post-change production build performance and rebuilt Pagefind behavior. The parent adjusted intrinsic grid sizing after the first reflow retest, so the final copy/layout pass must repeat affected viewport checks. This audit does not authorize or establish deployment.


## Final integrated built-site verification

The lead repeated checks after the final copy and keyboard fixes against Astro's static production output, served by this checkout's Watcher process. All 13 pages, including the custom 404, passed the axe WCAG A/AA tagged scan at 320 pixels with zero reported violations and no ordinary text/control clipping. The scan includes all ten docs routes, homepage, and changelog. Wide reference regions keep their own horizontal scrolling.

This final scan found and repaired two unfocusable homepage reference blocks and wide native reference tables. The compile-time Satteri table plugin supplies tabindex without client JavaScript. Actual keyboard focus plus ArrowRight moved a narrow CLI table from scrollLeft 0 to 26. Final visual inspection also found the mobile docs search control extending beneath the menu. Both controls now use the shared 44-pixel role and have separate bounds at 320, 768, and 799 pixels; 800 and 1440 pixels use desktop navigation without a mobile menu. Final screenshots showed the intended terminal palette, readable body text, stable command controls, and no mobile overlap.

The built Pagefind dialog opened with Enter, focused its input, returned eight results for install without Copy/Copied contamination, and closed with Escape while restoring the visible outlined Search docs control. A rejected clipboard promise showed Retry and the manual-copy status within the 320-pixel viewport; the subsequent successful mock captured the exact npx linersh command and announced Command copied. Both button states were 92 pixels wide. These mock checks do not inspect the operating-system clipboard. Homepage, install docs, and changelog also passed 640-pixel geometry checks with the root font size doubled. This remains a text-size simulation rather than actual browser zoom.

One final local built-homepage trace used 390 × 844 pixels, DPR 3, mobile/touch, Slow 4G, CPU ×4, normal motion, and a cache-bypass reload. LCP was 1,544 ms and CLS 0.00. The local server and disabled local analytics differ from production, so this is not a production speed-delta claim. No field data was available. The final root MCP Lighthouse request timed out and is not counted as a pass; the successful earlier bounded Lighthouse checks and the final axe/browser evidence are reported separately.

Final browser evidence is held locally in /tmp/liner-final-browser-qa.json, /tmp/liner-final-header-qa.json, /tmp/liner-final-enlarged-text.json, /tmp/liner-final-interactions.json, and /tmp/liner-final-mobile-trace.json.gz. The browser scan confirms static output rather than a Vite development page. The temporary built-preview package script was restored to astro dev before delivery. Physical screen-reader order, Safari/device behavior, actual browser zoom, and field Core Web Vitals remain unverified. No confirmed blocking defect remains in the inspected scope.

## Targeted headline correction — 2026-10-17

Against `02f3cf6`, removed the homepage's explicit mobile-only break between “AI” and “context.” and the two unused `.hero-title-break` rules. Wording, font scale, color roles, and other surfaces are unchanged. The supplied 1252-pixel-wide @2x screenshot was compared with fresh 626- and 1252-CSS-pixel preview screenshots; the intended two lines now render. Headless Chromium character-range measurements after fonts loaded confirmed exactly “Build reusable” / “AI context.” at 320, 390, 626, 768, 1024, 1252, 1280, and 1440 CSS pixels, with no document horizontal overflow. Visually inspected `/tmp/liner-heading-320.png`, `/tmp/liner-heading-626.png`, and `/tmp/liner-heading-1252.png`. A 640px check with the root font doubled retained readable content without document overflow; this simulates enlarged text, not actual browser zoom.

`npm run build` passed with Node 22.22.0, including all 13 pages and Pagefind. Watcher reused the exact `marketing/site` preview at `http://liner-site-copy-20261016-site-1ce1ccb23419.localhost:8765/`: branch `codex/site-copy-20261016`, commit `02f3cf6` plus this source edit, owned, not stale, zero leftover servers. No performance trace or unrelated interaction retest was needed for this markup/CSS-only correction. No commit, push, or publication was performed by the worker. Skills: `better-typography`, `better-layout`, and `good-css`, selected through `jakub-interfaces`; Chrome DevTools and its CLI guide for rendered verification.
