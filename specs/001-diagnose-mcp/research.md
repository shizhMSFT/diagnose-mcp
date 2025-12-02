# Research: diagnose-mcp Technical Decisions

**Phase**: 0 - Research & Unknowns Resolution  
**Date**: 2025-12-02  
**Purpose**: Resolve all NEEDS CLARIFICATION items from Technical Context and validate technology choices

## Research Tasks

### 1. MCP Protocol Specification

**Question**: What is the exact MCP protocol message format, transport requirements, and message types?

**Research Findings**:
- **Protocol**: Model Context Protocol (MCP) uses JSON-RPC 2.0 format
- **Message Types**: 
  - Requests: `method`, `params`, `id` fields
  - Responses: `result` or `error`, `id` fields
  - Notifications: `method`, `params` fields (no `id`)
  - Progress Updates: Special notification type for long-running operations
- **Transport**: 
  - Local: stdio (JSON messages delimited by newlines)
  - Remote: HTTP with Server-Sent Events (SSE) or WebSocket
- **Version Negotiation**: `initialize` request/response at session start

**Decision**: Implement JSON-RPC 2.0 parser in `pkg/mcp/parser.go` with support for all four message types. Use `encoding/json` standard library for parsing. Newline-delimited JSON for stdio transport.

**Rationale**: JSON-RPC 2.0 is well-specified, standard library JSON support is sufficient, newline delimiting is simple and robust.

**Alternatives Considered**:
- Custom binary protocol → Rejected: MCP spec requires JSON-RPC
- Third-party JSON-RPC library → Rejected: Simple enough to implement directly with standard library

---

### 2. Go Libraries for WebSocket Client

**Question**: Which WebSocket library provides reliable client support for remote MCP servers?

**Research Findings**:
- **gorilla/websocket**: Most popular Go WebSocket library, mature (10+ years), excellent documentation
  - Version: v1.5.1 (stable)
  - Features: Client/server support, compression, connection upgrade
  - License: BSD-2-Clause (permissive)
  - Maintenance: Active, widely used (50k+ stars on GitHub)
- **nhooyr.io/websocket**: Modern alternative, simpler API
  - Version: v1.8.10
  - Features: Context-based cancellation, cleaner API
  - License: ISC (permissive)
  - Cons: Less mature, smaller ecosystem

**Decision**: Use `github.com/gorilla/websocket` v1.5.1

**Rationale**: Proven stability, extensive documentation, large community for troubleshooting, comprehensive feature set including ping/pong for connection health.

**Alternatives Considered**:
- nhooyr.io/websocket → Rejected: Prefer battle-tested library for production diagnostic tool
- Standard library (net/http upgrade) → Rejected: Too low-level, would reinvent gorilla features

---

### 3. File System Watching (fsnotify)

**Question**: How to efficiently watch multiple files across Windows/macOS/Linux with minimal latency?

**Research Findings**:
- **fsnotify/fsnotify**: Cross-platform file system notifications
  - Version: v1.7.0 (latest stable)
  - Backends: inotify (Linux), FSEvents (macOS), ReadDirectoryChangesW (Windows)
  - Features: Create, Write, Remove, Rename, Chmod events
  - Latency: <100ms typical (OS-dependent)
  - Limitations: Doesn't auto-detect line appends (only file Write events)
- **Alternative - Polling**: Check file mtime/size periodically
  - Pros: Simple, no OS dependencies
  - Cons: Higher latency (100ms+ poll interval), higher CPU usage

**Decision**: Use `github.com/fsnotify/fsnotify` v1.7.0 with custom line-counting logic

**Rationale**: Native OS event APIs provide lowest latency (<500ms requirement). For "line append" detection, will track file size and read new bytes to count newlines.

**Implementation Approach**:
```
On Write event:
  1. Get new file size
  2. Seek to previous size position
  3. Read new bytes
  4. Count newlines in delta
  5. Log: "File X: +N lines appended"
```

**Alternatives Considered**:
- Polling every 100ms → Rejected: Wasteful CPU, inconsistent latency
- tail-like implementation → Rejected: Complexity not needed, fsnotify + size tracking sufficient

---

### 4. Structured Logging (slog)

**Question**: Which logging library provides structured logs with JSON output and minimal overhead?

**Research Findings**:
- **log/slog**: Go standard library structured logging (added in Go 1.21)
  - Features: Levels (Debug, Info, Warn, Error), attributes, handlers (JSON, Text)
  - Performance: <1μs per log call (benchmarks show minimal overhead)
  - Integration: Native, no external dependencies
- **zerolog**: Third-party, zero-allocation logging
  - Pros: Fastest (claims zero allocation)
  - Cons: External dependency, JSON-only (no human-readable)
- **zap**: Uber's structured logger
  - Pros: Fast, flexible
  - Cons: Complex API, external dependency

**Decision**: Use `log/slog` from Go standard library

**Rationale**: Standard library is sufficient, performance meets <100ms requirement (1μs ≪ 100ms), supports both JSON and human-readable text output, native integration with Go ecosystem.

**Output Format Examples**:
```
# Human-readable (default)
2025-12-02T14:30:45.123Z INFO [C->S] tool_call method=list_files id=123

# JSON (--json flag)
{"time":"2025-12-02T14:30:45.123Z","level":"INFO","dir":"C->S","type":"request","method":"list_files","id":"123"}
```

