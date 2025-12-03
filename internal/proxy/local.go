// Package proxy provides local MCP server proxy implementation
package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/shizhMSFT/diagnose-mcp/pkg/mcp"
)

// LocalProxy manages a local MCP server process and proxies stdio communication
type LocalProxy struct {
	// Config
	serverBinary string
	serverArgs   []string
	sessionID    string

	// Process management
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	running bool
	mu      sync.RWMutex

	// Message handling
	session          *ProxySession
	sequenceNumber   int64
	messageHandler   MessageHandler
	stderrBuffer     []byte
	stderrBufferLock sync.Mutex
}

// MessageHandler processes intercepted messages
type MessageHandler func(msg *MCPMessage) error

// NewLocalProxy creates a new local proxy instance
func NewLocalProxy(serverBinary string, serverArgs []string) *LocalProxy {
	sessionID := fmt.Sprintf("local-%d", os.Getpid())
	session := NewProxySession(sessionID)
	session.ServerBinary = serverBinary
	session.ServerArgs = serverArgs

	return &LocalProxy{
		serverBinary: serverBinary,
		serverArgs:   serverArgs,
		sessionID:    sessionID,
		session:      session,
		stderrBuffer: make([]byte, 0, 4096),
	}
}

// SetMessageHandler sets the handler for intercepted messages
func (p *LocalProxy) SetMessageHandler(handler MessageHandler) {
	p.messageHandler = handler
}

// Start spawns the server process and sets up stdio pipes
func (p *LocalProxy) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("proxy already running")
	}

	// Create command with context
	p.cmd = exec.CommandContext(ctx, p.serverBinary, p.serverArgs...)

	// Inherit parent environment variables
	p.cmd.Env = os.Environ()

	// Set up stdio pipes
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	p.running = true
	p.session.SetState(SessionStateActive)

	// Start goroutines to handle stdio
	go p.readStderr()

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		p.Stop()
	}()

	return nil
}

// Stop terminates the server process gracefully
func (p *LocalProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.running = false
	p.session.SetState(SessionStateShuttingDown)

	// Close stdin to signal server to shut down
	if p.stdin != nil {
		p.stdin.Close()
	}

	// Try graceful shutdown first
	if p.cmd != nil && p.cmd.Process != nil {
		// Send SIGTERM on Unix-like systems
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, force kill
			p.cmd.Process.Kill()
		}

		// Wait for process to exit
		p.cmd.Wait()
	}

	p.session.SetState(SessionStateClosed)
	return nil
}

// IsRunning returns true if the proxy is currently running
func (p *LocalProxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetPID returns the process ID of the server
func (p *LocalProxy) GetPID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// GetStderr returns the captured stderr output
func (p *LocalProxy) GetStderr() []byte {
	p.stderrBufferLock.Lock()
	defer p.stderrBufferLock.Unlock()
	result := make([]byte, len(p.stderrBuffer))
	copy(result, p.stderrBuffer)
	return result
}

// readStderr captures stderr output separately
func (p *LocalProxy) readStderr() {
	if p.stderr == nil {
		return
	}

	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		line := scanner.Bytes()
		p.stderrBufferLock.Lock()
		p.stderrBuffer = append(p.stderrBuffer, line...)
		p.stderrBuffer = append(p.stderrBuffer, '\n')
		p.stderrBufferLock.Unlock()
	}
}

// ForwardClientMessage forwards a message from client to server
func (p *LocalProxy) ForwardClientMessage(data []byte) error {
	if !p.IsRunning() {
		return fmt.Errorf("proxy not running")
	}

	// Strip newline for parsing only
	lineData := data
	if len(data) > 0 && data[len(data)-1] == '\n' {
		lineData = data[:len(data)-1]
	}

	// Parse and intercept the message
	msg, err := mcp.ParseMessage(lineData)
	if err != nil {
		p.session.IncrementErrorCount()
		return fmt.Errorf("failed to parse client message: %w", err)
	}

	// Create MCP message with metadata
	p.sequenceNumber++
	mcpMsg := NewMCPMessage(
		p.sessionID,
		MessageDirectionOutbound,
		msg,
		data, // Keep newline for forwarding
		p.sequenceNumber,
	)

	// Call message handler if set
	if p.messageHandler != nil {
		if err := p.messageHandler(mcpMsg); err != nil {
			return fmt.Errorf("message handler failed: %w", err)
		}
	}

	// Forward to server stdin
	if _, err := p.stdin.Write(data); err != nil {
		p.session.IncrementErrorCount()
		return fmt.Errorf("failed to write to server stdin: %w", err)
	}

	p.session.IncrementMessageCount()
	return nil
}

// ReadServerMessage reads a message from the server stdout
func (p *LocalProxy) ReadServerMessage() (*MCPMessage, error) {
	if !p.IsRunning() {
		return nil, fmt.Errorf("proxy not running")
	}

	// Read newline-delimited message (keeping the newline)
	reader := bufio.NewReader(p.stdout)
	data, err := reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		p.session.IncrementErrorCount()
		return nil, fmt.Errorf("failed to read from server stdout: %w", err)
	}

	// Strip newline for parsing only
	lineData := data
	if len(data) > 0 && data[len(data)-1] == '\n' {
		lineData = data[:len(data)-1]
	}

	// Parse the message
	msg, err := mcp.ParseMessage(lineData)
	if err != nil {
		p.session.IncrementErrorCount()
		// Include the actual data in the error message to help diagnose issues
		dataPreview := string(data)
		if len(dataPreview) > 100 {
			dataPreview = dataPreview[:100] + "..."
		}

		// Still create an MCPMessage with the raw data so it can be logged and forwarded
		// Message will be nil to indicate parse failure
		p.sequenceNumber++
		mcpMsg := &MCPMessage{
			SessionID:      p.sessionID,
			Direction:      MessageDirectionInbound,
			Timestamp:      time.Now(),
			Message:        nil,  // nil indicates parse failure
			RawBytes:       data, // Keep newline for forwarding
			SequenceNumber: p.sequenceNumber,
		}

		// Call message handler to log the unparseable data
		if p.messageHandler != nil {
			p.messageHandler(mcpMsg)
		}

		// Return the message with error so caller can decide what to do
		return mcpMsg, fmt.Errorf("failed to parse server message: %w (received: %q)", err, dataPreview)
	}

	// Create MCP message with metadata
	p.sequenceNumber++
	return NewMCPMessage(
		p.sessionID,
		MessageDirectionInbound,
		msg,
		data, // Keep newline for forwarding
		p.sequenceNumber,
	), nil
}

// GetSession returns the proxy session
func (p *LocalProxy) GetSession() *ProxySession {
	return p.session
}

// GetStats returns session statistics
func (p *LocalProxy) GetStats() SessionStats {
	return p.session.GetStats()
}
