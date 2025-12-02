# diagnose-mcp Quickstart Guide

**Welcome!** This guide will get you up and running with diagnose-mcp in minutes.

---

## What is diagnose-mcp?

diagnose-mcp is a **transparent MCP protocol proxy** that sits between an MCP client and server, logging all communication for debugging purposes. Think of it as a "wire tap" that shows you exactly what's happening in the MCP protocol.

**Use cases**:
- 🐛 Debug why tool calls are failing
- 📊 Understand MCP protocol flow
- 🔍 Correlate MCP traffic with server logs
- ⚡ Diagnose performance issues

---

## Installation

### Prerequisites
- Go 1.25.4 or later installed
- Git (for cloning the repository)

### Build from Source
```bash
# Clone repository
git clone https://github.com/shizhMSFT/diagnose-mcp.git
cd diagnose-mcp

# Build binary
go build -o diagnose-mcp ./cmd/diagnose-mcp

# (Optional) Install to PATH
sudo mv diagnose-mcp /usr/local/bin/
```

### Verify Installation
```bash
diagnose-mcp --version
# Output: diagnose-mcp version 1.0.0
```

---

## Quick Start: Proxy a Local MCP Server

### Example 1: Basic Proxying

**Scenario**: You have an MCP server binary called `my-mcp-server` and want to see all requests/responses.

```bash
# Instead of running your server directly:
# ./my-mcp-server

# Run it through diagnose-mcp:
diagnose-mcp ./my-mcp-server
```

**What you'll see** (example output):
```
2025-12-02T14:30:45.123Z INFO [SYSTEM] Proxy started: local server ./my-mcp-server
2025-12-02T14:30:45.200Z INFO [C→S] REQUEST id=1 method=initialize
2025-12-02T14:30:45.250Z INFO [S→C] RESPONSE id=1 (success)
2025-12-02T14:30:45.300Z INFO [C→S] REQUEST id=2 method=tools/list
2025-12-02T14:30:45.320Z INFO [S→C] RESPONSE id=2 (success)
2025-12-02T14:30:45.400Z INFO [C→S] REQUEST id=3 method=tools/call
2025-12-02T14:30:45.450Z INFO [S→C] RESPONSE id=3 (success)
```

**Legend**:
- `[C→S]`: Client sending to Server
- `[S→C]`: Server responding to Client
- `REQUEST/RESPONSE`: MCP message types

---

### Example 2: Verbose Logging (See Full Payloads)

**Note**: Payloads are automatically shown as readable text (if printable) or base64 (if binary).

**Scenario**: You want to see the actual JSON content of messages.

```bash
diagnose-mcp --verbose ./my-mcp-server
```

**Output** (now includes params and results):
```
2025-12-02T14:30:45.400Z INFO [C→S] REQUEST id=3 method=tools/call params={"name":"read_file","arguments":{"path":"/tmp/data.json"}}
2025-12-02T14:30:45.450Z INFO [S→C] RESPONSE id=3 result={"content":[{"type":"text","text":"...file contents..."}]}
```

---

### Example 3: JSON Output for Log Analysis

**Scenario**: You want to analyze logs programmatically or grep for specific events.

```bash
diagnose-mcp --json ./my-mcp-server > mcp-logs.jsonl
```

**Output** (one JSON object per line):
```json
{"time":"2025-12-02T14:30:45.400Z","level":"INFO","type":"mcp_message","direction":"client_to_server","message_type":"request","id":"3","method":"tools/call"}
{"time":"2025-12-02T14:30:45.450Z","level":"INFO","type":"mcp_message","direction":"server_to_client","message_type":"response","id":"3","status":"success"}
```

**Analyze with jq**:
```bash
# Count requests by method
cat mcp-logs.jsonl | jq -r 'select(.message_type=="request") | .method' | sort | uniq -c

# Find all errors
cat mcp-logs.jsonl | jq 'select(.level=="ERROR")'

# Extract all tool calls
cat mcp-logs.jsonl | jq 'select(.method=="tools/call")'
```

---

## Advanced: Monitor Server Log Files

### Example 4: Correlate MCP Traffic with Server Logs

**Scenario**: Your MCP server writes to `/var/log/myserver.log` and you want to see log events alongside MCP traffic.

```bash
diagnose-mcp --watch /var/log/myserver.log ./my-mcp-server
```

**Output** (interleaved MCP and file events):
```
2025-12-02T14:30:45.400Z INFO [C→S] REQUEST id=3 method=tools/call
2025-12-02T14:30:45.420Z INFO [FILE] /var/log/myserver.log: +2 lines appended (total: 145 lines)
2025-12-02T14:30:45.450Z INFO [S→C] RESPONSE id=3 (success)
```

**Multiple files**:
```bash
diagnose-mcp \
  --watch /var/log/myserver.log \
  --watch /tmp/debug.txt \
  ./my-mcp-server
```

---

## Advanced: Proxy a Remote MCP Server

### Example 5: Diagnose a Remote Server

**Scenario**: Your MCP server is running on a remote machine accessible via WebSocket.

```bash
diagnose-mcp --remote wss://example.com/mcp
```

