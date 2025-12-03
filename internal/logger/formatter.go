// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"unicode/utf8"
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
	if err := l.encoder.Encode(jsonEntry); err != nil {
		return err
	}
	// Sync if the writer supports it (for immediate flush)
	if syncer, ok := l.writer.(interface{ Sync() error }); ok {
		syncer.Sync()
	}
	return nil
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
		jsonEntry.Payload = formatPayloadForJSON(entry.Payload)
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

// formatPayloadForJSON converts payload to JSON-friendly format:
// - If it's a byte array and printable UTF-8, return as string
// - If it's a byte array but not printable, return as base64 string
// - Otherwise, return as-is
func formatPayloadForJSON(payload interface{}) interface{} {
	// Check if it's a byte slice
	if bytes, ok := payload.([]byte); ok {
		// Check if it's valid UTF-8 and printable
		if utf8.Valid(bytes) && isPrintableJSON(bytes) {
			return string(bytes)
		}
		// Not printable, return as base64
		return base64.StdEncoding.EncodeToString(bytes)
	}
	// Not a byte slice, return as-is
	return payload
}

// isPrintableJSON checks if a byte slice contains only printable characters
func isPrintableJSON(data []byte) bool {
	for _, b := range data {
		// Allow printable ASCII, tabs, newlines, and carriage returns
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
		if b == 127 { // DEL character
			return false
		}
	}
	return true
}
