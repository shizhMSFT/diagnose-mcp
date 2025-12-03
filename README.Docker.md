# Docker Usage

## Build the Image

```bash
docker build -t diagnose-mcp .
```

## Run Examples

### Basic Usage
```bash
# Show help
docker run --rm diagnose-mcp

# Proxy a local MCP server (mount current directory)
docker run --rm -it -v ${PWD}:/workspace diagnose-mcp npx -y @modelcontextprotocol/server-everything
```

### With File Watching
```bash
# Watch files in the workspace
docker run --rm -it \
  -v ${PWD}:/workspace \
  diagnose-mcp \
  --watch /workspace/server.log \
  npx -y @modelcontextprotocol/server-everything
```

### With Log File Output
```bash
# Write logs to file with pattern
docker run --rm -it \
  -v ${PWD}:/workspace \
  diagnose-mcp \
  --log-file "/workspace/logs/{timestamp}-{session}.log" \
  npx -y @modelcontextprotocol/server-everything
```

### JSON Output Format
```bash
# Output logs in JSON format
docker run --rm -it \
  -v ${PWD}:/workspace \
  diagnose-mcp \
  --json \
  npx -y @modelcontextprotocol/server-everything
```

### Interactive Mode with Custom Server
```bash
# Run with your own MCP server binary
docker run --rm -it \
  -v ${PWD}:/workspace \
  -v /path/to/your/server:/server \
  diagnose-mcp \
  /server/my-mcp-server --server-args
```

## Notes

- The working directory inside the container is `/workspace`
- Mount your local directory to `/workspace` to access files
- Use absolute paths when watching files or writing logs
- The container includes Node.js and npx for running JavaScript-based MCP servers
- For Windows PowerShell, replace `${PWD}` with `${PWD}` or use full paths
