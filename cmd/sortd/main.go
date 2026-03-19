package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
		fmt.Println("Starting sort daemon...")
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping sort daemon...")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the daemon is running",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking sort daemon status...")
	},
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent sort history",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Showing sort log...")
	},
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "List files in .unsorted/ for interactive resolve",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Reviewing unsorted files...")
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manually trigger a sort pass on watched folders",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running a manual sort pass...")
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
	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd, logCmd, reviewCmd, runCmd, indexCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
