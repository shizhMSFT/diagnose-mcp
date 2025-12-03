//go:build windows

package watcher

import (
	"io"
	"syscall"
)

// readNewContent reads new content from the last offset using Windows-specific file sharing
func (fs *FileState) readNewContent() (string, error) {
	// Open file with FILE_SHARE_DELETE|FILE_SHARE_READ|FILE_SHARE_WRITE to avoid locking
	pathPtr, err := syscall.UTF16PtrFromString(fs.Path)
	if err != nil {
		return "", err
	}

	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)

	// Seek to last offset
	_, err = syscall.Seek(handle, fs.Offset, io.SeekStart)
	if err != nil {
		return "", err
	}

	// Read new content (limit to 100KB to avoid huge logs)
	maxRead := int64(100 * 1024)
	toRead := fs.Size - fs.Offset
	if toRead > maxRead {
		toRead = maxRead
	}

	buf := make([]byte, toRead)
	var bytesRead uint32
	err = syscall.ReadFile(handle, buf, &bytesRead, nil)
	if err != nil && err != io.EOF {
		return "", err
	}

	return string(buf[:bytesRead]), nil
}
