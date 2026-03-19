---
phase: 2
plan: 1
wave: 1
---

# Plan 2.1: Watcher Implementation

## Objective
Implement the filesystem watcher that monitors specified folders, applies debounce logic, and filters out irrelevant files before passing them to the pipeline.

## Context
- docs/SPEC.md
- docs/ARCHITECTURE.md
- docs/TASKS.md: Task 2.1
- internal/config/config.go

## Tasks

<task type="auto">
  <name>Watcher Lifecycle and Watch Management</name>
  <files>internal/watcher/watcher.go</files>
  <action>
    Implement the basic Watcher struct and lifecycle methods in internal/watcher/watcher.go.
    - Define Watcher struct: `cfg *config.Config`, `Out chan string`.
    - Implement `New(cfg *config.Config)` and `Start(ctx context.Context)`.
    - Use `fsnotify` to add all folders from `cfg.Watch.Folders` to the watcher.
  </action>
  <verify>go test -v ./internal/watcher/...</verify>
  <done>Watcher correctly initializes and manages folder watches via fsnotify.</done>
</task>

<task type="auto">
  <name>Event Handling, Debounce, and Filtering</name>
  <files>internal/watcher/watcher.go</files>
  <action>
    Implement the main event loop with debounce and filtering.
    - Listen for `CREATE` and `RENAME` events.
    - Implement a debounce timer (default 2s) for each file path.
    - Filter out: `*.crdownload`, `*.part`, `*.tmp`, `*.download`, hidden files (`.*`), and the `.unsorted` folders.
    - Emit the absolute path to the `Out` channel after the debounce interval.
  </action>
  <verify>go test -v ./internal/watcher/...</verify>
  <done>The watcher emits valid file paths only after the debounce interval and correctly filters out temporary or hidden files.</done>
</task>

## Success Criteria
- [ ] Watcher monitors all configured folders.
- [ ] File creation events are debounced correctly.
- [ ] Temporary and hidden files are filtered without emitting events.
- [ ] Watcher emits absolute paths of valid files to the output channel.
