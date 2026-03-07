package ai

import (
	"github.com/iamanishx/go-ai/agent"
	"github.com/iamanishx/go-ai/provider"
	"github.com/iamanishx/go-ai/provider/bedrock"
	"github.com/iamanishx/go-ai/stream"
)

type Provider = bedrock.Provider
type ChatModel = bedrock.ChatModel
type BedrockProviderSettings = bedrock.BedrockProviderSettings
type BedrockProvider = bedrock.BedrockProvider

func CreateBedrock(settings bedrock.BedrockProviderSettings) *bedrock.BedrockProvider {
	return bedrock.Create(settings)
}

type Credentials = bedrock.Credentials
type CredentialProvider = bedrock.CredentialProvider
type StaticCredentialProvider = bedrock.StaticCredentialProvider
type EnvCredentialProvider = bedrock.EnvCredentialProvider
type SharedConfigCredentialProvider = bedrock.SharedConfigCredentialProvider
type WebIdentityCredentialProvider = bedrock.WebIdentityCredentialProvider
type DefaultCredentialProviderChain = bedrock.DefaultCredentialProviderChain

func NewDefaultCredentialProviderChain() *bedrock.DefaultCredentialProviderChain {
	return bedrock.NewDefaultCredentialProviderChain()
}

type ToolLoopAgent = agent.ToolLoopAgent
type ToolLoopAgentSettings = agent.ToolLoopAgentSettings

func CreateToolLoopAgent(settings agent.ToolLoopAgentSettings) *agent.ToolLoopAgent {
	return agent.CreateToolLoopAgent(settings)
}

type GenerateTextOptions = provider.GenerateTextOptions
type GenerateTextResult = provider.GenerateTextResult
type Message = provider.Message
type ToolCall = provider.ToolCall
type Tool = provider.Tool
type ToolChoice = provider.ToolChoice
type Usage = provider.Usage

type StreamReader = stream.StreamReader
type TextStreamWriter = stream.TextStreamWriter
