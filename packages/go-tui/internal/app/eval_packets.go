package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

type evalVariant struct {
	ID      string
	Label   string
	Context string
	Notes   string
}

type evalVariantComparison struct {
	Variant    evalVariant
	RelPath    string
	Exists     bool
	TaskOutput [3]string
	WordCount  [3]int
	Score      [3]string
	Notes      [3]string
}

type evalSummaryCell struct {
	Score string
	Notes string
}

func writeEvalTaskset(project string, t tape.Tape) (evalFile, error) {
	dir := filepath.Join(project, "working", "evals", "tasksets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	base := slug(t.Title)
	if base == "" {
		base = "mixtape"
	}
	name := now.Format("2006-01-02-150405") + "-" + base + "-impact-taskset"
	rel := filepath.Join("working", "evals", "tasksets", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderEvalTaskset(now, t, project)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("tasksets", name+".md"),
		RelPath: rel,
		Path:    path,
		Area:    "taskset",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func renderEvalTaskset(now time.Time, t tape.Tape, project string) string {
	var b strings.Builder
	title := fallbackText(t.Title, "Mixtape")
	fmt.Fprintf(&b, "# Impact Test Taskset: %s\n\n", title)
	fmt.Fprintf(&b, "Date: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Purpose\n\n")
	b.WriteString("Use this opt-in local impact test to compare whether the mixtape changes answers: baseline model, corpus only, corpus plus `LINER.md`, and corpus plus reusable skills. Start with human review; add an external LLM judge only after the rubric feels right.\n\n")
	b.WriteString("## Project Context\n\n")
	fmt.Fprintf(&b, "- Title: %s\n", title)
	if jtbd := optionalString(t.JTBD); jtbd != "" {
		fmt.Fprintf(&b, "- JTBD: %s\n", jtbd)
	} else {
		b.WriteString("- JTBD: Add the job this taskset should protect.\n")
	}
	fmt.Fprintf(&b, "- Saved sources: %d\n", len(t.Sources))
	fmt.Fprintf(&b, "- `MIXTAPE.md`: %s\n", yesNo(projectFileExists(project, "MIXTAPE.md")))
	fmt.Fprintf(&b, "- `LINER.md`: %s\n", yesNo(projectFileExists(project, "LINER.md")))
	fmt.Fprintf(&b, "- `skills/*.md`: %d\n\n", countTopLevelRegularFiles(filepath.Join(project, "skills"), ".md"))
	b.WriteString("## Protocol\n\n")
	b.WriteString("- Run each variant in a fresh agent/session so context does not leak between variants.\n")
	b.WriteString("- Keep task wording identical across variants.\n")
	b.WriteString("- Paste raw outputs before scoring, comparing, or judging.\n")
	b.WriteString("- Treat missing context files as blocked variants, not as evidence that a layer failed.\n\n")
	b.WriteString("## Variants\n\n")
	b.WriteString("| Variant | Context To Load | Notes |\n")
	b.WriteString("| --- | --- | --- |\n")
	b.WriteString("| baseline | no project files | Tests what the model does without this mixtape. |\n")
	b.WriteString("| corpus | `MIXTAPE.md` only | Tests whether source grounding changes the answer. |\n")
	b.WriteString("| operating layer | `MIXTAPE.md` + `LINER.md` | Tests whether the artifact's operating rules change behavior. |\n")
	b.WriteString("| skills | `MIXTAPE.md` + `LINER.md` + `skills/*.md` | Tests whether repeatable methods improve the work. |\n\n")
	b.WriteString("## Tasks\n\n")
	b.WriteString("### Task 1: Real User Request\n\n")
	b.WriteString("Input: Paste a realistic request this mixtape should answer.\n\n")
	b.WriteString("Expected output:\n\n")
	b.WriteString("- identifies the user's job\n")
	b.WriteString("- applies specific source-backed guidance when project context is loaded\n")
	b.WriteString("- names tradeoffs or boundaries\n")
	b.WriteString("- avoids generic advice\n\n")
	b.WriteString("### Task 2: Draft Critique\n\n")
	b.WriteString("Input: Paste a weak or generic draft answer in this domain.\n\n")
	b.WriteString("Expected output:\n\n")
	b.WriteString("- finds what is unsupported by the corpus\n")
	b.WriteString("- catches contradictions with `LINER.md` or skill rules\n")
	b.WriteString("- proposes concrete revisions\n")
	b.WriteString("- cites or names the relevant source, rule, or skill\n\n")
	b.WriteString("### Task 3: Boundary Check\n\n")
	b.WriteString("Input: Ask for advice that is near the edge of this mixtape's scope.\n\n")
	b.WriteString("Expected output:\n\n")
	b.WriteString("- states what the mixtape can answer confidently\n")
	b.WriteString("- refuses or narrows claims outside the source boundary\n")
	b.WriteString("- asks for missing context when needed\n")
	b.WriteString("- suggests what source would strengthen the answer\n\n")
	b.WriteString("## Human Rubric\n\n")
	b.WriteString("Score each variant from 1-5.\n\n")
	b.WriteString("| Criterion | 1 | 3 | 5 |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	b.WriteString("| Task fit | misses the actual job | addresses the job broadly | directly serves the user's job |\n")
	b.WriteString("| Source grounding | generic or invented | some source-shaped guidance | clearly grounded in the corpus |\n")
	b.WriteString("| Operating behavior | ignores boundaries | follows some rules | behaves like the intended artifact |\n")
	b.WriteString("| Specificity | vague | partly concrete | concrete, inspectable, and actionable |\n")
	b.WriteString("| Restraint | overclaims | mostly scoped | names limits and asks for missing context |\n")
	b.WriteString("| Impact delta | indistinguishable from baseline | some useful improvement | clear useful improvement caused by the loaded layer |\n\n")
	b.WriteString("## Results Notes\n\n")
	b.WriteString("| Task | Variant | Score | Qualitative Notes |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	b.WriteString("| Task 1 | baseline |  |  |\n")
	b.WriteString("| Task 1 | corpus |  |  |\n")
	b.WriteString("| Task 1 | operating layer |  |  |\n")
	b.WriteString("| Task 1 | skills |  |  |\n")
	b.WriteString("| Task 2 | baseline |  |  |\n")
	b.WriteString("| Task 2 | corpus |  |  |\n")
	b.WriteString("| Task 2 | operating layer |  |  |\n")
	b.WriteString("| Task 2 | skills |  |  |\n")
	b.WriteString("| Task 3 | baseline |  |  |\n")
	b.WriteString("| Task 3 | corpus |  |  |\n")
	b.WriteString("| Task 3 | operating layer |  |  |\n")
	b.WriteString("| Task 3 | skills |  |  |\n")
	return b.String()
}

func writeEvalRunPacket(project string, taskset evalFile) (evalFile, error) {
	tasksetBody, err := os.ReadFile(taskset.Path)
	if err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	base := slug(strings.TrimSuffix(filepath.Base(taskset.Name), filepath.Ext(taskset.Name)))
	if base == "" {
		base = "taskset"
	}
	runID := now.Format("2006-01-02-150405") + "-" + base
	runRel := filepath.Join("working", "evals", "runs", runID)
	runDir := filepath.Join(project, runRel)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return evalFile{}, err
	}
	variants := evalRunVariants()
	readmePath := filepath.Join(runDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(renderEvalRunReadme(now, taskset, runRel, variants)), 0o644); err != nil {
		return evalFile{}, err
	}
	for _, variant := range variants {
		path := filepath.Join(runDir, variant.ID+".md")
		if err := os.WriteFile(path, []byte(renderEvalVariantRun(now, taskset, string(tasksetBody), variant)), 0o644); err != nil {
			return evalFile{}, err
		}
	}
	summaryDir := filepath.Join(project, "working", "evals", "summaries")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		return evalFile{}, err
	}
	summaryRel := filepath.Join("working", "evals", "summaries", runID+"-summary.md")
	if err := os.WriteFile(filepath.Join(project, summaryRel), []byte(renderEvalRunSummary(now, taskset, runRel, variants)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("runs", runID, "README.md"),
		RelPath: filepath.Join(runRel, "README.md"),
		Path:    readmePath,
		Area:    "run",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func evalRunVariants() []evalVariant {
	return []evalVariant{
		{
			ID:      "baseline",
			Label:   "Baseline",
			Context: "no project files",
			Notes:   "Run without loading any project files.",
		},
		{
			ID:      "corpus",
			Label:   "Corpus",
			Context: "MIXTAPE.md only",
			Notes:   "Load only the compiled corpus.",
		},
		{
			ID:      "operating-layer",
			Label:   "Operating Layer",
			Context: "MIXTAPE.md + LINER.md",
			Notes:   "Load the corpus and operating layer.",
		},
		{
			ID:      "skills",
			Label:   "Skills",
			Context: "MIXTAPE.md + LINER.md + skills/*.md",
			Notes:   "Load the corpus, operating layer, and reusable methods.",
		},
	}
}

func renderEvalRunReadme(now time.Time, taskset evalFile, runRel string, variants []evalVariant) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Run Packet\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Taskset: `%s`\n\n", taskset.RelPath)
	b.WriteString("## How To Run\n\n")
	b.WriteString("1. Open each variant file below.\n")
	b.WriteString("2. Run each variant in a fresh agent/session.\n")
	b.WriteString("3. Load exactly the context named for that variant.\n")
	b.WriteString("4. Run the same taskset prompts and paste raw outputs into the variant file.\n")
	b.WriteString("5. Score the outputs in the summary template under `working/evals/summaries/`.\n\n")
	b.WriteString("## Variants\n\n")
	b.WriteString("| Variant | File | Context | Notes |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, variant := range variants {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n",
			escapeMarkdownTable(variant.Label),
			filepath.Join(runRel, variant.ID+".md"),
			escapeMarkdownTable(variant.Context),
			escapeMarkdownTable(variant.Notes),
		)
	}
	b.WriteString("\n## Summary\n\n")
	fmt.Fprintf(&b, "Use `working/evals/summaries/%s-summary.md` for scores and qualitative notes.\n", strings.TrimPrefix(filepath.Base(runRel), string(filepath.Separator)))
	return b.String()
}

func renderEvalVariantRun(now time.Time, taskset evalFile, tasksetBody string, variant evalVariant) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Variant: %s\n\n", variant.Label)
	fmt.Fprintf(&b, "Date: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Taskset: `%s`\n", taskset.RelPath)
	fmt.Fprintf(&b, "Context to load: %s\n\n", variant.Context)
	b.WriteString("## Run Instructions\n\n")
	b.WriteString("- Load only the context listed above.\n")
	b.WriteString("- For baseline, load no project files even if the session starts inside this project folder.\n")
	b.WriteString("- Run every task in the taskset without changing the task wording.\n")
	b.WriteString("- Paste raw model output under the matching task heading.\n")
	b.WriteString("- Add human observations only in the notes section.\n\n")
	b.WriteString("## Outputs\n\n")
	b.WriteString("### Task 1 Output\n\n")
	b.WriteString("_Paste output here._\n\n")
	b.WriteString("### Task 2 Output\n\n")
	b.WriteString("_Paste output here._\n\n")
	b.WriteString("### Task 3 Output\n\n")
	b.WriteString("_Paste output here._\n\n")
	b.WriteString("## Human Notes\n\n")
	b.WriteString("- Score:\n")
	b.WriteString("- What improved:\n")
	b.WriteString("- What regressed:\n")
	b.WriteString("- Source or rule gaps:\n\n")
	b.WriteString("## Taskset Snapshot\n\n")
	b.WriteString(tasksetBody)
	if !strings.HasSuffix(tasksetBody, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func renderEvalRunSummary(now time.Time, taskset evalFile, runRel string, variants []evalVariant) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Summary\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Taskset: `%s`\n", taskset.RelPath)
	fmt.Fprintf(&b, "Run packet: `%s`\n\n", runRel)
	b.WriteString("## Score Table\n\n")
	b.WriteString("| Task | Variant | Score | Qualitative Notes |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for task := 1; task <= 3; task++ {
		for _, variant := range variants {
			fmt.Fprintf(&b, "| Task %d | %s |  |  |\n", task, escapeMarkdownTable(variant.Label))
		}
	}
	b.WriteString("\n## Comparison Notes\n\n")
	b.WriteString("- Best variant:\n")
	b.WriteString("- Weakest variant:\n")
	b.WriteString("- Largest useful delta over baseline:\n")
	b.WriteString("- Did `MIXTAPE.md` improve the answer?\n")
	b.WriteString("- Did `LINER.md` change behavior?\n")
	b.WriteString("- Did skills improve repeatability?\n")
	b.WriteString("- Source, note, skill, or boundary fixes to make next:\n")
	return b.String()
}

func writeEvalComparisonReport(project string, item evalFile) (evalFile, error) {
	runRel, ok, err := evalJudgeRunRelForItem(project, item)
	if err != nil {
		return evalFile{}, err
	}
	if !ok {
		return evalFile{}, fmt.Errorf("Select a run packet or summary before creating a comparison.")
	}
	runDir := filepath.Join(project, runRel)
	info, err := os.Stat(runDir)
	if err != nil {
		return evalFile{}, err
	}
	if !info.IsDir() {
		return evalFile{}, fmt.Errorf("Impact test run path is not a directory: %s", runRel)
	}
	summaryRel := evalSummaryRelForRun(runRel)
	scores := evalSummaryScores(filepath.Join(project, summaryRel))
	comparisons := evalVariantComparisons(project, runRel, scores)
	dir := filepath.Join(project, "working", "evals", "comparisons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	runID := filepath.Base(filepath.Clean(runRel))
	name := now.Format("2006-01-02-150405") + "-" + runID + "-comparison"
	rel := filepath.Join("working", "evals", "comparisons", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderEvalComparisonReport(now, runRel, summaryRel, comparisons)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("comparisons", name+".md"),
		RelPath: rel,
		Path:    path,
		Area:    "comparison",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeEvalReadinessReport(project string, item evalFile) (evalFile, error) {
	runRel, ok, err := evalJudgeRunRelForItem(project, item)
	if err != nil {
		return evalFile{}, err
	}
	if !ok {
		return evalFile{}, fmt.Errorf("Select an impact-test artifact linked to a run before creating a readiness report.")
	}
	runDir := filepath.Join(project, runRel)
	info, err := os.Stat(runDir)
	if err != nil {
		return evalFile{}, err
	}
	if !info.IsDir() {
		return evalFile{}, fmt.Errorf("Impact test run path is not a directory: %s", runRel)
	}
	summaryRel := evalSummaryRelForRun(runRel)
	scores := evalSummaryScores(filepath.Join(project, summaryRel))
	comparisons := evalVariantComparisons(project, runRel, scores)
	tasksetRel := evalTasksetRelForRun(project, runRel)
	dir := filepath.Join(project, "working", "evals", "readiness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	runID := filepath.Base(filepath.Clean(runRel))
	name := now.Format("2006-01-02-150405") + "-" + runID + "-readiness"
	rel := filepath.Join("working", "evals", "readiness", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderEvalReadinessReport(now, project, runRel, summaryRel, tasksetRel, item.RelPath, comparisons)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("readiness", name+".md"),
		RelPath: rel,
		Path:    path,
		Area:    "readiness",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeEvalAutomationPacket(project string, item evalFile) (evalFile, error) {
	runRel, ok, err := evalJudgeRunRelForItem(project, item)
	if err != nil {
		return evalFile{}, err
	}
	if !ok {
		return evalFile{}, fmt.Errorf("Select a run packet, summary, or comparison before creating a runner packet.")
	}
	runDir := filepath.Join(project, runRel)
	info, err := os.Stat(runDir)
	if err != nil {
		return evalFile{}, err
	}
	if !info.IsDir() {
		return evalFile{}, fmt.Errorf("Impact test run path is not a directory: %s", runRel)
	}
	summaryRel := evalSummaryRelForRun(runRel)
	scores := evalSummaryScores(filepath.Join(project, summaryRel))
	comparisons := evalVariantComparisons(project, runRel, scores)
	tasksetRel := evalTasksetRelForRun(project, runRel)
	dir := filepath.Join(project, "working", "evals", "automation")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	runID := filepath.Base(filepath.Clean(runRel))
	name := now.Format("2006-01-02-150405") + "-" + runID + "-automation-packet"
	rel := filepath.Join("working", "evals", "automation", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderEvalAutomationPacket(now, project, runRel, summaryRel, tasksetRel, item.RelPath, comparisons)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("automation", name+".md"),
		RelPath: rel,
		Path:    path,
		Area:    "automation",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func writeEvalJudgePacket(project string, item evalFile) (evalFile, error) {
	runRel, ok, err := evalJudgeRunRelForItem(project, item)
	if err != nil {
		return evalFile{}, err
	}
	if !ok {
		return evalFile{}, fmt.Errorf("Select a run packet, summary, or comparison before creating a judge packet.")
	}
	runDir := filepath.Join(project, runRel)
	info, err := os.Stat(runDir)
	if err != nil {
		return evalFile{}, err
	}
	if !info.IsDir() {
		return evalFile{}, fmt.Errorf("Impact test run path is not a directory: %s", runRel)
	}
	summaryRel := evalSummaryRelForRun(runRel)
	scores := evalSummaryScores(filepath.Join(project, summaryRel))
	comparisons := evalVariantComparisons(project, runRel, scores)
	dir := filepath.Join(project, "working", "evals", "judges")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return evalFile{}, err
	}
	now := time.Now()
	runID := filepath.Base(filepath.Clean(runRel))
	name := now.Format("2006-01-02-150405") + "-" + runID + "-judge-packet"
	rel := filepath.Join("working", "evals", "judges", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderEvalJudgePacket(now, runRel, summaryRel, item.RelPath, comparisons)), 0o644); err != nil {
		return evalFile{}, err
	}
	return evalFile{
		Name:    filepath.Join("judges", name+".md"),
		RelPath: rel,
		Path:    path,
		Area:    "judge",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func evalJudgeRunRelForItem(project string, item evalFile) (string, bool, error) {
	if runRel, ok := evalRunRelForItem(item); ok {
		return runRel, true, nil
	}
	path := item.Path
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(project, item.RelPath)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	if runRel, ok := evalRunRelFromMarkdown(string(body)); ok {
		return runRel, true, nil
	}
	return "", false, nil
}

func evalRunRelFromMarkdown(body string) (string, bool) {
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Run packet:") {
			continue
		}
		value := markdownBacktickValue(line)
		clean := filepath.Clean(filepath.FromSlash(value))
		slash := filepath.ToSlash(clean)
		if strings.HasPrefix(slash, "working/evals/runs/") {
			return clean, true
		}
	}
	return "", false
}

func evalTasksetRelForRun(project string, runRel string) string {
	body, err := os.ReadFile(filepath.Join(project, runRel, "README.md"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Taskset:") {
			continue
		}
		value := filepath.Clean(filepath.FromSlash(markdownBacktickValue(line)))
		slash := filepath.ToSlash(value)
		if value == "." || strings.HasPrefix(slash, "../") || filepath.IsAbs(value) {
			return ""
		}
		return value
	}
	return ""
}

func markdownBacktickValue(line string) string {
	start := strings.Index(line, "`")
	if start < 0 {
		return ""
	}
	rest := line[start+1:]
	end := strings.Index(rest, "`")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

func evalRunRelForItem(item evalFile) (string, bool) {
	rel := filepath.ToSlash(item.RelPath)
	const runPrefix = "working/evals/runs/"
	if strings.HasPrefix(rel, runPrefix) {
		rest := strings.TrimPrefix(rel, runPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return filepath.Join("working", "evals", "runs", parts[0]), true
		}
	}
	const summaryPrefix = "working/evals/summaries/"
	if strings.HasPrefix(rel, summaryPrefix) {
		name := filepath.Base(rel)
		if strings.HasSuffix(name, "-summary.md") {
			runID := strings.TrimSuffix(name, "-summary.md")
			if runID != "" {
				return filepath.Join("working", "evals", "runs", runID), true
			}
		}
	}
	return "", false
}

func evalSummaryRelForRun(runRel string) string {
	return filepath.Join("working", "evals", "summaries", filepath.Base(filepath.Clean(runRel))+"-summary.md")
}

func evalVariantComparisons(project string, runRel string, scores map[string]evalSummaryCell) []evalVariantComparison {
	variants := evalRunVariants()
	comparisons := make([]evalVariantComparison, 0, len(variants))
	for _, variant := range variants {
		rel := filepath.Join(runRel, variant.ID+".md")
		comparison := evalVariantComparison{Variant: variant, RelPath: rel}
		body, err := os.ReadFile(filepath.Join(project, rel))
		if err == nil {
			comparison.Exists = true
			for task := 1; task <= 3; task++ {
				output := evalTaskOutput(string(body), task)
				comparison.TaskOutput[task-1] = output
				comparison.WordCount[task-1] = evalWordCount(output)
			}
		}
		for task := 1; task <= 3; task++ {
			if cell, ok := scores[evalSummaryKey(task, variant.Label)]; ok {
				comparison.Score[task-1] = cell.Score
				comparison.Notes[task-1] = cell.Notes
			}
		}
		comparisons = append(comparisons, comparison)
	}
	return comparisons
}

func evalTaskOutput(body string, task int) string {
	marker := fmt.Sprintf("### Task %d Output", task)
	start := strings.Index(body, marker)
	if start < 0 {
		return ""
	}
	after := body[start+len(marker):]
	end := len(after)
	for _, next := range []string{"\n### Task ", "\n## Human Notes", "\n## Taskset Snapshot"} {
		if index := strings.Index(after, next); index >= 0 && index < end {
			end = index
		}
	}
	output := strings.TrimSpace(after[:end])
	output = strings.TrimPrefix(output, "\n")
	output = strings.TrimSpace(strings.ReplaceAll(output, "_Paste output here._", ""))
	return output
}

func evalSummaryScores(path string) map[string]evalSummaryCell {
	body, err := os.ReadFile(path)
	if err != nil {
		return map[string]evalSummaryCell{}
	}
	scores := map[string]evalSummaryCell{}
	for _, line := range strings.Split(string(body), "\n") {
		cells := evalMarkdownTableCells(line)
		if len(cells) < 4 || !strings.HasPrefix(strings.ToLower(cells[0]), "task ") {
			continue
		}
		task := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(cells[0]), "task"))
		if task == "" {
			continue
		}
		scores[evalSummaryKeyString(task, cells[1])] = evalSummaryCell{
			Score: strings.TrimSpace(cells[2]),
			Notes: strings.TrimSpace(cells[3]),
		}
	}
	return scores
}

func evalMarkdownTableCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func evalSummaryKey(task int, label string) string {
	return evalSummaryKeyString(fmt.Sprintf("%d", task), label)
}

func evalSummaryKeyString(task string, label string) string {
	return strings.TrimSpace(task) + ":" + strings.ToLower(strings.TrimSpace(label))
}

func renderEvalAutomationPacket(now time.Time, project string, runRel string, summaryRel string, tasksetRel string, sourceRel string, comparisons []evalVariantComparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Runner Packet\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Run packet: `%s`\n", runRel)
	fmt.Fprintf(&b, "Summary scores: `%s`\n", summaryRel)
	if tasksetRel != "" {
		fmt.Fprintf(&b, "Taskset: `%s`\n", tasksetRel)
	}
	fmt.Fprintf(&b, "Source artifact: `%s`\n\n", sourceRel)
	b.WriteString("## Purpose\n\n")
	b.WriteString("Use this packet to drive the impact test through an external runner or agent. The Go TUI prepares the prompts and output targets; it does not execute model calls, paste outputs, or score results automatically.\n\n")
	b.WriteString("## Execution Matrix\n\n")
	b.WriteString("| Variant | Context | Context Status | Output Target | Current Coverage |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, comparison := range comparisons {
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s |\n",
			escapeMarkdownTable(comparison.Variant.Label),
			escapeMarkdownTable(comparison.Variant.Context),
			escapeMarkdownTable(evalVariantContextStatus(project, comparison.Variant)),
			comparison.RelPath,
			escapeMarkdownTable(evalVariantCoverageStatus(comparison)),
		)
	}
	b.WriteString("\n## Runner Instructions\n\n")
	b.WriteString("1. Run each variant in a clean agent/session so previous outputs do not leak across variants.\n")
	b.WriteString("2. Load exactly the context listed for that variant. If a context file is missing, mark the variant blocked instead of substituting another artifact.\n")
	b.WriteString("3. Run every task from the taskset with identical wording.\n")
	b.WriteString("4. Paste raw outputs back into the variant file under `Task 1 Output`, `Task 2 Output`, and `Task 3 Output`.\n")
	b.WriteString("5. Fill the summary score table, then generate a comparison and optional judge packet from Impact Tests.\n\n")
	b.WriteString("## Variant Prompt Blocks\n\n")
	for _, comparison := range comparisons {
		fmt.Fprintf(&b, "### %s\n\n", comparison.Variant.Label)
		fmt.Fprintf(&b, "- Output target: `%s`\n", comparison.RelPath)
		fmt.Fprintf(&b, "- Context to load: %s\n", comparison.Variant.Context)
		fmt.Fprintf(&b, "- Context status: %s\n", evalVariantContextStatus(project, comparison.Variant))
		fmt.Fprintf(&b, "- Instruction: Run the taskset exactly, then paste raw outputs into `%s` without rewriting the task wording.\n\n", comparison.RelPath)
	}
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No model runs were executed by the Go TUI.\n")
	b.WriteString("- No impact-test outputs, scores, source files, skills, or operating-layer files were changed by this packet.\n")
	b.WriteString("- Treat this as a reproducible run plan for an external runner until in-TUI model execution has its own reviewed configuration and safety gates.\n")
	return b.String()
}

func renderEvalReadinessReport(now time.Time, project string, runRel string, summaryRel string, tasksetRel string, sourceRel string, comparisons []evalVariantComparison) string {
	blockers := evalReadinessBlockingItems(project, comparisons)
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Readiness Report\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Run packet: `%s`\n", runRel)
	fmt.Fprintf(&b, "Summary scores: `%s`\n", summaryRel)
	if tasksetRel != "" {
		fmt.Fprintf(&b, "Taskset: `%s`\n", tasksetRel)
	}
	fmt.Fprintf(&b, "Source artifact: `%s`\n\n", sourceRel)
	b.WriteString("## Decision\n\n")
	if len(blockers) == 0 {
		b.WriteString("Ready for comparison and optional judge review. Every variant has required context, three pasted outputs, and three summary scores.\n\n")
	} else {
		fmt.Fprintf(&b, "Needs review before comparison or judge review: %s.\n\n", intLabel(len(blockers), "blocking item"))
	}
	b.WriteString("## Readiness Matrix\n\n")
	b.WriteString("| Variant | Context | Outputs | Scores | Next Action |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, comparison := range comparisons {
		context := evalVariantContextStatus(project, comparison.Variant)
		outputs := evalVariantCoverageStatus(comparison)
		scores := evalScoreCoverageStatus(comparison)
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			escapeMarkdownTable(comparison.Variant.Label),
			escapeMarkdownTable(context),
			escapeMarkdownTable(outputs),
			escapeMarkdownTable(scores),
			escapeMarkdownTable(evalReadinessNextAction(context, outputs, scores)),
		)
	}
	b.WriteString("\n## Blocking Items\n\n")
	if len(blockers) == 0 {
		b.WriteString("- None. Generate a comparison report or judge packet when the review rubric is ready.\n")
	} else {
		for _, blocker := range blockers {
			fmt.Fprintf(&b, "- %s\n", blocker)
		}
	}
	b.WriteString("\n## Next Steps\n\n")
	b.WriteString("- Add missing context files before running context-dependent variants.\n")
	b.WriteString("- Run incomplete variants in fresh sessions and paste raw outputs into their variant files.\n")
	b.WriteString("- Fill summary scores and qualitative notes before treating a comparison as evidence.\n")
	b.WriteString("- Generate a comparison report and optional judge packet only after blockers are resolved or intentionally marked out of scope.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No model runs were executed by this readiness report.\n")
	b.WriteString("- No judge scoring was executed or written back automatically.\n")
	b.WriteString("- No variant outputs, summary scores, source files, skills, or operating-layer files were changed.\n")
	return b.String()
}

func evalReadinessBlockingItems(project string, comparisons []evalVariantComparison) []string {
	blockers := []string{}
	for _, comparison := range comparisons {
		context := evalVariantContextStatus(project, comparison.Variant)
		outputs := evalVariantCoverageStatus(comparison)
		scores := evalScoreCoverageStatus(comparison)
		if strings.Contains(context, "missing") {
			blockers = append(blockers, fmt.Sprintf("%s context: %s", comparison.Variant.Label, context))
		}
		if outputs != "ready" {
			blockers = append(blockers, fmt.Sprintf("%s outputs: %s", comparison.Variant.Label, outputs))
		}
		if scores != "ready" {
			blockers = append(blockers, fmt.Sprintf("%s scores: %s", comparison.Variant.Label, scores))
		}
	}
	return blockers
}

func evalReadinessNextAction(context string, outputs string, scores string) string {
	if strings.Contains(context, "missing") {
		return "Add context or mark this variant blocked."
	}
	if outputs != "ready" {
		return "Run variant and paste missing task outputs."
	}
	if scores != "ready" {
		return "Fill summary scores and qualitative notes."
	}
	return "Ready for comparison or judge review."
}

func evalVariantContextStatus(project string, variant evalVariant) string {
	switch variant.ID {
	case "baseline":
		return "ready: load no project files"
	case "corpus":
		return evalFilePresence(project, []string{"MIXTAPE.md"})
	case "operating-layer":
		return evalFilePresence(project, []string{"MIXTAPE.md", "LINER.md"})
	case "skills":
		status := evalFilePresence(project, []string{"MIXTAPE.md", "LINER.md"})
		count := countTopLevelRegularFiles(filepath.Join(project, "skills"), ".md")
		if count == 0 {
			return status + "; skills/*.md missing"
		}
		return fmt.Sprintf("%s; %d skill file(s)", status, count)
	default:
		return "review context manually"
	}
}

func evalFilePresence(project string, rels []string) string {
	missing := []string{}
	for _, rel := range rels {
		if !projectFileExists(project, rel) {
			missing = append(missing, rel+" missing")
		}
	}
	if len(missing) > 0 {
		return strings.Join(missing, "; ")
	}
	return "ready"
}

func renderEvalComparisonReport(now time.Time, runRel string, summaryRel string, comparisons []evalVariantComparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Comparison Report\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Run packet: `%s`\n", runRel)
	fmt.Fprintf(&b, "Summary scores: `%s`\n\n", summaryRel)
	b.WriteString("## Output Coverage\n\n")
	b.WriteString("| Variant | File | Task 1 | Task 2 | Task 3 | Status |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, comparison := range comparisons {
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %s | %s |\n",
			escapeMarkdownTable(comparison.Variant.Label),
			comparison.RelPath,
			escapeMarkdownTable(evalTaskCoverage(comparison, 1)),
			escapeMarkdownTable(evalTaskCoverage(comparison, 2)),
			escapeMarkdownTable(evalTaskCoverage(comparison, 3)),
			escapeMarkdownTable(evalVariantCoverageStatus(comparison)),
		)
	}
	b.WriteString("\n## Side-by-Side Excerpts\n\n")
	for task := 1; task <= 3; task++ {
		fmt.Fprintf(&b, "### Task %d\n\n", task)
		b.WriteString("| Variant | Score | Output Excerpt | Human Notes |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, comparison := range comparisons {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				escapeMarkdownTable(comparison.Variant.Label),
				escapeMarkdownTable(evalBlankFallback(comparison.Score[task-1], "unscored")),
				escapeMarkdownTable(evalOutputExcerpt(comparison.TaskOutput[task-1])),
				escapeMarkdownTable(evalBlankFallback(comparison.Notes[task-1], "no notes")),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Review Questions\n\n")
	b.WriteString("- What impact did each loaded layer add over baseline?\n")
	b.WriteString("- Did `MIXTAPE.md` improve factual grounding compared with baseline?\n")
	b.WriteString("- Did `LINER.md` change behavior in the intended direction?\n")
	b.WriteString("- Did skills improve repeatability or just add ceremony?\n")
	b.WriteString("- Which source, note, skill, or operating rule should change before the next run?\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No model runs were executed by this comparison.\n")
	b.WriteString("- No project files were changed except this comparison report.\n")
	b.WriteString("- Treat missing outputs as a run-packet completion issue, not as evidence that a variant failed.\n")
	return b.String()
}

func renderEvalJudgePacket(now time.Time, runRel string, summaryRel string, sourceRel string, comparisons []evalVariantComparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Impact Test Judge Packet\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Run packet: `%s`\n", runRel)
	fmt.Fprintf(&b, "Summary scores: `%s`\n", summaryRel)
	fmt.Fprintf(&b, "Source artifact: `%s`\n\n", sourceRel)
	b.WriteString("## Purpose\n\n")
	b.WriteString("Use this packet to ask a human or external LLM judge whether the mixtape's corpus, operating layer, and skills improved output quality over baseline. This packet does not run a model or write scores automatically.\n\n")
	b.WriteString("## Judge Instructions\n\n")
	b.WriteString("- Judge only the outputs and evidence below; do not reward claims that are not visible in the run packet.\n")
	b.WriteString("- Prefer the narrowest, source-grounded answer over generic confidence.\n")
	b.WriteString("- Penalize overreach outside the mixtape scope, missing boundaries, and unsupported advice.\n")
	b.WriteString("- Fill the judge score table before changing any source, note, skill, or operating rule.\n\n")
	b.WriteString("## Rubric\n\n")
	b.WriteString("| Criterion | 1 | 3 | 5 |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	b.WriteString("| Task fit | misses the user's job | addresses it broadly | directly serves the job |\n")
	b.WriteString("| Source grounding | generic or invented | partially grounded | clearly grounded in corpus evidence |\n")
	b.WriteString("| Operating behavior | ignores LINER/skills | follows some rules | behaves like the intended artifact |\n")
	b.WriteString("| Specificity | vague | partly concrete | concrete and inspectable |\n")
	b.WriteString("| Restraint | overclaims | mostly scoped | names limits and asks for missing context |\n\n")
	b.WriteString("## Judge Score Table\n\n")
	b.WriteString("| Task | Variant | Human Score | Judge Score | Judge Reason |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for task := 1; task <= 3; task++ {
		for _, comparison := range comparisons {
			fmt.Fprintf(&b, "| Task %d | %s | %s |  |  |\n",
				task,
				escapeMarkdownTable(comparison.Variant.Label),
				escapeMarkdownTable(evalBlankFallback(comparison.Score[task-1], "unscored")),
			)
		}
	}
	b.WriteString("\n## Side-by-Side Evidence\n\n")
	for task := 1; task <= 3; task++ {
		fmt.Fprintf(&b, "### Task %d\n\n", task)
		b.WriteString("| Variant | Coverage | Human Score | Output Excerpt | Human Notes |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, comparison := range comparisons {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				escapeMarkdownTable(comparison.Variant.Label),
				escapeMarkdownTable(evalTaskCoverage(comparison, task)),
				escapeMarkdownTable(evalBlankFallback(comparison.Score[task-1], "unscored")),
				escapeMarkdownTable(evalOutputExcerpt(comparison.TaskOutput[task-1])),
				escapeMarkdownTable(evalBlankFallback(comparison.Notes[task-1], "no notes")),
			)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No judge was run by the TUI.\n")
	b.WriteString("- No project files were changed except this judge packet.\n")
	b.WriteString("- Apply source, note, skill, or operating-layer changes only after reviewing the judge reasons.\n")
	return b.String()
}

func evalTaskCoverage(comparison evalVariantComparison, task int) string {
	if !comparison.Exists {
		return "missing file"
	}
	words := comparison.WordCount[task-1]
	if words == 0 {
		return "missing"
	}
	return fmt.Sprintf("%dw", words)
}

func evalVariantCoverageStatus(comparison evalVariantComparison) string {
	if !comparison.Exists {
		return "missing file"
	}
	filled := 0
	for _, words := range comparison.WordCount {
		if words > 0 {
			filled++
		}
	}
	switch filled {
	case 3:
		return "ready"
	case 0:
		return "empty"
	default:
		return fmt.Sprintf("partial %d/3", filled)
	}
}

func evalScoreCoverageStatus(comparison evalVariantComparison) string {
	scored := 0
	for _, score := range comparison.Score {
		if strings.TrimSpace(score) != "" {
			scored++
		}
	}
	switch scored {
	case 3:
		return "ready"
	case 0:
		return "unscored"
	default:
		return fmt.Sprintf("%d/3 scored", scored)
	}
}

func evalOutputExcerpt(output string) string {
	output = strings.Join(strings.Fields(output), " ")
	if output == "" {
		return "missing output"
	}
	if len(output) <= 180 {
		return output
	}
	return strings.TrimSpace(output[:177]) + "..."
}

func evalBlankFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func evalWordCount(value string) int {
	return len(strings.Fields(value))
}
