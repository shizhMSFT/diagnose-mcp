// Package proxy provides core proxy session management
package proxy

import (
	"time"

	"github.com/shizhMSFT/diagnose-mcp/pkg/mcp"
)

// MessageDirection represents the direction of a message
type MessageDirection string

const (
	// MessageDirectionInbound indicates a message from the MCP server to the client
	MessageDirectionInbound MessageDirection = "inbound"
	// MessageDirectionOutbound indicates a message from the client to the MCP server
	MessageDirectionOutbound MessageDirection = "outbound"
)

// MCPMessage represents a single MCP message with metadata
type MCPMessage struct {
	// SessionID identifies the proxy session
	SessionID string

	// Direction is inbound or outbound
	Direction MessageDirection

	// Timestamp is when the message was received
	Timestamp time.Time

	// Message is the parsed MCP message
	Message *mcp.Message

	// RawBytes is the original message bytes
	RawBytes []byte

	// SequenceNumber is the message number within the session
	SequenceNumber int64
}

// NewMCPMessage creates a new MCP message with metadata
func NewMCPMessage(sessionID string, direction MessageDirection, message *mcp.Message, rawBytes []byte, sequenceNumber int64) *MCPMessage {
	return &MCPMessage{
		SessionID:      sessionID,
		Direction:      direction,
		Timestamp:      time.Now(),
		Message:        message,
		RawBytes:       rawBytes,
		SequenceNumber: sequenceNumber,
	}
}

// IsRequest returns true if this is a request message
func (m *MCPMessage) IsRequest() bool {
	return m.Message != nil && m.Message.IsRequest()
}

// IsResponse returns true if this is a response message
func (m *MCPMessage) IsResponse() bool {
	return m.Message != nil && m.Message.IsResponse()
}

// IsNotification returns true if this is a notification message
func (m *MCPMessage) IsNotification() bool {
	return m.Message != nil && m.Message.IsNotification()
}

// IsProgressUpdate returns true if this is a progress notification
func (m *MCPMessage) IsProgressUpdate() bool {
	return m.Message != nil && m.Message.IsProgressUpdate()
}

// HasError returns true if this is an error response
func (m *MCPMessage) HasError() bool {
	return m.Message != nil && m.Message.HasError()
}

// GetMethod returns the MCP method name (for requests/notifications)
func (m *MCPMessage) GetMethod() string {
	if m.Message == nil {
		return ""
	}
	return m.Message.Method
}

// GetID returns the message ID (for requests/responses)
func (m *MCPMessage) GetID() interface{} {
	if m.Message == nil {
		return nil
	}
	return m.Message.ID
}
