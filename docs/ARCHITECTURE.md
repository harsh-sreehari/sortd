# sortd — Architecture

## Repository layout

```
sortd/
├── cmd/
│   └── sortd/
│       └── main.go          # entry point, cobra CLI setup
├── internal/
│   ├── watcher/
│   │   └── watcher.go       # inotify via fsnotify, debounce logic
│   ├── pipeline/
│   │   ├── pipeline.go      # orchestrates tier 1 → 2 → 3
│   │   ├── tier1.go         # rules engine
│   │   ├── tier2.go         # folder graph fuzzy matching
│   │   └── tier3.go         # LLM routing
│   ├── graph/
│   │   ├── graph.go         # folder tree crawler and indexer
│   │   └── index.go         # SQLite persistence for the index
│   ├── peek/
│   │   ├── peek.go          # dispatcher: routes files to correct peek strategy
│   │   ├── pdf.go           # pdftotext first page extraction
│   │   ├── text.go          # plain text / markdown / docx first 300 tokens
│   │   └── image.go         # passes image bytes to vision LLM
│   ├── llm/
│   │   ├── backend.go       # LLMBackend interface definition
│   │   ├── lmstudio.go      # HTTP backend for LM Studio
│   │   └── llamacpp.go      # CGo backend for embedded llama.cpp
│   ├── mover/
│   │   └── mover.go         # atomic file move logic
│   ├── store/
│   │   ├── store.go         # SQLite wrapper
│   │   └── schema.go        # table definitions
│   └── config/
│       └── config.go        # TOML config loading and defaults
├── sortd.service             # systemd user unit file
├── go.mod
├── go.sum
├── README.md
├── SPEC.md
├── ARCHITECTURE.md
└── TASKS.md
```

---

## Component breakdown

### `cmd/sortd/main.go`

Cobra CLI root. Registers subcommands:

```
sortd
├── daemon
│   ├── start      starts watcher as background process or foreground (--foreground)
│   ├── stop       sends SIGTERM to running daemon
│   └── status     checks if daemon is running
├── log            reads sort_log from SQLite, pretty prints
├── review         lists files in .unsorted/, interactive resolve
├── run            manual one-shot sort pass on watched folders
└── index          re-crawls folder tree and rebuilds index
```

---

### `internal/watcher`

Wraps `fsnotify`. Watches all folders from config. Applies debounce (2s default) on `CREATE` and `RENAME` events before emitting to the pipeline.

```go
type Watcher struct {
    folders  []string
    debounce time.Duration
    out      chan string   // emits absolute file paths ready for processing
}
```

Filters out:
- Partial download extensions (`.crdownload`, `.part`, `.tmp`, `.download`)
- Hidden files starting with `.`
- Files being written by the watcher's own moves (tracked by path)

---

### `internal/pipeline`

Receives a file path from the watcher. Runs it through tiers in order. Returns a `Decision`.

```go
type Decision struct {
    File        string
    Destination string
    Tags        []string
    Tier        int
    Confidence  float64
    IsNewFolder bool
    Action      string   // "moved" | "parked" | "skipped"
}

func (p *Pipeline) Process(path string) Decision
```

**Tier 1 (`tier1.go`):**
- Loads rules from a static table + any user-defined rules in config
- Returns immediately on match with confidence 1.0
- Returns confidence 0.0 to fall through to Tier 2

**Tier 2 (`tier2.go`):**
- Loads folder graph from the index
- Extracts tokens from filename
- Scores each folder node
- Returns top match if above threshold, else passes to Tier 3

**Tier 3 (`tier3.go`):**
- Calls `peek.Peek(path)` to get content preview
- Builds prompt with filename + peek + abbreviated folder tree
- Calls LLM backend
- Parses JSON response
- Returns decision

---

### `internal/graph`

Crawls the filesystem and builds a semantic map of the user's folder structure.

```go
type FolderNode struct {
    Path     string
    Keywords []string   // extracted from folder name
    Depth    int
    Children []*FolderNode
    Affinity map[string]float64  // tag → learned weight from sort history
}
```

**Crawl ignore list (hardcoded + user config):**
```
.git, node_modules, __pycache__, .cargo, .rustup,
vendor, .local/share/Steam, snap, flatpak,
Downloads/.unsorted, .config, .cache
```

Index is stored in SQLite and only re-crawled when `sortd index` is called or when the daemon detects that a new folder was created (via inotify on parent directories).

---

### `internal/peek`

