package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
	"gopkg.in/yaml.v3"
)

const legacyLinerDraftRelPath = "working/LINER-draft.md"
const operatingLayerStepDelay = 250 * time.Millisecond

func (m Model) canGenerateLiner() bool {
	return m.canCreateOperatingLayer()
}

func (m Model) canCreateOperatingLayer() bool {
	return m.hasCorpusReady() && !m.projectCapabilities().HasLiner
}

func (m Model) canRegenerateOperatingLayer() bool {
	return strings.TrimSpace(m.currentPath) != "" &&
		m.projectCapabilities().HasLiner &&
		m.hasCorpusReady() &&
		!m.projectCompileNeedsAttention()
}

func (m Model) startLinerDraftReview() (Model, tea.Cmd) {
	if !m.hasCorpusReady() {
		m.err = "Reach Corpus Ready before creating the Operating Layer."
		return m, nil
	}
	if strings.TrimSpace(m.currentPath) == "" {
		m.err = "Cannot create the Operating Layer without a project path."
		return m, nil
	}
	m.operatingLayerRunning = false
	m.operatingLayerComplete = false
	m.operatingLayerStep = 0
	m.operatingLayerContent = ""
	m.operatingLayerSkillName, m.operatingLayerSkillPath = projectSkillProposal(m.currentTape)
	m.screen = screenLinerReview
	m.note = ""
	return m, nil
}

func (m Model) handleLinerReviewKey(keyMsg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.operatingLayerRunning {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.operatingLayerComplete {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter", "esc":
			return m.returnToProjectAfterOperatingLayer(), nil
		}
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenProject
		return m, nil
	case "enter":
		return m.startOperatingLayerCreation()
	}
	return m, nil
}

func (m Model) startOperatingLayerCreation() (Model, tea.Cmd) {
	m.operatingLayerRunning = true
	m.operatingLayerComplete = false
	m.operatingLayerStep = 0
	m.operatingLayerContent = ""
	m.operatingLayerSkillName, m.operatingLayerSkillPath = projectSkillProposal(m.currentTape)
	m.note = ""
	m.err = ""
	return m, delayedOperatingLayerCmd(writeOperatingLayerLinerCmd(m.currentPath, m.currentTape))
}

func (m Model) applyOperatingLayerStep(msg operatingLayerStepMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.operatingLayerRunning = false
		m.err = msg.err.Error()
		return m, nil
	}
	if strings.TrimSpace(msg.content) != "" {
		m.operatingLayerContent = msg.content
	}
	if strings.TrimSpace(msg.skillName) != "" {
		m.operatingLayerSkillName = msg.skillName
	}
	if strings.TrimSpace(msg.skillPath) != "" {
		m.operatingLayerSkillPath = msg.skillPath
	}
	if msg.step >= 0 {
		m.operatingLayerStep = max(m.operatingLayerStep, msg.step+1)
	}
	switch msg.step {
	case 0:
		return m, delayedOperatingLayerCmd(writeOperatingLayerSkillCmd(m.currentPath, m.currentTape))
	case 1:
		return m, delayedOperatingLayerCmd(writeOperatingLayerMetadataCmd(m.currentPath, m.operatingLayerSkillName, m.operatingLayerSkillPath))
	default:
		return m.finishOperatingLayerCreation()
	}
}

func delayedOperatingLayerCmd(cmd tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(operatingLayerStepDelay)
		return cmd()
	}
}

func writeOperatingLayerLinerCmd(project string, current tape.Tape) tea.Cmd {
	return func() tea.Msg {
		skillName, skillPath := projectSkillProposal(current)
		content, err := buildLinerContent(project, current)
		if err != nil {
			return operatingLayerStepMsg{step: 0, err: fmt.Errorf("could not generate LINER.md: %w", err)}
		}
		if err := os.WriteFile(filepath.Join(project, "LINER.md"), []byte(content), 0o644); err != nil {
			return operatingLayerStepMsg{step: 0, err: fmt.Errorf("could not write LINER.md: %w", err)}
		}
		return operatingLayerStepMsg{step: 0, content: content, skillName: skillName, skillPath: skillPath}
	}
}

func writeOperatingLayerSkillCmd(project string, current tape.Tape) tea.Cmd {
	return func() tea.Msg {
		skillName, skillPath, err := writeProjectSkillFile(project, current)
		if err != nil {
			return operatingLayerStepMsg{step: 1, err: fmt.Errorf("could not write Project Skill: %w", err)}
		}
		return operatingLayerStepMsg{step: 1, skillName: skillName, skillPath: skillPath}
	}
}

