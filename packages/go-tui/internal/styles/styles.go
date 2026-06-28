package styles

import "charm.land/lipgloss/v2"

var (
	Ink        = lipgloss.Color("#F7F7F2")
	Muted      = lipgloss.Color("#8B8C89")
	SoftMuted  = lipgloss.Color("#C3C5CC")
	Accent     = lipgloss.Color("#FF5A1F")
	AccentTwo  = lipgloss.Color("#C3C5CC")
	Orange     = lipgloss.Color("#FF5A1F")
	OrangeSoft = lipgloss.Color("#FF9A66")
	Teal       = lipgloss.Color("#2EE6BF")
	Success    = lipgloss.Color("#7ACB7A")
	Warning    = lipgloss.Color("#F2C94C")
	Error      = lipgloss.Color("#FF5C5C")
	Panel      = lipgloss.Color("#30323D")

	Title = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	Brand = lipgloss.NewStyle().
		Foreground(Ink).
		Bold(true)

	BrandDot = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	BrandDim = lipgloss.NewStyle().
			Foreground(Muted)

	SlashA = lipgloss.NewStyle().
		Foreground(SoftMuted)

	SlashB = lipgloss.NewStyle().
		Foreground(Accent)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted)

	SoftText = lipgloss.NewStyle().
			Foreground(SoftMuted)

	Section = lipgloss.NewStyle().
		Foreground(AccentTwo)

	ReportSection = lipgloss.NewStyle().
			Foreground(Muted).
			Bold(true)

	ReportBody = lipgloss.NewStyle().
			Foreground(Ink)

	PrimaryText = lipgloss.NewStyle().
			Foreground(Ink)

	MutedText = lipgloss.NewStyle().
			Foreground(Muted)

	DimText = lipgloss.NewStyle().
		Foreground(SoftMuted)

	AccentText = lipgloss.NewStyle().
			Foreground(Accent)

	WarningText = lipgloss.NewStyle().
			Foreground(Warning)

	ReportListMarker = lipgloss.NewStyle().
				Foreground(Success)

	ReportListAccent = lipgloss.NewStyle().
				Foreground(Accent)

	ActivityPrompt = lipgloss.NewStyle().
			Foreground(Accent)

	ActivityText = lipgloss.NewStyle().
			Foreground(Muted)

	ActivityHot = lipgloss.NewStyle().
			Foreground(Ink).
			Background(Accent)

	NextActionTitle = lipgloss.NewStyle().
			Foreground(Orange)

	NextCueTitle = lipgloss.NewStyle().
			Foreground(OrangeSoft)

	NextActionText = lipgloss.NewStyle().
			Foreground(Muted)

	Help = lipgloss.NewStyle().
		Foreground(Muted)

	ErrorText = lipgloss.NewStyle().
			Foreground(Error)

	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	TableHeader = lipgloss.NewStyle().
			Foreground(Muted).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Panel).
			BorderBottom(true).
			Bold(false)

	TableSelected = lipgloss.NewStyle().
			Foreground(Ink).
			Bold(false)

	TableSelectedFocused = TableSelected.
				Background(Panel)

	InputFocused = lipgloss.NewStyle().
			Foreground(Ink)

	InputBlurred = lipgloss.NewStyle().
			Foreground(SoftMuted)

	InputPlaceholder = lipgloss.NewStyle().
				Foreground(Muted)

	InputCursor = Accent

	ProgressTrack = Panel
	ProgressFill  = Accent
)

func ClampWidth(width int) int {
	if width < 60 {
		return 60
	}
	if width > 118 {
		return 118
	}
	return width
}
