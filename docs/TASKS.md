# sortd — Build Tasks

This file is the primary execution plan for the Antigravity agent. Tasks are ordered by dependency. Complete each task fully before moving to the next. Each task includes acceptance criteria the agent should verify before marking it done.

---

## Stage 1 — Project scaffolding

### Task 1.1 — Initialise Go module

```
go mod init github.com/yourusername/sortd
go mod tidy
```

Create the full directory structure from ARCHITECTURE.md. Create empty `.go` files with just the `package` declaration in each. This gives the compiler a complete picture of the project from the start.

**Acceptance:** `go build ./...` succeeds with no errors (empty packages are fine at this stage).

---

### Task 1.2 — Config system

Implement `internal/config/config.go`.

Define the `Config` struct matching this TOML shape:

```toml
[watch]
folders = ["~/Downloads"]
ignore = []

[llm]
backend = "lmstudio"
host = "http://localhost:1234"
model = "default"

[behaviour]
split_by_type = false
confidence_threshold = 0.75
create_folders = true
log_path = "~/.local/share/sortd/sortd.log"
db_path = "~/.local/share/sortd/sortd.db"
debounce_seconds = 2
```

Requirements:
- Expand `~` to the actual home directory when loading paths
- Provide a `DefaultConfig()` function that returns working defaults for every field
- If config file does not exist, use defaults silently (do not error)
- Load from `~/.config/sortd/config.toml`

**Acceptance:** Write a test that loads a minimal config and verifies defaults fill in missing fields correctly.

---

### Task 1.3 — Storage layer

Implement `internal/store/schema.go` and `internal/store/store.go`.

Create and migrate these SQLite tables on first open:

```sql
CREATE TABLE IF NOT EXISTS sort_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp   TEXT NOT NULL,
    filename    TEXT NOT NULL,
    source      TEXT NOT NULL,
    destination TEXT NOT NULL,
    tier        INTEGER NOT NULL,
    confidence  REAL NOT NULL,
    tags        TEXT,
    action      TEXT NOT NULL,
    corrected   INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS folder_index (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    path     TEXT UNIQUE NOT NULL,
    keywords TEXT NOT NULL,
    depth    INTEGER NOT NULL,
    parent   TEXT
);

CREATE TABLE IF NOT EXISTS affinities (
    tag     TEXT NOT NULL,
    folder  TEXT NOT NULL,
    weight  REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (tag, folder)
);
```

Implement:
- `store.Open(dbPath string) (*Store, error)`
- `store.LogDecision(d Decision) error`
- `store.RecentLog(n int) ([]LogEntry, error)`
- `store.UnsortedFiles() ([]string, error)` — files with action="parked"
- `store.MarkCorrected(id int, newDest string) error`

**Acceptance:** Write a test that opens an in-memory DB (`:memory:`), logs a decision, reads it back, marks it corrected.

---

## Stage 2 — Filesystem watcher

### Task 2.1 — Watcher implementation

Implement `internal/watcher/watcher.go`.

```go
type Watcher struct {
    cfg *config.Config
    Out chan string  // emits ready file paths
}

func New(cfg *config.Config) (*Watcher, error)
func (w *Watcher) Start(ctx context.Context) error
func (w *Watcher) Stop()
```

Requirements:
- Use `fsnotify` to watch all folders in `cfg.Watch.Folders`
- Debounce `CREATE` and `RENAME` events by `cfg.Behaviour.DebounceSeconds`
- Skip files matching these patterns: `*.crdownload`, `*.part`, `*.tmp`, `*.download`, `.*` (hidden)
- Skip the `.unsorted` subfolder
- Emit the full absolute path to `Out` after debounce

**Acceptance:** Manual test — start the watcher against a temp directory, copy a file in, verify the path is emitted after the debounce window and not before.

---

## Stage 3 — Pipeline tiers

### Task 3.1 — Tier 1 rules engine

Implement `internal/pipeline/tier1.go`.

```go
type Rule struct {
    Extensions []string
    Pattern    string  // optional glob on filename
    Destination string  // relative to home
}

func (t *Tier1) Match(path string) (Decision, bool)
```

Built-in rules (hardcoded, not user-configurable yet):

| Extensions | Destination |
|---|---|
| `.AppImage`, `.deb`, `.rpm`, `.flatpak` | `Software/` |
| `.iso` | `Software/` |
| `.exe` | `Software/` |
| `.crdownload`, `.part`, `.tmp` | skip |
| `.torrent` | skip |

Returns `(Decision{Action: "skipped"}, true)` for skip rules.
Returns `(Decision{}, false)` for no match — pass to Tier 2.

**Acceptance:** Unit tests covering each rule. Test that an unknown extension returns false.

