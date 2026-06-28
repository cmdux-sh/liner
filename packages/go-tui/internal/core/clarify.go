package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Agent struct {
	ID   string
	Name string
	Bin  string
}

func GenerateClarifyingQuestions(cwd string, jtbd string, timeout time.Duration) ([]string, error) {
	jtbd = strings.TrimSpace(jtbd)
	if jtbd == "" {
		return nil, errors.New("job to be done is empty")
	}
	agent, err := resolveAgent()
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	prompt := buildClarifyPrompt(jtbd)

	failures := []string{}
	for attempt := 1; attempt <= 2; attempt++ {
		questions, err := generateClarifyingQuestionsAttempt(agent, cwd, prompt, timeout)
		if err == nil {
			return questions, nil
		}
		failures = append(failures, err.Error())
	}
	return nil, fmt.Errorf("%s could not generate clarification questions after retry: first attempt: %s; retry: %s", agent.Name, failures[0], failures[1])
}

func generateClarifyingQuestionsAttempt(agent Agent, cwd string, prompt string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := runClarifyAgent(ctx, agent, cwd, prompt)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("%s did not return clarification questions before timeout", agent.Name)
	}
	if err != nil {
		return nil, err
	}
	questions := ParseClarifyingQuestions(output)
	if len(questions) == 0 {
		return nil, fmt.Errorf("%s did not return a JSON array of questions", agent.Name)
	}
	if len(questions) > 6 {
		questions = questions[:6]
	}
	return questions, nil
}

func resolveAgent() (Agent, error) {
	if preferred := strings.ToLower(strings.TrimSpace(os.Getenv("LINER_AGENT"))); preferred != "" {
		if agent, ok := lookupAgent(preferred); ok {
			return agent, nil
		}
		return Agent{}, fmt.Errorf("LINER_AGENT=%s is not available", preferred)
	}
	if configured := configuredAgentID(); configured != "" {
		if agent, ok := lookupAgent(configured); ok {
			return agent, nil
		}
	}
	for _, id := range []string{"claude", "codex"} {
		if agent, ok := lookupAgent(id); ok {
			return agent, nil
		}
	}
	return Agent{}, errors.New("no Claude Code or Codex agent found on PATH")
}

func lookupAgent(id string) (Agent, bool) {
	switch id {
	case "claude":
		if bin := resolveBin("claude", "LINER_CLAUDE_BIN"); bin != "" {
			return Agent{ID: "claude", Name: "Claude Code", Bin: bin}, true
		}
	case "codex":
		if bin := resolveBin("codex", "LINER_CODEX_BIN"); bin != "" {
			return Agent{ID: "codex", Name: "OpenAI Codex", Bin: bin}, true
		}
	}
	return Agent{}, false
}

func resolveBin(name string, envVar string) string {
	if override := strings.TrimSpace(os.Getenv(envVar)); override != "" {
		return override
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

func configuredAgentID() string {
	configPath := filepath.Join(os.Getenv("HOME"), ".liner", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "agent:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "agent:"))
			value = strings.Trim(value, `"'`)
			if value == "claude" || value == "codex" {
				return value
			}
			return ""
		}
	}
	return ""
}

func runClarifyAgent(ctx context.Context, agent Agent, cwd string, prompt string) (string, error) {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	var args []string
	switch agent.ID {
	case "claude":
		args = []string{"-p", "--output-format", "stream-json", "--verbose"}
	case "codex":
		args = []string{"exec", "--cd", cwd, "--skip-git-repo-check", "-s", "workspace-write", "--json", "-"}
	default:
		return "", fmt.Errorf("unsupported agent %q", agent.ID)
	}

	cmd := exec.CommandContext(ctx, agent.Bin, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "MAX_THINKING_TOKENS=0")
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w: %s", agent.Name, err, strings.TrimSpace(string(out)))
	}
	return extractAgentText(agent.ID, string(out)), nil
}

