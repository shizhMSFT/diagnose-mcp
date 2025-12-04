# diagnose-mcp

MCP Protocol Proxy Server - A transparent proxy for debugging and monitoring Model Context Protocol (MCP) servers.

## Features

- **Local MCP Server Proxying**: Intercept and log all MCP messages between client and local server
- **Remote WebSocket Support**: Connect to remote MCP servers via HTTP/WebSocket
- **Transparent Pass-Through**: Messages are forwarded without modification
- **Detailed Logging**: Track requests, responses, notifications, and progress updates
- **Log File Support**: Write logs to files with dynamic pattern support (`{timestamp}`, `{session}`, `{pid}`)
- **Azure Blob Logging**: Periodically upload logs to Azure Blob Storage (every 10s) for dev/test environments
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

# Periodically upload logs to Azure Blob Storage (for dev/test without file access)
diagnose-mcp --log-blob-url "https://<account>.blob.core.windows.net/<container>/<blob>?<sas>" ./my-mcp-server

# Watch files for changes
diagnose-mcp --watch /tmp/server.log --watch /tmp/config.json ./my-mcp-server

# Remote WebSocket server
diagnose-mcp --remote ws://localhost:8080/mcp
```

### Azure Blob Storage Logging (Dev/Test)

For environments where log files can't be exported, upload logs to Azure Blob Storage.  
Logs are written locally and uploaded every 10 seconds to a block blob.  
**Note:** Requires an existing Azure Storage Account.

```bash
# Quick setup with helper script
./examples/setup-azure-blob-logging.sh --account mystorageaccount

# Use the generated URL
export LOG_BLOB_URL="https://..."
diagnose-mcp --log-blob-url "$LOG_BLOB_URL" ./my-mcp-server
```

See [examples/azure-blob-logging.md](examples/azure-blob-logging.md) for detailed setup and usage.

## GitHub Copilot Integration

You can use `diagnose-mcp` as a transparent proxy to debug and monitor MCP servers used by GitHub Copilot Coding Agent.

### MCP Configuration Template

Add the following configuration to your Copilot MCP settings to proxy an existing MCP server:

```json
{
  "mcpServers": {
    "your-mcp-server": {
      "type": "local",
      "command": "bash",
      "args": [
        "-c",
        "cd $(mktemp -d) && curl -sSL https://github.com/shizhMSFT/diagnose-mcp/releases/download/v0.2.0/diagnose-mcp_0.2.0_linux_amd64.tar.gz | tar -xz && ./diagnose-mcp \"$@\"",
        "_",
        "--verbose",
        "--log-blob-url",
        "$LOG_BLOB_URL",
        "your-mcp-server-command",
        "server-args..."
      ],
      "env": {
        "LOG_BLOB_URL": "COPILOT_MCP_LOG_BLOB_URL"
      }
    }
  }
}
```

**Configuration Parameters:**

- `your-mcp-server`: Replace with your MCP server name
- `your-mcp-server-command`: Replace with the actual command to start your MCP server (e.g., `npx`, `python`, etc.)
- `server-args...`: Replace with any arguments your MCP server requires
- `COPILOT_MCP_LOG_BLOB_URL`: The name of the GitHub Actions secret containing your Azure Blob Storage SAS URL for log uploads (see Azure Blob Storage Logging section above). Configure this secret in your GitHub Copilot environment settings.
- `env`: You can add additional environment variables as needed. Values can be either GitHub Actions secret names (beginning with `COPILOT_MCP_`) or string literals.
- `--verbose`: Include full message payloads in logs (optional, remove for less verbose output)
- `--log-blob-url`: Upload logs to Azure Blob Storage (required for GitHub Copilot environments where the container filesystem is not directly accessible)

**Additional Options:**

You can add other `diagnose-mcp` flags to the `args` array as needed:
- `--json`: Output logs in JSON format
- `--watch`: Monitor additional files (e.g., `--watch`, `/path/to/file.log`)

For more details on extending GitHub Copilot with MCP servers, see the [official documentation](https://docs.github.com/en/enterprise-cloud@latest/copilot/how-tos/use-copilot-agents/coding-agent/extend-coding-agent-with-mcp).
