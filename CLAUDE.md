# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
./admin build    # go build ./...
./admin dev      # go run ./cmd/cnav  (TUI draws on stderr)
./admin test     # go test ./...
./admin vet      # go vet ./...
./admin fmt      # gofmt -w .
./admin deploy   # go build -o $GOPATH/bin/cnav-bin ./cmd/cnav
```

Run a single test: `go test ./internal/sessions/ -run TestName -v`

After code changes while `./admin dev` is running in another terminal: `./admin reload` to rebuild and relaunch.

## Documentation

This project has a VitePress docs site under `docs/`. Run `./admin docs` (or `bun run docs:dev`) to read it on `http://localhost:5193`.

Keep these in sync as you work:

| File | Update when |
|---|---|
| `docs/PRD.md` | Product behavior, scope, or surface area changes |
| `docs/roadmap.md` | Direction shifts, an initiative ships, or a decision is deferred |
| `docs/file-map.md` | Major files/folders are added, removed, renamed, or moved |
| `docs/api.md` | CLI surface changes |
| `docs/guide/` | Install or usage flow changes |

Don't write new top-level planning / phase / feature docs in `docs/` — file a GitHub issue instead. `roadmap.md` is the only forward-looking doc.

## Architecture

`cnav` is a Bubble Tea TUI that reads `~/.claude/projects/` and outputs a shell command on stdout, which the shell wrapper `eval`s to `cd` and optionally launch/resume Claude. The binary cannot change the parent shell's directory directly — stdout stays clean for the eval; the TUI renders entirely on stderr.

**Data flow:**

1. `sessions.Scan` walks `~/.claude/projects/` in parallel (32-goroutine semaphore), parsing each `.jsonl` file into a `Session`. Large files (>5 MB) use a two-phase scan: first 200 lines for metadata, last 256 KB for preview.
2. `sessions.GroupByProject` buckets sessions by `CWD` into `Project` slices.
3. `ui.New` builds the Bubble Tea model; `ui.Model.Update` handles all key events and produces an `Action` on exit.
4. `main` calls `model.Action.Render()` which emits a `cd '...' && claude [--resume id]` string to stdout for the wrapper to eval.

**Packages:**

- `internal/sessions` — JSONL scanner and `Session`/`Project` types. No UI dependency.
- `internal/ui` — Entire Bubble Tea model and view. Depends on sessions and shell for `Action`.
- `internal/config` — Reads/writes `~/.config/cnav/config.toml` (hide/rename per-project). Atomic saves via `.tmp` rename.
- `internal/shell` — `Action.Render()` (shell quoting) and the `WrapperScript` constant installed into `.zshrc`.
- `cmd/cnav` — Entry point: flag dispatch, wires sessions + config + ui, evals the resulting action.

**Key invariants:**
- `ui.Model` is value-typed (Bubble Tea pattern) — all state mutations return a new model.
- Config writes are non-fatal: `saveConfig` prints to stderr and keeps running on failure.
- Session CWD comes from the first `cwd` field in the JSONL, not the slug-encoded directory name (which isn't reliably reversible).
- Sessions whose `CWD` no longer exists on disk are silently dropped in `Scan` — there is nowhere to `cd`, so the row is dead weight. The footer reports the count.
- A session's repo root comes from reading `<cwd>/.git` — a git worktree's `.git` is a file holding `gitdir: <repo>/.git/worktrees/<name>`. Worktrees can live anywhere on disk, so path patterns are never used to detect them. `GroupByProject` buckets on that root, so a repo's worktree chats appear under the main checkout, tagged `⎇ <worktree>` on the chat row.
- Basename collisions in the project list are auto-disambiguated by appending the parent directory (`base (parent/)`); a manual `name` in config always wins.
