package sessions

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Session is one Claude Code session (one .jsonl file).
type Session struct {
	ID       string    // sessionId
	CWD      string    // project dir claude ran in
	File     string    // absolute path to jsonl
	ModTime  time.Time // file mtime
	Started  time.Time // first user-message timestamp (fallback: ModTime)
	Preview  string    // first user message text, truncated
}

// Project groups sessions by CWD.
type Project struct {
	CWD          string
	Sessions     []*Session // newest first
	LastActivity time.Time
}

const previewMax = 120

// jsonl line shape we care about. Fields we don't need are ignored.
type line struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	CWD       string          `json:"cwd"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type userMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Scan walks ~/.claude/projects/ and returns every session it can parse.
// Sessions are returned sorted by Started desc.
func Scan(claudeProjectsDir string) ([]*Session, error) {
	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(claudeProjectsDir, e.Name())
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range ents {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			files = append(files, filepath.Join(dir, f.Name()))
		}
	}

	sessions := make([]*Session, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)
	for i, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			s, err := parseSession(path)
			if err == nil {
				sessions[i] = s
			}
		}(i, f)
	}
	wg.Wait()

	// Compact nils.
	out := sessions[:0]
	for _, s := range sessions {
		if s != nil {
			out = append(out, s)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Started.After(out[j].Started)
	})
	return out, nil
}

func parseSession(path string) (*Session, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &Session{
		File:    path,
		ModTime: fi.ModTime(),
		Started: fi.ModTime(),
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var l line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			continue
		}
		if s.ID == "" && l.SessionID != "" {
			s.ID = l.SessionID
		}
		if s.CWD == "" && l.CWD != "" {
			s.CWD = l.CWD
		}
		if l.Type == "user" && len(l.Message) > 0 && s.Preview == "" {
			var um userMsg
			if err := json.Unmarshal(l.Message, &um); err == nil && um.Role == "user" {
				s.Preview = extractText(um.Content)
				if l.Timestamp != "" {
					if t, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil {
						s.Started = t
					}
				}
			}
		}
		if s.ID != "" && s.CWD != "" && s.Preview != "" {
			break
		}
	}

	if s.ID == "" {
		// Fall back to filename stem.
		base := filepath.Base(path)
		s.ID = strings.TrimSuffix(base, ".jsonl")
	}
	if s.CWD == "" {
		// Fall back to slug → path. Best-effort: replace leading "-" then "-" → "/".
		dir := filepath.Base(filepath.Dir(path))
		if strings.HasPrefix(dir, "-") {
			s.CWD = "/" + strings.ReplaceAll(dir[1:], "-", "/")
		}
	}
	return s, nil
}

// extractText pulls human-readable text out of a user message Content field,
// which may be a string or an array of content blocks.
func extractText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return truncate(strings.TrimSpace(s), previewMax)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return truncate(strings.TrimSpace(b.Text), previewMax)
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

// GroupByProject buckets sessions by CWD, newest activity first.
func GroupByProject(sessions []*Session) []*Project {
	m := map[string]*Project{}
	for _, s := range sessions {
		p, ok := m[s.CWD]
		if !ok {
			p = &Project{CWD: s.CWD}
			m[s.CWD] = p
		}
		p.Sessions = append(p.Sessions, s)
		if s.Started.After(p.LastActivity) {
			p.LastActivity = s.Started
		}
	}
	out := make([]*Project, 0, len(m))
	for _, p := range m {
		sort.Slice(p.Sessions, func(i, j int) bool {
			return p.Sessions[i].Started.After(p.Sessions[j].Started)
		})
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastActivity.After(out[j].LastActivity)
	})
	return out
}

// DefaultProjectsDir returns ~/.claude/projects.
func DefaultProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}
