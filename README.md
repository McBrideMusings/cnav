# cnav

A small TUI for jumping between Claude Code projects and resuming past chats.

## What it does

`cnav` reads `~/.claude/projects/` and shows two views:

- **Chats** — every Claude Code session you've ever had, newest first. Pick one and jump straight back in.
- **Projects** — every directory Claude has run in, sorted by most-recent activity. Pick one to `cd` there.

## Why a shell function

A child process can't change its parent shell's working directory. `cnav` is a shell function (installed in your `.zshrc`) that runs the binary, captures the chosen `cd …` command on stdout, and `eval`s it. The TUI itself draws on stderr so stdout stays clean.

## Install

    ./install.sh

That builds `$GOPATH/bin/cnav-bin` and appends a one-line `eval "$(cnav-bin init)"` to `~/.zshrc`. Open a new shell and run `cnav`.

`$GOPATH/bin` must be on `PATH`. Override the install location with `CNAV_BIN_DIR=...`.

## Keys

| Key                     | Action                                              |
|-------------------------|-----------------------------------------------------|
| `tab` / `←` / `→`      | toggle between Chats and Projects                   |
| `1` / `2`               | jump to Chats / Projects view                       |
| `j`/`k`, `↑`/`↓`       | move cursor                                         |
| `enter` (Chats)         | cd + resume that chat                               |
| `enter` (Projects)      | cd + launch fresh `claude`                          |
| `shift+enter`           | cd only                                             |
| `c`                     | cd only                                             |
| `s`                     | toggle sort: recent / name                          |
| `p`                     | toggle preview: your last message / Claude's reply  |
| `/`                     | filter (hotkey bar updates to show only active keys)|
| `q` / `esc`             | quit                                                |

## Layout

    cmd/cnav/         entrypoint, flag parsing, plumbing
    internal/sessions/ jsonl scanner — reads ~/.claude/projects
    internal/ui/      Bubble Tea model and views
    internal/shell/   wrapper script and shell-quoting

## Notes

- "Resume" runs `claude --resume <session-id>` after `cd`.
- The session list is built fresh on every launch (no cache). Scanning is parallel; on a few hundred sessions it's instant. Session files larger than 1 MB use a two-phase scan (first 50 lines for metadata, last 256 KB for preview) to stay fast on long sessions.
- A session's project path comes from the first `cwd` field in its jsonl (the slug-encoded directory name isn't reversible if a path component contains `-`).
- The preview column shows your last message by default; press `p` to switch to Claude's last reply. `/clear`, `/compact`, and `/reset` are skipped when determining the last user message. System-injected XML blocks are stripped from preview text.
- Worktree sessions are hidden if the worktree directory no longer exists on disk.
- Active state (sort order, preview mode, filter text) is shown inline in the header next to the tab indicator.
