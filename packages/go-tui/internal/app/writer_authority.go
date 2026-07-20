package app

import "fmt"

const coreWriterRemediation = "Use Maintain Project to inspect, plan, and apply a Liner Core Change Set; do not edit canonical Project files directly."

func legacyCoreWriterError(action string) error {
	return fmt.Errorf("cannot %s: this legacy Go TUI path is disabled because Liner Core is the sole Project write authority. %s", action, coreWriterRemediation)
}
