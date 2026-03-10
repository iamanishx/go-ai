---
title: MCP
description: Connect MCP servers over stdio, SSE, and HTTP and expose tools to the agent
---

The `mcp` module provides a Go equivalent of `createMCPClient` and converts MCP tools into `provider.Tool` for the tool-loop agent.

## Install

```bash
go get github.com/iamanishx/go-ai/mcp
```

## Create MCP Client (Stdio)

```go
import (
    "context"
    "os"

    "github.com/iamanishx/go-ai/mcp"
)

ctx := context.Background()

client, err := mcp.CreateMCPClient(ctx, mcp.MCPClientConfig{
    Transport: mcp.NewStdioClientTransport(mcp.StdioConfig{
        Command: "npx",
        Args: []string{
            "-y",
            "@modelcontextprotocol/server-filesystem",
            ".",
        },
        Stderr: os.Stderr,
    }),
})
if err != nil {
    panic(err)
}
defer client.Close()
```

## Create MCP Client (SSE)

```go
client, err := mcp.CreateMCPClient(ctx, mcp.MCPClientConfig{
    Transport: mcp.NewSSEClientTransport(mcp.SSETransportConfig{
        URL: "https://your-server.com/sse",
    }),
})
```

## Create MCP Client (HTTP)

```go
client, err := mcp.CreateMCPClient(ctx, mcp.MCPClientConfig{
    Transport: mcp.NewHTTPClientTransport(mcp.HTTPTransportConfig{
        URL: "https://your-server.com/mcp",
    }),
})
```

## Use MCP Tools with Agent

```go
mcpToolsMap, err := client.Tools(ctx)
if err != nil {
    panic(err)
}

tools := make([]provider.Tool, 0, len(mcpToolsMap))
for _, t := range mcpToolsMap {
    tools = append(tools, t)
}

toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        chatModel,
    Tools:        tools,
    ExecuteTools: true,
    MaxSteps:     10,
})
```

## Available Client Methods

- `Tools(ctx)`
- `ListTools(ctx, cursor)`
- `ToolsFromDefinitions(defs)`
- `ListResources(ctx, cursor)`
- `ReadResource(ctx, uri)`
- `ListResourceTemplates(ctx, cursor)`
- `ListPrompts(ctx, cursor)`
- `GetPrompt(ctx, name, args)`
- `Close()`
