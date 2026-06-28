package app

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

func openPath(path string) tea.Cmd {
	return func() tea.Msg {
		cmd, args, err := openCommand(path)
		if err != nil {
			return pathOpenedMsg{err: err}
		}
		if err := runDetached(cmd, args...); err != nil {
			return pathOpenedMsg{err: err}
		}
		return pathOpenedMsg{}
	}
}

func openCommand(path string) (string, []string, error) {
	return openCommandFor(runtime.GOOS, path, exec.LookPath)
}

func openCommandFor(goos string, path string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{path}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}, nil
	default:
		for _, name := range []string{"xdg-open", "wslview"} {
			if resolved, err := lookPath(name); err == nil && resolved != "" {
				return resolved, []string{path}, nil
			}
		}
		return "", nil, fmt.Errorf("could not find an opener for %s; install xdg-open or set up wslview", goos)
	}
}

func copyMixtape(project string) tea.Cmd {
	return func() tea.Msg {
		path := projectAbsPath(project, "MIXTAPE.md")
		data, err := os.ReadFile(path)
		if err != nil {
			return mixtapeCopiedMsg{err: fmt.Errorf("could not read MIXTAPE.md: %w", err)}
		}
		if err := copyTextToClipboard(string(data)); err != nil {
			return mixtapeCopiedMsg{err: err}
		}
		return mixtapeCopiedMsg{}
	}
}

func shareProject(r core.Runner, project string) tea.Cmd {
	return func() tea.Msg {
		path, err := r.Share(project)
		return projectSharedMsg{path: path, err: err}
	}
}

func copyTextToClipboard(text string) error {
	cmd, args, err := clipboardCommand()
	if err != nil {
		return err
	}
	return runWithInput(cmd, args, text)
}

func clipboardCommand() (string, []string, error) {
	return clipboardCommandFor(runtime.GOOS, exec.LookPath)
}

func clipboardCommandFor(goos string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		return "pbcopy", nil, nil
	case "windows":
		return "clip", nil, nil
	default:
		candidates := []struct {
			name string
			args []string
		}{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
		for _, candidate := range candidates {
			if resolved, err := lookPath(candidate.name); err == nil && resolved != "" {
				return resolved, candidate.args, nil
			}
		}
		return "", nil, fmt.Errorf("no clipboard helper found for %s; install wl-copy, xclip, or xsel", goos)
	}
}

func runWithInput(name string, args []string, input string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
		return err
	}
	return nil
}

func runDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
