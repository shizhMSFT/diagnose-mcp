// Package mcp provides types and utilities for the MCP (Model Context Protocol) JSON-RPC 2.0 messages.
package mcp

import "encoding/json"

// MessageType represents the type of MCP message
type MessageType string

const (
	// MessageTypeRequest represents a JSON-RPC 2.0 request (has method and id)
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse represents a JSON-RPC 2.0 response (has result or error, and id)
	MessageTypeResponse MessageType = "response"
	// MessageTypeNotification represents a JSON-RPC 2.0 notification (has method, no id)
	MessageTypeNotification MessageType = "notification"
	// MessageTypeProgressUpdate represents a special notification for progress updates (method="$/progress")
	MessageTypeProgressUpdate MessageType = "progress"
)

// Message represents a parsed MCP protocol message
type Message struct {
	// Type is the detected message type
	Type MessageType
	// RawJSON is the original message bytes
	RawJSON []byte
	// JSONRPCVersion should always be "2.0"
	JSONRPCVersion string
	// Method is present for requests and notifications
	Method string
	// Params contains the parameters (if present)
	Params json.RawMessage
	// ID is present for requests and responses (may be string, number, or null)
	ID interface{}
	// Result is present for successful responses
	Result json.RawMessage
	// Error is present for error responses
	Error *RPCError
}

// RPCError represents a JSON-RPC 2.0 error object
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// BaseMessage represents the base JSON-RPC 2.0 structure
type BaseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// IsRequest returns true if the message is a request (has method and id)
func (m *Message) IsRequest() bool {
	return m.Type == MessageTypeRequest
}

// IsResponse returns true if the message is a response (has result or error)
func (m *Message) IsResponse() bool {
	return m.Type == MessageTypeResponse
}

// IsNotification returns true if the message is a notification (has method, no id)
func (m *Message) IsNotification() bool {
	return m.Type == MessageTypeNotification || m.Type == MessageTypeProgressUpdate
}

// IsProgressUpdate returns true if the message is a progress notification
func (m *Message) IsProgressUpdate() bool {
	return m.Type == MessageTypeProgressUpdate
}

// HasError returns true if the response message contains an error
func (m *Message) HasError() bool {
	return m.Error != nil
}
