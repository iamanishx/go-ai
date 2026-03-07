package provider

import "context"

type LanguageModel interface {
	GenerateText(ctx context.Context, opts GenerateTextOptions) (GenerateTextResult, error)
	StreamText(ctx context.Context, opts GenerateTextOptions) (<-chan StreamPart, error)
}

type ChatModel = LanguageModel
type Model = LanguageModel

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

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type GenerateTextResult struct {
	Text             string
	FinishReason     string
	ToolCalls        []ToolCall
	ToolResults      []ToolCall
	Usage            Usage
	ResponseMessages []Message
}

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type ToolCall struct {
	ID     string
	Name   string
	Input  map[string]interface{}
	Output string
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Execute     func(input map[string]interface{}) (string, error)
}

type ToolChoice struct {
	Type     string
	ToolName string
}

type GenerateTextOptions struct {
	Model         string
	Prompt        string
	System        string
	Messages      []Message
	Tools         []Tool
	ToolChoice    ToolChoice
	MaxTokens     int
	Temperature   float64
	TopP          float64
	TopK          int
	StopSequences []string
	Seed          *int
	MaxRetries    int
	Timeout       int
	AbortSignal   <-chan struct{}
}
