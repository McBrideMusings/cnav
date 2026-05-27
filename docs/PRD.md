# Product Spec (PRD)

## Problem

Claude Code stores sessions under `~/.claude/projects/`, one subdirectory per working directory. There is no built-in way to browse past sessions, jump to a project, or resume a specific chat from the terminal.

## Solution

`cnav` is a small TUI that reads `~/.claude/projects/` and presents every project sorted by most-recent activity. The user picks a project or individual chat; the TUI emits a shell command that is `eval`d by a wrapper function to `cd` and optionally launch or resume Claude.

## Scope

**In scope:**
- Read-only access to session history (no writes to `~/.claude/`)
- `cd` + launch fresh Claude session
- `cd` + resume a specific session by ID
- Hide and rename projects (stored in `~/.config/cnav/config.toml`)
- Filter/search across all sessions
- Preview of last user message or longest assistant reply

**Out of scope:**
- Editing or deleting sessions
- Cross-machine sync
- Non-zsh shells (bash wrapper not provided)
- Integration with the Claude API

## Constraints

- Binary must not change its parent shell's directory — shell wrapper required
- TUI must render on stderr; stdout reserved for the eval'd command
- Session list built fresh on every launch (no persistent cache)
- Large files (>5 MB) require a two-phase scan to stay fast
