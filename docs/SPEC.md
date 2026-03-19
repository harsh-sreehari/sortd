# sortd — Technical Specification

## Problem statement

Downloads folders accumulate files of mixed types, subjects, and origins. Existing organiser tools sort by file type only, producing generic buckets (`Documents/`, `Images/`) that ignore the actual subject matter of files. A compiler design lecture recording should live with compiler design notes, not with holiday videos.

sortd solves this by treating context as the primary routing signal, and file type as secondary metadata.

---

## Core principle

> Ask "what is this about?", not "what format is this?"

File type is used only to decide *how* to peek inside a file, never *where* it goes. The destination is always derived from semantic content and your existing folder structure.

---

## Scope

**sortd watches:**
- `~/Downloads` by default
- Any additional folders explicitly listed in config

**sortd never touches:**
- Folders not listed in the watch config
- Files created by other software in their own managed directories
- Incomplete downloads (`.crdownload`, `.part`, `.tmp`)

---

## The three-tier pipeline

Every file passes through all three tiers in sequence. Each tier either achieves sufficient confidence to act, or passes the file to the next tier.

### Tier 1 — Rules engine

Pure extension and filename pattern matching. No LLM, no file reading, runs in microseconds.

**Handles:**
- System-level files where type determines destination unambiguously
- Incomplete file detection (skip and wait)

**Rules table:**

| Pattern | Destination | Notes |
|---|---|---|
| `.AppImage`, `.deb`, `.rpm`, `.flatpak` | `Software/` | |
| `.iso` | `Software/` | |
| `.exe` (rare on Linux, but possible) | `Software/` | |
| `.crdownload`, `.part`, `.tmp` | ignore | Wait for complete file |
| `.torrent` | ignore | Let the torrent client handle its own output |

**Output:** destination path + confidence score (1.0 for matched rules, 0.0 for pass-through)

**Resolves:** ~5–10% of files in a typical Downloads folder

---

### Tier 2 — Folder graph matching

Crawls the user's existing folder tree, builds a semantic index, and fuzzy-matches the incoming filename against it. No LLM involved — this is string similarity and keyword extraction only.

**Folder graph indexing:**
- Crawls home directory (respecting an ignore list in config)
- Extracts semantic labels from folder names by splitting on separators (`-`, `_`, spaces, camelCase)
- Builds a weighted graph: each folder node has a bag of keywords
- Index is persisted to `~/.local/share/sortd/index.db` (SQLite)
- Re-indexed on `sortd index` command or when folder structure changes significantly

**Matching algorithm:**
1. Extract keywords from filename (strip extension, split on separators, lowercase)
2. Score each folder node: sum of keyword overlap weights
3. Boost score for folders that recently received similar files (learned affinity)
4. Return top match if score exceeds `confidence_threshold` from config

**Examples:**

```
"Compiler-mod-1.pdf"
  keywords: [compiler, mod, 1]
  top match: Academics/Sem6/Compiler Design/ → score 0.91 → move

"networks-assignment-2.pdf"
  keywords: [networks, assignment, 2]
  top match: Academics/Sem6/Networks/ → score 0.88 → move

"invoice_march_2025.pdf"
  keywords: [invoice, march, 2025]
  top match: Finance/Invoices/ → score 0.93 → move

"cnsp292sqsel231.pdf"
  keywords: [cnsp292sqsel231] (no meaningful splits)
  top match score: 0.04 → pass to Tier 3
```

**Output:** destination path + confidence score

**Resolves:** ~50–65% of files that reach it

---

### Tier 3 — LLM inference

Invoked only when Tiers 1 and 2 both fail to produce a confident match. Sends a structured prompt to the local LLM backend with filename, content peek, and folder tree context.

**Content peek strategy:**

| File type | Peek method |
|---|---|
| PDF | Extract text from first page using `pdftotext` |
| `.txt`, `.md`, `.docx` | Read first 300 tokens |
| Image (`.jpg`, `.png`, etc.) | Send to vision-capable model, get description |
| Video, audio | Filename only — too expensive to peek |
| Unknown binary | Filename + extension only |

**Prompt structure:**

```
You are a file organiser. Given a file's name, content preview, and the user's 
existing folder tree, determine the best destination folder.

Filename: {filename}
Extension: {ext}
Content preview: {content_peek}

Existing folder tree:
{folder_tree_abbreviated}

Tasks:
1. Generate 3–5 semantic tags for this file
2. Identify the best matching folder from the tree above
3. If no good match exists, suggest a new folder path to create
4. Return a confidence score 0.0–1.0

Respond only in JSON:
{
  "tags": [...],
  "destination": "path/relative/to/home",
  "is_new_folder": false,
  "confidence": 0.87,
  "reasoning": "one sentence"
}
```

