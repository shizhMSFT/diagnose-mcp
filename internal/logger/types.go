// Package logger provides structured logging for diagnose-mcp
package logger

import (
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	// LogLevelInfo for informational messages
	LogLevelInfo LogLevel = "INFO"
	// LogLevelError for error messages
	LogLevelError LogLevel = "ERROR"
	// LogLevelDebug for debug messages (verbose mode)
	LogLevelDebug LogLevel = "DEBUG"
)

// LogEntryType represents the type of diagnostic event
type LogEntryType string

const (
	// LogEntryTypeProxy for proxy lifecycle events (start, stop)
	LogEntryTypeProxy LogEntryType = "proxy"
	// LogEntryTypeRequest for MCP request messages
	LogEntryTypeRequest LogEntryType = "request"
	// LogEntryTypeResponse for MCP response messages
	LogEntryTypeResponse LogEntryType = "response"
	// LogEntryTypeNotification for MCP notification messages
	LogEntryTypeNotification LogEntryType = "notification"
	// LogEntryTypeProgress for MCP progress notifications
	LogEntryTypeProgress LogEntryType = "progress"
	// LogEntryTypeFile for file change events
	LogEntryTypeFile LogEntryType = "file"
	// LogEntryTypeError for error events
	LogEntryTypeError LogEntryType = "error"
)

// LogEntry represents a single diagnostic log entry
type LogEntry struct {
	// Timestamp is when the event occurred
	Timestamp time.Time

	// Level is the severity level
	Level LogLevel

	// Type is the type of diagnostic event
	Type LogEntryType

	// Direction is "->" (outbound) or "<-" (inbound) for messages
	Direction string

	// Method is the MCP method name for requests/responses/notifications
	Method string

	// ID is the JSON-RPC message ID for requests/responses
	ID interface{}

	// Message is a human-readable description
	Message string

	// Context provides additional key-value pairs
	Context map[string]interface{}

	// Payload is the full message payload (verbose mode only)
	Payload interface{}

	// Error contains error details if applicable
	Error *ErrorDetails
}

// ErrorDetails represents error information in a log entry
type ErrorDetails struct {
	// Code is the JSON-RPC error code
	Code int
	// Message is the error message
	Message string
	// Data contains additional error data
	Data interface{}
}

// NewLogEntry creates a new log entry with default values
func NewLogEntry(level LogLevel, entryType LogEntryType, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Type:      entryType,
		Message:   message,
		Context:   make(map[string]interface{}),
	}
}

// WithDirection sets the direction for message log entries
func (e *LogEntry) WithDirection(direction string) *LogEntry {
	e.Direction = direction
	return e
}

// WithMethod sets the method for message log entries
func (e *LogEntry) WithMethod(method string) *LogEntry {
	e.Method = method
	return e
}

// WithID sets the ID for request/response log entries
func (e *LogEntry) WithID(id interface{}) *LogEntry {
	e.ID = id
	return e
}

// WithContext adds a context key-value pair
func (e *LogEntry) WithContext(key string, value interface{}) *LogEntry {
	e.Context[key] = value
	return e
}

// WithPayload sets the full message payload (verbose mode)
func (e *LogEntry) WithPayload(payload interface{}) *LogEntry {
	e.Payload = payload
	return e
}

// WithError sets error details
func (e *LogEntry) WithError(code int, message string, data interface{}) *LogEntry {
	e.Error = &ErrorDetails{
		Code:    code,
		Message: message,
		Data:    data,
	}
	return e
}
