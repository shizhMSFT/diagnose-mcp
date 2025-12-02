# MCP Protocol Message Schemas

**Purpose**: Define the structure of MCP protocol messages that diagnose-mcp intercepts and forwards

**Based On**: JSON-RPC 2.0 Specification + MCP Protocol Extension

---

## Base JSON-RPC 2.0 Structure

All MCP messages MUST include:
```json
{
  "jsonrpc": "2.0"
}
```

---

## 1. Request Message

**Structure**:
```json
{
  "jsonrpc": "2.0",
  "method": "<string>",
  "params": <any>,
  "id": <string|number|null>
}
```

**Fields**:
- `method` (required): String identifying the RPC method to invoke
- `params` (optional): Structured value holding parameter values (object or array)
- `id` (required): Request identifier (string, number, or null for no response expected)

**Example - MCP Tool Call**:
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "read_file",
    "arguments": {
      "path": "/tmp/data.json"
    }
  },
  "id": "req-123"
}
```

**Validation Rules**:
- `jsonrpc` MUST equal "2.0"
- `method` MUST be non-empty string
- `id` MUST be present (string, number, or null)
- If `id` is null, no response expected (notification-like request)

---

## 2. Response Message

**Success Response**:
```json
{
  "jsonrpc": "2.0",
  "result": <any>,
  "id": <string|number|null>
}
```

**Error Response**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": <number>,
    "message": "<string>",
    "data": <any>
  },
  "id": <string|number|null>
}
```

**Fields**:
- `result` (for success): The result of the method invocation
- `error` (for errors): Error object with code, message, and optional data
- `id` (required): MUST match the request `id` that triggered this response

**Example - Successful Tool Call Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"key\": \"value\"}"
      }
    ]
  },
  "id": "req-123"
}
```

**Example - Error Response**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "detail": "path parameter is required"
    }
  },
  "id": "req-123"
}
```

**Validation Rules**:
- MUST have either `result` or `error`, never both
- `id` MUST match a previous request
- Error `code` MUST be integer
- Error `message` MUST be string

---

## 3. Notification Message

**Structure**:
```json
{
  "jsonrpc": "2.0",
  "method": "<string>",
  "params": <any>
}
```

**Characteristics**:
- NO `id` field (distinguishes from request)
- Server-initiated (typically)
- No response expected

**Example - Logging Notification**:
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/message",
  "params": {
    "level": "info",
    "message": "Processing started"
  }
}
```

**Validation Rules**:
- MUST NOT have `id` field
- `method` MUST be non-empty string
- `params` optional

---

## 4. Progress Notification (MCP Extension)

**Structure**:
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "<string|number>",
    "progress": <number>,
    "total": <number>
  }
}
```

**Special Case**: Progress notifications use a specific method pattern

**Example - Tool Execution Progress**:
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "req-123",
    "progress": 50,
    "total": 100
  }
}
```

**Validation Rules**:
- NO `id` field (it's a notification)
- `method` typically contains "progress"
- `params.progressToken` links to originating request
- `progress` and `total` should be numeric

---

## Message Type Detection Algorithm

```
IF message has "id" field AND has "method" field:
    → Request
ELSE IF message has "id" field AND (has "result" OR "error"):
    → Response
ELSE IF message has "method" field AND NO "id":
    IF method contains "progress":
        → Progress Notification
    ELSE:
        → Notification
ELSE:
    → Invalid (log error, forward anyway per "transparent proxy" requirement)
```

---

## Transport-Specific Framing

### Stdio Transport (Local)
- **Delimiter**: Newline (`\n`)
- **Encoding**: UTF-8 JSON
- **Example**:
  ```
  {"jsonrpc":"2.0","method":"initialize","params":{...},"id":1}\n
  {"jsonrpc":"2.0","result":{...},"id":1}\n
  ```

### HTTP Transport (Remote - Server-Sent Events)
- **Content-Type**: `text/event-stream`
- **Format**: 
  ```
  data: {"jsonrpc":"2.0",...}\n\n
  ```

### WebSocket Transport (Remote)
- **Encoding**: UTF-8 JSON text frames
- **Framing**: One message per WebSocket frame
- **Example**: Each JSON object sent as single text frame

---

## Contract for diagnose-mcp

**Proxy Behavior**:
1. **Parse**: Attempt to parse incoming bytes as JSON
2. **Classify**: Detect message type (Request/Response/Notification/Progress)
3. **Log**: Create LogEntry with parsed structure
4. **Forward**: Send original bytes unchanged to destination
5. **Error Handling**: If parse fails, log as malformed but still forward

**Guarantees**:
- ✅ Message content NEVER modified
- ✅ Message order preserved
- ✅ Malformed messages logged and forwarded (transparency over validation)
- ✅ All four message types logged

**Performance**:
- Parsing overhead: <1ms per message (typical)
- Logging overhead: <100ms per message (async)
- Forwarding latency: <10ms p95 (constitution requirement)

---

## Examples of Real MCP Messages

### Initialize Handshake
```json
// Client → Server (Request)
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "roots": { "listChanged": true }
    },
    "clientInfo": {
      "name": "example-client",
      "version": "1.0.0"
    }
  },
  "id": 0
}

// Server → Client (Response)
{
  "jsonrpc": "2.0",
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {},
      "prompts": {}
    },
    "serverInfo": {
      "name": "example-server",
      "version": "1.0.0"
    }
  },
  "id": 0
}
```

### Tool Discovery
```json
// Client → Server (Request)
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "params": {},
  "id": 1
}

// Server → Client (Response)
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "read_file",
        "description": "Read file contents",
        "inputSchema": {
          "type": "object",
          "properties": {
            "path": { "type": "string" }
          },
          "required": ["path"]
        }
      }
    ]
  },
  "id": 1
}
```

---

**Contract Status**: ✅ **COMPLETE** - MCP protocol message schemas defined with validation rules and transport details
