---
phase: 5
plan: 1
wave: 1
---

# Plan 5.1: Daemon and Run Commands

## Objective
Implement the `run` CLI command to manually sort folders, and set up the foundation for `daemon` mode execution using pipeline integration.

## Context
- docs/SPEC.md: CLI for management
- cmd/sortd/main.go
- internal/pipeline/pipeline.go

## Tasks

<task type="auto">
  <name>Process Directory Logic (Run Command)</name>
  <files>cmd/sortd/main.go</files>
  <action>
    Implement logic for the `runCmd` in cmd/sortd/main.go.
    - Initialize `config`, `store`, `graph`, and `pipeline`.
    - Recursively walk through directories configured in `cfg.Watch.Folders`.
    - For every file found (not filtering out directories, just applying normal ignore/hidden checks), call `pipeline.Process(path)`.
    - Print summary statistics (# moved, # parked, # skipped) to stdout.
  </action>
  <verify>go build ./cmd/sortd/...</verify>
  <done>The run command properly loads the pipeline and executes decisions continuously on existing files.</done>
</task>

<task type="auto">
  <name>Daemon Start Logic</name>
  <files>cmd/sortd/main.go</files>
  <action>
    Implement logic for the `daemonStartCmd`.
    - Initialize `config`, `watcher`, `store`, `graph`, and `pipeline`.
    - Start the `watcher` listening for events.
    - In a goroutine, consume `watcher.Out` channel and pass each emitted path to `pipeline.Process(path)`.
    - Capture `SIGINT` and `SIGTERM` to gracefully shutdown the daemon.
  </action>
  <verify>go build ./cmd/sortd/...</verify>
  <done>The daemon start command hooks up the watcher output directly to the pipeline engine.</done>
</task>

## Success Criteria
- [ ] Manual `sortd run` correctly processes all files currently inside watched folders.
- [ ] `sortd daemon start` boots successfully, watches for new files, and processes them.
