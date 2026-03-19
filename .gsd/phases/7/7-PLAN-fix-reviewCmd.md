---
phase: 7
plan: fix-reviewCmd
wave: 1
gap_closure: true
---

# Fix: Interactive Review Command

## Problem
Phase 5 shipped `sortd review` with only a mocked print statement "Interactive review logic initialized. (mock)", preventing users from being able to meaningfully clean the `.unsorted/` directory without manual GUI file management.

## Root Cause
Original scope of Phase 5 execution omitted the final `bufio` STDIN iteration logic in favor of mocked out commands to satisfy the milestone quickly.

## Tasks

<task type="auto">
  <name>Fix Interactive Prompts</name>
  <files>cmd/sortd/main.go</files>
  <action>
    - Add `bufio.NewScanner(os.Stdin)` to read user input.
    - Find files natively in the `cfg.Watch.Folders[0] + "/.unsorted"` directory using `os.ReadDir`.
    - Loop through each parked file:
      - Display filename to the user.
      - Wait for path destination string.
      - If destination entered and isn't empty, pipe through `mover.Move()`.
  </action>
  <verify>go run ./cmd/sortd/main.go review --help</verify>
  <done>User can be interactively prompted for parked files sequentially.</done>
</task>
