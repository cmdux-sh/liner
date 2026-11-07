package navigator

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/styles"
)

// row is one line of a list. Rows with no target can't be selected.
type row struct {
	cells      []cell
	selectable bool
	slug       string
	path       string
	title      string
	evidence   bool
	line       int
	filter     string
	connect    string
	context    string
}

// cell is one column. Width 0 takes what's left.
type cell struct {
	text  string
	width int
	color any
	right bool
	bold  bool
}

func textRow(text string, color any) row {
	return row{cells: []cell{{text: text, color: color}}}
}

func blank() row {
	return row{}
}

func nearestSelectable(rows []row, cursor int) int {
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(rows) {
		cursor = len(rows) - 1
	}
	for index := cursor; index >= 0 && index < len(rows); index++ {
		if rows[index].selectable {
			return index
		}
	}
	for index := cursor; index >= 0 && index < len(rows); index-- {
		if rows[index].selectable {
			return index
		}
	}
	return 0
}

func stepSelectable(rows []row, cursor, step int) int {
	for index := cursor + step; index >= 0 && index < len(rows); index += step {
		if rows[index].selectable {
			return index
		}
	}
	return cursor
}

func (m Model) rowsFor(page *page) []row {
	switch page.kind {
	case pageLiner:
		return m.linerRows(page.slug)
	case pageSettings:
		return m.settingsRows()
	case pageLibrary:
		switch {
		case page.view == viewAttention:
			return m.attentionRows()
		case page.view == viewSources:
			return m.sourceRows(page)
		case page.view == viewConnections:
			return m.connectionRows()
		}
		return m.allRows()
	}
	return nil
}

func style(color any) lipgloss.Style {
	base := lipgloss.NewStyle()
	if value, ok := color.(interface {
		RGBA() (uint32, uint32, uint32, uint32)
	}); ok {
		base = base.Foreground(value)
	}
	return base
}

func clip(text string, width int) string {
	runes := []rune(text)
	if width <= 0 {
		return ""
	}
	if len(runes) <= width {
		return text
	}
	if width <= 2 {
		return string(runes[:width])
	}
	return string(runes[:width-2]) + ".."
}

func pad(text string, width int, right bool) string {
	if width <= 0 {
		return ""
	}
	text = clip(text, width)
	space := strings.Repeat(" ", width-len([]rune(text)))
	if right {
		return space + text
	}
	return text + space
}

func (m Model) renderRow(item row, selected bool) string {
	width := m.width - 2
	used := 0
	flexible := -1
	for index, column := range item.cells {
		if column.width == 0 {
			flexible = index
		}
		used += column.width
	}
	var line strings.Builder
	for index, column := range item.cells {
		columnWidth := column.width
		if index == flexible {
			columnWidth = width - used
		}
		text := pad(column.text, columnWidth, column.right)
		cellStyle := style(column.color).Bold(column.bold)
		if selected {
			cellStyle = cellStyle.Background(styles.Panel)
			if column.color == nil {
				cellStyle = cellStyle.Foreground(styles.Ink)
			}
		}
		line.WriteString(cellStyle.Render(text))
	}
	if selected {
		return lipgloss.NewStyle().Foreground(styles.Accent).Background(styles.Panel).Render("▌ ") + line.String()
	}
	return "  " + line.String()
}

func stateColor(state string) any {
	switch state {
	case "waiting":
		return styles.Accent
	case "ready":
		return styles.Success
	case "draft":
		return styles.Muted
	}
	return styles.Warning
}

func stateOrder(state string) int {
	switch state {
	case "waiting":
		return 0
	case "building":
		return 1
	case "sources moved":
		return 2
	case "ready":
		return 3
	}
	return 4
}

func stateLabel(liner Liner) string {
	if liner.State == "building" && liner.BuildStep != nil {
		return fmt.Sprintf("building %d/%d", *liner.BuildStep, liner.BuildSteps)
	}
	return liner.State
}

func count(value *int) string {
	if value == nil {
		return "?"
	}
	return fmt.Sprint(*value)
}

func testsLabel(tests *Tests) string {
	if tests == nil {
		return "?"
	}
	if tests.Runs == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d", tests.Better, tests.Runs)
}

func jobOf(liner Liner) string {
	if liner.Job != "" {
		return liner.Job
	}
	if liner.Description != "" {
		return liner.Description
	}
	return liner.Title
}

