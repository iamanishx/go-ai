package bedrock

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	profile := p.Profile
	if profile == "" {
		profile = os.Getenv("AWS_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	configPath := filepath.Join(os.Getenv("HOME"), ".aws", "config")
	credentialsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")

	creds, err := loadCredentialsFromFile(credentialsPath, profile)
	if err == nil {
		return creds, nil
	}

	creds, err = loadCredentialsFromFile(configPath, profile)
	if err == nil {
		return creds, nil
	}

	return Credentials{}, fmt.Errorf("could not load credentials for profile %s", profile)
}

func loadCredentialsFromFile(path, profile string) (Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, err
	}

	lines := strings.Split(string(data), "\n")
	inSection := false
	creds := Credentials{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(strings.Trim(line, "[]"))
			inSection = section == profile || section == "profile "+profile
			continue
		}

		if !inSection {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "aws_access_key_id":
			creds.AccessKeyID = value
		case "aws_secret_access_key":
			creds.SecretAccessKey = value
		case "aws_session_token":
			creds.SessionToken = value
		}
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return Credentials{}, fmt.Errorf("credentials not found for profile %s", profile)
	}

	return creds, nil
}

type WebIdentityCredentialProvider struct {
	RoleARN     string
	SessionName string
}

func (p *WebIdentityCredentialProvider) Retrieve(ctx context.Context) (Credentials, error) {
	roleARN := os.Getenv("AWS_ROLE_ARN")
	sessionName := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")

	if roleARN == "" || sessionName == "" {
		return Credentials{}, fmt.Errorf("AWS_ROLE_ARN and AWS_WEB_IDENTITY_TOKEN_FILE must be set")
	}

	tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return Credentials{}, err
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	stsEndpoint := "https://sts." + region + ".amazonaws.com"
	stsReq, _ := http.NewRequestWithContext(ctx, "GET", stsEndpoint+"/?Action=AssumeRoleWithWebIdentity&RoleSessionName=go-ai-session&WebIdentityToken="+string(token)+"&Version=2011-06-15&RoleArn="+roleARN, nil)

	resp, err := http.DefaultClient.Do(stsReq)
	if err != nil {
		return Credentials{}, err
	}
	defer resp.Body.Close()

	return Credentials{
		AccessKeyID:     "temp",
		SecretAccessKey: "temp",
		SessionToken:    "temp",
	}, nil
}

type DefaultCredentialProviderChain struct {
	providers []CredentialProvider
}

func NewDefaultCredentialProviderChain() *DefaultCredentialProviderChain {
	return &DefaultCredentialProviderChain{
		providers: []CredentialProvider{
			&EnvCredentialProvider{},
			&SharedConfigCredentialProvider{},
		},
	}
}

func (c *DefaultCredentialProviderChain) Retrieve(ctx context.Context) (Credentials, error) {
	var lastErr error
	for _, p := range c.providers {
		creds, err := p.Retrieve(ctx)
		if err == nil {
			return creds, nil
		}
		lastErr = err
	}
	return Credentials{}, lastErr
}

type BedrockProvider struct {
	region             string
	credentials        Credentials
	credentialProvider CredentialProvider
	apiKey             string
	baseURL            string
	headers            map[string]string
	fetch              func(req *http.Request) (*http.Response, error)
	mu                 sync.RWMutex
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
	apiKey := getEnvOrDefault("AWS_BEARER_TOKEN_BEDROCK", settings.APIKey, "")

	p := &BedrockProvider{
		region:             region,
		apiKey:             apiKey,
		baseURL:            settings.BaseURL,
		headers:            settings.Headers,
		fetch:              http.DefaultClient.Do,
		credentialProvider: settings.CredentialProvider,
	}

	if settings.CredentialProvider != nil {
		p.credentials = Credentials{
			AccessKeyID:     settings.AccessKeyID,
			SecretAccessKey: settings.SecretAccessKey,
			SessionToken:    settings.SessionToken,
		}
	} else if settings.AccessKeyID != "" && settings.SecretAccessKey != "" {
		p.credentials = Credentials{
			AccessKeyID:     settings.AccessKeyID,
			SecretAccessKey: settings.SecretAccessKey,
			SessionToken:    settings.SessionToken,
		}
	} else if settings.Profile != "" {
		p.credentialProvider = &SharedConfigCredentialProvider{Profile: settings.Profile}
	} else {
		p.credentialProvider = NewDefaultCredentialProviderChain()
	}

	if p.baseURL == "" {
		p.baseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)
	}

	return p
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
	return &BedrockChatModel{
		provider: p,
		modelID:  modelID,
	}
}

