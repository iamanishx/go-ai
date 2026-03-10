# go-ai MCP module

This module provides a Go SDK wrapper for MCP client usage with `go-ai` tools.

Install:

```bash
go get github.com/iamanishx/go-ai/mcp
```

Basic usage:

```go
ctx := context.Background()

client, err := mcp.CreateMCPClient(ctx, mcp.MCPClientConfig{
    Transport: mcp.NewStdioClientTransport(mcp.StdioConfig{
        Command: "npx",
        Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
    }),
})
if err != nil {
    panic(err)
}
defer client.Close()

tools, err := client.Tools(ctx)
if err != nil {
    panic(err)
}

_ = tools
```

Supported transport helpers:

- `NewStdioClientTransport(...)`
- `NewSSEClientTransport(...)`
- `NewHTTPClientTransport(...)`
