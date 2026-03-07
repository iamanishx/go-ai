package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ai-sdk/go-ai/agent"
	"github.com/ai-sdk/go-ai/provider"
	"github.com/ai-sdk/go-ai/provider/bedrock"
)

func main() {
	ctx := context.Background()

	bedrockProvider := bedrock.Create(bedrock.BedrockProviderSettings{
		Region:  "us-east-1",
		Profile: "myprofile",
	})

	chatModel := bedrockProvider.Chat("anthropic.claude-3-sonnet-20240229-v1:0")

	weatherTool := provider.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The location to get weather for",
				},
			},
			"required": []string{"location"},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			location := input["location"].(string)
			return fmt.Sprintf("The weather in %s is sunny and 72°F", location), nil
		},
	}

	calcTool := provider.Tool{
		Name:        "calculate",
		Description: "Perform a calculation",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "The math expression to calculate",
				},
			},
			"required": []string{"expression"},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			expr := input["expression"].(string)
			return fmt.Sprintf("Result of %s = 42", expr), nil
		},
	}

	toolLoopAgent := agent.CreateToolLoopAgent(agent.ToolLoopAgentSettings{
		Model:        chatModel,
		Tools:        []provider.Tool{weatherTool, calcTool},
		ExecuteTools: true,
		MaxSteps:     10,
		OnStepFinish: func(event agent.OnStepFinishEvent) {
			fmt.Printf("Step %d finished: %s\n", event.StepNumber, event.Text)
			if len(event.ToolCalls) > 0 {
				fmt.Printf("Tool calls: %v\n", event.ToolCalls)
			}
		},
	})

	result, err := toolLoopAgent.Generate(ctx, agent.AgentCallOptions{
		Prompt: "What's the weather in San Francisco?",
		System: "You are a helpful assistant that can use tools to answer questions.",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== Non-streaming result ===\n")
	fmt.Printf("Final result: %s\n", result.Text)
	fmt.Printf("Finish reason: %s\n", result.FinishReason)
	fmt.Printf("Total steps: %d\n", len(result.Steps))

	fmt.Printf("\n=== Streaming result ===\n")
	stream, err := toolLoopAgent.Stream(ctx, agent.AgentCallOptions{
		Prompt: "Calculate 5 + 7 and tell me the weather in Tokyo",
		System: "You are a helpful assistant that can use tools to answer questions.",
	})

	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	for part := range stream.Part() {
		switch part.Type {
		case "text-delta":
			fmt.Print(part.Text)
		case "tool-call":
			fmt.Printf("\n[Tool call: %s]\n", part.ToolName)
		case "tool-result":
			fmt.Printf("[Tool result: %s]\n", part.ToolResult)
		case "finish":
			fmt.Printf("\n[Finish: %s, Usage: %+v]\n", part.FinishReason, part.Usage)
		}
	}

	fmt.Printf("\nFinal text: %s\n", stream.Text())
}
