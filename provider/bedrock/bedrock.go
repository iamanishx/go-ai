package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockdoc "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/iamanishx/go-ai/provider"
)

type Provider interface {
	Chat(modelID string) ChatModel
}

type ChatModel interface {
	GenerateText(ctx context.Context, opts provider.GenerateTextOptions) (provider.GenerateTextResult, error)
	StreamText(ctx context.Context, opts provider.GenerateTextOptions) (<-chan provider.StreamPart, error)
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type CredentialProvider interface {
	Retrieve(ctx context.Context) (Credentials, error)
}

type StaticCredentialProvider struct {
	creds Credentials
}

func (p *StaticCredentialProvider) Retrieve(ctx context.Context) (Credentials, error) {
	return p.creds, nil
}

type EnvCredentialProvider struct{}

func (p *EnvCredentialProvider) Retrieve(ctx context.Context) (Credentials, error) {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKeyID == "" || secretAccessKey == "" {
		return Credentials{}, fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
	}

	return Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}, nil
}

type SharedConfigCredentialProvider struct {
	Profile string
}

func (p *SharedConfigCredentialProvider) Retrieve(ctx context.Context) (Credentials, error) {
	opts := []func(*config.LoadOptions) error{}
	if p.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(p.Profile))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return Credentials{}, err
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{AccessKeyID: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey, SessionToken: creds.SessionToken}, nil
}

type WebIdentityCredentialProvider struct {
	RoleARN     string
	SessionName string
}

func (p *WebIdentityCredentialProvider) Retrieve(ctx context.Context) (Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Credentials{}, err
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{AccessKeyID: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey, SessionToken: creds.SessionToken}, nil
}

type DefaultCredentialProviderChain struct{}

func NewDefaultCredentialProviderChain() *DefaultCredentialProviderChain {
	return &DefaultCredentialProviderChain{}
}

func (c *DefaultCredentialProviderChain) Retrieve(ctx context.Context) (Credentials, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Credentials{}, err
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{AccessKeyID: creds.AccessKeyID, SecretAccessKey: creds.SecretAccessKey, SessionToken: creds.SessionToken}, nil
}

type BedrockProvider struct {
	region  string
	client  *bedrockruntime.Client
	initErr error
	mu      sync.RWMutex
}

type BedrockProviderSettings struct {
	Region             string
	AccessKeyID        string
	SecretAccessKey    string
	SessionToken       string
	APIKey             string
	BaseURL            string
	Headers            map[string]string
	Profile            string
	CredentialProvider CredentialProvider
}

func Create(settings BedrockProviderSettings) *BedrockProvider {
	region := getEnvOrDefault("AWS_REGION", settings.Region, "us-east-1")

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(region)}

	if settings.Profile != "" {
		loadOpts = append(loadOpts, config.WithSharedConfigProfile(settings.Profile))
	}

	if settings.AccessKeyID != "" && settings.SecretAccessKey != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(settings.AccessKeyID, settings.SecretAccessKey, settings.SessionToken),
		))
	}

	if settings.CredentialProvider != nil {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(&customCredentialProviderAdapter{provider: settings.CredentialProvider}))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return &BedrockProvider{region: region, initErr: err}
	}

	headers := map[string]string{}
	for k, v := range settings.Headers {
		headers[k] = v
	}
	if settings.APIKey != "" {
		headers["Authorization"] = "Bearer " + settings.APIKey
	}
	if v := os.Getenv("AWS_BEARER_TOKEN_BEDROCK"); v != "" {
		headers["Authorization"] = "Bearer " + v
	}

	client := bedrockruntime.NewFromConfig(cfg, func(o *bedrockruntime.Options) {
		if settings.BaseURL != "" {
			o.BaseEndpoint = &settings.BaseURL
		}
		if len(headers) > 0 {
			o.APIOptions = append(o.APIOptions, addHeadersMiddleware(headers))
		}
	})

	return &BedrockProvider{region: region, client: client}
}

func addHeadersMiddleware(headers map[string]string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Build.Add(middleware.BuildMiddlewareFunc("goai-bedrock-headers", func(ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
			req, ok := in.Request.(*smithyhttp.Request)
			if ok {
				for k, v := range headers {
					req.Header.Set(k, v)
				}
			}
			return next.HandleBuild(ctx, in)
		}), middleware.After)
	}
}

