# sortd — Intelligent File Organizer Daemon

> An autonomous background daemon that watches your directories and moves files exactly where they belong — using a multi-tier AI decision engine powered by a local LLM.

---

## How It Works

`sortd` uses a **3-tier decision pipeline** on every incoming file:

| Tier | Method | Example |
|------|--------|---------|
| **1 — Rules** | Instant extension matching | `.deb` → `Software/`, `.crdownload` → skip |
| **2 — Fuzzy** | Keyword overlap against your folder index | `CST302_COMPILER_DESIGN.pdf` → `Documents/College/Notes/Compiler Design/` |
| **3 — LLM** | Local LLM content analysis for ambiguous files | Image with chart → `Research/Charts/` |

If confidence is too low at all 3 tiers, the file is **safely parked** in `.unsorted/` for you to review manually — nothing ever gets silently lost.

---

## Installation

### Prerequisites
- [Go 1.21+](https://go.dev/dl/)
- A running [LM Studio](https://lmstudio.ai/) instance (or any OpenAI-compatible local API)

### Install

```bash
git clone https://github.com/harsh-sreehari/sortd.git
cd sortd
make install
```

This builds the binary and registers the `sortd` daemon as a **systemd user service** that starts automatically on login.

> **Or install without `make`:**
> ```bash
> go install ./cmd/sortd/...
> systemctl --user enable sortd --now
> ```

---

## First Run

### 1. Index your folders

Before sortd can place files intelligently, it needs to scan your existing folder tree:

```bash
sortd index
```

This crawls your `~/Documents`, `~/Desktop`, and `~/Downloads` directories and builds a local keyword index. Run this again anytime you reorganise your folders.

### 2. Verify the daemon is running

```bash
systemctl --user status sortd
```

From this point on, any file dropped into your `~/Downloads` will be automatically organised.

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `sortd run` | Manually trigger a sort pass on all watched folders |
| `sortd index` | Re-crawl and rebuild the folder keyword index |
| `sortd log` | Show the last 20 file move decisions |
| `sortd review` | Interactively map files parked in `.unsorted/` |
| `sortd daemon start` | Start the background file watcher |

---

## Configuration

Config is auto-created at `~/.config/sortd/config.toml` on first run.

```toml
[watch]
# Directories to monitor
folders = ["/home/user/Downloads"]

[llm]
# Your local LLM API endpoint (LM Studio, Ollama, etc.)
host  = "http://localhost:1234"
model = "qwen3-VL-4b"

[behaviour]
# Minimum confidence (0.0–1.0) before a file is moved instead of parked
confidence_threshold = 0.75
db_path = "~/.local/share/sortd/sortd.db"
```

### Changing your LLM model

`sortd` works with any model exposed through an OpenAI-compatible API. After updating `config.toml`, restart the daemon:

```bash
systemctl --user restart sortd
```

---

## Where are my files?

| Scenario | Location |
|----------|----------|
| File moved automatically | Inside your `~/Documents` subfolder hierarchy |
| File was uncertain | `~/Downloads/.unsorted/` — run `sortd review` to handle |
| Decision history | `~/.local/share/sortd/sortd.db` (view via `sortd log`) |

---

## Uninstall

```bash
systemctl --user disable sortd --now
rm ~/.config/sortd/config.toml
rm -rf ~/.local/share/sortd/
rm ~/go/bin/sortd
```

---

## License

MIT — see [LICENSE](LICENSE).
