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
    - Usage: `sortd rename <path>`.
    - AI-suggested behavior:
        - Peek at content and call `SuggestRename`.
        - Prompt user: "Rename to 'Context_Rich_Name.ext'? [Y/n/edit]".
        - If the name already exists, the AI is asked for a *different* descriptive name instead of using suffixes (`_1`).
    - Move should happen atomically via `mover.Move`.
  </action>
  <verify>sortd rename mystery_blob.bin</verify>
  <done>Renames using descriptive context-rich titles.</done>
</task>

<task type="auto">
  <name>Implement "Teaching" (Affinities) Logic</name>
  <files>
    - internal/llm/backend.go
    - internal/store/store.go
    - internal/pipeline/tier3.go
  </files>
  <action>
    - Ensure `MarkCorrected` in `store.go` captures the user's manual correction intent.
    - In `MatchTier3`, fetch the `affinities` from the DB for the current file's tags/type.
    - Include these as "Past Lessons" in the LLM prompt: "In the past, the user manually moved files with similar content/tags to: [Folders]".
    - This biases the AI toward user-taught destinations.
  </action>
  <verify>sortd review (and check if subsequent similar files follow the learned path)</verify>
  <done>User corrections improve future AI suggestions.</done>
</task>

## Success Criteria
- [ ] `sortd rename` successfully renames a file using AI.
- [ ] `sortd tutor` explains a previous decision clearly.
- [ ] All move operations maintain unique filename safety.