func (m Model) allRows() []row {
	rows := []row{{cells: []cell{
		{text: "NAME", width: 20, color: styles.Muted},
		{text: "JOB", color: styles.Muted},
		{text: "STATE", width: 15, color: styles.Muted},
		{text: "SRC", width: 4, color: styles.Muted, right: true},
		{text: "GAPS", width: 6, color: styles.Muted, right: true},
		{text: "FOUND", width: 7, color: styles.Muted, right: true},
		{text: "TESTS", width: 7, color: styles.Muted, right: true},
	}}}
	liners := append([]Liner(nil), m.library.Liners...)
	sort.SliceStable(liners, func(a, b int) bool { return stateOrder(liners[a].State) < stateOrder(liners[b].State) })
	for _, liner := range liners {
		rows = append(rows, row{selectable: true, slug: liner.Slug, cells: []cell{
			{text: liner.Slug, width: 20},
			{text: jobOf(liner)},
			{text: stateLabel(liner), width: 15, color: stateColor(liner.State)},
			{text: fmt.Sprint(liner.SourceCount), width: 4, right: true},
			{text: count(liner.OpenGaps), width: 6, right: true},
			{text: count(liner.Pending), width: 7, right: true},
			{text: testsLabel(liner.Test), width: 7, right: true},
		}})
	}
	return rows
}

var attentionGroups = []struct {
	key   string
	label string
	color any
}{
	{"waiting", "WAITING FOR YOU", styles.Accent},
	{"changed", "HAS CHANGED", styles.Warning},
	{"stronger", "COULD BE STRONGER", styles.Warning},
	{"working", "WORKING NOW", styles.Muted},
}

func (m Model) attentionRows() []row {
	var rows []row
	for _, group := range attentionGroups {
		var items []row
		for _, item := range m.library.Attention {
			if item.Group == group.key {
				items = append(items, row{
					selectable: true, slug: item.Slug, context: item.Context,
					cells: []cell{{text: item.Title, width: 46}, {text: item.Slug}},
				})
			}
		}
		if len(items) == 0 {
			continue
		}
		if len(rows) > 0 {
			rows = append(rows, blank())
		}
		rows = append(rows, textRow(group.label, group.color))
		rows = append(rows, items...)
	}
	for index, broken := range m.library.Unreadable {
		if index == 0 {
			if len(rows) > 0 {
				rows = append(rows, blank())
			}
			rows = append(rows, textRow("CAN'T READ", styles.Error))
		}
		rows = append(rows, row{cells: []cell{{text: shortFolder(broken.Path), width: 24}, {text: broken.Reason, color: styles.Muted}}})
	}
	if len(rows) == 0 {
		return []row{textRow("Nothing needs you right now.", styles.Muted)}
	}
	return rows
}

func shortFolder(path string) string {
	return shortFile(strings.ReplaceAll(path, string(os.PathSeparator), "/"))
}

func sourceKind(source Source) string {
	if source.Kind == nil {
		return "unspecified"
	}
	return *source.Kind
}

func (m Model) pageSources(page *page) []Source {
	sources := m.library.Sources
	if page.filter != "" {
		if liner := m.liner(page.filter); liner != nil {
			sources = liner.Sources
		}
	}
	return sources
}

func (m Model) sourceValues(page *page, value func(Source) string) []string {
	seen := map[string]bool{}
	var values []string
	for _, source := range m.pageSources(page) {
		if key := value(source); !seen[key] {
			seen[key] = true
			values = append(values, key)
		}
	}
	sort.Strings(values)
	return values
}

func tally(sources []Source, value func(Source) string) ([]string, map[string]int) {
	counts := map[string]int{}
	for _, source := range sources {
		counts[value(source)]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(a, b int) bool {
		if counts[keys[a]] != counts[keys[b]] {
			return counts[keys[a]] > counts[keys[b]]
		}
		return keys[a] < keys[b]
	})
	return keys, counts
}

func bar(value, largest, width int) string {
	if largest == 0 || value == 0 {
		return ""
	}
	size := value * width / largest
	if size < 1 {
		size = 1
	}
	return strings.Repeat("=", size)
}

