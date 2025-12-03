// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/appendblob"
)

// BlobWriter implements io.Writer to send logs to Azure Blob Storage (append blob).
// It batches writes to reduce API calls and supports async flushing.
// For dev/test use: create an append blob and provide its SAS URL.
type BlobWriter struct {
	client       *appendblob.Client
	buffer       *bytes.Buffer
	mu           sync.Mutex
	flushSize    int           // Flush when buffer exceeds this size
	flushPeriod  time.Duration // Flush at this interval even if buffer is small
	stopCh       chan struct{}
	doneCh       chan struct{}
	flushOnWrite bool // If true, flush on every write (for low-latency dev mode)
}

// BlobWriterConfig configures the BlobWriter behavior
type BlobWriterConfig struct {
	// FlushSize is the number of bytes to buffer before flushing (default: 64KB)
	FlushSize int
	// FlushPeriod is how often to flush even if buffer is not full (default: 5s)
	FlushPeriod time.Duration
	// FlushOnWrite flushes immediately after each write for low-latency (default: false)
	FlushOnWrite bool
}

// DefaultBlobWriterConfig returns sensible defaults for dev/test
func DefaultBlobWriterConfig() *BlobWriterConfig {
	return &BlobWriterConfig{
		FlushSize:    64 * 1024, // 64KB
		FlushPeriod:  5 * time.Second,
		FlushOnWrite: false,
	}
}

// NewBlobWriter creates a BlobWriter from a blob URL.
// The blobURL should be a full URL to an append blob, including SAS token if needed.
// Example: https://<account>.blob.core.windows.net/<container>/<blob>?<sas>
//
// For dev/test, you can:
// 1. Create an append blob via Azure Portal or CLI
// 2. Generate a SAS token with write permissions
// 3. Pass the full URL to this function
func NewBlobWriter(blobURL string, config *BlobWriterConfig) (*BlobWriter, error) {
	if config == nil {
		config = DefaultBlobWriterConfig()
	}

	// Create append blob client (using no credential since URL contains SAS)
	client, err := appendblob.NewClientWithNoCredential(blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	bw := &BlobWriter{
		client:       client,
		buffer:       &bytes.Buffer{},
		flushSize:    config.FlushSize,
		flushPeriod:  config.FlushPeriod,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		flushOnWrite: config.FlushOnWrite,
	}

	// Start periodic flush goroutine
	go bw.periodicFlush()

	return bw, nil
}

// Write implements io.Writer. It buffers data and flushes when buffer is full or periodically.
func (bw *BlobWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// Write to buffer
	n, err = bw.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Flush if buffer is full or if FlushOnWrite is enabled
	if bw.flushOnWrite || bw.buffer.Len() >= bw.flushSize {
		if err := bw.flushLocked(); err != nil {
			return n, fmt.Errorf("flush failed: %w", err)
		}
	}

	return n, nil
}

// periodicFlush runs in the background and flushes the buffer periodically
func (bw *BlobWriter) periodicFlush() {
	defer close(bw.doneCh)

	ticker := time.NewTicker(bw.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bw.mu.Lock()
			if bw.buffer.Len() > 0 {
				_ = bw.flushLocked() // Best effort; errors are logged but don't block
			}
			bw.mu.Unlock()
		case <-bw.stopCh:
			// Final flush before exit
			bw.mu.Lock()
			_ = bw.flushLocked()
			bw.mu.Unlock()
			return
		}
	}
}

// flushLocked sends the buffer to the append blob. Must be called with lock held.
func (bw *BlobWriter) flushLocked() error {
	if bw.buffer.Len() == 0 {
		return nil
	}

	// Copy buffer for upload (so we can clear it immediately)
	data := make([]byte, bw.buffer.Len())
	copy(data, bw.buffer.Bytes())
	bw.buffer.Reset()

	// Release lock before network call to allow new writes
	bw.mu.Unlock()
	defer bw.mu.Lock()

	// Upload to append blob
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a ReadSeekCloser from the data
	reader := &bytesReadSeekCloser{data: data}
	_, err := bw.client.AppendBlock(ctx, reader, nil)
	if err != nil {
		// Do NOT put data back in buffer - that causes duplicates
		// Just return the error and let the caller handle it
		return fmt.Errorf("failed to append to blob: %w", err)
	}

	return nil
}

// Close flushes any remaining data and stops the background goroutine
func (bw *BlobWriter) Close() error {
	close(bw.stopCh)
	<-bw.doneCh // Wait for flush goroutine to finish
	return nil
}

// bytesReadSeekCloser wraps a byte slice to implement io.ReadSeekCloser
type bytesReadSeekCloser struct {
	data []byte
	pos  int
}

func (r *bytesReadSeekCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var newPos int
	switch whence {
	case io.SeekStart:
		newPos = int(offset)
	case io.SeekCurrent:
		newPos = r.pos + int(offset)
	case io.SeekEnd:
		newPos = len(r.data) + int(offset)
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 || newPos > len(r.data) {
		return 0, fmt.Errorf("seek out of range")
	}

	r.pos = newPos
	return int64(r.pos), nil
}

func (r *bytesReadSeekCloser) Close() error {
	return nil
}
