// Package proxy provides MCP proxy coordination and logging
package proxy

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shizhMSFT/diagnose-mcp/internal/config"
	"github.com/shizhMSFT/diagnose-mcp/internal/logger"
	"github.com/shizhMSFT/diagnose-mcp/internal/watcher"
)

// Proxy coordinates the MCP proxy session with logging
type Proxy struct {
	config       *config.Config
	localProxy   *LocalProxy
	remoteProxy  *RemoteProxy
	fileWatcher  *watcher.FileWatcher
	logger       *logger.Logger
	jsonLogger   *logger.JSONLogger
	logWriter    io.Writer
	blobUploader io.Closer // Track blob uploader for cleanup
	fileWriter   *os.File  // Track file writer for cleanup
	clientIn     io.Reader
	clientOut    io.Writer
}

// NewProxy creates a new proxy instance
func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		config:    cfg,
		clientIn:  os.Stdin,
		clientOut: os.Stdout,
	}
}

// initLogger initializes the logger with session ID pattern expansion
func (p *Proxy) initLogger(sessionID string) {
	var logWriter io.Writer = os.Stderr // Default to stderr
	var logFilePath string

	// Determine the log file path (either from --log-file or temp file for blob)
	if p.config.LogFile != "" {
		// Expand patterns in log file path
		logPath := p.config.LogFile
		if sessionID != "" {
			logPath = replacePattern(logPath, "{session}", sessionID)
		}
		logPath = replacePattern(logPath, "{pid}", fmt.Sprintf("%d", os.Getpid()))
		// Add timestamp pattern for ordering: {timestamp} -> 20250103-104338
		logPath = replacePattern(logPath, "{timestamp}", time.Now().Format("20060102-150405"))
		logFilePath = logPath
	} else if p.config.LogBlobURL != "" {
		// Create temp file path for blob upload only if --log-file not specified
		logFilePath = getTempLogPath(sessionID)
	}

	// Open log file if needed
	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to open log file %s: %v\n", logFilePath, err)
		} else {
			logWriter = file
			p.fileWriter = file // Store for cleanup

			// Start blob uploader if blob URL specified
			if p.config.LogBlobURL != "" {
				uploader, err := logger.NewBlobUploader(logFilePath, p.config.LogBlobURL, nil)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to create blob uploader for %s: %v\n", p.config.LogBlobURL, err)
				} else {
					uploader.Start()
					p.blobUploader = uploader // Store for cleanup
				}
			}
		}
	}

	p.logWriter = logWriter

	if p.config.OutputFormat == config.OutputJSON {
		p.jsonLogger = logger.NewJSONLogger(logWriter, p.config.Verbose)
	} else {
		p.logger = logger.NewLogger(logWriter, p.config.Verbose)
	}
}

// getTempLogPath generates a temporary file path for blob upload logs
func getTempLogPath(sessionID string) string {
	if sessionID == "" {
		sessionID = fmt.Sprintf("%d", time.Now().Unix())
	}
	return fmt.Sprintf("%s%cdiagnose-mcp-%s.log", os.TempDir(), os.PathSeparator, sessionID)
}

// replacePattern replaces pattern in string
func replacePattern(s, pattern, value string) string {
	result := s
	for i := 0; i < len(result); i++ {
		if i+len(pattern) <= len(result) && result[i:i+len(pattern)] == pattern {
			result = result[:i] + value + result[i+len(pattern):]
			i += len(value) - 1
		}
	}
	return result
}

// Run starts the proxy session
func (p *Proxy) Run(ctx context.Context) error {
	// Ensure cleanup of blob writer if used
	defer p.cleanup()

	// Check if remote mode
	if p.config.ConnectionType == config.ConnectionTypeRemote {
		return p.runRemoteProxy(ctx)
	}

	// Default to local mode
	return p.runLocalProxy(ctx)
}

// cleanup closes any resources (like blob uploader)
func (p *Proxy) cleanup() {
	if p.fileWriter != nil {
		if err := p.fileWriter.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close file writer: %v\n", err)
		}
	}
	if p.blobUploader != nil {
		if err := p.blobUploader.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close blob uploader: %v\n", err)
		}
	}
}

