package watcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shizhMSFT/diagnose-mcp/internal/watcher"
	"github.com/stretchr/testify/require"
)

// T043: Unit test - File creation detection
func TestFileWatcher_DetectsFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	err = w.Watch(testFile, eventChan)
	require.NoError(t, err)

	// Create the file
	err = os.WriteFile(testFile, []byte("initial content\n"), 0644)
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeCreated, event.Type)
		require.Equal(t, testFile, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file creation event")
	}
}

func TestFileWatcher_DetectsFileCreation_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.log")
	file2 := filepath.Join(tmpDir, "test2.log")

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	err = w.Watch(file1, eventChan)
	require.NoError(t, err)
	err = w.Watch(file2, eventChan)
	require.NoError(t, err)

	// Small delay to ensure watchers are set up
	time.Sleep(100 * time.Millisecond)

	// Create both files
	os.WriteFile(file1, []byte("file1\n"), 0644)
	os.WriteFile(file2, []byte("file2\n"), 0644)

	// Should receive 2 creation events (or modification events on some platforms)
	events := 0
	timeout := time.After(2 * time.Second)
	for events < 2 {
		select {
		case event := <-eventChan:
			// Accept both Created and Modified (Windows sometimes reports Write for new files)
			require.Contains(t, []watcher.EventType{watcher.EventTypeCreated, watcher.EventTypeModified}, event.Type)
			events++
		case <-timeout:
			t.Fatalf("Only received %d events, expected 2", events)
		}
	}
}

// T044: Unit test - Line append detection
func TestFileWatcher_DetectsLineAppend(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create initial file
	err := os.WriteFile(testFile, []byte("line 1\n"), 0644)
	require.NoError(t, err)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	err = w.Watch(testFile, eventChan)
	require.NoError(t, err)

	// Append a line
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("line 2\n")
	require.NoError(t, err)
	f.Close()

	// Wait for event
	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeModified, event.Type)
		require.Equal(t, testFile, event.Path)
		require.Equal(t, int64(1), event.LinesAdded) // 1 new line
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file modification event")
	}
}

func TestFileWatcher_DetectsMultipleLineAppends(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create initial file
	os.WriteFile(testFile, []byte("line 1\n"), 0644)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	w.Watch(testFile, eventChan)

	// Append multiple lines
	f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("line 2\nline 3\nline 4\n")
	f.Close()

	// Wait for event
	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeModified, event.Type)
		require.Equal(t, int64(3), event.LinesAdded) // 3 new lines
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for modification event")
	}
}

func TestFileWatcher_IgnoresPartialLines(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	os.WriteFile(testFile, []byte("line 1\n"), 0644)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	w.Watch(testFile, eventChan)

	// Append without newline
	f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("partial line without newline")
	f.Close()

	// Wait for event
	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeModified, event.Type)
		require.Equal(t, int64(0), event.LinesAdded) // No complete lines added
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for modification event")
	}
}

// T045: Unit test - File deletion detection
func TestFileWatcher_DetectsFileDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create file
	os.WriteFile(testFile, []byte("content\n"), 0644)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)
	w.Watch(testFile, eventChan)

	// Delete the file
	err = os.Remove(testFile)
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeDeleted, event.Type)
		require.Equal(t, testFile, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file deletion event")
	}
}

func TestFileWatcher_HandlesNonExistentFile(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nonexistent.log")

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)
	defer w.Stop()

	eventChan := make(chan watcher.FileEvent, 10)

	// Should not error when watching non-existent file
	err = w.Watch(testFile, eventChan)
	require.NoError(t, err)

	// Creating the file later should trigger event
	os.WriteFile(testFile, []byte("new file\n"), 0644)

	select {
	case event := <-eventChan:
		require.Equal(t, watcher.EventTypeCreated, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for creation event")
	}
}

func TestFileWatcher_StopsWatching(t *testing.T) {
	t.Skip("TDD: Test written first - implementation pending")

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")
	os.WriteFile(testFile, []byte("initial\n"), 0644)

	w, err := watcher.NewFileWatcher()
	require.NoError(t, err)

	eventChan := make(chan watcher.FileEvent, 10)
	w.Watch(testFile, eventChan)

	// Stop the watcher
	err = w.Stop()
	require.NoError(t, err)

	// Modify file after stopping
	f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("after stop\n")
	f.Close()

	// Should not receive any event
	select {
	case <-eventChan:
		t.Fatal("Received event after watcher was stopped")
	case <-time.After(500 * time.Millisecond):
		// Expected - no event
	}
}
