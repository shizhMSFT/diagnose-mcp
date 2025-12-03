//go:build !windows

package watcher

import (
	"io"
	"os"
)

// readNewContent reads new content from the last offset
func (fs *FileState) readNewContent() (string, error) {
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
