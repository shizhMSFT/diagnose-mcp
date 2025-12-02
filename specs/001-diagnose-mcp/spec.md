# Feature Specification: diagnose-mcp Proxy Server

**Feature Branch**: `001-diagnose-mcp`  
**Created**: 2025-12-02  
**Status**: Draft  
**Input**: User description: "Build a MCP server named diagnose-mcp. It is a proxy MCP can proxy a local (stdio) or remote MCP server. For example, to proxy a local MCP server, the CLI UX would be diagnose-mcp [options] <local-mcp-server-binary> [local-mcp-server-options]. All proxied requests, including all the tool calling, should be logged in the server log and by default output to stdout. The diagnose-mcp should pass all environment variables to the proxied MCP server if it is a local MCP server. Meanwhile, the diagnose-mcp should have an ability to monitor text-based files with the paths specified by the user in the CLI options. Status of the files should be monitored (created, line-appended, deleted, etc.) and reported in the server log and by default output to stdout."

## Clarifications

### Session 2025-12-02

- Q: Which MCP protocol message types should diagnose-mcp log? → A: All MCP protocol messages including requests, responses, notifications, and progress updates

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Proxy Local MCP Server with Request Logging (Priority: P1)

A developer debugging an MCP server wants to see all requests and responses flowing through the MCP protocol. They run `diagnose-mcp --verbose my-mcp-server --server-option value` and observe all tool calls, prompts, responses, notifications, and progress updates logged to stdout in real-time, while the MCP client communicates transparently through the proxy.

**Why this priority**: Core value proposition - without transparent proxying and logging, the tool has no purpose. This is the minimum viable diagnostic capability.

**Independent Test**: Can be fully tested by running diagnose-mcp with any stdio-based MCP server, sending a simple tool call request, and verifying both the request/response logging and successful proxy behavior.

**Acceptance Scenarios**:

1. **Given** a local MCP server binary exists, **When** user runs `diagnose-mcp my-mcp-server`, **Then** diagnose-mcp starts, proxies all MCP protocol messages (requests, responses, notifications, progress updates) between client and server, and both client and server operate normally
2. **Given** diagnose-mcp is proxying a local MCP server, **When** an MCP client sends a tool call request, **Then** the full request JSON is logged to stdout with timestamp and direction indicator, forwarded to the target server, and the response is logged and returned to client
3. **Given** diagnose-mcp is proxying a local MCP server, **When** the server sends progress notifications during a long-running operation, **Then** all progress notifications are logged with timestamps and forwarded to the client
4. **Given** diagnose-mcp is running, **When** the target MCP server crashes or exits, **Then** diagnose-mcp logs the termination, cleanly shuts down, and exits with appropriate status code
5. **Given** a local MCP server requires environment variables, **When** user runs diagnose-mcp with those variables set, **Then** all environment variables are passed through to the proxied server

---

### User Story 2 - Proxy Remote MCP Server (Priority: P2)

A developer needs to diagnose an MCP server running on a remote machine or accessible via network transport. They run `diagnose-mcp --remote <connection-string>` and observe the same logging capabilities as local proxying, but for network-based MCP communication.

**Why this priority**: Extends diagnostic capability to remote servers, critical for production debugging scenarios. Builds on P1 infrastructure but adds network transport handling.

**Independent Test**: Can be tested independently by running diagnose-mcp against a network-accessible MCP server (HTTP/WebSocket endpoint), verifying request/response logging without requiring local server execution.

**Acceptance Scenarios**:

1. **Given** a remote MCP server is accessible via HTTP/WebSocket, **When** user runs `diagnose-mcp --remote https://example.com/mcp`, **Then** diagnose-mcp establishes connection, proxies all protocol messages, and logs them to stdout
2. **Given** diagnose-mcp is proxying a remote server, **When** network connection fails or times out, **Then** diagnose-mcp logs the error with connection details and retry attempts (if configured)
3. **Given** remote server requires authentication, **When** user provides credentials via CLI options, **Then** diagnose-mcp authenticates and maintains session throughout proxying