func writeOperatingLayerMetadataCmd(project string, skillName string, skillPath string) tea.Cmd {
	return func() tea.Msg {
		if err := writeOperatingLayerMetadata(project, skillName, skillPath); err != nil {
			return operatingLayerStepMsg{step: 2, err: fmt.Errorf("LINER.md and SKILL.md were written, but liner.yaml could not be updated: %w", err)}
		}
		return operatingLayerStepMsg{step: 2}
	}
}

func (m Model) finishOperatingLayerCreation() (Model, tea.Cmd) {
	_ = os.Remove(filepath.Join(m.currentPath, legacyLinerDraftRelPath))
	m = m.withCompletedOperatingLayerStatus(m.operatingLayerSkillName, m.operatingLayerSkillPath)
	m.operatingLayerRunning = false
	m.operatingLayerComplete = true
	m.operatingLayerStep = len(operatingLayerArtifacts())
	m.operatingLayerContent = ""
	m.hasPreviewBack = false
	m.note = ""
	if strings.TrimSpace(m.runner.Command) == "" {
		return m, nil
	}
	return m, loadProjectStatus(m.runner, m.currentPath)
}

func (m Model) returnToProjectAfterOperatingLayer() Model {
	m.operatingLayerRunning = false
	m.operatingLayerComplete = false
	m.operatingLayerStep = 0
	m.operatingLayerContent = ""
	m.screen = screenProject
	m.hasPreviewBack = false
	m.note = "Created Operating Layer."
	return m
}

func (m Model) viewLinerReview() string {
	width := styles.ClampWidth(m.width - 4)
	title := "Create Operating Layer"
	if m.operatingLayerRunning {
		title = "Creating Operating Layer"
	}
	parts := []string{
		m.renderLoadingTitle(title, m.operatingLayerRunning),
		renderWrappedStyledText(width, styles.Subtitle, "Creates the operating instructions and root SKILL.md entrypoint for this corpus."),
		"",
	}
	if m.operatingLayerRunning || m.operatingLayerComplete {
		parts = append(parts,
			m.operatingLayerProgressView(width),
			"",
		)
	} else {
		parts = append(parts,
			operatingLayerIdleDetails(width),
			"",
		)
	}
	parts = append(parts, operatingLayerRunControl(m.operatingLayerRunning, m.operatingLayerComplete, m.operatingLayerSpin.View()))
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.NewStyle().MaxWidth(width).Render(body)
}

type projectSkillProfile struct {
	Full      bool
	Rationale string
}

type operatingLayerArtifact struct {
	Name        string
	Path        string
	Description string
}

type operatingCorpusContract struct {
	CoreAction     string
	Thesis         string
	Rules          []string
	Pattern        string
	InputDomains   string
	Translation    string
	CallerHandoff  string
	Constraint     string
	QualityFinding string
}

func projectSkillProfileFor(current tape.Tape) projectSkillProfile {
	stats := projectSkillRecommendationStats(current.Sources)
	if stats.sourceCount >= 8 && stats.reusableCount >= 3 && (stats.sectionCount >= 3 || stats.skillSourceCount > 0) {
		return projectSkillProfile{
			Full:      true,
			Rationale: "This project has reusable guidance, so the skill will include specific rules for using this corpus.",
		}
	}
	return projectSkillProfile{
		Full:      false,
		Rationale: "The skill will stay short and point AI sessions to LINER.md for the detailed guidance.",
	}
}

func operatingLayerIdleDetails(width int) string {
	rows := []operatingLayerArtifact{
		{
			Name:        "LINER.md",
			Description: "Turns MIXTAPE.md into operating guidance: what the corpus is for, how AI sessions should use evidence, where the source boundaries are, and when to abstain.",
		},
		{
			Name:        "SKILL.md",
			Description: "Adds the root skill entrypoint. It lets future AI sessions find this project by name, load LINER.md and MIXTAPE.md in the right order, and stay inside the corpus.",
		},
	}
	labelWidth := 0
	for _, row := range rows {
		labelWidth = max(labelWidth, lipgloss.Width(row.Name))
	}
	labelWidth = min(14, max(8, labelWidth))
	valueWidth := max(24, width-labelWidth-3)
	lines := []string{}
	for _, row := range rows {
		descriptionLines := wrapLabelValue(row.Description, valueWidth)
		label := styles.PrimaryText.Width(labelWidth).Render(row.Name)
		if len(descriptionLines) == 0 {
			lines = append(lines, label)
			continue
		}
		lines = append(lines, label+"  "+styles.Subtitle.Render(descriptionLines[0]))
		indent := strings.Repeat(" ", labelWidth+2)
		for _, line := range descriptionLines[1:] {
			lines = append(lines, indent+styles.Subtitle.Render(line))
		}
	}
	return strings.Join(lines, "\n")
}

