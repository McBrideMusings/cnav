package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pierce/cnav/internal/config"
	"github.com/pierce/cnav/internal/sessions"
	"github.com/pierce/cnav/internal/shell"
)

type sortOrder int

const (
	sortRecent sortOrder = iota
	sortName
)

type Model struct {
	sessions []*sessions.Session
	projects []*sessions.Project
	cfg      *config.Config

	cursor              int
	expanded            map[string]bool
	filter              string
	filtering           bool
	sort                sortOrder
	showAssistant       bool
	showHidden          bool
	renaming            bool
	renameBuf           string
	renameTarget        string
	width               int
	height              int
	hiddenWorktreeCount int

	Action shell.Action
	Done   bool
}

// row is one visible line in the unified list.
// session == nil means the row is a project header.
type row struct {
	project *sessions.Project
	session *sessions.Session
}

func (r row) isProject() bool { return r.session == nil }

func New(ss []*sessions.Session, hiddenWorktreeCount int, cfg *config.Config) Model {
	return Model{
		sessions:            ss,
		projects:            sessions.GroupByProject(ss),
		cfg:                 cfg,
		expanded:            map[string]bool{},
		hiddenWorktreeCount: hiddenWorktreeCount,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.renaming {
			return m.updateRename(msg)
		}
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		return m.endRename(), nil
	case tea.KeyEnter:
		m.cfg.SetName(m.renameTarget, strings.TrimSpace(m.renameBuf))
		m.saveConfig()
		return m.endRename(), nil
	case tea.KeyBackspace:
		if r := []rune(m.renameBuf); len(r) > 0 {
			m.renameBuf = string(r[:len(r)-1])
		}
	case tea.KeyRunes, tea.KeySpace:
		m.renameBuf += string(msg.Runes)
	}
	return m, nil
}

func (m Model) endRename() Model {
	m.renaming = false
	m.renameBuf = ""
	m.renameTarget = ""
	return m
}

// saveConfig persists the config and reports any error to stderr so the TUI
// keeps running. Persistence failures are non-fatal — the in-memory state still
// reflects the user's intent for the current session.
func (m Model) saveConfig() {
	if err := m.cfg.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "cnav: save config:", err)
	}
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filter = ""
		m.cursor = 0
	case tea.KeyEnter:
		m.filtering = false
	case tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.cursor = 0
		}
	case tea.KeyRunes, tea.KeySpace:
		m.filter += string(msg.Runes)
		m.cursor = 0
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.Done = true
		return m, tea.Quit
	case "j", "down":
		if m.cursor < m.maxCursor() {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = m.maxCursor()
	case "s":
		if m.sort == sortRecent {
			m.sort = sortName
		} else {
			m.sort = sortRecent
		}
		m.cursor = 0
	case "p":
		m.showAssistant = !m.showAssistant
	case "/":
		m.filtering = true
		m.filter = ""
	case " ", "space":
		return m.toggleExpand(), nil
	case "right", "l":
		return m.expandOrDescend(), nil
	case "left", "h":
		return m.collapseOrAscend(), nil
	case "R":
		return m.activateResumeLatest()
	case "H":
		m.showHidden = !m.showHidden
		m.clampCursor()
	case "x":
		return m.toggleHidden(), nil
	case "r":
		return m.beginRename(), nil
	case "enter":
		return m.activate()
	case "shift+enter":
		return m.activateCD()
	}
	return m, nil
}

func (m Model) toggleHidden() Model {
	r, ok := m.selectedRow()
	if !ok || !r.isProject() {
		return m
	}
	cwd := r.project.CWD
	m.cfg.SetHidden(cwd, !m.cfg.Lookup(cwd).Hidden)
	m.saveConfig()
	// Cursor may now point past the end if we hid the last visible row.
	m.clampCursor()
	return m
}

// clampCursor pulls the cursor back into range after the visible row set
// shrinks (hiding the last row, toggling show-hidden off, etc.).
func (m *Model) clampCursor() {
	if n := len(m.visibleRows()); n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m Model) beginRename() Model {
	r, ok := m.selectedRow()
	if !ok || !r.isProject() {
		return m
	}
	m.renaming = true
	m.renameTarget = r.project.CWD
	m.renameBuf = m.displayLabel(r.project.CWD, nil)
	return m
}

// toggleExpand flips expansion on the project owning the current row.
func (m Model) toggleExpand() Model {
	r, ok := m.selectedRow()
	if !ok {
		return m
	}
	cwd := r.project.CWD
	if m.expanded[cwd] {
		delete(m.expanded, cwd)
		// If cursor was on a child of this project, move it to the project row.
		if !r.isProject() {
			m.cursor = m.projectRowIndex(cwd)
		}
	} else {
		m.expanded[cwd] = true
	}
	return m
}

