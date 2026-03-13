package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/iamanishx/go-ai/provider"
	"github.com/iamanishx/go-ai/stream"
)

type stringBuilder struct {
	buf []string
}

func (b *stringBuilder) WriteString(s string) {
	b.buf = append(b.buf, s)
}

func (b *stringBuilder) String() string {
	result := ""
	for _, s := range b.buf {
		result += s
	}
	return result
}

type ToolLoopAgent struct {
	id             string
	model          provider.Model
	tools          map[string]provider.Tool
	stopConditions []StopCondition
	executeTools   bool

	onStart          func(event OnStartEvent)
	onStepStart      func(event OnStepStartEvent)
	onStepFinish     func(event OnStepFinishEvent)
	onToolCallStart  func(event OnToolCallStartEvent)
	onToolCallFinish func(event OnToolCallFinishEvent)
	onFinish         func(event OnFinishEvent)

	maxSteps int
}

type ToolLoopAgentSettings struct {
	Model        provider.Model
	Tools        []provider.Tool
	ExecuteTools bool
	ID           string

	OnStart          func(event OnStartEvent)
	OnStepStart      func(event OnStepStartEvent)
	OnStepFinish     func(event OnStepFinishEvent)
	OnToolCallStart  func(event OnToolCallStartEvent)
	OnToolCallFinish func(event OnToolCallFinishEvent)
	OnFinish         func(event OnFinishEvent)

	StopWhen []StopCondition
	MaxSteps int
}

type OnStartEvent struct {
	Model    string
	Prompt   string
	System   string
	Messages []provider.Message
}

type OnStepStartEvent struct {
	StepNumber int
	Model      string
}

type OnStepFinishEvent struct {
	StepNumber   int
	Text         string
	ToolCalls    []provider.ToolCall
	ToolResults  []provider.ToolCall
	FinishReason string
	Usage        provider.Usage
}

type OnToolCallStartEvent struct {
	ToolCallID string
	ToolName   string
	Input      map[string]interface{}
}

type OnToolCallFinishEvent struct {
	ToolCallID string
	ToolName   string
	Output     string
	Error      error
}

type OnFinishEvent struct {
	Text         string
	ToolCalls    []provider.ToolCall
	ToolResults  []provider.ToolCall
	FinishReason string
	Usage        provider.Usage
	Steps        []StepResult
}

type StepResult struct {
	StepNumber   int
	Text         string
	ToolCalls    []provider.ToolCall
	ToolResults  []provider.ToolCall
	FinishReason string
	Usage        provider.Usage
	Messages     []provider.Message
}

type StopCondition interface {
	ShouldStop(stepResult StepResult) bool
}

type stepCountStopCondition struct {
	maxSteps int
}

func StepCountIs(maxSteps int) StopCondition {
	return &stepCountStopCondition{maxSteps: maxSteps}
}

func (s *stepCountStopCondition) ShouldStop(result StepResult) bool {
	return result.StepNumber >= s.maxSteps
}

func CreateToolLoopAgent(settings ToolLoopAgentSettings) *ToolLoopAgent {
	tools := make(map[string]provider.Tool)
	for _, t := range settings.Tools {
		tools[t.Name] = t
	}

	stopConditions := settings.StopWhen
	if len(stopConditions) == 0 {
		stopConditions = []StopCondition{StepCountIs(20)}
	}

	maxSteps := settings.MaxSteps
	if maxSteps == 0 {
		maxSteps = 20
	}

	return &ToolLoopAgent{
		id:               settings.ID,
		model:            settings.Model,
		tools:            tools,
		executeTools:     settings.ExecuteTools,
		stopConditions:   stopConditions,
		maxSteps:         maxSteps,
		onStart:          settings.OnStart,
		onStepStart:      settings.OnStepStart,
		onStepFinish:     settings.OnStepFinish,
		onToolCallStart:  settings.OnToolCallStart,
		onToolCallFinish: settings.OnToolCallFinish,
		onFinish:         settings.OnFinish,
	}
}

func (a *ToolLoopAgent) ID() string {
	return a.id
}

func (a *ToolLoopAgent) Tools() map[string]provider.Tool {
	return a.tools
}

func (a *ToolLoopAgent) Generate(ctx context.Context, opts AgentCallOptions) (AgentGenerateResult, error) {
	return a.generate(ctx, opts, false)
}

