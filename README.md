# Go AI SDK

> **Note**: This is an unofficial Go port of the [Vercel AI SDK](https://github.com/vercel/ai). 
> It's still in early development and not affiliated with Vercel.

A Go SDK for building AI-powered applications with Large Language Models (LLMs). Ported from the [TypeScript AI SDK](https://github.com/vercel/ai) by Vercel.

## Features

- **Tool Loop Agent** - Autonomous agent that runs tools in a loop until a task is complete
- **Provider System** - Modular provider architecture supporting multiple LLM backends
- **Amazon Bedrock Provider** - Native integration with AWS Bedrock (Claude, Llama, etc.)
- **Streaming Support** - Full streaming response support with real-time text and tool call handling
- **AWS Credential Providers** - Support for multiple AWS credential methods

## Installation

```bash
go get github.com/ai-sdk/go-ai
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ai-sdk/go-ai/agent"
	"github.com/ai-sdk/go-ai/provider"
	"github.com/ai-sdk/go-ai/provider/bedrock"
)

func main() {
	ctx := context.Background()

	// Create Bedrock provider
	provider := bedrock.Create(bedrock.BedrockProviderSettings{
		Region: "us-east-1",
	})

	// Get chat model
	chatModel := provider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")

	// Define tools
	weatherTool := provider.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The location to get weather for",
				},
			},
			"required": []string{"location"},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			location := input["location"].(string)
			return fmt.Sprintf("The weather in %s is sunny and 72°F", location), nil
		},
	}

	// Create agent
	agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
		Model:        chatModel,
		Tools:        []provider.Tool{weatherTool},
		ExecuteTools: true,
		MaxSteps:     10,
	})

	// Generate text
	result, err := agent.Generate(ctx, agent.AgentCallOptions{
		Prompt: "What's the weather in San Francisco?",
		System: "You are a helpful assistant.",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %s\n", result.Text)
}
```

## Providers

### Amazon Bedrock

The Bedrock provider supports multiple authentication methods:

#### 1. Using Profile (like TypeScript's `fromIni`)

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",  // loads from ~/.aws/config and ~/.aws/credentials
})
```

#### 2. Using Environment Variables

```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
```

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

#### 3. Using Static Credentials

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:          "us-east-1",
    AccessKeyID:     "YOUR_ACCESS_KEY_ID",
    SecretAccessKey: "YOUR_SECRET_ACCESS_KEY",
    SessionToken:    "OPTIONAL_SESSION_TOKEN",
})
```

#### 4. Using Custom Credential Provider

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
    CredentialProvider: &bedrock.SharedConfigCredentialProvider{
        Profile: "myprofile",
    },
})
```

#### 5. Using AWS SDK Credential Provider Chain

```go
// Uses: env vars -> shared config -> default
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:            "us-east-1",
    CredentialProvider: bedrock.NewDefaultCredentialProviderChain(),
})
```

#### Built-in Credential Providers

- `EnvCredentialProvider` - Loads from environment variables
- `SharedConfigCredentialProvider` - Loads from ~/.aws/config and ~/.aws/credentials  
- `WebIdentityCredentialProvider` - Loads from Web Identity Token (EKS IAM roles)
- `StaticCredentialProvider` - Static credentials
- `DefaultCredentialProviderChain` - Tries env -> shared config -> default

### Supported Models

```go
// Anthropic Claude
model := provider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")
model := provider.Chat("anthropic.claude-3-haiku-20240307-v1:0")

// Amazon Titan
model := provider.Chat("amazon.titan-text-premium-v1:0")

// Meta Llama
model := provider.Chat("meta.llama3-70b-instruct-v1:0")
```

## Streaming

### Basic Streaming

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
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
        fmt.Printf("Tool call: %s\n", part.ToolName)
    case "tool-result":
        fmt.Printf("Tool result: %s\n", part.ToolResult)
    case "finish":
        fmt.Printf("Finish reason: %s\n", part.FinishReason)
    }
}
```

### Using StreamReader

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})

