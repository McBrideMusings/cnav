# ADR 0002 — Two-phase JSONL scan for large session files

Date: 2026-05-27  
Status: Accepted

## Context

Claude Code session files are append-only JSONL. Most are small, but active long-running sessions can exceed tens of MB. Scanning the full file for every session on startup would make launch feel slow and consume significant memory.

cnav needs two distinct things from each file:
1. **Metadata** (session ID, CWD, start time) — always in the first few lines.
2. **Preview text** (last user message, longest assistant reply) — always near the end.

## Decision

Files above 5 MB ("large files") use a two-pass strategy:

1. **Head pass** — read the first 200 lines. Extract session ID, CWD, and start time. Stop early once all three are found.
2. **Tail pass** — seek to `fileSize - 256 KB`, discard a partial line, then scan to EOF. Extract preview text.

Files under 5 MB are scanned in a single pass (both metadata and preview).

A 32-goroutine semaphore caps parallelism so `Scan` doesn't open hundreds of file descriptors simultaneously.

## Consequences

- Preview text for large files reflects the last 256 KB only. In practice this covers many pages of conversation, so the most-recent user message is almost always captured.
- The head/tail boundary could theoretically miss a preview that falls between byte 200-of-lines and `fileSize - 256 KB`, but this is extremely rare and acceptable for a nav tool.
- `scanSessionLines` is called twice for large files with different `wantMeta`/`wantPreview` flags; the `scanState` struct is threaded through both calls so the tail pass can accumulate on top of what the head pass found.
