// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

// BlobUploader periodically uploads a local log file to Azure Blob Storage as a block blob.
// It reads the entire file and overwrites the blob on each upload.
// This is simpler and more reliable than streaming to append blobs.
type BlobUploader struct {
	filePath       string
	blobURL        string
	uploadPeriod   time.Duration
	client         *blockblob.Client
	stopCh         chan struct{}
	doneCh         chan struct{}
	mu             sync.Mutex
	lastUpload     time.Time
	lastUploadSize int64 // Size of file at last upload
}

// BlobUploaderConfig configures the BlobUploader behavior
type BlobUploaderConfig struct {
	// UploadPeriod is how often to upload the file (default: 10s)
	UploadPeriod time.Duration
}

// DefaultBlobUploaderConfig returns sensible defaults
func DefaultBlobUploaderConfig() *BlobUploaderConfig {
	return &BlobUploaderConfig{
		UploadPeriod: 10 * time.Second,
	}
}

// NewBlobUploader creates a BlobUploader that periodically uploads a local file to Azure Blob Storage.
// The blobURL should be a full URL to a block blob, including SAS token if needed.
// Example: https://<account>.blob.core.windows.net/<container>/<blob>?<sas>
//
// The blob will be created if it doesn't exist, or overwritten on each upload.
func NewBlobUploader(filePath, blobURL string, config *BlobUploaderConfig) (*BlobUploader, error) {
	if config == nil {
		config = DefaultBlobUploaderConfig()
	}

	// Create block blob client (using no credential since URL contains SAS)
	client, err := blockblob.NewClientWithNoCredential(blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	bu := &BlobUploader{
		filePath:     filePath,
		blobURL:      blobURL,
		uploadPeriod: config.UploadPeriod,
		client:       client,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	return bu, nil
}

// Start begins the periodic upload process in a background goroutine
func (bu *BlobUploader) Start() {
	go bu.uploadLoop()
}

// uploadLoop runs in the background and uploads the file periodically
func (bu *BlobUploader) uploadLoop() {
	defer close(bu.doneCh)

	ticker := time.NewTicker(bu.uploadPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = bu.upload() // Best effort; errors are logged but don't stop the loop
		case <-bu.stopCh:
			// Final upload before exit
			_ = bu.upload()
			return
		}
	}
}

// upload reads the entire file and uploads it to the block blob (overwrites existing)
func (bu *BlobUploader) upload() error {
	bu.mu.Lock()
	defer bu.mu.Unlock()

	// Check file size first to avoid reading if nothing changed
	fileInfo, err := os.Stat(bu.filePath)
	if err != nil {
		// File might not exist yet, or might be temporarily locked
		// This is not fatal - we'll try again on next interval
		return fmt.Errorf("failed to stat log file %s: %w", bu.filePath, err)
	}

	fileSize := fileInfo.Size()

	// Skip upload if file is empty
	if fileSize == 0 {
		return nil
	}

	// Skip upload if file size hasn't changed since last upload
	if bu.lastUploadSize > 0 && fileSize == bu.lastUploadSize {
		return nil
	}

	// Read the entire file
	data, err := os.ReadFile(bu.filePath)
	if err != nil {
		return fmt.Errorf("failed to read log file %s: %w", bu.filePath, err)
	}

	// Upload to block blob (overwrites existing)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Upload the entire file as a block blob
	_, err = bu.client.UploadBuffer(ctx, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upload to blob: %w", err)
	}

	bu.lastUpload = time.Now()
	bu.lastUploadSize = fileSize
	return nil
}

// Close stops the upload loop and performs a final upload
func (bu *BlobUploader) Close() error {
	close(bu.stopCh)
	<-bu.doneCh // Wait for upload goroutine to finish
	return nil
}

// LastUploadTime returns the time of the last successful upload
func (bu *BlobUploader) LastUploadTime() time.Time {
	bu.mu.Lock()
	defer bu.mu.Unlock()
	return bu.lastUpload
}
