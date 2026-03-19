# 🌌 sortd: Intelligent Context-Aware Organizer

> **"Your digital life, autonomously organized."**

**Version: v1.2.0**

`sortd` is a lightweight background daemon that monitors your messy directories and uses a multi-tier AI engine to put files exactly where they belong. No more hunting for that one PDF in a sea of downloads.

---

## 🚀 Why sortd?

Most file organizers rely on simple extension rules. `sortd` goes deeper. It understands **context**. 

- 🏎️ **Speed**: Immediate organization for known file types (ISO, DEB, APK).
- 🧠 **Intelligence**: Uses keyword-overlap to find existing folders (e.g., `compiler_design.pdf` → `College/Notes/Compiler Design`).
- 👁️ **Vision (LLM)**: When logic fails, `sortd` uses a local LLM (like `qwen3-VL-4b`) to see what the file actually is.
- 🎓 **Learning**: Correct its mistakes, and it remembers your preferences via an intelligent feedback loop.
- 🛡️ **Zero-Loss Safety**: Unsure about a file? We **park** it in `.unsorted/`. We never delete or misplace based on a guess.

---

## 📥 Downloading and Running `sortd`

1. **Download/Install:**  
   You can clone the repository and build it locally using Go:
   ```bash
   git clone https://github.com/harsh-sreehari/sortd.git
   cd sortd
   go build -o sortd ./cmd/sortd/
   sudo mv sortd /usr/local/bin/
   ```

2. **Initialization:**  
   Run the initialization command to set up configuration files and install the systemd background service:
   ```bash
   sortd init
   ```

3. **Start the Background Daemon:**  
   Enable and run the background service so your files are sorted automatically:
   ```bash
   systemctl --user enable --now sortd
   sortd daemon start
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
```

### 5. Check the Logs 📜
Wondering why a file moved? See the history:
```bash
sortd log
```

### 6. Subcommand Autocompletion ✨
`sortd completion` is a built-in feature that generates shell auto-completion scripts (for bash, zsh, fish, or powershell), allowing you to press `[Tab]` to autocomplete `sortd` commands.
To load completions in your shell:
```bash
source <(sortd completion bash)
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

## 📄 License

MIT © 2026
Built with ❤️ for unorganized humans.