type customCredentialProviderAdapter struct {
	provider CredentialProvider
}

func (c *customCredentialProviderAdapter) Retrieve(ctx context.Context) (aws.Credentials, error) {
	creds, err := c.provider.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return aws.Credentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Source:          "go-ai-custom-provider",
	}, nil
}

func getEnvOrDefault(env, value, def string) string {
	if value != "" {
		return value
	}
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

func (p *BedrockProvider) Chat(modelID string) *BedrockChatModel {
	return &BedrockChatModel{provider: p, modelID: modelID}
}

type BedrockChatModel struct {
	provider *BedrockProvider
	modelID  string
}

func (m *BedrockChatModel) GenerateText(ctx context.Context, opts provider.GenerateTextOptions) (provider.GenerateTextResult, error) {
	if m.provider.initErr != nil {
		return provider.GenerateTextResult{}, m.provider.initErr
	}

	in := buildConverseInput(m.modelID, opts)
	out, err := m.provider.client.Converse(ctx, in)
	if err != nil {
		return provider.GenerateTextResult{}, err
	}

	var textBuilder strings.Builder
	toolCalls := make([]provider.ToolCall, 0)

	if msgOut, ok := out.Output.(*bedrocktypes.ConverseOutputMemberMessage); ok {
		for _, block := range msgOut.Value.Content {
			switch b := block.(type) {
			case *bedrocktypes.ContentBlockMemberText:
				textBuilder.WriteString(b.Value)
			case *bedrocktypes.ContentBlockMemberToolUse:
				input := toMap(b.Value.Input)
				id := ""
				name := ""
				if b.Value.ToolUseId != nil {
					id = *b.Value.ToolUseId
				}
				if b.Value.Name != nil {
					name = *b.Value.Name
				}
				toolCalls = append(toolCalls, provider.ToolCall{ID: id, Name: name, Input: input})
			}
		}
	}

	usage := usageFromTokenUsage(out.Usage)

	return provider.GenerateTextResult{
		Text:         textBuilder.String(),
		FinishReason: string(out.StopReason),
		ToolCalls:    toolCalls,
		Usage:        usage,
	}, nil
}

func (m *BedrockChatModel) StreamText(ctx context.Context, opts provider.GenerateTextOptions) (<-chan provider.StreamPart, error) {
	if m.provider.initErr != nil {
		return nil, m.provider.initErr
	}

	in := buildConverseStreamInput(m.modelID, opts)
	out, err := m.provider.client.ConverseStream(ctx, in)
	if err != nil {
		return nil, err
	}

	parts := make(chan provider.StreamPart, 256)

	go func() {
		defer close(parts)
		defer out.GetStream().Close()

		type toolState struct {
			id    string
			name  string
			input strings.Builder
		}

		toolStates := map[int32]*toolState{}
		finishReason := ""
		usage := provider.Usage{}

		emitToolCall := func(idx int32) {
			state, ok := toolStates[idx]
			if !ok {
				return
			}
			parts <- provider.StreamPart{
				Type:       "tool-call",
				ToolCallID: state.id,
				ToolName:   state.name,
				ToolInput:  state.input.String(),
			}
			delete(toolStates, idx)
		}

		for event := range out.GetStream().Events() {
			switch e := event.(type) {
			case *bedrocktypes.ConverseStreamOutputMemberContentBlockDelta:
				if e.Value.ContentBlockIndex == nil {
					continue
				}
				idx := *e.Value.ContentBlockIndex
				switch d := e.Value.Delta.(type) {
				case *bedrocktypes.ContentBlockDeltaMemberText:
					parts <- provider.StreamPart{Type: "text-delta", Text: d.Value}
				case *bedrocktypes.ContentBlockDeltaMemberToolUse:
					state, ok := toolStates[idx]
					if !ok {
						state = &toolState{}
						toolStates[idx] = state
					}
					if d.Value.Input != nil {
						state.input.WriteString(*d.Value.Input)
						parts <- provider.StreamPart{Type: "tool-input-delta", ToolInputDelta: *d.Value.Input}
					}
				}
			case *bedrocktypes.ConverseStreamOutputMemberContentBlockStart:
				if e.Value.ContentBlockIndex == nil {
					continue
				}
				idx := *e.Value.ContentBlockIndex
				switch s := e.Value.Start.(type) {
				case *bedrocktypes.ContentBlockStartMemberToolUse:
					state := &toolState{}
					if s.Value.ToolUseId != nil {
						state.id = *s.Value.ToolUseId
					}
					if s.Value.Name != nil {
						state.name = *s.Value.Name
					}
					toolStates[idx] = state
				}
			case *bedrocktypes.ConverseStreamOutputMemberContentBlockStop:
				if e.Value.ContentBlockIndex != nil {
					emitToolCall(*e.Value.ContentBlockIndex)
				}
			case *bedrocktypes.ConverseStreamOutputMemberMessageStop:
				finishReason = string(e.Value.StopReason)
			case *bedrocktypes.ConverseStreamOutputMemberMetadata:
				usage = usageFromTokenUsage(e.Value.Usage)
			}
		}

		for idx := range toolStates {
			emitToolCall(idx)
		}

		if err := out.GetStream().Err(); err != nil {
			parts <- provider.StreamPart{Type: "error", Error: err}
			return
		}

		parts <- provider.StreamPart{Type: "finish", FinishReason: finishReason, Usage: usage}
	}()

	return parts, nil
}

func buildConverseInput(modelID string, opts provider.GenerateTextOptions) *bedrockruntime.ConverseInput {
	inferenceConfig, additionalFields := buildInference(opts)
	in := &bedrockruntime.ConverseInput{
		ModelId:         &modelID,
		Messages:        convertMessages(opts),
		InferenceConfig: inferenceConfig,
	}
	if additionalFields != nil {
		in.AdditionalModelRequestFields = additionalFields
	}
	if opts.System != "" {
		in.System = []bedrocktypes.SystemContentBlock{&bedrocktypes.SystemContentBlockMemberText{Value: opts.System}}
	}
	if len(opts.Tools) > 0 {
		in.ToolConfig = convertToolConfig(opts.Tools, opts.ToolChoice)
	}
	return in
}

func buildConverseStreamInput(modelID string, opts provider.GenerateTextOptions) *bedrockruntime.ConverseStreamInput {
	inferenceConfig, additionalFields := buildInference(opts)
	in := &bedrockruntime.ConverseStreamInput{
		ModelId:         &modelID,
		Messages:        convertMessages(opts),
		InferenceConfig: inferenceConfig,
	}
	if additionalFields != nil {
		in.AdditionalModelRequestFields = additionalFields
	}
	if opts.System != "" {
		in.System = []bedrocktypes.SystemContentBlock{&bedrocktypes.SystemContentBlockMemberText{Value: opts.System}}
	}
	if len(opts.Tools) > 0 {
		in.ToolConfig = convertToolConfig(opts.Tools, opts.ToolChoice)
	}
	return in
}

func buildInference(opts provider.GenerateTextOptions) (*bedrocktypes.InferenceConfiguration, bedrockdoc.Interface) {
	maxTokens := int32(opts.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}
	cfg := &bedrocktypes.InferenceConfiguration{MaxTokens: &maxTokens}
	if opts.Temperature != 0 {
		t := float32(opts.Temperature)
		cfg.Temperature = &t
	}
	if opts.TopP != 0 {
		p := float32(opts.TopP)
		cfg.TopP = &p
	}
	if len(opts.StopSequences) > 0 {
		cfg.StopSequences = append([]string{}, opts.StopSequences...)
	}
	if opts.TopK > 0 {
		return cfg, bedrockdoc.NewLazyDocument(map[string]interface{}{"top_k": opts.TopK})
	}
	return cfg, nil
}

func convertMessages(opts provider.GenerateTextOptions) []bedrocktypes.Message {
	messages := make([]bedrocktypes.Message, 0, len(opts.Messages)+1)

	for _, msg := range opts.Messages {
		if len(msg.ToolResults) > 0 {
			content := make([]bedrocktypes.ContentBlock, 0, len(msg.ToolResults))
			for _, tr := range msg.ToolResults {
				id := tr.ID
				content = append(content, &bedrocktypes.ContentBlockMemberToolResult{Value: bedrocktypes.ToolResultBlock{
					ToolUseId: &id,
					Content:   []bedrocktypes.ToolResultContentBlock{&bedrocktypes.ToolResultContentBlockMemberText{Value: tr.Output}},
				}})
			}
			messages = append(messages, bedrocktypes.Message{
				Role:    bedrocktypes.ConversationRoleUser,
				Content: content,
			})
			continue
		}

		if msg.ToolCallID != "" {
			id := msg.ToolCallID
			messages = append(messages, bedrocktypes.Message{
				Role: bedrocktypes.ConversationRoleUser,
				Content: []bedrocktypes.ContentBlock{
					&bedrocktypes.ContentBlockMemberToolResult{Value: bedrocktypes.ToolResultBlock{
						ToolUseId: &id,
						Content:   []bedrocktypes.ToolResultContentBlock{&bedrocktypes.ToolResultContentBlockMemberText{Value: msg.Content}},
					}},
				},
			})
			continue
		}

		if len(msg.ToolCalls) > 0 {
			content := make([]bedrocktypes.ContentBlock, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				id := tc.ID
				name := tc.Name
				content = append(content, &bedrocktypes.ContentBlockMemberToolUse{Value: bedrocktypes.ToolUseBlock{
					ToolUseId: &id,
					Name:      &name,
					Input:     bedrockdoc.NewLazyDocument(tc.Input),
				}})
			}
			messages = append(messages, bedrocktypes.Message{Role: bedrocktypes.ConversationRoleAssistant, Content: content})
			continue
		}

		role := bedrocktypes.ConversationRoleUser
		if strings.EqualFold(msg.Role, "assistant") {
			role = bedrocktypes.ConversationRoleAssistant
		}
		messages = append(messages, bedrocktypes.Message{
			Role:    role,
			Content: []bedrocktypes.ContentBlock{&bedrocktypes.ContentBlockMemberText{Value: msg.Content}},
		})
	}

	if opts.Prompt != "" {
		messages = append(messages, bedrocktypes.Message{
			Role:    bedrocktypes.ConversationRoleUser,
			Content: []bedrocktypes.ContentBlock{&bedrocktypes.ContentBlockMemberText{Value: opts.Prompt}},
		})
	}

	return messages
}

func convertToolConfig(tools []provider.Tool, choice provider.ToolChoice) *bedrocktypes.ToolConfiguration {
	toolDefs := make([]bedrocktypes.Tool, 0, len(tools))
	for _, tool := range tools {
		name := tool.Name
		desc := tool.Description
		schema := &bedrocktypes.ToolInputSchemaMemberJson{Value: bedrockdoc.NewLazyDocument(tool.Parameters)}
		tspec := bedrocktypes.ToolSpecification{Name: &name, Description: &desc, InputSchema: schema}
		toolDefs = append(toolDefs, &bedrocktypes.ToolMemberToolSpec{Value: tspec})
	}

	cfg := &bedrocktypes.ToolConfiguration{Tools: toolDefs}

	if choice.ToolName != "" {
		name := choice.ToolName
		cfg.ToolChoice = &bedrocktypes.ToolChoiceMemberTool{Value: bedrocktypes.SpecificToolChoice{Name: &name}}
	} else if strings.EqualFold(choice.Type, "required") {
		cfg.ToolChoice = &bedrocktypes.ToolChoiceMemberAny{Value: bedrocktypes.AnyToolChoice{}}
	} else if strings.EqualFold(choice.Type, "auto") {
		cfg.ToolChoice = &bedrocktypes.ToolChoiceMemberAuto{Value: bedrocktypes.AutoToolChoice{}}
	}

	return cfg
}

func usageFromTokenUsage(u *bedrocktypes.TokenUsage) provider.Usage {
	if u == nil {
		return provider.Usage{}
	}
	usage := provider.Usage{}
	if u.InputTokens != nil {
		usage.InputTokens = int(*u.InputTokens)
	}
	if u.OutputTokens != nil {
		usage.OutputTokens = int(*u.OutputTokens)
	}
	if u.TotalTokens != nil {
		usage.TotalTokens = int(*u.TotalTokens)
	} else {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	return usage
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