Dispatcher that routes a file to the correct peek strategy based on extension.

```go
func Peek(path string) (string, error)
// returns text description of file content, or "" if unable to peek
```

| Extension group | Strategy |
|---|---|
| `.pdf` | run `pdftotext -l 1 {path} -` and take first 800 chars |
| `.txt .md .rst` | read first 2000 bytes |
| `.docx` | extract raw text via `pandoc -t plain` |
| `.jpg .jpeg .png .webp` | encode to base64, send to vision LLM endpoint |
| `.mp4 .mkv .mp3 .wav` | return "" (no peek) |
| everything else | return "" |

For the image path, `peek/image.go` makes a direct call to the LLM backend's vision endpoint and returns a text description, not raw image bytes. The description is then passed to Tier 3 like any other content peek.

---

### `internal/llm`

**Interface:**

```go
type Backend interface {
    Complete(ctx context.Context, prompt string) (string, error)
    DescribeImage(ctx context.Context, imageBytes []byte) (string, error)
}
```

**LM Studio backend (`lmstudio.go`):**

Calls `POST http://localhost:1234/v1/chat/completions` with the OpenAI-compatible payload. Handles vision by including image as base64 in the messages array when the model supports it.

Config required:
```toml
[llm]
backend = "lmstudio"
host = "http://localhost:1234"
model = "your-model-name"   # must match what's loaded in LM Studio
```

**llama.cpp backend (`llamacpp.go`):**

CGo bindings to llama.cpp. Loads a `.gguf` model file at daemon startup, keeps it in memory. Vision via LLaVA-style multimodal models.

Config required:
```toml
[llm]
backend = "llamacpp"
model = "~/.local/share/sortd/model.gguf"
```

---

### `internal/mover`

Handles the actual filesystem move. Designed to never lose data.

```go
func Move(src, dst string) error
```

**Logic:**
1. Check `dst` doesn't already exist (if it does, append `_1`, `_2`, etc.)
2. If src and dst are on the same filesystem: `os.Rename()` (atomic)
3. If different filesystems: `io.Copy` + verify hash + `os.Remove(src)`
4. Write to sort log only after successful move

If `IsNewFolder` is true, `mover.Move` calls `os.MkdirAll` first.

---

### `internal/store`

SQLite wrapper using `modernc.org/sqlite` (pure Go, no CGo required for storage layer).

Tables:
- `sort_log` — every file decision with full metadata
- `folder_index` — cached folder graph nodes and keywords
- `affinities` — learned tag → folder weights updated from user corrections

---

### `internal/config`

TOML config loaded from `~/.config/sortd/config.toml`. Provides typed struct with defaults for every field so the tool works out of the box with an empty or missing config.

---

## Data flow diagram

```
Downloads/          fsnotify (inotify)
new file ──────────────────────────────► Watcher
                                             │
                                    debounce 2s
                                             │
                                             ▼
                                         Pipeline
                                             │
                                    ┌────────▼────────┐
                                    │   Tier 1         │
                                    │   rules engine   │
                                    └────────┬─────────┘
                                      match? │ no match
                                             ▼
                                    ┌────────▼────────┐
                                    │   Tier 2         │
                                    │   graph fuzzy    │
                                    └────────┬─────────┘
                                      match? │ no match
                                             ▼
                                    ┌────────▼────────┐
                                    │   peek           │
                                    │   (pdf/img/text) │
                                    └────────┬─────────┘
                                             │
                                    ┌────────▼────────┐
                                    │   Tier 3         │
                                    │   LLM via        │
                                    │   LM Studio      │
                                    └────────┬─────────┘
                             confident?      │       uncertain
                        ┌───────────────────┘              │
                        ▼                                  ▼
                    Mover                            .unsorted/
                  (atomic move)                    (review queue)
                        │
                        ▼
                    sort_log
                    (SQLite)
```

---

## External dependencies

| Package | Purpose |
|---|---|
| `github.com/fsnotify/fsnotify` | inotify wrapper |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/BurntSushi/toml` | config parsing |
| `modernc.org/sqlite` | pure Go SQLite (no CGo) |
| `github.com/sahilm/fuzzy` | fuzzy string matching for Tier 2 |
| `github.com/go-llama/llama.cpp` | llamacpp backend (CGo, optional) |

System dependencies:
- `pdftotext` (from `poppler-utils`) — PDF peek
- `pandoc` — docx peek (optional, degrades gracefully if missing)
- LM Studio — LLM backend (optional if using llamacpp backend)
