package progress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type PhaseID string

const (
	PhaseFraming    PhaseID = "framing"
	PhaseGate0      PhaseID = "gate0"
	PhaseCandidates PhaseID = "candidates"
	PhaseGate1      PhaseID = "gate1"
	PhaseEvaluation PhaseID = "evaluation"
	PhaseQuality    PhaseID = "quality"
	PhaseGate2      PhaseID = "gate2"
	PhaseSynthesis  PhaseID = "synthesis"
	PhaseAssembly   PhaseID = "assembly"
	PhaseCompile    PhaseID = "compile"
)

var PhaseOrder = []PhaseID{
	PhaseFraming,
	PhaseGate0,
	PhaseCandidates,
	PhaseGate1,
	PhaseEvaluation,
	PhaseQuality,
	PhaseGate2,
	PhaseSynthesis,
	PhaseAssembly,
	PhaseCompile,
}

type Progress struct {
	Step        int    `json:"step"`
	LastTouched string `json:"lastTouched"`
}

type GateState struct {
	Gate0Accepted bool `json:"gate0Accepted"`
	Gate1Accepted bool `json:"gate1Accepted"`
	Gate2Accepted bool `json:"gate2Accepted"`
}

func Read(project string) Progress {
	data, err := os.ReadFile(filepath.Join(project, ".liner-progress.json"))
	if err != nil {
		return Progress{Step: 0, LastTouched: time.Now().Format(time.RFC3339)}
	}
	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return Progress{Step: 0, LastTouched: time.Now().Format(time.RFC3339)}
	}
	return clamp(p)
}

func Write(project string, p Progress) error {
	p = clamp(p)
	p.LastTouched = time.Now().Format(time.RFC3339)
	out, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(project, ".liner-progress.json"), append(out, '\n'), 0o644)
}

func MarkPhaseComplete(project string, phase PhaseID) (Progress, error) {
	current := Read(project)
	expected := PhaseIndex(phase)
	if expected < 0 || current.Step != expected {
		return current, nil
	}
	next := Progress{Step: min(len(PhaseOrder), current.Step+1)}
	if err := Write(project, next); err != nil {
		return current, err
	}
	return Read(project), nil
}

func PhaseIndex(phase PhaseID) int {
	for index, candidate := range PhaseOrder {
		if candidate == phase {
			return index
		}
	}
	return -1
}

func GateAfterPhase(phase PhaseID) (PhaseID, bool) {
	switch phase {
	case PhaseFraming:
		return PhaseGate0, true
	case PhaseCandidates:
		return PhaseGate1, true
	case PhaseQuality:
		return PhaseGate2, true
	default:
		return "", false
	}
}

func ReadGateState(project string) GateState {
	data, err := os.ReadFile(filepath.Join(project, ".liner-gates.json"))
	if err != nil {
		return GateState{}
	}
	var state GateState
	if err := json.Unmarshal(data, &state); err != nil {
		return GateState{}
	}
	return state
}

func WriteGateState(project string, state GateState) error {
	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(project, ".liner-gates.json"), append(out, '\n'), 0o644)
}

func AcceptGate(project string, gate PhaseID) (Progress, error) {
	state := ReadGateState(project)
	switch gate {
	case PhaseGate0:
		state.Gate0Accepted = true
	case PhaseGate1:
		state.Gate1Accepted = true
	case PhaseGate2:
		state.Gate2Accepted = true
	default:
		return Read(project), nil
	}
	if err := WriteGateState(project, state); err != nil {
		return Read(project), err
	}
	return MarkPhaseComplete(project, gate)
}

func clamp(p Progress) Progress {
	if p.Step < 0 {
		p.Step = 0
	}
	if p.Step > len(PhaseOrder) {
		p.Step = len(PhaseOrder)
	}
	if p.LastTouched == "" {
		p.LastTouched = time.Now().Format(time.RFC3339)
	}
	return p
}
