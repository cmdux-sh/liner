package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

// jsSetupSize is what `liner setup-js` installs: Playwright's headless
// Chromium shell and its ffmpeg helper, measured with Playwright 1.64 on macOS
// arm64. Keep it in step with HEADLESS_CHROMIUM_SIZE in src/liner/cli.py.
const jsSetupSize = "about 100 MB to download, about 220 MB on disk"

func setupJS(r core.Runner) tea.Cmd {
	return func() tea.Msg {
		return jsSetupFinishedMsg{err: r.SetupJS()}
	}
}

func compileWarningNeedsJSSetup(warning core.CompileWarningPayload) bool {
	return compileMessageNeedsJSSetup(warning.Message)
}

func compileMessageNeedsJSSetup(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "liner setup-js") ||
		strings.Contains(message, "playwright chromium isn't installed") ||
		strings.Contains(message, "render: js needs playwright") ||
		strings.Contains(message, "js-rendering support")
}

func (m Model) compileNeedsJSSetup() bool {
	if m.compileResult == nil {
		return false
	}
	for _, warning := range m.compileResult.Warnings {
		if compileWarningNeedsJSSetup(warning) {
			return true
		}
	}
	return false
}

// compileJSSourceCount counts the Sources that still need a browser before
// Liner can read them.
func (m Model) compileJSSourceCount() int {
	if m.compileResult == nil {
		return 0
	}
	count := 0
	for _, warning := range m.compileResult.Warnings {
		if compileWarningNeedsJSSetup(warning) {
			count++
		}
	}
	return count
}
