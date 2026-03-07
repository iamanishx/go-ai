---
title: Getting Started
description: Install, configure Bedrock, and run your first tool loop
---

This guide walks through the current production path: Bedrock provider + ToolLoopAgent.

## Installation

```bash
go get github.com/iamanishx/go-ai
```

## 1) Create a Bedrock Provider

```go
import "github.com/iamanishx/go-ai/provider/bedrock"

bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})
```

You can also authenticate with environment variables:

```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
```

```go
bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

## 2) Define a Tool

```go
import (
    "fmt"

    "github.com/iamanishx/go-ai/provider"
)

weatherTool := provider.Tool{
    Name:        "get_weather",
    Description: "Get weather for a location",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{"type": "string"},
        },
        "required": []string{"location"},
    },
    Execute: func(input map[string]interface{}) (string, error) {
        location, _ := input["location"].(string)
        return fmt.Sprintf("Weather in %s: Sunny", location), nil
    },
}
```

## 3) Create a ToolLoopAgent

```go
import "github.com/iamanishx/go-ai/agent"

toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        bedrockProvider.Chat("anthropic.claude-3-sonnet-20240229-v1:0"),
    Tools:        []provider.Tool{weatherTool},
    ExecuteTools: true,
    MaxSteps:     10,
})
```

## 4) Generate Text

```go
ctx := context.Background()

result, err := toolAgent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
    System: "You are a concise assistant.",
})

if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
```

## 5) Stream Responses

```go
stream, err := toolAgent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
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
        fmt.Printf("Calling tool: %s\n", part.ToolName)
    case "error":
        fmt.Printf("Stream error: %v\n", part.Error)
    case "finish":
        fmt.Printf("Done: %s\n", part.FinishReason)
    }
}
```

## Configuration

### Agent Options

| Option | Type | Description |
|--------|------|-------------|
| `Model` | `provider.ChatModel` | The language model to use |
| `Tools` | `[]provider.Tool` | Available tools |
| `ExecuteTools` | `bool` | Whether to execute tools |
| `MaxSteps` | `int` | Maximum steps (default: 20) |
| `OnStart` | `func` | Called when agent starts |
| `OnStepFinish` | `func` | Called after each step |
| `OnFinish` | `func` | Called when agent finishes |

### Provider Options

| Option | Type | Description |
|--------|------|-------------|
| `Region` | `string` | AWS region |
| `Profile` | `string` | AWS profile name |
| `AccessKeyID` | `string` | AWS access key |
| `SecretAccessKey` | `string` | AWS secret key |
| `SessionToken` | `string` | AWS session token |
| `APIKey` | `string` | Bearer token |
| `BaseURL` | `string` | Custom endpoint |

## Next Steps

- [Agent](/go-ai/agent/) - Tool loop behavior and callbacks
- [Provider](/go-ai/provider/) - Provider architecture overview
- [Amazon Bedrock](/go-ai/provider/bedrock/) - Bedrock settings and auth
- [Examples](/go-ai/examples/) - Ready-to-run snippets
