# ADR 0003 — Unified project+session list instead of separate tabs

Date: 2026-05-27  
Status: Accepted

## Context

cnav originally had two separate views (Chats / Projects) switchable by tab. Each view was a flat list. This meant filtering only applied to the active view, and users had to switch tabs to see both projects and their sessions.

## Decision

Replace the two-tab layout with a **single unified list** where project rows are inline-expandable. Sessions are shown as indented child rows under their parent project. Projects are collapsed by default; space/→ expands them.

## Consequences

- A single `visibleRows()` function drives both rendering and cursor logic. Each `row` struct carries a `project` pointer and an optional `session` pointer; `row.isProject()` distinguishes the two types.
- Filtering now searches all sessions across all projects simultaneously. When a project name doesn't match but some of its sessions do, the project is auto-expanded and only the matching sessions are shown (`autoExpand` map in `filteredProjects`).
- The header bar shows active state inline (sort order, preview mode, filter text) rather than via tab labels.
- Navigation keys (`←`/`h` to collapse/ascend, `→`/`l` to expand/descend) replace the old tab-switch keys.
- The footer key hint changes dynamically based on whether the cursor is on a project row or a chat row.
