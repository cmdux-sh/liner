package navigator

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cmdux/liner/packages/go-tui/internal/core"
)

const refreshEvery = 3 * time.Second

type pageKind int

const (
	pageLibrary pageKind = iota
	pageLiner
	pageReader
	pageSettings
	pageSearch
	pageHandoff
	pageKeys
)

const (
	viewAll = iota
	viewAttention
	viewSources
	viewConnections
)

// page is one place in the back stack.
type page struct {
	kind      pageKind
	view      int
	slug      string
	filter    string // Shows one liner's Sources.
	cursor    int
	peek      bool
	typeOnly  string
	kindOnly  string
	byName    bool
	doc       *document
	scroll    int
	mark      int
	query     string
	searched  string
	results   []SearchResult
	searching bool
	prompts   []Prompt
}

// Options configures the navigator.
type Options struct {
	Root    string
	Version string
}

// Model is the navigator.
type Model struct {
	client  Client
	version string
	width   int
	height  int
	library Library
	loaded  bool
	loading bool
	err     string
	note    string
	now     time.Time
	agents  *Agents
	stack   []page
}

type libraryMsg struct {
	library Library
	err     error
}
type tickMsg time.Time
type searchMsg struct {
	query   string
	results []SearchResult
	err     error
}
type promptsMsg struct {
	prompts []Prompt
	err     error
}
type agentsMsg Agents
type connectedMsg struct {
	environment string
	err         error
}

// New builds the navigator.
func New(options Options) (Model, error) {
	root := options.Root
	if root == "" {
		root = core.DefaultBaseDir()
	}
	runner, err := core.Resolve()
	if err != nil {
		return Model{}, err
	}
	return newModel(Client{Runner: runner, Root: root}, options.Version), nil
}

func newModel(client Client, version string) Model {
	return Model{
		client:  client,
		version: version,
		width:   100,
		height:  32,
		now:     time.Now(),
		loading: true,
		stack:   []page{{kind: pageLibrary}},
	}
}

// Init loads the library and starts watching it.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.load(), tick())
}

func (m Model) load() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		library, err := client.Library()
		return libraryMsg{library: library, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) top() *page {
	return &m.stack[len(m.stack)-1]
}

func (m *Model) push(next page) {
	m.stack = append(m.stack, next)
	m.note = ""
}

func (m *Model) pop() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
	m.note = ""
}

func (m Model) liner(slug string) *Liner {
	for index := range m.library.Liners {
		if m.library.Liners[index].Slug == slug {
			return &m.library.Liners[index]
		}
	}
	return nil
}

// Update handles keys and results. Nothing here changes a liner.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.relayout()
		return m, nil
	case libraryMsg:
		m.loaded = true
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.library = msg.library
		m.clampCursors()
		return m, nil
	case tickMsg:
		m.now = time.Time(msg)
		if m.loading { // A slow Core must not pile up, or an old report replace a newer one.
			return m, tick()
		}
		m.loading = true
		return m, tea.Batch(m.load(), tick())
	case searchMsg:
		page := m.top()
		if page.kind == pageSearch && page.query == msg.query {
			page.searching = false
			page.searched = msg.query
			page.results = msg.results
			page.cursor = 0
			if msg.err != nil {
				m.note = msg.err.Error()
			}
		}
		return m, nil
	case promptsMsg:
		if m.top().kind != pageHandoff {
			return m, nil
		}
		if msg.err != nil {
			m.pop()
			m.note = msg.err.Error()
			return m, nil
		}
		m.top().prompts = msg.prompts
		return m, nil
	case agentsMsg:
		agents := Agents(msg)
		m.agents = &agents
		return m, nil
	case connectedMsg:
		if msg.err != nil {
			m.note = msg.err.Error()
		} else {
			m.note = "Connected. Start a new " + agentName(msg.environment) + " session to use it."
		}
		return m, m.checkAgents()
	case tea.KeyPressMsg:
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m *Model) relayout() {
	for index := range m.stack {
		if doc := m.stack[index].doc; doc != nil && doc.error == "" {
			if data, err := readAll(doc.path); err == nil {
				doc.layout(data, m.readWidth())
			}
		}
	}
	m.clampCursors()
}

