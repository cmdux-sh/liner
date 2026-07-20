package airunner

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
	if normalizeAgent(provider) != "codex" {
		return nil
	}
	return []ReasoningEffortOption{
		{Label: "Model default"},
		{Label: "None", Value: "none"},
		{Label: "Low", Value: "low"},
		{Label: "Medium", Value: "medium"},
		{Label: "High", Value: "high"},
		{Label: "Extra high", Value: "xhigh"},
		{Label: "Maximum", Value: "max"},
	}
}

func ReasoningEffortLabel(value string) string {
	for _, option := range ReasoningEffortOptions("codex") {
		if option.Value == value {
			return option.Label
		}
	}
	return "Model default"
}

func ModelOptions(provider string) []ModelOption {
	if normalizeAgent(provider) == "claude" {
		return []ModelOption{
			{Label: "Claude default"},
			{Label: "Sonnet", Value: "sonnet"},
			{Label: "Opus", Value: "opus"},
			{Label: "Custom model ID", Custom: true},
		}
	}
	return []ModelOption{
		{Label: "Auto", Auto: true},
		{Label: "OpenAI default"},
		{Label: "GPT-5.6 Sol", Value: "gpt-5.6-sol"},
		{Label: "GPT-5.6 Terra", Value: "gpt-5.6-terra"},
		{Label: "GPT-5.6 Luna", Value: "gpt-5.6-luna"},
		{Label: "Custom model ID", Custom: true},
	}
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

// ApplyAutoModelPolicy resolves built-in Auto choices for Go-owned AI tasks.
func ApplyAutoModelPolicy(descriptor Descriptor, task string) Descriptor {
	if descriptor.ID != "codex" || descriptor.Model != "" || descriptor.ModelMode == "default" {
		return descriptor
	}
	switch task {
	case "jtbd-clarify", "candidates", "evaluation":
		descriptor.Model = "gpt-5.6-luna"
		descriptor.ModelIsAuto = true
		if descriptor.ReasoningEffort == "" {
			descriptor.ReasoningEffort = "high"
			descriptor.EffortIsAuto = true
		}
	case "framing", "quality", "synthesis", "improvement", "assembly":
		descriptor.Model = "gpt-5.6-sol"
		descriptor.ModelIsAuto = true
		if descriptor.ReasoningEffort == "" {
			descriptor.ReasoningEffort = "medium"
			descriptor.EffortIsAuto = true
		}
	}
	return descriptor
}
