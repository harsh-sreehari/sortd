---
phase: 7
plan: fix-logCmd
wave: 1
gap_closure: true
---

# Fix: Log Command Output

## Problem
Phase 5 exported a mocked stdout table with no read connection to the `sortd.db` SQLite storage.

## Root Cause
Database records insert but were never wired to `logCmd` because `store.go` was heavily stubbed for querying.

## Tasks

<task type="auto">
  <name>Wire DB Retrieval Logic</name>
  <files>internal/store/store.go, cmd/sortd/main.go</files>
  <action>
    - Add `GetRecentDecisions(limit int) ([]DecisionRecord, error)` to `store.go`.
    - Update `logCmd` to iterate this output and print it using a simple tabular or list layout (like `[Timestamp] /filename -> /destination`).
  </action>
  <verify>go test ./internal/store/... && go run ./cmd/sortd/main.go log</verify>
  <done>User visually retrieves prior moves and sorts safely queried from SQLite.</done>
</task>