**Decision logic after LLM response:**

| Condition | Action |
|---|---|
| confidence ≥ threshold AND folder exists | Move silently, log |
| confidence ≥ threshold AND new folder | Create folder, move, log with `new_folder` flag |
| confidence < threshold | Park in `.unsorted/`, log with `uncertain` flag |

**Resolves:** majority of files that reach it; genuine unknowns go to `.unsorted/`

---

## LLM backend interface

The LLM is accessed through a thin interface that abstracts the backend:

```go
type LLMBackend interface {
    Tag(ctx context.Context, req TagRequest) (TagResponse, error)
}

type TagRequest struct {
    Filename    string
    Extension   string
    ContentPeek string       // text or image description
    FolderTree  []FolderNode // abbreviated tree
}

type TagResponse struct {
    Tags        []string
    Destination string
    IsNewFolder bool
    Confidence  float64
    Reasoning   string
}
```

**Supported backends:**

`lmstudio` — HTTP calls to LM Studio's OpenAI-compatible API at `localhost:1234`. Recommended. Requires LM Studio running with a vision-capable model loaded.

`llamacpp` — Direct inference via llama.cpp CGo bindings. Self-contained, no external server needed. User downloads a `.gguf` model file on `sortd init`.

Backend is set in `config.toml`. Default is `lmstudio` since the user is on Omarchy/Hyprland and is already using LM Studio.

**Model requirements:**
- Vision capability required (for image files)
- Recommended: Qwen2-VL 2B, LLaVA 1.6 7B, or equivalent
- 1B–7B parameter models work well; 1–3B recommended for low latency

---

## Filesystem watcher

Uses Go's `fsnotify` library which wraps Linux's `inotify` kernel API.

**Events handled:**
- `CREATE` — new file appeared
- `RENAME` (to a watched folder) — treat as new file

**Events ignored:**
- `WRITE` — file being written to (could be mid-download)
- `CHMOD`, `REMOVE`

**Debounce logic:**
After a `CREATE` event, wait 2 seconds before processing. This handles:
- Browsers that create a `.crdownload` then rename to final filename
- Tools that write files in chunks
- Torrent clients that move completed files into Downloads

---

## Storage and logging

All state lives in `~/.local/share/sortd/`:

```
~/.local/share/sortd/
├── sortd.db          # SQLite — sort log, learned affinities, folder index
├── model.gguf        # LLM model (if using llamacpp backend)
└── sortd.log         # plaintext log (mirrors DB for easy grepping)
```

**Sort log schema (SQLite):**

```sql
CREATE TABLE sort_log (
    id          INTEGER PRIMARY KEY,
    timestamp   TEXT,
    filename    TEXT,
    source      TEXT,
    destination TEXT,
    tier        INTEGER,   -- 1, 2, or 3
    confidence  REAL,
    tags        TEXT,      -- JSON array
    action      TEXT,      -- moved | parked | skipped
    corrected   INTEGER    -- 1 if user manually corrected this decision
);
```

**`sortd log` output example:**
```
2025-03-12 14:32  [T2] Compiler-mod-1.pdf
                       → Academics/Sem6/Compiler Design/  (conf: 0.91)

2025-03-12 14:35  [T3] cnsp292sqsel231.pdf
                       → Academics/Sem6/Compiler Design/  (conf: 0.84, new peek)

2025-03-12 14:40  [T3] photo_scan_0042.jpg
                       → .unsorted/  (conf: 0.31, uncertain)
```

---

## systemd user service

sortd runs as a systemd user service, not a system service. This means:
- No root required
- Starts on user login
- Can be managed with `systemctl --user`

**`sortd.service`:**
```ini
[Unit]
Description=sortd file organiser daemon
After=graphical-session.target

[Service]
ExecStart=%h/.local/bin/sortd daemon start --foreground
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```

**`sortd daemon start` installs and enables this automatically.**

---

## Error handling philosophy

- **Never lose a file.** If sortd is unsure, park it. Never delete, never overwrite.
- **Fail loudly in the log, silently to the user.** Errors go to the log file, not to desktop notifications.
- **A crash should not corrupt anything.** All moves are atomic (same-filesystem rename where possible, copy+verify+delete otherwise).

---

## Out of scope (explicitly)

- GUI or tray app
- Cloud sync or remote LLM APIs
- Sorting folders inside folders (only watches the top-level watch folders)
- Auto-renaming files (sortd moves, never renames)
- Windows or macOS support (Linux only for now)
