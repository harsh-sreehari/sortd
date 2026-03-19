# ROADMAP.md

> **Current Phase**: Not started
> **Milestone**: v1.0

## Must-Haves (from SPEC)
- [ ] Functional three-tier sorting pipeline
- [ ] Reliable background daemon via systemd
- [ ] Atomic file moves with collision avoidance
- [ ] CLI for management and review

## Phases

### Phase 1: Foundation
**Status**: ✅ Complete
**Objective**: Project scaffolding, config system, and storage layer.
**Requirements**: REQ-05, REQ-08

### Phase 2: Watcher
**Status**: ✅ Complete
**Objective**: Filesystem monitoring with debounce and filtering.
**Requirements**: REQ-01

### Phase 3: Pipeline Tiers
**Status**: ✅ Complete
**Objective**: Implementation of all three sorting tiers and the orchestrator.
**Requirements**: REQ-02, REQ-03, REQ-04

### Phase 4: Mover
**Status**: ✅ Complete
**Objective**: Atomic filesystem operations and parking logic.
**Requirements**: REQ-06

### Phase 5: CLI
**Status**: ✅ Complete
**Objective**: CLI commands for daemon control, logging, and interactive review.
**Requirements**: REQ-08

### Phase 6: Polish
**Status**: ⬜ Not Started
**Objective**: Graceful shutdown, first-run experience, and final documentation.
**Requirements**: REQ-07
