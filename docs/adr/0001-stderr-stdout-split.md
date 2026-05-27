# ADR 0001 — stderr/stdout split for shell-eval coordination

Date: 2026-05-27  
Status: Accepted

## Context

`cnav` is a TUI that must `cd` the calling shell and optionally launch `claude`. A child process cannot change its parent shell's working directory directly — `chdir(2)` only affects the child's own process.

## Decision

The binary writes its TUI entirely to **stderr** and outputs exactly one line to **stdout**: the shell command to eval (`cd '...'`, optionally `&& claude [--resume ...]`). A thin zsh wrapper function captures stdout and `eval`s it:

```zsh
cnav() {
  local __cnav_cmd
  __cnav_cmd=$(command cnav-bin "$@") || return $?
  [ -n "$__cnav_cmd" ] && eval "$__cnav_cmd"
}
```

Alternatives considered:
- **Temp file** — works but requires cleanup and a shared path convention.
- **PTY / process substitution** — complex, fragile across shells.
- **Named pipe** — same cleanup concerns as temp file.

## Consequences

- All TUI rendering must go to `os.Stderr`. Lipgloss is initialized with `lipgloss.NewRenderer(os.Stderr)`.
- stdout must stay clean — no debug prints, no progress output.
- The binary name is `cnav-bin`; `cnav` is reserved for the wrapper function so the user always runs through the wrapper.
- Empty stdout (user quit) is a valid no-op; the wrapper checks `[ -n "$__cnav_cmd" ]` before eval.