---

### Task 3.2 — Folder graph indexer

Implement `internal/graph/graph.go` and `internal/graph/index.go`.

```go
type FolderNode struct {
    Path     string
    Keywords []string
    Depth    int
}

func Crawl(roots []string, ignore []string) ([]FolderNode, error)
func TokenisePath(folderName string) []string
```

`TokenisePath` splits a folder name into lowercase tokens:
- Split on `-`, `_`, space
- Split camelCase: `CompilerDesign` → `[compiler, design]`
- Lowercase all tokens
- Deduplicate

`Crawl` walks the filesystem from each root folder, skipping ignored paths and paths deeper than 6 levels.

Persist the crawl result to the `folder_index` table in SQLite.

**Acceptance:** Unit test `TokenisePath` against: `Compiler-Design`, `compilerDesign`, `sem_6`, `Operating Systems`, `sortd`. Verify expected token lists.

---

### Task 3.3 — Tier 2 fuzzy matcher

Implement `internal/pipeline/tier2.go`.

```go
func (t *Tier2) Match(path string, threshold float64) (Decision, bool)
```

Algorithm:
1. Extract tokens from filename (same `TokenisePath` logic)
2. Load folder nodes from the index
3. For each folder node, compute a score: intersection of filename tokens and folder keywords divided by max of either set (Jaccard similarity)
4. Boost by affinity weight from the `affinities` table
5. Return best match if score >= threshold

**Acceptance:** Integration test with a seeded folder index matching the test case table in SPEC.md. Verify that `Compiler-mod-1.pdf` matches `Academics/Sem6/Compiler Design/` and `cnsp292sqsel231.pdf` falls through.

---

### Task 3.4 — Content peek

Implement `internal/peek/peek.go`, `peek/pdf.go`, `peek/text.go`.

```go
func Peek(path string) (string, error)
// Returns a text description of the file content, or "" if unable.
```

PDF peek:
```go
// run: pdftotext -l 1 {path} -
// take first 800 characters of output
// if pdftotext not available, return ""
```

Text peek:
```go
// read first 2000 bytes
// decode as UTF-8, replace bad bytes
```

Docx peek:
```go
// run: pandoc --to plain {path}
// take first 800 characters
// if pandoc not available, return ""
```

For images: return `""` for now — image description will come from Tier 3 directly when the LLM is given the image bytes.

**Acceptance:** Test PDF peek with a small test PDF. Test text peek with a UTF-8 and a binary file. Verify binary file returns `""` gracefully.

---

### Task 3.5 — LLM backend interface and LM Studio implementation

Implement `internal/llm/backend.go` and `internal/llm/lmstudio.go`.

```go
type Backend interface {
    Tag(ctx context.Context, req TagRequest) (TagResponse, error)
}

type TagRequest struct {
    Filename    string
    Extension   string
    ContentPeek string
    ImageBytes  []byte     // nil for non-image files
    FolderTree  []string   // top 50 folder paths from the index
}

type TagResponse struct {
    Tags        []string
    Destination string
    IsNewFolder bool
    Confidence  float64
    Reasoning   string
}
```

LM Studio backend sends to `POST {host}/v1/chat/completions`. Build the prompt from SPEC.md. Parse the JSON response from the model's text output (strip any markdown code fences before parsing).

For image files, include the image as base64 in the messages array following the OpenAI vision format.

Error handling:
- If the LLM returns malformed JSON, retry once with a stricter prompt
- If the second attempt also fails, return confidence 0.0 (will park the file)
- Log all LLM errors to the sort log with action="error"

**Acceptance:** Integration test against a running LM Studio instance (can be skipped in CI with a build tag). Unit test the JSON parsing with a mock response including edge cases: missing fields, extra fields, malformed JSON.

---

### Task 3.6 — Tier 3 LLM router

Implement `internal/pipeline/tier3.go`.

```go
func (t *Tier3) Match(path string, threshold float64) (Decision, bool)
```

Calls `peek.Peek(path)` then calls the LLM backend. Applies decision logic from SPEC.md. Returns a `Decision` always (never `false`) — worst case is `Action: "parked"`.

**Acceptance:** Test with a mocked LLM backend returning various confidence scores. Verify that high confidence creates a move decision, low confidence creates a park decision.

---

### Task 3.7 — Pipeline orchestrator

Implement `internal/pipeline/pipeline.go`.

```go
type Pipeline struct {
    t1    *Tier1
    t2    *Tier2
    t3    *Tier3
    store *store.Store
    cfg   *config.Config
}

func (p *Pipeline) Process(path string) Decision
```

Runs Tier 1 → Tier 2 → Tier 3 in order. After getting a decision, calls `mover.Move()` for move decisions or creates the `.unsorted` symlink for park decisions. Logs every decision to the store.

