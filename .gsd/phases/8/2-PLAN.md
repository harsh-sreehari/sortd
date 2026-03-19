---
phase: 8
plan: 2
wave: 1
---

# Plan 8.2: Database Hygiene & Pruning

## Objective
Implement `sortd prune` to keep the database small and relevant by removing entries for files that no longer exist on disk.

## Context
- .gsd/SPEC.md
- cmd/sortd/main.go
- internal/store/store.go

## Tasks

<task type="auto">
  <name>Implement store.Prune</name>
  <files>
    - internal/store/store.go
  </files>
  <action>
    - Loop through all `sort_log` and `folder_index` entries.
    - Check if the `destination` or `path` still exists on disk using `os.Stat`.
    - If missing, delete the record from the database.
    - Log how many records were pruned.
  </action>
  <verify>go test ./internal/store/...</verify>
  <done>Returns count of pruned items.</done>
</task>

<task type="auto">
  <name>Implement sortd prune command</name>
  <files>
    - cmd/sortd/main.go
  </files>
  <action>
    - Add a `prune` subcommand.
    - Usage: `sortd prune`.
    - Call `store.Prune`.
    - Display: "Pruned {n} stale index entries." and "Pruned {m} log entries."
  </action>
  <verify>sortd prune</verify>
  <done>Command runs and clears missing records.</done>
</task>

## Success Criteria
- [ ] Running `prune` on a deleted folder removes it from `folder_index`.
- [ ] Running `prune` after deleting a sorted file removes its `sort_log` entry.