type projectSkillRecommendationStatsResult struct {
	sourceCount      int
	sectionCount     int
	reusableCount    int
	skillSourceCount int
}

func projectSkillRecommendationStats(sources []tape.Source) projectSkillRecommendationStatsResult {
	sections := map[string]bool{}
	stats := projectSkillRecommendationStatsResult{sourceCount: len(sources)}
	for _, src := range sources {
		if src.Section != nil && strings.TrimSpace(*src.Section) != "" {
			sections[strings.TrimSpace(*src.Section)] = true
		}
		if strings.EqualFold(strings.TrimSpace(src.Type), "skill") {
			stats.skillSourceCount++
			stats.reusableCount++
			continue
		}
		if src.Kind != nil {
			switch strings.ToLower(strings.TrimSpace(*src.Kind)) {
			case "principle", "prescription":
				stats.reusableCount++
			}
		}
	}
	stats.sectionCount = len(sections)
	return stats
}

func operatingLayerArtifacts() []operatingLayerArtifact {
	return []operatingLayerArtifact{
		{
			Name:        "LINER.md",
			Path:        "LINER.md",
			Description: "Generating operating guidance.",
		},
		{
			Name:        "SKILL.md",
			Path:        "SKILL.md",
			Description: "Creating root skill entrypoint.",
		},
		{
			Name:        "Finalize project",
			Path:        "project status",
			Description: "Marking Operating Layer ready.",
		},
	}
}

func (m Model) operatingLayerProgressView(width int) string {
	artifacts := operatingLayerArtifacts()
	total := len(artifacts)
	completed := min(max(m.operatingLayerStep, 0), total)
	status := "Working"
	detail := operatingLayerStepDetail(completed, false)
	if m.operatingLayerComplete {
		status = "Project complete"
		detail = "LINER.md and SKILL.md are ready."
		completed = total
	}
	percent := 0.0
	if total > 0 {
		percent = float64(completed) / float64(total)
	}
	count := fmt.Sprintf("%d/%d steps", completed, total)
	bar := newTaskProgressBar(taskProgressWidth(width))
	lines := []string{
		renderProgressStatusBlock(width, bar, percent, status, detail, count),
		"",
		operatingLayerArtifactsView(width, completed, m.operatingLayerRunning, m.operatingLayerComplete, m.operatingLayerSpin.View()),
	}
	if m.operatingLayerComplete {
		lines = append(lines,
			"",
			styles.SuccessText.Render("Project complete. LINER.md and SKILL.md are ready."),
			styles.Subtitle.Render("Press Enter to go back to Project."),
		)
	}
	return strings.Join(lines, "\n")
}

