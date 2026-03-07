---
title: Getting Started
description: Get started with Go AI SDK in minutes
---

This guide will help you get started with Go AI SDK.

## Installation

```bash
go get github.com/iamanishx/go-ai
```

## Basic Usage

### 1. Create a Provider

```go
import "github.com/iamanishx/go-ai/provider/bedrock"

// Using AWS profile
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})

// Or using environment variables
// export AWS_REGION=us-east-1
// export AWS_ACCESS_KEY_ID=your-key
// export AWS_SECRET_ACCESS_KEY=your-secret
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

### 2. Define Tools

```go
import "github.com/iamanishx/go-ai/provider"

weatherTool := provider.Tool{
    Name:        "get_weather",
    Description: "Get weather for a location",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{
                "type":        "string",
                "description": "City name",
            },
        },
        "required": []string{"location"},
    },
    Execute: func(input map[string]interface{}) (string, error) {
        location := input["location"].(string)
        return fmt.Sprintf("Weather in %s: Sunny, 72°F", location), nil
    },
}
```

### 3. Create Agent

```go
import "github.com/iamanishx/go-ai/agent"

agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        provider.Chat("anthropic.claude-3-sonnet-v1:0"),
    Tools:        []provider.Tool{weatherTool},
    ExecuteTools: true,
    MaxSteps:     10,
})
```

### 4. Generate Text

```go
ctx := context.Background()

result, err := agent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
    System: "You are a helpful assistant.",
})

if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text)
```

## Streaming

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})

defer stream.Close()

for part := range stream.Part() {
    switch part.Type {
    case "text-delta":
        fmt.Print(part.Text)
    case "tool-call":
        fmt.Printf("Calling tool: %s\n", part.ToolName)
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

- [Agent Guide](/guide/agent/) - Learn about the Tool Loop Agent
- [Provider Guide](/guide/provider/) - Configure Providers
- [Examples](/guide/examples/) - More code examples
