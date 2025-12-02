// Package proxy provides MCP protocol proxy implementations
package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shizhMSFT/diagnose-mcp/pkg/mcp"
)

// RemoteProxy represents a proxy connection to a remote MCP server
type RemoteProxy struct {
	url            string
	conn           *websocket.Conn
	httpClient     *http.Client
	isWebSocket    bool
	mu             sync.Mutex
	connected      bool
	ctx            context.Context
	cancel         context.CancelFunc
	connectTimeout time.Duration
	readTimeout    time.Duration
}

// NewRemoteProxy creates a new remote proxy instance
func NewRemoteProxy(serverURL string) (*RemoteProxy, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	isWS := parsedURL.Scheme == "ws" || parsedURL.Scheme == "wss"

	return &RemoteProxy{
		url:            serverURL,
		isWebSocket:    isWS,
		httpClient:     &http.Client{Timeout: 5 * time.Minute},
		connectTimeout: 30 * time.Second,
		readTimeout:    5 * time.Minute,
	}, nil
}

// Connect establishes connection to the remote server
func (r *RemoteProxy) Connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	r.ctx, r.cancel = context.WithCancel(ctx)

	if r.isWebSocket {
		return r.connectWebSocket()
	}

	// HTTP connections are established per-request
	r.connected = true
	return nil
}

// connectWebSocket establishes a WebSocket connection
func (r *RemoteProxy) connectWebSocket() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: r.connectTimeout,
	}

	conn, _, err := dialer.Dial(r.url, nil)
	if err != nil {
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	r.conn = conn
	r.connected = true
	return nil
}

// IsConnected returns whether the proxy is connected
func (r *RemoteProxy) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.connected
}

// ForwardClientMessage sends a message from client to remote server
func (r *RemoteProxy) ForwardClientMessage(msgBytes []byte) error {
	if r.isWebSocket {
		return r.forwardWebSocketMessage(msgBytes)
	}
	return r.forwardHTTPMessage(msgBytes)
}

// forwardWebSocketMessage sends message over WebSocket
func (r *RemoteProxy) forwardWebSocketMessage(msgBytes []byte) error {
	r.mu.Lock()
	conn := r.conn
	r.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected to websocket server")
	}

	err := conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send websocket message: %w", err)
	}

	return nil
}

// forwardHTTPMessage sends message over HTTP POST
func (r *RemoteProxy) forwardHTTPMessage(msgBytes []byte) error {
	req, err := http.NewRequestWithContext(r.ctx, "POST", r.url, strings.NewReader(string(msgBytes)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For HTTP, we need to read and forward the response
	// This will be handled by ReadServerMessage

	return nil
}

// ReadServerMessage reads a message from the remote server
func (r *RemoteProxy) ReadServerMessage() ([]byte, error) {
	if r.isWebSocket {
		return r.readWebSocketMessage()
	}
	return r.readHTTPMessage()
}

// readWebSocketMessage reads from WebSocket connection
func (r *RemoteProxy) readWebSocketMessage() ([]byte, error) {
	r.mu.Lock()
	conn := r.conn
	r.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected to websocket server")
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read websocket message: %w", err)
	}

	return message, nil
}

// readHTTPMessage reads from HTTP response
func (r *RemoteProxy) readHTTPMessage() ([]byte, error) {
	// For HTTP, responses are read in ForwardClientMessage
	// This is a placeholder for consistency with the interface
	return nil, fmt.Errorf("HTTP response reading not implemented in this method")
}

// SendToClient writes a message to client stdout
func (r *RemoteProxy) SendToClient(msgBytes []byte) error {
	// Write message to stdout
	_, err := fmt.Println(string(msgBytes))
	return err
}

// Stop closes the remote connection
func (r *RemoteProxy) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	if r.cancel != nil {
		r.cancel()
	}

	if r.conn != nil {
		err := r.conn.Close()
		r.conn = nil
		r.connected = false
		return err
	}

	r.connected = false
	return nil
}

// Run starts the remote proxy session
func (r *RemoteProxy) Run(ctx context.Context, logger io.Writer) error {
	if err := r.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer r.Stop()

	// Read from stdin and forward to remote server
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse as MCP message
		msg, err := mcp.ParseMessage(line)
		if err != nil {
			// Log error but continue
			fmt.Fprintf(logger, "Failed to parse message: %v\n", err)
			continue
		}

		// Forward to remote server
		if err := r.ForwardClientMessage(line); err != nil {
			fmt.Fprintf(logger, "Failed to forward message: %v\n", err)
			continue
		}

		// Log the outbound message
		fmt.Fprintf(logger, "-> %s #%v\n", msg.Method, msg.ID)

		// Read response (for WebSocket)
		if r.isWebSocket {
			response, err := r.ReadServerMessage()
			if err != nil {
				fmt.Fprintf(logger, "Failed to read response: %v\n", err)
				continue
			}

			// Parse response
			respMsg, err := mcp.ParseMessage(response)
			if err != nil {
				fmt.Fprintf(logger, "Failed to parse response: %v\n", err)
				continue
			}

			// Log the inbound message
			fmt.Fprintf(logger, "<- #%v\n", respMsg.ID)

			// Send to client
			if err := r.SendToClient(response); err != nil {
				fmt.Fprintf(logger, "Failed to send to client: %v\n", err)
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}