// expandOrDescend: on a collapsed project row, expand. On an already-expanded
// project row, move cursor to first child. On a chat row, no-op.
func (m Model) expandOrDescend() Model {
	r, ok := m.selectedRow()
	if !ok || !r.isProject() {
		return m
	}
	if !m.expanded[r.project.CWD] {
		m.expanded[r.project.CWD] = true
		return m
	}
	if m.cursor < m.maxCursor() {
		m.cursor++
	}
	return m
}

// collapseOrAscend: on an expanded project row, collapse. On a chat row, jump
// to parent project and collapse it.
func (m Model) collapseOrAscend() Model {
	r, ok := m.selectedRow()
	if !ok {
		return m
	}
	cwd := r.project.CWD
	if !r.isProject() {
		m.cursor = m.projectRowIndex(cwd)
	}
	delete(m.expanded, cwd)
	return m
}

func (m Model) projectRowIndex(cwd string) int {
	for i, r := range m.visibleRows() {
		if r.isProject() && r.project.CWD == cwd {
			return i
		}
	}
	return 0
}

func (m Model) selectedRow() (row, bool) {
	rows := m.visibleRows()
	if len(rows) == 0 || m.cursor >= len(rows) {
		return row{}, false
	}
	return rows[m.cursor], true
}

func (m Model) activate() (tea.Model, tea.Cmd) {
	r, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if r.isProject() {
		m.Action = shell.Action{Dir: r.project.CWD, NewClaude: true}
	} else {
		m.Action = shell.Action{Dir: r.session.CWD, Resume: r.session.ID}
	}
	m.Done = true
	return m, tea.Quit
}

func (m Model) activateCD() (tea.Model, tea.Cmd) {
	r, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	dir := r.project.CWD
	if !r.isProject() {
		dir = r.session.CWD
	}
	m.Action = shell.Action{Dir: dir}
	m.Done = true
	return m, tea.Quit
}

func (m Model) activateResumeLatest() (tea.Model, tea.Cmd) {
	r, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if !r.isProject() {
		// On a chat row, R is equivalent to enter — resume this chat.
		m.Action = shell.Action{Dir: r.session.CWD, Resume: r.session.ID}
		m.Done = true
		return m, tea.Quit
	}
	if len(r.project.Sessions) == 0 {
		return m.activate()
	}
	m.Action = shell.Action{Dir: r.project.CWD, Resume: r.project.Sessions[0].ID}
	m.Done = true
	return m, tea.Quit
}

func (m Model) maxCursor() int {
	n := len(m.visibleRows())
	if n == 0 {
		return 0
	}
	return n - 1
}

// filteredProjects narrows projects by the active filter. When a project name
// doesn't match but some of its sessions do, only those matching sessions are
// returned and the project is marked auto-expanded. Hidden projects are
// excluded unless show-hidden mode is on (decision Q5/C: filter scope follows
// show-hidden mode).
func (m Model) filteredProjects() (projs []*sessions.Project, sessionOverride map[string][]*sessions.Session, autoExpand map[string]bool) {
	visible := make([]*sessions.Project, 0, len(m.projects))
	for _, p := range m.projects {
		if m.cfg.Lookup(p.CWD).Hidden && !m.showHidden {
			continue
		}
		visible = append(visible, p)
	}
	if m.filter == "" {
		return visible, nil, nil
	}
	q := strings.ToLower(m.filter)
	sessionOverride = map[string][]*sessions.Session{}
	autoExpand = map[string]bool{}
	for _, p := range visible {
		ov := m.cfg.Lookup(p.CWD)
		nameMatch := containsCI(p.CWD, q) || containsCI(filepath.Base(p.CWD), q)
		if ov.Name != "" {
			nameMatch = nameMatch || containsCI(ov.Name, q)
		}
		var matched []*sessions.Session
		for _, s := range p.Sessions {
			if containsCI(s.Preview, q) || (m.showAssistant && containsCI(s.AssistantPreview, q)) {
				matched = append(matched, s)
			}
		}
		switch {
		case nameMatch:
			projs = append(projs, p)
		case len(matched) > 0:
			projs = append(projs, p)
			sessionOverride[p.CWD] = matched
			autoExpand[p.CWD] = true
		}
	}
	return projs, sessionOverride, autoExpand
}