**Alternatives Considered**:
- zerolog → Rejected: Premature optimization, slog performance adequate
- zap → Rejected: Complexity not justified, slog simpler API

---

### 5. Child Process Management (os/exec)

**Question**: How to launch local MCP server, capture stdio, pass environment, handle signals?

**Research Findings**:
- **os/exec.Command**: Standard library process execution
  - Features: Set stdin/stdout/stderr, environment variables, working directory
  - Signal handling: Process groups for clean termination
- **Pattern for stdio proxying**:
  ```go
  cmd := exec.Command(binary, args...)
  cmd.Env = os.Environ() // Pass all environment
  cmd.Stdin = clientStdin  // Pipe from MCP client
  cmd.Stdout = toClient    // Pipe to MCP client (intercepted)
  cmd.Stderr = os.Stderr   // Server errors to our stderr
  ```
- **Signal forwarding**: Intercept SIGTERM/SIGINT, forward to child, wait for graceful exit

**Decision**: Use `os/exec.Command` with custom stdio piping for interception

**Rationale**: Standard library sufficient, well-documented pattern for subprocess management, supports all requirements (env passing, stdio redirection, signal handling).

**Implementation Notes**:
- Use `io.TeeReader` to intercept and log messages while forwarding
- Set up process group to ensure child cleanup on parent termination
- Context-based cancellation for graceful shutdown

**Alternatives Considered**:
- Third-party process managers → Rejected: os/exec handles all requirements
- Shell invocation (`sh -c`) → Rejected: Cross-platform issues, unnecessary layer

---

### 6. Signal Handling (os/signal)

**Question**: How to handle SIGTERM/SIGINT for graceful shutdown on Windows and Unix?

**Research Findings**:
- **os/signal**: Standard library signal handling
- **Cross-platform support**:
  - Unix: SIGTERM, SIGINT, SIGHUP
  - Windows: os.Interrupt (Ctrl+C), os.Kill
- **Pattern**:
  ```go
  sigChan := make(chan os.Signal, 1)
  signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
  <-sigChan // Block until signal
  // Cleanup: close connections, stop goroutines, wait for child
  ```

**Decision**: Use `os/signal.Notify` with context cancellation pattern

**Rationale**: Standard approach, works across platforms, integrates with context-based goroutine cancellation.

**Graceful Shutdown Sequence**:
1. Receive signal → Cancel context
2. Stop accepting new messages
3. Drain in-flight messages (max 5s timeout)
4. Signal child process to terminate (SIGTERM)
5. Wait for child exit (max 10s)
6. Flush logs
7. Exit with appropriate code

**Alternatives Considered**:
- Immediate termination → Rejected: Violates constitution requirement for graceful shutdown
- Third-party shutdown libraries → Rejected: Simple enough with stdlib

---

## Summary of Decisions

| Area | Technology | Version | Rationale |
|------|-----------|---------|-----------|
| MCP Protocol | JSON-RPC 2.0 (stdlib encoding/json) | Go 1.25 | Standard, sufficient performance |
| WebSocket | gorilla/websocket | v1.5.1 | Mature, battle-tested, comprehensive |
| File Watching | fsnotify/fsnotify | v1.7.0 | Cross-platform, low latency |
| Logging | log/slog (stdlib) | Go 1.21+ | Standard, performant, flexible |
| Process Mgmt | os/exec (stdlib) | Go 1.25 | Sufficient, well-documented |
| Signal Handling | os/signal (stdlib) | Go 1.25 | Standard, cross-platform |

## Best Practices Identified

### Go Idioms for Proxy Implementation
- **Goroutines per connection**: Separate goroutines for client→server and server→client forwarding
- **Context propagation**: Use context.Context for cancellation across goroutines
- **Defer cleanup**: Use `defer` for connection/file closing
- **Error groups**: Use `errgroup` for coordinating multiple goroutines

### Testing Strategy
- **Table-driven tests**: Go convention for testing multiple scenarios
- **Test fixtures**: Mock MCP servers for integration tests (in-memory buffers for stdio)
- **Benchmarks**: Use `testing.B` for performance validation (<10ms latency)
- **Race detector**: Run tests with `-race` flag to catch concurrency issues

### Performance Optimization
- **Buffer sizes**: 8KB buffers for stdio reading (typical OS page size)
- **Minimal allocations**: Reuse buffers where possible
- **Avoid reflection**: Use json.Decoder streaming for large messages
- **Bounded queues**: Limit log queue size to prevent memory growth under load

## Open Questions & Mitigations

**Q**: What if MCP servers use chunked/streaming JSON instead of newline-delimited?  
**A**: Start with newline-delimited (simplest), monitor real-world usage, add chunked support if needed (Assumption A-001 documents this choice)

**Q**: How to handle binary data in JSON payloads (e.g., base64-encoded)?  
**A**: JSON parser handles strings natively, base64 decoding not needed for logging (Risk R-002 mitigation already planned)

**Q**: What if file watching misses events due to rapid changes?  
**A**: Document limitation, use file size checksums to detect missed states (Risk R-003 mitigation already planned)

---

**Phase 0 Status**: ✅ **COMPLETE** - All NEEDS CLARIFICATION items resolved, technology stack validated, best practices identified.

**Next Step**: Proceed to Phase 1 (Design & Contracts) - Generate data-model.md, contracts/, quickstart.md
