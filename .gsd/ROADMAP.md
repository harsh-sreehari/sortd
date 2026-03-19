# ROADMAP.md

> **Current Milestone**: v1.2.0
> **Goal**: Fix broken components, add missing operational commands, and lay groundwork for v1.5 features (tags, find, and intelligent review).

## Must-Haves
- [ ] Fix Tier 1 "Software/" action string
- [ ] Fix store.LogDecision tags handling (nil check + json.Marshal the slice)
- [ ] Implement store.UnsortedFiles() and store.MarkCorrected()
- [ ] Fix sortd.service After= dependency
- [ ] Implement `sortd init` command
- [ ] Implement `sortd daemon stop` and `sortd daemon status` using PID files
- [ ] Implement `sortd find <query>` command
- [ ] Add `original_filename` and `reasoning` columns to `sort_log`
- [ ] Improve `sortd log` with tags, color coding, and filter flags
- [ ] Implement `sortd tags` command
- [ ] Rebuild `sortd review` with NLP routing and affinity updates

## Phases

### Phase 1: Storage Layer & Schema Maint
**Status**: ✅ Complete
**Objective**: Add `original_filename` and `reasoning` columns to SQLite schema. Fix `store.LogDecision` tag handling. Add helper methods (`UnsortedFiles`, `MarkCorrected`).

### Phase 2: Pipeline Fixes & Maint
**Status**: ✅ Complete
**Objective**: Fix Tier 1 "Software/" action string in pipeline. Ensure `sortd.service` has the correct `After=` dependency (e.g., `network.target` or whatever is appropriate). Set up PID file creation.

### Phase 3: Core Commands
**Status**: ✅ Complete
**Objective**: Implement `sortd init` to write default config, create dirs, run initial index, and install the systemd service. Implement `daemon stop` and `daemon status` via PID polling.

### Phase 4: History & Log Improvements
**Status**: ✅ Complete
**Objective**: Overhaul `sortd log` with colors, tags, and new filters (`--tag`, `--tier`, `--parked`, `--today`). Implement `sortd tags` to view aggregated tag data, and `sortd find <query>` to search the history.

### Phase 5: NLP Review System
**Status**: ✅ Complete
**Objective**: Transform `sortd review` into a conversational CLI input. Use intent detection, pass unrecognized input to `MatchTier2` for fuzzy routing, fallback to LLM for unknown descriptions. Hook up the `affinities` table.

---

## Milestone: v1.5.0 (Intelligence & Dashboard)
**Goal**: provide a visual interface for log auditing and manual resolution, improve AI steering, and implement batch operations.

## Must-Haves
- [ ] Implement `sortd dashboard` command (Vite + React UI)
- [ ] Log visualization in the dashboard with "Click to re-route" NLP support
- [ ] Batch resolution of `.unsorted` via UI
- [ ] Implement confidence threshold steering (via UI or config)
- [ ] Add `sortd prune` to clean up logs or old index entries

## Phases

### Phase 6: Web Dashboard Foundation
**Status**: ⬜ Not Started
**Objective**: Scaffold a Vite/React application inside `cmd/sortd/ui/`. Implement a local API in the daemon to serve logs and folder tree data. Register `sortd dashboard` to open the local server.

### Phase 7: Interactive Log Resolution
**Status**: ⬜ Not Started
**Objective**: Implement "Quick Resolve" in the dashboard. Allow users to click a log entry and suggest a new folder via NLP, triggering an automated move and updating affinities.

### Phase 8: Batch Intelligence
**Status**: ⬜ Not Started
**Objective**: Implement multi-select in the dashboard for batch moving. Use the AI to cluster similar files and suggest bulk destinations.