**Output** (same logging format):
```
2025-12-02T14:30:45.123Z INFO [SYSTEM] Proxy started: remote server wss://example.com/mcp
2025-12-02T14:30:45.200Z INFO [C→S] REQUEST id=1 method=initialize
2025-12-02T14:30:45.250Z INFO [S→C] RESPONSE id=1 (success)
```

**With authentication** (future enhancement):
```bash
diagnose-mcp --remote https://example.com/mcp --auth-header "Authorization: Bearer <token>"
```

---

## Common Use Cases

### Use Case 1: Debugging Tool Call Failures

**Problem**: Your MCP client says "tool call failed" but you don't know why.

**Solution**:
```bash
diagnose-mcp --verbose my-mcp-server 2>&1 | grep -A 5 "ERROR"
```

**What to look for**:
- Error responses with error codes
- Malformed requests (invalid params)
- Timeout issues (long gap between request and response)

---

### Use Case 2: Performance Profiling

**Problem**: Your MCP server feels slow, but you don't know which calls are taking time.

**Solution**:
```bash
diagnose-mcp --json my-mcp-server > perf-logs.jsonl

# Analyze with jq (calculate request-response time deltas)
jq -s 'group_by(.id) | map({id: .[0].id, method: .[0].method, duration: (.[1].time - .[0].time)})' perf-logs.jsonl
```

---

### Use Case 3: Understanding Protocol Flow

**Problem**: You're new to MCP and want to understand the initialization handshake.

**Solution**:
```bash
diagnose-mcp --verbose my-mcp-server
# Observe the initialize → initialized → tools/list → ... flow
```

---

## Tips & Tricks

### Tip 1: Save Logs for Later Analysis
```bash
diagnose-mcp --json my-mcp-server > session-$(date +%Y%m%d-%H%M%S).jsonl
```

### Tip 2: Filter Logs by Direction
```bash
# See only client requests
diagnose-mcp my-mcp-server | grep "\[C→S\]"

# See only server responses
diagnose-mcp my-mcp-server | grep "\[S→C\]"
```

### Tip 3: Combine with Server Debug Logs
```bash
# Terminal 1: Run server with debug logs to file
# Terminal 2: Proxy with file watching
diagnose-mcp --watch /tmp/server-debug.log my-mcp-server
```

### Tip 4: Graceful Shutdown
```bash
# Press Ctrl+C to gracefully shut down
# diagnose-mcp will:
# 1. Stop forwarding new messages
# 2. Drain in-flight messages
# 3. Signal child process to terminate
# 4. Flush logs
# 5. Exit
```

---

## Configuration Cheat Sheet

| Flag | Short | Description | Example |
|------|-------|-------------|---------|
| `--remote` | `-r` | Proxy remote server (URL) | `-r wss://example.com/mcp` |
| `--watch` | `-w` | Monitor file changes | `-w /tmp/server.log` |
| `--verbose` | `-v` | Show full message payloads | `-v` |
| `--json` | `-j` | JSON output format | `-j` |
| `--help` | `-h` | Show help message | `-h` |
| `--version` | `-V` | Show version | `-V` |

---

## Troubleshooting

### Problem: "Error: MCP server binary not found"

**Cause**: The server binary path is incorrect.

**Solution**: Verify the path exists and is executable:
```bash
ls -la ./my-mcp-server
chmod +x ./my-mcp-server  # If not executable
```

---

### Problem: "Error: Failed to parse MCP message"

**Cause**: The server is sending malformed JSON.

**Solution**: Run with `--verbose` to see the raw output:
```bash
diagnose-mcp --verbose my-mcp-server 2>&1 | grep "Failed to parse"
# Look at the raw_input field to see what was sent
```

---

### Problem: No output appears

**Cause**: MCP client may be buffering stdout.

**Solution**: Check stderr for errors:
```bash
diagnose-mcp my-mcp-server 2>&1
```

---

### Problem: File watch events not appearing

**Cause**: File might not be changing, or permissions issue.

**Solution**: Verify file changes are happening:
```bash
# Terminal 1
diagnose-mcp --watch /tmp/test.log my-mcp-server

# Terminal 2
echo "test line" >> /tmp/test.log  # Should trigger event
```

---

## Next Steps

- 📖 Read the [CLI Interface Contract](contracts/cli-interface.md) for full flag documentation
- 🔍 See [Log Output Format](contracts/log-output.md) for details on JSON fields
- 🏗️ Review [Architecture](data-model.md) to understand internals
- 🐛 Report issues at https://github.com/shizhMSFT/diagnose-mcp/issues

---

## Quick Reference: Common Commands

```bash
# Basic proxying
diagnose-mcp ./my-server

# Verbose logging
diagnose-mcp -v ./my-server

# JSON output to file
diagnose-mcp -j ./my-server > logs.jsonl

# Watch log files
diagnose-mcp -w /tmp/server.log ./my-server

# Remote server
diagnose-mcp -r wss://example.com/mcp

# All options combined
diagnose-mcp -v -j -w /tmp/app.log -w /tmp/errors.log ./my-server > session.jsonl
```

---

**Happy Debugging!** 🎉

For questions or feedback, open an issue on GitHub: https://github.com/shizhMSFT/diagnose-mcp