func (a *ToolLoopAgent) Stream(ctx context.Context, opts AgentCallOptions) (*AgentStreamReader, error) {
	reader, err := a.stream(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &AgentStreamReader{
		reader: reader,
	}, nil
}

type AgentStreamReader struct {
	reader       *stream.StreamReader
	mu           sync.RWMutex
	text         strings.Builder
	toolCalls    []provider.ToolCall
	toolResults  []provider.ToolCall
	steps        []StepResult
	finishReason string
	usage        provider.Usage
	closed       bool
}

func (r *AgentStreamReader) Part() <-chan provider.StreamPart {
	return r.reader.Part()
}

func (r *AgentStreamReader) Text() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.text.String()
}

func (r *AgentStreamReader) ToolCalls() []provider.ToolCall {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.toolCalls
}

func (r *AgentStreamReader) ToolResults() []provider.ToolCall {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.toolResults
}

func (r *AgentStreamReader) Steps() []StepResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.steps
}

func (r *AgentStreamReader) FinishReason() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.finishReason
}

func (r *AgentStreamReader) Usage() provider.Usage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.usage
}

func (r *AgentStreamReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return r.reader.Close()
}

func (r *AgentStreamReader) Err() error {
	return r.reader.Err()
}

type AgentCallOptions struct {
	Prompt      string
	System      string
	Messages    []provider.Message
	AbortSignal <-chan struct{}
	MaxRetries  int
	Timeout     int
}

type AgentGenerateResult struct {
	Text         string
	FinishReason string
	ToolCalls    []provider.ToolCall
	ToolResults  []provider.ToolCall
	Usage        provider.Usage
	Steps        []StepResult
}

type StreamPart struct {
	provider.StreamPart
}

func (a *ToolLoopAgent) generate(ctx context.Context, opts AgentCallOptions, streaming bool) (AgentGenerateResult, error) {
	messages := make([]provider.Message, 0)

	for _, m := range opts.Messages {
		messages = append(messages, m)
	}

	if opts.System != "" {
		messages = append([]provider.Message{{Role: "system", Content: opts.System}}, messages...)
	}

	if opts.Prompt != "" {
		messages = append(messages, provider.Message{Role: "user", Content: opts.Prompt})
	}

	if a.onStart != nil {
		a.onStart(OnStartEvent{
			Model:    "",
			Prompt:   opts.Prompt,
			System:   opts.System,
			Messages: messages,
		})
	}

	var steps []StepResult
	var allToolCalls []provider.ToolCall
	var allToolResults []provider.ToolCall
	var totalUsage provider.Usage

	for stepNum := 0; stepNum < a.maxSteps; stepNum++ {
		if a.onStepStart != nil {
			a.onStepStart(OnStepStartEvent{
				StepNumber: stepNum,
				Model:      "",
			})
		}

		result, err := a.executeStep(ctx, messages, streaming, stepNum, nil)
		if err != nil {
			return AgentGenerateResult{}, err
		}

		messages = result.Messages

		stepResult := StepResult{
			StepNumber:   stepNum,
			Text:         result.Text,
			ToolCalls:    result.ToolCalls,
			ToolResults:  result.ToolResults,
			FinishReason: result.FinishReason,
			Usage:        result.Usage,
			Messages:     messages,
		}

		steps = append(steps, stepResult)
		allToolCalls = append(allToolCalls, result.ToolCalls...)
		allToolResults = append(allToolResults, result.ToolResults...)

		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens
		totalUsage.TotalTokens += result.Usage.TotalTokens

		if a.onStepFinish != nil {
			a.onStepFinish(OnStepFinishEvent{
				StepNumber:   stepNum,
				Text:         result.Text,
				ToolCalls:    result.ToolCalls,
				ToolResults:  result.ToolResults,
				FinishReason: result.FinishReason,
				Usage:        result.Usage,
			})
		}

		shouldStop := false
		for _, cond := range a.stopConditions {
			if cond.ShouldStop(stepResult) {
				shouldStop = true
				break
			}
		}

		if shouldStop {
			break
		}

		if len(result.ToolCalls) == 0 {
			break
		}
	}

	finishReason := "stop"
	if len(steps) > 0 {
		finishReason = steps[len(steps)-1].FinishReason
	}

	finalText := ""
	if len(steps) > 0 {
		finalText = steps[len(steps)-1].Text
	}

	if a.onFinish != nil {
		a.onFinish(OnFinishEvent{
			Text:         finalText,
			ToolCalls:    allToolCalls,
			ToolResults:  allToolResults,
			FinishReason: finishReason,
			Usage:        totalUsage,
			Steps:        steps,
		})
	}

	return AgentGenerateResult{
		Text:         finalText,
		FinishReason: finishReason,
		ToolCalls:    allToolCalls,
		ToolResults:  allToolResults,
		Usage:        totalUsage,
		Steps:        steps,
	}, nil
}

