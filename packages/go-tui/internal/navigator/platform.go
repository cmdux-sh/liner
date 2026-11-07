package navigator

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func copyToClipboard(text string) error {
	name, args, err := clipboardCommand()
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func clipboardCommand() (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil, nil
	case "windows":
		return "clip", nil, nil
	}
	for _, option := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}} {
		if path, err := exec.LookPath(option[0]); err == nil {
			return path, option[1:], nil
		}
	}
	return "", nil, fmt.Errorf("no clipboard tool found")
}

func openExternally(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	}
	for _, name := range []string{"xdg-open", "wslview"} {
		if path, err := exec.LookPath(name); err == nil {
			return exec.Command(path, target).Start()
		}
	}
	return fmt.Errorf("no way to open files found")
}
