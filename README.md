# Go AI SDK

Unofficial Go port of the Vercel AI SDK concepts.

This project is not affiliated with Vercel and is currently evolving quickly.

## Installation

Note: submodule imports are available on this branch and will be fully consumable from released tags after merge/versioning.

Install only what you need:

```bash
go get github.com/iamanishx/go-ai/agent
go get github.com/iamanishx/go-ai/provider/bedrock
go get github.com/iamanishx/go-ai/mcp
```

Or install a single module only.

## Core Install

```bash
go get github.com/iamanishx/go-ai
```

## What You Get

- `agent` package with a tool-loop style agent
- `provider` core types and model interfaces
- `provider/bedrock` for Amazon Bedrock chat models
- `stream` helpers for stream readers and parsing

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/iamanishx/go-ai/agent"
	"github.com/iamanishx/go-ai/provider"
	"github.com/iamanishx/go-ai/provider/bedrock"
)

func main() {
	ctx := context.Background()

	bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
		Region:  "us-east-1",
		Profile: "myprofile",
	})

	chatModel := bedrockProvider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")

	weatherTool := provider.Tool{
		Name:        "get_weather",
		Description: "Get weather for a location",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"location"},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			location, _ := input["location"].(string)
			return fmt.Sprintf("Weather in %s: Sunny", location), nil
		},
	}

	toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
		Model:        chatModel,
		Tools:        []provider.Tool{weatherTool},
		ExecuteTools: true,
		MaxSteps:     10,
	})

	stream, err := toolAgent.Stream(ctx, agent.AgentCallOptions{
		Prompt: "What is the weather in San Francisco?",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	for part := range stream.Part() {
		switch part.Type {
		case "text-delta":
			fmt.Print(part.Text)
		case "tool-call":
			fmt.Printf("\n[tool-call: %s id=%s input=%s]\n", part.ToolName, part.ToolCallID, part.ToolInput)
		case "tool-input-delta":
			fmt.Print(part.ToolInputDelta)
		case "tool-result":
			fmt.Printf("\n[tool-result: %s id=%s output=%s]\n", part.ToolName, part.ToolCallID, part.ToolResult)
		case "finish":
			fmt.Printf("\n[finish: %s tokens(in=%d out=%d total=%d)]\n",
				part.FinishReason, part.Usage.InputTokens, part.Usage.OutputTokens, part.Usage.TotalTokens)
		case "error":
			fmt.Printf("\n[error: %v]\n", part.Error)
		}
	}
}
```

## Bedrock Authentication

### AWS profile

```go
bedrock.Create(bedrock.BedrockProviderSettings{
	Region:  "us-east-1",
	Profile: "myprofile",
})
```

### Environment variables

```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

```go
bedrock.Create(bedrock.BedrockProviderSettings{
	Region: "us-east-1",
})
```

### Static credentials

```go
bedrock.Create(bedrock.BedrockProviderSettings{
	Region:          "us-east-1",
	AccessKeyID:     "...",
	SecretAccessKey: "...",
	SessionToken:    "...",
})
```

### Custom credential provider

```go
bedrock.Create(bedrock.BedrockProviderSettings{
	Region: "us-east-1",
	CredentialProvider: &bedrock.SharedConfigCredentialProvider{
		Profile: "myprofile",
	},
})
```

## Streaming

The quick start above uses streaming by default.

## MCP Client (stdio / sse / http)

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
        Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
        Stderr: os.Stderr,
    }),
})

if err != nil {
    panic(err)
}
defer client.Close()

mcpTools, err := client.Tools(ctx)
if err != nil {
    panic(err)
}

_ = mcpTools
```

Alternative transports:

- `mcp.NewSSEClientTransport(mcp.SSETransportConfig{URL: "https://.../sse"})`
- `mcp.NewHTTPClientTransport(mcp.HTTPTransportConfig{URL: "https://.../mcp"})`

## Package Layout

```text
go-ai/
├── ai.go
├── agent/
├── provider/
│   ├── types.go
│   └── bedrock/
└── stream/
```

## Notes

- This project currently ships the Bedrock provider.
- Bedrock auth and request signing are handled by AWS SDK for Go v2.
- Bedrock streaming uses `ConverseStream` for incremental text deltas.
- API behavior may change while the library stabilizes.

## License

MIT
