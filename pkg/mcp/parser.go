package mcp

import (
	"encoding/json"
	"fmt"
)

// ParseMessage parses a single MCP protocol message from JSON bytes
func ParseMessage(data []byte) (*Message, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate JSON-RPC version
	if base.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid or missing jsonrpc field (must be \"2.0\")")
	}

	msg := &Message{
		RawJSON:        data,
		JSONRPCVersion: base.JSONRPC,
		Method:         base.Method,
		Params:         base.Params,
		ID:             base.ID,
		Result:         base.Result,
		Error:          base.Error,
	}

	// Detect message type based on fields present
	msg.Type = detectMessageType(&base)

	// Validate message structure
	if err := validateMessage(msg, &base); err != nil {
		return nil, err
	}

	return msg, nil
}

// detectMessageType determines the message type based on fields present
func detectMessageType(base *BaseMessage) MessageType {
	hasMethod := base.Method != ""
	hasID := base.ID != nil
	hasResult := len(base.Result) > 0
	hasError := base.Error != nil

	// Response: has result OR error (and typically has id)
	if hasResult || hasError {
		return MessageTypeResponse
	}

	// Request: has method AND id
	if hasMethod && hasID {
		return MessageTypeRequest
	}

	// Notification: has method, no id
	if hasMethod && !hasID {
		// Special case: progress notifications
		if base.Method == "$/progress" {
			return MessageTypeProgressUpdate
		}
		return MessageTypeNotification
	}

	// Default to notification if unclear
	return MessageTypeNotification
}

// validateMessage validates the message structure based on type
func validateMessage(msg *Message, base *BaseMessage) error {
	switch msg.Type {
	case MessageTypeRequest:
		if base.Method == "" {
			return fmt.Errorf("request must have method field")
		}
		if base.ID == nil {
			return fmt.Errorf("request must have id field")
		}

	case MessageTypeResponse:
		hasResult := len(base.Result) > 0
		hasError := base.Error != nil

		if hasResult && hasError {
			return fmt.Errorf("response cannot have both result and error fields")
		}
		if !hasResult && !hasError {
			return fmt.Errorf("response must have either result or error field")
		}

	case MessageTypeNotification, MessageTypeProgressUpdate:
		if base.Method == "" {
			return fmt.Errorf("notification must have method field")
		}
		// ID should not be present for notifications, but we allow id=null
		// since some implementations may include it
	}

	return nil
}
