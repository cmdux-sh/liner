package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/navigator"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.BoolVar(showVersion, "v", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("liner-tui %s\n", version)
		return
	}
	model, err := navigator.New(navigator.Options{Root: os.Getenv("LINER_DIR"), Version: version})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if os.Getenv("LINER_TUI_INITIALIZATION_PROBE") == "1" {
		fmt.Println("liner-tui initialization probe passed")
		return
	}

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
