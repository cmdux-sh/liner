package agent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	NodeBin    string
	ScriptPath string
}

type RunArgs struct {
	Project   string
	PhaseID   string
	Agent     string
	SkillPath string
	Resume    bool
}

type Run struct {
	Events <-chan Event
	Done   <-chan error
}

func (r Runner) CommandArgs(args RunArgs) ([]string, error) {
	if strings.TrimSpace(r.ScriptPath) == "" {
		return nil, fmt.Errorf("headless runner script path is required")
	}
	if strings.TrimSpace(args.Project) == "" {
		return nil, fmt.Errorf("project path is required")
	}
	if strings.TrimSpace(args.PhaseID) == "" {
		return nil, fmt.Errorf("phase id is required")
	}
	agent := strings.TrimSpace(args.Agent)
	if agent == "" {
		agent = "auto"
	}
	out := []string{
		r.ScriptPath,
		"--project", args.Project,
		"--phase", args.PhaseID,
		"--agent", agent,
	}
	if strings.TrimSpace(args.SkillPath) != "" {
		out = append(out, "--skill-path", args.SkillPath)
	}
	if args.Resume {
		out = append(out, "--resume")
	}
	return out, nil
}

func (r Runner) Start(ctx context.Context, args RunArgs) (Run, error) {
	node := strings.TrimSpace(r.NodeBin)
	if node == "" {
		node = "node"
	}
	commandArgs, err := r.CommandArgs(args)
	if err != nil {
		return Run{}, err
	}

	cmd := exec.CommandContext(ctx, node, commandArgs...)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = 5 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Run{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Run{}, err
	}
	if err := cmd.Start(); err != nil {
		return Run{}, err
	}

	events := make(chan Event, 16)
	done := make(chan error, 1)
	var stderrBuf bytes.Buffer
	var stderrWG sync.WaitGroup

	stderrWG.Add(1)
	go func() {
		defer stderrWG.Done()
		_, _ = io.Copy(&stderrBuf, stderr)
	}()

	go func() {
		defer close(events)
		scanErr := scanEvents(ctx, stdout, events)
		waitErr := cmd.Wait()
		stderrWG.Wait()
		if scanErr != nil {
			done <- scanErr
			close(done)
			return
		}
		if waitErr != nil {
			stderrText := strings.TrimSpace(stderrBuf.String())
			if stderrText != "" {
				done <- fmt.Errorf("%w: %s", waitErr, stderrText)
			} else {
				done <- waitErr
			}
			close(done)
			return
		}
		done <- nil
		close(done)
	}()

	return Run{Events: events, Done: done}, nil
}

func scanEvents(ctx context.Context, stdout io.Reader, events chan Event) error {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		event, err := DecodeEvent(scanner.Bytes())
		if err != nil {
			return err
		}
		select {
		case events <- event:
		case <-ctx.Done():
			if isTerminalRunnerEvent(event) {
				deliverTerminalRunnerEvent(events, event)
			}
		}
	}
	return scanner.Err()
}

func isTerminalRunnerEvent(event Event) bool {
	switch event.Kind {
	case "runner_cancelled", "runner_failure", "runner_error", "runner_done":
		return true
	default:
		return false
	}
}

func deliverTerminalRunnerEvent(events chan Event, event Event) {
	if cap(events) == 0 {
		select {
		case events <- event:
		default:
		}
		return
	}
	for {
		select {
		case events <- event:
			return
		default:
		}
		// The detached UI no longer needs queued progress. Evict the oldest
		// buffered item so terminal outcomes always retain a route across the
		// bridge without blocking process shutdown.
		select {
		case <-events:
		default:
		}
	}
}
