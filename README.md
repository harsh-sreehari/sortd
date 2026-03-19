# sortd

`sortd` is a **silent, context-aware file organizer daemon** for Linux. It treats "context" as the primary routing signal, using a three-tier pipeline (Rules → Fuzzy Semantic Folder Matching → LLM Inference) to organically sort your files without you having to define tedious rules.

It runs locally with tools like LM Studio, ensuring full privacy and keeping your personal data out of the cloud.

## Features

- **Three-Tier Sorting Pipeline:** 
  1. **Rules:** Instantly handles strict extensions like `.deb` and skips temporary `.crdownload` files.
  2. **Fuzzy:** Indexes your folder names and matches files based on Jaccard similarity.
  3. **LLM:** Reads PDF text, docs, and integrates an open-source model (via LM Studio) to infer destinations. 
- **Atomic Operations:** Uses `os.Rename` natively and seamlessly handles cross-device links and filename collisions (e.g., `file_1.zip`).
- **Fully Offline:** Bring your own Local LLM. Privacy-first structure.
- **Daemon First:** Implemented specifically to run as a reliable systemd user service.

## Installation

1. Ensure Go is installed on your Linux machine.
2. Build and install the binary:
   ```bash
   go install github.com/harsh-sreehari/sortd/cmd/sortd@latest
   ```

### Setup systemd (optional but recommended)

To run `sortd` in the background continuously:
```bash
cp sortd.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable sortd --now
```

*(Note: Ensure your `systemd` user units can resolve `~/go/bin/sortd`. Adjust `ExecStart` in `sortd.service` if you installed the binary elsewhere).*

## Configuration

On its first run, `sortd` will automatically scaffold a default configuration at `~/.config/sortd/config.toml` that monitors your `~/Downloads` directory.

You can customize this TOML:
```toml
[watch]
folders = ["/home/user/Downloads"]

[llm]
host = "http://localhost:1234"
model = "lmstudio-community/Meta-Llama-3-8B-Instruct-GGUF"

[behaviour]
confidence_threshold = 0.70
```

## CLI Usage

```
sortd run        # Runs a one-time pass over all watched folders
sortd log        # View the history of sorted files
sortd review     # Interactive mode for deciding destinations of files parked in `.unsorted/`
sortd daemon     # Commands to start, stop, or check daemon status
```

## How It Handles Uncertainty

When `sortd` lacks confidence about a file or encounters an LLM error, it **parks** the file in an `.unsorted/` directory inside your root watched folder. This guarantees that nothing gets misplaced due to a hallucination. Use `sortd review` to process parked files manually.
