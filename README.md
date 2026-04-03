# 🌌 sortd: Intelligent Context-Aware Organizer

> **"Your digital life, autonomously organized."**

**Version: v1.3.0** — Reliability & Polish

`sortd` is a lightweight background daemon that monitors your messy directories and uses a multi-tier AI engine to put files exactly where they belong. No more hunting for that one PDF in a sea of downloads.

---

## 🚀 Why sortd?

Most file organizers rely on simple extension rules. `sortd` goes deeper. It understands **context**. 

- 🏎️ **Speed**: Immediate organization for known file types (ISO, DEB, APK).
- 🧠 **Intelligence**: Uses keyword-overlap to find existing folders (e.g., `compiler_design.pdf` → `College/Notes/Compiler Design`).
- 👁️ **Vision (LLM)**: When logic fails, `sortd` uses a local LLM (like `qwen3-VL-4b`) to see what the file actually is.
- 🎓 **Learning**: Correct its mistakes, and it remembers your preferences via an intelligent feedback loop.
- 🛡️ **Zero-Loss Safety**: Unsure about a file? We **park** it in `.unsorted/`. We never delete or misplace based on a guess.
- 🔔 **Notifications**: Optional desktop alerts via `notify-send` when files are organized.
- 🏷️ **Metadata Tagging**: Write sort tags to file extended attributes (`user.sortd.tags`) for searchability.

---

## 📥 Downloading and Running `sortd`

1. **Download/Install:**  
   You can clone the repository and build it locally using Go:
   ```bash
   git clone https://github.com/harsh-sreehari/sortd.git
   cd sortd
   go build -ldflags "-X main.version=v1.3.0" -o sortd ./cmd/sortd/
   sudo mv sortd /usr/local/bin/
   ```

2. **Initialization:**  
   Run the initialization command to set up a fully documented configuration file and install the systemd background service:
   ```bash
   sortd init
   ```

3. **Start the Background Daemon:**  
   Enable and run the background service so your files are sorted automatically:
   ```bash
   systemctl --user enable --now sortd
   ```

---

## 🛠️ Getting Started & Core Commands

### 1. Build the "Brain" 🧠
Before `sortd` can be smart, it needs to learn your current folder taxonomy:
```bash
sortd index
```
*Run this anytime you create new important folders in your Documents or Desktop.*

### 2. Manual Sort 🏃
Force a pass over all your watched folders currently tracking:
```bash
sortd run
```

### 3. Review Parked Files 👩‍💻
Handle tricky "Parked" files in `.unsorted/` interactively and teach the AI:
```bash
sortd review
```

### 4. Smart Rename 🏷️
Let the AI suggest a professional, descriptive name based on file content:
```bash
sortd rename path/to/file.pdf
# Or rename an entire folder's contents
sortd rename --batch path/to/folder/
```

### 5. Check the Logs 📜
Wondering why a file moved? See the history:
```bash
sortd log
sortd log --verbose       # Show LLM reasoning
sortd log --since 7d      # Last 7 days
sortd log --tag pdf       # Filter by tag
```

### 6. System Status 📊
Check the daemon health, LLM connectivity, and lifetime metrics:
```bash
sortd status
```

### 7. Search & Tags 🔍
Find past sort decisions or inspect tag analytics:
```bash
sortd find "project report"
sortd tags --folder ~/Documents
```

### 8. Undo a Sort ⏪
Made a mistake? Restore files to their original location:
```bash
sortd undo      # Undo last move
sortd undo 5    # Undo last 5 moves
```

### 9. Prune Stale Records 🧹
Clean up DB entries for files that no longer exist:
```bash
sortd prune             # Dry-run (safe preview, default)
sortd prune --confirm   # Actually apply the cleanup
```

---

## ⚙️ Configuration

Located at `~/.config/sortd/config.toml`. Running `sortd init` generates a fully commented version — every option is explained inline.

```toml
[watch]
# Directories to monitor for new files
folders = ["/home/user/Downloads"]

[llm]
# Your local LLM endpoint (LM Studio / Ollama)
host  = "http://localhost:1234"
model = "qwen3-VL-4b"

[behaviour]
# Minimum LLM confidence before moving a file (0.0-1.0)
confidence_threshold = 0.75
create_folders       = true

# Optional: desktop notifications via notify-send
notifications = false

# Optional: write sort tags to file extended attributes
xattr = false
```

---

## 🆕 What's New in v1.3.0

- **🔍 LM Studio health check** on daemon startup — warns if offline, keeps sorting with rules.
- **🔔 Desktop notifications** (`notify-send`) for successful file moves, gated by config.
- **🏷️ xattr tagging** — writes `user.sortd.tags` to organized files for filesystem-level searchability.
- **🧹 Safe prune** — `sortd prune` now defaults to a dry-run preview; requires `--confirm` to apply.
- **📝 Self-documenting config** — `sortd init` generates a fully annotated `config.toml`.
- **All Phase 12–13 features**: `sortd undo`, `sortd rename --batch`, `sortd tags --folder`, `--verbose` logs, `sortd status` dashboard, and more.

---

## 📄 License

MIT © 2026  
Built with ❤️ for unorganized humans.
