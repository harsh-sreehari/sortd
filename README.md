# 🌌 sortd: Intelligent Context-Aware Organizer

> **"Your digital life, autonomously organized."**

**Version: v1.4.0** — Accuracy & UX Milestone

`sortd` is a lightweight background daemon that monitors your messy directories and uses a multi-tier AI engine to put files exactly where they belong. No more hunting for that one PDF in a sea of downloads.

---

## 🚀 Why sortd?

Most file organizers rely on simple extension rules. `sortd` goes deeper. It understands **context**.

- 🏎️ **Speed**: Immediate organization for known file types (ISO, DEB, APK) via rule-based Tier 1.
- 🧠 **Intelligence**: Jaccard similarity + stemming over folder keywords finds the right folder blazing fast (Tier 2).
- 👁️ **Vision (LLM)**: When logic fails, `sortd` uses a local LLM (like `qwen3-VL-4b`) to see what the file actually is (Tier 3).
- 🎓 **Learning**: Correct its mistakes, and it remembers your preferences via an intelligent feedback loop.
- 🛡️ **Zero-Loss Safety**: Unsure about a file? We **park** it in `.unsorted/`. We never delete or misplace based on a guess.
- 🔔 **Notifications**: Optional desktop alerts via `notify-send` when files are organized.
- 🏷️ **Metadata Tagging**: Write sort tags to file extended attributes (`user.sortd.tags`) for searchability.
- ⚡ **Cached Crawl**: The folder index uses `mtime`-based caching, making repeated startup re-indexing near-instant.

---

## 📥 Install

Clone the repository and build from source:

```bash
git clone https://github.com/harsh-sreehari/sortd.git
cd sortd
go build -ldflags "-X main.version=v1.4.0" -o ~/.local/bin/sortd ./cmd/sortd/
```

> **Note:** Install to `~/.local/bin/` (already in `$PATH` on most systems) to avoid conflicts with any existing system-wide binary.

Then run `sortd init`:

```bash
sortd init
```

This will:
1. Create a fully documented `~/.config/sortd/config.toml`
2. Perform the initial folder index crawl
3. Install the systemd user service to `~/.config/systemd/user/sortd.service`

Finally, start the background daemon:

```bash
systemctl --user daemon-reload
systemctl --user enable --now sortd
```

---

## 🛠️ Core Commands

### 1. Build the "Brain" 🧠
Before `sortd` can be smart, it needs to learn your current folder taxonomy:
```bash
sortd index crawl
```
*Run this anytime you create new important folders.*

### 2. Audit a Classification 🔍 *(New in v1.4.0)*
Curious why `sortd` would move a file? Get a full reasoning breakdown without touching the file:
```bash
sortd explain path/to/file.pdf
```

### 3. Manual Sort 🏃
Force a pass over all your watched folders:
```bash
sortd run
```

### 4. Review Parked Files 👩‍💻
Handle tricky "Parked" files in `.unsorted/` interactively and teach the AI:
```bash
sortd review
```

### 5. Smart Rename 🏷️
Let the AI suggest a professional, descriptive name based on file content:
```bash
sortd rename path/to/file.pdf
# Or rename an entire folder's contents
sortd rename --batch path/to/folder/
```

### 6. Check the Logs 📜
```bash
sortd log                    # Recent history
sortd log --verbose          # Show LLM reasoning
sortd log --since 7d         # Last 7 days
sortd log --tag pdf          # Filter by tag
sortd log --page 2           # Paginated view
```

### 7. Export Sort History 📤 *(New in v1.4.0)*
Export the full sort history for external auditing:
```bash
sortd export --format json --output history.json
sortd export --format csv   # Print CSV to stdout
```

### 8. System Status 📊
```bash
sortd status
sortd config check   # Validate config + connectivity
```

### 9. Search & Tags 🔍
```bash
sortd find "project report"
sortd tags                    # Global tag analytics
sortd tags --folder ~/Documents
```
Tags support hierarchy — `College/Forensics` displays nested under `College`.

### 10. Undo a Sort ⏪
```bash
sortd undo      # Undo last move
sortd undo 5    # Undo last 5 moves
```

### 11. Prune Stale Records 🧹
```bash
sortd prune             # Dry-run (safe preview)
sortd prune --confirm   # Apply cleanup
```

### 12. View Index Tree 🌳 *(New in v1.4.0)*
Visualize your indexed folder hierarchy:
```bash
sortd index tree
```

---

## ⚙️ Configuration

Located at `~/.config/sortd/config.toml`. Running `sortd init` generates a fully commented version.

```toml
[watch]
folders = ["/home/user/Downloads"]

[llm]
host  = "http://localhost:1234"
model = "qwen3-VL-4b"

[behaviour]
confidence_threshold = 0.75
create_folders       = true
notifications        = true
xattr                = true

# NEW in v1.4.0
conflict_policy      = "rename"   # rename | skip
auto_rename          = true       # normalize filenames before classification
```

---

## 🆕 What's New in v1.4.0

### Accuracy Core
- **Jaccard similarity + stemming** for Tier 2 fuzzy matching — dramatically increases folder match rates.
- **Sibling filename indexing** — folder keywords enriched with context from files they contain.
- **Watch depth restriction** — prevents the daemon from recursing into deep directory trees.
- **Structural pattern learning** — schema inference gives a confidence boost to schema-consistent folders.
- **`mtime`-based crawl cache** — re-indexing stable directories is now near-instant.

### Visibility & Diagnostics
- **`sortd explain <file>`** — audit exactly why sortd would classify a file, including tier scores and LLM reasoning.
- **`sortd index tree`** — ASCII visualization of the entire indexed folder hierarchy.
- **Paged log** — `sortd log --page N` for large sort histories.
- **`sortd export`** — CSV/JSON export of full sort history for external tools.
- **`sortd config check`** — validates config, directory permissions, and LLM connectivity.

### Advanced Logic & Policy
- **`conflict_policy`** config option — choose `rename` (smart `_1` suffixing) or `skip` when a file already exists at the destination.
- **`auto_rename`** toggle — normalizes filenames like `report_(1).pdf` → `report.pdf` before classification to boost accuracy.
- **Hierarchical tags** — tags like `College/Forensics` are displayed nested in `sortd tags`.

---

## 📄 License

MIT © 2026
Built with ❤️ for unorganized humans.


