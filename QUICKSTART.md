# Quickstart Guide

Get started with `diagnose-mcp` in 5 minutes!

## Installation

### Option 1: Build from source (recommended)

```bash
git clone https://github.com/shizhMSFT/diagnose-mcp.git
cd diagnose-mcp
go build -o diagnose-mcp ./cmd/diagnose-mcp
```

### Option 2: Install with go install

```bash
go install github.com/shizhMSFT/diagnose-mcp/cmd/diagnose-mcp@latest
```

## Basic Usage

### 1. Verify installation

```bash
diagnose-mcp --version
# Output: diagnose-mcp version 1.0.0

diagnose-mcp --help
# Shows usage information
```

### 2. Proxy a local MCP server

Create a simple test server (`test-server.go`):

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

type Message struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var msg Message
        json.Unmarshal(scanner.Bytes(), &msg)
        
        response := Message{
            JSONRPC: "2.0",
            ID:      msg.ID,
            Result:  json.RawMessage(`{"status":"ok"}`),
        }
        
        data, _ := json.Marshal(response)
        fmt.Println(string(data))
    }
}
```

Run the proxy:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  diagnose-mcp go run test-server.go
```

**Output (on stderr - the logs):**
```
2025-12-02T10:30:45.123Z [INFO] [proxy] Proxy session starting
2025-12-02T10:30:45.150Z [INFO] [proxy] Server started
  Context: binary=go, args=[run test-server.go], pid=12345
2025-12-02T10:30:45.200Z [INFO] [request] → initialize #1
2025-12-02T10:30:45.250Z [INFO] [response] ← #1
2025-12-02T10:30:45.300Z [INFO] [proxy] Proxy session ended
  Context: duration=150ms, message_count=2, error_count=0
```

**Output (on stdout - the MCP response):**
```json
{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}
```

### 3. Use verbose mode

See full message payloads:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}' | \
  diagnose-mcp --verbose go run test-server.go
```

Logs now include readable payloads:
```
2025-12-02T10:30:45.200Z [INFO] [request] → test #1
  Payload: {"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}
```

**Note**: Payloads are shown as readable text if they contain printable characters, or as base64 encoding for binary data.

### 4. Use JSON output format

For machine-parsable logs:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"test"}' | \
  diagnose-mcp --json go run test-server.go 2>&1 | grep '^\{'
```

**Output:**
```json
{"timestamp":"2025-12-02T10:30:45.123Z","level":"INFO","type":"proxy","message":"Proxy session starting"}
{"timestamp":"2025-12-02T10:30:45.150Z","level":"INFO","type":"proxy","message":"Server started","context":{"binary":"go","args":["run","test-server.go"],"pid":12345}}
{"timestamp":"2025-12-02T10:30:45.200Z","level":"INFO","type":"request","direction":"→","method":"test","id":1,"message":"→ test"}
```

### 5. Graceful shutdown

Press Ctrl+C to stop the proxy:

```bash
diagnose-mcp ./my-mcp-server
# Press Ctrl+C

# Output:
# Received signal: interrupt
# [INFO] [proxy] Proxy session ended
#   Context: duration=5m30s, message_count=142, error_count=0
```

## Common Use Cases

### Debugging a tool call

```bash
cat <<EOF | diagnose-mcp --verbose ./my-server
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}
EOF
```

### Piping from a client

```bash
# Your MCP client sends to diagnose-mcp's stdin
# diagnose-mcp forwards to the real server
# Responses flow back through stdout
my-mcp-client | diagnose-mcp ./my-server
```

### Capturing logs for analysis

```bash
diagnose-mcp --json ./my-server 2>logs.jsonl

# Later, analyze with jq:
cat logs.jsonl | jq 'select(.type=="request") | .method'
```

## Next Steps

- See [README.md](../README.md) for full documentation
- Check [examples/](../examples/) for real-world examples
- Read the [MCP Specification](https://spec.modelcontextprotocol.io/)

## Troubleshooting

### Server not starting

```bash
# Check if the server binary exists and is executable
ls -la ./my-server

# Check if the server runs standalone
./my-server

# Use absolute path if needed
diagnose-mcp /full/path/to/my-server
```

### No logs appearing

Logs go to **stderr**, MCP messages go to **stdout**. Redirect separately:

```bash
# Save logs to file, see MCP messages on screen
diagnose-mcp ./my-server 2>debug.log

# Save both separately
diagnose-mcp ./my-server >messages.jsonl 2>logs.txt
```

### Permission denied

```bash
# Make server executable
chmod +x ./my-server

# Or run with interpreter
diagnose-mcp python my-server.py
diagnose-mcp node my-server.js
```

## Performance

- **Latency**: <10ms p95 overhead (transparent proxy)
- **Memory**: <100MB for 1-hour sessions
- **Startup**: <500ms initialization time

Verified with:
```bash
go test ./... -bench=. -benchmem
```
