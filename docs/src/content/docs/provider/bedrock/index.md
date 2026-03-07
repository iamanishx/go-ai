---
title: Amazon Bedrock
description: Configure Bedrock models, auth methods, and generation options
---

Amazon Bedrock is the current provider implementation in this project.

This provider now uses the official AWS SDK for Go v2 for credentials, SigV4 signing, non-stream requests (`Converse`), and real-time streaming (`ConverseStream`).

## Create Provider

```go
import "github.com/iamanishx/go-ai/provider/bedrock"

bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})
```

## Authentication Options

### AWS profile

```go
bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})
```

### Environment variables

```bash
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
```

```go
bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
})
```

### Static credentials

```go
bedrock.Create(bedrock.BedrockProviderSettings{
    Region:          "us-east-1",
    AccessKeyID:     "YOUR_ACCESS_KEY_ID",
    SecretAccessKey: "YOUR_SECRET_ACCESS_KEY",
    SessionToken:    "OPTIONAL_SESSION_TOKEN",
})
```

### Custom credential provider

```go
bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
    CredentialProvider: &bedrock.SharedConfigCredentialProvider{
        Profile: "myprofile",
    },
})
```

### Default chain

```go
bedrock.Create(bedrock.BedrockProviderSettings{
    Region:             "us-east-1",
    CredentialProvider: bedrock.NewDefaultCredentialProviderChain(),
})
```

Credential resolution follows AWS SDK defaults when explicit settings are not provided.

## Generate Text

```go
model := bedrockProvider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")

result, err := model.GenerateText(ctx, provider.GenerateTextOptions{
    Prompt:      "Hello, how are you?",
    System:      "You are a helpful assistant.",
    MaxTokens:   1000,
    Temperature: 0.7,
})

if err != nil {
    panic(err)
}

_ = result.Text
```

## Stream Text

```go
stream, err := model.StreamText(ctx, provider.GenerateTextOptions{
    Prompt: "Write a short poem about Go",
})

if err != nil {
    panic(err)
}

for part := range stream {
    switch part.Type {
    case "text-delta":
        fmt.Print(part.Text)
    case "tool-call":
        fmt.Printf("\n[tool call: %s]\n", part.ToolName)
    case "finish":
        fmt.Printf("\n[finish: %s]\n", part.FinishReason)
    case "error":
        fmt.Printf("\n[error: %v]\n", part.Error)
    }
}
```

## Supported Settings

| Option | Type | Description |
|--------|------|-------------|
| `Region` | `string` | AWS region (defaults to `us-east-1`) |
| `Profile` | `string` | AWS profile name |
| `AccessKeyID` | `string` | Static access key ID |
| `SecretAccessKey` | `string` | Static secret access key |
| `SessionToken` | `string` | Optional session token |
| `APIKey` | `string` | Bearer token alternative |
| `BaseURL` | `string` | Custom Bedrock endpoint |
| `Headers` | `map[string]string` | Extra request headers |
| `CredentialProvider` | `CredentialProvider` | Custom credentials source |

When `GenerateTextOptions.MaxTokens` is not set, the Bedrock request defaults to `4096`.

`StreamText` uses Bedrock `ConverseStream`, so `text-delta` events are emitted incrementally.

## Built-in Credential Providers

- `EnvCredentialProvider`
- `SharedConfigCredentialProvider`
- `WebIdentityCredentialProvider`
- `StaticCredentialProvider`
- `DefaultCredentialProviderChain`
