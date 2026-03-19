# SPEC.md — Project Specification

> **Status**: `FINALIZED`

## Vision
sortd is a silent, context-aware file organiser daemon for Linux that treats context as the primary routing signal, using a three-tier pipeline to sort files into existing folder structures based on "what is this about?" rather than just "what format is this?".

## Goals
1. Implement a three-tier sorting pipeline (Rules → Fuzzy Folder Matching → LLM Inference).
2. Ensure fully offline operation using local LLM backends like LM Studio or llama.cpp.
3. Provide seamless background operation via a systemd user service with atomic, non-destructive file moves.

## Non-Goals (Out of Scope)
- No GUI or tray application (CLI-only daemon).
- No cloud-based LLM APIs or remote syncing.
- No auto-renaming of files; only moving/sorting.
- No support for Windows or macOS (Linux-only).

## Users
Linux users (specifically those on systemd-based distributions like Omarchy/Hyprland) who want automated, semantic organization of their Downloads (or other watched) folders without manual intervention.

## Constraints
- Must be written in Go.
- Requires local LLM infrastructure (LM Studio or llama.cpp).
- Depends on Linux-specific tools like `inotify` and `systemd`.
- External tools like `pdftotext` required for PDF content peeking.

## Success Criteria
- [ ] Files are accurately routed to semantic folders with high confidence.
- [ ] Uncertain files are safely parked in `.unsorted/` for manual review.
- [ ] The daemon runs reliably without data loss or corruption during moves.
- [ ] System resource usage remains low by prioritizing non-LLM tiers.