func operatingLayerArtifactsView(width int, completed int, running bool, complete bool, spinnerView string) string {
	artifacts := operatingLayerArtifacts()
	lines := []string{}
	for index, artifact := range artifacts {
		status := ""
		statusStyle := styles.Subtitle
		if complete {
			status = "done"
			statusStyle = styles.SuccessText
		} else if running {
			switch {
			case index < completed:
				status = "done"
				statusStyle = styles.SuccessText
			case index == completed:
				status = strings.TrimSpace(spinnerView + " " + operatingLayerStepDetail(index, true))
				statusStyle = styles.AccentText
			default:
				status = "queued"
			}
		}
		headParts := []string{}
		if status != "" {
			headParts = append(headParts, statusStyle.Render(status))
		}
		headParts = append(headParts, styles.PrimaryText.Render(artifact.Name))
		head := strings.Join(headParts, "  ")
		lines = append(lines, head)
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func operatingLayerStepDetail(step int, active bool) string {
	switch step {
	case 0:
		if active {
			return "Generating LINER.md"
		}
		return "Generating LINER.md."
	case 1:
		if active {
			return "Creating SKILL.md"
		}
		return "Creating SKILL.md."
	case 2:
		if active {
			return "Finalizing project"
		}
		return "Finalizing project."
	default:
		return "Finishing Operating Layer."
	}
}

func operatingLayerRunControl(running bool, complete bool, spinnerView string) string {
	if running {
		label := strings.TrimSpace(spinnerView + " Creating Operating Layer")
		return styles.AccentText.Render(label)
	}
	if complete {
		return renderNextCue("Go back to Project.")
	}
	return renderNextCue("Create Operating Layer.")
}

func renderWrappedStyledText(width int, style lipgloss.Style, text string) string {
	width = max(24, width)
	lines := wrapLabelValue(text, width)
	for index, line := range lines {
		lines[index] = style.Render(line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) withCompletedOperatingLayerStatus(skillName string, skillPath string) Model {
	status := core.ProjectStatus{}
	if current := m.currentProjectStatus(); current != nil {
		status = *current
	}
	now := time.Now().Format(time.RFC3339)
	status.Snapshot.Milestone = "project_complete"
	status.Snapshot.Stale = false
	status.Snapshot.Updated = now
	if strings.TrimSpace(status.Snapshot.Corpus.State) == "" {
		status.Snapshot.Corpus.State = "ready"
		status.Snapshot.Corpus.Evidence = "MIXTAPE.md"
	}
	status.Snapshot.OperatingLayer.State = "ready"
	status.Snapshot.OperatingLayer.Evidence = "LINER.md"
	status.Snapshot.OperatingLayer.Audit = nil
	status.ProjectSkill = core.ProjectSkillStatus{Status: "active"}
	if strings.TrimSpace(skillName) != "" {
		name := skillName
		status.ProjectSkill.Name = &name
	}
	if strings.TrimSpace(skillPath) != "" {
		path := skillPath
		status.ProjectSkill.Path = &path
	}
	m.statusPath = m.currentPath
	m.status = &status
	m.statusErr = ""
	return m
}

func (m *Model) setPreviewContent(rel string, content string) {
	rendered, err := glamour.Render(content, "dark")
	if err != nil {
		rendered = content
	}
	m.preview.SetContent(rendered)
	m.preview.GotoTop()
	m.previewRel = rel
}

func buildLinerContent(project string, current tape.Tape, _ ...string) (string, error) {
	if !projectFileExists(project, "MIXTAPE.md") {
		return "", fmt.Errorf("MIXTAPE.md is required before generating LINER.md")
	}
	title := fallbackText(current.Title, filepath.Base(project))
	jtbd := "No job is recorded. Stay inside the documented corpus scope."
	if current.JTBD != nil && strings.TrimSpace(*current.JTBD) != "" {
		jtbd = strings.TrimSpace(*current.JTBD)
	} else if strings.TrimSpace(current.Description) != "" {
		jtbd = strings.TrimSpace(current.Description)
	}

	kinds := sourceKindSummary(current.Sources)
	sections := sourceSectionSummary(current.Sources)
	availability := compiledSourceAvailabilitySummary(project, len(current.Sources))
	synthesisNote := optionalArtifactNote(project, "synthesis.md")
	qualityNote := optionalArtifactNote(project, filepath.Join("working", "04-quality-checks.md"))
	contract := readOperatingCorpusContract(project)
	skillName, skillPath := projectSkillProposal(current)
	skillProfile := projectSkillProfileFor(current)
	mixtapePath := displayProjectPath(project, "MIXTAPE.md")
	sourcesPath := strings.TrimSuffix(displayProjectPath(project, "sources"), "/")

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	b.WriteString("## Scope\n\n")
	fmt.Fprintf(&b, "This Operating Layer is for this job:\n\n> %s\n\n", jtbd)
	b.WriteString("Use this file to guide how AI sessions work with this local Mixtape. ")
	fmt.Fprintf(&b, "Use `%s` as the main reading artifact. Open deeper source files when the answer needs evidence.\n\n", mixtapePath)

	b.WriteString("## How To Use This Project\n\n")
	fmt.Fprintf(&b, "- Load `LINER.md` first, then `%s` for the source-grounded corpus.\n", mixtapePath)
	b.WriteString("- Use this file to choose working mode, source priority, boundaries, and maintenance behavior.\n")
	b.WriteString("- Do not present the project as a persona, chatbot, or generic knowledge bot.\n")
	b.WriteString("- Before giving a strong recommendation, name the corpus stance, rule, source section, or Project Skill that supports it.\n")
	b.WriteString("- If the task could change project files, draft changes under `working/` and require review before touching production files.\n\n")

	b.WriteString("## Working Loop\n\n")
	b.WriteString("1. Orient: restate the user's job in the language of this mixtape.\n")
	fmt.Fprintf(&b, "2. Retrieve: load `%s`, then source files or the active Project Skill only when the answer needs deeper evidence.\n", mixtapePath)
	b.WriteString("3. Apply: turn source-grounded stances into a concrete answer, critique, plan, or question.\n")
	b.WriteString("4. Check: test the answer against conflict, abstention, and source-use rules before finalizing.\n")
	b.WriteString("5. Maintain: when the corpus is thin, name the missing source, note, or Project Skill boundary that should be updated.\n\n")

	writeLinerCorpusContract(&b, contract)

	b.WriteString("## Operating Stance\n\n")
	b.WriteString("- Be opinionated only where the corpus is opinionated.\n")
	b.WriteString("- Translate the corpus into decisions, not summaries.\n")
	b.WriteString("- Start from the curator's synthesis before answering.\n")
	b.WriteString("- Prefer source-grounded tradeoffs over generic advice.\n")
	b.WriteString("- Keep the project's scope narrow; do not claim coverage beyond the corpus.\n")
	b.WriteString("- When a request falls outside the corpus, say what is missing and ask for a source.\n\n")

	b.WriteString("## Resource Map\n\n")
	b.WriteString("- `LINER.md`: operating layer and behavior rules.\n")
	fmt.Fprintf(&b, "- `%s`: compiled corpus and source-grounded reading packet.\n", mixtapePath)
	fmt.Fprintf(&b, "- `%s/`: deeper evidence for claims that need source-level detail.\n", sourcesPath)
	fmt.Fprintf(&b, "- Project Skill: `%s` at `%s`.\n", skillName, filepath.ToSlash(skillPath))
	b.WriteString("- `working/`: drafts and maintenance notes for changes that should be reviewed before production files change.\n\n")

	b.WriteString("## Source Use Rules\n\n")
	fmt.Fprintf(&b, "- Corpus size: %d saved source(s).\n", len(current.Sources))
	fmt.Fprintf(&b, "- Compiled availability: %s.\n", availability)
	fmt.Fprintf(&b, "- Source kinds: %s.\n", fallbackText(kinds, "not classified yet"))
	fmt.Fprintf(&b, "- Source sections: %s.\n", fallbackText(sections, "not sectioned yet"))
	fmt.Fprintf(&b, "- Synthesis status: %s.\n", synthesisNote)
	fmt.Fprintf(&b, "- Quality status: %s.\n", qualityNote)
	fmt.Fprintf(&b, "- Load `%s` first, then open individual source files when the answer depends on detail.\n", mixtapePath)
	b.WriteString("- Respect source notes and source kinds; a canonical/reference source should outweigh a loose example.\n")
	b.WriteString("- Cite source titles or sections when making a strong recommendation.\n\n")

	b.WriteString("## Project Skill\n\n")
	fmt.Fprintf(&b, "Active Project Skill: `%s` at `%s`.\n\n", skillName, filepath.ToSlash(skillPath))
	if skillProfile.Full {
		b.WriteString("- This project has reusable guidance, so the Project Skill includes specific rules for using this corpus.\n")
	} else {
		b.WriteString("- The Project Skill stays short and points AI sessions to this file for the detailed guidance.\n")
	}
	b.WriteString("- Use it only when the user's request matches this project's job and corpus stance.\n")
	fmt.Fprintf(&b, "- Treat the skill as grounded in `%s`; do not let it override source hierarchy, conflict rules, or abstention rules.\n", mixtapePath)
	b.WriteString("- If the skill needs to change, draft an update under `working/` and review it before editing project files.\n\n")

	b.WriteString("## Conflict Rules\n\n")
	b.WriteString("- Prefer newer or more canonical sources when time-sensitive guidance conflicts.\n")
	b.WriteString("- Prefer direct product or platform documentation over commentary when implementation details conflict.\n")
	b.WriteString("- Preserve minority perspectives when the corpus intentionally includes them.\n")
	b.WriteString("- Record unresolved contradictions in `working/audits/` instead of smoothing them away.\n\n")

	b.WriteString("## Abstention Rules\n\n")
	b.WriteString("- Do not answer as if the corpus covers laws, medical advice, finances, or safety-critical claims unless those sources are present.\n")
	b.WriteString("- Do not invent source-backed confidence. Mark unsupported claims as outside scope.\n")
	b.WriteString("- If a user asks for a decision the corpus cannot support, explain the missing evidence and propose what to add.\n\n")

	b.WriteString("## Maintenance Rules\n\n")
	b.WriteString("- New books, recordings, local notes, and URLs should enter as sources before they change this file.\n")
	b.WriteString("- After adding sources, refresh synthesis before changing operating rules.\n")
	b.WriteString("- Keep generated changes review-first: draft, inspect, then update the project files deliberately.\n")

	return b.String(), nil
}

func readOperatingCorpusContract(project string) operatingCorpusContract {
	contract := operatingCorpusContract{}
	if synthesis, ok := readProjectMarkdown(project, "synthesis.md"); ok {
		contract.Thesis = firstMarkdownParagraph(synthesis)
		contract.Rules = markdownBullets(markdownHeadingBody(synthesis, "## generative rules"), 7)
	}
	if quality, ok := readProjectMarkdown(project, filepath.Join("working", "04-quality-checks.md")); ok {
		if value, ok := markdownMetadataValue(quality, "Core action"); ok {
			contract.CoreAction = normalizeInlineMarkdown(value)
		}
		test8 := qualityHeadingBody(quality, "## test 8")
		if value, ok := markdownMetadataValue(test8, "Pattern"); ok {
			contract.Pattern = normalizeInlineMarkdown(value)
		}
		if value, ok := markdownMetadataValue(test8, "Input/reference domains"); ok {
			contract.InputDomains = normalizeInlineMarkdown(value)
		}
		if value, ok := markdownMetadataValue(test8, "Translation-method sources"); ok {
			contract.Translation = normalizeInlineMarkdown(value)
		}
		if value, ok := markdownMetadataValue(test8, "Target/caller handoff"); ok {
			contract.CallerHandoff = normalizeInlineMarkdown(value)
		} else if value, ok := markdownMetadataValue(test8, "Caller-handoff sources"); ok {
			contract.CallerHandoff = normalizeInlineMarkdown(value)
		}
		if value, ok := markdownMetadataValue(test8, "Constraint balance"); ok {
			contract.Constraint = normalizeInlineMarkdown(value)
		}
		if value, ok := markdownMetadataValue(test8, "Finding"); ok {
			contract.QualityFinding = normalizeInlineMarkdown(value)
		}
	}
	return contract
}

func readProjectMarkdown(project string, rel string) (string, bool) {
	if strings.TrimSpace(project) == "" {
		return "", false
	}
	body, err := os.ReadFile(projectAbsPath(project, rel))
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(string(body))
	return text, text != ""
}

func (c operatingCorpusContract) hasContent() bool {
	return c.CoreAction != "" ||
		c.Thesis != "" ||
		len(c.Rules) > 0 ||
		c.Pattern != "" ||
		c.InputDomains != "" ||
		c.Translation != "" ||
		c.CallerHandoff != "" ||
		c.Constraint != "" ||
		c.QualityFinding != ""
}

func writeLinerCorpusContract(b *strings.Builder, contract operatingCorpusContract) {
	if !contract.hasContent() {
		return
	}
	b.WriteString("## Corpus-Derived Operating Contract\n\n")
	if contract.CoreAction != "" {
		fmt.Fprintf(b, "- Core action: %s.\n", trimTrailingPeriod(contract.CoreAction))
	}
	if contract.Thesis != "" {
		fmt.Fprintf(b, "- Operating thesis: %s.\n", trimTrailingPeriod(contract.Thesis))
	}
	if contract.Pattern != "" {
		fmt.Fprintf(b, "- Capability pattern: %s.\n", trimTrailingPeriod(contract.Pattern))
	}
	if contract.InputDomains != "" {
		fmt.Fprintf(b, "- Input domains: %s.\n", trimTrailingPeriod(contract.InputDomains))
	}
	if contract.Translation != "" {
		fmt.Fprintf(b, "- Translation evidence: %s.\n", trimTrailingPeriod(contract.Translation))
	}
	if contract.CallerHandoff != "" {
		fmt.Fprintf(b, "- Caller handoff: %s.\n", trimTrailingPeriod(contract.CallerHandoff))
	}
	if len(contract.Rules) > 0 {
		b.WriteString("\n### Required Method\n\n")
		for _, rule := range contract.Rules {
			fmt.Fprintf(b, "- %s.\n", trimTrailingPeriod(rule))
		}
	}
	if contract.QualityFinding != "" {
		b.WriteString("\n### Quality Gate\n\n")
		if contract.Constraint != "" {
			fmt.Fprintf(b, "- Constraint balance: %s.\n", trimTrailingPeriod(contract.Constraint))
		}
		fmt.Fprintf(b, "- %s.\n", trimTrailingPeriod(contract.QualityFinding))
	}
	b.WriteString("\n")
}

func optionalArtifactNote(project string, rel string) string {
	info, err := os.Stat(projectAbsPath(project, rel))
	display := displayProjectPath(project, rel)
	if err != nil {
		return display + " missing"
	}
	if info.IsDir() {
		return display + " is a directory"
	}
	if info.Size() == 0 {
		return display + " is empty"
	}
	return display + " present"
}

func sourceKindSummary(sources []tape.Source) string {
	counts := map[string]int{}
	for _, src := range sources {
		kind := "unclassified"
		if src.Kind != nil && strings.TrimSpace(*src.Kind) != "" {
			kind = strings.TrimSpace(*src.Kind)
		}
		counts[kind]++
	}
	return countSummary(counts)
}

func sourceSectionSummary(sources []tape.Source) string {
	counts := map[string]int{}
	for _, src := range sources {
		section := "unsectioned"
		if src.Section != nil && strings.TrimSpace(*src.Section) != "" {
			section = strings.TrimSpace(*src.Section)
		}
		counts[section]++
	}
	return countSummary(counts)
}

func compiledSourceAvailabilitySummary(project string, savedSources int) string {
	sourceFiles, unavailable := compiledSourceAvailability(project)
	if sourceFiles == 0 {
		if savedSources == 0 {
			return "no saved sources yet"
		}
		return fmt.Sprintf("no compiled source files detected for %d saved source(s)", savedSources)
	}
	usable := sourceFiles - unavailable
	if usable < 0 {
		usable = 0
	}
	if unavailable > 0 {
		return fmt.Sprintf("%d usable compiled source file(s), %d unavailable placeholder(s); do not cite unavailable sources as evidence", usable, unavailable)
	}
	if savedSources > 0 && sourceFiles != savedSources {
		return fmt.Sprintf("%d compiled source file(s) for %d saved source(s); verify missing entries before relying on them", sourceFiles, savedSources)
	}
	return fmt.Sprintf("%d usable compiled source file(s)", sourceFiles)
}

func compiledSourceAvailability(project string) (int, int) {
	entries, err := os.ReadDir(projectAbsPath(project, "sources"))
	if err != nil {
		return 0, 0
	}
	total := 0
	unavailable := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		total++
		body, err := os.ReadFile(projectAbsPath(project, filepath.Join("sources", entry.Name())))
		if err == nil && strings.Contains(string(body), "_Source unavailable") {
			unavailable++
		}
	}
	return total, unavailable
}

func countSummary(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func projectSkillProposal(current tape.Tape) (string, string) {
	base := fallbackText(slug(current.Title), "project")
	base = strings.TrimPrefix(base, "liner-")
	name := "liner-" + base
	return name, "SKILL.md"
}

func writeProjectSkillFile(project string, current tape.Tape) (string, string, error) {
	name, relPath := projectSkillProposal(current)
	path := filepath.Join(project, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", err
	}
	jtbd := "No job is recorded. Stay inside the documented project scope."
	if current.JTBD != nil && strings.TrimSpace(*current.JTBD) != "" {
		jtbd = strings.TrimSpace(*current.JTBD)
	}
	mixtapePath := displayProjectPath(project, "MIXTAPE.md")
	sourcesPath := strings.TrimSuffix(displayProjectPath(project, "sources"), "/")
	description := fallbackText(current.Description, "Use this Liner project's corpus.")
	contract := readOperatingCorpusContract(project)
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", name)
	fmt.Fprintf(&b, "description: %s\n", yamlSingleQuotedScalar(fmt.Sprintf("Use the %s Liner project. %s", fallbackText(current.Title, name), description)))
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s\n\n", name)
	b.WriteString("## Use When\n\n")
	b.WriteString("Use this Project Skill when an AI agent needs this Liner project's sources for its job:\n\n")
	fmt.Fprintf(&b, "> %s\n\n", jtbd)
	b.WriteString("## Source Grounding\n\n")
	b.WriteString("- Load `LINER.md` first.\n")
	fmt.Fprintf(&b, "- Load `%s` before making source-backed claims.\n", mixtapePath)
	fmt.Fprintf(&b, "- Treat source files under `%s/` as deeper evidence when an answer depends on detail.\n\n", sourcesPath)
	writeSkillCorpusMethod(&b, contract)
	if projectSkillProfileFor(current).Full {
		b.WriteString("## Behavior\n\n")
		b.WriteString("1. Restate the user's request in the language of this Liner project.\n")
		b.WriteString("2. Identify the relevant corpus stance, source section, or boundary.\n")
		b.WriteString("3. Produce the smallest useful answer, critique, plan, or question grounded in that evidence.\n")
		b.WriteString("4. Name missing evidence when the corpus cannot support a strong answer.\n\n")
	}
	b.WriteString("## Boundaries\n\n")
	b.WriteString("- Use this Project Skill only for this Liner project's job and sources.\n")
	b.WriteString("- Do not override `LINER.md`, source hierarchy, conflict rules, or abstention rules.\n")
	b.WriteString("- Draft changes under `working/` and require review before changing project files.\n")
	body := b.String()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", "", err
	}
	legacyPath := filepath.Join(project, "skills", name+".md")
	_ = os.Remove(legacyPath)
	_ = os.Remove(filepath.Dir(legacyPath))
	return name, relPath, nil
}

func writeSkillCorpusMethod(b *strings.Builder, contract operatingCorpusContract) {
	if !contract.hasContent() {
		return
	}
	b.WriteString("## Corpus Method\n\n")
	if contract.CoreAction != "" {
		fmt.Fprintf(b, "- Core action: %s.\n", trimTrailingPeriod(contract.CoreAction))
	}
	if contract.Pattern != "" {
		fmt.Fprintf(b, "- Capability pattern: %s.\n", trimTrailingPeriod(contract.Pattern))
	}
	if contract.Translation != "" {
		fmt.Fprintf(b, "- Translation evidence: %s.\n", trimTrailingPeriod(contract.Translation))
	}
	if contract.CallerHandoff != "" {
		fmt.Fprintf(b, "- Caller handoff: %s.\n", trimTrailingPeriod(contract.CallerHandoff))
	}
	if len(contract.Rules) > 0 {
		b.WriteString("- Apply these corpus-derived rules:\n")
		for _, rule := range contract.Rules {
			fmt.Fprintf(b, "  - %s.\n", trimTrailingPeriod(rule))
		}
	}
	b.WriteString("\n")
}

func markdownHeadingBody(markdown string, headingPrefix string) string {
	lines := strings.Split(markdown, "\n")
	start := -1
	prefix := strings.ToLower(strings.TrimSpace(headingPrefix))
	for index, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(normalized, prefix) {
			start = index
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), "## ") {
			end = index
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func firstMarkdownParagraph(markdown string) string {
	var paragraph []string
	for _, rawLine := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case line == "":
			if len(paragraph) > 0 {
				return normalizeInlineMarkdown(strings.Join(paragraph, " "))
			}
		case strings.HasPrefix(line, "#"):
			if len(paragraph) > 0 {
				return normalizeInlineMarkdown(strings.Join(paragraph, " "))
			}
		case strings.HasPrefix(line, "---"):
			continue
		default:
			paragraph = append(paragraph, line)
		}
	}
	if len(paragraph) == 0 {
		return ""
	}
	return normalizeInlineMarkdown(strings.Join(paragraph, " "))
}

func markdownBullets(markdown string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	bullets := []string{}
	for _, rawLine := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(rawLine, " ") || strings.HasPrefix(rawLine, "\t") {
			continue
		}
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		bullet := normalizeInlineMarkdown(strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		if bullet == "" {
			continue
		}
		bullets = append(bullets, bullet)
		if len(bullets) >= limit {
			break
		}
	}
	return bullets
}

func normalizeInlineMarkdown(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func trimTrailingPeriod(value string) string {
	value = strings.TrimSpace(value)
	return strings.TrimRight(value, ".")
}

func yamlSingleQuotedScalar(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		value = "Use this Liner project."
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func writeOperatingLayerMetadata(project string, skillName string, skillPath string) error {
	path := filepath.Join(project, "liner.yaml")
	metadata := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		if err := yaml.Unmarshal(raw, &metadata); err != nil {
			return err
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["version"] = 2
	metadata["artifact"] = "liner"
	metadata["mixtape"] = "mixtape"
	projectSkill := map[string]any{
		"status": "active",
		"name":   skillName,
		"path":   filepath.ToSlash(skillPath),
	}
	metadata["project_skill"] = projectSkill
	metadata["status"] = map[string]any{
		"milestone": "project_complete",
		"stale":     false,
		"updated":   time.Now().UTC().Format(time.RFC3339),
		"corpus": map[string]any{
			"state":    "ready",
			"evidence": "mixtape/MIXTAPE.md",
		},
		"operating_layer": map[string]any{
			"state":    "ready",
			"evidence": "LINER.md",
		},
	}
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
