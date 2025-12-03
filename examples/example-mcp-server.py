#!/usr/bin/env python3
"""
Example MCP Server - Implements basic MCP protocol for testing diagnose-mcp
Supports: initialize, tools/list, tools/call
"""

import json
import sys

def handle_initialize(msg_id, params):
    """Handle initialize request"""
    return {
        "jsonrpc": "2.0",
        "id": msg_id,
        "result": {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {}
            },
            "serverInfo": {
                "name": "example-mcp-server",
                "version": "1.0.0"
            }
        }
    }

def handle_tools_list(msg_id, params):
    """Handle tools/list request"""
    return {
        "jsonrpc": "2.0",
        "id": msg_id,
        "result": {
            "tools": [
                {
                    "name": "echo",
                    "description": "Echo back the input text",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "text": {"type": "string"}
                        },
                        "required": ["text"]
                    }
                },
                {
                    "name": "add",
                    "description": "Add two numbers",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "a": {"type": "number"},
                            "b": {"type": "number"}
                        },
                        "required": ["a", "b"]
                    }
                }
            ]
        }
    }

def handle_tools_call(msg_id, params):
    """Handle tools/call request"""
    tool_name = params.get("name")
    arguments = params.get("arguments", {})
    
    if tool_name == "echo":
        text = arguments.get("text", "")
        result = {
            "content": [
                {
                    "type": "text",
                    "text": f"Echo: {text}"
                }
            ]
        }
    elif tool_name == "add":
        a = arguments.get("a", 0)
        b = arguments.get("b", 0)
        result = {
            "content": [
                {
                    "type": "text",
                    "text": f"Result: {a + b}"
                }
            ]
        }
    else:
        return {
            "jsonrpc": "2.0",
            "id": msg_id,
            "error": {
                "code": -32601,
                "message": f"Unknown tool: {tool_name}"
            }
        }
    
    return {
        "jsonrpc": "2.0",
        "id": msg_id,
        "result": result
    }

def main():
    """Main message loop"""
    # Send initialized notification
    initialized = {
        "jsonrpc": "2.0",
        "method": "initialized"
    }
    print(json.dumps(initialized), flush=True)
    
    # Message loop
    for line in sys.stdin:
        try:
            msg = json.loads(line)
            msg_id = msg.get("id")
            method = msg.get("method")
            params = msg.get("params", {})
            
            # Route to handlers
            if method == "initialize":
                response = handle_initialize(msg_id, params)
            elif method == "tools/list":
                response = handle_tools_list(msg_id, params)
            elif method == "tools/call":
                response = handle_tools_call(msg_id, params)
            else:
                response = {
                    "jsonrpc": "2.0",
                    "id": msg_id,
                    "error": {
                        "code": -32601,
                        "message": f"Method not found: {method}"
                    }
                }
            
            print(json.dumps(response), flush=True)
            
        except json.JSONDecodeError as e:
            sys.stderr.write(f"JSON decode error: {e}\n")
        except Exception as e:
            sys.stderr.write(f"Error: {e}\n")

if __name__ == "__main__":
    main()
