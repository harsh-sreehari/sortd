## Phase 1 Verification

### Scaffolding
- [x] All packages (watcher, pipeline, graph, peek, llm, mover, store, config) initialized with basic structure.
- [x] CLI entry point (main.go) correctly handles multiple subcommands using Cobra.
- VERIFIED (evidence: `go run cmd/sortd/main.go --help` output pass)

### Configuration System
- [x] Defined all necessary config structs and implemented DefaultConfig.
- [x] LoadConfig correctly parses TOML and merges with defaults.
- [x] Implemented home-directory (~) expansion for all path fields.
- VERIFIED (evidence: `go test ./internal/config/...` output pass)

### Storage Layer
- [x] Implemented schema migration for sort_log, folder_index, and affinities tables.
- [x] Successfully configured pure Go SQLite (modernc.org/sqlite).
- VERIFIED (evidence: `go test ./internal/store/...` output pass)

### Verdict: PASS
