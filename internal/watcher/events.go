// Package watcher provides file system watching functionality for MCP proxy sessions
package watcher

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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
	// Size is the current file size in bytes
	Size int64
}

// FileWatcher watches files for changes
type FileWatcher struct {
	watcher    *fsnotify.Watcher
	states     map[string]*FileState
	eventChans map[string]chan<- FileEvent
	mu         sync.RWMutex
	stopChan   chan struct{}
	stopped    bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:    fsWatcher,
		states:     make(map[string]*FileState),
		eventChans: make(map[string]chan<- FileEvent),
		stopChan:   make(chan struct{}),
	}

	go fw.eventLoop()

	return fw, nil
}

// Watch starts watching a file for changes
func (w *FileWatcher) Watch(path string, eventChan chan<- FileEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return fmt.Errorf("watcher has been stopped")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Watch the parent directory (fsnotify watches directories)
	dir := filepath.Dir(absPath)
	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Initialize file state if file exists
	state, err := NewFileState(absPath)
	if err != nil {
		// File doesn't exist yet, create placeholder state
		state = &FileState{
			Path:         absPath,
			Size:         0,
			Offset:       0,
			LastModified: 0,
		}
	}

	w.states[absPath] = state
	w.eventChans[absPath] = eventChan

	return nil
}

// Stop stops the file watcher
func (w *FileWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.stopped = true
	close(w.stopChan)
	return w.watcher.Close()
}

// eventLoop processes fsnotify events
func (w *FileWatcher) eventLoop() {
	for {
		select {
		case <-w.stopChan:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue
			_ = err
		}
	}
}

// handleEvent processes a single fsnotify event
func (w *FileWatcher) handleEvent(event fsnotify.Event) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Check if this file is being watched
	state, exists := w.states[event.Name]
	if !exists {
		return
	}

	eventChan := w.eventChans[event.Name]
	if eventChan == nil {
		return
	}

	var fileEvent FileEvent
	fileEvent.Path = event.Name
	fileEvent.Timestamp = time.Now()

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		fileEvent.Type = EventTypeCreated
		// Update state for created file
		if newState, err := NewFileState(event.Name); err == nil {
			w.states[event.Name] = newState
			fileEvent.Size = newState.Size
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		fileEvent.Type = EventTypeModified
		// Update state and get new content
		if content, err := state.Update(); err == nil {
			fileEvent.Content = content
			fileEvent.Size = state.Size
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		fileEvent.Type = EventTypeDeleted

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		fileEvent.Type = EventTypeDeleted
	}

	// Send event (non-blocking)
	select {
	case eventChan <- fileEvent:
	default:
		// Channel full, skip event
	}
}
