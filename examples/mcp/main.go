package main

import (
	"context"
	"fmt"
	"log"
	"os"

	agentpkg "github.com/iamanishx/go-ai/agent"
	"github.com/iamanishx/go-ai/mcp"
	"github.com/iamanishx/go-ai/provider"
	"github.com/iamanishx/go-ai/provider/bedrock"
)

func main() {
	ctx := context.Background()

	mcpClient, err := mcp.CreateMCPClient(ctx, mcp.MCPClientConfig{
		Transport: mcp.NewStdioClientTransport(mcp.StdioConfig{
			Command: "npx",
			Args: []string{
				"-y",
				"@modelcontextprotocol/server-filesystem",
				".",
			},
			Stderr: os.Stderr,
		}),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer mcpClient.Close()

	mcpToolsMap, err := mcpClient.Tools(ctx)
	if err != nil {
		log.Fatal(err)
	}

	mcpTools := make([]provider.Tool, 0, len(mcpToolsMap))
	for _, tool := range mcpToolsMap {
		mcpTools = append(mcpTools, tool)
	}

	profile := os.Getenv("AWS_PROFILE")

	bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
		Region:  "us-east-1",
		Profile: profile,
	})

	ag := agentpkg.CreateToolLoopAgent(agentpkg.ToolLoopAgentSettings{
		Model:        bedrockProvider.Chat("us.anthropic.claude-sonnet-4-5-20250929-v1:0"),
		Tools:        mcpTools,
		ExecuteTools: true,
		MaxSteps:     10,
	})

	s, err := ag.Stream(ctx, agentpkg.AgentCallOptions{Prompt: "List files in current directory"})
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	for part := range s.Part() {
		switch part.Type {
		case "text-delta":
			fmt.Print(part.Text)
		case "tool-call":
			fmt.Printf("\n[tool-call: %s id=%s input=%s]\n", part.ToolName, part.ToolCallID, part.ToolInput)
		case "tool-result":
			fmt.Printf("\n[tool-result: %s]\n", part.ToolResult)
		case "finish":
			fmt.Printf("\n[finish: %s]\n", part.FinishReason)
		case "error":
			fmt.Printf("\n[error: %v]\n", part.Error)
		}
	}
}
