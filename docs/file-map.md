# File Map

## Entry point

| Path | Role |
|---|---|
| `cmd/cnav/main.go` | Flag dispatch, wires sessions + config + ui, evals action |

## Internal packages

| Path | Role |
|---|---|
| `internal/sessions/` | JSONL scanner — reads `~/.claude/projects/`, produces `Session`/`Project` types |
| `internal/ui/` | Bubble Tea model and all view rendering |
| `internal/config/` | Reads/writes `~/.config/cnav/config.toml` (hide/rename) |
| `internal/shell/` | `Action.Render()` shell quoting + `WrapperScript` constant |

## Project config

| Path | Role |
|---|---|
| `admin.toml` | Task runner commands (build, dev, test, deploy, docs) |
| `go.mod` / `go.sum` | Go module definition |
| `package.json` / `bun.lock` | Node deps for VitePress docs site |
| `install.sh` | Builds binary and wires zsh wrapper |
| `.gitignore` | Build artifacts, local config |

## Docs

| Path | Role |
|---|---|
| `docs/` | VitePress site |
| `docs/.vitepress/config.mts` | VitePress config |
| `docs/guide/` | User-facing docs |
| `docs/api.md` | CLI reference |
| `docs/PRD.md` | Product spec |
| `docs/roadmap.md` | Now / Next / Later / Deferred |
| `docs/file-map.md` | This file |
| `docs/CONTEXT.md` | Project glossary and domain vocabulary |
| `docs/adr/` | Architecture decision records |