func (m *Model) clampCursors() {
	for index := range m.stack {
		page := &m.stack[index]
		if page.kind == pageLibrary || page.kind == pageLiner || page.kind == pageSettings {
			m.clampCursor(page)
		}
	}
}

func (m Model) clampCursor(page *page) {
	rows := m.rowsFor(page)
	page.cursor = nearestSelectable(rows, page.cursor)
}

func (m Model) readWidth() int {
	width := m.width
	if width > 100 {
		width = 100
	}
	return width
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	page := m.top()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	switch page.kind {
	case pageSearch:
		return m.searchKey(key)
	case pageHandoff:
		return m.handoffKey(key)
	case pageKeys:
		m.pop()
		return m, nil
	}
	switch key {
	case "?":
		m.push(page2(pageKeys))
		return m, nil
	case "/":
		m.push(page2(pageSearch))
		return m, nil
	case "y":
		return m.openHandoff()
	case "g":
		if page.kind != pageSettings {
			m.push(page2(pageSettings))
			return m, m.checkAgents()
		}
	case "q":
		if len(m.stack) == 1 {
			return m, tea.Quit
		}
		m.pop()
		return m, nil
	case "esc", "backspace":
		m.pop()
		return m, nil
	}
	if page.kind == pageReader {
		return m.readerKey(key)
	}
	return m.listKey(key)
}

func page2(kind pageKind) page {
	return page{kind: kind, mark: -1}
}

func (m Model) listKey(key string) (tea.Model, tea.Cmd) {
	page := m.top()
	rows := m.rowsFor(page)
	switch key {
	case "up", "k":
		if page.kind != pageLibrary || page.view != viewSources || key == "up" {
			page.cursor = stepSelectable(rows, page.cursor, -1)
			return m, nil
		}
	case "down", "j":
		page.cursor = stepSelectable(rows, page.cursor, 1)
		return m, nil
	case "pgup":
		page.cursor = nearestSelectable(rows, page.cursor-10)
		return m, nil
	case "pgdown":
		page.cursor = nearestSelectable(rows, page.cursor+10)
		return m, nil
	}
	if page.kind == pageLibrary {
		switch key {
		case "1", "2", "3", "4":
			if page.filter == "" {
				page.view = int(key[0] - '1')
				page.cursor = 0
				m.clampCursor(page)
			}
			return m, nil
		case " ", "space":
			page.peek = !page.peek
			return m, nil
		case "t":
			page.typeOnly = nextValue(page.typeOnly, m.sourceValues(page, func(s Source) string { return s.Type }))
			page.cursor = 0
			m.clampCursor(page)
			return m, nil
		case "k":
			page.kindOnly = nextValue(page.kindOnly, m.sourceValues(page, sourceKind))
			page.cursor = 0
			m.clampCursor(page)
			return m, nil
		case "s":
			page.byName = !page.byName
			return m, nil
		}
		if len(m.library.Liners) == 0 && !m.libraryHasErrors() && key == "enter" {
			return m.connect(firstAgent())
		}
	}
	if key == "enter" {
		if page.cursor >= 0 && page.cursor < len(rows) {
			return m.follow(rows[page.cursor])
		}
	}
	if key == "o" && page.cursor >= 0 && page.cursor < len(rows) {
		row := rows[page.cursor]
		if row.path != "" {
			m.openExternal(row.path)
		}
	}
	return m, nil
}

func (m Model) libraryHasErrors() bool {
	return m.err != "" || !m.loaded
}

func (m *Model) openExternal(target string) {
	if err := openExternally(target); err != nil {
		m.note = err.Error()
	}
}

