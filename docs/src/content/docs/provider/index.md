---
title: Provider
description: Provider model interfaces and available implementations
---

The provider layer defines model interfaces and concrete provider implementations.

Install provider interfaces only:

```bash
go get github.com/iamanishx/go-ai/provider
```

## Overview

Providers are modular, and the agent consumes models through the shared `GenerateText` and `StreamText` interface.

## Available Providers

### Amazon Bedrock

Native integration with AWS Bedrock supporting:
- Anthropic Claude
- Amazon Titan
- Meta Llama
- And more...

[See Bedrock Provider ->](/go-ai/provider/bedrock/)

## Creating a Provider

```go
import (
    "github.com/iamanishx/go-ai/agent"
    "github.com/iamanishx/go-ai/provider/bedrock"
)

provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

## Getting a Model

```go
// Get a chat model
model := provider.Chat("anthropic.claude-3-sonnet-v1:0")

// Use with agent
toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: model,
    Tools: tools,
})

_ = toolAgent
```

## Adding New Providers

To add a new provider, implement the `ChatModel` interface:

```go
type ChatModel interface {
    GenerateText(ctx context.Context, opts GenerateTextOptions) (GenerateTextResult, error)
    StreamText(ctx context.Context, opts GenerateTextOptions) (<-chan StreamPart, error)
}
```

See the [Bedrock provider](https://github.com/iamanishx/go-ai/tree/main/provider/bedrock) for reference implementation.
