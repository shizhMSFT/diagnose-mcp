// Package watcher provides file system watching functionality for MCP proxy sessions
package watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/shizhMSFT/fspoll-go"
)

// LogFunc is a function that logs file events
type LogFunc func(eventType, path, content string)

// FileWatcher watches files for changes using polling
type FileWatcher struct {
	watchers map[string]*fspoll.FileWatcher
	mu       sync.RWMutex
	stopped  bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	return &FileWatcher{
		watchers: make(map[string]*fspoll.FileWatcher),
	}, nil
}

// Watch starts watching a file for changes
func (w *FileWatcher) Watch(path string, logFunc LogFunc) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return fmt.Errorf("watcher has been stopped")
	}

	// Create a callback that logs events directly
	callback := func(filePath string, event fspoll.Event, content []byte) {
		var eventType string
		switch event {
		case fspoll.EventCreate:
			eventType = "created"
		case fspoll.EventAppend, fspoll.EventTruncate:
			eventType = "modified"
		case fspoll.EventDelete:
			eventType = "deleted"
		}
		logFunc(eventType, filePath, string(content))
	}

	// Create a new fspoll FileWatcher with 1 second polling interval
	fw, err := fspoll.NewFileWatcher(path, time.Second, callback)
	if err != nil {
		return fmt.Errorf("failed to create file watcher for %s: %w", path, err)
	}

	w.watchers[path] = fw
	return nil
}

// Unwatch stops watching a file
func (w *FileWatcher) Unwatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	fw, exists := w.watchers[path]
	if !exists {
		return fmt.Errorf("file %s is not being watched", path)
	}

	fw.Stop()
	delete(w.watchers, path)
	return nil
}

// Stop stops all file watchers
func (w *FileWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.stopped = true
	for _, fw := range w.watchers {
		fw.Stop()
	}
	w.watchers = make(map[string]*fspoll.FileWatcher)
}
