# CLI Reference

## Usage

```
cnav                    launch the TUI (use via the cnav shell function)
cnav init               print the shell wrapper function (eval into your shell)
cnav --print-shell      same as init
cnav -h, --help         show help
```

## Shell wrapper

`cnav` must be invoked as a shell function, not a bare binary — a child process cannot change its parent shell's working directory. The wrapper captures the chosen command on stdout and `eval`s it:

```sh
cnav() {
  local __cnav_cmd
  __cnav_cmd=$(command cnav-bin "$@") || return $?
  [ -n "$__cnav_cmd" ] && eval "$__cnav_cmd"
}
```

Add to your shell by running `eval "$(cnav-bin init)"` in `.zshrc`.

## Output protocol

The binary emits exactly one line to stdout on a successful selection (empty on quit/escape):

```
cd '/path/to/project'
cd '/path/to/project' && claude
cd '/path/to/project' && claude --resume 'session-id'
```

All TUI output goes to stderr so stdout remains clean for the wrapper.

## Config file

`~/.config/cnav/config.toml` — written by the TUI, readable by hand. Schema:

```toml
[[project]]
cwd    = "/absolute/path"   # required key
hidden = true               # omit or false to show
name   = "custom label"     # omit to use basename
```

Records with no active overrides are pruned on save.
