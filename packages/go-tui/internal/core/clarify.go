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

	"github.com/cmdux/liner/packages/go-tui/internal/airunner"
)

type Agent = airunner.Descriptor

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

	failures := []error{}
	for attempt := 1; attempt <= 2; attempt++ {
		questions, err := generateClarifyingQuestionsAttempt(agent, cwd, prompt, timeout)
		if err == nil {
			return questions, nil
		}
		failures = append(failures, err)
		if clarifyExplicitEffortRejection(agent, err) {
			if agent.EffortIsAuto && attempt == 1 {
				agent.Model = ""
				agent.ReasoningEffort = ""
				agent.ModelIsAuto = false
				agent.EffortIsAuto = false
				continue
			}
			break
		}
		if clarifyExplicitModelRejection(agent, err) {
			if agent.ModelIsAuto && attempt == 1 {
				agent.Model = ""
				agent.ModelIsAuto = false
				if agent.EffortIsAuto {
					agent.ReasoningEffort = ""
					agent.EffortIsAuto = false
				}
				continue
			}
			break
		}
	}
	return nil, clarifyFailure(agent, failures)
}

func generateClarifyingQuestionsAttempt(agent Agent, cwd string, prompt string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := runClarifyAgent(ctx, agent, cwd, prompt)
	if ctx.Err() == context.DeadlineExceeded {
		cause := fmt.Errorf("%s did not return clarification questions before timeout", agent.Name)
		if err != nil {
			var attempt clarifyAttemptError
			if errors.As(err, &attempt) {
				return nil, clarifyAttemptError{kind: clarifyFailureTimeout, cause: cause, output: attempt.output, logPath: attempt.logPath}
			}
		}
		return nil, clarifyAttemptError{kind: clarifyFailureTimeout, cause: cause}
	}
	if err != nil {
		return nil, err
	}
	questions := ParseClarifyingQuestions(response.text)
	if len(questions) == 0 {
		return nil, clarifyAttemptError{
			kind:    clarifyFailureResponse,
			cause:   fmt.Errorf("%s did not return a JSON array of questions", agent.Name),
			output:  strings.TrimSpace(response.raw),
			logPath: response.logPath,
		}
	}
	if len(questions) > 6 {
		questions = questions[:6]
	}
	return questions, nil
}

func resolveAgent() (Agent, error) {
	agent, err := airunner.Resolve()
	if err != nil {
		return Agent{}, err
	}
	return airunner.ApplyAutoModelPolicy(agent, "jtbd-clarify"), nil
}

type clarifyAgentResponse struct {
	text    string
	raw     string
	logPath string
}

func runClarifyAgent(ctx context.Context, agent Agent, cwd string, prompt string) (clarifyAgentResponse, error) {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	var args []string
	switch agent.ID {
	case "claude":
		args = []string{"-p", "--output-format", "stream-json", "--verbose"}
	case "codex":
		args = []string{"exec", "--cd", cwd, "--skip-git-repo-check", "-s", "workspace-write", "--json"}
	default:
		return clarifyAgentResponse{}, fmt.Errorf("unsupported agent %q", agent.ID)
	}
	if agent.Model != "" {
		args = append(args, "--model", agent.Model)
	}
	if agent.ID == "codex" && agent.ReasoningEffort != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", agent.ReasoningEffort))
	}
	if agent.ID == "codex" {
		args = append(args, "-")
	}

	startedAt := time.Now().UTC()
	cmd := exec.CommandContext(ctx, agent.Bin, args...)
	cmd.Dir = cwd
	cmd.Env = airunner.Environment(os.Environ(), agent)
	if agent.ID == "claude" {
		cmd.Env = append(cmd.Env, "MAX_THINKING_TOKENS=0")
	}
	cmd.Stdin = strings.NewReader(prompt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	logPath, logErr := writeClarifyRunLog(cwd, agent, startedAt, stdout.Bytes(), stderr.Bytes(), err)
	detail := strings.TrimSpace(strings.Join(nonEmptyStrings(stdout.String(), stderr.String()), "\n"))
	if logErr != nil {
		return clarifyAgentResponse{}, clarifyAttemptError{kind: clarifyFailureLog, cause: logErr, output: detail}
	}
	if err != nil {
		return clarifyAgentResponse{}, clarifyAttemptError{kind: clarifyFailureInvocation, cause: err, output: detail, logPath: logPath}
	}
	raw := stdout.String()
	return clarifyAgentResponse{text: extractAgentText(agent.ID, raw), raw: raw, logPath: logPath}, nil
}

