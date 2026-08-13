# cnav

A small TUI for jumping between Claude Code projects and resuming past chats.

## Language

### Domain

<!-- Project-specific terms get added here as they resolve. -->

**session** — one Claude Code conversation, stored as a single `.jsonl` file under `~/.claude/projects/<slug>/`.

**project** — a git repository's main checkout (or, for a non-repo directory, the directory itself); groups one or more sessions by their **repo root**.

**repo root** — the main checkout a session's `CWD` belongs to. Read from `<cwd>/.git`: in a git worktree that's a file holding `gitdir: <repo>/.git/worktrees/<name>`, which is the only reliable link back since a worktree can live anywhere on disk. Everything else (a normal checkout, a submodule, a non-repo dir) is its own root.

**CWD** — the working directory recorded in the first `cwd` field of a session's JSONL; the canonical key for a project. Note: the slug-encoded directory name in the filesystem path is *not* used because the encoding isn't reversible when a path component contains a hyphen.

**worktree** — one git worktree of a project, holding the chats run inside it. Rendered as a `⎇ <dir-name>` row under its project, expandable into its chats exactly like a project row.

**missing session** — a session whose `CWD` no longer exists on disk (retired worktree, deleted temp dir, moved project). Dropped by `Scan` with no trace: not hidden, not counted, not recoverable, since there is nowhere to `cd`. Distinct from a **hidden project**, which the user hid with `opt+x` and can bring back with `opt+h`.

**action** — the `shell.Action` struct that the TUI sets on exit. `Action.Render()` produces the shell command string (`cd`, optionally `&& claude` or `&& claude --resume <id>`) written to stdout for the wrapper to eval. Empty string means quit with no action.

**wrapper** — the `cnav()` zsh function installed into `.zshrc` via `eval "$(cnav-bin init)"`. Captures the binary's stdout and `eval`s it so the parent shell changes directory. The binary cannot change its parent shell's directory directly.

**preview** — the truncated text shown on each row. Two modes toggled by `p`: *you* (last non-skippable user message) and *claude* (longest assistant reply). Stored as `Session.Preview` and `Session.AssistantPreview`.

**large file** — a session JSONL exceeding 5 MB. Triggers a two-phase scan: first 200 lines for metadata (ID, CWD, start time), then the last 256 KB for preview text. Avoids unbounded memory use on very long sessions.

**slug-encoded path** — the directory name Claude Code uses under `~/.claude/projects/` (hyphens replace slashes). cnav does *not* derive `CWD` from this slug because the reverse mapping is ambiguous; it reads `CWD` from the JSONL content instead.

**auto-disambiguation** — when two visible projects share the same `filepath.Base(CWD)`, cnav appends the parent directory name: `.claude (pierce/)` vs `.claude (Projects/)`. A custom `name` in config always overrides this.

**config** — per-project overrides (hide/rename) stored in `~/.config/cnav/config.toml`. Written atomically (write to `.tmp`, then rename). Records with no active overrides are pruned on each save.

### Architecture

<!-- Seeded on first run of improve-codebase-architecture. Don't seed up front. -->

## Relationships

<!-- Filled in as the model matures. -->

## Flagged ambiguities

<!-- When terms get used ambiguously and resolved, capture here. -->
