package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchNonExistentDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a path where neither the file nor its parent directory exists
	nonExistentDir := filepath.Join(tmpDir, "does", "not", "exist")
	nonExistentFile := filepath.Join(nonExistentDir, "test.log")

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Create event channel
	eventChan := make(chan FileEvent, 10)

	// Watch the non-existent file (should not error)
	err = watcher.Watch(nonExistentFile, eventChan)
	if err != nil {
		t.Errorf("Watch failed for non-existent directory: %v", err)
	}

	// Verify the placeholder state was created
	watcher.mu.RLock()
	state, exists := watcher.states[nonExistentFile]
	watcher.mu.RUnlock()

	if !exists {
		t.Error("Expected placeholder state to be created")
	}
	if state.Size != 0 {
		t.Errorf("Expected size 0, got %d", state.Size)
	}
}

func TestWatchExistingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "existing")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// File path (file doesn't exist yet, but directory does)
	testFile := filepath.Join(subDir, "test.log")

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Create event channel
	eventChan := make(chan FileEvent, 10)

	// Watch the file (should succeed and actually watch the directory)
	err = watcher.Watch(testFile, eventChan)
	if err != nil {
		t.Errorf("Watch failed for existing directory: %v", err)
	}

	// Verify the state was created
	watcher.mu.RLock()
	_, exists := watcher.states[testFile]
	watcher.mu.RUnlock()

	if !exists {
		t.Error("Expected state to be created")
	}

	// Now create the file and verify we get an event
	if err := os.WriteFile(testFile, []byte("test content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for event (with timeout)
	select {
	case event := <-eventChan:
		if event.Type != EventTypeCreated && event.Type != EventTypeModified {
			t.Errorf("Expected created or modified event, got %s", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file creation event")
	}
}

func TestWatchExistingFile(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.log")

	// Create the file
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Create event channel
	eventChan := make(chan FileEvent, 10)

	// Watch the existing file
	err = watcher.Watch(testFile, eventChan)
	if err != nil {
		t.Errorf("Watch failed for existing file: %v", err)
	}

	// Verify state was created with correct size
	watcher.mu.RLock()
	state, exists := watcher.states[testFile]
	watcher.mu.RUnlock()

	if !exists {
		t.Fatal("Expected state to be created")
	}
	if state.Size != 16 { // "initial content\n"
		t.Errorf("Expected size 16, got %d", state.Size)
	}

	// Append to the file
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	if _, err := f.WriteString("appended\n"); err != nil {
		f.Close()
		t.Fatalf("Failed to write to file: %v", err)
	}
	f.Close()

	// Wait for modification event
	select {
	case event := <-eventChan:
		if event.Type != EventTypeModified {
			t.Errorf("Expected modified event, got %s", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file modification event")
	}
}

func TestWatchThenCreateDirectoryAndFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a path where neither the file nor its parent directories exist
	nonExistentDir := filepath.Join(tmpDir, "level1", "level2", "level3")
	testFile := filepath.Join(nonExistentDir, "test.log")

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Create event channel
	eventChan := make(chan FileEvent, 10)

	// Watch the non-existent file first
	err = watcher.Watch(testFile, eventChan)
	if err != nil {
		t.Fatalf("Watch failed for non-existent directory: %v", err)
	}

	// Verify placeholder state was created
	watcher.mu.RLock()
	state, exists := watcher.states[testFile]
	watcher.mu.RUnlock()

	if !exists {
		t.Fatal("Expected placeholder state to be created")
	}
	if state.Size != 0 {
		t.Errorf("Expected size 0, got %d", state.Size)
	}

	// Now create the parent directories
	if err := os.MkdirAll(nonExistentDir, 0755); err != nil {
		t.Fatalf("Failed to create parent directories: %v", err)
	}

	// We need to manually watch the directory now that it exists
	// (since the initial watch attempt didn't set up fsnotify)
	err = watcher.Watch(testFile, eventChan)
	if err != nil {
		t.Fatalf("Failed to watch after directory creation: %v", err)
	}

	// Create the file
	if err := os.WriteFile(testFile, []byte("first line\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for creation event
	select {
	case event := <-eventChan:
		if event.Type != EventTypeCreated && event.Type != EventTypeModified {
			t.Errorf("Expected created or modified event, got %s", event.Type)
		}
		t.Logf("Received event: type=%s, path=%s", event.Type, event.Path)
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file creation event")
	}

	// Append content to the file
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	if _, err := f.WriteString("second line\n"); err != nil {
		f.Close()
		t.Fatalf("Failed to write to file: %v", err)
	}
	f.Close()

	// Wait for modification event
	select {
	case event := <-eventChan:
		if event.Type != EventTypeModified {
			t.Errorf("Expected modified event, got %s", event.Type)
		}
		t.Logf("Received event: type=%s, path=%s, content=%s", event.Type, event.Path, event.Content)
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file modification event")
	}
}
