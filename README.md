# diagnose-mcp

MCP Protocol Proxy Server - A transparent proxy for debugging and monitoring Model Context Protocol (MCP) servers.

## Features

- **Local MCP Server Proxying**: Intercept and log all MCP messages between client and local server
- **Remote WebSocket Support**: Connect to remote MCP servers via HTTP/WebSocket
- **Transparent Pass-Through**: Messages are forwarded without modification
- **Detailed Logging**: Track requests, responses, notifications, and progress updates
- **Log File Support**: Write logs to files with dynamic pattern support (`{timestamp}`, `{session}`, `{pid}`)
- **File Monitoring**: Watch files for changes and display new content (tail-like behavior, non-blocking)
- **Text Format** (default): Human-readable timestamps, log levels, message types
- **JSON Format** (`--json`): Structured output for parsing by other tools
- **Verbose Mode** (`--verbose`): Include full message payloads (readable text or base64 for binary data)
- **Environment Pass-Through**: Server inherits all parent environment variables
- **Graceful Shutdown**: Signal handling (SIGTERM/SIGINT) with session statistics

## Installation

```bash
go install github.com/shizhMSFT/diagnose-mcp/cmd/diagnose-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/shizhMSFT/diagnose-mcp.git
cd diagnose-mcp
go build -o diagnose-mcp ./cmd/diagnose-mcp
```

## Usage

### Local Mode (Proxy Local MCP Server)

```bash
diagnose-mcp <server-binary> [server-args...]
```

**Examples:**

```bash
# Basic usage
diagnose-mcp ./my-mcp-server

# With server arguments
diagnose-mcp ./my-mcp-server --port 8080 --config server.json

# Verbose mode (shows full message payloads)
diagnose-mcp --verbose ./my-mcp-server

# JSON output format
diagnose-mcp --json ./my-mcp-server | jq .

# Write logs to file with timestamp and session ID (ordered chronologically)
diagnose-mcp --log-file "logs/{timestamp}-{session}.log" ./my-mcp-server

# Watch files for changes
diagnose-mcp --watch /tmp/server.log --watch /tmp/config.json ./my-mcp-server

# Remote WebSocket server
diagnose-mcp --remote ws://localhost:8080/mcp
```

For more information, see the [documentation](docs/).
