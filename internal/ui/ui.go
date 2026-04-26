package ui

import (
	"fmt"
	"path/filepath"
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
	viewProjectChats // chats filtered to one project
)

type Model struct {
	sessions []*sessions.Session
	projects []*sessions.Project

	view         viewID
	cursor       int
	filter       string
	filtering    bool
	width        int
	height       int
	focusProject *sessions.Project // when in viewProjectChats

	// Result, set when user picks an action.
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
		if m.view == viewProjectChats {
			m.view = viewProjects
			m.cursor = 0
			return m, nil
		}
		m.Done = true
		return m, tea.Quit
	case "tab", "1":
		m.view = viewChats
		m.cursor = 0
	case "2":
		m.view = viewProjects
		m.cursor = 0
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
	case "/":
		m.filtering = true
		m.filter = ""
	case "enter":
		return m.activate(actionDefault)
	case "c":
		return m.activate(actionCDOnly)
	case "r":
		return m.activate(actionNewClaude)
	case "l":
		return m.activate(actionDrillIn)
	}
	return m, nil
}

type actionKind int

const (
	actionDefault actionKind = iota
	actionCDOnly
	actionNewClaude
	actionDrillIn
)

func (m Model) activate(kind actionKind) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewChats:
		list := m.filteredSessions()
		if len(list) == 0 {
			return m, nil
		}
		s := list[m.cursor]
		switch kind {
		case actionDefault:
			m.Action = shell.Action{Dir: s.CWD, Resume: s.ID}
		case actionCDOnly:
			m.Action = shell.Action{Dir: s.CWD}
		case actionNewClaude:
			m.Action = shell.Action{Dir: s.CWD, NewClaude: true}
		}
		m.Done = true
		return m, tea.Quit

	case viewProjects:
		list := m.filteredProjects()
		if len(list) == 0 {
			return m, nil
		}
		p := list[m.cursor]
		switch kind {
		case actionDefault, actionCDOnly:
			m.Action = shell.Action{Dir: p.CWD}
			m.Done = true
			return m, tea.Quit
		case actionNewClaude:
			m.Action = shell.Action{Dir: p.CWD, NewClaude: true}
			m.Done = true
			return m, tea.Quit
		case actionDrillIn:
			m.focusProject = p
			m.view = viewProjectChats
			m.cursor = 0
		}

	case viewProjectChats:
		if m.focusProject == nil {
			m.view = viewProjects
			m.cursor = 0
			return m, nil
		}
		list := m.focusProject.Sessions
		if len(list) == 0 {
			return m, nil
		}
		s := list[m.cursor]
		switch kind {
		case actionDefault:
			m.Action = shell.Action{Dir: s.CWD, Resume: s.ID}
		case actionCDOnly:
			m.Action = shell.Action{Dir: s.CWD}
		case actionNewClaude:
			m.Action = shell.Action{Dir: s.CWD, NewClaude: true}
		}
		m.Done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) maxCursor() int {
	var n int
	switch m.view {
	case viewChats:
		n = len(m.filteredSessions())
	case viewProjects:
		n = len(m.filteredProjects())
	case viewProjectChats:
		if m.focusProject != nil {
			n = len(m.focusProject.Sessions)
		}
	}
	if n == 0 {
		return 0
	}
	return n - 1
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

// ---------- view ----------

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hiStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("12"))
	tabActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("12")).Padding(0, 1)
	tabIdle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1)
)

func (m Model) View() string {
	var b strings.Builder

	// Tabs.
	t1 := "1 Chats"
	t2 := "2 Projects"
	style1 := tabActive
	style2 := tabIdle
	if m.view == viewProjects {
		style1, style2 = style2, style1
	}
	b.WriteString(style1.Render(t1))
	b.WriteString(style2.Render(t2))
	if m.view == viewProjectChats && m.focusProject != nil {
		b.WriteString(tabActive.Render("→ " + filepath.Base(m.focusProject.CWD)))
	}
	b.WriteString("\n\n")

	// Filter bar.
	if m.filtering || m.filter != "" {
		caret := ""
		if m.filtering {
			caret = "█"
		}
		b.WriteString(dimStyle.Render("/ ") + m.filter + caret + "\n\n")
	}

	// Body.
	listH := m.height - 6
	if listH < 5 {
		listH = 5
	}
	switch m.view {
	case viewChats:
		b.WriteString(m.renderSessionList(m.filteredSessions(), listH, true))
	case viewProjects:
		b.WriteString(m.renderProjectList(m.filteredProjects(), listH))
	case viewProjectChats:
		if m.focusProject != nil {
			b.WriteString(m.renderSessionList(m.focusProject.Sessions, listH, false))
		}
	}

	// Footer.
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.footerKeys()))
	return b.String()
}

func (m Model) renderSessionList(list []*sessions.Session, h int, showProject bool) string {
	if len(list) == 0 {
		return dimStyle.Render("  no sessions")
	}
	start, end := windowAround(m.cursor, len(list), h)
	var b strings.Builder
	for i := start; i < end; i++ {
		s := list[i]
		ago := humanAgo(s.Started)
		proj := filepath.Base(s.CWD)
		preview := s.Preview
		if preview == "" {
			preview = dimStyle.Render("(no user message)")
		}
		var line string
		if showProject {
			line = fmt.Sprintf("%-10s  %-22s  %s", ago, truncRunes(proj, 22), truncRunes(preview, m.width-40))
		} else {
			line = fmt.Sprintf("%-10s  %s", ago, truncRunes(preview, m.width-14))
		}
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
		line := fmt.Sprintf("%-10s  %-12s  %s", ago, count, truncRunes(p.CWD, m.width-30))
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
	case viewChats, viewProjectChats:
		return "↵ cd+resume   c cd   r cd+claude   tab/1/2 switch view   / filter   q quit"
	case viewProjects:
		return "↵ cd   l list chats   r cd+claude   tab/1/2 switch view   / filter   q quit"
	}
	return ""
}

// ---------- helpers ----------

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
