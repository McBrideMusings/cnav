package shell

import (
	"fmt"
	"strings"
)

// Action is what the wrapper shell function should do after the TUI exits.
type Action struct {
	Dir       string // required: directory to cd into
	Resume    string // optional: session id to resume
	NewClaude bool   // optional: launch fresh `claude` after cd
}

// Render produces the shell command string that the wrapper will eval.
// Empty string means "do nothing" (user quit).
func (a Action) Render() string {
	if a.Dir == "" {
		return ""
	}
	parts := []string{"cd " + quote(a.Dir)}
	switch {
	case a.Resume != "":
		parts = append(parts, "claude --resume "+quote(a.Resume))
	case a.NewClaude:
		parts = append(parts, "claude")
	}
	return strings.Join(parts, " && ")
}

// quote single-quotes a string for safe shell eval.
func quote(s string) string {
	escaped := strings.ReplaceAll(s, "'", `'\''`)
	return "'" + escaped + "'"
}

// WrapperScript is the zsh/bash function the user installs.
const WrapperScript = `cnav() {
  local __cnav_cmd
  __cnav_cmd=$(command cnav-bin "$@") || return $?
  [ -n "$__cnav_cmd" ] && eval "$__cnav_cmd"
}`

// PrintWrapper writes the wrapper to stdout (used by --print-shell).
func PrintWrapper() {
	fmt.Println(WrapperScript)
}
