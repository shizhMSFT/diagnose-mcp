# Data Model: diagnose-mcp

**Phase**: 1 - Design & Contracts  
**Date**: 2025-12-02  
**Purpose**: Define core entities, their attributes, relationships, and validation rules

## Entity Definitions

### 1. MCPMessage

**Purpose**: Represents a single MCP protocol message (request, response, notification, or progress update)

**Attributes**:
- `Timestamp` (time.Time): When message was intercepted by proxy
- `Direction` (MessageDirection enum): `ClientToServer` or `ServerToClient`
- `MessageType` (MessageType enum): `Request`, `Response`, `Notification`, `ProgressUpdate`
- `RawJSON` ([]byte): Original message bytes for transparent forwarding
- `ParsedData` (interface{}): Parsed JSON structure for logging
  - For requests: `{method: string, params: any, id: any}`
  - For responses: `{result: any, error: any, id: any}`
  - For notifications: `{method: string, params: any}`
  - For progress: `{method: "$/progress", params: {token: any, value: any}}`

**Validation Rules**:
- `RawJSON` MUST be valid JSON
- `Direction` MUST be one of the enum values
- `MessageType` MUST match JSON-RPC structure (presence of `id`, `result`/`error`, `method`)
- Requests and responses MUST have `id` field
- Notifications MUST NOT have `id` field
- All messages MUST have `jsonrpc: "2.0"` field

**State Transitions**: N/A (immutable value object)

**Relationships**:
- Belongs to one ProxySession
- May be referenced by LogEntry

---

### 2. ProxySession

**Purpose**: Represents an active proxying session with configuration and runtime state

**Attributes**:
- `SessionID` (string): Unique identifier (UUID)
- `StartTime` (time.Time): When session began
- `ConnectionType` (ConnectionType enum): `Local` (stdio) or `Remote` (HTTP/WebSocket)
- `TargetInfo` (string): Local binary path or remote URL
- `Config` (SessionConfig): User configuration
  - `VerboseLogging` (bool): Detailed vs. summary logging
  - `OutputFormat` (OutputFormat enum): `Text` or `JSON`
  - `WatchedFiles` ([]string): File paths to monitor
- `Stats` (SessionStats): Runtime statistics
  - `MessagesForwarded` (int64): Total message count
  - `BytesForwarded` (int64): Total bytes proxied
  - `ErrorCount` (int64): Protocol errors encountered
  - `LastMessageTime` (time.Time): Last activity timestamp

**Validation Rules**:
- `ConnectionType` MUST be `Local` or `Remote`
- For `Local`: `TargetInfo` MUST be valid executable path
- For `Remote`: `TargetInfo` MUST be valid HTTP/HTTPS/WS/WSS URL
- `WatchedFiles` paths MUST be absolute (or relative to cwd)
- `OutputFormat` MUST be `Text` or `JSON`

**State Transitions**:
```
Initializing → Active → Shutting Down → Terminated
```
- `Initializing`: Session created, connections not yet established
- `Active`: Actively forwarding messages
- `Shutting Down`: Signal received, draining in-flight messages
- `Terminated`: All connections closed, child process exited

**Relationships**:
- Has many MCPMessage instances (lifetime of session)
- Has many FileWatch instances (one per watched file)
- Has many LogEntry instances (lifetime of session)

---

### 3. FileWatch

**Purpose**: Represents a monitored file with current state and event history

**Attributes**:
- `FilePath` (string): Absolute path to monitored file
- `State` (FileState struct):
  - `Exists` (bool): Whether file currently exists
  - `Size` (int64): Current file size in bytes
  - `Offset` (int64): Last read position for tail-like behavior
  - `LastModified` (time.Time): Last modification timestamp
- `EventHistory` ([]FileEvent): Recent events (ring buffer, max 100 entries)

**FileEvent Structure**:
- `Timestamp` (time.Time): When event occurred
- `EventType` (FileEventType enum): `Created`, `Modified`, `Deleted`
- `Path` (string): File path
- `Content` (string): New content added (for Modified events, up to 100KB)
- `Size` (int64): Current file size after event

**Validation Rules**:
- `FilePath` MUST be non-empty
- `FilePath` MUST be absolute or resolvable relative path
- `Size` and `Offset` MUST be non-negative
- `Offset` MUST be ≤ `Size`
- `EventHistory` bounded to prevent unbounded memory growth

**State Transitions**:
```
NotExists → Exists (Created event)
Exists → NotExists (Deleted event)
Exists → Exists (Modified events with content changes)
```

**Relationships**:
- Belongs to one ProxySession
- Generates LogEntry instances for each event

