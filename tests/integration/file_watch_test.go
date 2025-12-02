package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shizhMSFT/diagnose-mcp/internal/watcher"
	"github.com/stretchr/testify/require"
)

// T047: Integration test - File watching with real filesystem
func TestFileWatcher_Integration_RealFilesystem(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "integration.log")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 100)
	err = w.Watch(testFile, eventChan)
	require.NoError(t, err)

	// Scenario 1: Create file
	os.WriteFile(testFile, []byte("line 1\n"), 0644)
	event := waitForEvent(t, ctx, eventChan)
	require.Equal(t, watcher.EventTypeCreated, event.Type)

	// Scenario 2: Append lines
	f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("line 2\nline 3\n")
	f.Close()
	event = waitForEvent(t, ctx, eventChan)
	require.Equal(t, watcher.EventTypeModified, event.Type)
	require.Equal(t, int64(2), event.LinesAdded)

	// Scenario 3: Delete file
	os.Remove(testFile)
	event = waitForEvent(t, ctx, eventChan)
	require.Equal(t, watcher.EventTypeDeleted, event.Type)
}

func TestFileWatcher_Integration_MultipleFiles(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.log")
	file2 := filepath.Join(tmpDir, "file2.log")
	file3 := filepath.Join(tmpDir, "file3.log")

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 100)
	w.Watch(file1, eventChan)
	w.Watch(file2, eventChan)
	w.Watch(file3, eventChan)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create all files
	os.WriteFile(file1, []byte("f1 line 1\n"), 0644)
	os.WriteFile(file2, []byte("f2 line 1\n"), 0644)
	os.WriteFile(file3, []byte("f3 line 1\n"), 0644)

	// Should receive 3 creation events
	events := make(map[string]bool)
	for i := 0; i < 3; i++ {
		event := waitForEvent(t, ctx, eventChan)
		require.Equal(t, watcher.EventTypeCreated, event.Type)
		events[event.Path] = true
	}

	require.Len(t, events, 3)
	require.True(t, events[file1])
	require.True(t, events[file2])
	require.True(t, events[file3])
}

func TestFileWatcher_Integration_RapidChanges(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "rapid.log")

	os.WriteFile(testFile, []byte("initial\n"), 0644)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 100)
	w.Watch(testFile, eventChan)

	// Make rapid changes
	for i := 0; i < 10; i++ {
		f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
		f.WriteString("line\n")
		f.Close()
		time.Sleep(50 * time.Millisecond)
	}

	// Should receive multiple modification events
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventCount := 0
	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-eventChan:
			if event.Type == watcher.EventTypeModified {
				eventCount++
			}
		case <-timeout:
			require.Greater(t, eventCount, 0, "Should receive at least one modification event")
			return
		case <-ctx.Done():
			t.Fatal("Context timeout")
		}
	}
}

func TestFileWatcher_Integration_LargeFile(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.log")

	// Create file with 1000 lines
	f, _ := os.Create(testFile)
	for i := 0; i < 1000; i++ {
		f.WriteString("This is a test line with some content\n")
	}
	f.Close()

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	w.Watch(testFile, eventChan)

	// Append 100 more lines
	f, _ = os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	for i := 0; i < 100; i++ {
		f.WriteString("Additional line\n")
	}
	f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := waitForEvent(t, ctx, eventChan)
	require.Equal(t, watcher.EventTypeModified, event.Type)
	require.Equal(t, int64(100), event.LinesAdded)
}

func TestFileWatcher_Integration_DirectoryOfFiles(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 100)

	// Watch 5 files in the directory
	for i := 1; i <= 5; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".log")
		w.Watch(file, eventChan)
	}

	// Create and modify files
	for i := 1; i <= 5; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".log")
		os.WriteFile(file, []byte("initial\n"), 0644)
		time.Sleep(100 * time.Millisecond)

		f, _ := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
		f.WriteString("appended\n")
		f.Close()
		time.Sleep(100 * time.Millisecond)
	}

	// Count events
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	creationEvents := 0
	modificationEvents := 0
	timeout := time.After(3 * time.Second)

	for {
		select {
		case event := <-eventChan:
			switch event.Type {
			case watcher.EventTypeCreated:
				creationEvents++
			case watcher.EventTypeModified:
				modificationEvents++
			}
		case <-timeout:
			require.Equal(t, 5, creationEvents, "Should detect all file creations")
			require.Equal(t, 5, modificationEvents, "Should detect all modifications")
			return
		case <-ctx.Done():
			t.Fatal("Context timeout")
		}
	}
}

// Helper function to wait for an event with timeout
func waitForEvent(t *testing.T, ctx context.Context, eventChan <-chan watcher.FileEvent) watcher.FileEvent {
	select {
	case event := <-eventChan:
		return event
	case <-ctx.Done():
		t.Fatal("Timeout waiting for file event")
		return watcher.FileEvent{}
	}
}
