---
phase: 6
plan: 1
wave: 1
---

# Plan 6.1: Default Config and First Run

## Objective
Implement first-run experience: automatically scaffolding default configuration logic so users don't face errors on the first execution.

## Context
- docs/SPEC.md: First-run experience and config fallbacks
- cmd/sortd/main.go
- internal/config/config.go

## Tasks

<task type="auto">
  <name>Scaffold Default Config</name>
  <files>internal/config/config.go, cmd/sortd/main.go</files>
  <action>
    - Add logic inside `config.LoadConfig` to build `~/.config/sortd/` if it doesn't exist.
    - If `config.toml` does not exist, write a sensible default template explicitly to disk before attempting to decode it.
    - The default template should watch `~/Downloads` and map SQLite to `~/.config/sortd/sortd.db`.
  </action>
  <verify>rm -rf ~/.config/sortd && go run ./cmd/sortd/main.go log</verify>
  <done>Initialization cleanly handles absence of config layout by creating standard parameters instead of crashing.</done>
</task>

## Success Criteria
- [ ] Users do not need to manually configure the tool for a standard Downloads folder setup.
- [ ] Required folders and config hierarchies accurately scaffold.
