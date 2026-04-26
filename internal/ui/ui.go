package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pierce/cnav/internal/sessions"
	"github.com/pierce/cnav/internal/shell"
)

type viewID int

const (
	viewChats viewID = iota
	viewProjects
)

type sortOrder int

const (
	sortRecent sortOrder = iota
	sortName
)

type Model struct {
	sessions []*sessions.Session
	projects []*sessions.Project

	view      viewID
	cursor    int
	filter    string
	filtering bool
	sort      sortOrder
	width     int
	height    int

	Action shell.Action
	Done   bool
}

func New(ss []*sessions.Session) Model {
	return Model{
		sessions: ss,
		projects: sessions.GroupByProject(ss),
		view:     viewChats,
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
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
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
	case "tab", "left", "right":
		if m.view == viewChats {
			m.view = viewProjects
		} else {
			m.view = viewChats
		}
		m.cursor = 0
		m.sort = sortRecent
	case "1":
		m.view = viewChats
		m.cursor = 0
		m.sort = sortRecent
	case "2":
		m.view = viewProjects
		m.cursor = 0
		m.sort = sortRecent
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
	case "/":
		m.filtering = true
		m.filter = ""
	case "enter":
		return m.activate()
	case "c":
		return m.activateCD()
	case "r":
		return m.activateNewClaude()
	}
	return m, nil
}

func (m Model) selectedDir() (string, string) {
	switch m.view {
	case viewChats:
		list := m.sortedSessions()
		if len(list) == 0 {
			return "", ""
		}
		s := list[m.cursor]
		return s.CWD, s.ID
	case viewProjects:
		list := m.sortedProjects()
		if len(list) == 0 {
			return "", ""
		}
		return list[m.cursor].CWD, ""
	}
	return "", ""
}

func (m Model) activate() (tea.Model, tea.Cmd) {
	dir, id := m.selectedDir()
	if dir == "" {
		return m, nil
	}
	if id != "" {
		m.Action = shell.Action{Dir: dir, Resume: id}
	} else {
		m.Action = shell.Action{Dir: dir, NewClaude: true}
	}
	m.Done = true
	return m, tea.Quit
}

func (m Model) activateCD() (tea.Model, tea.Cmd) {
	dir, _ := m.selectedDir()
	if dir == "" {
		return m, nil
	}
	m.Action = shell.Action{Dir: dir}
	m.Done = true
	return m, tea.Quit
}

func (m Model) activateNewClaude() (tea.Model, tea.Cmd) {
	dir, _ := m.selectedDir()
	if dir == "" {
		return m, nil
	}
	m.Action = shell.Action{Dir: dir, NewClaude: true}
	m.Done = true
	return m, tea.Quit
}

func (m Model) maxCursor() int {
	var n int
	switch m.view {
	case viewChats:
		n = len(m.sortedSessions())
	case viewProjects:
		n = len(m.sortedProjects())
	}
	if n == 0 {
		return 0
	}
	return n - 1
}

func (m Model) sortedSessions() []*sessions.Session {
	list := m.filteredSessions()
	if m.sort == sortName {
		out := make([]*sessions.Session, len(list))
		copy(out, list)
		sort.Slice(out, func(i, j int) bool {
			return filepath.Base(out[i].CWD) < filepath.Base(out[j].CWD)
		})
		return out
	}
	return list
}

func (m Model) sortedProjects() []*sessions.Project {
	list := m.filteredProjects()
	if m.sort == sortName {
		out := make([]*sessions.Project, len(list))
		copy(out, list)
		sort.Slice(out, func(i, j int) bool {
			return projectLabel(out[i].CWD) < projectLabel(out[j].CWD)
		})
		return out
	}
	return list
}

func (m Model) filteredProjects() []*sessions.Project {
	if m.filter == "" {
		return m.projects
	}
	q := strings.ToLower(m.filter)
	var out []*sessions.Project
	for _, p := range m.projects {
		if strings.Contains(strings.ToLower(p.CWD), q) {
			out = append(out, p)
		}
	}
	return out
}

func (m Model) filteredSessions() []*sessions.Session {
	if m.filter == "" {
		return m.sessions
	}
	q := strings.ToLower(m.filter)
	var out []*sessions.Session
	for _, s := range m.sessions {
		if strings.Contains(strings.ToLower(s.CWD), q) ||
			strings.Contains(strings.ToLower(s.Preview), q) {
			out = append(out, s)
		}
	}
	return out
}

// ---------- view ----------

var (
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hiStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("12"))
	tabActive = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("12")).Padding(0, 1)
	tabIdle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1)
)

func (m Model) View() string {
	var b strings.Builder

	t1s, t2s := tabIdle, tabIdle
	if m.view == viewChats {
		t1s = tabActive
	} else {
		t2s = tabActive
	}
	b.WriteString(t1s.Render("1 Chats"))
	b.WriteString(t2s.Render("2 Projects"))
	b.WriteString("\n\n")

	if m.filtering || m.filter != "" {
		caret := ""
		if m.filtering {
			caret = "█"
		}
		b.WriteString(dimStyle.Render("/ ") + m.filter + caret + "\n\n")
	}

	listH := m.height - 6
	if listH < 5 {
		listH = 5
	}
	switch m.view {
	case viewChats:
		b.WriteString(m.renderSessionList(m.sortedSessions(), listH))
	case viewProjects:
		b.WriteString(m.renderProjectList(m.sortedProjects(), listH))
	}

	sortLabel := "recent"
	if m.sort == sortName {
		sortLabel = "name"
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.footerKeys() + "   sort:" + sortLabel))
	return b.String()
}

func (m Model) renderSessionList(list []*sessions.Session, h int) string {
	if len(list) == 0 {
		return dimStyle.Render("  no sessions")
	}
	start, end := windowAround(m.cursor, len(list), h)
	var b strings.Builder
	for i := start; i < end; i++ {
		s := list[i]
		ago := humanAgo(s.Started)
		proj := projectLabel(s.CWD)
		preview := s.Preview
		if preview == "" {
			preview = dimStyle.Render("(no user message)")
		}
		line := fmt.Sprintf("%-10s  %-22s  %s", ago, truncRunes(proj, 22), truncRunes(preview, m.width-40))
		if i == m.cursor {
			b.WriteString(hiStyle.Render("▶ " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderProjectList(list []*sessions.Project, h int) string {
	if len(list) == 0 {
		return dimStyle.Render("  no projects")
	}
	start, end := windowAround(m.cursor, len(list), h)
	var b strings.Builder
	for i := start; i < end; i++ {
		p := list[i]
		ago := humanAgo(p.LastActivity)
		count := fmt.Sprintf("%d session%s", len(p.Sessions), plural(len(p.Sessions)))
		label := projectLabel(p.CWD)
		line := fmt.Sprintf("%-10s  %-12s  %s", ago, count, truncRunes(label, m.width-30))
		if i == m.cursor {
			b.WriteString(hiStyle.Render("▶ " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) footerKeys() string {
	switch m.view {
	case viewChats:
		return "↵ cd+resume   c cd   r cd+claude   tab switch   s sort   / filter   q quit"
	case viewProjects:
		return "↵ cd+claude   c cd   r cd+claude   tab switch   s sort   / filter   q quit"
	}
	return ""
}

// ---------- helpers ----------

func projectLabel(cwd string) string {
	const wtSep = "/.worktrees/"
	if idx := strings.Index(cwd, wtSep); idx >= 0 {
		return filepath.Base(cwd[:idx]) + " → " + cwd[idx+len(wtSep):]
	}
	return filepath.Base(cwd)
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
