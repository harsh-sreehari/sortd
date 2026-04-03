package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/mover"
	"github.com/harsh-sreehari/sortd/internal/peek"
	"github.com/harsh-sreehari/sortd/internal/pipeline"
	"github.com/harsh-sreehari/sortd/internal/store"
	"github.com/harsh-sreehari/sortd/internal/watcher"
)

// serviceTemplate is the systemd user service unit embedded directly in the binary.
// This ensures sortd init works from any install location without needing the
// sortd.service file present on disk.
const serviceTemplate = `[Unit]
Description=sortd context-aware file organizer daemon
After=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/sortd daemon start
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

func getPidPath() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".local/share/sortd/sortd.pid")
	os.MkdirAll(filepath.Dir(path), 0755)
	return path
}

func initPipeline() (*config.Config, *store.Store, *pipeline.Pipeline, error) {
	cfg, err := config.LoadConfig("~/.config/sortd/config.toml") // Simplified
	if err != nil {
		return nil, nil, nil, err
	}

	st, err := store.Open(cfg.Behaviour.DBPath)
	if err != nil {
		return nil, nil, nil, err
	}

	gr := &graph.Graph{Store: st}
	llmBackend := &llm.LMStudioBackend{
		Host:  cfg.LLM.Host,
		Model: cfg.LLM.Model,
	}
	mv := mover.New()

	pipe := pipeline.New(cfg, st, gr, llmBackend, mv)

	// B6: Determine AllowedRoots from config or derive from real crawl-root directories.
	allowedRoots := cfg.Behaviour.AllowedRoots
	if len(allowedRoots) == 0 {
		// Auto-derive: check which of the default XDG-ish roots actually exist on disk.
		// This respects the user's actual filesystem rather than assuming a fixed layout.
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
			filepath.Join(home, "Pictures"),
			filepath.Join(home, "Videos"),
			filepath.Join(home, "Music"),
		}
		for _, r := range candidates {
			if info, err := os.Stat(r); err == nil && info.IsDir() {
				// Store as bare name with trailing slash to match AllowedRoots format
				allowedRoots = append(allowedRoots, filepath.Base(r)+"/")
			}
		}
	}
	pipe.SetAllowedRoots(allowedRoots)

	return cfg, st, pipe, nil
}

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "sortd",
	Short:   "sortd is a context-aware file organiser daemon",
	Version: version,
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background watcher",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background watcher",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPidPath()
		if _, err := os.Stat(pidPath); err == nil {
			// Check if process still exists
			data, _ := os.ReadFile(pidPath)
			var pid int
			fmt.Sscanf(string(data), "%d", &pid)
			process, err := os.FindProcess(pid)
			if err == nil && process.Signal(syscall.Signal(0)) == nil {
				log.Fatalf("Daemon already running with PID %d", pid)
			}
		}

		if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
			log.Fatalf("Failed to write PID file: %v", err)
		}
		defer os.Remove(pidPath)

		cfg, st, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		w, err := watcher.New(cfg)
		if err != nil {
			log.Fatalf("Watcher failed: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := w.Start(ctx); err != nil {
			log.Fatalf("Failed to start watcher: %v", err)
		}

		fmt.Println("sortd daemon is actively watching...")

		// Handle graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			for path := range w.Out {
				pipe.Process(path)
			}
		}()

		<-sigCh
		fmt.Println("Shutting down sortd daemon...")
		w.Stop()
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background watcher",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPidPath()
		data, err := os.ReadFile(pidPath)
		if err != nil {
			fmt.Println("Daemon is NOT running (no PID file).")
			return
		}

		var pid int
		fmt.Sscanf(string(data), "%d", &pid)
		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println("Process not found. Cleaning up stale PID file.")
			os.Remove(pidPath)
			return
		}

		fmt.Printf("Stopping daemon (PID %d)...\n", pid)
		process.Signal(syscall.SIGTERM)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the background watcher status",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPidPath()
		data, err := os.ReadFile(pidPath)
		if err != nil {
			fmt.Println("Status: Stopped")
			return
		}

		var pid int
		fmt.Sscanf(string(data), "%d", &pid)
		process, err := os.FindProcess(pid)
		if err == nil && process.Signal(syscall.Signal(0)) == nil {
			fmt.Printf("Status: Running (PID %d)\n", pid)
		} else {
			fmt.Println("Status: Stale (PID file exists but process is dead)")
		}
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manually trigger a sort pass on watched folders",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, st, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		moved, parked, skipped := 0, 0, 0

		for _, folder := range cfg.Watch.Folders {
			filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				// Skip hidden and .unsorted
				if strings.Contains(path, "/.unsorted") || strings.HasPrefix(filepath.Base(path), ".") {
					return nil
				}

				decision := pipe.Process(path)
				switch decision.Action {
				case "moved":
					moved++
				case "parked":
					parked++
				case "skipped":
					skipped++
				}
				return nil
			})
		}

		fmt.Printf("Run Complete: Moved: %d, Parked: %d, Skipped: %d\n", moved, parked, skipped)
	},
}

var (
	logTier   string
	logAction string
	logParked bool
	logToday  bool
	logTag    string
	logLimit  int
	logVerbose bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent sort history",
	Run: func(cmd *cobra.Command, args []string) {
		_, st, _, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		filters := make(map[string]string)
		if logTier != "" {
			filters["tier"] = logTier
		}
		if logAction != "" {
			filters["action"] = logAction
		}
		if logParked {
			filters["action"] = "parked"
		}
		if logToday {
			filters["today"] = "true"
		}
		if logTag != "" {
			filters["tag"] = logTag
		}

		logs, err := st.SearchLog(logLimit, filters)
		if err != nil {
			log.Fatalf("Failed to fetch logs: %v", err)
		}

		if len(logs) == 0 {
			fmt.Println("No matching activity.")
			return
		}

		// ANSI Colors
		reset := "\033[0m"
		bold := "\033[1m"
		green := "\033[32m"
		yellow := "\033[33m"
		cyan := "\033[36m"
		gray := "\033[90m"

		if logVerbose {
			fmt.Printf("Displaying %d detailed logs:\n\n", len(logs))
			for _, l := range logs {
				base := l.OriginalFilename
				if base == "" {
					base = filepath.Base(l.Filename)
				}

				tagsStr := ""
				var tags []string
				if err := json.Unmarshal([]byte(l.Tags), &tags); err == nil && len(tags) > 0 {
					tagsStr = strings.Join(tags, ", ")
				}

				color := reset
				switch l.Action {
				case "moved":
					color = green
				case "parked":
					color = yellow
				case "skipped":
					color = gray
				}

				fmt.Printf("%s%s%s %s%s%s -> %s\n", gray, l.Timestamp[:16], reset, color, l.Action, reset, l.Destination)
				fmt.Printf("      File  : %s\n", base)
				if tagsStr != "" {
					fmt.Printf("      Tags  : %s[%s]%s\n", gray, tagsStr, reset)
				}
				fmt.Printf("      Tier  : %d\n", l.Tier)
				if l.Reasoning != "" {
					fmt.Printf("      Reason: %s%s%s\n", cyan, l.Reasoning, reset)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("%s%-20s | %-10s | %-40s | %-8s | %s%s\n", bold, "Timestamp", "Action", "Filename", "Tier", "Destination", reset)
			fmt.Println(strings.Repeat("-", 120))
			for _, l := range logs {
				base := l.OriginalFilename
				if base == "" {
					base = filepath.Base(l.Filename)
				}
				if len(base) > 38 {
					base = base[:35] + "..."
				}

				// Format tags
				tagsStr := ""
				var tags []string
				if err := json.Unmarshal([]byte(l.Tags), &tags); err == nil && len(tags) > 0 {
					tagsStr = fmt.Sprintf(" %s[%s]%s", gray, strings.Join(tags, ","), reset)
				}

				color := reset
				switch l.Action {
				case "moved":
					color = green
				case "parked":
					color = yellow
				case "skipped":
					color = gray
				}

				fmt.Printf("%s%-20s%s | %s%-10s%s | %-40s | %sTier %-2d%s | %s%s\n",
					gray, l.Timestamp, reset,
					color, l.Action, reset,
					base+tagsStr,
					cyan, l.Tier, reset,
					l.Destination, reset)
			}
		}
	},
}

var (
	findTag   string
	findSince string
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search sort history for a specific file or destination",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, st, _, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		filters := map[string]string{"query": args[0]}
		if findTag != "" {
			filters["tag"] = findTag
		}
		if findSince != "" {
			filters["since"] = findSince
		}
		logs, err := st.SearchLog(50, filters)
		if err != nil {
			log.Fatalf("Search failed: %v", err)
		}

		if len(logs) == 0 {
			fmt.Printf("No results found for '%s'\n", args[0])
			return
		}

		// Reuse colored output logic (simplified or extract to helper if needed)
		fmt.Printf("Found %d results for '%s':\n\n", len(logs), args[0])
		for _, l := range logs {
			fmt.Printf("\033[90m%s\033[0m \033[32m%s\033[0m -> %s\n", l.Timestamp[:16], l.Action, l.Destination)
			fmt.Printf("      File: %s\n", l.OriginalFilename)
			if l.Reasoning != "" {
				fmt.Printf("      Why : \033[36m%s\033[0m\n", l.Reasoning)
			}
			fmt.Println()
		}
	},
}

var tagsFolder string

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Show frequency of tags found in your files",
	Run: func(cmd *cobra.Command, args []string) {
		_, st, _, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		stats, err := st.AggregatedTags(tagsFolder)
		if err != nil {
			log.Fatalf("Failed to aggregate tags: %v", err)
		}

		if len(stats) == 0 {
			fmt.Println("No tags found yet.")
			return
		}

		if tagsFolder != "" {
			fmt.Printf("\n🏷️  Tag Analytics for %s\n", tagsFolder)
		} else {
			fmt.Println("\n🏷️  Tag Analytics (Global)")
		}
		fmt.Println(strings.Repeat("-", 30))
		for _, s := range stats {
			// Basic text bar representation
			bar := ""
			for i := 0; i < s.Count && i < 40; i++ {
				bar += "■"
			}
			fmt.Printf("%15s | %-5d %s\n", s.Tag, s.Count, bar)
		}
	},
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Interactive resolving of parked files with NLP support",
	Run: func(cmd *cobra.Command, args []string) {
		_, st, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		parked, err := st.UnsortedFiles()
		if err != nil {
			log.Fatalf("Failed to fetch unsorted files: %v", err)
		}

		if len(parked) == 0 {
			fmt.Println("No parked files to review! Everything is sorted. 🎉")
			return
		}

		fmt.Printf("Found %d files needing review.\n", len(parked))
		scanner := bufio.NewScanner(os.Stdin)

		folders, _ := pipe.Graph.ListFolders()
		folderPaths := make([]string, len(folders))
		for i, f := range folders {
			folderPaths[i] = f.Path
		}

		for i, entry := range parked {
			fmt.Printf("\n--- [%d/%d] ---\n", i+1, len(parked))
			fmt.Printf("📄 File: \033[1m%s\033[0m\n", entry.OriginalFilename)
			if entry.Reasoning != "" {
				fmt.Printf("🤖 System said: \033[90m%s\033[0m\n", entry.Reasoning)
			}
			fmt.Print("🤔 What is this? [skip/path/description]: ")

			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())

			if input == "" || input == "skip" {
				fmt.Println("⏭️  Skipped.")
				continue
			}

			var dest string

			// 1. Check if input is a valid direct path (relative to home or partial)
			if _, err := os.Stat(input); err == nil {
				dest = input
			} else {
				// 2. Try Fuzzy Description Match (Tier 2 logic)
				if matched, ok := pipeline.MatchDescription(input, folders); ok {
					fmt.Printf("💡 Fuzzy match found: \033[36m%s\033[0m. Use this? [Y/n]: ", matched)
					scanner.Scan()
					if strings.ToLower(scanner.Text()) != "n" {
						dest = matched
					}
				}

				// 3. Fallback to LLM
				if dest == "" {
					fmt.Println("🧠 Asking LLM for best destination...")
					resp, err := pipe.LLM.ResolveReview(input, entry.OriginalFilename, folderPaths)
					if err == nil && resp.Confidence > 0.5 {
						fmt.Printf("🤖 LLM suggests: \033[36m%s\033[0m (%s). Use this? [Y/n]: ", resp.Destination, resp.Reasoning)
						scanner.Scan()
						if strings.ToLower(scanner.Text()) != "n" {
							dest = resp.Destination
						}
					}
				}
			}

			if dest == "" {
				fmt.Println("❌ Could not determine destination. Skipped.")
				continue
			}

			// Use Destination if it was successfully moved/parked, otherwise Filename
			srcPath := entry.Filename
			if entry.Action == "parked" || entry.Action == "moved" {
				srcPath = entry.Destination
			}

			// B4: Check the file still exists before attempting the move.
			// If it was manually moved out of .unsorted/ we mark it corrected
			// so it stops appearing in future reviews.
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				fmt.Printf("⚠️  File no longer at %s — may have been moved manually. Marking as resolved.\n", srcPath)
				st.MarkCorrected(entry.ID, srcPath, "")
				continue
			}

			finalPath, err := pipe.Mover.Move(srcPath, dest)
			if err != nil {
				fmt.Printf("❌ Failed to move: %v\n", err)
			} else {
				fmt.Printf("✅ Moved to: %s\n", finalPath)
				// Update affinities for the folder we moved to
				st.MarkCorrected(entry.ID, finalPath, dest)
			}
		}

		fmt.Println("\nReview complete.")
	},
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Re-crawl the folder tree and rebuild the index",
	Run: func(cmd *cobra.Command, args []string) {
		runIndex()
	},
}

func runIndex() {
	_, _, pipe, err := initPipeline()
	if err != nil {
		log.Fatalf("Init failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	roots := []string{
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Pictures"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Music"),
	}
	
	fmt.Printf("Re-indexing your folders in: %v...\n", roots)
	err = pipe.Graph.Crawl(roots, []string{"/node_modules", "/.git", "/.unsorted", "/.local", "/.cache", "/.gemini", "/.agent"})
	if err != nil {
		log.Fatalf("Crawl failed: %v", err)
	}
	fmt.Println("Indexing complete.")
}

var undoCmd = &cobra.Command{
	Use:   "undo [n]",
	Short: "Undo the last n file moves (default 1)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		n := 1
		if len(args) > 0 {
			var err error
			n, err = strconv.Atoi(args[0])
			if err != nil || n < 1 {
				log.Fatalf("Invalid number of moves: %v", args[0])
			}
		}

		_, st, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		moves, err := st.GetUndoableMoves(n)
		if err != nil {
			log.Fatalf("Failed to fetch recent moves: %v", err)
		}

		if len(moves) == 0 {
			fmt.Println("No recent moves to undo.")
			return
		}

		fmt.Printf("⏪ Undoing last %d move(s)...\n", len(moves))
		
		successCount := 0
		for _, m := range moves {
			if _, err := os.Stat(m.Destination); os.IsNotExist(err) {
				fmt.Printf("⚠️  Skipping %s (file no longer at destination)\n", filepath.Base(m.Destination))
				continue
			}

			// Reverse the move: destination back to source dir
			srcDir := filepath.Dir(m.Source)
			
			// If source directory doesn't exist anymore, create it
			os.MkdirAll(srcDir, 0755)

			finalPath, err := pipe.Mover.Move(m.Destination, srcDir)
			if err != nil {
				fmt.Printf("❌ Failed to move %s back: %v\n", filepath.Base(m.Destination), err)
				continue
			}

			// If the original name differs, try to rename it back
			if m.OriginalFilename != "" && filepath.Base(finalPath) != m.OriginalFilename {
				origPath := filepath.Join(srcDir, m.OriginalFilename)
				if _, err := os.Stat(origPath); os.IsNotExist(err) {
					os.Rename(finalPath, origPath)
					finalPath = origPath
				}
			}

			// Delete from log
			st.DeleteLogEntry(m.ID)
			fmt.Printf("✅ Restored %s to %s\n", filepath.Base(m.Destination), srcDir)
			successCount++
		}
		fmt.Printf("\nDone! Successfully undid %d file(s).\n", successCount)
	},
}

var renameBatch bool

var renameCmd = &cobra.Command{
	Use:   "rename [path]",
	Short: "Rename a file (or recursively rename a batch of files) using AI-suggested context-rich names",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		absPath, _ := filepath.Abs(path)
		info, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			log.Fatalf("Path not found: %s", path)
		}

		if renameBatch && !info.IsDir() {
			log.Fatalf("--batch requires a directory path")
		} else if !renameBatch && info.IsDir() {
			log.Fatalf("Path %s is a directory. Use --batch to rename files inside it.", absPath)
		}

		_, _, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Pipeline init failed: %v", err)
		}

		// Build list of target files
		var targets []string
		if renameBatch {
			entries, err := os.ReadDir(absPath)
			if err != nil {
				log.Fatalf("Failed to read directory: %v", err)
			}
			for _, e := range entries {
				if !e.IsDir() {
					targets = append(targets, filepath.Join(absPath, e.Name()))
				}
			}
			if len(targets) == 0 {
				fmt.Println("No files found in directory to rename.")
				return
			}
			fmt.Printf("📦 Found %d files for batch rename.\n\n", len(targets))
		} else {
			targets = append(targets, absPath)
		}

		type proposal struct {
			src  string
			dest string
		}
		var proposals []proposal
		scanner := bufio.NewScanner(os.Stdin)

		for _, fileTarget := range targets {
			content := peek.PeekDispatcher(fileTarget, pipe.LLM)
			fmt.Printf("🧠 Analyzing \033[1m%s\033[0m...\n", filepath.Base(fileTarget))
			
			newName, err := pipe.LLM.SuggestRename(filepath.Base(fileTarget), content)
			if err != nil {
				fmt.Printf("❌ LLM failed for %s: %v\n", filepath.Base(fileTarget), err)
				continue
			}

			if renameBatch {
				fmt.Printf("💡 Suggestion: \033[36m%s\033[0m\n", newName)
				fmt.Println(strings.Repeat("-", 40))
				
				// Keep track of proposals to apply later in batch
				destPath := filepath.Join(filepath.Dir(fileTarget), newName)
				proposals = append(proposals, proposal{src: fileTarget, dest: destPath})
			} else {
				// Interactive mode for a single file
				for {
					fmt.Printf("💡 Suggestion: \033[36m%s\033[0m\n", newName)
					fmt.Print("🤔 Apply rename? [Y/n/edit]: ")
					
					if !scanner.Scan() {
						return
					}
					input := strings.ToLower(strings.TrimSpace(scanner.Text()))

					if input == "n" {
						fmt.Println("⏭️  Skipped.")
						return
					}

					if input == "edit" {
						fmt.Print("📝 Enter new name: ")
						if scanner.Scan() {
							newName = scanner.Text()
						}
					}

					destPath := filepath.Join(filepath.Dir(fileTarget), newName)
					if _, err := os.Stat(destPath); err == nil && destPath != fileTarget {
						fmt.Printf("⚠️  File '%s' already exists! Asking AI for a variation...\n", newName)
						newName, _ = pipe.LLM.SuggestRename(newName, "The previous suggestion already exists in the folder. Provide a DIFFERENT descriptive name.")
						continue
					}

					finalPath, err := pipe.Mover.Move(fileTarget, destPath)
					if err != nil {
						fmt.Printf("❌ Failed to rename: %v\n", err)
						return
					}
					fmt.Printf("✅ Renamed to: \033[32m%s\033[0m\n", filepath.Base(finalPath))
					break
				}
			}
		}

		if renameBatch && len(proposals) > 0 {
			fmt.Printf("\n🚀 Apply %d renames? [y/N]: ", len(proposals))
			if scanner.Scan() {
				input := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if input == "y" || input == "yes" {
					successCount := 0
					for _, p := range proposals {
						// Simple existence check avoiding exact overwrites during batch
						if _, err := os.Stat(p.dest); err == nil && p.dest != p.src {
							fmt.Printf("⚠️  Skipping %s (destination already exists)\n", filepath.Base(p.src))
							continue
						}
						
						finalPath, err := pipe.Mover.Move(p.src, p.dest)
						if err != nil {
							fmt.Printf("❌ Failed to rename %s: %v\n", filepath.Base(p.src), err)
							continue
						}
						fmt.Printf("✅ Renamed: \033[32m%s\033[0m\n", filepath.Base(finalPath))
						successCount++
					}
					fmt.Printf("\nDone! Successfully renamed %d file(s).\n", successCount)
				} else {
					fmt.Println("⏭️  Batch rename aborted.")
				}
			}
		}
	},
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale records for files that no longer exist",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".config/sortd/config.toml")
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		st, err := store.Open(cfg.Behaviour.DBPath)
		if err != nil {
			log.Fatalf("Failed to open DB: %v", err)
		}
		defer st.Close()

		fmt.Println("🧹 Pruning stale records...")
		prunedIndex, prunedLog, err := st.Prune(cfg.Watch.Folders)
		if err != nil {
			log.Fatalf("Prune failed: %v", err)
		}

		fmt.Printf("✅ Pruned %d folder index entries.\n", prunedIndex)
		fmt.Printf("✅ Pruned %d log entries.\n", prunedLog)
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sortd configuration and install systemd service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Initializing sortd...")

		// 1. Scaffold Config & Dirs
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".config/sortd/config.toml")
		_, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Failed to load/create config: %v", err)
		}

		shareDir := filepath.Join(home, ".local/share/sortd")
		os.MkdirAll(shareDir, 0755)
		fmt.Printf("✅ Folders and configuration prepared at %s\n", configPath)

		// 2. Perform initial index
		runIndex()

		// 3. Install systemd service (content is embedded in the binary at compile time)
		dest := filepath.Join(home, ".config/systemd/user/sortd.service")
		os.MkdirAll(filepath.Dir(dest), 0755)

		// Resolve the actual binary path; fall back to the XDG-conventional location.
		exePath, err := os.Executable()
		if err != nil || exePath == "" {
			exePath = filepath.Join(home, ".local/bin/sortd")
		}

		// Replace the template ExecStart placeholder with the resolved absolute path
		// so systemd does not need to expand %h at runtime.
		serviceContent := strings.Replace(
			serviceTemplate,
			"ExecStart=%h/.local/bin/sortd daemon start",
			"ExecStart="+exePath+" daemon start",
			1,
		)

		if err := os.WriteFile(dest, []byte(serviceContent), 0644); err != nil {
			log.Fatalf("Failed to write service file: %v", err)
		}

		fmt.Printf("✅ Systemd user service installed to %s\n", dest)
		fmt.Println("\nTo start the daemon, run:")
		fmt.Println("  systemctl --user daemon-reload")
		fmt.Println("  systemctl --user enable --now sortd")

		fmt.Println("\nInitialization complete! sortd is ready to go.")
	},
}

func init() {
	logCmd.Flags().StringVar(&logTier, "tier", "", "Filter by tier (1, 2, 3)")
	logCmd.Flags().StringVar(&logAction, "action", "", "Filter by action (moved, parked, skipped)")
	logCmd.Flags().BoolVar(&logParked, "parked", false, "Shortcut for --action=parked")
	logCmd.Flags().BoolVar(&logToday, "today", false, "Show only today's logs")
	logCmd.Flags().StringVar(&logTag, "tag", "", "Filter by tag")
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Number of logs to show")
	logCmd.Flags().BoolVar(&logVerbose, "verbose", false, "Show detailed reasoning from LLM")

	findCmd.Flags().StringVar(&findTag, "tag", "", "Filter results by specific tag")
	findCmd.Flags().StringVar(&findSince, "since", "", "Filter results since a duration (e.g. 24h, 7d)")
	tagsCmd.Flags().StringVar(&tagsFolder, "folder", "", "Show tags only for this destination folder")
	renameCmd.Flags().BoolVar(&renameBatch, "batch", false, "Rename all files in the given directory")

	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd, logCmd, reviewCmd, runCmd, indexCmd, initCmd, findCmd, tagsCmd, renameCmd, pruneCmd, undoCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
