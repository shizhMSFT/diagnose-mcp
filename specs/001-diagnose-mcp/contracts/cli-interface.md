# CLI Interface Contract

**Purpose**: Define command-line interface for diagnose-mcp, including flags, arguments, exit codes, and usage patterns

---

## Command Syntax

### Local MCP Server (stdio)
```bash
diagnose-mcp [options] <server-binary> [server-args...]
```

### Remote MCP Server (HTTP/WebSocket)
```bash
diagnose-mcp --remote <url> [options]
```

---

## Positional Arguments

### `<server-binary>` (Local mode only)
- **Type**: String (path to executable)
- **Required**: Yes (unless `--remote` specified)
- **Description**: Path to MCP server binary to proxy
- **Validation**: 
  - Must be executable file or in PATH
  - Error if not found: "Error: MCP server binary not found: <path>"

### `[server-args...]` (Local mode only)
- **Type**: String array
- **Required**: No
- **Description**: Arguments passed directly to proxied server
- **Example**: `diagnose-mcp ./my-server --port 8080 --verbose`

---

## Flags & Options

### `--remote <url>`, `-r <url>`
- **Type**: String (URL)
- **Required**: No (mutually exclusive with positional `<server-binary>`)
- **Description**: URL of remote MCP server (HTTP/WebSocket)
- **Validation**: 
  - Must be valid URL with scheme (http://, https://, ws://, wss://)
  - Error: "Error: Invalid remote URL: <url>"
- **Example**: 
  ```bash
  diagnose-mcp --remote https://example.com/mcp
  diagnose-mcp -r ws://localhost:8080/mcp
  ```

### `--watch <path>`, `-w <path>` (Multiple allowed)
- **Type**: String (file path)
- **Required**: No
- **Repeatable**: Yes
- **Description**: Monitor file for changes (creation, modification, deletion)
- **Validation**: 
  - Path can be relative or absolute
  - Warning if file doesn't exist at start: "Warning: Watched file not found: <path> (will monitor for creation)"
- **Example**:
  ```bash
  diagnose-mcp --watch /tmp/server.log --watch ./debug.txt my-server
  ```

### `--verbose`, `-v`
- **Type**: Boolean flag
- **Default**: false
- **Description**: Enable detailed logging (full message payloads, debug info)
- **Effect**: 
  - If false: Log message summaries (method, id, direction)
  - If true: Log full JSON payloads
- **Example**:
  ```bash
  diagnose-mcp --verbose my-server
  ```

### `--json`, `-j`
- **Type**: Boolean flag
- **Default**: false (human-readable text)
- **Description**: Output logs in JSON format for machine parsing
- **Effect**: Changes log output from text to JSON lines
- **Example**:
  ```bash
  diagnose-mcp --json my-server | jq .
  ```

### `--help`, `-h`
- **Type**: Boolean flag
- **Description**: Display help message and exit
- **Exit Code**: 0

### `--version`, `-V`
- **Type**: Boolean flag
- **Description**: Display version and exit
- **Example Output**: `diagnose-mcp version 1.0.0`
- **Exit Code**: 0

---

## Input/Output Streams

### stdin
- **Purpose**: MCP client → proxy → MCP server
- **Format**: Newline-delimited JSON (MCP protocol messages)
- **Behavior**: Proxy reads MCP messages from stdin, forwards to server

### stdout
- **Purpose**: Diagnostic logs (NOT MCP protocol messages)
- **Format**: Text (default) or JSON (`--json` flag)
- **Behavior**: All diagnostic output (message logs, file events, system events)
- **Note**: MCP client should NOT read from diagnose-mcp stdout

### stderr
- **Purpose**: Error messages, warnings
- **Format**: Human-readable text
- **Behavior**: Critical errors, validation failures, system errors

### MCP Server stdout/stderr
- **Proxy Behavior**:
  - MCP server stdout → forward to MCP client (transparent)
  - MCP server stderr → forward to diagnose-mcp stderr (with prefix)

---

## Exit Codes

| Code | Meaning | Example Scenario |
|------|---------|------------------|
| 0 | Success | Normal shutdown, server exited cleanly |
| 1 | General error | Invalid arguments, configuration error |
| 2 | Server binary not found | `<server-binary>` doesn't exist or not executable |
| 3 | Server crashed | Proxied MCP server exited with error |
| 4 | Network error | Remote server connection failed (--remote) |
| 130 | SIGINT (Ctrl+C) | User interrupted |
| 143 | SIGTERM | Terminated by signal |

---

## Usage Examples

### Example 1: Proxy local MCP server with verbose logging
```bash
diagnose-mcp --verbose ./my-mcp-server --config server.json
```

**Effect**:
- Runs `./my-mcp-server --config server.json` as child process
- Logs all MCP messages with full payloads to stdout (text format)
- Forwards MCP protocol transparently between client and server

### Example 2: Proxy remote server with JSON logging
```bash
diagnose-mcp --remote wss://prod.example.com/mcp --json > logs.jsonl
```

**Effect**:
- Connects to WebSocket MCP server
- Logs all messages in JSON format
- Redirects logs to `logs.jsonl` file

### Example 3: Monitor server logs while proxying
```bash
diagnose-mcp --watch /var/log/server.log --watch /tmp/debug.txt my-server
```

**Effect**:
- Proxies `my-server`
- Monitors `/var/log/server.log` and `/tmp/debug.txt`
- Logs file events (creation, appends, deletion) alongside MCP messages

### Example 4: Combine all options
```bash
diagnose-mcp \
  --verbose \
  --json \
  --watch ./logs/access.log \
  --watch ./logs/error.log \
  ./bin/mcp-server --port 8080
```

**Effect**:
- Verbose, JSON-formatted logs
- Two files monitored
- Server binary at `./bin/mcp-server` with `--port 8080` argument

---

## Help Message

```
diagnose-mcp - MCP protocol proxy with diagnostic logging

USAGE:
    diagnose-mcp [OPTIONS] <server-binary> [server-args...]
    diagnose-mcp --remote <url> [OPTIONS]

OPTIONS:
    -r, --remote <url>       Proxy a remote MCP server (HTTP/WebSocket)
    -w, --watch <path>       Monitor file for changes (repeatable)
    -v, --verbose            Enable detailed logging
    -j, --json               Output logs in JSON format
    -h, --help               Display this help message
    -V, --version            Display version

EXAMPLES:
    # Proxy local MCP server
    diagnose-mcp ./my-server --config server.json

    # Proxy remote server with JSON logs
    diagnose-mcp --remote https://example.com/mcp --json

    # Monitor log files while proxying
    diagnose-mcp --watch /tmp/server.log my-server

For more information: https://github.com/shizhMSFT/diagnose-mcp
```

---

## Validation Rules

### Mutual Exclusion
- `<server-binary>` and `--remote` are mutually exclusive
- Error if both specified: "Error: Cannot specify both server binary and --remote URL"
- Error if neither specified: "Error: Must specify either server binary or --remote URL"

### File Paths
- `--watch` paths: Resolve relative paths to absolute
- Emit warning (not error) if watched file doesn't exist at start

### URL Validation
- `--remote` URL must have valid scheme (http, https, ws, wss)
- Must be parseable by Go's `url.Parse`

---

## Configuration File (Future Enhancement - Out of Scope for v1)

**Note**: Current version uses CLI flags only. Configuration file support may be added in future versions.

Potential format (for reference):
```yaml
# diagnose-mcp.yaml (not implemented in v1)
remote: https://example.com/mcp
watch:
  - /var/log/server.log
  - /tmp/debug.txt
verbose: true
output: json
```

---

**Contract Status**: ✅ **COMPLETE** - CLI interface fully specified with flags, arguments, exit codes, and usage examples