---

### User Story 3 - Monitor External Files During Session (Priority: P3)

A developer wants to correlate MCP server behavior with log files or state files the server writes. They run `diagnose-mcp --watch /path/to/server.log --watch /path/to/state.json my-mcp-server` and observe file change events (creation, line appends, deletion) logged alongside MCP protocol messages in chronological order.

**Why this priority**: Enhances diagnostic context by correlating MCP traffic with external file system activity. Valuable but not essential for basic proxy functionality.

**Independent Test**: Can be tested by running diagnose-mcp with file watch options, modifying watched files externally, and verifying file events appear in the log stream without requiring MCP traffic.

**Acceptance Scenarios**:

1. **Given** user specifies `--watch /path/to/file.log`, **When** diagnose-mcp starts, **Then** it monitors the file and logs its initial state (exists/size or not found)
2. **Given** a file is being watched, **When** new lines are appended to the file, **Then** diagnose-mcp logs the event with timestamp, file path, and number of lines added
3. **Given** a file is being watched, **When** the file is deleted, **Then** diagnose-mcp logs the deletion event and continues monitoring for recreation
4. **Given** a file is being watched, **When** the file is created (was not present at startup), **Then** diagnose-mcp logs the creation event with timestamp and initial size
5. **Given** multiple files are watched, **When** events occur simultaneously, **Then** all events are logged in chronological order interleaved with MCP protocol messages

---

### Edge Cases

- What happens when the proxied local MCP server binary doesn't exist or isn't executable?
- How does the system handle malformed MCP protocol messages from either client or server?
- What happens when stdout/stderr buffers fill up during high-volume logging?
- How does file watching behave with files on network drives or with permissions issues?
- What happens when the proxied server spawns child processes or forks?
- How does the system handle binary data in MCP messages vs. text-based logging?
- What happens when a watched file is truncated or completely rewritten?
- How does the proxy handle MCP protocol version mismatches between client and server?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a local MCP server binary path as a positional argument and execute it as a child process with stdio transport
- **FR-002**: System MUST pass all command-line arguments following the binary path directly to the proxied server
- **FR-003**: System MUST pass all environment variables from the diagnose-mcp process to the proxied local MCP server
- **FR-004**: System MUST intercept and log all MCP protocol messages (requests, responses, notifications, and progress updates) in both directions between client and server
- **FR-005**: System MUST output logs to stdout by default in a structured, timestamped format
- **FR-006**: System MUST forward MCP protocol messages transparently without modifying content or introducing protocol errors
- **FR-007**: System MUST support a `--remote <url>` option to proxy a remote MCP server via HTTP/WebSocket transport
- **FR-008**: System MUST accept one or more `--watch <file-path>` options to monitor text-based files
- **FR-009**: System MUST detect and log file events: creation, deletion, and line appends for watched files
- **FR-010**: System MUST interleave file event logs with MCP protocol logs in chronological order
- **FR-011**: System MUST provide a `--verbose` flag to control log verbosity (detailed vs. summary logging)
- **FR-012**: System MUST output logs in both human-readable format (default) and JSON format (when `--json` flag specified)
- **FR-013**: System MUST handle graceful shutdown when receiving SIGTERM/SIGINT, closing proxy connections cleanly
- **FR-014**: System MUST log errors separately to stderr while maintaining MCP protocol on stdout/stdin
- **FR-015**: System MUST exit with non-zero status code when the proxied server exits with error or crashes

### Key Entities