// runRemoteProxy runs the remote proxy mode
func (p *Proxy) runRemoteProxy(ctx context.Context) error {
	var err error
	p.remoteProxy, err = NewRemoteProxy(p.config.RemoteURL)
	if err != nil {
		// Need to initialize logger with empty session for error logging
		p.initLogger("")
		p.logError("Failed to create remote proxy", err)
		return err
	}

	// Initialize logger with remote session ID (remote-<timestamp>)
	remoteSessionID := fmt.Sprintf("remote-%d", time.Now().Unix())
	p.initLogger(remoteSessionID)

	p.logEvent(logger.LogLevelInfo, logger.LogEntryTypeProxy, "Remote proxy session starting")

	p.logProxyEvent("Connecting to remote server", map[string]interface{}{
		"url": p.config.RemoteURL,
	})

	if err := p.remoteProxy.Run(ctx, p.logWriter); err != nil {
		p.logError("Remote proxy error", err)
		return err
	}

	p.logProxyEvent("Remote proxy session ended", map[string]interface{}{})
	return nil
}

// runLocalProxy runs the local proxy mode
func (p *Proxy) runLocalProxy(ctx context.Context) error {
	// Create and start local proxy
	p.localProxy = NewLocalProxy(p.config.ServerBinary, p.config.ServerArgs)

	// Initialize logger with session ID
	p.initLogger(p.localProxy.GetSession().ID)

	// Log session start
	p.logEvent(logger.LogLevelInfo, logger.LogEntryTypeProxy, "Proxy session starting")

	// Log session start
	p.logEvent(logger.LogLevelInfo, logger.LogEntryTypeProxy, "Proxy session starting")

	// Start file watching if requested
	if len(p.config.WatchedFiles) > 0 {
		if err := p.startFileWatching(ctx); err != nil {
			p.logError("Failed to start file watching", err)
			// Continue anyway - file watching is optional
		}
	}

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
		if p.fileWatcher != nil {
			p.fileWatcher.Stop()
		}
		if err != nil && err != io.EOF {
			p.logError("Proxy error", err)
			return err
		}
	case <-ctx.Done():
		p.localProxy.Stop()
		if p.fileWatcher != nil {
			p.fileWatcher.Stop()
		}
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
			if err != nil && msg == nil {
				// Fatal error (EOF or read error) - terminate
				return err
			}
			if err != nil {
				// Parse error but we have the raw data - just log and forward
				// Don't call message handler since it will log again
			}

			// Forward raw data to client stdout (even if parse failed)
			if _, err := p.clientOut.Write(msg.RawBytes); err != nil {
				p.logError("Failed to write to client stdout", err)
				return err
			}
		}
	}
}

// handleMessage logs intercepted MCP messages
func (p *Proxy) handleMessage(msg *MCPMessage) error {
	// Check if this is unparseable data (Message == nil)
	if msg.Message == nil {
		// Log as forwarded non-MCP content
		// Strip trailing newline for logging (it's preserved in RawBytes for forwarding)
		content := string(msg.RawBytes)
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}
		entry := logger.NewLogEntry(logger.LogLevelInfo, logger.LogEntryTypeForward, content)
		return p.logEntry(entry)
	}

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
		direction = "->"
	} else {
		direction = "<-"
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
				// Include the newline in the returned data
				line := make([]byte, i+1)
				copy(line, ls.buffer[:i+1])
				ls.buffer = ls.buffer[i+1:]
				return line, nil
			}
		}
	}
}

// startFileWatching initializes file watching for configured files
func (p *Proxy) startFileWatching(ctx context.Context) error {
	fw, err := watcher.NewFileWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	p.fileWatcher = fw

	// Create event channel
	eventChan := make(chan watcher.FileEvent, 100)

	// Watch all configured files
	for _, filePath := range p.config.WatchedFiles {
		if err := fw.Watch(filePath, eventChan); err != nil {
			p.logError(fmt.Sprintf("Failed to watch file: %s", filePath), err)
			continue
		}
		message := "watching " + filePath
		entry := logger.NewLogEntry(logger.LogLevelInfo, logger.LogEntryTypeFile, message)
		p.logEntry(entry)
	}

	// Start goroutine to handle file events
	go p.handleFileEvents(ctx, eventChan)

	return nil
}

// handleFileEvents processes file system events
func (p *Proxy) handleFileEvents(ctx context.Context, eventChan <-chan watcher.FileEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventChan:
			if event.Type == watcher.EventTypeCreated || event.Type == watcher.EventTypeDeleted {
				// Single line: "created <path>" or "deleted <path>"
				message := string(event.Type) + " " + event.Path
				entry := logger.NewLogEntry(logger.LogLevelInfo, logger.LogEntryTypeFile, message)
				p.logEntry(entry)
			} else if event.Type == watcher.EventTypeModified && event.Content != "" {
				// Modified with content: "modified <path>" with content in context
				message := "modified " + event.Path
				entry := logger.NewLogEntry(logger.LogLevelInfo, logger.LogEntryTypeFile, message)
				entry.Context = map[string]interface{}{
					"content": event.Content,
				}
				p.logEntry(entry)
			}
		}
	}
}
