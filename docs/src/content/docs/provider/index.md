---
title: Provider
description: Learn about the provider system and available providers
---

The Provider system allows Go AI SDK to work with any LLM backend.

## Overview

Providers are modular - you can swap between different LLM services without changing your agent code.

## Available Providers

### Amazon Bedrock

Native integration with AWS Bedrock supporting:
- Anthropic Claude
- Amazon Titan
- Meta Llama
- And more...

[See Bedrock Provider →](/reference/provider/bedrock/)

## Creating a Provider

```go
import "github.com/iamanishx/go-ai/provider/bedrock"

provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

## Getting a Model

```go
// Get a chat model
model := provider.Chat("anthropic.claude-3-sonnet-v1:0")

// Use with agent
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: model,
    Tools: tools,
})
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
