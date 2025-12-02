package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shizhMSFT/diagnose-mcp/internal/watcher"
	"github.com/stretchr/testify/require"
)

// T046: Unit test - File state tracking (size, line count)
func TestFileState_TracksLineCount(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create file with 3 lines
	os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644)

	state, err := watcher.NewFileState(testFile)
	require.NoError(t, err)

	require.Equal(t, int64(3), state.LineCount)
	require.Greater(t, state.Size, int64(0))
}

func TestFileState_TracksFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	content := []byte("Hello, World!\n")
	os.WriteFile(testFile, content, 0644)

	state, err := watcher.NewFileState(testFile)
	require.NoError(t, err)

	require.Equal(t, int64(len(content)), state.Size)
}

func TestFileState_UpdateDetectsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Initial state
	os.WriteFile(testFile, []byte("line 1\n"), 0644)
	state, err := watcher.NewFileState(testFile)
	require.NoError(t, err)

	oldLineCount := state.LineCount
	oldSize := state.Size

	// Append lines
	f, _ := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("line 2\nline 3\n")
	f.Close()

	// Update state
	linesAdded, err := state.Update()
	require.NoError(t, err)

	require.Equal(t, int64(2), linesAdded)
	require.Equal(t, oldLineCount+2, state.LineCount)
	require.Greater(t, state.Size, oldSize)
}

func TestFileState_HandlesEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.log")

	os.WriteFile(testFile, []byte(""), 0644)

	state, err := watcher.NewFileState(testFile)
	require.NoError(t, err)

	require.Equal(t, int64(0), state.LineCount)
	require.Equal(t, int64(0), state.Size)
}

func TestFileState_HandlesFileWithoutTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// File without trailing newline
	os.WriteFile(testFile, []byte("line 1\nline 2"), 0644)

	state, err := watcher.NewFileState(testFile)
	require.NoError(t, err)

	// Should count 1 complete line (only lines ending with \n)
	require.Equal(t, int64(1), state.LineCount)
}

func TestFileState_UpdateHandlesFileDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	os.WriteFile(testFile, []byte("line 1\n"), 0644)
	state, _ := watcher.NewFileState(testFile)

	// Delete file
	os.Remove(testFile)

	// Update should return error
	_, err := state.Update()
	require.Error(t, err)
}

func TestFileState_UpdateDetectsTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create file with content
	os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644)
	state, _ := watcher.NewFileState(testFile)

	// Truncate file
	os.WriteFile(testFile, []byte("line 1\n"), 0644)

	linesAdded, err := state.Update()
	require.NoError(t, err)

	// File was truncated, so it's essentially a reset
	require.Equal(t, int64(-2), linesAdded) // Lost 2 lines
	require.Equal(t, int64(1), state.LineCount)
}

func TestFileState_CountsLinesCorrectly(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected int64
	}{
		{"empty", "", 0},
		{"one line", "line 1\n", 1},
		{"three lines", "line 1\nline 2\nline 3\n", 3},
		{"no trailing newline", "line 1\nline 2", 1},
		{"blank lines", "line 1\n\nline 3\n", 3},
		{"only newlines", "\n\n\n", 3},
	}

	tmpDir := t.TempDir()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tc.name+".log")
			os.WriteFile(testFile, []byte(tc.content), 0644)

			state, err := watcher.NewFileState(testFile)
			require.NoError(t, err)
			require.Equal(t, tc.expected, state.LineCount, "Content: %q", tc.content)
		})
	}
}
