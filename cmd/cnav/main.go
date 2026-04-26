package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pierce/cnav/internal/sessions"
	"github.com/pierce/cnav/internal/shell"
	"github.com/pierce/cnav/internal/ui"
)

func main() {
	for _, a := range os.Args[1:] {
		switch a {
		case "--print-shell", "init":
			shell.PrintWrapper()
			return
		case "-h", "--help", "help":
			printHelp()
			return
		}
	}

	dir, err := sessions.DefaultProjectsDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnav: locate home dir:", err)
		os.Exit(1)
	}
	ss, err := sessions.Scan(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnav: scan:", err)
		os.Exit(1)
	}
	if len(ss) == 0 {
		fmt.Fprintln(os.Stderr, "cnav: no sessions found in", dir)
		os.Exit(1)
	}

	// TUI on stderr so stdout stays clean for the wrapper to eval.
	p := tea.NewProgram(ui.New(ss), tea.WithOutput(os.Stderr), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnav:", err)
		os.Exit(1)
	}
	model, ok := m.(ui.Model)
	if !ok || !model.Done {
		return
	}
	cmd := model.Action.Render()
	if cmd != "" {
		fmt.Println(cmd)
	}
}

func printHelp() {
	fmt.Print(`cnav — jump between Claude Code projects and resume past sessions.

Usage:
  cnav                    launch TUI (use via the 'cnav' shell function)
  cnav init               print shell wrapper function (eval into your shell)
  cnav --print-shell      same as 'init'
  cnav -h, --help         show this help

Install:
  1. go install ./cmd/cnav   (or build and place 'cnav' on $PATH as cnav-bin)
  2. add to ~/.zshrc:        eval "$(cnav-bin init)"
  3. then run 'cnav' from anywhere.

Keys:
  tab / 1 / 2   switch between Chats and Projects views
  j / k         move cursor (or arrows)
  enter         primary action (cd + resume / cd into project)
  c             cd only
  r             cd + start fresh 'claude'
  l             (Projects view) drill into that project's chats
  /             filter
  q / esc       quit
`)
}
