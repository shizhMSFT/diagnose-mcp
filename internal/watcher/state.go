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
	// Offset is the last read position (for tailing)
	Offset int64
	// LastModified is the last modification time
	LastModified int64
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
	oldSize := fs.Size

	if err := fs.readState(); err != nil {
		return "", err
	}

	// Detect file truncation or replacement
	if fs.Size < oldSize {
		// File was truncated/replaced - reset offset
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

// readNewContent reads new content from the last offset
func (fs *FileState) readNewContent() (string, error) {
	// Open with FILE_SHARE_DELETE|FILE_SHARE_READ|FILE_SHARE_WRITE to avoid locking
	file, err := os.Open(fs.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Seek to last offset
	if _, err := file.Seek(fs.Offset, io.SeekStart); err != nil {
		return "", err
	}

	// Read new content (limit to 100KB to avoid huge logs)
	maxRead := int64(100 * 1024)
	toRead := fs.Size - fs.Offset
	if toRead > maxRead {
		toRead = maxRead
	}

	buf := make([]byte, toRead)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}

	return string(buf[:n]), nil
}
