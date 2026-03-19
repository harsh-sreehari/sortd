# 🌌 Sortd: Intelligent Context-Aware Organizer

> **"Your digital life, autonomously organized."**

`sortd` is a lightweight, background daemon that monitors your messy directories and uses a multi-tier AI engine to put files exactly where they belong. No more hunting for that one PDF in a sea of downloads.

---

## 🚀 Why sortd?

Most file organizers rely on simple extension rules. `sortd` goes deeper. It understands **context**. 

- 🏎️ **Speed**: Immediate organization for known file types (ISO, DEB, APK).
- 🧠 **Intelligence**: Uses keyword-overlap to find existing folders (e.g., `compiler_design.pdf` → `College/Notes/Compiler Design`).
- 👁️ **Vision (LLM)**: When logic fails, sortd uses a local LLM (like `qwen3-VL-4b`) to see what the file actually is.
- 🛡️ **Zero-Loss Safety**: Unsure about a file? We **park** it in `.unsorted/`. We never delete or misplace based on a guess.

---

## 📥 Installation

```bash
# Clone and install with one command
git clone https://github.com/harsh-sreehari/sortd.git
cd sortd
make install
```

This builds the binary and registers the `sortd` systemd service so it starts automatically when you log in.

---

## 🛠️ Getting Started

### 1. Build the "Brain" 🧠
Before `sortd` can be smart, it needs to learn your current folder taxonomy:

```bash
sortd index
```
*Run this anytime you create new important folders in your Documents or Desktop.*

### 2. Manual Sort 🏃
Force a pass over all your watched folders:

```bash
sortd run
```

### 3. Check the Logs 📜
Wondering why a file moved? See the history:

```bash
sortd log
```

### 4. Human-In-The-Loop 👩‍💻
Handle those tricky "Parked" files in `.unsorted/` interactively:

```bash
sortd review
```

---

## ⚙️ Configuration

Located at `~/.config/sortd/config.toml`.

```toml
[watch]
folders = ["/home/user/Downloads"]

[llm]
host  = "http://localhost:1234" # LM Studio / Ollama
model = "qwen3-VL-4b"

[behaviour]
confidence_threshold = 0.75
```

---

## 🪵 Folder Guide

| Type | Destination |
|------|-------------|
| **Rule Match** | Root categories (`Software/`, `Documents/`) |
| **Fuzzy Match** | Existing deep hierarchies (`College/CS/Year3/...`) |
| **Ambiguous** | `~/Downloads/.unsorted/` (Hidden by default) |

---

## 📄 License

MIT © 2026 Harsh Sreehari. Built with ❤️ for organized humans.
