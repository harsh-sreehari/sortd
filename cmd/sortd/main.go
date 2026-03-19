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
)

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
				case "moved", "Software/":
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

		fmt.Printf("%-20s | %-10s | %-30s | %-10s | %s\n", "Timestamp", "Action", "Filename", "Tier", "Destination")
		fmt.Println(strings.Repeat("-", 100))
		for _, l := range logs {
			base := filepath.Base(l.Filename)
			if len(base) > 28 {
				base = base[:25] + "..."
			}
			dest := l.Destination
			if len(dest) > 30 {
				dest = "..." + dest[len(dest)-27:]
			}
			fmt.Printf("%-20s | %-10s | %-30s | Tier %-5d | %s\n", l.Timestamp, l.Action, base, l.Tier, dest)
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
		fmt.Println("Re-indexing folder tree...")
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	rootCmd.AddCommand(daemonCmd, logCmd, reviewCmd, runCmd, indexCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
