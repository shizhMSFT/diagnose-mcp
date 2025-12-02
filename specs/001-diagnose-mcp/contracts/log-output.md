# Log Output Format Contract

**Purpose**: Define the structure and format of diagnostic logs output by diagnose-mcp to stdout

---

## Output Modes

### Text Mode (Default)
- **Human-readable format**
- **Target audience**: Developers reading logs in terminal
- **Activation**: Default (or explicit `--no-json`)

### JSON Mode
- **Machine-parsable format**
- **Target audience**: Log analysis tools, grep, jq
- **Activation**: `--json` flag

---

## Text Format Specification

### Format Template
```
<timestamp> <level> [<context>] <message>
```

**Components**:
- `<timestamp>`: ISO 8601 format with milliseconds (UTC)
  - Example: `2025-12-02T14:30:45.123Z`
- `<level>`: Log level in uppercase
  - Values: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `<context>`: Context indicator in brackets
  - For MCP messages: Direction (`C->S` or `S->C`)
  - For file events: `FILE`
  - For system events: `SYSTEM`
- `<message>`: Human-readable description

### Text Format Examples

#### MCP Request (Summary Mode - default)
```
2025-12-02T14:30:45.123Z INFO [C→S] REQUEST id=req-123 method=tools/call
```

#### MCP Request (Verbose Mode - with --verbose)
```
2025-12-02T14:30:45.123Z INFO [C->S] REQUEST id=req-123 method=tools/call params={"name":"read_file","arguments":{"path":"/tmp/data.json"}}
```

#### MCP Response (Summary)
```
2025-12-02T14:30:45.234Z INFO [S->C] RESPONSE id=req-123 (success)
```

#### MCP Response (Verbose)
```
2025-12-02T14:30:45.234Z INFO [S->C] RESPONSE id=req-123 result={"content":[{"type":"text","text":"..."}]}
```

#### MCP Notification
```
2025-12-02T14:30:46.000Z INFO [S->C] NOTIFICATION method=notifications/message params={"level":"info","message":"Processing started"}
```

#### MCP Progress Notification
```
2025-12-02T14:30:47.000Z INFO [S->C] PROGRESS token=req-123 progress=50/100
```

#### File Event - Created
```
2025-12-02T14:30:48.000Z INFO [FILE] /tmp/server.log: created (size=0)
```

#### File Event - Line Appended
```
2025-12-02T14:30:49.000Z INFO [FILE] /tmp/server.log: +5 lines appended (total: 42 lines)
```

#### File Event - Deleted
```
2025-12-02T14:30:50.000Z INFO [FILE] /tmp/server.log: deleted
```

#### System Event - Startup
```
2025-12-02T14:30:40.000Z INFO [SYSTEM] Proxy started: local server ./my-server
```

#### System Event - Shutdown
```
2025-12-02T14:31:00.000Z INFO [SYSTEM] Shutting down (signal: SIGTERM)
```

#### Error
```
2025-12-02T14:30:51.000Z ERROR [SYSTEM] Failed to parse MCP message: invalid JSON at offset 42
```

---

## JSON Format Specification

### Base Structure
```json
{
  "time": "<ISO8601>",
  "level": "<DEBUG|INFO|WARN|ERROR>",
  "type": "<mcp_message|file_event|system_event|error>",
  ...type-specific fields...
}
```

### JSON Format Examples

#### MCP Request
```json
{
  "time": "2025-12-02T14:30:45.123Z",
  "level": "INFO",
  "type": "mcp_message",
  "direction": "client_to_server",
  "message_type": "request",
  "id": "req-123",
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {"path": "/tmp/data.json"}
  }
}
```

**Note**: `params` field only present in verbose mode

#### MCP Response (Success)
```json
{
  "time": "2025-12-02T14:30:45.234Z",
  "level": "INFO",
  "type": "mcp_message",
  "direction": "server_to_client",
  "message_type": "response",
  "id": "req-123",
  "status": "success",
  "result": {
    "content": [{"type": "text", "text": "..."}]
  }
}
```

#### MCP Response (Error)
```json
{
  "time": "2025-12-02T14:30:45.234Z",
  "level": "WARN",
  "type": "mcp_message",
  "direction": "server_to_client",
  "message_type": "response",
  "id": "req-123",
  "status": "error",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {"detail": "path parameter is required"}
  }
}
```

#### MCP Notification
```json
{
  "time": "2025-12-02T14:30:46.000Z",
  "level": "INFO",
  "type": "mcp_message",
  "direction": "server_to_client",
  "message_type": "notification",
  "method": "notifications/message",
  "params": {
    "level": "info",
    "message": "Processing started"
  }
}
```

