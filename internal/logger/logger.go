// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Logger handles formatting and writing log entries
type Logger struct {
	writer  io.Writer
	verbose bool
}

// NewLogger creates a new logger with the specified writer
func NewLogger(writer io.Writer, verbose bool) *Logger {
	return &Logger{
		writer:  writer,
		verbose: verbose,
	}
}

// Log writes a log entry in text format
func (l *Logger) Log(entry *LogEntry) error {
	formatted := l.formatText(entry)
	_, err := l.writer.Write([]byte(formatted + "\n"))
	return err
}

// formatText formats a log entry as human-readable text
func (l *Logger) formatText(entry *LogEntry) string {
	var parts []string

	// Timestamp: 2024-01-15T10:30:45.123Z
	timestamp := entry.Timestamp.Format(time.RFC3339Nano)
	parts = append(parts, timestamp)

	// Level: [INFO], [ERROR], [DEBUG]
	level := fmt.Sprintf("[%s]", entry.Level)
	parts = append(parts, level)

	// Type: [proxy], [request], [response], etc.
	entryType := fmt.Sprintf("[%s]", entry.Type)
	parts = append(parts, entryType)

	// Direction + Method (for messages): → initialize, ← initialized
	if entry.Direction != "" {
		var methodPart string
		if entry.Method != "" {
			methodPart = fmt.Sprintf("%s %s", entry.Direction, entry.Method)
		} else {
			methodPart = entry.Direction
		}
		parts = append(parts, methodPart)
	}

	// ID (for requests/responses): #123, #"abc"
	if entry.ID != nil {
		idPart := fmt.Sprintf("#%v", entry.ID)
		parts = append(parts, idPart)
	}

	// Message
	if entry.Message != "" {
		parts = append(parts, entry.Message)
	}

	// Build first line
	firstLine := strings.Join(parts, " ")

	var lines []string
	lines = append(lines, firstLine)

	// Context (key=value pairs)
	if len(entry.Context) > 0 {
		var contextParts []string
		for k, v := range entry.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", k, v))
		}
		contextLine := "  Context: " + strings.Join(contextParts, ", ")
		lines = append(lines, contextLine)
	}

	// Error details
	if entry.Error != nil {
		errorLine := fmt.Sprintf("  Error: [%d] %s", entry.Error.Code, entry.Error.Message)
		lines = append(lines, errorLine)
		if entry.Error.Data != nil {
			dataLine := fmt.Sprintf("  Error Data: %v", entry.Error.Data)
			lines = append(lines, dataLine)
		}
	}

	// Payload (verbose mode only)
	if l.verbose && entry.Payload != nil {
		payloadLine := fmt.Sprintf("  Payload: %v", entry.Payload)

		// Truncate if too long
		const maxLength = 1000
		if len(payloadLine) > maxLength {
			payloadLine = payloadLine[:maxLength] + "... (truncated)"
		}

		lines = append(lines, payloadLine)
	}

	return strings.Join(lines, "\n")
}

// SetVerbose updates the verbose setting
func (l *Logger) SetVerbose(verbose bool) {
	l.verbose = verbose
}
