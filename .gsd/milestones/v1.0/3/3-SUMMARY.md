# Plan 3.1 Summary: Rules Engine (Tier 1)

- Initialized `internal/pipeline/tier1.go` with Type Definitions and Decision struct.
- Implemented static rules for Software (`.appimage`, `.deb`, `.iso`, etc.).
- Added skip rules for temporary files.
- Implemented `MatchTier1` function with confidence 1.0 logic.
