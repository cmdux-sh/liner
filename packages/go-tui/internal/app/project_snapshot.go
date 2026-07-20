package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

type projectNextKind int

const (
	projectNextUnavailable projectNextKind = iota
	projectNextContinueCorpus
	projectNextCreateOperatingLayer
	projectNextOpenLiner
	projectNextRefreshStatus
	projectNextReviewSynthesis
	projectNextCompileRefresh
	projectNextReviewOperatingLayer
)

func loadProjectSnapshot(runner core.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := runner.InspectMaintenanceProject(path)
		return projectSnapshotMsg{path: path, snapshot: snapshot, err: err}
	}
}

func refreshProjectStatus(runner core.Runner, path string) tea.Cmd {
	return func() tea.Msg {
		status, err := runner.RefreshProjectStatus(path)
		return projectStatusRefreshedMsg{path: path, status: status, err: err}
	}
}

func (m *Model) beginProjectSnapshotLoad(path string) {
	m.projectSnapshotPath = path
	m.projectSnapshot = nil
	m.projectSnapshotErr = ""
	m.projectSnapshotAttempted = true
	m.projectSnapshotLoading = true
}

func (m Model) currentProjectSnapshot() *core.MaintenanceProjectSnapshot {
	if m.projectSnapshot == nil || m.projectSnapshotPath != m.currentPath {
		return nil
	}
	return m.projectSnapshot
}

func (m Model) projectSnapshotDegraded() bool {
	return m.projectSnapshotAttempted && m.currentProjectSnapshot() == nil
}

func (m Model) projectMutationsAvailable() bool {
	if snapshot := m.currentProjectSnapshot(); snapshot != nil {
		return snapshot.Capabilities["plan"] && snapshot.Capabilities["apply"]
	}
	return !m.projectSnapshotAttempted
}

func (m Model) projectNextKind() projectNextKind {
	snapshot := m.currentProjectSnapshot()
	if snapshot == nil {
		if m.projectSnapshotAttempted {
			return projectNextUnavailable
		}
		return projectNextContinueCorpus
	}
	if !snapshot.Capabilities["plan"] || !snapshot.Capabilities["apply"] {
		if !snapshot.Lifecycle.Stale && snapshot.Lifecycle.Milestone == "project_complete" && snapshot.Lifecycle.OperatingLayer.State == "ready" {
			return projectNextOpenLiner
		}
		return projectNextUnavailable
	}
	if snapshot.Lifecycle.Stale {
		if refresh := snapshot.Lifecycle.Refresh; refresh != nil {
			switch {
			case refresh.Synthesis.State == "review_required":
				return projectNextReviewSynthesis
			case refresh.Corpus.State == "compile_required":
				return projectNextCompileRefresh
			case refresh.OperatingLayer.State == "review_required":
				return projectNextReviewOperatingLayer
			}
		}
		return projectNextRefreshStatus
	}
	switch snapshot.Lifecycle.Milestone {
	case "project_complete":
		return projectNextOpenLiner
	case "corpus_ready":
		return projectNextCreateOperatingLayer
	default:
		return projectNextContinueCorpus
	}
}

func (m Model) projectSnapshotDiagnostic() string {
	if strings.TrimSpace(m.projectSnapshotErr) != "" {
		return m.projectSnapshotErr
	}
	if m.projectSnapshotLoading {
		return "Waiting for Liner Core to return a trustworthy Project Snapshot."
	}
	return "Liner Core did not return a trustworthy Project Snapshot."
}
