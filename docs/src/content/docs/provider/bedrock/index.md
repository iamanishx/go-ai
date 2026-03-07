---
title: Amazon Bedrock
description: AWS Bedrock provider documentation
---

Amazon Bedrock is a fully managed service that offers a choice of high-performing foundation models.

## Setup

### Installation

No additional dependencies required.

### Authentication

The Bedrock provider supports multiple authentication methods:

#### 1. AWS Profile (Recommended)

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile", // loads from ~/.aws/credentials
})
```

#### 2. Environment Variables

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

#### 3. Static Credentials

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:          "us-east-1",
    AccessKeyID:     "YOUR_ACCESS_KEY_ID",
    SecretAccessKey: "YOUR_SECRET_ACCESS_KEY",
    SessionToken:    "OPTIONAL_SESSION_TOKEN",
})
```

#### 4. Custom Credential Provider

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region: "us-east-1",
    CredentialProvider: &bedrock.SharedConfigCredentialProvider{
        Profile: "myprofile",
    },
})
```

#### 5. Default Chain

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:            "us-east-1",
    CredentialProvider: bedrock.NewDefaultCredentialProviderChain(),
})
```

## Supported Models

### Anthropic Claude

```go
model := provider.Chat("anthropic.claude-3-opus-20240229-v1:0")
model := provider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")
model := provider.Chat("anthropic.claude-3-haiku-20240307-v1:0")
```

### Amazon Titan

```go
model := provider.Chat("amazon.titan-text-premium-v1:0")
```

### Meta Llama

```go
model := provider.Chat("meta.llama3-70b-instruct-v1:0")
model := provider.Chat("meta.llama3-8b-instruct-v1:0")
```

## Options

| Option | Type | Description |
|--------|------|-------------|
| `Region` | `string` | AWS region (default: us-east-1) |
| `Profile` | `string` | AWS profile name |
| `AccessKeyID` | `string` | AWS access key ID |
| `SecretAccessKey` | `string` | AWS secret access key |
| `SessionToken` | `string` | AWS session token |
| `APIKey` | `string` | Bearer token (for API key auth) |
| `BaseURL` | `string` | Custom endpoint URL |
| `Headers` | `map[string]string` | Custom headers |
| `CredentialProvider` | `CredentialProvider` | Custom credential provider |

## Credential Providers

### Built-in Providers

- `EnvCredentialProvider` - Loads from environment variables
- `SharedConfigCredentialProvider` - Loads from `~/.aws/config` and `~/.aws/credentials`
- `WebIdentityCredentialProvider` - Loads from Web Identity Token
- `StaticCredentialProvider` - Static credentials
- `DefaultCredentialProviderChain` - Tries env → shared config

### Custom Provider

Implement the `CredentialProvider` interface:

```go
type CredentialProvider interface {
    Retrieve(ctx context.Context) (Credentials, error)
}

type Credentials struct {
    AccessKeyID     string
    SecretAccessKey string
    SessionToken    string
}
```

## Usage with Agent

```go
provider := bedrock.Create(bedrock.BedrockProviderSettings{
    Region:  "us-east-1",
    Profile: "myprofile",
})

model := provider.Chat("anthropic.claude-3-sonnet-v1:0")

agent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
    Model:        model,
    Tools:        []provider.Tool{weatherTool},
    ExecuteTools: true,
})
```

## Direct Model Usage

### Generate Text

```go
result, err := model.GenerateText(ctx, provider.GenerateTextOptions{
    Prompt:  "Hello, how are you?",
    System:  "You are a helpful assistant.",
    MaxTokens: 1000,
    Temperature: 0.7,
})
```

### Stream Text

```go
stream, err := model.StreamText(ctx, provider.GenerateTextOptions{
    Prompt: "Hello, how are you?",
})

for part := range stream {
    fmt.Print(part.Text)
}
```