func (m Model) sourceRows(page *page) []row {
	all := m.pageSources(page)
	var rows []row
	types, typeCounts := tally(all, func(s Source) string { return s.Type })
	kinds, kindCounts := tally(all, sourceKind)
	lines := len(types)
	if len(kinds) > lines {
		lines = len(kinds)
	}
	if page.filter != "" {
		rows = append(rows, textRow("SOURCES IN "+strings.ToUpper(page.filter), styles.Ink))
	} else {
		rows = append(rows, textRow("YOUR COLLECTION", styles.Ink))
	}
	for index := 0; index < lines; index++ {
		cells := make([]cell, 0, 6)
		if index < len(types) {
			key := types[index]
			cells = append(cells,
				cell{text: key, width: 13},
				cell{text: bar(typeCounts[key], typeCounts[types[0]], 24), width: 25, color: styles.Accent},
				cell{text: fmt.Sprint(typeCounts[key]), width: 6},
			)
		} else {
			cells = append(cells, cell{width: 44})
		}
		if index < len(kinds) {
			key := kinds[index]
			cells = append(cells,
				cell{text: key, width: 14},
				cell{text: bar(kindCounts[key], kindCounts[kinds[0]], 16), width: 17, color: styles.OrangeSoft},
				cell{text: fmt.Sprint(kindCounts[key])},
			)
		} else {
			cells = append(cells, cell{})
		}
		rows = append(rows, row{cells: cells})
	}
	filters := "all sources"
	if page.typeOnly != "" || page.kindOnly != "" {
		filters = "showing " + strings.TrimSpace(page.typeOnly+" "+page.kindOnly)
	}
	rows = append(rows, blank(), textRow(filters, styles.Muted))
	rows = append(rows, row{cells: []cell{
		{text: "SOURCE", color: styles.Muted},
		{text: "TYPE", width: 11, color: styles.Muted},
		{text: "KIND", width: 14, color: styles.Muted},
		{text: "USED IN", width: 18, color: styles.Muted},
		{text: "FETCHED", width: 10, color: styles.Muted},
	}})
	shown := make([]Source, 0, len(all))
	for _, source := range all {
		if (page.typeOnly == "" || source.Type == page.typeOnly) && (page.kindOnly == "" || sourceKind(source) == page.kindOnly) {
			shown = append(shown, source)
		}
	}
	if page.byName {
		sort.SliceStable(shown, func(a, b int) bool {
			return strings.ToLower(shown[a].Title) < strings.ToLower(shown[b].Title)
		})
	}
	for _, source := range shown {
		rows = append(rows, m.sourceRow(page, source))
	}
	return rows
}

func usedIn(source Source, fallback string) (string, any) {
	switch len(source.Liners) {
	case 0:
		return fallback, nil
	case 1:
		return source.Liners[0], nil
	}
	return fmt.Sprintf("%d liners", len(source.Liners)), styles.OrangeSoft
}

func (m Model) sourceRow(page *page, source Source) row {
	used, usedColor := usedIn(source, page.filter)
	fetched, fetchedColor := m.ago(source.FetchedAt), any(nil)
	if source.Changed {
		fetched, fetchedColor = "changed", styles.Warning
	}
	slug := page.filter
	if len(source.Liners) > 0 {
		slug = source.Liners[0]
	}
	item := row{selectable: true, slug: slug, title: source.Title, evidence: true, context: "source", cells: []cell{
		{text: source.Title},
		{text: source.Type, width: 11},
		{text: sourceKind(source), width: 14},
		{text: used, width: 18, color: usedColor},
		{text: fetched, width: 10, color: fetchedColor},
	}}
	if source.File != nil {
		item.path = *source.File
	}
	return item
}

func (m Model) ago(stamp *string) string {
	if stamp == nil {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, *stamp)
	if err != nil {
		return "-"
	}
	return humanAge(m.now.Sub(parsed))
}

func humanAge(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "now"
	case age < time.Hour:
		return fmt.Sprintf("%d min", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%d hours", int(age.Hours()))
	case age < 14*24*time.Hour:
		return fmt.Sprintf("%d days", int(age.Hours()/24))
	}
	return fmt.Sprintf("%d weeks", int(age.Hours()/24/7))
}

func (m Model) connectionRows() []row {
	citations := m.library.Connections.Citations
	rows := []row{textRow("HOW YOUR LINERS CITE EACH OTHER", styles.Ink)}
	if len(citations) == 0 {
		rows = append(rows, textRow("No liner cites another yet.", styles.Muted))
	}
	cited := map[string][]Citation{}
	var roots []string
	for _, citation := range citations {
		if _, seen := cited[citation.To]; !seen {
			roots = append(roots, citation.To)
		}
		cited[citation.To] = append(cited[citation.To], citation)
	}
	sort.Strings(roots)
	var stale []Citation
	for _, root := range roots {
		rows = append(rows, row{selectable: true, slug: root, cells: []cell{{text: root}}})
		for index, citation := range cited[root] {
			branch := "|-- "
			if index == len(cited[root])-1 {
				branch = "\\-- "
			}
			rows = append(rows, row{selectable: true, slug: citation.From, cells: []cell{{text: branch + citation.From}}})
			if citation.Stale {
				stale = append(stale, citation)
			}
		}
	}
	for _, slug := range m.library.Connections.Uncited {
		if len(citations) > 0 {
			rows = append(rows, row{selectable: true, slug: slug, cells: []cell{{text: slug, width: 24}, {text: "no citations", color: styles.Muted}}})
		}
	}
	shared := false
	for _, source := range m.library.Sources {
		if len(source.Liners) > 1 {
			if !shared {
				rows = append(rows, blank(), textRow("SOURCES THEY SHARE", styles.Ink))
				shared = true
			}
			item := row{selectable: true, slug: source.Liners[0], title: source.Title, evidence: true, context: "source",
				cells: []cell{{text: source.Title, width: 44}, {text: strings.Join(source.Liners, ", "), color: styles.Muted}}}
			if source.File != nil {
				item.path = *source.File
			}
			rows = append(rows, item)
		}
	}
	for _, citation := range stale {
		rows = append(rows, blank(), textRow(fmt.Sprintf("! %s changed. %s still has the old copy.", citation.To, citation.From), styles.Warning))
	}
	return rows
}

