# diagnose-mcp

MCP Protocol Proxy Server - A transparent proxy for debugging and monitoring Model Context Protocol (MCP) servers.

## Features

- **Local MCP Server Proxying**: Intercept and log all MCP messages between client and local server
- **Transparent Pass-Through**: Messages are forwarded without modification
- **Detailed Logging**: Track requests, responses, notifications, and progress updates
- **Multiple Output Formats**: Human-readable text or machine-parsable JSON logs
- **Verbose Mode**: Include full message payloads for deep debugging
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
```

For more information, see the [documentation](docs/).
