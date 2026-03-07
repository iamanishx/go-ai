# Go AI SDK

Unofficial Go port of the Vercel AI SDK concepts.

This project is not affiliated with Vercel and is currently evolving quickly.

## Installation

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
			fmt.Printf("\n[tool: %s]\n", part.ToolName)
		case "finish":
			fmt.Printf("\n[finish: %s]\n", part.FinishReason)
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
- API behavior may change while the library stabilizes.

## License

MIT
