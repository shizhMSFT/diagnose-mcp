# Examples

This directory contains example MCP servers for testing `diagnose-mcp`.

## Example MCP Server (Python)

A simple MCP server implementing the Model Context Protocol specification.

### Features

- **initialize**: Protocol handshake
- **tools/list**: List available tools
- **tools/call**: Execute tools (echo, add)
- **Notifications**: Sends `initialized` notification

### Usage

```bash
# Run with diagnose-mcp
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  diagnose-mcp python examples/example-mcp-server.py

# Verbose mode to see full messages
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  diagnose-mcp --verbose python examples/example-mcp-server.py

# Call a tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello"}}}' | \
  diagnose-mcp python examples/example-mcp-server.py

# JSON output format
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"add","arguments":{"a":5,"b":3}}}' | \
  diagnose-mcp --json python examples/example-mcp-server.py
```

### Expected Output

**Text format (default):**
```
2025-12-02T10:30:45.123Z [INFO] [proxy] Proxy session starting
2025-12-02T10:30:45.150Z [INFO] [proxy] Server started
  Context: binary=python, args=[examples/example-mcp-server.py], pid=12345
2025-12-02T10:30:45.200Z [INFO] [notification] initialized
2025-12-02T10:30:45.210Z [INFO] [request] -> initialize #1
2025-12-02T10:30:45.250Z [INFO] [response] <- #1
```

**JSON format:**
```json
{"timestamp":"2025-12-02T10:30:45.123Z","level":"INFO","type":"proxy","message":"Proxy session starting"}
{"timestamp":"2025-12-02T10:30:45.200Z","level":"INFO","type":"notification","method":"initialized","message":"initialized"}
{"timestamp":"2025-12-02T10:30:45.210Z","level":"INFO","type":"request","direction":"->","method":"initialize","id":1,"message":"-> initialize"}
```

## Testing the Example

### 1. Initialize protocol

```bash
cat <<EOF | diagnose-mcp python examples/example-mcp-server.py
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
EOF
```

### 2. List available tools

```bash
cat <<EOF | diagnose-mcp python examples/example-mcp-server.py
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF
```

### 3. Call the echo tool

```bash
cat <<EOF | diagnose-mcp --verbose python examples/example-mcp-server.py
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"text":"Hello, MCP!"}}}
EOF
```

### 4. Call the add tool

```bash
cat <<EOF | diagnose-mcp python examples/example-mcp-server.py
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add","arguments":{"a":42,"b":58}}}
EOF
```

### 5. Test error handling

```bash
cat <<EOF | diagnose-mcp python examples/example-mcp-server.py
{"jsonrpc":"2.0","id":1,"method":"unknown/method","params":{}}
EOF
```

## Creating Your Own MCP Server

Use this example as a template:

1. Implement the MCP protocol handlers
2. Read JSON-RPC messages from stdin (line-delimited)
3. Write responses to stdout (line-delimited)
4. Write logs/errors to stderr
5. Test with `diagnose-mcp`

See the [MCP Specification](https://spec.modelcontextprotocol.io/) for details.