func (a *ToolLoopAgent) stream(ctx context.Context, opts AgentCallOptions) (*stream.StreamReader, error) {
	reader := stream.NewStreamReader(ctx)

	go func() {
		messages := make([]provider.Message, 0)
		var totalUsage provider.Usage
		lastFinishReason := "stop"

		for _, m := range opts.Messages {
			messages = append(messages, m)
		}

		if opts.System != "" {
			messages = append([]provider.Message{{Role: "system", Content: opts.System}}, messages...)
		}

		if opts.Prompt != "" {
			messages = append(messages, provider.Message{Role: "user", Content: opts.Prompt})
		}

		if a.onStart != nil {
			a.onStart(OnStartEvent{
				Model:    "",
				Prompt:   opts.Prompt,
				System:   opts.System,
				Messages: messages,
			})
			reader.WritePart(provider.StreamPart{Type: "start"})
		}

		for stepNum := 0; stepNum < a.maxSteps; stepNum++ {
			if a.onStepStart != nil {
				a.onStepStart(OnStepStartEvent{
					StepNumber: stepNum,
					Model:      "",
				})
				reader.WritePart(provider.StreamPart{
					Type: "step-start",
					Text: fmt.Sprintf("%d", stepNum),
				})
			}

			result, err := a.executeStep(ctx, messages, true, stepNum, func(part provider.StreamPart) {
				if part.Type == "text-delta" || part.Type == "tool-call" {
					reader.WritePart(part)
				}
			})
			if err != nil {
				reader.SetError(err)
				reader.WritePart(provider.StreamPart{Type: "error", Error: err})
				reader.Close()
				return
			}

			messages = result.Messages

			totalUsage.InputTokens += result.Usage.InputTokens
			totalUsage.OutputTokens += result.Usage.OutputTokens
			totalUsage.TotalTokens += result.Usage.TotalTokens

			for _, tr := range result.ToolResults {
				reader.WritePart(provider.StreamPart{
					Type:       "tool-result",
					ToolCallID: tr.ID,
					ToolName:   tr.Name,
					ToolResult: tr.Output,
				})
			}

			if result.FinishReason != "" {
				lastFinishReason = result.FinishReason
			}

			if a.onStepFinish != nil {
				a.onStepFinish(OnStepFinishEvent{
					StepNumber:   stepNum,
					Text:         result.Text,
					ToolCalls:    result.ToolCalls,
					ToolResults:  result.ToolResults,
					FinishReason: result.FinishReason,
					Usage:        result.Usage,
				})
			}

			shouldStop := false
			for _, cond := range a.stopConditions {
				stepResult := StepResult{
					StepNumber:   stepNum,
					Text:         result.Text,
					ToolCalls:    result.ToolCalls,
					FinishReason: result.FinishReason,
				}
				if cond.ShouldStop(stepResult) {
					shouldStop = true
					break
				}
			}

			if shouldStop {
				break
			}

			if len(result.ToolCalls) == 0 {
				break
			}
		}

		reader.WritePart(provider.StreamPart{
			Type:         "finish",
			FinishReason: lastFinishReason,
			Usage:        totalUsage,
		})

		reader.Close()
	}()

	return reader, nil
}

type stepResult struct {
	Text         string
	ToolCalls    []provider.ToolCall
	ToolResults  []provider.ToolCall
	FinishReason string
	Usage        provider.Usage
	Messages     []provider.Message
}

