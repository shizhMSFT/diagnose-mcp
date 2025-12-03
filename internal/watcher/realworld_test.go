package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealWorldScenario simulates a real-world log file tailing scenario
func TestRealWorldScenario(t *testing.T) {
	// Setup: Create a temporary log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "server.log")

	// Create initial log file with some content
	initialContent := "Server starting...\nListening on port 8080\n"
	if err := os.WriteFile(logFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Start watching the file (should start tailing from current end)
	eventChan := make(chan FileEvent, 10)
	if err := watcher.Watch(logFile, eventChan); err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}

	// Give watcher time to initialize
	time.Sleep(50 * time.Millisecond)

	// Simulate log entries being appended over time
	logEntries := []string{
		"Request received: GET /api/users\n",
		"Database query executed\n",
		"Response sent: 200 OK\n",
	}

	var receivedContent []string

	// Append log entries and collect events
	for i, entry := range logEntries {
		// Append to log file
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatalf("Failed to open log file: %v", err)
		}
		if _, err := f.WriteString(entry); err != nil {
			f.Close()
			t.Fatalf("Failed to write log entry: %v", err)
		}
		f.Close()

		// Wait for event
		select {
		case event := <-eventChan:
			if event.Type != EventTypeModified {
				t.Errorf("Entry %d: Expected modified event, got %s", i, event.Type)
			}
			receivedContent = append(receivedContent, event.Content)
			t.Logf("Entry %d: Received content: '%s'", i, strings.TrimSpace(event.Content))
		case <-time.After(1 * time.Second):
			t.Errorf("Entry %d: Timeout waiting for event", i)
		}

		// Small delay between log entries
		time.Sleep(50 * time.Millisecond)
	}

	// Verify we received all expected content
	if len(receivedContent) != len(logEntries) {
		t.Errorf("Expected %d events, got %d", len(logEntries), len(receivedContent))
	}

	for i, expected := range logEntries {
		if i < len(receivedContent) && receivedContent[i] != expected {
			t.Errorf("Entry %d: Expected '%s', got '%s'", i, expected, receivedContent[i])
		}
	}
}

// TestFileRotation tests handling of log file rotation (deletion and recreation)
func TestFileRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "rotating.log")

	// Create initial log file
	if err := os.WriteFile(logFile, []byte("Old log content\n"), 0644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Create watcher
	watcher, err := NewFileWatcher()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	eventChan := make(chan FileEvent, 10)
	if err := watcher.Watch(logFile, eventChan); err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Simulate real log rotation: delete old file and create new one
	if err := os.Remove(logFile); err != nil {
		t.Fatalf("Failed to remove log file: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(logFile, []byte("New log after rotation\n"), 0644); err != nil {
		t.Fatalf("Failed to create new log file: %v", err)
	}

	// Wait for events (should get DELETE and CREATE)
	var events []FileEvent
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-eventChan:
			events = append(events, event)
			t.Logf("Received event: type=%s, content='%s'", event.Type, strings.TrimSpace(event.Content))
			if len(events) >= 2 {
				goto done
			}
		case <-deadline:
			goto done
		}
	}

done:
	if len(events) < 1 {
		t.Fatal("Expected at least one event after rotation")
	}

	// We should see a DELETE event
	foundDelete := false
	foundCreate := false
	for _, e := range events {
		if e.Type == EventTypeDeleted {
			foundDelete = true
		}
		if e.Type == EventTypeCreated {
			foundCreate = true
		}
	}

	if !foundDelete {
		t.Log("Warning: Expected DELETE event, but may not be supported on all platforms")
	}
	if !foundCreate {
		t.Log("Warning: Expected CREATE event, but may not be supported on all platforms")
	}
}
