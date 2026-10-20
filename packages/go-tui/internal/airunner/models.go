package airunner

import "strings"

// Built-in defaults. OpenAI Auto runs Sol 6.1 at Medium for every task; Claude
// runs Sonnet 5.5 at Medium unless Settings chooses otherwise.
const (
	AutoOpenAIModel     = "gpt-6.1-sol"
	AutoOpenAIEffort    = "medium"
	DefaultClaudeModel  = "claude-sonnet-5-5"
	DefaultClaudeEffort = "medium"
)

// legacyClaudeModels maps Claude Code aliases saved by earlier Liner versions
// to the models that replaced them in Settings.
var legacyClaudeModels = map[string]string{
	"sonnet": "claude-sonnet-5-5",
	"opus":   "claude-opus-5-5",
}

type ModelOption struct {
	Label  string
	Value  string
	Auto   bool
	Custom bool
}

type ReasoningEffortOption struct {
	Label string
	Value string
}

func ReasoningEffortOptions(provider string) []ReasoningEffortOption {
	switch normalizeAgent(provider) {
	case "codex":
		return []ReasoningEffortOption{
			{Label: "Model default"},
			{Label: "None", Value: "none"},
			{Label: "Low", Value: "low"},
			{Label: "Medium", Value: "medium"},
			{Label: "High", Value: "high"},
			{Label: "Extra high", Value: "xhigh"},
			{Label: "Maximum", Value: "max"},
		}
	case "claude":
		// Claude Code's --effort levels. An unset preference runs Medium.
		return []ReasoningEffortOption{
			{Label: "Low", Value: "low"},
			{Label: "Medium", Value: "medium"},
			{Label: "High", Value: "high"},
			{Label: "Extra high", Value: "xhigh"},
			{Label: "Maximum", Value: "max"},
		}
	}
	return nil
}

// DefaultReasoningEffort is the effort a provider runs when Settings leaves it
// unset: Medium for Claude; for OpenAI it depends on Auto (see ApplyDefaults).
func DefaultReasoningEffort(provider string) string {
	if normalizeAgent(provider) == "claude" {
		return DefaultClaudeEffort
	}
	return ""
}

func ReasoningEffortLabel(provider string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultReasoningEffort(provider)
	}
	for _, option := range ReasoningEffortOptions(provider) {
		if option.Value == value {
			return option.Label
		}
	}
	return "Model default"
}

func ValidReasoningEffort(provider string, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, option := range ReasoningEffortOptions(provider) {
		if option.Value == value {
			return true
		}
	}
	return false
}

func ModelOptions(provider string) []ModelOption {
	if normalizeAgent(provider) == "claude" {
		return []ModelOption{
			{Label: "Sonnet 5.5", Value: DefaultClaudeModel},
			{Label: "Opus 5.5", Value: "claude-opus-5-5"},
			{Label: "Custom model ID", Custom: true},
		}
	}
	return []ModelOption{
		{Label: "Auto", Auto: true},
		{Label: "OpenAI default"},
		{Label: "Sol 6.1", Value: AutoOpenAIModel},
		{Label: "Custom model ID", Custom: true},
	}
}

// NormalizeModel maps a saved model to the catalog: an empty Claude model is
// Sonnet 5.5, and the old Claude aliases map to their 5.5 models.
func NormalizeModel(provider string, model string) string {
	model = strings.TrimSpace(model)
	if normalizeAgent(provider) != "claude" {
		return model
	}
	if model == "" {
		return DefaultClaudeModel
	}
	if replacement, ok := legacyClaudeModels[strings.ToLower(model)]; ok {
		return replacement
	}
	return model
}

func IsCuratedModel(provider string, model string) bool {
	if model == "" {
		return true
	}
	for _, option := range ModelOptions(provider) {
		if !option.Custom && option.Value == model {
			return true
		}
	}
	return false
}

func ModelLabel(provider string, model string) string {
	model = NormalizeModel(provider, model)
	for _, option := range ModelOptions(provider) {
		if !option.Auto && !option.Custom && option.Value == model {
			return option.Label
		}
	}
	if model != "" {
		return "Custom: " + model
	}
	return ModelOptions(provider)[0].Label
}

// ApplyDefaults fills in the built-in model and Thinking effort for a new run.
// Values filled here are marked Auto so a provider that rejects them can be
// retried once on its own default; values chosen in Settings never are.
func ApplyDefaults(descriptor Descriptor) Descriptor {
	switch descriptor.ID {
	case "claude":
		if descriptor.Model == "" {
			descriptor.Model = DefaultClaudeModel
			descriptor.ModelIsAuto = true
		}
		if descriptor.ReasoningEffort == "" {
			descriptor.ReasoningEffort = DefaultClaudeEffort
			descriptor.EffortIsAuto = true
		}
	case "codex":
		if descriptor.Model != "" || descriptor.ModelMode == "default" {
			return descriptor
		}
		descriptor.Model = AutoOpenAIModel
		descriptor.ModelIsAuto = true
		if descriptor.ReasoningEffort == "" {
			descriptor.ReasoningEffort = AutoOpenAIEffort
			descriptor.EffortIsAuto = true
		}
	}
	return descriptor
}
