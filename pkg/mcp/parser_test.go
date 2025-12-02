package mcp_test

import (
	"testing"

	"github.com/shizhMSFT/diagnose-mcp/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMessage_ValidRequest(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0"}}`)

	msg, err := mcp.ParseMessage(data)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, mcp.MessageTypeRequest, msg.Type)
	assert.Equal(t, "initialize", msg.Method)
	assert.Equal(t, float64(1), msg.ID)
	assert.True(t, msg.IsRequest())
	assert.False(t, msg.IsResponse())
}

func TestParseMessage_ValidResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`)

	msg, err := mcp.ParseMessage(data)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, mcp.MessageTypeResponse, msg.Type)
	assert.Equal(t, float64(1), msg.ID)
	assert.True(t, msg.IsResponse())
	assert.False(t, msg.IsRequest())
	assert.False(t, msg.HasError())
}

func TestParseMessage_ErrorResponse(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid request"}}`)

	msg, err := mcp.ParseMessage(data)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.True(t, msg.IsResponse())
	assert.True(t, msg.HasError())
	assert.Equal(t, -32600, msg.Error.Code)
	assert.Equal(t, "Invalid request", msg.Error.Message)
}

func TestParseMessage_Notification(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"initialized"}`)

	msg, err := mcp.ParseMessage(data)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, mcp.MessageTypeNotification, msg.Type)
	assert.Equal(t, "initialized", msg.Method)
	assert.True(t, msg.IsNotification())
	assert.False(t, msg.IsRequest())
}

func TestParseMessage_ProgressNotification(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"$/progress","params":{"token":"abc","value":50}}`)

	msg, err := mcp.ParseMessage(data)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, mcp.MessageTypeProgressUpdate, msg.Type)
	assert.True(t, msg.IsProgressUpdate())
	assert.True(t, msg.IsNotification())
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestParseMessage_MissingJSONRPC(t *testing.T) {
	data := []byte(`{"id":1,"method":"test"}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing jsonrpc field")
}

func TestParseMessage_InvalidJSONRPCVersion(t *testing.T) {
	data := []byte(`{"jsonrpc":"1.0","id":1,"method":"test"}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jsonrpc")
}

func TestParseMessage_RequestMissingMethod(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "method")
}

func TestParseMessage_NotificationMissingMethod(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0"}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification must have method")
}

func TestParseMessage_ResponseMissingResultAndError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
}

func TestParseMessage_ResponseWithBothResultAndError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{},"error":{"code":-1,"message":"error"}}`)

	_, err := mcp.ParseMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot have both result and error")
}

func TestMessage_HelperMethods(t *testing.T) {
	// Request
	req := &mcp.Message{Type: mcp.MessageTypeRequest}
	assert.True(t, req.IsRequest())
	assert.False(t, req.IsResponse())
	assert.False(t, req.IsNotification())
	assert.False(t, req.IsProgressUpdate())

	// Response without error
	resp := &mcp.Message{Type: mcp.MessageTypeResponse}
	assert.True(t, resp.IsResponse())
	assert.False(t, resp.HasError())

	// Response with error
	errResp := &mcp.Message{
		Type:  mcp.MessageTypeResponse,
		Error: &mcp.RPCError{Code: -1, Message: "error"},
	}
	assert.True(t, errResp.HasError())

	// Notification
	notif := &mcp.Message{Type: mcp.MessageTypeNotification}
	assert.True(t, notif.IsNotification())

	// Progress
	prog := &mcp.Message{Type: mcp.MessageTypeProgressUpdate}
	assert.True(t, prog.IsProgressUpdate())
	assert.True(t, prog.IsNotification())
}
