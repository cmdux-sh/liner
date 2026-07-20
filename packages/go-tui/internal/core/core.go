package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProjectSummary struct {
	Path        string   `json:"path"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Curator     string   `json:"curator"`
	Mode        *string  `json:"mode"`
	JTBD        *string  `json:"jtbd"`
	Tags        []string `json:"tags"`
	SourceCount int      `json:"source_count"`
	ModifiedISO string   `json:"modified_iso"`
}

type ProjectStatus struct {
	Progress     StatusProgress     `json:"progress"`
	Phases       []StatusPhase      `json:"phases"`
	Snapshot     StatusSnapshot     `json:"status_snapshot"`
	ProjectSkill ProjectSkillStatus `json:"project_skill"`
}

type StatusProgress struct {
	Step         int     `json:"step"`
	Total        int     `json:"total"`
	CurrentPhase *string `json:"current_phase"`
	Source       string  `json:"source"`
	LastTouched  *string `json:"last_touched"`
	Error        *string `json:"error"`
}

type StatusPhase struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Index    int             `json:"index"`
	Status   string          `json:"status"`
	Artifact *StatusArtifact `json:"artifact"`
	Runs     StatusRuns      `json:"runs"`
	Gate     *StatusGate     `json:"gate,omitempty"`
}

type StatusArtifact struct {
	Path           string `json:"path"`
	Exists         bool   `json:"exists"`
	Bytes          int    `json:"bytes"`
	HasRealContent bool   `json:"has_real_content"`
}

type StatusRuns struct {
	Count          int     `json:"count"`
	LatestExitCode *int    `json:"latest_exit_code"`
	LatestLogPath  *string `json:"latest_log_path"`
}

type StatusGate struct {
	Key      string `json:"key"`
	Accepted bool   `json:"accepted"`
}

type StatusSnapshot struct {
	Milestone      string                       `json:"milestone"`
	Stale          bool                         `json:"stale"`
	Updated        string                       `json:"updated"`
	Corpus         StatusSnapshotEvidence       `json:"corpus"`
	OperatingLayer StatusSnapshotOperatingLayer `json:"operating_layer"`
}

type StatusSnapshotEvidence struct {
	State    string `json:"state"`
	Evidence string `json:"evidence"`
}

type StatusSnapshotOperatingLayer struct {
	State    string  `json:"state"`
	Evidence string  `json:"evidence"`
	Audit    *string `json:"audit,omitempty"`
}

type ProjectSkillStatus struct {
	Status string  `json:"status"`
	Name   *string `json:"name,omitempty"`
	Path   *string `json:"path,omitempty"`
}

type CompileEvent struct {
	Type      string                `json:"type"`
	Total     int                   `json:"total,omitempty"`
	Spec      *CompileSourceSpec    `json:"spec,omitempty"`
	URL       string                `json:"url,omitempty"`
	Title     *string               `json:"title,omitempty"`
	Message   string                `json:"message,omitempty"`
	Severity  string                `json:"severity,omitempty"`
	BodyChars int                   `json:"body_chars,omitempty"`
	Summary   *CompileSummary       `json:"summary,omitempty"`
	Payload   *CompileResultPayload `json:"payload,omitempty"`
}

type CompileSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type CompileSourceSpec struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	Note     *string `json:"note,omitempty"`
	Section  *string `json:"section,omitempty"`
	Priority string  `json:"priority,omitempty"`
}

type CompileResultPayload struct {
	MixtapePath string                  `json:"mixtape_path"`
	Sources     []CompiledSourceRecord  `json:"sources"`
	Warnings    []CompileWarningPayload `json:"warnings"`
	Summary     CompileSummary          `json:"summary"`
}

type CompiledSourceRecord struct {
	Index     int     `json:"index"`
	Filename  string  `json:"filename"`
	Path      string  `json:"path"`
	URL       string  `json:"url"`
	Type      string  `json:"type"`
	Section   *string `json:"section"`
	Title     *string `json:"title"`
	Succeeded bool    `json:"succeeded"`
}

type CompileWarningPayload struct {
	URL      string `json:"url"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type CompileExitError struct {
	Code   int
	Stderr string
	Err    error
}

func (e CompileExitError) Error() string {
	message := strings.TrimSpace(e.Stderr)
	if message == "" {
		message = fmt.Sprintf("compile exited with code %d", e.Code)
	}
	return message
}

func (e CompileExitError) Unwrap() error {
	return e.Err
}

type Runner struct {
	Command string
	Args    []string
}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func Resolve() (Runner, error) {
	if env := os.Getenv("LINER_BIN"); env != "" {
		if fileExists(env) {
			return Runner{Command: env}, nil
		}
	}
	if bundled := bundledBinary(); bundled != "" {
		return Runner{Command: bundled}, nil
	}
	if venv := repoVenvBinary(); venv != "" {
		return Runner{Command: venv}, nil
	}
	if path, err := exec.LookPath("liner"); err == nil {
		return Runner{Command: path}, nil
	}
	return Runner{}, errors.New("could not find liner core binary; set LINER_BIN=/path/to/liner")
}

func (r Runner) Run(args ...string) ([]byte, error) {
	cmd := exec.Command(r.Command, append(r.Args, args...)...)
	cmd.Env = append(os.Environ(), "LINER_TUI_SHIM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func (r Runner) ListProjects(baseDir string) ([]ProjectSummary, error) {
	out, err := r.Run("list", "--json", "--dir", baseDir)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(out)) == "" {
		return []ProjectSummary{}, nil
	}
	var projects []ProjectSummary
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (r Runner) InitProject(path string) error {
	_, err := r.Run("init", path)
	return err
}

// InitProjectWithMetadata keeps first-run Project creation inside Liner Core.
// The TUI passes user-reviewed setup values as CLI arguments and never rewrites
// the canonical tape after Core creates it.
func (r Runner) InitProjectWithMetadata(path string, title string, description string, curator string, jtbd string) error {
	_, err := r.Run(
		"init", path,
		"--title", title,
		"--description", description,
		"--curator", curator,
		"--jtbd", jtbd,
		"--tui-construction",
	)
	return err
}

func (r Runner) SetupJS() error {
	_, err := r.Run("setup-js", "--yes")
	return err
}

func (r Runner) ProjectStatus(path string) (ProjectStatus, error) {
	out, err := r.Run("status", path, "--json", "--no-write")
	if err != nil {
		return ProjectStatus{}, err
	}
	return parseProjectStatus(out)
}

// RefreshProjectStatus asks Core to update only the durable Status Snapshot.
// Unlike ProjectStatus, this deliberately omits --no-write.
func (r Runner) RefreshProjectStatus(path string) (ProjectStatus, error) {
	out, err := r.Run("status", path, "--json", "--status-only")
	if err != nil {
		return ProjectStatus{}, err
	}
	return parseProjectStatus(out)
}

func parseProjectStatus(out []byte) (ProjectStatus, error) {
	var status ProjectStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return ProjectStatus{}, err
	}
	return status, nil
}

func (r Runner) Share(path string) (string, error) {
	out, err := r.Run("share", path)
	if err != nil {
		return "", err
	}
	return sharedArchivePath(string(out), path), nil
}

func (r Runner) ImportArchive(archive string, destination string, noRefetch bool) (string, error) {
	args := []string{"import", archive, destination}
	if noRefetch {
		args = append(args, "--no-refetch")
	}
	out, err := r.Run(args...)
	if err != nil {
		return "", err
	}
	return importedProjectPath(string(out), archive, destination), nil
}

func importedProjectPath(output string, archive string, destination string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(ansiEscapePattern.ReplaceAllString(line, ""))
		if after, ok := strings.CutPrefix(line, "Extracted "); ok {
			return strings.TrimSpace(after)
		}
	}
	name := strings.TrimSuffix(filepath.Base(archive), filepath.Ext(archive))
	return filepath.Join(destination, name)
}

