package agent

import (
	"encoding/json"
	"fmt"
)

type Event struct {
	Kind string `json:"kind"`

	Model     string `json:"model,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Text      string `json:"text,omitempty"`
	Message   string `json:"message,omitempty"`
	Status    string `json:"status,omitempty"`

	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	OK      *bool           `json:"ok,omitempty"`
	Preview string          `json:"preview,omitempty"`

	InputTokens         int     `json:"inputTokens,omitempty"`
	OutputTokens        int     `json:"outputTokens,omitempty"`
	CacheReadTokens     int     `json:"cacheReadTokens,omitempty"`
	CacheCreationTokens int     `json:"cacheCreationTokens,omitempty"`
	DurationMs          int     `json:"durationMs,omitempty"`
	Turns               int     `json:"turns,omitempty"`
	FinalText           string  `json:"finalText,omitempty"`
	CostUsd             float64 `json:"costUsd,omitempty"`

	PhaseID   string `json:"phaseId,omitempty"`
	Project   string `json:"project,omitempty"`
	Agent     string `json:"agent,omitempty"`
	SkillPath string `json:"skillPath,omitempty"`
	Resume    bool   `json:"resume,omitempty"`

	Code        *int   `json:"code,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	Outcome     string `json:"outcome,omitempty"`
	FailureKind string `json:"failureKind,omitempty"`
	Recovery    string `json:"recovery,omitempty"`
	Category    string `json:"category,omitempty"`
	LogPath     string `json:"logPath,omitempty"`

	Raw json.RawMessage `json:"-"`
}

func DecodeEvent(line []byte) (Event, error) {
	var event Event
	if len(line) == 0 {
		return event, fmt.Errorf("empty agent event")
	}
	if err := json.Unmarshal(line, &event); err != nil {
		return event, err
	}
	if event.Kind == "" {
		return event, fmt.Errorf("agent event missing kind")
	}
	event.Raw = append(event.Raw[:0], line...)
	return event, nil
}