func (m Model) follow(row row) (tea.Model, tea.Cmd) {
	switch {
	case row.connect != "":
		return m.connect(row.connect)
	case row.filter != "":
		m.push(page{kind: pageLibrary, view: viewSources, filter: row.filter, slug: row.filter, mark: -1})
		m.clampCursor(m.top())
	case row.path != "":
		m.openReader(row.path, row.title, row.slug, row.evidence, row.line)
	case row.slug != "" && m.liner(row.slug) != nil:
		m.push(page{kind: pageLiner, slug: row.slug, mark: -1})
		m.clampCursor(m.top())
	}
	return m, nil
}

func (m *Model) openReader(path, title, slug string, evidence bool, line int) {
	doc := loadDocument(path, title, slug, evidence, m.readWidth())
	next := page{kind: pageReader, slug: slug, doc: &doc, mark: -1}
	if line > 0 {
		next.scroll = doc.lineFor(line)
		next.mark = -1
	}
	m.push(next)
}

func (m Model) readerKey(key string) (tea.Model, tea.Cmd) {
	page := m.top()
	doc := page.doc
	if doc == nil {
		return m, nil
	}
	switch key {
	case "up", "k":
		page.scroll--
	case "down", "j":
		page.scroll++
	case "pgup":
		page.scroll -= m.bodyHeight()
	case "pgdown", " ", "space":
		page.scroll += m.bodyHeight()
	case "tab":
		m.moveMark(page, 1)
	case "shift+tab":
		m.moveMark(page, -1)
	case "enter":
		m.followCitation(page)
	case "o":
		m.openExternal(doc.path)
	}
	page = m.top()
	if page.doc != nil {
		limit := len(page.doc.lines) - m.bodyHeight() + 1
		if page.scroll > limit {
			page.scroll = limit
		}
		if page.scroll < 0 {
			page.scroll = 0
		}
	}
	return m, nil
}

func (m Model) moveMark(page *page, step int) {
	marks := page.doc.marks
	if len(marks) == 0 {
		return
	}
	page.mark += step
	if page.mark >= len(marks) {
		page.mark = 0
	}
	if page.mark < 0 {
		page.mark = len(marks) - 1
	}
	line := marks[page.mark].line
	if line < page.scroll || line >= page.scroll+m.bodyHeight()-4 {
		page.scroll = line - 3
	}
}

func (m *Model) followCitation(page *page) {
	if page.mark < 0 {
		return
	}
	file := m.markedFile(page)
	if file == "" {
		m.note = "This citation has no compiled source to open."
		return
	}
	title := file
	if source := m.markedSource(page); source != nil {
		title = source.Title
	}
	m.openReader(file, title, page.slug, true, 0)
}

// markedFile is the compiled file behind the selected citation, as Core resolved it.
func (m Model) markedFile(page *page) string {
	if page.doc == nil || page.mark < 0 || page.mark >= len(page.doc.marks) {
		return ""
	}
	liner := m.liner(page.slug)
	if liner == nil {
		return ""
	}
	target := page.doc.targets[page.doc.marks[page.mark].label]
	if liner.Synthesis == nil || liner.Synthesis.Files == nil {
		// An older Core sends no resolved files, so match the locator as it used to.
		for _, source := range liner.Sources {
			if source.Locator == target && source.File != nil {
				return *source.File
			}
		}
		return ""
	}
	if file := liner.Synthesis.Files[target]; file != nil {
		return *file
	}
	return ""
}

// markedSource finds the Source whose compiled file the selected citation opens.
func (m Model) markedSource(page *page) *Source {
	file := m.markedFile(page)
	liner := m.liner(page.slug)
	if file == "" || liner == nil {
		return nil
	}
	for index := range liner.Sources {
		if source := &liner.Sources[index]; source.File != nil && *source.File == file {
			return source
		}
	}
	return nil
}

