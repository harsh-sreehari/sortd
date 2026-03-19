package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harsh-sreehari/sortd/internal/config"
)

func TestWatcher(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "sortd-watcher-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Watch.Folders = []string{tmpDir}
	cfg.Behaviour.DebounceSeconds = 1 // 1s for test

	w, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Test CREATE
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case path := <-w.Out:
		if path != testFile {
			t.Errorf("expected path %s, got %s", testFile, path)
		}
	case <-time.After(3 * time.Second): // Debounce (1s) + buffer
		t.Error("timeout waiting for watcher event")
	}

	// Test FILTER (.crdownload)
	crFile := filepath.Join(tmpDir, "test.crdownload")
	if err := os.WriteFile(crFile, []byte("downloading"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case path := <-w.Out:
		t.Errorf("unexpected event for filtered file %s", path)
	case <-time.After(2 * time.Second):
		// SUCCESS - filter should NOT emit
	}

	// Test FILTER (hidden)
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("hidden"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case path := <-w.Out:
		t.Errorf("unexpected event for hidden file %s", path)
	case <-time.After(2 * time.Second):
		// SUCCESS
	}
}