type BedrockChatModel struct {
	provider *BedrockProvider
	modelID  string
}

type BedrockRequest struct {
	AnthropicVersion string                   `json:"anthropic_version"`
	Messages         []map[string]interface{} `json:"messages"`
	System           string                   `json:"system,omitempty"`
	MaxTokens        int                      `json:"max_tokens,omitempty"`
	Temperature      float64                  `json:"temperature,omitempty"`
	TopP             float64                  `json:"top_p,omitempty"`
	TopK             int                      `json:"top_k,omitempty"`
	StopSequences    []string                 `json:"stop_sequences,omitempty"`
	Tools            []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice       map[string]interface{}   `json:"tool_choice,omitempty"`
}

type BedrockResponse struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Role         string                   `json:"role"`
	Content      []map[string]interface{} `json:"content"`
	Model        string                   `json:"model"`
	StopReason   string                   `json:"stop_reason"`
	StopSequence string                   `json:"stop_sequence"`
	Usage        map[string]int           `json:"usage"`
}

func (m *BedrockChatModel) GenerateText(ctx context.Context, opts provider.GenerateTextOptions) (provider.GenerateTextResult, error) {
	streamParts, err := m.StreamText(ctx, opts)
	if err != nil {
		return provider.GenerateTextResult{}, err
	}

	var text strings.Builder
	var toolCalls []provider.ToolCall
	var finishReason string
	var usage provider.Usage

	for part := range streamParts {
		if part.Type == "error" {
			return provider.GenerateTextResult{}, part.Error
		} else if part.Type == "text-delta" {
			text.WriteString(part.Text)
		} else if part.Type == "tool-call" {
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:   part.ToolCallID,
				Name: part.ToolName,
			})
		} else if part.Type == "finish" {
			finishReason = part.FinishReason
			usage = part.Usage
		}
	}

	return provider.GenerateTextResult{
		Text:         text.String(),
		FinishReason: finishReason,
		ToolCalls:    toolCalls,
		Usage:        usage,
	}, nil
}

