// Package navigator is the terminal app: a live, read-only view of your library.
// Agents do the work. This reads what they leave behind.
package navigator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

const coreTimeout = 30 * time.Second

var escapedControl = regexp.MustCompile(`\\u00(0[0-8bcefBCEF]|1[0-9a-fA-F]|7[fF]|[89][0-9a-fA-F])`)

// Library is Core's `liner library` report.
type Library struct {
	Root        string       `json:"root"`
	RootExists  bool         `json:"root_exists"`
	Liners      []Liner      `json:"liners"`
	Attention   []Attention  `json:"attention"`
	Sources     []Source     `json:"sources"`
	Connections Connections  `json:"connections"`
	Unreadable  []Unreadable `json:"unreadable"`
}

// Unreadable is a folder Core could not read.
type Unreadable struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Liner is one liner.
type Liner struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Job         string     `json:"job"`
	Path        string     `json:"path"`
	State       string     `json:"state"`
	WaitingOn   *string    `json:"waiting_on"`
	BuildStep   *int       `json:"build_step"`
	BuildSteps  int        `json:"build_steps"`
	SourceCount int        `json:"source_count"`
	OpenGaps    *int       `json:"open_gaps"`
	Pending     *int       `json:"pending_findings"`
	Test        *Tests     `json:"test"`
	Cites       []string   `json:"cites"`
	CitedBy     []string   `json:"cited_by"`
	Sources     []Source   `json:"sources"`
	Sections    []Section  `json:"sections"`
	ChangedAt   *float64   `json:"changed_at"`
	RefreshedAt *string    `json:"refreshed_at"`
	Synthesis   *Synthesis `json:"synthesis"`
	Gaps        []Gap      `json:"gaps"`
	Findings    []Finding  `json:"findings"`
	Files       []string   `json:"files"`
}

// Tests sums up a liner's with-and-without tests.
type Tests struct {
	Runs    int  `json:"runs"`
	Better  int  `json:"better"`
	Same    int  `json:"same"`
	Worse   int  `json:"worse"`
	Current bool `json:"current"`
}

// Source is one Source. In the library report, Liners says who uses it.
type Source struct {
	Locator   string   `json:"locator"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Kind      *string  `json:"kind"`
	Section   string   `json:"section"`
	Note      *string  `json:"note"`
	File      *string  `json:"file"`
	FetchedAt *string  `json:"fetched_at"`
	Changed   bool     `json:"changed"`
	Liners    []string `json:"liners"`
}

// Section counts the Sources in one part of a liner.
type Section struct {
	Name    string `json:"name"`
	Sources int    `json:"sources"`
}

// Synthesis describes a liner's synthesis.
type Synthesis struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Sections  int    `json:"sections"`
	Citations int    `json:"citations"`
	// Files maps each citation target to the file it opens, or null. Core resolves it.
	Files map[string]*string `json:"files"`
}

// Gap is a question a liner could not answer.
type Gap struct {
	ID       string `json:"id"`
	Question string `json:"question"`
}

// Finding is something real work learned, waiting for review.
type Finding struct {
	ID       string `json:"id"`
	Finding  string `json:"finding"`
	Evidence string `json:"evidence"`
}

// Attention is one thing worth a look.
type Attention struct {
	Group   string `json:"group"`
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Context string `json:"context"`
}

// Connections says how liners relate.
type Connections struct {
	Citations []Citation `json:"citations"`
	Uncited   []string   `json:"uncited"`
}

// Citation is one liner citing another.
type Citation struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Stale bool   `json:"stale"`
}

// SearchResult is one passage from `liner search`.
type SearchResult struct {
	Liner   string `json:"liner"`
	File    string `json:"file"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Heading string `json:"heading"`
	Snippet string `json:"snippet"`
}

// Prompt is one ready sentence for your agent.
type Prompt struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

// Agents says what is connected.
type Agents struct {
	Claude   string
	Codex    string
	Subagent string
}

// Client reads from Core. It only asks for reports.
type Client struct {
	Runner core.Runner
	Root   string
}

func (c Client) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), coreTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Runner.Command, append(c.Runner.Args, args...)...)
	cmd.Env = append(os.Environ(), "LINER_TUI_SHIM=1", "NO_COLOR=1")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if detail := strings.TrimSpace(cleanText(stderr.String())); detail != "" {
			return nil, errors.New(detail)
		}
		return nil, err
	}
	return cleanReport(out), nil
}

// cleanReport removes control characters from Core's JSON. Titles and snippets come
// from web pages, and an escape sequence could fake the screen or set the clipboard.
func cleanReport(report []byte) []byte {
	return []byte(cleanText(escapedControl.ReplaceAllString(string(report), "")))
}

// cleanText keeps newlines and turns tabs into spaces. Everything else that isn't
// printable goes, including the C1 range terminals also read as escapes.
func cleanText(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			if r == '\t' {
				return ' '
			}
			return -1
		}
		return r
	}, text)
}

// Library reads the whole library.
func (c Client) Library() (Library, error) {
	out, err := c.run("library", "--root", c.Root)
	if err != nil {
		return Library{}, err
	}
	var library Library
	if err := json.Unmarshal(out, &library); err != nil {
		return Library{}, fmt.Errorf("could not read the library report: %w", err)
	}
	return library, nil
}

// Search looks across the library.
func (c Client) Search(query string) ([]SearchResult, error) {
	out, err := c.run("search", "--root", c.Root, "--limit", "30", "--json", "--", query)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Results []SearchResult `json:"results"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("could not read the search results: %w", err)
	}
	return payload.Results, nil
}

// Handoff asks Core for prompts about what is on screen.
func (c Client) Handoff(context, slug string, line int, subject string) ([]Prompt, error) {
	out, err := c.run("handoff", context, "--slug", slug, "--line", strconv.Itoa(line), "--subject", subject)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Options []Prompt `json:"options"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("could not read the prompts: %w", err)
	}
	return payload.Options, nil
}

// Agents inspects what is connected. A failed check reads as unknown.
func (c Client) Agents() Agents {
	return Agents{
		Claude:   c.status("agent-skill", "inspect", "claude", "--json"),
		Codex:    c.status("agent-skill", "inspect", "codex", "--json"),
		Subagent: c.status("research", "subagent", "inspect", "--json"),
	}
}

func (c Client) status(args ...string) string {
	out, err := c.run(args...)
	if err != nil {
		return "unknown"
	}
	var payload struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(out, &payload) != nil || payload.Status == "" {
		return "unknown"
	}
	return payload.Status
}

// Connect installs the liner skill. This is the one change the app makes,
// because first run needs a way to reach an agent.
func (c Client) Connect(environment string) error {
	_, err := c.run("agent-skill", "install", environment, "--yes", "--json")
	return err
}
