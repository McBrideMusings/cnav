# Getting Started

## Install

```sh
./install.sh
```

This builds `$GOPATH/bin/cnav-bin` and appends `eval "$(cnav-bin init)"` to `~/.zshrc`. Open a new shell and run `cnav`.

`$GOPATH/bin` must be on `PATH`. Override the install location:

```sh
CNAV_BIN_DIR=/usr/local/bin ./install.sh
```

The list is always in **type-to-filter** mode: just start typing to narrow the projects by name. Navigation and commands use arrows and modifier chords so every letter stays free for the filter.

> **Note:** the `alt+` commands require your terminal to send Option as Meta (enable "Use Option as Meta key" in Terminal.app / iTerm, or it's on by default in Ghostty).

| Key | Action |
|---|---|
| *type* | Filter projects by name |
| `↑`/`↓`, `ctrl+p`/`ctrl+n` | Move cursor |
| `→` | Expand project, or descend into chats |
| `←` | Collapse project (jumps to parent if on a chat row) |
| `home` / `end` | Jump to top / bottom |
| `enter` (project row) | `cd` + launch fresh `claude` |
| `enter` (chat row) | `cd` + resume that chat |
| `ctrl+r` (project row) | `cd` + resume most recent session |
| `shift+enter` | `cd` only |
| `alt+s` | Toggle sort: recent / name |
| `alt+p` | Toggle preview: your last message / Claude's reply |
| `alt+x` (project row) | Toggle hidden — persists to config |
| `alt+r` (project row) | Rename project label inline |
| `alt+h` | Reveal hidden projects |
| `esc` | Clear filter (quits when the filter is already empty) |
| `ctrl+c` | Quit |

## Per-project overrides

`~/.config/cnav/config.toml` stores hide/rename state. The TUI writes it; you can also hand-edit:

```toml
[[project]]
cwd = "/Users/you/.claude"
hidden = true

[[project]]
cwd = "/Users/you/Projects/my-app"
name = "my-app (work)"
```

Identical basenames are auto-disambiguated in the list (`.claude (pierce/)`, `.claude (Projects/)`). A manual `name` always wins.
