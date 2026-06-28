package progress

import "testing"

func TestMarkPhaseCompleteAdvancesOnlyCurrentPhase(t *testing.T) {
	project := t.TempDir()

	got, err := MarkPhaseComplete(project, PhaseFraming)
	if err != nil {
		t.Fatal(err)
	}
	if got.Step != 1 {
		t.Fatalf("expected framing to advance to gate0 step, got %d", got.Step)
	}

	got, err = MarkPhaseComplete(project, PhaseCandidates)
	if err != nil {
		t.Fatal(err)
	}
	if got.Step != 1 {
		t.Fatalf("out-of-order candidates should not advance, got %d", got.Step)
	}
}

func TestAcceptGateWritesGateStateAndAdvancesCursor(t *testing.T) {
	project := t.TempDir()
	if _, err := MarkPhaseComplete(project, PhaseFraming); err != nil {
		t.Fatal(err)
	}

	got, err := AcceptGate(project, PhaseGate0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Step != 2 {
		t.Fatalf("expected accepted gate0 to advance to candidates, got %d", got.Step)
	}
	if !ReadGateState(project).Gate0Accepted {
		t.Fatal("expected gate0 to be written as accepted")
	}
}

func TestReadClampsProgressStep(t *testing.T) {
	project := t.TempDir()
	if err := Write(project, Progress{Step: 999}); err != nil {
		t.Fatal(err)
	}

	got := Read(project)
	if got.Step != len(PhaseOrder) {
		t.Fatalf("expected clamped step, got %d", got.Step)
	}
}

func TestGateAfterPhase(t *testing.T) {
	for _, tc := range []struct {
		phase PhaseID
		gate  PhaseID
		ok    bool
	}{
		{PhaseFraming, PhaseGate0, true},
		{PhaseCandidates, PhaseGate1, true},
		{PhaseQuality, PhaseGate2, true},
		{PhaseEvaluation, "", false},
	} {
		got, ok := GateAfterPhase(tc.phase)
		if got != tc.gate || ok != tc.ok {
			t.Fatalf("GateAfterPhase(%q) = %q, %v; want %q, %v", tc.phase, got, ok, tc.gate, tc.ok)
		}
	}
}
