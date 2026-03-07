# Examples

Here are complete examples to help you get started with Go AI SDK.

## Basic Agent

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

    provider := bedrock.Create(bedrock.BedrockProviderSettings{
        Region:  "us-east-1",
        Profile: "myprofile",
    })

    chatModel := provider.Chat("anthropic.claude-3-sonnet-v1:0")

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

    agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
        Model:        chatModel,
        Tools:        []provider.Tool{weatherTool},
        ExecuteTools: true,
    })

    result, err := agent.Generate(ctx, agent.AgentCallOptions{
        Prompt: "What's the weather in San Francisco?",
        System: "You are a helpful assistant.",
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text)
}
```

## Streaming Example

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "Tell me a story",
})

defer stream.Close()

for part := range stream.Part() {
    switch part.Type {
    case "text-delta":
        fmt.Print(part.Text)
    case "tool-call":
        fmt.Printf("\n[Tool: %s]\n", part.ToolName)
    case "tool-result":
        fmt.Printf("[Result: %s]\n", part.ToolResult)
    case "finish":
        fmt.Printf("\n[Done: %s]\n", part.FinishReason)
    }
}
```

## With Callbacks

```go
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        chatModel,
    Tools:        []provider.Tool{weatherTool},
    ExecuteTools: true,
    MaxSteps:     10,

    OnStart: func(event agent.OnStartEvent) {
        fmt.Println("🚀 Agent started")
    },

    OnStepStart: func(event agent.OnStepStartEvent) {
        fmt.Printf("📝 Step %d started\n", event.StepNumber)
    },

    OnStepFinish: func(event agent.OnStepFinishEvent) {
        fmt.Printf("✅ Step %d finished: %s\n", event.StepNumber, event.Text)
    },

    OnToolCallStart: func(event agent.OnToolCallStartEvent) {
        fmt.Printf("🔧 Calling tool: %s\n", event.ToolName)
    },

    OnToolCallFinish: func(event agent.OnToolCallFinishEvent) {
        fmt.Printf("� Result: %s\n", event.Output)
    },

    OnFinish: func(event agent.OnFinishEvent) {
        fmt.Printf("🎉 Done! Final: %s\n", event.Text)
    },
})
```

## Multiple Tools

```go
tools := []provider.Tool{
    {
        Name:        "get_weather",
        Description: "Get weather for a location",
        Parameters:  weatherSchema,
        Execute:     getWeather,
    },
    {
        Name:        "calculate",
        Description: "Perform calculation",
        Parameters:  calcSchema,
        Execute:     calculate,
    },
    {
        Name:        "search",
        Description: "Search the web",
        Parameters:  searchSchema,
        Execute:     search,
    },
}

agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        chatModel,
    Tools:        tools,
    ExecuteTools: true,
})
```

## Custom Stop Conditions

```go
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:    chatModel,
    Tools:    tools,
    MaxSteps: 5, // Stop after 5 steps
})
```

## Tool with Error Handling

```go
Execute: func(input map[string]interface{}) (string, error) {
    location, ok := input["location"].(string)
    if !ok {
        return "", fmt.Errorf("location is required")
    }

    weather, err := fetchWeather(location)
    if err != nil {
        return "", fmt.Errorf("failed to fetch weather: %w", err)
    }

    return weather, nil
},
```

## Environment Variables Setup

```bash
# AWS credentials
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Or use profile
export AWS_PROFILE=myprofile
```