func extractAgentText(agentID string, output string) string {
	var chunks []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			chunks = append(chunks, line)
			continue
		}
		switch agentID {
		case "claude":
			chunks = append(chunks, extractClaudeText(obj)...)
		case "codex":
			chunks = append(chunks, extractCodexText(obj)...)
		}
	}
	return strings.Join(chunks, "\n")
}

func extractClaudeText(obj map[string]any) []string {
	if obj["type"] == "result" {
		if result, ok := obj["result"].(string); ok && strings.TrimSpace(result) != "" {
			return []string{result}
		}
	}
	if obj["type"] != "assistant" {
		return nil
	}
	message, _ := obj["message"].(map[string]any)
	content, _ := message["content"].([]any)
	var chunks []string
	for _, item := range content {
		block, _ := item.(map[string]any)
		if block["type"] == "text" {
			if text, ok := block["text"].(string); ok && strings.TrimSpace(text) != "" {
				chunks = append(chunks, text)
			}
		}
	}
	return chunks
}

func extractCodexText(obj map[string]any) []string {
	if obj["type"] != "item.completed" {
		return nil
	}
	item, _ := obj["item"].(map[string]any)
	if item["type"] != "agent_message" {
		return nil
	}
	if text, ok := item["text"].(string); ok && strings.TrimSpace(text) != "" {
		return []string{text}
	}
	return nil
}

func buildClarifyPrompt(jtbd string) string {
	return strings.Join([]string{
		"You are helping a curator sharpen a plain-language capability goal for a Liner project.",
		"",
		"# The capability goal",
		jtbd,
		"",
		"# Your task",
		"",
		"Generate 3-4 targeted questions whose answers will help Liner build a hyper-specific, source-grounded resource for a future AI agent. The user should not have to name research lanes, source categories, or formal JTBD syntax. Ask only what is needed to infer those.",
		"",
		"Ask questions that are specific to this capability. Do not use a generic checklist. Prefer questions that clarify:",
		"- what future AI sessions should produce, decide, critique, translate, or prevent",
		"- the narrow niche or situation where the resource should be excellent",
		"- domain constraints, risk boundaries, regulated contexts, or things the agent must avoid",
		"- quality anchors only when the user already hinted at examples, people, products, or styles they care about",
		"- whether the future agent should ask follow-up questions at runtime or act autonomously from the request and corpus",
		"",
		"If the goal involves images, moodboards, references, examples, inspiration, style, art direction, visual language, or translating one medium/domain into another output, include at least one question that clarifies the runtime inputs and output contract. Ask what kinds of references the future agent may receive and what the caller needs back (for example: observations, interpretation, carry-forward rules, implementation vocabulary, avoid-list, or clarification questions). This is not asking the user to name research sources; it is clarifying the job the corpus must support.",
		"",
		"If the goal names multiple domains, ask how they relate in the capability. Do not ask the user \"what sources should we gather\" or \"what should Liner extract\"; Liner infers that from the capability.",
		"",
		"# Output format",
		"",
		"Respond with ONLY a JSON array of strings. No prose. No markdown code fences. Example:",
		"",
		"[\"First question?\", \"Second question?\", \"Third question?\"]",
		"",
		"Do not use tools. Do not write to disk.",
	}, "\n")
}

func ParseClarifyingQuestions(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	if match := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```").FindStringSubmatch(body); len(match) == 2 {
		body = strings.TrimSpace(match[1])
	}
	if start := strings.Index(body, "["); start >= 0 {
		if end := strings.LastIndex(body, "]"); end > start {
			body = body[start : end+1]
		}
	}

	var parsed []any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil
	}
	questions := make([]string, 0, len(parsed))
	for _, item := range parsed {
		switch value := item.(type) {
		case string:
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				questions = append(questions, trimmed)
			}
		case map[string]any:
			if raw, ok := value["question"].(string); ok {
				if trimmed := strings.TrimSpace(raw); trimmed != "" {
					questions = append(questions, trimmed)
				}
			}
		}
	}
	return questions
}