func sharedArchivePath(output string, project string) string {
	cleaned := strings.TrimSpace(ansiEscapePattern.ReplaceAllString(output, ""))
	flattened := strings.Join(strings.Fields(cleaned), " ")
	if match := regexp.MustCompile(`\bWrote\s+(.+?\.mixtape)(?:\s+\(|$)`).FindStringSubmatch(flattened); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	name := strings.TrimSuffix(filepath.Base(project), filepath.Ext(project))
	return filepath.Join(filepath.Dir(project), name+".mixtape")
}

func (r Runner) CompileStream(path string, send func(CompileEvent)) error {
	args := append(r.Args, "compile", path, "--emit-events")
	cmd := exec.Command(r.Command, args...)
	cmd.Env = append(os.Environ(), "LINER_TUI_SHIM=1", "PYTHONUNBUFFERED=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	errText := new(strings.Builder)
	errDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(errText, stderr)
		errDone <- copyErr
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event CompileEvent
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			send(event)
		}
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	err = cmd.Wait()
	if copyErr := <-errDone; copyErr != nil {
		return copyErr
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return CompileExitError{
				Code:   exitErr.ExitCode(),
				Stderr: strings.TrimSpace(errText.String()),
				Err:    err,
			}
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(errText.String()))
	}
	return nil
}

func (r Runner) StartCompile(path string) (<-chan CompileEvent, <-chan error) {
	events := make(chan CompileEvent)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		done <- r.CompileStream(path, func(event CompileEvent) {
			events <- event
		})
	}()
	return events, done
}

func DefaultBaseDir() string {
	if env := normalizeProjectDir(os.Getenv("LINER_DIR")); env != "" {
		return env
	}
	if configured := configuredProjectsDir(); configured != "" {
		return configured
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if home == "" {
		return filepath.Join("liner", "projects")
	}
	return filepath.Join(home, "liner", "projects")
}

func configuredProjectsDir() string {
	data, err := os.ReadFile(linerConfigPath())
	if err != nil {
		return ""
	}
	raw := map[string]any{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ""
	}
	projectsDir, _ := raw["projects_dir"].(string)
	return normalizeProjectDir(projectsDir)
}

func linerConfigPath() string {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".liner", "config.yaml")
}

func normalizeProjectDir(path string) string {
	path = expandUserPath(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func expandUserPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home := os.Getenv("HOME")
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		if home == "" {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func bundledBinary() string {
	// Packaged Go TUI lives beside the npm shim; the core binary still comes
	// from optional platform packages. The Go TUI can run standalone in dev,
	// so bundled lookup here is intentionally conservative.
	return ""
}

func repoVenvBinary() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	binDir := "bin"
	if runtime.GOOS == "windows" {
		binDir = "Scripts"
	}
	dir := cwd
	for i := 0; i < 12; i++ {
		candidate := filepath.Join(dir, ".venv", binDir, exeName("liner"))
		if fileExists(candidate) {
			return candidate
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return ""
}

func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
