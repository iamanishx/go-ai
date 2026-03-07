---
title: Agent
description: Tool loop execution, callbacks, and stop control
---

The `ToolLoopAgent` is the orchestration layer. It calls the model, executes tool calls, appends tool results, and continues until a stop condition is met.

## Overview

1. Sends current conversation state to the model
2. Reads text and tool calls from the model response
3. Executes matching tools when `ExecuteTools` is enabled
4. Appends tool outputs as tool-role messages
5. Repeats until stop condition or max steps

## Usage

### Creating an Agent

```go
toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        chatModel,
    Tools:        []provider.Tool{tool1, tool2},
    ExecuteTools: true,
    MaxSteps:     10,
})
```

### Generate (Non-Streaming)

```go
result, err := toolAgent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
    System: "You are a helpful assistant.",
})
```

### Stream (Streaming)

```go
stream, err := toolAgent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})
defer stream.Close()

for part := range stream.Part() {
    switch part.Type {
    case "text-delta":
        fmt.Print(part.Text)
    case "tool-call":
        fmt.Printf("\n[tool call: %s]\n", part.ToolName)
    case "error":
        fmt.Printf("\n[stream error: %v]\n", part.Error)
    }
}
```

## Callbacks

### OnStart

Called when the agent starts processing:

```go
OnStart: func(event agent.OnStartEvent) {
    fmt.Println("Agent started with prompt:", event.Prompt)
},
```

### OnStepStart

Called when each step begins:

```go
OnStepStart: func(event agent.OnStepStartEvent) {
    fmt.Printf("Step %d started\n", event.StepNumber)
},
```

### OnStepFinish

Called after each step completes:

```go
OnStepFinish: func(event agent.OnStepFinishEvent) {
    fmt.Printf("Step %d: %s\n", event.StepNumber, event.Text)
    for _, tc := range event.ToolCalls {
        fmt.Printf("  Tool: %s\n", tc.Name)
    }
},
```

### OnToolCallStart

Called before a tool executes:

```go
OnToolCallStart: func(event agent.OnToolCallStartEvent) {
    fmt.Printf("Executing tool: %s\n", event.ToolName)
},
```

### OnToolCallFinish

Called after a tool finishes:

```go
OnToolCallFinish: func(event agent.OnToolCallFinishEvent) {
    fmt.Printf("Tool %s result: %s\n", event.ToolName, event.Output)
},
```

### OnFinish

Called when the agent completes:

```go
OnFinish: func(event agent.OnFinishEvent) {
    fmt.Printf("Final: %s\n", event.Text)
    fmt.Printf("Steps: %d\n", len(event.Steps))
},
```

## Stop Conditions

By default, the agent stops after 20 steps. You can override this with `MaxSteps` or with custom `StopWhen` conditions.

```go
toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:    chatModel,
    Tools:    tools,
    MaxSteps: 5, // Custom max steps
})
```

## Tool Definition

Tools are declared with name, JSON-schema-like parameters, and optional execute function.

```go
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
        return fmt.Sprintf("Weather in %s: 22°C", location), nil
    },
}
```

## Response Types

### AgentGenerateResult

```go
type AgentGenerateResult struct {
    Text         string
    FinishReason string
    ToolCalls    []provider.ToolCall
    ToolResults  []provider.ToolCall
    Usage        provider.Usage
    Steps        []StepResult
}
```

### StepResult

```go
type StepResult struct {
    StepNumber   int
    Text         string
    ToolCalls    []provider.ToolCall
    ToolResults  []provider.ToolCall
    FinishReason string
    Usage        provider.Usage
    Messages     []provider.Message
}
```
