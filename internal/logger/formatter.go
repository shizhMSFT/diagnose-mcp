// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"encoding/json"
	"io"
)

// JSONLogger handles formatting and writing log entries in JSON format
type JSONLogger struct {
	writer  io.Writer
	encoder *json.Encoder
	verbose bool
}

// NewJSONLogger creates a new JSON logger with the specified writer
func NewJSONLogger(writer io.Writer, verbose bool) *JSONLogger {
	encoder := json.NewEncoder(writer)
	return &JSONLogger{
		writer:  writer,
		encoder: encoder,
		verbose: verbose,
	}
}

// Log writes a log entry in JSON format
func (l *JSONLogger) Log(entry *LogEntry) error {
	jsonEntry := l.toJSONEntry(entry)
	return l.encoder.Encode(jsonEntry)
}

// JSONEntry represents a log entry in JSON format
type JSONEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Type      string                 `json:"type"`
	Direction string                 `json:"direction,omitempty"`
	Method    string                 `json:"method,omitempty"`
	ID        interface{}            `json:"id,omitempty"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Payload   interface{}            `json:"payload,omitempty"`
	Error     *JSONErrorDetails      `json:"error,omitempty"`
}

// JSONErrorDetails represents error information in JSON format
type JSONErrorDetails struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// toJSONEntry converts a LogEntry to JSONEntry
func (l *JSONLogger) toJSONEntry(entry *LogEntry) *JSONEntry {
	jsonEntry := &JSONEntry{
		Timestamp: entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		Level:     string(entry.Level),
		Type:      string(entry.Type),
		Direction: entry.Direction,
		Method:    entry.Method,
		ID:        entry.ID,
		Message:   entry.Message,
	}

	// Include context if present
	if len(entry.Context) > 0 {
		jsonEntry.Context = entry.Context
	}

	// Include payload if verbose mode is enabled
	if l.verbose && entry.Payload != nil {
		jsonEntry.Payload = entry.Payload
	}

	// Include error if present
	if entry.Error != nil {
		jsonEntry.Error = &JSONErrorDetails{
			Code:    entry.Error.Code,
			Message: entry.Error.Message,
			Data:    entry.Error.Data,
		}
	}

	return jsonEntry
}

// SetVerbose updates the verbose setting
func (l *JSONLogger) SetVerbose(verbose bool) {
	l.verbose = verbose
}