func (m Model) linerRows(slug string) []row {
	liner := m.liner(slug)
	if liner == nil {
		return []row{textRow("This liner is gone.", styles.Muted)}
	}
	label := func(text string) cell { return cell{text: text, width: 13} }
	empty := cell{width: 13}
	rows := []row{textRow(jobOf(*liner), styles.Ink)}
	if liner.WaitingOn != nil {
		rows = append(rows, textRow("! Waiting for you: "+strings.ToLower(*liner.WaitingOn), styles.Accent))
	} else if liner.State == "building" {
		rows = append(rows, textRow("Your agent is building this.", styles.Warning))
	}
	rows = append(rows, blank())
	if liner.Synthesis != nil {
		synthesis := *liner.Synthesis
		rows = append(rows, row{selectable: true, slug: slug, path: synthesis.Path, title: "synthesis", cells: []cell{
			label("SYNTHESIS"),
			{text: fmt.Sprintf("%d sections, %d lines, %d citations", synthesis.Sections, synthesis.Lines, synthesis.Citations)},
		}}, blank())
	}
	first := true
	largest := 0
	for _, section := range liner.Sections {
		if section.Sources > largest {
			largest = section.Sources
		}
	}
	for _, section := range liner.Sections {
		name := empty
		if first {
			name = label(fmt.Sprintf("SOURCES %d", liner.SourceCount))
			first = false
		}
		rows = append(rows, row{selectable: true, slug: slug, filter: slug, cells: []cell{
			name,
			{text: section.Name, width: 26},
			{text: bar(section.Sources, largest, 12), width: 13, color: styles.Accent},
			{text: fmt.Sprint(section.Sources)},
		}})
	}
	rows = append(rows, blank())
	inbox := len(liner.Gaps) + len(liner.Findings)
	name := label(fmt.Sprintf("INBOX %d", inbox))
	if inbox == 0 {
		rows = append(rows, row{cells: []cell{name, {text: "empty", color: styles.Muted}}})
	}
	for _, finding := range liner.Findings {
		rows = append(rows, row{selectable: true, slug: slug, context: "finding", cells: []cell{name, {text: "finding", width: 9, color: styles.OrangeSoft}, {text: finding.Finding}}})
		name = empty
	}
	for _, gap := range liner.Gaps {
		rows = append(rows, row{selectable: true, slug: slug, context: "gap", cells: []cell{name, {text: "gap", width: 9, color: styles.Warning}, {text: gap.Question}}})
		name = empty
	}
	rows = append(rows, blank())
	rows = append(rows, row{cells: []cell{label("HEALTH"), {text: refreshLine(m, *liner)}}})
	rows = append(rows, row{cells: []cell{empty, {text: testLine(liner.Test)}}})
	rows = append(rows, row{cells: []cell{empty, {text: citeLine(*liner)}}}, blank())
	name = label("FILES")
	for _, path := range liner.Files {
		rows = append(rows, row{selectable: true, slug: slug, path: path, title: shortPath(path, liner.Path), cells: []cell{name, {text: shortPath(path, liner.Path)}}})
		name = empty
	}
	return rows
}

func shortPath(path, base string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, base), string(os.PathSeparator))
}

func refreshLine(m Model, liner Liner) string {
	if liner.RefreshedAt == nil {
		return "Sources not checked online yet"
	}
	changed := 0
	for _, source := range liner.Sources {
		if source.Changed {
			changed++
		}
	}
	if changed > 0 {
		return fmt.Sprintf("Checked %s ago, %d changed", m.ago(liner.RefreshedAt), changed)
	}
	return fmt.Sprintf("Checked %s ago, none changed", m.ago(liner.RefreshedAt))
}

