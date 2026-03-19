---
phase: 4
plan: 1
wave: 1
---

# Plan 4.1: Atomic Move and Collision Avoidance

## Objective
Implement a robust filesystem mover that handles atomic cross-device moves and automatically avoids filename collisions.

## Context
- docs/SPEC.md: Goals -> Atomic file moves
- docs/TASKS.md: Task 4.1
- internal/mover/mover.go

## Tasks

<task type="auto">
  <name>Collision Avoidance</name>
  <files>internal/mover/mover.go</files>
  <action>
    Implement `GenerateUniquePath(dest string) string`.
    - If `dest` does not exist, return it.
    - If it does, append `_1`, `_2`, etc. before the extension until a unique path is found.
  </action>
  <verify>go test ./internal/mover/...</verify>
  <done>Unique paths are accurately generated when collisions occur.</done>
</task>

<task type="auto">
  <name>Atomic Move Implemention</name>
  <files>internal/mover/mover.go</files>
  <action>
    Implement `Move(src, dest string) (string, error)`.
    - Resolve the final unique path using `GenerateUniquePath`.
    - Ensure the destination directory exists; if not, create it.
    - Attempt `os.Rename(src, finalPath)`.
    - If it fails because of cross-device link errors, implement a fallback: copy the file securely, then delete the original.
    - Return the final path.
  </action>
  <verify>go test ./internal/mover/...</verify>
  <done>Files are reliably moved, even across mount boundaries, without loss or truncation.</done>
</task>

## Success Criteria
- [ ] No files are overwritten accidentally.
- [ ] Fallback copy-then-delete mechanism works for cross-filesystem moves.
