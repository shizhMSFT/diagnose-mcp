// Package watcher provides file state tracking for watched files
package watcher

import (
	"fmt"
	"io"
	"os"
)

// FileState tracks the state of a watched file
type FileState struct {
	// Path is the absolute path to the file
	Path string
	// Size is the current file size in bytes
	Size int64
	// LineCount is the number of lines in the file
	LineCount int64
	// LastModified is the last modification time
	LastModified int64
}

// NewFileState creates a new file state tracker
func NewFileState(path string) (*FileState, error) {
	fs := &FileState{
		Path: path,
	}

	// Initialize state by reading file
	if err := fs.readState(); err != nil {
		return nil, err
	}

	return fs, nil
}

// Update updates the file state and returns lines added
func (fs *FileState) Update() (int64, error) {
	oldLineCount := fs.LineCount
	oldSize := fs.Size

	if err := fs.readState(); err != nil {
		return 0, err
	}

	linesAdded := fs.LineCount - oldLineCount

	// Detect file truncation
	if fs.Size < oldSize {
		// File was truncated/replaced
		linesAdded = fs.LineCount - oldLineCount
	}

	return linesAdded, nil
}

// readState reads the current file state
func (fs *FileState) readState() error {
	info, err := os.Stat(fs.Path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fs.Size = info.Size()
	fs.LastModified = info.ModTime().Unix()

	// Count lines
	lineCount, err := countLines(fs.Path)
	if err != nil {
		return fmt.Errorf("failed to count lines: %w", err)
	}

	fs.LineCount = lineCount
	return nil
}

// countLines counts the number of lines in a file (lines ending with \n)
func countLines(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var count int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := file.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					count++
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	return count, nil
}
