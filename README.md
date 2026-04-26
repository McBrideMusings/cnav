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

That builds `~/bin/cnav-bin` and appends a one-line `eval "$(cnav-bin init)"` to `~/.zshrc`. Open a new shell and run `cnav`.

`~/bin` must be on `PATH`. Override the install location with `CNAV_BIN_DIR=...`.

## Keys

| Key             | Action                                    |
|-----------------|-------------------------------------------|
| `tab` / `1`/`2` | switch between Chats and Projects views   |
| `j`/`k`, arrows | move cursor                               |
| `enter`         | primary action (cd + resume / cd into)    |
| `c`             | cd only                                   |
| `r`             | cd + start fresh `claude`                 |
| `l`             | (Projects view) drill into that project   |
| `/`             | filter                                    |
| `q` / `esc`     | quit                                      |

## Layout

    cmd/cnav/         entrypoint, flag parsing, plumbing
    internal/sessions/ jsonl scanner — reads ~/.claude/projects
    internal/ui/      Bubble Tea model and views
    internal/shell/   wrapper script and shell-quoting

## Notes

- "Resume" runs `claude --resume <session-id>` after `cd`.
- The session list is built fresh on every launch (no cache). Scanning is parallel; on a few hundred sessions it's instant.
- A session's project path comes from the first `cwd` field in its jsonl (the slug-encoded directory name isn't reversible if a path component contains `-`).
