---
title: Go AI SDK
description: Go-first toolkit for Bedrock models, tool loops, and streaming
template: splash
hero:
  tagline: Build Bedrock-powered AI workflows in Go with tool loops and streaming.
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

## Why this project

- Bedrock provider with multiple auth modes (profile, env, static creds, custom provider)
- Tool loop agent that can run with tools enabled or disabled
- Streaming APIs for both direct model calls and agent flows
- Simple package surface for `go get` usage in backend services

## Quick Example

```go
import (
    "github.com/iamanishx/go-ai/agent"
    "github.com/iamanishx/go-ai/provider"
    "github.com/iamanishx/go-ai/provider/bedrock"
)

bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})

toolAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model: bedrockProvider.Chat("anthropic.claude-3-sonnet-20240229-v1:0"),
    Tools: []provider.Tool{
        {
            Name:        "get_weather",
            Description: "Get weather for a location",
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "location": map[string]interface{}{"type": "string"},
                },
                "required": []string{"location"},
            },
        },
    },
    ExecuteTools: true,
})

result, err := toolAgent.Generate(ctx, agent.AgentCallOptions{
    Prompt: "What's the weather in San Francisco?",
})

if err != nil {
    panic(err)
}

_ = result.Text
```

## Read next

- [Getting Started](/go-ai/guides/getting-started/)
- [Agent](/go-ai/agent/)
- [Amazon Bedrock Provider](/go-ai/provider/bedrock/)
