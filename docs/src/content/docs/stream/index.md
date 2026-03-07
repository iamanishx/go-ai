---
title: Stream
description: Stream utilities for real-time responses
---

The Stream module provides utilities for handling streaming responses.

## Overview

Go AI SDK supports real-time streaming of:
- Text deltas
- Tool calls
- Tool results
- Usage information

## StreamReader

### Creating a Stream Reader

```go
reader := stream.NewStreamReader(ctx)
```

### Methods

| Method | Description |
|--------|-------------|
| `Part()` | Returns channel of stream parts |
| `Close()` | Closes the stream |
| `Err()` | Returns any error |

### StreamPart Types

```go
type StreamPart struct {
    Type           string
    Text           string
    ToolCallID     string
    ToolName       string
    ToolInput      string
    ToolInputDelta string
    ToolResult     string
    FinishReason   string
    Usage          Usage
    Error          error
}
```

## Part Types

| Type | Description |
|------|-------------|
| `text-delta` | Text content chunk |
| `text-start` | Start of text block |
| `text-end` | End of text block |
| `tool-call` | Tool call request |
| `tool-input-start` | Start of tool input |
| `tool-input-delta` | Tool input chunk |
| `tool-result` | Tool execution result |
| `finish` | Stream finished |
| `error` | Error occurred |

## Usage

### Reading Stream

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "Your prompt here",
})

for part := range stream.Part() {
    switch part.Type {
    case "text-delta":
        fmt.Print(part.Text)
    case "tool-call":
        fmt.Printf("Tool: %s\n", part.ToolName)
    case "tool-result":
        fmt.Printf("Result: %s\n", part.ToolResult)
    case "finish":
        fmt.Printf("Done: %s\n", part.FinishReason)
    }
}
```

### Accumulated Values

```go
stream, err := agent.Stream(ctx, agent.AgentCallOptions{
    Prompt: "Your prompt here",
})

// Wait for stream to complete or read in chunks
text := stream.Text()
toolCalls := stream.ToolCalls()
finishReason := stream.FinishReason()
usage := stream.Usage()
```

## TextStreamWriter

For building custom streams:

```go
writer := stream.NewTextStreamWriter()
writer.OnChunk(func(chunk string) {
    fmt.Print(chunk)
})
writer.OnFinish(func(text string) {
    fmt.Println("\nFinal:", text)
})

writer.WriteChunk("Hello ")
writer.WriteChunk("World")
writer.Close()
```

## SSE Support

The stream module includes SSE (Server-Sent Events) parsing:

```go
parts, err := stream.ParseSSEStream(reader)
for _, part := range parts {
    fmt.Println(part.Type, part.Text)
}
```
