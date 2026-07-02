package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

func (m Model) viewBanner() string {
	width := chromeWidth(m.width)
	brand := styles.BrandDot.Render("o") + " " + styles.Brand.Render("liner") + " " + styles.BrandDim.Render("v1")
	if width < 50 {
		return lipgloss.NewStyle().Width(width).Render(brand + " " + styles.Subtitle.Render(m.screenLabel()))
	}
	texture := slashTexture()
	metaWidth := max(8, width-lipgloss.Width(brand)-lipgloss.Width(texture)-2)
	meta := styles.Subtitle.Render(truncateMiddle(m.bannerMeta(), metaWidth))
	return lipgloss.NewStyle().Width(width).Render(
		brand + " " + texture + " " + meta,
	)
}

func (m Model) bannerMeta() string {
	label := m.screenLabel()
	if m.screenShowsProjectMeta() && m.currentTape.Title != "" {
		return fmt.Sprintf("%s  •  %s  •  %d sources", label, m.currentTape.Title, m.visibleSourceCount())
	}
	return label
}

func (m Model) screenShowsProjectMeta() bool {
	switch m.screen {
	case screenHome, screenProjects, screenSettings, screenCreate, screenClarify, screenImport, screenOnboarding:
		return false
	default:
		return true
	}
}

func (m Model) visibleSourceCount() int {
	if (m.screen == screenSources || m.screen == screenSourceReview || m.screen == screenAssemblyReview || m.screen == screenBoard) && len(m.sourceItems) > 0 {
		return len(source.ActiveSources(m.sourceItems))
	}
	return len(m.currentTape.Sources)
}

func (m Model) screenLabel() string {
	switch m.screen {
	case screenHome:
		return "home"
	case screenProjects:
		return "projects"
	case screenProject:
		return "project"
	case screenSources:
		return "sources"
	case screenSourceReview:
		return "review"
	case screenResearch:
		return "build corpus"
	case screenAssemblyReview:
		return "assembly"
	case screenLinerReview:
		return "liner"
	case screenSkills:
		return "skills"
	case screenAudits:
		return "audits"
	case screenEvals:
		return "v2 prototype"
	case screenComposition:
		return "v2 prototype"
	case screenCompositionReview:
		return "v2 prototype"
	case screenReport:
		return "report"
	case screenBoard:
		return "board"
	case screenCompile:
		return "compile"
	case screenImprovementReview:
		return "improve"
	case screenPreview:
		return "preview"
	case screenCreate:
		return "setup"
	case screenClarify:
		return "clarify"
	case screenImport:
		return "import"
	case screenSettings:
		return "settings"
	case screenOnboarding:
		return "set up"
	default:
		return "liner"
	}
}

func slashTexture() string {
	pattern := []lipgloss.Style{
		styles.SlashB, styles.SlashB,
		styles.SlashA, styles.SlashA, styles.SlashA,
		styles.SlashB, styles.SlashB,
		styles.SlashA, styles.SlashA, styles.SlashA,
	}
	var b strings.Builder
	for repeat := 0; repeat < 3; repeat++ {
		for _, style := range pattern {
			b.WriteString(style.Render("/"))
		}
	}
	return b.String()
}

func (m Model) viewActivity() string {
	action := m.nextAction()
	if action == "" {
		return ""
	}
	return renderNextCue(action)
}

func (m Model) nextAction() string {
	switch m.screen {
	case screenCreate:
		switch m.createStep {
		case 0:
			return "Name the mixtape."
		case 1:
			return "Describe what this Liner should help your AI agent do."
		case 2:
			return "Add the curator."
		case 3:
			return "Confirm whether you want to add custom sources."
		}
	case screenClarify:
		if m.clarifyLoading {
			return "AI is working on the clarification questions."
		}
		if len(m.clarifyQuestions) == 0 {
			return "Retry clarification question generation."
		}
		if m.clarifyStep < len(m.clarifyQuestions)-1 {
			return "Answer this question, then continue."
		}
		return "Finish clarification, then start research."
	case screenSources:
		if strings.TrimSpace(m.sourceInput.Value()) != "" {
			return "Add this source."
		}
		if len(m.sourceItems) > 0 {
			return "Review the sources."
		}
		return "Paste one source, or finish without custom sources."
	case screenSourceReview:
		return "Save active sources, then clarify the AI-agent goal."
	case screenResearch:
		if !m.researchDone {
			return "Let Liner build the corpus."
		}
		return "Review the corpus artifacts on disk."
	case screenAssemblyReview:
		return "Save checked sources, then compile MIXTAPE.md."
	case screenLinerReview:
		return ""
	case screenSkills:
		return "Inspect existing skill files."
	case screenAudits:
		return "Inspect existing audit reports."
	case screenEvals:
		return "Parked V2 prototype; return to Project."
	case screenComposition:
		return "Parked V2 prototype; return to Project."
	case screenCompositionReview:
		return "Review the parked V2 draft, then apply or discard it."
	case screenReport:
		return reportNextAction(m.currentTape, m.sourceItems)
	case screenBoard:
		return "Optional: adjust personal sources, then compile the mixtape."
	case screenCompile:
		return m.compileNextActionLabel()
	case screenImprovementReview:
		return m.improvementNextAction()
	case screenPreview:
		back := m.previewBackLabel()
		if m.previewRel != "" && m.previewRel != "MIXTAPE.md" {
			return fmt.Sprintf("Read %s, or return to %s.", m.previewRel, back)
		}
		return fmt.Sprintf("Read MIXTAPE.md, copy it, or return to %s.", back)
	case screenProject:
		return m.projectMilestoneNextAction()
	case screenHome:
		return "Run the selected command."
	case screenProjects:
		return "Open a project, or return home."
	case screenImport:
		if m.importBusy {
			return "Wait for import to finish."
		}
		highlighted := m.importPicker.HighlightedPath()
		if strings.EqualFold(filepath.Ext(highlighted), ".mixtape") {
			return "Import the selected project."
		}
		if strings.TrimSpace(highlighted) != "" {
			return "Open folders until you choose a .mixtape project."
		}
		return "Choose a .mixtape project file."
	case screenOnboarding:
		if m.onboardingStep <= onboardingStepLibrary {
			if m.onboardingEditingDir {
				return "Save the projects folder."
			}
			return "Choose an AI runner."
		}
		if m.onboardingStep == onboardingStepProvider {
			return "Choose JS rendering setup."
		}
		if m.jsSetupRunning {
			return "Setup complete when JS rendering setup finishes."
		}
		return "Setup complete."
	default:
		return ""
	}
	return ""
}

func (m Model) previewBackLabel() string {
	if !m.hasPreviewBack {
		if m.currentPath == "" {
			return "Home"
		}
		return "Project"
	}
	switch m.previewBack {
	case screenCompile:
		return "Compile Console"
	case screenAssemblyReview:
		return "Review Draft Sources"
	case screenSourceReview:
		return "Review Local Sources"
	case screenSkills:
		return "Skills"
	case screenAudits:
		return "Audits"
	case screenEvals:
		return "Project"
	case screenComposition:
		return "Project"
	case screenSettings:
		return "Settings"
	case screenHome:
		return "Home"
	case screenProjects:
		return "Projects"
	default:
		return "Project"
	}
}

func chromeWidth(width int) int {
	if width <= 0 {
		return 118
	}
	if width < 64 {
		return max(20, width-2)
	}
	return styles.ClampWidth(width - 4)
}