func testLine(tests *Tests) string {
	if tests == nil || tests.Runs == 0 {
		return "Not tested yet"
	}
	line := fmt.Sprintf("Better in %d of %d tests", tests.Better, tests.Runs)
	if !tests.Current {
		line += ", changed since the last one"
	}
	return line
}

func citeLine(liner Liner) string {
	cites := "Cites nothing"
	if len(liner.Cites) > 0 {
		cites = "Cites " + strings.Join(liner.Cites, ", ")
	}
	if len(liner.CitedBy) > 0 {
		return cites + ", cited by " + strings.Join(liner.CitedBy, ", ")
	}
	return cites
}

func firstAgent() string {
	if _, err := exec.LookPath("claude"); err == nil {
		return "claude"
	}
	if _, err := exec.LookPath("codex"); err == nil {
		return "codex"
	}
	return ""
}

func connectionLabel(status string) (string, any) {
	switch status {
	case "current", "installed", "update_required":
		return "connected", styles.Success
	case "absent", "missing":
		return "not connected", styles.Muted
	}
	return status, styles.Warning
}

func (m Model) settingsRows() []row {
	rows := []row{
		textRow("LIBRARY", styles.Muted),
		{cells: []cell{{text: "Folder", width: 20}, {text: m.client.Root, width: 56}, {text: fmt.Sprintf("%d liners", len(m.library.Liners))}}},
		blank(), textRow("YOUR AGENTS", styles.Muted),
	}
	if m.agents == nil {
		rows = append(rows, textRow("Checking...", styles.Muted))
	} else {
		claude, claudeColor := connectionLabel(m.agents.Claude)
		codex, codexColor := connectionLabel(m.agents.Codex)
		subagent, subagentColor := connectionLabel(m.agents.Subagent)
		rows = append(rows,
			row{selectable: true, connect: "claude", cells: []cell{{text: "Claude Code", width: 20}, {text: claude, color: claudeColor}}},
			row{selectable: true, connect: "codex", cells: []cell{{text: "Codex", width: 20}, {text: codex, color: codexColor}}},
			row{cells: []cell{{text: "MCP", width: 20}, {text: "for agents with no shell, use the command liner mcp", color: styles.Muted}}},
			blank(), textRow("SAFER READING", styles.Muted),
			row{cells: []cell{{text: "Research subagent", width: 20}, {text: subagent, color: subagentColor}}},
		)
	}
	return append(rows, blank(), textRow("THIS VERSION", styles.Muted), row{cells: []cell{{text: "liner " + m.version}}})
}

