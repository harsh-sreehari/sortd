# sortd

A silent, context-aware file organiser daemon for Linux. Watches your Downloads folder and sorts files into your existing folder structure using a three-tier pipeline — rule matching, fuzzy folder-graph matching, and a local LLM for ambiguous cases.

No cloud. No config required to start. No interruptions.

---

## How it works

sortd watches your Downloads folder via inotify. When a file lands, it runs through three tiers in order:

**Tier 1 — Rules engine**
Extension and filename pattern matching. Handles obvious system files instantly (`.AppImage`, `.iso`, `.deb`). Costs nothing.

**Tier 2 — Folder graph matching**
Crawls your existing folder tree, builds a semantic map, and fuzzy-matches the incoming file against it. A file named `networks-assignment-2.pdf` finds `Academics/Sem6/Networks/` without needing the LLM.

**Tier 3 — LLM inference**
For files that pass through Tiers 1 and 2 without a confident match. Peeks at file content (first page of PDFs, vision model for images), generates tags, and routes to the best destination. Creates new folders when needed and logs them for review.

Files the LLM is still uncertain about land in `~/Downloads/.unsorted/` for you to handle when you feel like it.

---

## Key design decisions

- **Context over file type.** A lecture recording `.mp4` of a compiler design class goes into `Academics/Sem6/Compiler Design/`, not `Media/Videos/`.
- **Watch only what you opt in.** sortd never touches folders it was not explicitly told about. Your `node_modules`, game libraries, and torrent client folders are completely safe.
- **LLM runs rarely.** Tiers 1 and 2 handle most files. The LLM wakes up only for genuinely ambiguous cases — roughly 10–30% of files depending on your Downloads habits.
- **Fully offline after setup.** sortd connects to a locally running LM Studio instance over HTTP. Nothing leaves your machine.
- **No interruptions.** sortd never pops up dialogs or blocks your workflow. You check what it did when you want to, not when it demands.

---

## Installation

```bash
# Download the binary and default model
curl -L https://github.com/you/sortd/releases/latest/download/sortd-linux-amd64 -o sortd
chmod +x sortd
./sortd init
```

`sortd init` will:
1. Create `~/.config/sortd/config.toml` with sane defaults
2. Download the default model (~600MB, one time only)
3. Install the systemd user service

---

## Usage

```bash
sortd daemon start       # start the background watcher
sortd daemon stop        # stop it
sortd daemon status      # check if running

sortd log                # show recent sort history
sortd log --today        # today only
sortd review             # show files parked in .unsorted/
sortd review --decide    # interactively resolve uncertain files

sortd run                # manually trigger a sort pass on Downloads
sortd index              # re-index your folder tree (run after big restructures)
```

---

## Configuration

sortd generates a default config on first run at `~/.config/sortd/config.toml`.

```toml
[watch]
folders = ["~/Downloads"]
ignore = ["~/Downloads/.unsorted"]

[llm]
backend = "lmstudio"          # lmstudio | llamacpp
host = "http://localhost:1234"
model = "default"             # or path to a .gguf file

[behaviour]
split_by_type = false         # if true, creates pdf/ img/ subfolders inside destinations
confidence_threshold = 0.75   # below this, park in .unsorted instead of moving
create_folders = true         # allow LLM to create new folders when needed
log_path = "~/.local/share/sortd/sortd.log"
```

---

## Requirements

- Linux (tested on Hyprland/Omarchy, works on any systemd distro)
- LM Studio running locally with a vision-capable model loaded (e.g. Qwen2-VL, LLaVA)
- Go 1.22+ (only if building from source)

---

## Project status

Active development. Core pipeline (Tiers 1–2) is stable. Tier 3 LLM routing is in progress. See `TASKS.md` for the full build roadmap.