func (m Model) visibleRows() []row {
	projs, sessionOverride, autoExpand := m.filteredProjects()
	if m.sort == sortName {
		sorted := make([]*sessions.Project, len(projs))
		copy(sorted, projs)
		sort.Slice(sorted, func(i, j int) bool {
			return m.displayLabel(sorted[i].CWD, nil) < m.displayLabel(sorted[j].CWD, nil)
		})
		projs = sorted
	}
	var rows []row
	for _, p := range projs {
		rows = append(rows, row{project: p})
		if !m.expanded[p.CWD] && !autoExpand[p.CWD] {
			continue
		}
		sess := p.Sessions
		if override, ok := sessionOverride[p.CWD]; ok {
			sess = override
		}
		for _, s := range sess {
			rows = append(rows, row{project: p, session: s})
		}
	}
	return rows
}

// ---------- view ----------

var rnd = lipgloss.NewRenderer(os.Stderr)

var (
	orange     = lipgloss.Color("#D4825A")
	dimStyle   = rnd.NewStyle().Foreground(lipgloss.Color("241"))
	hiStyle    = rnd.NewStyle().Foreground(lipgloss.Color("0")).Background(orange)
	titleStyle = rnd.NewStyle().Bold(true).Foreground(orange)
)

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("cnav"))
	b.WriteString("   ")
	b.WriteString(m.stateIndicators())
	b.WriteString("\n\n")

	hiddenCount := m.hiddenProjectCount()

	listH := m.height - 4
	if m.hiddenWorktreeCount > 0 {
		listH--
	}
	if hiddenCount > 0 {
		listH--
	}
	if listH < 5 {
		listH = 5
	}
	b.WriteString(m.renderRows(m.visibleRows(), listH))

	if m.hiddenWorktreeCount > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf(
			"  (%d worktree session%s hidden — directories deleted)",
			m.hiddenWorktreeCount, plural(m.hiddenWorktreeCount))))
		b.WriteString("\n")
	}
	if hiddenCount > 0 {
		state := "H to view"
		if m.showHidden {
			state = "H to collapse"
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf(
			"  (%d project%s hidden — %s)",
			hiddenCount, plural(hiddenCount), state)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.footerKeys()))
	return b.String()
}

func (m Model) hiddenProjectCount() int {
	if m.cfg == nil {
		return 0
	}
	n := 0
	for _, p := range m.projects {
		if m.cfg.Lookup(p.CWD).Hidden {
			n++
		}
	}
	return n
}

func (m Model) stateIndicators() string {
	var parts []string

	sortLabel := "recent"
	if m.sort == sortName {
		sortLabel = "name"
	}
	parts = append(parts, "sort:"+sortLabel)

	preview := "you"
	if m.showAssistant {
		preview = "claude"
	}
	parts = append(parts, "preview:"+preview)

	if m.filter != "" || m.filtering {
		caret := ""
		if m.filtering {
			caret = "█"
		}
		parts = append(parts, "/"+m.filter+caret)
	}

	return dimStyle.Render(strings.Join(parts, "   "))
}

