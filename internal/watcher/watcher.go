package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/harsh-sreehari/sortd/internal/config"
)

type Watcher struct {
	cfg        *config.Config
	Out        chan string
	fsWatcher  *fsnotify.Watcher
	debounceMs map[string]*time.Timer
	timersMu   sync.Mutex
}

func New(cfg *config.Config) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %v", err)
	}

	return &Watcher{
		cfg:        cfg,
		Out:        make(chan string, 100),
		fsWatcher:  fsWatcher,
		debounceMs: make(map[string]*time.Timer),
	}, nil
}

func (w *Watcher) Start(ctx context.Context) error {
	// Add folders to watch
	for _, folder := range w.cfg.Watch.Folders {
		if err := w.fsWatcher.Add(folder); err != nil {
			log.Printf("failed to watch folder %s: %v", folder, err)
		}
	}

	go w.eventLoop(ctx)
	return nil
}

func (w *Watcher) eventLoop(ctx context.Context) {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				w.handleEvent(event.Name, false)
			} else if event.Has(fsnotify.Write) {
				w.handleEvent(event.Name, true)
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		case <-ctx.Done():
			w.fsWatcher.Close()
			return
		}
	}
}

func (w *Watcher) handleEvent(path string, isWrite bool) {
	// Filtering
	if w.shouldFilter(path) {
		return
	}

	// Abs path
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Printf("failed to get absolute path for %s: %v", path, err)
		return
	}

	w.timersMu.Lock()
	defer w.timersMu.Unlock()

	timer, exists := w.debounceMs[absPath]
	if isWrite && !exists {
		return
	}

	if exists {
		timer.Stop()
	}

	w.debounceMs[absPath] = time.AfterFunc(time.Duration(w.cfg.Behaviour.DebounceSeconds)*time.Second, func() {
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			w.Out <- absPath
		}
		
		w.timersMu.Lock()
		delete(w.debounceMs, absPath)
		w.timersMu.Unlock()
	})
}

func (w *Watcher) shouldFilter(path string) bool {
	filename := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(filename))

	// Hidden files
	if strings.HasPrefix(filename, ".") {
		return true
	}

	// Partial downloads
	switch ext {
	case ".crdownload", ".part", ".tmp", ".download":
		return true
	}

	// .unsorted directory
	if strings.Contains(path, "/.unsorted") {
		return true
	}

	return false
}

func (w *Watcher) Stop() {
	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}
}
