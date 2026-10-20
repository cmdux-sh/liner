package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

func browserRequiredWarning(url string) core.CompileWarningPayload {
	return core.CompileWarningPayload{
		URL:      url,
		Message:  "render: js needs Playwright. Run: liner setup-js",
		Severity: "error",
	}
}

func TestCompileRepairKeyNeverStartsBrowserInstall(t *testing.T) {
	for _, keyName := range []string{"r", "enter"} {
		t.Run(keyName, func(t *testing.T) {
			m := Model{
				screen:                 screenCompile,
				width:                  100,
				height:                 30,
				compilePane:            compilePaneSources,
				compileSourcesReviewed: true,
				compileResult: &core.CompileResultPayload{
					MixtapePath: "/tmp/MIXTAPE.md",
					Summary:     core.CompileSummary{Total: 4, Succeeded: 2, Failed: 2},
					Warnings: []core.CompileWarningPayload{
						browserRequiredWarning("https://example.test/app"),
						{URL: "https://example.test/blocked", Message: "The source blocked access.", Severity: "error"},
					},
				},
			}
			if got := m.compileRepairKeyHelp(); got != "retry compile" {
				t.Fatalf("repair key should name the retry it runs, got %q", got)
			}

			msg := tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
			if keyName == "r" {
				msg = tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"})
			}
			next, _ := m.handleKey(msg)
			if next.jsSetupRunning || next.jsSetupRetryCompile {
				t.Fatalf("%s must never start the browser download: running=%t retry=%t", keyName, next.jsSetupRunning, next.jsSetupRetryCompile)
			}
		})
	}
}

func TestCompileBrowserOnlyFailuresOfferInstallWithMeasuredSize(t *testing.T) {
	m := Model{
		screen:      screenCompile,
		width:       100,
		height:      30,
		compilePane: compilePaneIssues,
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 5, Succeeded: 3, Failed: 2},
			Warnings: []core.CompileWarningPayload{
				browserRequiredWarning("https://example.test/app"),
				browserRequiredWarning("https://example.test/other-app"),
			},
		},
	}
	if m.compileHasRepairableSources() {
		t.Fatal("Sources that only need a browser have no free repair; the install is a separate offer")
	}

	sources, cmd := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil || sources.jsSetupRunning {
		t.Fatalf("viewing sources must not start work: cmd=%v running=%t", cmd, sources.jsSetupRunning)
	}
	view := strings.Join(strings.Fields(stripANSICodesForTest(sources.viewCompile())), " ")
	for _, want := range []string{
		"2 need a browser (i installs it)",
		"Next: Press i to install JS rendering (about 100 MB download, 220 MB on disk).",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("source view should say how many need a browser and offer the install with its size, missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "150 MB") || strings.Contains(view, "reinstalling Playwright will not help") {
		t.Fatalf("source view should not show stale size or post-install copy:\n%s", view)
	}
	if got := sources.nextAction(); got != "Create the Operating Layer with 3 usable Sources." {
		t.Fatalf("after review, continuing without the browser Sources is the person's call, got %q", got)
	}
	if !hasHelpDesc(sources.helpForScreen().ShortHelp(), "install JS") {
		t.Fatalf("install stays available through i: %#v", sources.helpForScreen().ShortHelp())
	}
	if !hasHelpDesc(sources.helpForScreen().ShortHelp(), "create layer") {
		t.Fatalf("enter should continue with the usable Sources: %#v", sources.helpForScreen().ShortHelp())
	}

	repaired, _ := sources.handleKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))
	if repaired.jsSetupRunning {
		t.Fatal("r must not start the browser download")
	}

	installing, installCmd := sources.handleKey(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	if installCmd == nil || !installing.jsSetupRunning {
		t.Fatal("i is the one key that starts the browser install")
	}
}

func TestCompileDoneOffersBrowserInstallWhenSourcesStillNeedIt(t *testing.T) {
	m := Model{
		screen:                 screenCompile,
		width:                  100,
		height:                 30,
		compiling:              true,
		compileRepairAttempted: true,
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 3, Succeeded: 2, Failed: 1},
			Warnings:    []core.CompileWarningPayload{browserRequiredWarning("https://example.test/app")},
		},
	}
	updated, _ := m.Update(compileDoneMsg{})
	got := updated.(Model)
	view := strings.Join(strings.Fields(stripANSICodesForTest(got.viewCompile())), " ")
	for _, want := range []string{
		"One source needs a browser to reveal the article text.",
		"(about 100 MB to download, about 220 MB on disk)",
		"Repair never installs it.",
		"Press i to install JS rendering, or continue without these sources.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("compile result should offer the install with its size after a free repair, missing %q:\n%s", want, view)
		}
	}
	if got.jsSetupRunning {
		t.Fatal("finishing a compile must not start the browser download")
	}
}

func TestCompileSourcesPaneKeepsSourceDetailOnScreenAt100x30(t *testing.T) {
	m := Model{
		screen:     screenCompile,
		width:      100,
		height:     30,
		help:       help.New(),
		compileBar: newCompileProgress(48),
		compileErr: "Partial compile: 1/3 sources were usable. Review source issues before relying on MIXTAPE.md.",
		compileResult: &core.CompileResultPayload{
			MixtapePath: "/tmp/MIXTAPE.md",
			Summary:     core.CompileSummary{Total: 3, Succeeded: 1, Failed: 2},
			Warnings: []core.CompileWarningPayload{
				browserRequiredWarning("http://127.0.0.1:8000/spa/javascript-only-guidance"),
				{URL: "http://127.0.0.1:8000/missing/journal-study", Message: "Failed to fetch: HTTP 404 not found", Severity: "error"},
			},
		},
	}
	m.help.SetWidth(96)
	for _, repaired := range []bool{false, true} {
		state := m.viewCompileSourcesNext()
		state.compileRepairAttempted = repaired
		rendered := state.View().Content
		plain := stripANSICodesForTest(rendered)
		if got := lipgloss.Height(rendered); got > 30 {
			t.Fatalf("repaired=%t: view should fit 100 x 30, got %d lines", repaired, got)
		}
		if strings.Contains(plain, "more content hidden") {
			t.Fatalf("repaired=%t: source detail should stay on screen:\n%s", repaired, plain)
		}
		for _, want := range []string{"1 needs a browser (i installs it)", "Next:     Press i to install JS rendering (about 100 MB download, 220 MB on disk)."} {
			if !strings.Contains(plain, want) {
				t.Fatalf("repaired=%t: missing %q:\n%s", repaired, want, plain)
			}
		}
	}
}
