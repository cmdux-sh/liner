package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const (
	clarificationDraftRelPath   = "working/00-clarification-draft.json"
	clarificationStatusComplete = "complete"
	clarificationDraftSchema    = "liner.clarification-draft.v1"
)

type clarificationDraft struct {
	Schema    string   `json:"schema"`
	JTBD      string   `json:"jtbd"`
	Questions []string `json:"questions"`
	Answers   []string `json:"answers"`
	Step      int      `json:"step"`
	Updated   string   `json:"updated"`
}

func (m Model) needsClarificationBeforeMethodology() bool {
	if strings.TrimSpace(m.currentPath) == "" {
		return false
	}
	if strings.TrimSpace(tapeJTBD(m.currentTape)) == "" {
		return false
	}
	if tapeClarificationComplete(m.currentTape) {
		return false
	}
	if m.hasPendingAssemblyDraft() || m.hasCorpusReady() {
		return false
	}
	return projectPipelineStep(m.currentPath, m.currentTape) == 0
}

func tapeClarificationComplete(current tape.Tape) bool {
	if len(current.JTBDClarifications) > 0 {
		return true
	}
	if current.JTBDClarificationStatus == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(*current.JTBDClarificationStatus), clarificationStatusComplete)
}

func markTapeClarificationComplete(current tape.Tape) tape.Tape {
	status := clarificationStatusComplete
	current.JTBDClarificationStatus = &status
	return current
}

func (m Model) clarificationDraft() clarificationDraft {
	questions := make([]string, 0, len(m.clarifyQuestions))
	for _, question := range m.clarifyQuestions {
		text := strings.TrimSpace(question.question)
		if text != "" {
			questions = append(questions, text)
		}
	}
	answers := make([]string, len(questions))
	for i := range answers {
		if i < len(m.clarifyAnswers) {
			answers[i] = strings.TrimSpace(m.clarifyAnswers[i])
		}
	}
	step := m.clarifyStep
	if len(questions) == 0 {
		step = 0
	} else if step < 0 {
		step = 0
	} else if step >= len(questions) {
		step = len(questions) - 1
	}
	return clarificationDraft{
		Schema:    clarificationDraftSchema,
		JTBD:      tapeJTBD(m.currentTape),
		Questions: questions,
		Answers:   answers,
		Step:      step,
	}
}

func (m *Model) applyClarificationDraft(draft clarificationDraft) {
	m.screen = screenClarify
	m.clarifyLoading = false
	m.clarifyError = ""
	m.clarifyQuestions = make([]clarifyQuestion, 0, len(draft.Questions))
	for _, question := range draft.Questions {
		question = strings.TrimSpace(question)
		if question != "" {
			m.clarifyQuestions = append(m.clarifyQuestions, clarifyQuestion{question: question})
		}
	}
	m.clarifyAnswers = make([]string, len(m.clarifyQuestions))
	for i := range m.clarifyAnswers {
		if i < len(draft.Answers) {
			m.clarifyAnswers[i] = strings.TrimSpace(draft.Answers[i])
		}
	}
	m.setClarifyField(draft.Step)
	m.note = "Resumed clarification draft."
}

func (m *Model) persistClarificationDraft() {
	if strings.TrimSpace(m.currentPath) == "" || !m.canEditClarifyText() {
		return
	}
	if err := writeClarificationDraft(m.currentPath, m.clarificationDraft()); err != nil {
		m.err = "Could not save clarification draft: " + err.Error()
	}
}

func loadClarificationDraft(project string, jtbd string) (clarificationDraft, bool, error) {
	if strings.TrimSpace(project) == "" {
		return clarificationDraft{}, false, nil
	}
	data, err := os.ReadFile(projectAbsPath(project, clarificationDraftRelPath))
	if errors.Is(err, os.ErrNotExist) {
		return clarificationDraft{}, false, nil
	}
	if err != nil {
		return clarificationDraft{}, false, err
	}
	var draft clarificationDraft
	if err := json.Unmarshal(data, &draft); err != nil {
		return clarificationDraft{}, false, err
	}
	if strings.TrimSpace(draft.JTBD) != strings.TrimSpace(jtbd) {
		return clarificationDraft{}, false, nil
	}
	if len(draft.Questions) == 0 {
		return clarificationDraft{}, false, nil
	}
	return draft, true, nil
}

func writeClarificationDraft(project string, draft clarificationDraft) error {
	if strings.TrimSpace(project) == "" {
		return nil
	}
	draft.Schema = clarificationDraftSchema
	draft.Updated = time.Now().UTC().Format(time.RFC3339)
	path := projectAbsPath(project, clarificationDraftRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(draft, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func removeClarificationDraft(project string) error {
	if strings.TrimSpace(project) == "" {
		return nil
	}
	err := os.Remove(projectAbsPath(project, clarificationDraftRelPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
