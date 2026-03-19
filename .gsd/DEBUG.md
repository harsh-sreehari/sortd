# Debug Session: DEBUG-001

## Symptom
Files are being moved recursively into the same folder (or seen as new files after a rename), causing a loop where filenames grow with `_1` suffixes.

**When:** When a file is already in a destination folder that the LLM suggests (e.g. file in Downloads, LLM suggests moving to Downloads).
**Expected:** The system should recognize the file is already in the correct destination and skip the move.
**Actual:** The system moves the file, triggers a collision rename, and then re-processes the renamed file as a new event.

## Gather Evidence

## Resolution

**Root Cause:** The `Mover.Move` function was using `GenerateUniquePath` unconditionally, which added a `_1` suffix even if the target destination was the same as the source file. This caused a loop because the daemon saw the renamed file as a "new" file.

**Fix:** 
1. Updated `internal/mover/mover.go` to normalize paths and return early if `src == dest`.
2. Updated `internal/pipeline/pipeline.go` to re-categorize same-path moves as "skipped" in the log for clarity.

**Verified:** Created a reproduction script in `cmd/repro/main.go` that confirmed Case 1 (same path) and Case 2 (dest dir) no longer cause renames.

**Regression Check:** `go test ./...`
