// Package watcher provides file system watching functionality for MCP proxy sessions
package watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/shizhMSFT/fspoll-go"
)

// EventType represents the type of file system event
type EventType string

const (
	// EventTypeCreated indicates a file was created
	EventTypeCreated EventType = "created"
	// EventTypeModified indicates a file was modified
	EventTypeModified EventType = "modified"
	// EventTypeDeleted indicates a file was deleted
	EventTypeDeleted EventType = "deleted"
)

// FileEvent represents a file system event
type FileEvent struct {
	// Type is the event type (created, modified, deleted)
	Type EventType
	// Path is the absolute path to the file
	Path string
	// Timestamp when the event occurred
	Timestamp time.Time
	// Content is the new content added (for modifications)
	Content string
}

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
func (w *FileWatcher) Watch(path string, eventChan chan<- FileEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return fmt.Errorf("watcher has been stopped")
	}

	// Create a callback that translates fspoll events to FileEvent
	callback := func(filePath string, event fspoll.Event, content []byte) {
		fileEvent := FileEvent{
			Path:      filePath,
			Timestamp: time.Now(),
			Content:   string(content),
		}

		switch event {
		case fspoll.EventCreate:
			fileEvent.Type = EventTypeCreated
		case fspoll.EventAppend:
			fileEvent.Type = EventTypeModified
		case fspoll.EventTruncate:
			fileEvent.Type = EventTypeModified
		case fspoll.EventDelete:
			fileEvent.Type = EventTypeDeleted
		}

		// Send event to channel (non-blocking)
		select {
		case eventChan <- fileEvent:
		default:
			// Channel full, skip event
		}
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
