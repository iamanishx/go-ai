# Go AI SDK

> A Go SDK for building AI-powered applications with Large Language Models

**Go AI SDK** is a powerful, type-safe Go library for building AI Go port of the applications. It's a popular [Vercel AI SDK](https://github.com/vercel/ai).

## Why Go AI SDK?

- **Type Safety** - Full type safety with Go's strong type system
- **Provider Agnostic** - Works with any LLM provider
- **Tool Loop Agent** - Built-in autonomous agent with tool execution
- **Streaming Support** - Real-time streaming responses
- **AWS Native** - First-class support for Amazon Bedrock

## Quick Example

```go
import (
    "github.com/iamanishx/go-ai/agent"
    "github.com/iamanishx/go-ai/provider/bedrock"
    "github.com/iamanishx/go-ai/provider"
)

provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})

agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: provider.Chat("anthropic.claude-3-sonnet-v1:0"),
    Tools: []provider.Tool{weatherTool},
    ExecuteTools: true,
})

result, err := agent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})
```

## Features

| Feature | Description |
|---------|-------------|
| **Tool Loop Agent** | Autonomous agent that runs tools in a loop |
| **Provider System** | Modular provider architecture |
| **Amazon Bedrock** | Native AWS Bedrock integration |
| **Streaming** | Real-time SSE streaming |
| **Type Safety** | Full Go type safety |

## Installation

```bash
go get github.com/iamanishx/go-ai
```

## Documentation

- [Getting Started](./getting-started.md)
- [Agent](./agent/)
- [Provider](./provider/)
- [Stream](./stream/)
- [Examples](./examples/)

## Supported Providers

- [Amazon Bedrock](./provider/bedrock/) - Claude, Llama, Titan models
