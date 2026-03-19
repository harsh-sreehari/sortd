---
phase: 3
plan: 1
wave: 1
---

# Plan 3.1: Rules Engine (Tier 1)

## Objective
Implement a high-speed rules engine (Tier 1) that handles unambiguous file types (executables, installers, ISOs) and skip rules instantly without expensive processing.

## Context
- docs/SPEC.md: Tier 1 section
- docs/ARCHITECTURE.md: internal/pipeline/tier1.go
- docs/TASKS.md: Task 3.1

## Tasks

<task type="auto">
  <name>Rules Table and Type Definitions</name>
  <files>internal/pipeline/tier1.go</files>
  <action>
    Define the Rule struct and the static table for hardcoded system rules in internal/pipeline/tier1.go.
    - Rule struct: `Extensions []string`, `Pattern string`, `Destination string`.
    - Populate table: `.AppImage`, `.deb`, `.rpm`, `.flatpak`, `.iso`, `.exe` -> `Software/`.
    - Populate skip table: `.crdownload`, `.part`, `.tmp`, `.torrent` -> `skip`.
  </action>
  <verify>go test ./internal/pipeline/...</verify>
  <done>The rules table matches the specification for Tier 1.</done>
</task>

<task type="auto">
  <name>Tier 1 Matcher Logic</name>
  <files>internal/pipeline/tier1.go</files>
  <action>
    Implement the `Match(path string) (Decision, bool)` function.
    - Match by extension (case-insensitive) and filename glob pattern.
    - Return a Decision with confidence 1.0 for matches.
    - Return `Action: "skipped"` for skip rules.
  </action>
  <verify>go test ./internal/pipeline/...</verify>
  <done>Matcher correctly identifies files by extension/pattern and returns the correct decision.</done>
</task>

## Success Criteria
- [ ] Tier 1 identifies system files instantly.
- [ ] Rules accurately map to the specified destinations.
- [ ] Skip rules prevent further processing of temporary or non-target files.