// Get accumulated values
text := stream.Text()
toolCalls := stream.ToolCalls()
finishReason := stream.FinishReason()
```

## Callbacks

### Non-Streaming Callbacks

```go
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        chatModel,
    Tools:        []provider.Tool{weatherTool},
    ExecuteTools: true,
    OnStart: func(event agent.OnStartEvent) {
        fmt.Println("Agent started")
    },
    OnStepStart: func(event agent.OnStepStartEvent) {
        fmt.Printf("Step %d started\n", event.StepNumber)
    },
    OnStepFinish: func(event agent.OnStepFinishEvent) {
        fmt.Printf("Step %d finished: %s\n", event.StepNumber, event.Text)
    },
    OnToolCallStart: func(event agent.OnToolCallStartEvent) {
        fmt.Printf("Tool %s started\n", event.ToolName)
    },
    OnToolCallFinish: func(event agent.OnToolCallFinishEvent) {
        fmt.Printf("Tool %s finished: %s\n", event.ToolName, event.Output)
    },
    OnFinish: func(event agent.OnFinishEvent) {
        fmt.Printf("Agent finished: %s\n", event.Text)
    },
})
```

## Tool Loop Agent

The ToolLoopAgent is an autonomous agent that:

1. Calls the LLM with the current prompt and messages
2. If the model makes tool calls, executes them
3. Adds tool results back to the conversation
4. Repeats until:
   - A finish reason other than tool-calls is returned
   - A tool doesn't have an execute function
   - A stop condition is met

### Stop Conditions

```go
// Default: stop after 20 steps
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: chatModel,
    Tools: tools,
})

// Custom max steps
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:    chatModel,
    Tools:    tools,
    MaxSteps: 5,
})

// Custom stop conditions
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:         chatModel,
    Tools:         tools,
    StopWhen:      []agent.StopCondition{agent.StepCountIs(10)},
})
```

### Tool Definition

```go
weatherTool := provider.Tool{
    Name:        "get_weather",
    Description: "Get weather information for a location",
    Parameters: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{
                "type":        "string",
                "description": "City name",
            },
            "unit": map[string]interface{}{
                "type":        "string",
                "description": "Temperature unit",
                "enum":        []string{"celsius", "fahrenheit"},
            },
        },
        "required": []string{"location"},
    },
    Execute: func(input map[string]interface{}) (string, error) {
        location := input["location"].(string)
        unit := "celsius"
        if u, ok := input["unit"].(string); ok {
            unit = u
        }
        
        // Call weather API...
        return fmt.Sprintf("Weather in %s: 22°C", location), nil
    },
}
```

## Core Types

### Message

```go
type Message struct {
    Role       string
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
}
```

### ToolCall

```go
type ToolCall struct {
    ID     string
    Name   string
    Input  map[string]interface{}
    Output string
}
```

### GenerateTextResult

```go
type GenerateTextResult struct {
    Text             string
    FinishReason     string
    ToolCalls        []ToolCall
    ToolResults      []ToolCall
    Usage            Usage
    ResponseMessages []Message
    Steps            []StepResult
}
```

### Usage

```go
type Usage struct {
    InputTokens  int
    OutputTokens int
    TotalTokens  int
}
```

## Architecture

```
go-ai/
├── agent/              # ToolLoopAgent implementation
├── provider/           # Provider interfaces and types
│   ├── types.go       # Core types
│   └── bedrock/      # Amazon Bedrock provider
│       └── bedrock.go
├── stream/            # Streaming utilities
└── example/           # Usage examples
```

## Adding New Providers

To add a new provider (e.g., OpenAI), create a new directory under `provider/`:

```go
// provider/openai/openai.go
package openai

import (
    "context"
    "github.com/ai-sdk/go-ai/provider"
)

type Provider interface {
    Chat(modelID string) provider.ChatModel
}

type ChatModel struct {
    // provider implementation
}

func (m *ChatModel) GenerateText(ctx context.Context, opts provider.GenerateTextOptions) (provider.GenerateTextResult, error) {
    // implementation
}

func (m *ChatModel) StreamText(ctx context.Context, opts provider.GenerateTextOptions) (<-chan provider.StreamPart, error) {
    // implementation
}

func Create(settings Settings) *Provider {
    // implementation
}
```

## Examples

See the `example/` directory for complete examples:

- `example/main.go` - Basic agent usage
- More examples coming soon

## License

Apache 2.0
