package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/iamanishx/go-ai/provider"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPClientConfig struct {
	Transport       sdkmcp.Transport
	Name            string
	Version         string
	OnUncaughtError func(error)
}

type Client struct {
	client          *sdkmcp.Client
	session         *sdkmcp.ClientSession
	onUncaughtError func(error)

	mu     sync.Mutex
	closed bool
}

func CreateMCPClient(ctx context.Context, cfg MCPClientConfig) (*Client, error) {
	if cfg.Transport == nil {
		return nil, fmt.Errorf("transport is required")
	}

	name := cfg.Name
	if name == "" {
		name = "go-ai-mcp-client"
	}

	version := cfg.Version
	if version == "" {
		version = "1.0.0"
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: name, Version: version}, nil)
	session, err := client.Connect(ctx, cfg.Transport, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:          client,
		session:         session,
		onUncaughtError: cfg.OnUncaughtError,
	}, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.session.Close()
}

func (c *Client) ListTools(ctx context.Context, cursor string) (*sdkmcp.ListToolsResult, error) {
	params := &sdkmcp.ListToolsParams{}
	if cursor != "" {
		params.Cursor = cursor
	}
	return c.session.ListTools(ctx, params)
}

func (c *Client) Tools(ctx context.Context) (map[string]provider.Tool, error) {
	all := &sdkmcp.ListToolsResult{Tools: []*sdkmcp.Tool{}}
	cursor := ""

	for {
		res, err := c.ListTools(ctx, cursor)
		if err != nil {
			return nil, err
		}
		all.Tools = append(all.Tools, res.Tools...)
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return c.ToolsFromDefinitions(all), nil
}

func (c *Client) ToolsFromDefinitions(definitions *sdkmcp.ListToolsResult) map[string]provider.Tool {
	tools := make(map[string]provider.Tool)
	if definitions == nil {
		return tools
	}

	for _, toolDef := range definitions.Tools {
		if toolDef == nil || toolDef.Name == "" {
			continue
		}

		name := toolDef.Name
		description := toolDef.Description
		parameters := anyToMap(toolDef.InputSchema)

		toolName := name
		self := c
		tools[name] = provider.Tool{
			Name:        name,
			Description: description,
			Parameters:  parameters,
			Execute: func(input map[string]interface{}) (string, error) {
				if input == nil {
					input = map[string]interface{}{}
				}

				res, err := self.session.CallTool(context.Background(), &sdkmcp.CallToolParams{
					Name:      toolName,
					Arguments: input,
				})
				if err != nil {
					self.report(err)
					return "", err
				}

				return callToolResultToString(res), nil
			},
		}
	}

	return tools
}

func (c *Client) ListResources(ctx context.Context, cursor string) (*sdkmcp.ListResourcesResult, error) {
	params := &sdkmcp.ListResourcesParams{}
	if cursor != "" {
		params.Cursor = cursor
	}
	return c.session.ListResources(ctx, params)
}

func (c *Client) ReadResource(ctx context.Context, uri string) (*sdkmcp.ReadResourceResult, error) {
	return c.session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: uri})
}

func (c *Client) ListResourceTemplates(ctx context.Context, cursor string) (*sdkmcp.ListResourceTemplatesResult, error) {
	params := &sdkmcp.ListResourceTemplatesParams{}
	if cursor != "" {
		params.Cursor = cursor
	}
	return c.session.ListResourceTemplates(ctx, params)
}

func (c *Client) ListPrompts(ctx context.Context, cursor string) (*sdkmcp.ListPromptsResult, error) {
	params := &sdkmcp.ListPromptsParams{}
	if cursor != "" {
		params.Cursor = cursor
	}
	return c.session.ListPrompts(ctx, params)
}

func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*sdkmcp.GetPromptResult, error) {
	return c.session.GetPrompt(ctx, &sdkmcp.GetPromptParams{Name: name, Arguments: args})
}

func (c *Client) report(err error) {
	if err != nil && c.onUncaughtError != nil {
		c.onUncaughtError(err)
	}
}

func anyToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func callToolResultToString(res *sdkmcp.CallToolResult) string {
	if res == nil {
		return ""
	}

	if res.StructuredContent != nil {
		if data, err := json.Marshal(res.StructuredContent); err == nil {
			return string(data)
		}
	}

	parts := make([]string, 0, len(res.Content))
	for _, content := range res.Content {
		switch v := content.(type) {
		case *sdkmcp.TextContent:
			parts = append(parts, v.Text)
		default:
			if data, err := json.Marshal(v); err == nil {
				parts = append(parts, string(data))
			}
		}
	}

	return strings.Join(parts, "\n")
}