**Acceptance:** End-to-end test with a temp directory as the watch target. Seed a folder index. Run Process on files matching the test case table from SPEC.md. Verify final locations.

---

## Stage 4 — Mover

### Task 4.1 — Atomic file mover

Implement `internal/mover/mover.go`.

```go
func Move(src, dst string) error
func Park(src, unsortedDir string) error
```

`Move` requirements:
- If `dst` directory does not exist and `IsNewFolder` is true: `os.MkdirAll`
- If destination file already exists: append `_1`, `_2`, etc. until clear
- Same filesystem: use `os.Rename` (atomic)
- Different filesystem: `io.Copy` → verify file size matches → `os.Remove(src)`
- Never remove `src` before dst is confirmed written

`Park` moves the file into `~/Downloads/.unsorted/` with the same collision-avoidance logic.

**Acceptance:** Test same-filesystem and cross-filesystem moves. Test collision avoidance. Test that a failed copy does not remove the source.

---

## Stage 5 — CLI

### Task 5.1 — CLI entrypoint and daemon commands

Implement `cmd/sortd/main.go` with Cobra.

Implement:
- `sortd daemon start` — starts watcher + pipeline loop, installs systemd service if not installed
- `sortd daemon start --foreground` — runs in foreground (used by systemd)
- `sortd daemon stop` — sends SIGTERM to PID stored in `~/.local/share/sortd/sortd.pid`
- `sortd daemon status` — checks if daemon is running

**Acceptance:** `sortd daemon start` then `sortd daemon status` shows running. `sortd daemon stop` then `sortd daemon status` shows stopped.

---

### Task 5.2 — Log and review commands

Implement:

`sortd log`
- Reads `sort_log` from SQLite
- Pretty-prints last 20 entries by default
- Flags: `--today`, `--n <count>`, `--tier <1|2|3>`

`sortd review`
- Lists files in `.unsorted/` with their filenames
- Flags: `--decide` enters an interactive loop asking where each file should go

`sortd review --decide` interactive loop:
```
Unsorted: photo_scan_0042.jpg
  [1] Enter a destination path
  [2] Skip
  [3] Delete
Choice:
```
After a choice, updates the sort log with `corrected=1` and updates affinity weights.

**Acceptance:** Manually test all flags and the interactive decide loop against a seeded database.

---

### Task 5.3 — Init and index commands

`sortd init`:
1. Create `~/.config/sortd/config.toml` if it does not exist
2. Create `~/.local/share/sortd/` directory
3. Open/migrate the SQLite database
4. Run first folder graph crawl
5. Install systemd user service
6. Print a short summary of what was found

`sortd index`:
- Re-crawls the folder tree and updates `folder_index` in SQLite
- Prints count of folders indexed

**Acceptance:** Run `sortd init` on a clean system. Verify all directories and files created. Run `sortd index` and verify output.

---

## Stage 6 — Polish

### Task 6.1 — Graceful shutdown

The daemon should handle `SIGTERM` and `SIGINT` gracefully:
- Stop accepting new files from the watcher
- Finish processing any file currently in the pipeline
- Close the SQLite connection cleanly
- Exit with code 0

**Acceptance:** Start daemon, send SIGTERM, verify clean exit and no database corruption.

---

### Task 6.2 — First-run experience

If `sortd` is run with no subcommand and no config exists, print a friendly getting-started message:

```
sortd is not initialised yet.
Run: sortd init

This will set up your config, index your folders, and start the daemon.
```

If config exists but daemon is not running, suggest `sortd daemon start`.

---

### Task 6.3 — README final pass

After all above tasks pass, update README.md with:
- Accurate installation instructions
- Actual tested CLI commands
- Known limitations section
- Link to SPEC.md and ARCHITECTURE.md

---

## Testing strategy

| Layer | Approach |
|---|---|
| Tier 1 rules | Pure unit tests, no FS needed |
| Tier 2 matching | Unit tests with seeded in-memory SQLite index |
| Tier 3 LLM | Unit tests with mock backend, integration tests behind build tag |
| Peek | Unit tests with small fixture files |
| Mover | Integration tests using `os.TempDir()` |
| CLI | Manual testing + smoke tests |

Run all tests: `go test ./...`
Run without LLM integration tests: `go test -tags nollm ./...`

---

## Build and release

```bash
# Development build
go build -o sortd ./cmd/sortd

# Release build (static binary)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o sortd-linux-amd64 \
  ./cmd/sortd
```

Note: if using the `llamacpp` backend, `CGO_ENABLED=1` is required. The `lmstudio` backend (recommended and default) works with `CGO_ENABLED=0`.
