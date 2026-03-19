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

## Milestone: v1.5.0 (Vision & Advanced Intelligence)
**Goal**: Implement vision-assisted sorting, stabilize filesystem interactions, and provide advanced CLI steering.

## Must-Haves
- [x] Implement Vision-capable "Peek" strategy for images (Qwen2-VL support)
- [x] Stabilize Watcher with WRITE event debouncing (prevents premature sorting)
- [x] Improved "New Folder" logic (Pattern awareness of sibling folders)
- [x] Ghost folder prevention (Clearing index on re-crawl)
- [ ] Add `sortd rename` subcommand for AI-suggested filenames
- [ ] Implement `sortd prune` to clean records for missing files

## Phases

### Phase 6: Vision & Intelligence Bridge
**Status**: ✅ Complete
**Objective**: Integrate `DescribeImage` into the pipeline. Ensure Tier 3 uses visual content for OCR and subject identification. Fix stale indexing (DELETE before CRAWL).

### Phase 7: Operational Stability
**Status**: ✅ Complete
**Objective**: Re-calculate debounce on WRITE events to handle slow browsers/compilers. Fix `sortd review` source path bugs for parked files.

### Phase 8: Advanced CLI Steering
**Status**: ✅ Complete
**Objective**: Implement context-rich `sortd rename` (AI-suggested names). Implement "Teaching Mode" by feeding user feedback into the AI's future prompts. Implement global pruning.