func readAll(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func (m Model) bodyHeight() int {
	height := m.height - 6
	if height < 3 {
		height = 3
	}
	return height
}

// View draws the top page.
func (m Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	return view
}

func (m Model) rule() string {
	return lipgloss.NewStyle().Foreground(styles.Panel).Render(strings.Repeat("─", max(m.width, 0)))
}

func hints(pairs ...string) string {
	var parts []string
	for index := 0; index+1 < len(pairs); index += 2 {
		parts = append(parts, style(styles.Ink).Render(pairs[index])+" "+style(styles.Muted).Render(pairs[index+1]))
	}
	return strings.Join(parts, "   ")
}

func (m Model) brand() string {
	return style(styles.Accent).Bold(true).Render("liner")
}

func (m Model) live() string {
	var newest float64
	slug := ""
	for _, liner := range m.library.Liners {
		if liner.ChangedAt != nil && *liner.ChangedAt > newest {
			newest, slug = *liner.ChangedAt, liner.Slug
		}
	}
	if m.err != "" {
		return style(styles.Error).Render("can't read your library")
	}
	if slug == "" {
		return style(styles.Success).Render("*") + style(styles.Muted).Render(" live")
	}
	age := humanAge(m.now.Sub(time.Unix(int64(newest), 0)))
	return style(styles.Success).Render("*") + style(styles.Muted).Render(fmt.Sprintf(" live · %s updated %s", slug, agoSuffix(age)))
}

func agoSuffix(age string) string {
	if age == "now" {
		return "just now"
	}
	return age + " ago"
}

func (m Model) spread(left, right string) string {
	space := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 1 {
		space = 1
	}
	return left + strings.Repeat(" ", space) + right
}

func (m Model) render() string {
	page := m.stack[len(m.stack)-1]
	var header, body []string
	footer := ""
	switch page.kind {
	case pageLibrary:
		header, body, footer = m.renderLibrary(&page)
	case pageLiner:
		header = []string{m.spread(style(styles.Muted).Render("library > ")+style(styles.Ink).Render(page.slug), m.live()), m.rule()}
		body = m.renderRows(&page, m.height-4)
		footer = hints("enter", "read", "↑↓", "move", "y", "prompt for your agent", "o", "open in editor", "esc", "library")
	case pageReader:
		header, body, footer = m.renderReader(&page)
	case pageSettings:
		header = []string{style(styles.Muted).Render("library > ") + style(styles.Ink).Render("settings"), m.rule()}
		body = m.renderRows(&page, m.height-4)
		footer = hints("enter", "connect", "↑↓", "move", "esc", "library")
	case pageSearch:
		header, body, footer = m.renderSearch(&page)
	case pageHandoff:
		header, body, footer = m.renderHandoff(&page)
	case pageKeys:
		header, body, footer = m.renderKeys()
	}
	return m.frame(header, body, footer)
}

func (m Model) frame(header, body []string, footer string) string {
	var lines []string
	lines = append(lines, header...)
	lines = append(lines, body...)
	tail := []string{m.rule(), footer}
	if m.note != "" {
		tail = append([]string{m.rule(), style(styles.OrangeSoft).Render(clip(m.note, m.width))}, footer)
	}
	room := m.height - len(tail)
	if room < 0 {
		tail = tail[len(tail)-min(len(tail), max(m.height, 0)):]
		room = 0
	}
	if len(lines) > room {
		lines = lines[:room]
	}
	for len(lines) < room {
		lines = append(lines, "")
	}
	lines = append(lines, tail...)
	limit := lipgloss.NewStyle().MaxWidth(m.width)
	for index := range lines {
		lines[index] = limit.Render(lines[index])
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRows(page *page, room int) []string {
	rows := m.rowsFor(page)
	if room < 1 {
		room = 1
	}
	start := 0
	if page.cursor >= room {
		start = page.cursor - room + 1
	}
	end := start + room
	if end > len(rows) {
		end = len(rows)
	}
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		lines = append(lines, m.renderRow(rows[index], rows[index].selectable && index == page.cursor))
	}
	return lines
}

func (m Model) renderLibrary(page *page) ([]string, []string, string) {
	if page.filter != "" {
		header := []string{style(styles.Muted).Render("library > "+page.filter+" > ") + style(styles.Ink).Render("sources"), m.rule()}
		footer := hints("enter", "read it", "t", "type", "k", "kind", "s", "sort", "y", "prompt for your agent", "esc", "back")
		return header, m.renderRows(page, m.height-4), footer
	}
	tabs := []string{
		fmt.Sprintf("1 All %d", len(m.library.Liners)),
		fmt.Sprintf("2 Attention %d", m.attentionCount()),
		fmt.Sprintf("3 Sources %d", len(m.library.Sources)),
		fmt.Sprintf("4 Connections %d", len(m.library.Connections.Citations)),
	}
	for index := range tabs {
		if index == page.view {
			tabs[index] = style(styles.Ink).Bold(true).Render(tabs[index])
		} else {
			tabs[index] = style(styles.Muted).Render(tabs[index])
		}
	}
	header := []string{m.spread(m.brand(), m.live()), "", strings.Join(tabs, "     "), m.rule()}
	if !m.loaded {
		return header, []string{style(styles.Muted).Render("  Reading your library...")}, hints("q", "quit")
	}
	if m.err != "" && len(m.library.Liners) == 0 {
		return header, []string{style(styles.Error).Render("  " + clip(m.err, m.width-2))}, hints("g", "settings", "q", "quit")
	}
	if len(m.library.Liners) == 0 {
		return m.renderEmpty()
	}
	peek := m.peekLines(page)
	room := m.height - len(header) - 2 - len(peek)
	body := m.renderRows(page, room)
	for len(body) < room {
		body = append(body, "")
	}
	body = append(body, peek...)
	footer := hints("enter", "open", "space", "peek", "1-4", "views", "/", "search", "y", "prompt for your agent", "?", "keys")
	if page.view == viewSources {
		footer = hints("enter", "read it", "t", "type", "k", "kind", "s", "sort", "/", "search", "y", "prompt for your agent", "?", "keys")
	}
	return header, body, footer
}

func (m Model) attentionCount() int {
	total := 0
	for _, item := range m.library.Attention {
		if item.Group != "working" {
			total++
		}
	}
	return total + len(m.library.Unreadable)
}

func (m Model) peekLines(page *page) []string {
	if !page.peek {
		return nil
	}
	rows := m.rowsFor(page)
	if page.cursor < 0 || page.cursor >= len(rows) || rows[page.cursor].slug == "" {
		return nil
	}
	item := rows[page.cursor]
	liner := m.liner(item.slug)
	if liner == nil {
		return nil
	}
	lines := []string{m.rule(), style(styles.Ink).Bold(true).Render(liner.Slug)}
	for _, line := range wrap(firstParagraph(*liner), m.width) {
		if len(lines) < 5 {
			lines = append(lines, style(styles.SoftMuted).Render(line))
		}
	}
	return append(lines, style(styles.Muted).Render(clip(fmt.Sprintf("%d sources · %s · %s", liner.SourceCount, stateLabel(*liner), citeLine(*liner)), m.width)))
}

func firstParagraph(liner Liner) string {
	if liner.Synthesis != nil {
		if text, err := readAll(liner.Synthesis.Path); err == nil {
			for _, part := range strings.Split(cleanText(text), "\n\n") {
				part = strings.TrimSpace(part)
				if part != "" && !strings.HasPrefix(part, "#") {
					return citationPattern.ReplaceAllString(strings.ReplaceAll(part, "\n", " "), "")
				}
			}
		}
	}
	return jobOf(liner)
}

func (m Model) renderEmpty() ([]string, []string, string) {
	agent := firstAgent()
	found := "no agent found"
	if agent != "" {
		found = agentName(agent) + " found"
	}
	left := strings.Repeat(" ", 8)
	body := []string{
		"", left + style(styles.Ink).Bold(true).Render("Your library is empty."), "",
		left + style(styles.SoftMuted).Render("Your AI agent builds liners, and you read them here. A liner is reusable"),
		left + style(styles.SoftMuted).Render("research for one job: reviewed sources, a summary, and instructions."), "",
		left + style(styles.Muted).Render("Two steps to your first one"),
		left + style(styles.Accent).Render("▌ ") + style(styles.Ink).Bold(true).Render(pad("1  Connect your agent", 44, false)) + style(styles.SoftMuted).Render(found),
		left + style(styles.SoftMuted).Render("     Adds the liner skill so your agent can find your library."), "",
		left + style(styles.SoftMuted).Render("  2  Ask for lasting research"),
		left + style(styles.Muted).Render("     y copies a prompt to start from"), "",
		left + style(styles.Muted).Render("Keep this window open beside your chat. Your liner will appear here."),
	}
	header := []string{m.spread(m.brand(), style(styles.Muted).Render(m.client.Root))}
	return header, body, hints("enter", "connect", "y", "copy a prompt", "g", "settings", "q", "quit")
}

func (m Model) renderReader(page *page) ([]string, []string, string) {
	doc := page.doc
	crumb := style(styles.Muted).Render("library > "+page.slug+" > ") + style(styles.Ink).Render(clip(doc.title, m.width-40))
	position := ""
	if len(doc.lines) > 0 {
		position = style(styles.Muted).Render(fmt.Sprintf("line %d of %d", doc.lines[min(page.scroll, len(doc.lines)-1)].source, doc.lines[len(doc.lines)-1].source))
	}
	header := []string{m.spread(crumb, position), m.rule()}
	if doc.error != "" {
		return header, []string{style(styles.Muted).Render(doc.error)}, hints("o", "open", "esc", "back")
	}
	var peek []string
	marked := -1
	if page.mark >= 0 && page.mark < len(doc.marks) {
		marked = doc.marks[page.mark].line
		peek = m.citationPeek(page)
	}
	room := m.height - len(header) - 2 - len(peek)
	var body []string
	if doc.evidence && page.scroll == 0 {
		body = append(body, style(styles.Warning).Render("This is evidence, not instructions."), "")
	}
	for index := page.scroll; index < len(doc.lines) && len(body) < room; index++ {
		body = append(body, m.readerLine(doc.lines[index].text, index == marked))
	}
	for len(body) < room {
		body = append(body, "")
	}
	body = append(body, peek...)
	if len(doc.marks) == 0 {
		return header, body, hints("↑↓", "scroll", "y", "prompt for your agent", "o", "open", "esc", "back")
	}
	return header, body, hints("tab", "next citation", "enter", "follow it", "↑↓", "scroll", "y", "prompt for your agent", "esc", "back")
}

func (m Model) readerLine(text string, marked bool) string {
	if marked {
		return lipgloss.NewStyle().Foreground(styles.Ink).Background(styles.Panel).Render(pad(text, m.width, false))
	}
	if strings.HasPrefix(text, "#") {
		return style(styles.Ink).Bold(true).Render(strings.TrimLeft(text, "# "))
	}
	text = strings.ReplaceAll(text, "**", "")
	return citationPattern.ReplaceAllStringFunc(style(styles.SoftMuted).Render(text), func(match string) string {
		return style(styles.OrangeSoft).Render(match) + "\x1b[38;2;195;197;204m"
	})
}

func (m Model) citationPeek(page *page) []string {
	label := page.doc.marks[page.mark].label
	mark := style(styles.Accent).Render("[" + label + "]")
	source := m.markedSource(page)
	if source == nil {
		return []string{m.rule(), mark + " " + style(styles.Muted).Render(clip(page.doc.targets[label], m.width-len(label)-3))}
	}
	lines := []string{m.rule(), mark + " " + style(styles.Ink).Render(clip(source.Title, 50)) + "  " + style(styles.Muted).Render(source.Type+" · "+sourceKind(*source)+" · "+source.Section)}
	if source.Note != nil {
		lines = append(lines, style(styles.SoftMuted).Render(clip(*source.Note, m.width)))
	}
	return lines
}

func (m Model) renderSearch(page *page) ([]string, []string, string) {
	status := ""
	switch {
	case page.searching:
		status = "searching"
	case page.searched != "":
		status = fmt.Sprintf("%d passages", len(page.results))
	}
	prompt := style(styles.Accent).Bold(true).Render("/ ") + style(styles.Ink).Render(page.query) + style(styles.Accent).Render("_")
	header := []string{m.spread(prompt, style(styles.Muted).Render(status)), m.rule()}
	var body []string
	visible := (m.height - 4) / 2
	if visible < 1 {
		visible = 1
	}
	first := 0
	if page.cursor >= visible {
		first = page.cursor - visible + 1
	}
	for index, result := range page.results {
		if index < first || index >= first+visible {
			continue
		}
		selected := index == page.cursor
		where := fmt.Sprintf("%s:%d", shortFile(result.File), result.Line)
		title := m.renderRow(row{cells: []cell{{text: result.Liner, width: 20, color: styles.Muted}, {text: result.Heading}, {text: where, width: 30, right: true, color: styles.Muted}}}, selected)
		snippet := m.renderRow(row{cells: []cell{{text: cleanSnippet(result.Snippet), color: styles.Muted}}}, selected)
		body = append(body, title, snippet)
	}
	if page.searched != "" && len(page.results) == 0 && !page.searching {
		body = []string{style(styles.Muted).Render("  Nothing found.")}
	}
	return header, body, hints("enter", "search, then read", "↑↓", "move", "esc", "close")
}

func cleanSnippet(snippet string) string {
	text := citationPattern.ReplaceAllString(snippet, "")
	text = strings.NewReplacer("**", "", "##", "", "# ", "").Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func shortFile(file string) string {
	parts := strings.Split(file, "/")
	return parts[len(parts)-1]
}

func (m Model) renderHandoff(page *page) ([]string, []string, string) {
	header := []string{style(styles.Ink).Bold(true).Render("Ask your agent"), m.rule()}
	body := []string{""}
	if len(page.prompts) == 0 {
		body = append(body, style(styles.Muted).Render("  Writing prompts..."))
	}
	for index, prompt := range page.prompts {
		marker, name := "  ", style(styles.SoftMuted).Render(prompt.Label)
		if index == page.cursor {
			marker, name = style(styles.Accent).Render("▌ "), style(styles.Ink).Bold(true).Render(prompt.Label)
		}
		body = append(body, marker+name)
		for _, line := range wrap(prompt.Prompt, m.readWidth()-4) {
			body = append(body, "  "+style(styles.Muted).Render(line))
		}
		body = append(body, "")
	}
	return header, body, hints("enter", "copy", "↑↓", "choose", "esc", "close") + style(styles.Muted).Render("   Nothing runs from here.")
}

func (m Model) renderKeys() ([]string, []string, string) {
	keys := [][2]string{
		{"1-4", "All, Attention, Sources, Connections"}, {"enter", "open or read"}, {"space", "peek below the table"},
		{"/", "search your library"}, {"y", "copy a prompt for your agent"}, {"tab", "next citation, in the reader"},
		{"t k s", "filter by type or kind, sort, in Sources"}, {"o", "open in your editor or browser"},
		{"g", "settings"}, {"esc", "back"}, {"q", "quit"},
	}
	body := []string{""}
	for _, key := range keys {
		body = append(body, "  "+style(styles.Ink).Render(pad(key[0], 8, false))+style(styles.SoftMuted).Render(key[1]))
	}
	return []string{style(styles.Ink).Bold(true).Render("Keys"), m.rule()}, body, hints("any key", "close")
}