func (m *BedrockChatModel) StreamText(ctx context.Context, opts provider.GenerateTextOptions) (<-chan provider.StreamPart, error) {
	body := BedrockRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		Messages:         convertMessages(opts),
		MaxTokens:        opts.MaxTokens,
		Temperature:      opts.Temperature,
		TopP:             opts.TopP,
		TopK:             opts.TopK,
		StopSequences:    opts.StopSequences,
	}

	if body.MaxTokens == 0 {
		body.MaxTokens = 4096
	}

	if opts.System != "" {
		body.System = opts.System
	}

	if len(opts.Tools) > 0 {
		body.Tools = convertTools(opts.Tools)
		if opts.ToolChoice.ToolName != "" {
			body.ToolChoice = map[string]interface{}{
				"type": "tool",
				"name": opts.ToolChoice.ToolName,
			}
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/model/%s/invoke", m.provider.baseURL, m.modelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if m.provider.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.provider.apiKey))
	} else {
		creds := m.provider.credentials
		if m.provider.credentialProvider != nil && (creds.AccessKeyID == "" || creds.SecretAccessKey == "") {
			if c, err := m.provider.credentialProvider.Retrieve(ctx); err == nil {
				creds = c
			}
		}
		signRequest(req, m.provider.region, creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
	}

	for k, v := range m.provider.headers {
		req.Header.Set(k, v)
	}

	parts := make(chan provider.StreamPart, 100)

	go func() {
		defer close(parts)

		resp, err := m.provider.fetch(req)
		if err != nil {
			parts <- provider.StreamPart{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			parts <- provider.StreamPart{
				Type:  "error",
				Error: fmt.Errorf("bedrock request failed with status %d: %s", resp.StatusCode, string(body)),
			}
			return
		}

		var bedrockResp BedrockResponse
		if err := json.NewDecoder(resp.Body).Decode(&bedrockResp); err != nil {
			parts <- provider.StreamPart{Type: "error", Error: err}
			return
		}

		for _, content := range bedrockResp.Content {
			if content["type"] == "text" {
				if text, ok := content["text"].(string); ok {
					parts <- provider.StreamPart{
						Type: "text-delta",
						Text: text,
					}
				}
			} else if content["type"] == "tool_use" {
				id, _ := content["id"].(string)
				name, _ := content["name"].(string)
				input, _ := json.Marshal(content["input"])
				parts <- provider.StreamPart{
					Type:       "tool-call",
					ToolCallID: id,
					ToolName:   name,
					ToolInput:  string(input),
				}
			}
		}

		usage := provider.Usage{
			InputTokens:  bedrockResp.Usage["input_tokens"],
			OutputTokens: bedrockResp.Usage["output_tokens"],
			TotalTokens:  bedrockResp.Usage["input_tokens"] + bedrockResp.Usage["output_tokens"],
		}

		parts <- provider.StreamPart{
			Type:         "finish",
			FinishReason: bedrockResp.StopReason,
			Usage:        usage,
		}
	}()

	return parts, nil
}

func convertMessages(opts provider.GenerateTextOptions) []map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	for _, msg := range opts.Messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]interface{}, 0)
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":    tc.ID,
					"name":  tc.Name,
					"input": tc.Input,
				})
			}
			m["content"] = toolCalls
		}
		if msg.ToolCallID != "" {
			resultContent := []map[string]interface{}{
				{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.Content,
				},
			}
			m["content"] = resultContent
		}
		messages = append(messages, m)
	}

	if opts.Prompt != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": opts.Prompt,
		})
	}

	return messages
}

func convertTools(tools []provider.Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.Parameters,
		})
	}
	return result
}

func signRequest(req *http.Request, region, accessKeyID, secretAccessKey, sessionToken string) {
	service := "bedrock"
	host := req.URL.Host
	endpoint := req.URL.Path
	method := req.Method

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Content-Type", "application/json")

	payloadHash := sha256Hash([]byte{})

	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
		payloadHash = sha256Hash(body)
	}

	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	var canonicalHeaders string
	var signedHeaders string

	if sessionToken != "" {
		canonicalHeaders = fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\nx-amz-security-token:%s\n",
			req.Header.Get("Content-Type"),
			host,
			payloadHash,
			amzDate,
			sessionToken,
		)
		signedHeaders = "content-type;host;x-amz-content-sha256;x-amz-date;x-amz-security-token"
	} else {
		canonicalHeaders = fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
			req.Header.Get("Content-Type"),
			host,
			payloadHash,
			amzDate,
		)
		signedHeaders = "content-type;host;x-amz-content-sha256;x-amz-date"
	}

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		endpoint,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	hashedCanonicalRequest := sha256Hash([]byte(canonicalRequest))

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		amzDate,
		credentialScope,
		string(hashedCanonicalRequest),
	)

	signingKey := getSignatureKey(secretAccessKey, dateStamp, region, service)
	signature := hmacSHA256Hex(signingKey, []byte(stringToSign))

	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		accessKeyID,
		credentialScope,
		signedHeaders,
		signature,
	)

	req.Header.Set("Authorization", authHeader)
}

func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func hmacSHA256Hex(key []byte, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getSignatureKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
