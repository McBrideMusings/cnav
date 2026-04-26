package sessions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Session is one Claude Code session (one .jsonl file).
type Session struct {
	ID              string    // sessionId
	CWD             string    // project dir claude ran in
	File            string    // absolute path to jsonl
	ModTime         time.Time // file mtime
	Started          time.Time // first user-message timestamp (fallback: ModTime)
	Preview          string    // last non-skippable user message, truncated
	AssistantPreview string    // last assistant text block, truncated
}

// Project groups sessions by CWD.
type Project struct {
	CWD          string
	Sessions     []*Session // newest first
	LastActivity time.Time
}

const (
	previewMax         = 2000
	largeFileThreshold = 5 << 20   // 5MB
	tailScanBytes      = 256 << 10 // 256KB tail for preview on large files
	sysTagRe           = `local-command-caveat|command-message|command-name|command-args|system-reminder`
)

var (
	xmlBlockRe = regexp.MustCompile(`(?s)<(` + sysTagRe + `)[^>]*>.*?</\1>`)
	xmlTagRe   = regexp.MustCompile(`</?(?:` + sysTagRe + `)[^>]*>`)
)

type scanState struct {
	seenFirstUser        bool
	lastPreview          string
	lastAssistantPreview string
}

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
func Scan(claudeProjectsDir string) ([]*Session, int, error) {
	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return nil, 0, err
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

	hiddenCount := 0
	out := slices.DeleteFunc(sessions, func(s *Session) bool {
		if s == nil {
			return true
		}
		if !strings.Contains(s.CWD, "/.worktrees/") {
			return false
		}
		if _, err := os.Stat(s.CWD); err == nil {
			return false
		}
		hiddenCount++
		return true
	})

	sort.Slice(out, func(i, j int) bool {
		return out[i].Started.After(out[j].Started)
	})
	return out, hiddenCount, nil
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

	state := &scanState{}

	isLarge := fi.Size() > largeFileThreshold
	if isLarge {
		scanSessionLines(f, s, 200, true, false, state, path)

		tailOffset := fi.Size() - tailScanBytes
		if _, err := f.Seek(tailOffset, io.SeekStart); err == nil {
			br := bufio.NewReaderSize(f, 1024*1024)
			if _, err := br.ReadString('\n'); err == nil || err == io.EOF {
				scanSessionLines(br, s, 0, false, true, state, path)
			} else {
				if _, seekErr := f.Seek(0, io.SeekStart); seekErr == nil {
					scanSessionLines(f, s, 0, false, true, state, path)
				}
			}
		}
	} else {
		scanSessionLines(f, s, 0, true, true, state, path)
	}

	s.Preview = state.lastPreview
	s.AssistantPreview = state.lastAssistantPreview

	if s.ID == "" {
		base := filepath.Base(path)
		s.ID = strings.TrimSuffix(base, ".jsonl")
	}
	if s.CWD == "" {
		dir := filepath.Base(filepath.Dir(path))
		if strings.HasPrefix(dir, "-") {
			s.CWD = "/" + strings.ReplaceAll(dir[1:], "-", "/")
		}
	}
	return s, nil
}

func scanSessionLines(r io.Reader, s *Session, lineLimit int, wantMeta, wantPreview bool, state *scanState, path string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	lineCount := 0
	for sc.Scan() {
		lineCount++
		var l line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			continue
		}

		if wantMeta {
			if s.ID == "" && l.SessionID != "" {
				s.ID = l.SessionID
			}
			if s.CWD == "" && l.CWD != "" {
				s.CWD = l.CWD
			}
		}

		if (wantMeta || wantPreview) && len(l.Message) > 0 {
			var um userMsg
			if err := json.Unmarshal(l.Message, &um); err != nil {
				continue
			}

			switch um.Role {
			case "user":
				if !state.seenFirstUser {
					state.seenFirstUser = true
					if l.Timestamp != "" {
						if t, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil {
							s.Started = t
						}
					}
				}
				if wantPreview {
					if text := extractText(um.Content); text != "" && !isSkippableCommand(text) {
						state.lastPreview = text
					}
				}
			case "assistant":
				if wantPreview {
					if text := extractAssistantText(um.Content); text != "" {
						state.lastAssistantPreview = text
					}
				}
			}
		}

		if wantMeta && !wantPreview && state.seenFirstUser && s.ID != "" && s.CWD != "" {
			break
		}
		if lineLimit > 0 && lineCount >= lineLimit {
			break
		}
	}

	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "cnav: scanner error in %s: %v\n", path, err)
	}
}

// extractAssistantText pulls the first text block from an assistant content array,
// skipping thinking blocks.
func extractAssistantText(raw json.RawMessage) string {
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			return truncate(strings.TrimSpace(b.Text), previewMax)
		}
	}
	return ""
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

func isSkippableCommand(s string) bool {
	if s == "" {
		return false
	}
	switch strings.Fields(s)[0] {
	case "/clear", "/compact", "/reset":
		return true
	}
	return false
}

func stripTags(s string) string {
	s = xmlBlockRe.ReplaceAllString(s, " ")
	s = xmlTagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	s = stripTags(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
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
func DefaultProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}
