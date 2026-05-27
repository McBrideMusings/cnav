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

## Keys

| Key | Action |
|---|---|
| `j`/`k`, `↑`/`↓` | Move cursor |
| `space` | Toggle expansion of current project |
| `→` / `l` | Expand project, or descend into chats |
| `←` / `h` | Collapse project (jumps to parent if on a chat row) |
| `enter` (project row) | `cd` + launch fresh `claude` |
| `enter` (chat row) | `cd` + resume that chat |
| `R` (project row) | `cd` + resume most recent session |
| `shift+enter` | `cd` only |
| `g` / `G` | Jump to top / bottom |
| `s` | Toggle sort: recent / name |
| `p` | Toggle preview: your last message / Claude's reply |
| `/` | Filter |
| `x` (project row) | Toggle hidden — persists to config |
| `r` (project row) | Rename project label inline |
| `H` | Reveal hidden projects |
| `q` / `esc` | Quit |

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
