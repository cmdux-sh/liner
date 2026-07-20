package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/app"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.BoolVar(showVersion, "v", false, "print version")
	maintenanceSmokeProject := flag.String("maintenance-smoke-project", "", "run the installed Core maintenance release probe against a temporary Project")
	maintenanceSmokeOperation := flag.String("maintenance-smoke-operation", "", "one JSON maintenance operation for the installed release probe")
	flag.Parse()

	if *showVersion {
		fmt.Printf("liner-tui %s\n", version)
		return
	}
	if *maintenanceSmokeProject != "" || *maintenanceSmokeOperation != "" {
		if *maintenanceSmokeProject == "" || *maintenanceSmokeOperation == "" {
			fmt.Fprintln(os.Stderr, "maintenance smoke requires both --maintenance-smoke-project and --maintenance-smoke-operation")
			os.Exit(2)
		}
		result, err := app.RunInstalledMaintenanceSmoke(*maintenanceSmokeProject, *maintenanceSmokeOperation)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(result))
		return
	}

	model, err := app.New(app.Options{
		BaseDir: os.Getenv("LINER_DIR"),
	})
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
