---
phase: 1
plan: 1
wave: 1
---

# Plan 1.1: Core Scaffolding

## Objective
Initialize the standard Go multi-package directory structure and create the project entry point.

## Context
- .gsd/SPEC.md
- docs/ARCHITECTURE.md
- docs/TASKS.md

## Tasks

<task type="auto">
  <name>Directory Structure Initialization</name>
  <files>cmd/sortd/, internal/watcher/, internal/pipeline/, internal/graph/, internal/peek/, internal/llm/, internal/mover/, internal/store/, internal/config/</files>
  <action>
    Create the directory structure as specified in docs/ARCHITECTURE.md.
    - Create cmd/sortd/
    - Create internal/ subdirectories for watcher, pipeline, graph, peek, llm, mover, store, and config.
  </action>
  <verify>ls -R cmd internal</verify>
  <done>All specified directories exist.</done>
</task>

<task type="auto">
  <name>CLI Entry Point Skeleton</name>
  <files>cmd/sortd/main.go</files>
  <action>
    Initialize cmd/sortd/main.go with a basic Cobra CLI setup.
    - Define root command 'sortd'.
    - Add placeholder subcommands for daemon (start, stop, status), log, review, run, and index.
  </action>
  <verify>go run cmd/sortd/main.go --help</verify>
  <done>The help output shows the registered subcommands.</done>
</task>

## Success Criteria
- [ ] Directory structure initialized according to ARCHITECTURE.md.
- [ ] Go executable can be run and shows the intended CLI structure.