- **MCP Message**: Represents a single MCP protocol request, response, notification, or progress update, containing message type, direction (client→server or server→client), timestamp, and payload content
- **Proxy Session**: Represents an active proxying session with connection details (local vs remote), target server information, start time, and configuration (verbosity, output format, watched files)
- **File Watch**: Represents a monitored file with path, current state (exists/size/line count), and event history (creation, modifications, deletions)
- **Log Entry**: Represents a single diagnostic log line with timestamp, entry type (MCP message, file event, system event), severity, and formatted content

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully proxy any stdio-based MCP server with zero protocol errors introduced by the proxy (verified via protocol conformance tests)
- **SC-002**: All MCP tool calls, prompts, responses, notifications, and progress updates are logged with <100ms overhead compared to direct client-server communication
- **SC-003**: Proxy maintains <10ms p95 latency overhead for message passthrough as specified in constitution
- **SC-004**: File change events are detected and logged within 500ms of the actual file system change
- **SC-005**: System handles proxy sessions exceeding 1 hour without memory leaks (memory footprint stays <100MB per constitution)
- **SC-006**: 100% of environment variables are correctly passed through to local proxied servers
- **SC-007**: Structured logs (JSON mode) are parsable by standard log analysis tools without errors
- **SC-008**: Users can diagnose common MCP server issues (tool call failures, timeout errors, malformed responses) by reading logs without needing source code access

## Assumptions *(included when reasonable defaults chosen)*

- **A-001**: MCP protocol uses JSON-RPC 2.0 format over stdio (local) or HTTP/WebSocket (remote) - standard MCP assumption
- **A-002**: Watched files are text-based with line-oriented content (not binary files)
- **A-003**: File watching uses polling or OS-native file system events (inotify/FSEvents/ReadDirectoryChangesW) depending on platform
- **A-004**: Default log format is human-readable text; JSON format requires explicit `--json` flag
- **A-005**: Network timeouts default to 30s connect, 5m read per constitution performance standards
- **A-006**: Proxy operates as a transparent man-in-the-middle; does not validate or transform MCP message content beyond logging
- **A-007**: Child process (local server) stdout/stderr are captured separately from MCP protocol stdio
- **A-008**: Multiple concurrent file watches are supported (reasonable limit: 100 files)

## Constraints

- **C-001**: Must maintain MCP protocol compatibility - no breaking changes to message format or transport
- **C-002**: Performance overhead must comply with constitution: <10ms p95 latency, <100MB memory
- **C-003**: Must not require modification of proxied MCP servers - works with any compliant server
- **C-004**: Log output must be structured enough for automated parsing while remaining human-readable
- **C-005**: Must handle both Unix and Windows platforms (path separators, process handling, signals)

## Out of Scope *(explicitly excluded)*

- **OS-001**: Message replay or modification capabilities (pure proxy, no traffic manipulation)
- **OS-002**: Performance profiling or resource usage monitoring of proxied server (focus on protocol diagnostics only)
- **OS-003**: GUI or web-based log viewer (CLI-only per constitution)
- **OS-004**: Persistent log storage to disk (output to stdout; users pipe to file if needed)
- **OS-005**: Multi-server proxying or load balancing (single proxy session per invocation)
- **OS-006**: Authentication/authorization enforcement (passes through any auth; doesn't validate)

## Dependencies

- **D-001**: MCP protocol specification (for message format understanding and transport requirements)
- **D-002**: OS-specific file system event APIs (inotify on Linux, FSEvents on macOS, ReadDirectoryChangesW on Windows)
- **D-003**: Child process management libraries for stdio redirection and signal handling
- **D-004**: HTTP/WebSocket client libraries for remote server proxying (if not in standard library)

## Risks & Mitigations

- **R-001**: **Risk**: High-volume MCP traffic or large messages may cause logging to fall behind, introducing backpressure  
  **Mitigation**: Implement async logging with bounded queues; drop log entries under extreme load with counter of dropped messages

- **R-002**: **Risk**: Binary data in MCP messages may break text-based logging assumptions  
  **Mitigation**: ✅ IMPLEMENTED - Payloads automatically shown as readable text if printable UTF-8, or base64-encoded if binary

- **R-003**: **Risk**: File watching may miss rapid successive changes if filesystem event granularity is coarse  
  **Mitigation**: Document limitation; use checksums or line counts to detect missed intermediate states

- **R-004**: **Risk**: Proxied server may not flush stdout, causing protocol messages to be buffered and delayed  
  **Mitigation**: Document requirement for servers to flush stdio; detect and log warning if buffering detected
