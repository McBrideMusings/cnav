# cnav

A small TUI for jumping between Claude Code projects and resuming past chats.

## What it does

`cnav` reads `~/.claude/projects/` and shows every directory Claude has run in, sorted by most-recent activity. Pick one to `cd` there and launch a fresh session, or expand it inline to see and resume any past chat in that project.

## Why a shell function

A child process can't change its parent shell's working directory. `cnav` is a shell function (installed in your `.zshrc`) that runs the binary, captures the chosen `cd …` command on stdout, and `eval`s it. The TUI itself draws on stderr so stdout stays clean.

## Install

    ./install.sh

That builds `$GOPATH/bin/cnav-bin` and appends a one-line `eval "$(cnav-bin init)"` to `~/.zshrc`. Open a new shell and run `cnav`.

`$GOPATH/bin` must be on `PATH`. Override the install location with `CNAV_BIN_DIR=...`.

## Keys

| Key                     | Action                                              |
|-------------------------|-----------------------------------------------------|
| `j`/`k`, `↑`/`↓`       | move cursor                                         |
| `space`                 | toggle expansion of the current project             |
| `→` / `l`               | expand project, or descend into its chats           |
| `←` / `h`               | collapse project (jumps to parent if on a chat row) |
| `enter` (project row)   | cd + launch fresh `claude`                          |
| `enter` (chat row)      | cd + resume that chat                               |
| `R` (project row)       | cd + resume most recent session for that project    |
| `shift+enter`           | cd only                                             |
| `g` / `G`               | jump to top / bottom of list                        |
| `s`                     | toggle sort: recent / name                          |
| `p`                     | toggle preview: your last message / Claude's reply  |
| `/`                     | filter (hotkey bar updates to show only active keys)|
| `x` (project row)       | toggle hidden — persists to config                  |
| `r` (project row)       | rename project label inline                         |
| `H`                     | reveal hidden projects (dimmed, still interactable) |
| `q` / `esc`             | quit                                                |

## Per-project overrides

`~/.config/cnav/config.toml` stores hide/rename state. The TUI writes it; you can also hand-edit:

```toml
[[project]]
cwd = "/Users/pierce/.claude"
hidden = true

[[project]]
cwd = "/Users/pierce/Projects/.claude"
name = "dotclaude-projects"
```

Identical basenames are auto-disambiguated in the list (`.claude (pierce/)`, `.claude (Projects/)`); a manual `name` wins when set.

## Layout

    cmd/cnav/         entrypoint, flag parsing, plumbing
    internal/sessions/ jsonl scanner — reads ~/.claude/projects
    internal/ui/      Bubble Tea model and views
    internal/shell/   wrapper script and shell-quoting

## Notes

- "Resume" runs `claude --resume <session-id>` after `cd`.
- The session list is built fresh on every launch (no cache). Scanning is parallel; on a few hundred sessions it's instant. Session files larger than 1 MB use a two-phase scan (first 50 lines for metadata, last 256 KB for preview) to stay fast on long sessions.
- A session's project path comes from the first `cwd` field in its jsonl (the slug-encoded directory name isn't reversible if a path component contains `-`).
- The preview column shows your last message by default; press `p` to switch to Claude's longest reply. Each row shows a dim `you` / `ai` prefix so the active mode is visible without looking at the header. `/clear`, `/compact`, and `/reset` are skipped when determining the last user message. System-injected XML blocks are stripped from preview text.
- Filtering searches across **all** sessions in every project, not just the newest. If a match comes from an older chat, that project auto-expands and only the matching chats are shown.
- Worktree sessions are hidden if the worktree directory no longer exists on disk.
- Active state (sort order, preview mode, filter text) is shown inline in the header next to the `cnav` title.