#### MCP Progress Notification
```json
{
  "time": "2025-12-02T14:30:47.000Z",
  "level": "INFO",
  "type": "mcp_message",
  "direction": "server_to_client",
  "message_type": "progress",
  "progress_token": "req-123",
  "progress": 50,
  "total": 100
}
```

#### File Event - Line Appended
```json
{
  "time": "2025-12-02T14:30:49.000Z",
  "level": "INFO",
  "type": "file_event",
  "event": "line_appended",
  "path": "/tmp/server.log",
  "delta": 5,
  "total_lines": 42
}
```

#### File Event - Created
```json
{
  "time": "2025-12-02T14:30:48.000Z",
  "level": "INFO",
  "type": "file_event",
  "event": "created",
  "path": "/tmp/server.log",
  "size": 0
}
```

#### File Event - Deleted
```json
{
  "time": "2025-12-02T14:30:50.000Z",
  "level": "INFO",
  "type": "file_event",
  "event": "deleted",
  "path": "/tmp/server.log"
}
```

#### System Event
```json
{
  "time": "2025-12-02T14:30:40.000Z",
  "level": "INFO",
  "type": "system_event",
  "event": "proxy_started",
  "details": {
    "mode": "local",
    "target": "./my-server",
    "watched_files": ["/tmp/server.log"]
  }
}
```

#### Error
```json
{
  "time": "2025-12-02T14:30:51.000Z",
  "level": "ERROR",
  "type": "error",
  "message": "Failed to parse MCP message",
  "details": {
    "error": "invalid JSON at offset 42",
    "raw_input": "...truncated..."
  }
}
```

---

## Field Specifications

### Common Fields (All Log Types)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `time` | string (ISO 8601) | Yes | Timestamp when log entry created (UTC) |
| `level` | string enum | Yes | Log level: DEBUG, INFO, WARN, ERROR |
| `type` | string enum | Yes | Entry type: mcp_message, file_event, system_event, error |

### MCP Message Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `direction` | string enum | Yes | client_to_server or server_to_client |
| `message_type` | string enum | Yes | request, response, notification, progress |
| `id` | string/number | For req/res | JSON-RPC message ID |
| `method` | string | For req/notif | RPC method name |
| `params` | object | Verbose only | Request/notification parameters |
| `result` | any | For success | Response result |
| `error` | object | For errors | Error object (code, message, data) |
| `status` | string enum | For responses | success or error |
| `progress_token` | string/number | For progress | Token linking to originating request |
| `progress` | number | For progress | Current progress value |
| `total` | number | For progress | Total progress value |

### File Event Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event` | string enum | Yes | created, modified, deleted, line_appended |
| `path` | string | Yes | Absolute path to file |
| `size` | number | For created | File size in bytes |
| `delta` | number | For line_appended | Number of lines added |
| `total_lines` | number | For line_appended | Total line count after append |

### System Event Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event` | string | Yes | Event name (proxy_started, shutting_down, etc.) |
| `details` | object | Optional | Event-specific details |

### Error Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `message` | string | Yes | Human-readable error message |
| `details` | object | Optional | Error details (stack trace, context) |

---

## Verbosity Levels

### Summary Mode (Default)
- **MCP Messages**: Log method, id, direction, status only
- **File Events**: Log event type, path, delta only
- **System Events**: Log event name only
- **Params/Result**: NOT logged

### Verbose Mode (`--verbose`)
- **MCP Messages**: Log full params and result fields
  - Payloads shown as readable text if printable UTF-8
  - Binary payloads automatically base64-encoded
- **File Events**: Same as summary (already detailed)
- **System Events**: Include configuration details
- **Truncation**: Large payloads (>1KB) truncated with indicator

**Example Truncation**:
```json
{
  "params": {"data": "...truncated (5.2KB)..."}
}
```

---

## Performance Characteristics

**Logging Overhead**:
- Text formatting: <1ms per entry
- JSON marshaling: <2ms per entry
- Async buffering: 1000-entry queue before backpressure

**Backpressure Behavior**:
- When log queue full: Drop oldest entries
- Emit counter: `WARN [SYSTEM] Dropped 42 log entries due to backpressure`

---

## Compatibility

**JSON Output Compatibility**:
- Each line is valid, standalone JSON object
- Compatible with `jq`, `grep`, log analysis tools
- Example pipeline:
  ```bash
  diagnose-mcp --json my-server | jq 'select(.type == "mcp_message" and .method == "tools/call")'
  ```

**Text Output Compatibility**:
- Grep-friendly: All important info on single line
- Example:
  ```bash
  diagnose-mcp my-server | grep "ERROR"
  diagnose-mcp my-server | grep "\[C->S\]"
  ```

---

**Contract Status**: ✅ **COMPLETE** - Log output formats fully specified for both text and JSON modes with all field definitions
