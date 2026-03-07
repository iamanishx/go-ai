---
title: Go AI SDK
description: A Go SDK for building AI-powered applications with Large Language Models
template: splash
hero:
  tagline: A powerful, type-safe Go library for building AI-powered applications. Inspired by the Vercel AI SDK.
  actions:
    - text: Get Started
      link: /go-ai/guides/getting-started/
      icon: right-arrow
      variant: primary
    - text: View on GitHub
      link: https://github.com/iamanishx/go-ai
      icon: external
      variant: minimal
---

import { Card, CardGrid } from '@astrojs/starlight/components';

## Why Go AI SDK?

<CardGrid stagger>
	<Card title="Type Safety" icon="approve-check">
		Full type safety with Go's strong type system. Catch errors at compile time.
	</Card>
	<Card title="Provider Agnostic" icon="setting">
		Works with any LLM provider. Easily swap between different models.
	</Card>
	<Card title="Tool Loop Agent" icon="rocket">
		Built-in autonomous agent that runs tools in a loop until the task is complete.
	</Card>
	<Card title="Streaming Support" icon="list-format">
		Real-time streaming responses for text, tool calls, and tool results.
	</Card>
</CardGrid>

<br />

## Quick Example

```go
import (
    "github.com/iamanishx/go-ai/agent"
    "github.com/iamanishx/go-ai/provider/bedrock"
    "github.com/iamanishx/go-ai/provider"
)

// 1. Initialize AWS Bedrock Provider
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})

// 2. Create the Agent
agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: provider.Chat("anthropic.claude-3-sonnet-v1:0"),
    Tools: []provider.Tool{weatherTool},
    ExecuteTools: true,
})

// 3. Generate a Response
result, err := agent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})
```