type clarifyFailureKind string

const (
	clarifyFailureInvocation clarifyFailureKind = "invocation"
	clarifyFailureResponse   clarifyFailureKind = "response"
	clarifyFailureTimeout    clarifyFailureKind = "timeout"
	clarifyFailureLog        clarifyFailureKind = "log"
)

type clarifyAttemptError struct {
	kind    clarifyFailureKind
	cause   error
	output  string
	logPath string
}

func (e clarifyAttemptError) Error() string { return e.cause.Error() }

func (e clarifyAttemptError) Unwrap() error { return e.cause }

func clarifyFailure(agent Agent, failures []error) error {
	message := clarifyPrimaryFailure(agent, failures)
	logDir := ""
	for _, failure := range failures {
		var attempt clarifyAttemptError
		if errors.As(failure, &attempt) && attempt.logPath != "" {
			logDir = filepath.Dir(attempt.logPath)
		}
	}
	if logDir != "" {
		return errors.New(message + " Full logs: " + logDir)
	}
	return errors.New(message)
}

func clarifyPrimaryFailure(agent Agent, failures []error) string {
	// Effort errors often contain the token "model_reasoning_effort", which
	// also resembles a generic model error. Classify the more specific native
	// setting first so Liner gives the curator the correct recovery action.
	if agent.ReasoningEffort != "" {
		for _, failure := range failures {
			if clarifyExplicitEffortRejection(agent, failure) {
				return fmt.Sprintf("%s rejected configured Thinking effort %q. Choose another effort in Settings; Liner did not substitute another effort or model.", agent.Name, agent.ReasoningEffort)
			}
		}
	}
	if agent.Model != "" {
		for _, failure := range failures {
			if clarifyExplicitModelRejection(agent, failure) {
				return fmt.Sprintf("%s rejected configured model %q. Choose another model in Settings; Liner did not substitute another model.", agent.Name, agent.Model)
			}
		}
	}
	for _, failure := range failures {
		detail := strings.ToLower(clarifyFailureDetail(failure))
		if strings.Contains(detail, "unauthorized") || strings.Contains(detail, "missing bearer") ||
			strings.Contains(detail, "status 401") || strings.Contains(detail, "authentication failed") ||
			strings.Contains(detail, "authentication required") || strings.Contains(detail, "not logged in") ||
			strings.Contains(detail, "invalid api key") || strings.Contains(detail, "authentication token expired") {
			homeEnv := "CODEX_HOME"
			login := "login"
			if agent.ID == "claude" {
				homeEnv = "CLAUDE_CONFIG_DIR"
				login = "auth login"
			}
			return fmt.Sprintf("%s authentication is not ready for the configured runner profile. Verify %s in Settings, run %s %s for that profile, then retry.", agent.Name, homeEnv, agent.Bin, login)
		}
	}
	if len(failures) == 0 {
		return agent.Name + " could not generate Clarify Job questions. Retry from Settings."
	}
	var attempt clarifyAttemptError
	if !errors.As(failures[len(failures)-1], &attempt) {
		return agent.Name + " could not generate Clarify Job questions. Verify the runner in Settings, then retry."
	}
	switch attempt.kind {
	case clarifyFailureInvocation:
		return fmt.Sprintf("%s invocation failed (%s). Verify the executable and CLI version in Settings, then retry.", agent.Name, attempt.cause)
	case clarifyFailureResponse:
		return agent.Name + " returned an invalid Clarify Job response. Retry; if it repeats, inspect the full runner logs."
	case clarifyFailureTimeout:
		return agent.Name + " timed out before returning Clarify Job questions. Check the runner connection, then retry."
	case clarifyFailureLog:
		return fmt.Sprintf("Liner could not write the private Clarify Job run log (%s). Check project-folder permissions, then retry.", attempt.cause)
	default:
		return agent.Name + " could not generate Clarify Job questions. Verify the runner in Settings, then retry."
	}
}