func (a *ToolLoopAgent) executeStep(ctx context.Context, messages []provider.Message, streaming bool, stepNum int, onPart func(provider.StreamPart)) (stepResult, error) {
	tools := make([]provider.Tool, 0)
	for _, t := range a.tools {
		tools = append(tools, t)
	}

	genOpts := provider.GenerateTextOptions{
		Messages: messages,
		Tools:    tools,
	}

	if streaming {
		streamParts, err := a.model.StreamText(ctx, genOpts)
		if err != nil {
			return stepResult{}, err
		}

		var text stringBuilder
		var toolCalls []provider.ToolCall

		var finishReason string
		var usage provider.Usage

		for part := range streamParts {
			if part.Type == "error" {
				return stepResult{}, part.Error
			} else if part.Type == "text-delta" {
				if onPart != nil {
					onPart(part)
				}
				text.WriteString(part.Text)
			} else if part.Type == "tool-call" {
				if onPart != nil {
					onPart(part)
				}
				toolCalls = append(toolCalls, provider.ToolCall{
					ID:    part.ToolCallID,
					Name:  part.ToolName,
					Input: parseInput(part.ToolInput),
				})
			} else if part.Type == "finish" {
				finishReason = part.FinishReason
				usage = part.Usage
			}
		}

		toolResults, newMessages, err := a.executeToolCalls(ctx, toolCalls, messages)
		if err != nil {
			return stepResult{}, err
		}

		messages = newMessages

		if finishReason == "" && len(toolCalls) > 0 {
			finishReason = "tool-use"
		}

		if len(toolCalls) == 0 && text.String() != "" {
			messages = append(messages, provider.Message{
				Role:    "assistant",
				Content: text.String(),
			})
		}

		return stepResult{
			Text:         text.String(),
			ToolCalls:    toolCalls,
			ToolResults:  toolResults,
			FinishReason: finishReason,
			Usage:        usage,
			Messages:     messages,
		}, nil
	} else {
		result, err := a.model.GenerateText(ctx, genOpts)
		if err != nil {
			return stepResult{}, err
		}

		toolResults, newMessages, err := a.executeToolCalls(ctx, result.ToolCalls, messages)
		if err != nil {
			return stepResult{}, err
		}

		messages = newMessages
		messages = append(messages, provider.Message{
			Role:    "assistant",
			Content: result.Text,
		})

		return stepResult{
			Text:         result.Text,
			ToolCalls:    result.ToolCalls,
			ToolResults:  toolResults,
			FinishReason: result.FinishReason,
			Usage:        result.Usage,
			Messages:     messages,
		}, nil
	}
}

func (a *ToolLoopAgent) executeToolCalls(ctx context.Context, toolCalls []provider.ToolCall, messages []provider.Message) ([]provider.ToolCall, []provider.Message, error) {
	if len(toolCalls) == 0 {
		return nil, messages, nil
	}

	toolResults := make([]provider.ToolCall, 0)
	newMessages := make([]provider.Message, 0, len(messages)+len(toolCalls)+1)

	for _, m := range messages {
		newMessages = append(newMessages, m)
	}

	assistantMessage := provider.Message{
		Role:      "assistant",
		ToolCalls: toolCalls,
		Content:   "",
	}
	newMessages = append(newMessages, assistantMessage)

	for _, tc := range toolCalls {
		tool, exists := a.tools[tc.Name]
		if !exists {
			toolResults = append(toolResults, provider.ToolCall{
				ID:     tc.ID,
				Name:   tc.Name,
				Output: fmt.Sprintf("Error: Tool %s not found", tc.Name),
			})
			continue
		}

		if a.onToolCallStart != nil {
			a.onToolCallStart(OnToolCallStartEvent{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Input:      tc.Input,
			})
		}

		var output string
		var toolErr error

		if a.executeTools && tool.Execute != nil {
			result, err := tool.Execute(tc.Input)
			if err != nil {
				toolErr = err
				output = err.Error()
			} else {
				output = result
			}
		} else {
			output = "Tool execution disabled"
		}

		toolResult := provider.ToolCall{
			ID:     tc.ID,
			Name:   tc.Name,
			Input:  tc.Input,
			Output: output,
		}
		toolResults = append(toolResults, toolResult)

		if a.onToolCallFinish != nil {
			a.onToolCallFinish(OnToolCallFinishEvent{
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				Output:     output,
				Error:      toolErr,
			})
		}

	}

	if len(toolResults) > 0 {
		newMessages = append(newMessages, provider.Message{
			Role:        "tool",
			ToolResults: append([]provider.ToolCall{}, toolResults...),
		})
	}

	return toolResults, newMessages, nil
}

func parseInput(input string) map[string]interface{} {
	if input == "" {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(input), &result)
	if result == nil {
		return map[string]interface{}{}
	}
	return result
}
