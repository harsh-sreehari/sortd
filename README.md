# 🌌 Sortd: Intelligent Context-Aware File Organizer

![Sortd Banner](./banner.png)

> **"Because your computer shouldn't be a messy garage."**

`sortd` is a lightweight, background daemon that monitors your directories and automatically organizes incoming files into a logical folder taxonomy. It uses a triple-tier decision system—ranging from fast extension rules to local LLM context analysis—to ensure every file ends up exactly where it belongs.

## ✨ Features

- 🧊 **Triple-Tier Intelligence**:
  - **Tier 1 (Rules)**: Instant matching for known extensions (ISO, DEB, APPIMAGE, etc).
  - **Tier 2 (Fuzzy Keywords)**: Smart overlap matching against your existing folder taxonomy.
  - **Tier 3 (LLM Reasoning)**: High-confidence content analysis for complex files.
- 🚦 **Safety First**: Zero hallucinations. If the confidence is too low, the file is safely **parked** in `.unsorted/` for human review.
- ⚡ **Native Performance**: Written in Go with `fsnotify` for real-time file system events.
- 🪵 **History Tracking**: Full SQLite-backed logging of every decision—know exactly why and where your files moved.
- 🖥️ **Interactive Review**: Use `sortd review` to quickly map unknown files with a simple CLI prompt.

## 🚀 Quick Start

### 1. Installation

Ensure you have [Go](https://go.dev/dl/) installed, then run:

```bash
git clone https://github.com/harsh-sreehari/sortd.git
cd sortd
make install
```

This will build the binary and register the `sortd` systemd service for your user.

### 2. Populate your Index

Tell `sortd` about your existing folders so it can learn your taxonomy:

```bash
sortd index
```

### 3. Start Organising

You can run a manual sorting pass:

```bash
sortd run
```

Or check your history:

```bash
sortd log
```

## ⚙️ Configuration

`sortd` creates a default configuration at `~/.config/sortd/config.toml`.

```toml
[watch]
folders = ["/home/user/Downloads"]

[llm]
host = "http://localhost:1234"
model = "llama-3-8b-instruct"

[behaviour]
confidence_threshold = 0.75
db_path = "~/.local/share/sortd/sortd.db"
```

## 🛠️ Architecture

`sortd` is built with modularity in mind:
- **Watcher**: `fsnotify` event loop.
- **Graph**: Lexical folder indexing and keyword extraction.
- **Pipeline**: Tiered decision orchestration.
- **Mover**: Atomic, cross-device file shifting with collision handling.
- **LLM Context**: Pluggable backend for local inference (LMStudio).

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
