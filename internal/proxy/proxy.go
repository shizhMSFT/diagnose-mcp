// Package proxy provides MCP proxy coordination and logging
package proxy

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/logger"
)

// Proxy coordinates the MCP proxy session with logging
type Proxy struct {
	config     *config.Config
	localProxy *LocalProxy
	logger     *logger.Logger
	jsonLogger *logger.JSONLogger
	logWriter  io.Writer
	clientIn   io.Reader
	clientOut  io.Writer
}

// NewProxy creates a new proxy instance
func NewProxy(cfg *config.Config) *Proxy {
	// Determine log writer (stderr for text/JSON logs, stdout is for MCP messages)
	logWriter := os.Stderr

	var textLogger *logger.Logger
	var jsonLogger *logger.JSONLogger

	if cfg.OutputFormat == config.OutputJSON {
		jsonLogger = logger.NewJSONLogger(logWriter, cfg.Verbose)
	} else {
		textLogger = logger.NewLogger(logWriter, cfg.Verbose)
	}

	return &Proxy{
		config:     cfg,
		logger:     textLogger,
		jsonLogger: jsonLogger,
		logWriter:  logWriter,
		clientIn:   os.Stdin,
		clientOut:  os.Stdout,
	}
}

// Run starts the proxy session
func (p *Proxy) Run(ctx context.Context) error {
	// Log session start
	p.logEvent(logger.LogLevelInfo, logger.LogEntryTypeProxy, "Proxy session starting")

	// Create and start local proxy
	p.localProxy = NewLocalProxy(p.config.ServerBinary, p.config.ServerArgs)

	// Set message handler to log all messages
	p.localProxy.SetMessageHandler(p.handleMessage)

	if err := p.localProxy.Start(ctx); err != nil {
		p.logError("Failed to start server", err)
		return err
	}

	p.logProxyEvent("Server started", map[string]interface{}{
		"binary": p.config.ServerBinary,
		"args":   p.config.ServerArgs,
		"pid":    p.localProxy.GetPID(),
	})

	// Run bidirectional message forwarding
	errChan := make(chan error, 2)

	// Goroutine to forward client -> server
	go func() {
		errChan <- p.forwardClientToServer(ctx)
	}()

	// Goroutine to forward server -> client
	go func() {
		errChan <- p.forwardServerToClient(ctx)
	}()

	// Wait for either goroutine to finish or context cancellation
	select {
	case err := <-errChan:
		p.localProxy.Stop()
		if err != nil && err != io.EOF {
			p.logError("Proxy error", err)
			return err
		}
	case <-ctx.Done():
		p.localProxy.Stop()
	}

	// Log session statistics
	stats := p.localProxy.GetStats()
	p.logProxyEvent("Proxy session ended", map[string]interface{}{
		"duration":      stats.Duration.String(),
		"message_count": stats.MessageCount,
		"error_count":   stats.ErrorCount,
	})

	return nil
}

// forwardClientToServer reads from client stdin and forwards to server
func (p *Proxy) forwardClientToServer(ctx context.Context) error {
	scanner := newLineScanner(p.clientIn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := scanner.ReadLine()
			if err != nil {
				return err
			}

			// Forward to server (this will also log via message handler)
			if err := p.localProxy.ForwardClientMessage(line); err != nil {
				p.logError("Failed to forward client message", err)
				continue
			}
		}
	}
}

// forwardServerToClient reads from server stdout and forwards to client
func (p *Proxy) forwardServerToClient(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := p.localProxy.ReadServerMessage()
			if err != nil {
				return err
			}

			// Forward to client stdout
			if _, err := p.clientOut.Write(msg.RawBytes); err != nil {
				p.logError("Failed to write to client stdout", err)
				return err
			}
			if _, err := p.clientOut.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}
}

// handleMessage logs intercepted MCP messages
func (p *Proxy) handleMessage(msg *MCPMessage) error {
	var entryType logger.LogEntryType
	var direction string

	// Determine entry type and direction
	if msg.IsRequest() {
		entryType = logger.LogEntryTypeRequest
	} else if msg.IsResponse() {
		entryType = logger.LogEntryTypeResponse
	} else if msg.IsProgressUpdate() {
		entryType = logger.LogEntryTypeProgress
	} else if msg.IsNotification() {
		entryType = logger.LogEntryTypeNotification
	}

	if msg.Direction == MessageDirectionOutbound {
		direction = "→"
	} else {
		direction = "←"
	}

	// Create log entry
	entry := logger.NewLogEntry(logger.LogLevelInfo, entryType, fmt.Sprintf("%s %s", direction, msg.GetMethod()))
	entry.WithDirection(direction)
	entry.WithMethod(msg.GetMethod())

	if msg.GetID() != nil {
		entry.WithID(msg.GetID())
	}

	// Add payload if verbose mode
	if p.config.Verbose && msg.Message != nil {
		entry.WithPayload(msg.Message.RawJSON)
	}

	// Add error details if this is an error response
	if msg.HasError() && msg.Message.Error != nil {
		entry.WithError(msg.Message.Error.Code, msg.Message.Error.Message, msg.Message.Error.Data)
	}

	// Log the entry
	return p.logEntry(entry)
}

// logEvent logs a proxy event
func (p *Proxy) logEvent(level logger.LogLevel, entryType logger.LogEntryType, message string) {
	entry := logger.NewLogEntry(level, entryType, message)
	p.logEntry(entry)
}

// logProxyEvent logs a proxy lifecycle event with context
func (p *Proxy) logProxyEvent(message string, context map[string]interface{}) {
	entry := logger.NewLogEntry(logger.LogLevelInfo, logger.LogEntryTypeProxy, message)
	for k, v := range context {
		entry.WithContext(k, v)
	}
	p.logEntry(entry)
}

// logError logs an error event
func (p *Proxy) logError(message string, err error) {
	entry := logger.NewLogEntry(logger.LogLevelError, logger.LogEntryTypeError, message)
	entry.WithContext("error", err.Error())
	p.logEntry(entry)
}

// logEntry writes a log entry using the configured logger
func (p *Proxy) logEntry(entry *logger.LogEntry) error {
	if p.jsonLogger != nil {
		return p.jsonLogger.Log(entry)
	}
	if p.logger != nil {
		return p.logger.Log(entry)
	}
	return nil
}

// lineScanner reads newline-delimited lines
type lineScanner struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	buffer []byte
}

func newLineScanner(r io.Reader) *lineScanner {
	pr, pw := io.Pipe()
	ls := &lineScanner{
		reader: pr,
		writer: pw,
		buffer: make([]byte, 0, 4096),
	}

	// Copy input to pipe
	go func() {
		io.Copy(pw, r)
		pw.Close()
	}()

	return ls
}

func (ls *lineScanner) ReadLine() ([]byte, error) {
	buf := make([]byte, 4096)
	for {
		n, err := ls.reader.Read(buf)
		if err != nil {
			if err == io.EOF && len(ls.buffer) > 0 {
				// Return remaining buffer
				line := make([]byte, len(ls.buffer))
				copy(line, ls.buffer)
				ls.buffer = ls.buffer[:0]
				return line, nil
			}
			return nil, err
		}

		ls.buffer = append(ls.buffer, buf[:n]...)

		// Look for newline
		for i := 0; i < len(ls.buffer); i++ {
			if ls.buffer[i] == '\n' {
				line := make([]byte, i)
				copy(line, ls.buffer[:i])
				ls.buffer = ls.buffer[i+1:]
				return line, nil
			}
		}
	}
}