func clarifyExplicitModelRejection(agent Agent, failure error) bool {
	if strings.TrimSpace(agent.Model) == "" || failure == nil {
		return false
	}
	detail := strings.ToLower(clarifyFailureDetail(failure))
	model := strings.ToLower(strings.TrimSpace(agent.Model))
	mentionsModel := strings.Contains(detail, model) || strings.Contains(detail, "model")
	if !mentionsModel {
		return false
	}
	for _, signature := range []string{
		"model not found",
		"model_not_found",
		"unknown model",
		"unsupported model",
		"invalid model",
		"does not exist",
		"not have access to model",
		"do not have access to model",
		"not available for this account",
	} {
		if strings.Contains(detail, signature) {
			return true
		}
	}
	return false
}

func clarifyExplicitEffortRejection(agent Agent, failure error) bool {
	if strings.TrimSpace(agent.ReasoningEffort) == "" || failure == nil {
		return false
	}
	detail := strings.ToLower(clarifyFailureDetail(failure))
	mentionsEffort := strings.Contains(detail, "reasoning effort") ||
		strings.Contains(detail, "reasoning_effort") ||
		strings.Contains(detail, "model_reasoning_effort")
	if !mentionsEffort {
		return false
	}
	for _, signature := range []string{"invalid", "unsupported", "unknown", "not supported", "does not support", "not available", "unrecognized"} {
		if strings.Contains(detail, signature) {
			return true
		}
	}
	return false
}

func writeClarifyRunLog(cwd string, agent Agent, startedAt time.Time, stdout []byte, stderr []byte, runErr error) (string, error) {
	dir := filepath.Join(cwd, ".liner-runs", "jtbd-clarify")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stamp := strings.ReplaceAll(startedAt.Format("2006-01-02T15-04-05.000000000Z"), ":", "-")
	path := filepath.Join(dir, stamp+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(map[string]any{
		"type":      "_liner_meta",
		"taskLabel": "jtbd-clarify",
		"agent":     agent.ID,
		"resume":    false,
		"startedAt": startedAt.Format(time.RFC3339Nano),
	}); err != nil {
		return "", err
	}
	if err := writeClarifyJSONLLines(file, encoder, stdout, "_liner_raw"); err != nil {
		return "", err
	}
	if err := writeClarifyJSONLLines(file, encoder, stderr, "_liner_stderr"); err != nil {
		return "", err
	}
	var exitCode any = 0
	stderrBytes := len(stderr)
	if runErr != nil {
		exitCode = nil
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			exitCode = exitError.ExitCode()
		}
	}
	if err := encoder.Encode(map[string]any{
		"type":        "_liner_close",
		"exitCode":    exitCode,
		"stderrBytes": stderrBytes,
		"endedAt":     time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return "", err
	}
	return path, nil
}

func writeClarifyJSONLLines(file *os.File, encoder *json.Encoder, raw []byte, fallbackType string) error {
	lines := bytes.Split(raw, []byte("\n"))
	for index, line := range lines {
		if index == len(lines)-1 && len(line) == 0 {
			continue
		}
		if json.Valid(line) {
			if _, err := file.Write(line); err != nil {
				return err
			}
			if _, err := file.WriteString("\n"); err != nil {
				return err
			}
			continue
		}
		if err := encoder.Encode(map[string]any{"type": fallbackType, "text": string(line)}); err != nil {
			return err
		}
	}
	return nil
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func clarifyFailureDetail(err error) string {
	var attempt clarifyAttemptError
	if errors.As(err, &attempt) {
		if attempt.output == "" {
			return attempt.cause.Error()
		}
		return attempt.cause.Error() + ": " + attempt.output
	}
	return err.Error()
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
