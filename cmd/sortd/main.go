package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/harsh-sreehari/sortd/internal/config"
	"github.com/harsh-sreehari/sortd/internal/graph"
	"github.com/harsh-sreehari/sortd/internal/llm"
	"github.com/harsh-sreehari/sortd/internal/mover"
	"github.com/harsh-sreehari/sortd/internal/pipeline"
	"github.com/harsh-sreehari/sortd/internal/store"
	"github.com/harsh-sreehari/sortd/internal/watcher"
	"io"
)

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
	return cfg, st, pipe, nil
}

var rootCmd = &cobra.Command{
	Use:   "sortd",
	Short: "sortd is a context-aware file organiser daemon",
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

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent sort history",
	Run: func(cmd *cobra.Command, args []string) {
		_, st, _, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		logs, err := st.RecentLog(20)
		if err != nil {
			log.Fatalf("Failed to fetch logs: %v", err)
		}

		if len(logs) == 0 {
			fmt.Println("No recent activity.")
			return
		}

		fmt.Printf("%-20s | %-10s | %-40s | %-8s | %s\n", "Timestamp", "Action", "Filename", "Tier", "Destination")
		fmt.Println(strings.Repeat("-", 120))
		for _, l := range logs {
			base := filepath.Base(l.Filename)
			if len(base) > 38 {
				base = base[:35] + "..."
			}
			dest := l.Destination
			// Give extra room for destination
			fmt.Printf("%-20s | %-10s | %-40s | Tier %-2d | %s\n", l.Timestamp, l.Action, base, l.Tier, dest)
		}
	},
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "List files in .unsorted/ for interactive resolve",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, st, pipe, err := initPipeline()
		if err != nil {
			log.Fatalf("Init failed: %v", err)
		}
		defer st.Close()

		var root string
		if len(cfg.Watch.Folders) > 0 {
			root = cfg.Watch.Folders[0]
		}
		unsortedDir := filepath.Join(root, ".unsorted")

		files, err := os.ReadDir(unsortedDir)
		if err != nil {
			fmt.Printf("No unsorted files found in %s\n", unsortedDir)
			return
		}

		scanner := bufio.NewScanner(os.Stdin)
		for _, f := range files {
			if f.IsDir() {
				continue
			}

			srcPath := filepath.Join(unsortedDir, f.Name())
			fmt.Printf("\nFile: %s\nWhere to? [skip/path]: ", f.Name())
			
			if !scanner.Scan() {
				break
			}
			
			dest := strings.TrimSpace(scanner.Text())
			if dest == "" || dest == "skip" {
				fmt.Println("Skipped.")
				continue
			}

			finalPath, err := pipe.Mover.Move(srcPath, dest)
			if err != nil {
				fmt.Printf("Failed to move: %v\n", err)
			} else {
				fmt.Printf("Moved to: %s\n", finalPath)
			}
		}
		fmt.Println("Review complete.")
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
	}
	
	fmt.Printf("Re-indexing your folders in: %v...\n", roots)
	err = pipe.Graph.Crawl(roots, []string{"/node_modules", "/.git", "/.unsorted", "/.local", "/.cache", "/.gemini", "/.agent"})
	if err != nil {
		log.Fatalf("Crawl failed: %v", err)
	}
	fmt.Println("Indexing complete.")
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

		// 3. Install systemd service
		serviceFile := "sortd.service"
		if _, err := os.Stat(serviceFile); err == nil {
			dest := filepath.Join(home, ".config/systemd/user/sortd.service")
			os.MkdirAll(filepath.Dir(dest), 0755)

			src, _ := os.Open(serviceFile)
			defer src.Close()
			dst, _ := os.Create(dest)
			defer dst.Close()
			io.Copy(dst, src)

			fmt.Printf("✅ Systemd user service installed to %s\n", dest)
			fmt.Println("\nTo start the daemon, run:")
			fmt.Println("  systemctl --user daemon-reload")
			fmt.Println("  systemctl --user enable --now sortd")
		} else {
			fmt.Println("⚠️  sortd.service file not found in current directory, skipping service installation.")
		}

		fmt.Println("\nInitialization complete! sortd is ready to go.")
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd, logCmd, reviewCmd, runCmd, indexCmd, initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