func (m Model) searchKey(key string) (tea.Model, tea.Cmd) {
	page := m.top()
	switch key {
	case "esc":
		m.pop()
	case "up":
		if page.cursor > 0 {
			page.cursor--
		}
	case "down":
		if page.cursor < len(page.results)-1 {
			page.cursor++
		}
	case "backspace":
		if runes := []rune(page.query); len(runes) > 0 {
			page.query = string(runes[:len(runes)-1])
		}
	case "enter":
		if page.query != page.searched {
			return m.runSearch()
		}
		if page.cursor < len(page.results) {
			result := page.results[page.cursor]
			m.openReader(result.Path, result.Heading, result.Liner, result.File != "mixtape/synthesis.md", result.Line)
		}
	case "space":
		page.query += " "
	default:
		if len([]rune(key)) == 1 {
			page.query += key
		}
	}
	return m, nil
}

func (m Model) runSearch() (tea.Model, tea.Cmd) {
	page := m.top()
	query := strings.TrimSpace(page.query)
	if query == "" {
		return m, nil
	}
	page.query = query
	page.searching = true
	client := m.client
	return m, func() tea.Msg {
		results, err := client.Search(query)
		return searchMsg{query: query, results: results, err: err}
	}
}

func (m Model) openHandoff() (tea.Model, tea.Cmd) {
	context, slug, line, subject := m.handoffContext()
	m.push(page2(pageHandoff))
	client := m.client
	return m, func() tea.Msg {
		prompts, err := client.Handoff(context, slug, line, subject)
		return promptsMsg{prompts: prompts, err: err}
	}
}

// handoffContext describes what is on screen, so the prompts fit it.
func (m Model) handoffContext() (string, string, int, string) {
	page := m.stack[len(m.stack)-1]
	if page.kind == pageReader && page.doc != nil {
		line := 0
		if page.mark >= 0 && page.mark < len(page.doc.marks) {
			line = page.doc.lines[page.doc.marks[page.mark].line].source
		} else if page.scroll < len(page.doc.lines) && len(page.doc.lines) > 0 {
			line = page.doc.lines[page.scroll].source
		}
		if page.doc.evidence {
			return "source", page.slug, line, page.doc.title
		}
		return "synthesis", page.slug, line, ""
	}
	if len(m.library.Liners) == 0 {
		return "start", "", 0, ""
	}
	rows := m.rowsFor(&page)
	if page.cursor >= 0 && page.cursor < len(rows) {
		row := rows[page.cursor]
		context := row.context
		if context == "" {
			context = "liner"
		}
		if row.slug != "" {
			return context, row.slug, 0, row.title
		}
	}
	if page.slug != "" {
		return "liner", page.slug, 0, ""
	}
	return "start", "", 0, ""
}

func (m Model) handoffKey(key string) (tea.Model, tea.Cmd) {
	page := m.top()
	switch key {
	case "esc", "y":
		m.pop()
	case "up", "k":
		if page.cursor > 0 {
			page.cursor--
		}
	case "down", "j":
		if page.cursor < len(page.prompts)-1 {
			page.cursor++
		}
	case "enter":
		if page.cursor < len(page.prompts) {
			text := page.prompts[page.cursor].Prompt
			m.pop()
			if err := copyToClipboard(text); err != nil {
				m.note = "Copy this: " + text
			} else {
				m.note = "Copied. Paste it into your agent."
			}
		}
	}
	return m, nil
}

func (m Model) checkAgents() tea.Cmd {
	client := m.client
	return func() tea.Msg { return agentsMsg(client.Agents()) }
}

func (m Model) connect(environment string) (tea.Model, tea.Cmd) {
	if environment == "" {
		m.note = "Install Claude Code or Codex first, then connect it here."
		return m, nil
	}
	m.note = "Connecting " + agentName(environment) + "..."
	client := m.client
	return m, func() tea.Msg {
		return connectedMsg{environment: environment, err: client.Connect(environment)}
	}
}

func agentName(environment string) string {
	if environment == "codex" {
		return "Codex"
	}
	return "Claude Code"
}

func nextValue(current string, values []string) string {
	if current == "" {
		if len(values) > 0 {
			return values[0]
		}
		return ""
	}
	for index, value := range values {
		if value == current && index+1 < len(values) {
			return values[index+1]
		}
	}
	return ""
}