func (m Model) renderRows(rows []row, h int) string {
	if len(rows) == 0 {
		return dimStyle.Render("  no projects")
	}
	start, end := windowAround(m.cursor, len(rows), h)
	labelWidth := max(10, min(40, (m.width-20)/4))

	// Collisions are computed over the visible project set so the suffix
	// reflects what the user can actually see.
	var visibleProjects []*sessions.Project
	for _, r := range rows {
		if r.isProject() {
			visibleProjects = append(visibleProjects, r.project)
		}
	}
	collisions := m.collisionCounts(visibleProjects)

	var b strings.Builder
	for i := start; i < end; i++ {
		r := rows[i]
		var line string
		if r.isProject() {
			line = m.renderProjectRow(r.project, labelWidth, collisions)
		} else {
			line = m.renderChatRow(r.session, labelWidth)
		}
		hidden := r.isProject() && m.cfg.Lookup(r.project.CWD).Hidden
		switch {
		case i == m.cursor:
			b.WriteString(hiStyle.Render("▶ " + line))
		case hidden:
			b.WriteString(dimStyle.Render("  " + line))
		default:
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderProjectRow(p *sessions.Project, labelWidth int, collisions map[string]int) string {
	ago := humanAgo(p.LastActivity)
	chevron := "▸"
	if m.expanded[p.CWD] {
		chevron = "▾"
	}

	// Inline-edit mode for rename: replace the rest of the row with the editor
	// so the cursor stands out and long names aren't truncated.
	if m.renaming && m.renameTarget == p.CWD {
		return fmt.Sprintf("%-10s  %s %s█", ago, chevron, m.renameBuf)
	}

	projLabel := m.displayLabel(p.CWD, collisions)
	if isWorktree(p.CWD) {
		projLabel = "⎇ " + projLabel
	}
	label := chevron + " " + truncRunes(projLabel, labelWidth-2)
	previewWidth := max(1, m.width-20-labelWidth)

	if m.expanded[p.CWD] {
		n := len(p.Sessions)
		tail := dimStyle.Render(fmt.Sprintf("%d chat%s", n, plural(n)))
		return fmt.Sprintf("%-10s  %-*s  %s", ago, labelWidth, label, tail)
	}

	var indicator, previewText string
	if len(p.Sessions) > 0 {
		indicator, previewText = m.sessionPreview(p.Sessions[0])
	} else {
		indicator = dimStyle.Render("you ")
		previewText = dimStyle.Render("(no sessions)")
	}
	return fmt.Sprintf("%-10s  %-*s  %s%s", ago, labelWidth, label, indicator, truncRunes(previewText, previewWidth))
}

func (m Model) renderChatRow(s *sessions.Session, labelWidth int) string {
	ago := humanAgo(s.Started)
	indicator, preview := m.sessionPreview(s)
	label := dimStyle.Render("  └")
	previewWidth := max(1, m.width-20-labelWidth)
	return fmt.Sprintf("%-10s  %-*s  %s%s", ago, labelWidth, label, indicator, truncRunes(preview, previewWidth))
}

// sessionPreview returns the dim "you/ai " indicator and the preview text for a
// session, honoring the showAssistant toggle and substituting a placeholder
// when the chosen preview is empty.
func (m Model) sessionPreview(s *sessions.Session) (indicator, preview string) {
	if m.showAssistant {
		indicator = dimStyle.Render("ai  ")
		preview = s.AssistantPreview
		if preview == "" {
			preview = dimStyle.Render("(no assistant message)")
		}
		return indicator, preview
	}
	indicator = dimStyle.Render("you ")
	preview = s.Preview
	if preview == "" {
		preview = dimStyle.Render("(no user message)")
	}
	return indicator, preview
}

func (m Model) footerKeys() string {
	if m.renaming {
		return "↵  save   esc  cancel   (empty + ↵ to clear)"
	}
	if m.filtering {
		return "↵  apply   esc  clear"
	}
	r, ok := m.selectedRow()
	if ok && !r.isProject() {
		return "↵  cd+resume   shift+↵  cd   ← collapse   space toggle   g/G top/btm   s sort   p preview   / filter   q quit"
	}
	return "↵  cd+claude   R resume   x hide   r rename   shift+↵  cd   → expand   space toggle   g/G top/btm   s sort   p preview   / filter   H show-hidden   q quit"
}

// ---------- helpers ----------

const wtSep = "/.worktrees/"

func isWorktree(cwd string) bool {
	_, _, found := strings.Cut(cwd, wtSep)
	return found
}

// displayLabel returns the label to render for a project. Custom name wins;
// then worktree formatting; then auto-disambiguation when the basename collides
// with another visible project (collisions == nil disables disambiguation —
// callers like sort pass nil so order is stable regardless of which projects
// are currently visible).
func (m Model) displayLabel(cwd string, collisions map[string]int) string {
	if ov := m.cfg.Lookup(cwd); ov.Name != "" {
		return ov.Name
	}
	if before, after, ok := strings.Cut(cwd, wtSep); ok {
		return filepath.Base(before) + " → " + after
	}
	base := filepath.Base(cwd)
	if collisions != nil && collisions[base] > 1 {
		parent := filepath.Base(filepath.Dir(cwd))
		return base + " (" + parent + "/)"
	}
	return base
}

// collisionCounts tallies basenames across the given visible projects, skipping
// projects with a custom name (those don't need disambiguation) and worktree
// rows (those already disambiguate via the "→ branch-path" form).
func (m Model) collisionCounts(projs []*sessions.Project) map[string]int {
	counts := map[string]int{}
	for _, p := range projs {
		if m.cfg.Lookup(p.CWD).Name != "" {
			continue
		}
		if strings.Contains(p.CWD, wtSep) {
			continue
		}
		counts[filepath.Base(p.CWD)]++
	}
	return counts
}

func humanAgo(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
	default:
		return t.Format("2006-01-02")
	}
}

func truncRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func windowAround(cursor, total, h int) (int, int) {
	if total <= h {
		return 0, total
	}
	start := cursor - h/2
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > total {
		end = total
		start = end - h
	}
	return start, end
}

func containsCI(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), needle)
}
