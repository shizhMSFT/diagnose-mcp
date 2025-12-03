// Package watcher provides file state tracking for watched files
package watcher

import (
	"fmt"
	"os"
	"sync"
)

// FileState tracks the state of a watched file
type FileState struct {
	// Path is the absolute path to the file
	Path string
	// Size is the current file size in bytes
	Size int64
	// Offset is the last read position (for tailing)
	Offset int64
	// LastModified is the last modification time
	LastModified int64
	// mu protects concurrent access to state fields
	mu sync.Mutex
}

// NewFileState creates a new file state tracker
func NewFileState(path string) (*FileState, error) {
	fs := &FileState{
		Path: path,
	}

	// Initialize state by reading file info
	if err := fs.readState(); err != nil {
		return nil, err
	}

	// Start at end of file for tailing
	fs.Offset = fs.Size

	return fs, nil
}

// Update updates the file state and returns new content
func (fs *FileState) Update() (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	oldSize := fs.Size
	oldOffset := fs.Offset

	if err := fs.readState(); err != nil {
		return "", err
	}

	// Detect file truncation or replacement
	// File is considered rotated if:
	// 1. Size decreased (truncated)
	// 2. New size is less than our current offset (file was replaced with smaller content)
	// 3. Offset would be beyond new size
	if fs.Size < oldSize || fs.Size < oldOffset || oldOffset > fs.Size {
		// File was truncated/replaced - reset offset to read from beginning
		fs.Offset = 0
	}

	// Read new content from offset
	if fs.Size > fs.Offset {
		content, err := fs.readNewContent()
		if err != nil {
			return "", err
		}
		fs.Offset = fs.Size
		return content, nil
	}

	return "", nil
}

// readState reads the current file state (metadata only, no file locking)
func (fs *FileState) readState() error {
	info, err := os.Stat(fs.Path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fs.Size = info.Size()
	fs.LastModified = info.ModTime().Unix()
	return nil
}
