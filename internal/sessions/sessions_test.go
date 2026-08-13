package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

// writeGitFile lays down the ".git" file a git worktree carries, pointing at the
// main checkout's administrative directory.
func writeGitFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepoRootWorktree(t *testing.T) {
	tmp := t.TempDir()
	wt := filepath.Join(tmp, "elsewhere", "issue-114")
	writeGitFile(t, wt, "gitdir: /Users/pierce/Projects/apple-notepad/.git/worktrees/issue-114\n")

	root, name := repoRoot(wt)
	if root != "/Users/pierce/Projects/apple-notepad" {
		t.Errorf("root = %q, want /Users/pierce/Projects/apple-notepad", root)
	}
	if name != "issue-114" {
		t.Errorf("worktree = %q, want issue-114", name)
	}
}

func TestRepoRootNonWorktree(t *testing.T) {
	tmp := t.TempDir()

	plain := filepath.Join(tmp, "plain")
	if err := os.MkdirAll(filepath.Join(plain, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "sub")
	writeGitFile(t, sub, "gitdir: ../.git/modules/sub\n")

	for _, dir := range []string{plain, sub, filepath.Join(tmp, "missing")} {
		root, name := repoRoot(dir)
		if root != dir || name != "" {
			t.Errorf("repoRoot(%q) = (%q, %q), want (%q, \"\")", dir, root, name, dir)
		}
	}
}

func TestGroupByProjectFoldsWorktrees(t *testing.T) {
	root := "/Users/pierce/Projects/apple-notepad"
	ss := []*Session{
		{ID: "a", CWD: root, Root: root},
		{ID: "b", CWD: "/Users/pierce/.worktrees/apple-notepad/issue-114", Root: root, Worktree: "issue-114"},
		{ID: "c", CWD: "/tmp/other", Root: "/tmp/other"},
	}

	projs := GroupByProject(ss)
	if len(projs) != 2 {
		t.Fatalf("got %d projects, want 2", len(projs))
	}
	byCWD := map[string]*Project{}
	for _, p := range projs {
		byCWD[p.CWD] = p
	}
	p, ok := byCWD[root]
	if !ok {
		t.Fatalf("no project for %s (got %v)", root, byCWD)
	}
	if len(p.Sessions) != 2 {
		t.Errorf("%s has %d sessions, want 2 (main + worktree)", root, len(p.Sessions))
	}
}