---

### 4. LogEntry

**Purpose**: Represents a single diagnostic log line for output to stdout/stderr

**Attributes**:
- `Timestamp` (time.Time): When log entry was created
- `EntryType` (LogEntryType enum): `MCPMessage`, `FileEvent`, `SystemEvent`, `Error`
- `Severity` (LogLevel enum): `Debug`, `Info`, `Warn`, `Error`
- `Direction` (MessageDirection, optional): For `MCPMessage` type only
- `Content` (interface{}): Type-specific content
  - For `MCPMessage`: MCPMessage struct
  - For `FileEvent`: FileEvent struct
  - For `SystemEvent`: string (e.g., "Proxy started", "Child process exited")
  - For `Error`: error struct with stack trace
- `FormattedOutput` (string): Cached formatted string (lazy-computed)

**Validation Rules**:
- `EntryType` MUST match `Content` type
- `Severity` MUST be one of the defined log levels
- `MCPMessage` entries MUST have `Direction` set
- `FormattedOutput` length MUST be <10KB (prevent stdout flooding)

**Formatting**:
```
# Text format (human-readable, single line)
2025-12-02T14:30:45.123Z [INFO] [request] id=123 method=list_files
2025-12-02T14:30:45.234Z [INFO] [response] id=123 
2025-12-02T14:30:45.345Z [INFO] [file] created /tmp/server.log
2025-12-02T14:30:45.456Z [INFO] [file] modified /tmp/server.log
  Context: content="new log line\n"

# JSON format (machine-parsable)
{"time":"2025-12-02T14:30:45.123Z","level":"INFO","type":"request","message":"id=123 method=list_files"}
{"time":"2025-12-02T14:30:45.234Z","level":"INFO","type":"response","message":"id=123"}
{"time":"2025-12-02T14:30:45.345Z","level":"INFO","type":"file","message":"created /tmp/server.log"}
{"time":"2025-12-02T14:30:45.456Z","level":"INFO","type":"file","message":"modified /tmp/server.log","context":{"content":"new log line\n"}}
```

**Relationships**:
- Belongs to one ProxySession
- May reference one MCPMessage or one FileEvent

---

## Enumerations

### MessageDirection
```go
type MessageDirection int

const (
    ClientToServer MessageDirection = iota  // C→S
    ServerToClient                          // S→C
)
```

### MessageType
```go
type MessageType int

const (
    Request MessageType = iota
    Response
    Notification
    ProgressUpdate  // Special case of Notification
)
```

### ConnectionType
```go
type ConnectionType int

const (
    Local ConnectionType = iota   // stdio-based
    Remote                         // HTTP/WebSocket
)
```

### OutputFormat
```go
type OutputFormat int

const (
    Text OutputFormat = iota  // Human-readable
    JSON                      // Machine-parsable
)
```

### FileEventType
```go
type FileEventType int

const (
    FileCreated FileEventType = iota
    FileModified
    FileDeleted
)
```

### LogEntryType
```go
type LogEntryType int

const (
    MCPMessageLog LogEntryType = iota
    FileEventLog
    SystemEventLog
    ErrorLog
)
```

### LogLevel
```go
type LogLevel int

const (
    Debug LogLevel = iota
    Info
    Warn
    Error
)
```

---

## Entity Relationships Diagram

```
ProxySession (1)
    ├── has many (*)   MCPMessage
    ├── has many (*)   FileWatch
    │   └── generates (*) LogEntry (via FileEvent)
    └── has many (*)   LogEntry
        └── references (0..1) MCPMessage
```

---

## Validation Rules Summary

| Entity | Key Rules |
|--------|-----------|
| MCPMessage | Valid JSON, correct jsonrpc version, type matches structure |
| ProxySession | Valid connection type, target info format matches type |
| FileWatch | Absolute path, non-negative sizes, bounded event history |
| LogEntry | Type matches content, reasonable output length |

---

## Concurrency Considerations

**Thread Safety Requirements**:
- `ProxySession.Stats`: Use atomic operations (sync/atomic) for counters
- `FileWatch.State`: Protect with mutex (sync.RWMutex) - frequent reads, rare writes
- `LogEntry` queue: Use buffered channel with backpressure handling

**Immutability**:
- `MCPMessage`: Immutable after creation (safe to share across goroutines)
- `FileEvent`: Immutable after creation
- `LogEntry`: Immutable after creation

---

**Phase 1 Data Model Status**: ✅ **COMPLETE** - All entities defined with attributes, validation rules, relationships, and concurrency considerations.

**Next Step**: Define contracts (MCP protocol schemas, CLI interface, log formats)
