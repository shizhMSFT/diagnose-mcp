// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlobUploader(t *testing.T) {
	// Create a temp file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.log")

	// Note: We can't fully test without a real Azure connection
	// This test just verifies the constructor works
	t.Run("creates uploader with default config", func(t *testing.T) {
		// Use a mock URL (won't actually connect)
		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"

		uploader, err := NewBlobUploader(tempFile, blobURL, nil)

		// Should create uploader successfully (doesn't validate URL until upload)
		assert.NoError(t, err)
		assert.NotNil(t, uploader)
		assert.Equal(t, tempFile, uploader.filePath)
		assert.Equal(t, blobURL, uploader.blobURL)
		assert.Equal(t, 10*time.Second, uploader.uploadPeriod)
	})

	t.Run("creates uploader with custom config", func(t *testing.T) {
		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		config := &BlobUploaderConfig{
			UploadPeriod: 5 * time.Second,
		}

		uploader, err := NewBlobUploader(tempFile, blobURL, config)

		assert.NoError(t, err)
		assert.NotNil(t, uploader)
		assert.Equal(t, 5*time.Second, uploader.uploadPeriod)
	})
}

func TestBlobUploader_LocalFileHandling(t *testing.T) {
	t.Run("skips upload when file is empty", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "empty.log")

		// Create empty file
		err := os.WriteFile(tempFile, []byte{}, 0644)
		require.NoError(t, err)

		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		uploader, err := NewBlobUploader(tempFile, blobURL, nil)
		require.NoError(t, err)

		// Upload should succeed (skip) on empty file
		err = uploader.upload()
		assert.NoError(t, err)
	})

	t.Run("handles missing file gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "nonexistent.log")

		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		uploader, err := NewBlobUploader(tempFile, blobURL, nil)
		require.NoError(t, err)

		// Upload should return error but not crash
		err = uploader.upload()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat log file")
	})
}

func TestBlobUploader_StartAndClose(t *testing.T) {
	t.Run("can start and stop uploader", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test.log")

		// Create a file with some content
		err := os.WriteFile(tempFile, []byte("test log content\n"), 0644)
		require.NoError(t, err)

		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		config := &BlobUploaderConfig{
			UploadPeriod: 100 * time.Millisecond, // Short interval for testing
		}

		uploader, err := NewBlobUploader(tempFile, blobURL, config)
		require.NoError(t, err)

		// Start the uploader
		uploader.Start()

		// Let it run briefly
		time.Sleep(50 * time.Millisecond)

		// Close should complete without hanging
		err = uploader.Close()
		assert.NoError(t, err)
	})

	t.Run("close waits for upload goroutine", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test.log")

		// Create empty file so upload is skipped (no network call)
		err := os.WriteFile(tempFile, []byte{}, 0644)
		require.NoError(t, err)

		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		uploader, err := NewBlobUploader(tempFile, blobURL, nil)
		require.NoError(t, err)

		uploader.Start()

		// Close should block until goroutine completes
		start := time.Now()
		err = uploader.Close()
		duration := time.Since(start)

		assert.NoError(t, err)
		// Should complete quickly since file is empty (no actual upload)
		assert.Less(t, duration, 1*time.Second)
	})
}

func TestBlobUploader_LastUploadTime(t *testing.T) {
	t.Run("returns zero time initially", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test.log")

		blobURL := "https://test.blob.core.windows.net/container/blob?sig=test"
		uploader, err := NewBlobUploader(tempFile, blobURL, nil)
		require.NoError(t, err)

		lastUpload := uploader.LastUploadTime()
		assert.True(t, lastUpload.IsZero())
	})
}

// Note: Full integration tests with real Azure Blob Storage would require:
// - Valid Azure storage account
// - SAS token with write permissions
// - Network connectivity
// These should be in integration tests, not unit tests
