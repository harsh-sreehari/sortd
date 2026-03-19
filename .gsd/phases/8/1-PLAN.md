---
phase: 8
plan: 1
wave: 1
---

# Plan 8.1: AI-Assisted Renaming & Tutoring

## Objective
Enhance the CLI with the ability to suggest clean filenames based on content and provide deeper reasoning ("tutor") for why files were organized into specific tiers.

## Context
- .gsd/SPEC.md
- cmd/sortd/main.go
- internal/llm/backend.go
- internal/mover/mover.go

## Tasks

<task type="auto">
  <name>Implement SuggestRename in LLM</name>
  <files>
    - internal/llm/backend.go
    - internal/llm/llm.go
  </files>
  <action>
    - Add `SuggestRename(filename, content string) (string, error)` to the `LLMBackend` interface.
    - Implement it in `LMStudioBackend`.
    - The prompt should ask for a "clean, professional, and descriptive" filename based on the provided filename and optional content peek.
    - Ensure it returns ONLY the new filename (including the original extension).
  </action>
  <verify>go test ./internal/llm/...</verify>
  <done>Interface updated and LM Studio implementation returns a string.</done>
</task>

<task type="auto">
  <name>Implement sortd rename command</name>
  <files>
    - cmd/sortd/main.go
  </files>
  <action>
    - Add a `rename` subcommand to the sortd CLI.
    - Usage: `sortd rename <path>` (AI-suggested) or `sortd rename <path> <newname>` (manual).
    - If manual, call `mover.Move(path, newname)`.
    - If AI-suggested:
        - Call `SuggestRename` via the LLM.
        - Show the suggestion: "Rename to 'new_name.ext'? [Y/n]".
        - If confirmed, execute the move.
  </action>
  <verify>sortd rename --help</verify>
  <done>Command exists and renames files correctly.</done>
</task>

<task type="auto">
  <name>Implement sortd tutor command</name>
  <files>
    - cmd/sortd/main.go
  </files>
  <action>
    - Add a `tutor <query>` subcommand.
    - It should search for the last log entry matching the query.
    - Display the tier, tags, and reasoning.
    - Call the LLM with a "tutor prompt": "A user is confused why this file was moved here. Based on the reasoning '{reasoning}', explain in 2-3 sentences the logic behind this classification."
  </action>
  <verify>sortd tutor mystery_blob</verify>
  <done>Explains the sorting logic conversationally.</done>
</task>

## Success Criteria
- [ ] `sortd rename` successfully renames a file using AI.
- [ ] `sortd tutor` explains a previous decision clearly.
- [ ] All move operations maintain unique filename safety.
