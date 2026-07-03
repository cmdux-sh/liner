package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

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
