package ui

import (
	"testing"
	"time"

	"github.com/pierce/cnav/internal/config"
	"github.com/pierce/cnav/internal/sessions"
)

// at builds a timestamp n minutes before a fixed reference point, so "newer"
// reads the same way in the test as it does on screen.
func at(minutesAgo int) time.Time {
	base := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	return base.Add(-time.Duration(minutesAgo) * time.Minute)
}

// sample project: two of its own chats and two worktrees, activity interleaved.
//
//	 5m  wt feature-b
//	10m  own chat "own-new"
//	20m  wt feature-a
//	40m  own chat "own-old"
func sampleProject() *sessions.Project {
	root := "/repo"
	ss := []*sessions.Session{
		{ID: "own-new", CWD: root, Root: root, Started: at(10)},
		{ID: "own-old", CWD: root, Root: root, Started: at(40)},
		{ID: "b1", CWD: "/wt/feature-b", Root: root, Worktree: "feature-b", Started: at(5)},
		{ID: "a1", CWD: "/wt/feature-a", Root: root, Worktree: "feature-a", Started: at(20)},
	}
	return sessions.GroupByProject(ss)[0]
}

func newTestModel() Model {
	return Model{cfg: &config.Config{}, expanded: map[string]bool{}}
}

// rowIDs renders each row as a short tag so an expected layout reads as a list.
func rowIDs(rows []row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		switch {
		case r.isChat():
			out = append(out, "chat:"+r.session.ID)
		case r.isWorktree():
			out = append(out, "wt:"+r.worktree.Name)
		default:
			out = append(out, "proj:"+r.project.CWD)
		}
	}
	return out
}

func assertRows(t *testing.T, got []row, want []string) {
	t.Helper()
	ids := rowIDs(got)
	if len(ids) != len(want) {
		t.Fatalf("rows = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("rows = %v, want %v", ids, want)
		}
	}
}

func TestChildRowsInterleavesByRecency(t *testing.T) {
	m := newTestModel()
	assertRows(t, m.childRows(sampleProject()), []string{
		"wt:feature-b",
		"chat:own-new",
		"wt:feature-a",
		"chat:own-old",
	})
}

func TestChildRowsExpandedWorktreeShowsItsChats(t *testing.T) {
	m := newTestModel()
	m.expanded["/wt/feature-a"] = true
	assertRows(t, m.childRows(sampleProject()), []string{
		"wt:feature-b",
		"chat:own-new",
		"wt:feature-a",
		"chat:a1",
		"chat:own-old",
	})
}

func TestChildRowsNameSortGroupsWorktreesFirst(t *testing.T) {
	m := newTestModel()
	m.sort = sortName
	assertRows(t, m.childRows(sampleProject()), []string{
		"wt:feature-a",
		"wt:feature-b",
		"chat:own-new",
		"chat:own-old",
	})
}

func TestChildRowsHidesHiddenWorktreeUntilShowHidden(t *testing.T) {
	m := newTestModel()
	m.cfg.SetHidden("/wt/feature-b", true)
	assertRows(t, m.childRows(sampleProject()), []string{
		"chat:own-new",
		"wt:feature-a",
		"chat:own-old",
	})

	m.showHidden = true
	assertRows(t, m.childRows(sampleProject()), []string{
		"wt:feature-b",
		"chat:own-new",
		"wt:feature-a",
		"chat:own-old",
	})
}
